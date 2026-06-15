package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════
// مدل‌ها
// ══════════════════════════════════════════════════════════════

type BMUser struct {
	ID         string `json:"id"`
	TelegramID int64  `json:"telegram_id"`
	Role       string `json:"role"` // user|admin|owner
	IsBlocked  bool   `json:"is_blocked"`
}

type BMPlan struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	DurationDay int     `json:"duration_day"`
	MaxBots     int     `json:"max_bots"`
	IsActive    bool    `json:"is_active"`
	IsFree      bool    `json:"is_free"`
	BotType     string  `json:"bot_type"`
}

type BMInstance struct {
	ID            string `json:"id"`
	OwnerID       string `json:"owner_id"`
	BotType       string `json:"bot_type"`
	ContainerName string `json:"container_name"`
	Status        string `json:"status"` // pending|running|stopped|failed|deleted
	BotID         int64  `json:"bot_id"`
}

type BMSub struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	PlanID    string    `json:"plan_id"`
	ExpiresAt time.Time `json:"expires_at"`
	IsActive  bool      `json:"is_active"`
}

type BMWallet struct {
	UserID  string  `json:"user_id"`
	Balance float64 `json:"balance"`
}

// ── env ──────────────────────────────────────────────────────

type BMEnv struct {
	db    *MockDB
	cache *MockCache
	bot   *MockBot
	nats  *MockNATS
	state *MockStateStore
	ctx   context.Context
}

func newBMEnv() *BMEnv {
	c := NewMockCache()
	return &BMEnv{
		db: NewMockDB(), cache: c,
		bot: NewMockBot(), nats: NewMockNATS(),
		state: NewMockStateStore(c), ctx: context.Background(),
	}
}

func (e *BMEnv) seedUser(tgID int64, role string) BMUser {
	u := BMUser{ID: fmt.Sprintf("u_%d", tgID), TelegramID: tgID, Role: role}
	e.db.Insert("users", u.ID, u)
	return u
}

func (e *BMEnv) seedPlan(name, botType string, price float64, maxBots int, isFree bool) BMPlan {
	p := BMPlan{
		ID: "plan_" + name, Name: name, Price: price,
		MaxBots: maxBots, DurationDay: 30, IsActive: true,
		IsFree: isFree, BotType: botType,
	}
	e.db.Insert("plans", p.ID, p)
	return p
}

func (e *BMEnv) seedInstance(ownerID, botType, status string) BMInstance {
	inst := BMInstance{
		ID: "inst_" + ownerID, OwnerID: ownerID,
		BotType: botType, Status: status,
		ContainerName: botType + "_" + ownerID,
		BotID:         9876543210,
	}
	e.db.Insert("instances", inst.ID, inst)
	return inst
}

func (e *BMEnv) countUserInstances(userID string) int {
	count := 0
	for _, raw := range e.db.List("instances") {
		var inst BMInstance
		if unmarshalJSON(raw, &inst) == nil && inst.OwnerID == userID && inst.Status != "deleted" {
			count++
		}
	}
	return count
}

// ── simulate handlers ────────────────────────────────────────

func (e *BMEnv) handleStart(user BMUser) string {
	if user.IsBlocked {
		msg := "⛔️ دسترسی محدود شده است."
		e.bot.Send(user.TelegramID, msg)
		return msg
	}
	if user.Role == "owner" || user.Role == "admin" {
		msg := "👑 به پنل مدیریت خوش آمدید!"
		e.bot.Send(user.TelegramID, msg, [][]string{
			{"👥 کاربران", "🤖 ربات‌ها"},
			{"💎 پلن‌ها", "🔗 لینک‌ها"},
			{"🖥 سرورها", "📦 تمپلیت‌ها"},
			{"📈 آمار"},
		})
		return msg
	}
	msg := "👋 خوش آمدید!\n\nاز منوی زیر استفاده کنید:"
	e.bot.Send(user.TelegramID, msg, [][]string{
		{"🤖 ربات‌های من", "💎 پلن‌ها"},
		{"❓ راهنما", "💬 پشتیبانی"},
	})
	return msg
}

func (e *BMEnv) handleMyBots(user BMUser) string {
	instances := e.db.FindWhere("instances", func(raw string) bool {
		var inst BMInstance
		unmarshalJSON(raw, &inst)
		return inst.OwnerID == user.ID && inst.Status != "deleted"
	})

	if len(instances) == 0 {
		msg := "📭 هیچ ربات فعالی ندارید.\nبرای ساخت ربات از «💎 پلن‌ها» استفاده کنید."
		e.bot.Send(user.TelegramID, msg, [][]string{{"💎 پلن‌ها"}})
		return msg
	}

	for _, raw := range instances {
		var inst BMInstance
		unmarshalJSON(raw, &inst)
		icon := map[string]string{"running": "🟢", "stopped": "🔴", "pending": "🟡", "failed": "❌"}[inst.Status]
		msg := fmt.Sprintf("%s %s\n📛 %s\n%s وضعیت: %s",
			botTypeIcon(inst.BotType), botTypeLabel(inst.BotType),
			inst.ContainerName, icon, inst.Status)

		var btns [][]string
		switch inst.Status {
		case "running":
			btns = [][]string{
				{"📊 آمار", "⚙️ تنظیمات"},
				{"🔄 ری‌استارت", "⏸ توقف"},
				{"🗑 حذف سرویس"},
			}
		case "stopped":
			btns = [][]string{{"▶️ شروع", "🗑 حذف"}}
		case "pending":
			btns = [][]string{{"🔄 بررسی وضعیت"}}
		case "failed":
			btns = [][]string{{"🔄 تلاش مجدد", "🗑 حذف"}}
		}
		e.bot.Send(user.TelegramID, msg, btns)
	}
	return "sent"
}

func (e *BMEnv) handleWizard(user BMUser, token, planID string) string {
	// استخراج bot_id
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 || parts[0] == "" {
		msg := "❌ توکن نامعتبر است."
		e.bot.Send(user.TelegramID, msg)
		return msg
	}
	botID := int64(0)
	fmt.Sscan(parts[0], &botID)
	if botID <= 0 {
		msg := "❌ توکن نامعتبر است."
		e.bot.Send(user.TelegramID, msg)
		return msg
	}

	// پلن
	var plan BMPlan
	if !e.db.Find("plans", planID, &plan) {
		msg := "❌ پلن یافت نشد."
		e.bot.Send(user.TelegramID, msg)
		return msg
	}

	// ظرفیت
	currentCount := e.countUserInstances(user.ID)
	if currentCount >= plan.MaxBots {
		msg := fmt.Sprintf("❌ به حداکثر ظرفیت رسیده‌اید (%d/%d). پلن ارتقا دهید.", currentCount, plan.MaxBots)
		e.bot.Send(user.TelegramID, msg)
		return msg
	}

	// پرداخت
	if !plan.IsFree {
		var wallet BMWallet
		e.db.Find("wallets", user.ID, &wallet)
		if wallet.Balance < plan.Price {
			msg := fmt.Sprintf("❌ موجودی ناکافی. دارید: %.1f TON، نیاز دارید: %.1f TON", wallet.Balance, plan.Price)
			e.bot.Send(user.TelegramID, msg)
			return msg
		}
		wallet.Balance -= plan.Price
		e.db.Update("wallets", user.ID, wallet)
	}

	// ساخت instance
	inst := BMInstance{
		ID: fmt.Sprintf("inst_%d", botID), OwnerID: user.ID,
		BotType: plan.BotType, Status: "pending",
		ContainerName: fmt.Sprintf("%s_%d", plan.BotType, botID),
		BotID:         botID,
	}
	e.db.Insert("instances", inst.ID, inst)

	// اشتراک
	sub := BMSub{
		ID: "sub_" + user.ID, UserID: user.ID, PlanID: plan.ID,
		ExpiresAt: time.Now().AddDate(0, 0, plan.DurationDay), IsActive: true,
	}
	e.db.Insert("subscriptions", sub.ID, sub)

	// NATS events
	e.nats.Publish("service.creation.requested", map[string]interface{}{
		"instance_id": inst.ID, "owner_id": user.ID,
		"bot_type": plan.BotType, "plan_id": plan.ID,
	})
	e.nats.Publish("plan.upgraded", map[string]interface{}{
		"user_id": user.ID, "plan_id": plan.ID,
	})

	msg := fmt.Sprintf("🎉 سرویس در حال راه‌اندازی!\n📛 نام: %s\n⏳ وضعیت: در انتظار", inst.ContainerName)
	e.bot.Send(user.TelegramID, msg)
	return "created"
}

func (e *BMEnv) handleInstanceAction(user BMUser, instanceID, action string) string {
	var inst BMInstance
	if !e.db.Find("instances", instanceID, &inst) {
		msg := "❌ سرویس یافت نشد."
		e.bot.Send(user.TelegramID, msg)
		return msg
	}
	if inst.OwnerID != user.ID {
		msg := "❌ این سرویس متعلق به شما نیست."
		e.bot.Send(user.TelegramID, msg)
		return msg
	}

	var newStatus string
	switch action {
	case "stop":
		newStatus = "stopped"
	case "start":
		newStatus = "running"
	case "restart":
		newStatus = "running"
	case "delete":
		newStatus = "deleted"
	default:
		return "❌ عملیات نامعتبر."
	}

	inst.Status = newStatus
	e.db.Update("instances", instanceID, inst)

	e.nats.Publish("agent.deploy", map[string]interface{}{
		"type": action, "instance_id": instanceID,
	})

	icons := map[string]string{"stop": "⏸", "start": "▶️", "restart": "🔄", "delete": "🗑"}
	msg := fmt.Sprintf("%s دستور %s برای سرویس %s اجرا شد.", icons[action], action, inst.ContainerName)
	e.bot.Send(user.TelegramID, msg)
	return newStatus
}

func botTypeIcon(t string) string {
	m := map[string]string{"vpn": "🌐", "uploader": "📤", "member": "🔒", "archive": "📦"}
	if v, ok := m[t]; ok {
		return v
	}
	return "🤖"
}

func botTypeLabel(t string) string {
	m := map[string]string{"vpn": "VPN", "uploader": "آپلودر", "member": "ممبرشیپ", "archive": "آرشیو"}
	if v, ok := m[t]; ok {
		return v
	}
	return t
}

// ══════════════════════════════════════════════════════════════
// تست‌ها
// ══════════════════════════════════════════════════════════════

func TestBotManager_Start_Admin(t *testing.T) {
	e := newBMEnv()
	admin := e.seedUser(7631742375, "owner")
	e.handleStart(admin)

	msg := e.bot.Last()
	if msg == nil {
		t.Fatal("no message")
	}
	if !strings.Contains(msg.Text, "پنل مدیریت") {
		t.Errorf("admin msg = %q", msg.Text)
	}
	for _, btn := range []string{"👥 کاربران", "🤖 ربات‌ها", "💎 پلن‌ها", "📈 آمار"} {
		if !e.bot.HasButton(btn) {
			t.Errorf("missing button: %s", btn)
		}
	}
	t.Logf("✅ Admin start: %s", msg.Text)
}

func TestBotManager_Start_User(t *testing.T) {
	e := newBMEnv()
	user := e.seedUser(1234567, "user")
	e.handleStart(user)

	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "خوش آمدید") {
		t.Errorf("user msg = %q", msg.Text)
	}
	for _, btn := range []string{"🤖 ربات‌های من", "💎 پلن‌ها"} {
		if !e.bot.HasButton(btn) {
			t.Errorf("missing button: %s", btn)
		}
	}
	t.Logf("✅ User start: buttons correct")
}

func TestBotManager_Start_Blocked(t *testing.T) {
	e := newBMEnv()
	user := BMUser{ID: "u_blocked", TelegramID: 9999, Role: "user", IsBlocked: true}
	e.db.Insert("users", user.ID, user)
	e.handleStart(user)

	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "محدود") {
		t.Errorf("blocked msg = %q", msg.Text)
	}
	t.Logf("✅ Blocked user rejected")
}

func TestBotManager_MyBots_Empty(t *testing.T) {
	e := newBMEnv()
	user := e.seedUser(2001, "user")
	e.handleMyBots(user)

	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "پلن") {
		t.Errorf("expected plan suggestion, got: %s", msg.Text)
	}
	if !e.bot.HasButton("💎 پلن‌ها") {
		t.Error("expected plans button")
	}
	t.Logf("✅ Empty bots list: suggests plans")
}

func TestBotManager_MyBots_Running(t *testing.T) {
	e := newBMEnv()
	user := e.seedUser(2002, "user")
	e.seedInstance(user.ID, "uploader", "running")
	e.handleMyBots(user)

	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "آپلودر") {
		t.Errorf("expected uploader label, got: %s", msg.Text)
	}
	for _, btn := range []string{"📊 آمار", "🔄 ری‌استارت", "⏸ توقف", "🗑 حذف سرویس"} {
		if !e.bot.HasButton(btn) {
			t.Errorf("missing button for running: %s", btn)
		}
	}
	t.Logf("✅ Running bot: correct buttons")
}

func TestBotManager_MyBots_Stopped(t *testing.T) {
	e := newBMEnv()
	user := e.seedUser(2003, "user")
	e.seedInstance(user.ID, "vpn", "stopped")
	e.handleMyBots(user)

	if !e.bot.HasButton("▶️ شروع") {
		t.Error("expected start button for stopped bot")
	}
	t.Logf("✅ Stopped bot: start button present")
}

func TestBotManager_MyBots_AllStatuses(t *testing.T) {
	e := newBMEnv()
	user := e.seedUser(2004, "user")

	statuses := []struct {
		btype  string
		status string
		btn    string
	}{
		{"uploader", "running", "⏸ توقف"},
		{"vpn", "stopped", "▶️ شروع"},
		{"member", "pending", "🔄 بررسی وضعیت"},
		{"archive", "failed", "🔄 تلاش مجدد"},
	}

	for _, s := range statuses {
		t.Run(s.status, func(t *testing.T) {
			e2 := newBMEnv()
			u := e2.seedUser(user.TelegramID, "user")
			e2.seedInstance(u.ID, s.btype, s.status)
			e2.handleMyBots(u)

			if !e2.bot.HasButton(s.btn) {
				t.Errorf("status=%s: missing button %s", s.status, s.btn)
			}
			t.Logf("  ✅ %s/%s → button '%s'", s.btype, s.status, s.btn)
		})
	}
}

func TestBotManager_Wizard_FreePlan(t *testing.T) {
	e := newBMEnv()
	user := e.seedUser(3001, "user")
	e.seedPlan("رایگان", "uploader", 0, 1, true)

	token := "1234567890:AABBCCDDEEFFaabbccddeeff1234567890X"
	result := e.handleWizard(user, token, "plan_رایگان")

	if result != "created" {
		t.Fatalf("wizard result = %q, want created", result)
	}

	// تأیید instance
	if e.db.Count("instances") != 1 {
		t.Errorf("instances = %d, want 1", e.db.Count("instances"))
	}
	// تأیید NATS
	events := e.nats.Events("service.creation.requested")
	if len(events) != 1 {
		t.Errorf("nats events = %d, want 1", len(events))
	}
	if events[0].Data["bot_type"] != "uploader" {
		t.Errorf("bot_type = %v", events[0].Data["bot_type"])
	}
	t.Logf("✅ Free plan wizard: instance created, NATS published")
}

func TestBotManager_Wizard_PaidPlan_OK(t *testing.T) {
	e := newBMEnv()
	user := e.seedUser(3002, "user")
	e.seedPlan("Starter", "vpn", 5.0, 3, false)
	e.db.Insert("wallets", user.ID, BMWallet{UserID: user.ID, Balance: 20.0})

	result := e.handleWizard(user, "9876543210:xxxxxxxxxx", "plan_Starter")

	if result != "created" {
		t.Fatalf("wizard result = %q", result)
	}

	// بررسی کسر موجودی
	var wallet BMWallet
	e.db.Find("wallets", user.ID, &wallet)
	if wallet.Balance != 15.0 {
		t.Errorf("balance = %.1f, want 15.0", wallet.Balance)
	}

	// NATS plan.upgraded
	if len(e.nats.Events("plan.upgraded")) != 1 {
		t.Error("plan.upgraded not published")
	}
	t.Logf("✅ Paid plan: balance deducted, instance created")
}

func TestBotManager_Wizard_PaidPlan_NoBalance(t *testing.T) {
	e := newBMEnv()
	user := e.seedUser(3003, "user")
	e.seedPlan("Pro", "vpn", 15.0, 10, false)
	e.db.Insert("wallets", user.ID, BMWallet{UserID: user.ID, Balance: 3.0})

	result := e.handleWizard(user, "1111111111:xxxxxxxxxx", "plan_Pro")

	if result == "created" {
		t.Fatal("should fail with insufficient balance")
	}
	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "ناکافی") {
		t.Errorf("expected insufficient balance msg, got: %s", msg.Text)
	}
	if e.db.Count("instances") != 0 {
		t.Error("instance should not be created")
	}
	t.Logf("✅ Insufficient balance: rejected correctly")
}

func TestBotManager_Wizard_CapacityLimit(t *testing.T) {
	e := newBMEnv()
	user := e.seedUser(3004, "user")
	e.seedPlan("Mini", "uploader", 0, 1, true)

	// اول یه ربات بساز
	e.handleWizard(user, "1111111111:xxxxxxxxxx", "plan_Mini")
	e.nats.Clear()
	e.bot.Clear()

	// دومی باید رد بشه
	result := e.handleWizard(user, "2222222222:xxxxxxxxxx", "plan_Mini")
	if result == "created" {
		t.Fatal("should reject over capacity")
	}
	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "ظرفیت") {
		t.Errorf("expected capacity msg, got: %s", msg.Text)
	}
	t.Logf("✅ Capacity limit: second bot rejected")
}

func TestBotManager_Wizard_InvalidToken(t *testing.T) {
	e := newBMEnv()
	user := e.seedUser(3005, "user")
	e.seedPlan("رایگان", "uploader", 0, 1, true)

	result := e.handleWizard(user, "invalid_token", "plan_رایگان")
	if result == "created" {
		t.Fatal("invalid token should be rejected")
	}
	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "نامعتبر") {
		t.Errorf("expected invalid token msg, got: %s", msg.Text)
	}
	t.Logf("✅ Invalid token rejected")
}

func TestBotManager_InstanceActions(t *testing.T) {
	actions := []struct {
		action    string
		newStatus string
		natsEvent string
	}{
		{"stop", "stopped", "agent.deploy"},
		{"start", "running", "agent.deploy"},
		{"restart", "running", "agent.deploy"},
		{"delete", "deleted", "agent.deploy"},
	}

	for _, a := range actions {
		t.Run(a.action, func(t *testing.T) {
			e := newBMEnv()
			user := e.seedUser(4001, "user")
			inst := e.seedInstance(user.ID, "uploader", "running")
			e.nats.Clear()

			result := e.handleInstanceAction(user, inst.ID, a.action)
			if result != a.newStatus {
				t.Errorf("action=%s: status=%q, want %q", a.action, result, a.newStatus)
			}

			var updated BMInstance
			e.db.Find("instances", inst.ID, &updated)
			if updated.Status != a.newStatus {
				t.Errorf("DB status=%q, want %q", updated.Status, a.newStatus)
			}

			events := e.nats.Events(a.natsEvent)
			if len(events) == 0 {
				t.Errorf("NATS %s not published", a.natsEvent)
			}
			if events[0].Data["type"] != a.action {
				t.Errorf("NATS type=%v, want %s", events[0].Data["type"], a.action)
			}
			t.Logf("  ✅ %s → status=%s, NATS published", a.action, a.newStatus)
		})
	}
}

func TestBotManager_InstanceAction_WrongOwner(t *testing.T) {
	e := newBMEnv()
	owner := e.seedUser(5001, "user")
	attacker := e.seedUser(5002, "user")
	inst := e.seedInstance(owner.ID, "vpn", "running")

	result := e.handleInstanceAction(attacker, inst.ID, "stop")
	if result == "stopped" {
		t.Fatal("attacker should not control other user's instance")
	}
	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "متعلق به شما نیست") {
		t.Errorf("expected ownership error, got: %s", msg.Text)
	}
	t.Logf("✅ Ownership check: attacker rejected")
}

func TestBotManager_AllBotTypes(t *testing.T) {
	types := []string{"uploader", "vpn", "member", "archive"}
	for _, bt := range types {
		t.Run(bt, func(t *testing.T) {
			e := newBMEnv()
			user := e.seedUser(6001, "user")
			e.seedPlan("plan_"+bt, bt, 0, 1, true)
			token := fmt.Sprintf("3333333%s:xxxxxxxxxx", bt[:3])
			result := e.handleWizard(user, token, "plan_plan_"+bt)
			if result != "created" {
				t.Errorf("bot_type=%s: %s", bt, result)
			}
			var inst BMInstance
			for _, raw := range e.db.List("instances") {
				unmarshalJSON(raw, &inst)
			}
			if inst.BotType != bt {
				t.Errorf("bot_type=%q, want %q", inst.BotType, bt)
			}
			t.Logf("  ✅ %s bot created", bt)
		})
	}
}
