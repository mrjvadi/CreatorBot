package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════
// مدل‌های vpn-bot
// ══════════════════════════════════════════════════════════════

type VPNUser struct {
	ID         string     `json:"id"`
	TelegramID int64      `json:"telegram_id"`
	Username   string     `json:"username"`
	PanelUser  string     `json:"panel_user"` // نام در Marzban/Hiddify
	SubURL     string     `json:"sub_url"`
	ExpiresAt  *time.Time `json:"expires_at"`
	DataLimit  int64      `json:"data_limit"`
	UsedData   int64      `json:"used_data"`
	IsActive   bool       `json:"is_active"`
}

func (u VPNUser) IsExpired() bool {
	return u.ExpiresAt != nil && u.ExpiresAt.Before(time.Now())
}

func (u VPNUser) RemainingGB() float64 {
	if u.DataLimit == 0 {
		return -1 // unlimited
	}
	return float64(u.DataLimit-u.UsedData) / (1024 * 1024 * 1024)
}

type VPNPlan struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Days      int     `json:"days"`
	DataGB    float64 `json:"data_gb"`
	IsActive  bool    `json:"is_active"`
}

type VPNPanelConfig struct {
	Type     string `json:"type"` // marzban|hiddify|xui|marzneshin
	BaseURL  string `json:"base_url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// ── Mock Panel ────────────────────────────────────────────────

type MockVPNPanel struct {
	panelType string
	users     map[string]*VPNUser
}

func NewMockPanel(pType string) *MockVPNPanel {
	return &MockVPNPanel{panelType: pType, users: make(map[string]*VPNUser)}
}

func (p *MockVPNPanel) CreateUser(username string, dataGB float64, days int) (*VPNUser, error) {
	exp := time.Now().AddDate(0, 0, days)
	u := &VPNUser{
		ID:        "panel_" + username,
		PanelUser: username,
		SubURL:    fmt.Sprintf("https://panel.test/sub/%s", username),
		ExpiresAt: &exp,
		DataLimit: int64(dataGB * 1024 * 1024 * 1024),
		IsActive:  true,
	}
	p.users[username] = u
	return u, nil
}

func (p *MockVPNPanel) GetUser(username string) (*VPNUser, bool) {
	u, ok := p.users[username]
	return u, ok
}

func (p *MockVPNPanel) RenewUser(username string, days int) error {
	u, ok := p.users[username]
	if !ok {
		return fmt.Errorf("user not found")
	}
	exp := time.Now().AddDate(0, 0, days)
	u.ExpiresAt = &exp
	u.UsedData = 0
	return nil
}

func (p *MockVPNPanel) DeleteUser(username string) error {
	delete(p.users, username)
	return nil
}

// ── env ──────────────────────────────────────────────────────

type VPNEnv struct {
	db    *MockDB
	cache *MockCache
	bot   *MockBot
	nats  *MockNATS
	state *MockStateStore
	panel *MockVPNPanel
	ctx   context.Context
}

func newVPNEnv(panelType string) *VPNEnv {
	c := NewMockCache()
	return &VPNEnv{
		db: NewMockDB(), cache: c,
		bot: NewMockBot(), nats: NewMockNATS(),
		state: NewMockStateStore(c),
		panel: NewMockPanel(panelType),
		ctx:   context.Background(),
	}
}

func (e *VPNEnv) seedVPNUser(tgID int64, days int, usedGB float64) VPNUser {
	exp := time.Now().AddDate(0, 0, days)
	panelName := fmt.Sprintf("user_%d", tgID)
	u := VPNUser{
		ID: fmt.Sprintf("vu_%d", tgID), TelegramID: tgID,
		PanelUser: panelName, ExpiresAt: &exp,
		DataLimit: 10 * 1024 * 1024 * 1024,
		UsedData:  int64(usedGB * 1024 * 1024 * 1024),
		IsActive:  true,
	}
	// در پنل هم ثبت کن
	e.panel.users[panelName] = &u
	e.db.Insert("vpn_users", u.ID, u)
	return u
}

func (e *VPNEnv) seedPlan(name string, price float64, days int, dataGB float64) VPNPlan {
	p := VPNPlan{ID: "vp_" + name, Name: name, Price: price, Days: days, DataGB: dataGB, IsActive: true}
	e.db.Insert("vpn_plans", p.ID, p)
	return p
}

// ── simulate handlers ────────────────────────────────────────

func (e *VPNEnv) handleStart(tgID int64) string {
	var found VPNUser
	hasUser := false
	for _, raw := range e.db.List("vpn_users") {
		var u VPNUser
		if unmarshalJSON(raw, &u) == nil && u.TelegramID == tgID {
			found = u
			hasUser = true
			break
		}
	}

	if !hasUser {
		msg := "👋 خوش آمدید!\nبرای خرید VPN پلن انتخاب کنید:"
		e.bot.Send(tgID, msg, [][]string{{"💎 خرید VPN"}, {"❓ راهنما"}})
		return msg
	}

	remaining := found.RemainingGB()
	remStr := fmt.Sprintf("%.1f GB", remaining)
	if remaining < 0 {
		remStr = "نامحدود"
	}
	daysLeft := 0
	if found.ExpiresAt != nil && !found.IsExpired() {
		daysLeft = int(time.Until(*found.ExpiresAt).Hours() / 24)
	}

	msg := fmt.Sprintf("📊 وضعیت VPN شما:\n\n⏰ %d روز مانده\n📶 %s مانده\n%s",
		daysLeft, remStr, func() string {
			if found.IsExpired() {
				return "❌ منقضی شده"
			}
			return "✅ فعال"
		}())

	btns := [][]string{
		{"📥 دریافت لینک", "📊 مصرف"},
		{"🔄 تمدید", "❓ راهنما"},
	}
	e.bot.Send(tgID, msg, btns)
	return msg
}

func (e *VPNEnv) handleBuyPlan(tgID int64, planID string) string {
	var plan VPNPlan
	if !e.db.Find("vpn_plans", planID, &plan) {
		msg := "❌ پلن یافت نشد."
		e.bot.Send(tgID, msg)
		return msg
	}

	// بررسی پرداخت (شبیه‌سازی)
	panelName := fmt.Sprintf("user_%d", tgID)
	pvpnUser, err := e.panel.CreateUser(panelName, plan.DataGB, plan.Days)
	if err != nil {
		msg := "❌ خطا در ایجاد اکانت."
		e.bot.Send(tgID, msg)
		return msg
	}

	// ذخیره در DB
	exp := time.Now().AddDate(0, 0, plan.Days)
	u := VPNUser{
		ID: fmt.Sprintf("vpu_%d", tgID), TelegramID: tgID,
		PanelUser: panelName, SubURL: pvpnUser.SubURL,
		ExpiresAt: &exp, DataLimit: pvpnUser.DataLimit, IsActive: true,
	}
	e.db.Insert("vpn_users", u.ID, u)

	e.nats.Publish("vpn.user.created", map[string]interface{}{
		"telegram_id": tgID, "panel_user": panelName, "plan": plan.Name,
	})

	msg := fmt.Sprintf("✅ اکانت VPN ایجاد شد!\n\n📅 %d روز\n📶 %.0f GB\n\n🔗 لینک اشتراک:\n%s",
		plan.Days, plan.DataGB, pvpnUser.SubURL)
	e.bot.Send(tgID, msg, [][]string{{"📥 دریافت لینک کانفیگ"}})
	return "created"
}

func (e *VPNEnv) handleGetConfig(tgID int64) string {
	var u VPNUser
	found := false
	for _, raw := range e.db.List("vpn_users") {
		if unmarshalJSON(raw, &u) == nil && u.TelegramID == tgID {
			found = true
			break
		}
	}
	if !found {
		msg := "❌ اکانت VPN ندارید."
		e.bot.Send(tgID, msg)
		return msg
	}
	if u.IsExpired() {
		msg := "⏰ اکانت شما منقضی شده. برای تمدید اقدام کنید."
		e.bot.Send(tgID, msg, [][]string{{"🔄 تمدید"}})
		return msg
	}

	msg := fmt.Sprintf("🔗 لینک اشتراک شما:\n\n<code>%s</code>\n\nاین لینک را در اپلیکیشن وارد کنید.",
		u.SubURL)
	e.bot.Send(tgID, msg, [][]string{{"📱 راهنمای نصب"}})
	return msg
}

func (e *VPNEnv) handleRenew(tgID int64, planID string) string {
	var u VPNUser
	found := false
	for _, raw := range e.db.List("vpn_users") {
		if unmarshalJSON(raw, &u) == nil && u.TelegramID == tgID {
			found = true
			break
		}
	}
	if !found {
		msg := "❌ اکانت یافت نشد."
		e.bot.Send(tgID, msg)
		return msg
	}

	var plan VPNPlan
	if !e.db.Find("vpn_plans", planID, &plan) {
		msg := "❌ پلن یافت نشد."
		e.bot.Send(tgID, msg)
		return msg
	}

	if err := e.panel.RenewUser(u.PanelUser, plan.Days); err != nil {
		msg := "❌ خطا در تمدید."
		e.bot.Send(tgID, msg)
		return msg
	}

	exp := time.Now().AddDate(0, 0, plan.Days)
	u.ExpiresAt = &exp
	u.UsedData = 0
	e.db.Update("vpn_users", u.ID, u)

	e.nats.Publish("vpn.user.renewed", map[string]interface{}{
		"telegram_id": tgID, "days": plan.Days,
	})

	msg := fmt.Sprintf("✅ اکانت تمدید شد!\n📅 %d روز اضافه شد.", plan.Days)
	e.bot.Send(tgID, msg)
	return "renewed"
}

// ══════════════════════════════════════════════════════════════
// تست‌ها
// ══════════════════════════════════════════════════════════════

func TestVPNBot_Start_NewUser(t *testing.T) {
	e := newVPNEnv("marzban")
	e.handleStart(1001)

	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "خرید VPN") {
		t.Errorf("expected buy VPN prompt, got: %s", msg.Text)
	}
	if !e.bot.HasButton("💎 خرید VPN") {
		t.Error("missing buy button")
	}
	t.Logf("✅ New user sees buy prompt")
}

func TestVPNBot_Start_ExistingUser(t *testing.T) {
	e := newVPNEnv("marzban")
	e.seedVPNUser(2001, 25, 3.5) // 25 روز مانده، 3.5 GB مصرف

	e.handleStart(2001)
	msg := e.bot.Last()

	if !strings.Contains(msg.Text, "روز مانده") {
		t.Errorf("expected days remaining, got: %s", msg.Text)
	}
	if !strings.Contains(msg.Text, "GB") {
		t.Errorf("expected GB remaining, got: %s", msg.Text)
	}
	for _, btn := range []string{"📥 دریافت لینک", "📊 مصرف", "🔄 تمدید"} {
		if !e.bot.HasButton(btn) {
			t.Errorf("missing button: %s", btn)
		}
	}
	t.Logf("✅ Existing user sees status: %s", msg.Text[:50])
}

func TestVPNBot_BuyPlan(t *testing.T) {
	e := newVPNEnv("marzban")
	plan := e.seedPlan("ماهانه", 5.0, 30, 50)

	result := e.handleBuyPlan(3001, plan.ID)
	if result != "created" {
		t.Fatalf("buy result = %q", result)
	}

	// تأیید DB
	if e.db.Count("vpn_users") != 1 {
		t.Errorf("vpn_users = %d, want 1", e.db.Count("vpn_users"))
	}

	// تأیید پنل
	panelUser, ok := e.panel.GetUser("user_3001")
	if !ok {
		t.Fatal("user not created in panel")
	}
	if panelUser.DataLimit != int64(50*1024*1024*1024) {
		t.Errorf("data_limit wrong: %d", panelUser.DataLimit)
	}

	// NATS
	if len(e.nats.Events("vpn.user.created")) != 1 {
		t.Error("vpn.user.created not published")
	}

	// پیام با لینک اشتراک
	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "panel.test/sub/") {
		t.Errorf("expected sub URL, got: %s", msg.Text)
	}
	t.Logf("✅ Plan purchased: %s", msg.Text[:60])
}

func TestVPNBot_GetConfig_Active(t *testing.T) {
	e := newVPNEnv("hiddify")
	u := e.seedVPNUser(4001, 10, 0)
	u.SubURL = "https://panel.test/sub/user_4001"
	e.db.Update("vpn_users", u.ID, u)

	e.handleGetConfig(4001)
	msg := e.bot.Last()

	if !strings.Contains(msg.Text, "panel.test") {
		t.Errorf("expected sub URL in msg, got: %s", msg.Text)
	}
	if !e.bot.HasButton("📱 راهنمای نصب") {
		t.Error("missing install guide button")
	}
	t.Logf("✅ Config delivered with URL")
}

func TestVPNBot_GetConfig_Expired(t *testing.T) {
	e := newVPNEnv("xui")
	past := time.Now().Add(-48 * time.Hour)
	u := VPNUser{ID: "vpu_5001", TelegramID: 5001, ExpiresAt: &past, IsActive: false}
	e.db.Insert("vpn_users", u.ID, u)

	e.handleGetConfig(5001)
	msg := e.bot.Last()

	if !strings.Contains(msg.Text, "منقضی") {
		t.Errorf("expected expired msg, got: %s", msg.Text)
	}
	if !e.bot.HasButton("🔄 تمدید") {
		t.Error("expected renew button")
	}
	t.Logf("✅ Expired account shows renew button")
}

func TestVPNBot_Renew(t *testing.T) {
	e := newVPNEnv("marzneshin")
	u := e.seedVPNUser(6001, 2, 8.0) // 2 روز مانده
	plan := e.seedPlan("تمدید ماهانه", 4.0, 30, 50)
	e.nats.Clear()

	result := e.handleRenew(6001, plan.ID)
	if result != "renewed" {
		t.Fatalf("renew result = %q", result)
	}

	// بررسی تمدید
	var updated VPNUser
	e.db.Find("vpn_users", u.ID, &updated)
	daysLeft := int(time.Until(*updated.ExpiresAt).Hours() / 24)
	if daysLeft < 29 {
		t.Errorf("days left = %d, want ≥29", daysLeft)
	}
	if updated.UsedData != 0 {
		t.Error("used data should be reset after renewal")
	}

	// NATS
	if len(e.nats.Events("vpn.user.renewed")) != 1 {
		t.Error("vpn.user.renewed not published")
	}
	t.Logf("✅ Renewed: %d days left, usage reset", daysLeft)
}

func TestVPNBot_AllPanelTypes(t *testing.T) {
	panels := []string{"marzban", "hiddify", "xui", "marzneshin"}
	for _, pt := range panels {
		t.Run(pt, func(t *testing.T) {
			e := newVPNEnv(pt)
			plan := e.seedPlan("test", 5.0, 30, 30)
			result := e.handleBuyPlan(int64(7000+len(pt)), plan.ID)
			if result != "created" {
				t.Errorf("panel=%s: %s", pt, result)
			}
			t.Logf("  ✅ %s panel: user created", pt)
		})
	}
}
