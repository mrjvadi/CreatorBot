package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════
// مدل‌های member-bot
// ══════════════════════════════════════════════════════════════

type MemberUser struct {
	ID           string     `json:"id"`
	TelegramID   int64      `json:"telegram_id"`
	IsMember     bool       `json:"is_member"`
	SubExpiresAt *time.Time `json:"sub_expires_at"`
}

func (u MemberUser) HasActiveSub() bool {
	return u.SubExpiresAt != nil && u.SubExpiresAt.After(time.Now())
}

type MemberChannel struct {
	ID        string `json:"id"`
	ChatID    int64  `json:"chat_id"`
	Title     string `json:"title"`
	CheckBot  int64  `json:"check_bot"` // bot_id که ادمین کانال هست
	IsActive  bool   `json:"is_active"`
}

type MemberPlan struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Days  int     `json:"days"`
}

// ── Mock Membership Checker ───────────────────────────────────

type MockMembershipChecker struct {
	members map[int64]map[int64]bool // chatID → userID → isMember
}

func NewMockMembershipChecker() *MockMembershipChecker {
	return &MockMembershipChecker{members: make(map[int64]map[int64]bool)}
}

func (m *MockMembershipChecker) SetMember(chatID, userID int64, isMember bool) {
	if m.members[chatID] == nil {
		m.members[chatID] = make(map[int64]bool)
	}
	m.members[chatID][userID] = isMember
}

func (m *MockMembershipChecker) IsMember(chatID, userID int64) bool {
	if ch, ok := m.members[chatID]; ok {
		return ch[userID]
	}
	return false
}

// ── env ──────────────────────────────────────────────────────

type MemberEnv struct {
	db      *MockDB
	cache   *MockCache
	bot     *MockBot
	nats    *MockNATS
	state   *MockStateStore
	checker *MockMembershipChecker
	ctx     context.Context
}

func newMemberEnv() *MemberEnv {
	c := NewMockCache()
	return &MemberEnv{
		db: NewMockDB(), cache: c,
		bot: NewMockBot(), nats: NewMockNATS(),
		state:   NewMockStateStore(c),
		checker: NewMockMembershipChecker(),
		ctx:     context.Background(),
	}
}

func (e *MemberEnv) seedChannel(chatID int64, title string) MemberChannel {
	ch := MemberChannel{
		ID: fmt.Sprintf("ch_%d", chatID), ChatID: chatID,
		Title: title, CheckBot: 999999999, IsActive: true,
	}
	e.db.Insert("channels", ch.ID, ch)
	return ch
}

func (e *MemberEnv) seedMemberUser(tgID int64, days int) MemberUser {
	u := MemberUser{ID: fmt.Sprintf("mu_%d", tgID), TelegramID: tgID}
	if days > 0 {
		exp := time.Now().AddDate(0, 0, days)
		u.SubExpiresAt = &exp
		u.IsMember = true
	}
	e.db.Insert("member_users", u.ID, u)
	return u
}

func (e *MemberEnv) seedPlan(name string, price float64, days int) MemberPlan {
	p := MemberPlan{ID: "mp_" + name, Name: name, Price: price, Days: days}
	e.db.Insert("member_plans", p.ID, p)
	return p
}

// ── simulate handlers ─────────────────────────────────────────

func (e *MemberEnv) handleJoinCheck(tgID int64, chatID int64) string {
	isMember := e.checker.IsMember(chatID, tgID)

	var u MemberUser
	found := false
	for _, raw := range e.db.List("member_users") {
		if unmarshalJSON(raw, &u) == nil && u.TelegramID == tgID {
			found = true
			break
		}
	}

	if !found {
		// کاربر جدید
		u = MemberUser{ID: fmt.Sprintf("mu_%d", tgID), TelegramID: tgID}
		e.db.Insert("member_users", u.ID, u)
	}

	if isMember && u.HasActiveSub() {
		return "already_member"
	}

	if !u.HasActiveSub() {
		// نیاز به اشتراک
		var plans []MemberPlan
		for _, raw := range e.db.List("member_plans") {
			var p MemberPlan
			if unmarshalJSON(raw, &p) == nil {
				plans = append(plans, p)
			}
		}

		btns := [][]string{}
		for _, p := range plans {
			btns = append(btns, []string{fmt.Sprintf("💎 %s — %.0f تومان", p.Name, p.Price)})
		}
		msg := "🔒 برای عضویت در کانال باید اشتراک تهیه کنید:"
		e.bot.Send(tgID, msg, btns)
		return msg
	}

	// اشتراک دارد → اضافه کن به کانال
	e.checker.SetMember(chatID, tgID, true)
	e.nats.Publish("membership.joined", map[string]interface{}{
		"telegram_id": tgID, "chat_id": chatID,
	})

	msg := "✅ به کانال اضافه شدید!"
	e.bot.Send(tgID, msg)
	return "joined"
}

func (e *MemberEnv) handleBuySub(tgID int64, planID string, chatID int64) string {
	var plan MemberPlan
	if !e.db.Find("member_plans", planID, &plan) {
		msg := "❌ پلن یافت نشد."
		e.bot.Send(tgID, msg)
		return msg
	}

	// ایجاد/آپدیت کاربر
	var u MemberUser
	found := false
	for _, raw := range e.db.List("member_users") {
		if unmarshalJSON(raw, &u) == nil && u.TelegramID == tgID {
			found = true
			break
		}
	}
	if !found {
		u = MemberUser{ID: fmt.Sprintf("mu_%d", tgID), TelegramID: tgID}
	}

	exp := time.Now().AddDate(0, 0, plan.Days)
	u.SubExpiresAt = &exp
	u.IsMember = true
	e.db.Update("member_users", u.ID, u)

	// اضافه کردن به کانال
	e.checker.SetMember(chatID, tgID, true)

	e.nats.Publish("membership.joined", map[string]interface{}{
		"telegram_id": tgID, "chat_id": chatID, "plan": plan.Name,
	})

	msg := fmt.Sprintf("🎉 اشتراک %s فعال شد!\n📅 تا %s عضو هستید.",
		plan.Name, exp.Format("2006-01-02"))
	e.bot.Send(tgID, msg)
	return "subscribed"
}

func (e *MemberEnv) handleExpiredCheck(chatID int64) []int64 {
	// بررسی کاربران منقضی‌شده
	var expired []int64
	for _, raw := range e.db.List("member_users") {
		var u MemberUser
		if unmarshalJSON(raw, &u) == nil && !u.HasActiveSub() && u.IsMember {
			expired = append(expired, u.TelegramID)
			// حذف از کانال
			e.checker.SetMember(chatID, u.TelegramID, false)
			u.IsMember = false
			e.db.Update("member_users", u.ID, u)
			e.nats.Publish("membership.left", map[string]interface{}{
				"telegram_id": u.TelegramID, "chat_id": chatID, "reason": "expired",
			})
		}
	}
	return expired
}

// ══════════════════════════════════════════════════════════════
// تست‌ها
// ══════════════════════════════════════════════════════════════

func TestMemberBot_JoinCheck_NoSub(t *testing.T) {
	e := newMemberEnv()
	ch := e.seedChannel(-100123456, "کانال پرمیوم")
	e.seedPlan("ماهانه", 50000, 30)
	e.seedPlan("سه ماهه", 120000, 90)

	result := e.handleJoinCheck(1001, ch.ChatID)

	if !strings.Contains(result, "اشتراک") {
		t.Errorf("expected sub required, got: %s", result)
	}
	if !e.bot.HasButton("💎 ماهانه — 50000 تومان") {
		t.Error("expected monthly plan button")
	}
	t.Logf("✅ No sub: shows plans")
}

func TestMemberBot_JoinCheck_WithSub(t *testing.T) {
	e := newMemberEnv()
	ch := e.seedChannel(-100123456, "کانال پرمیوم")
	e.seedMemberUser(2001, 30)
	e.nats.Clear()

	result := e.handleJoinCheck(2001, ch.ChatID)

	if result != "joined" {
		t.Errorf("expected joined, got: %s", result)
	}
	if !e.checker.IsMember(ch.ChatID, 2001) {
		t.Error("user should be added to channel")
	}
	if len(e.nats.Events("membership.joined")) != 1 {
		t.Error("membership.joined not published")
	}
	t.Logf("✅ Sub active: user joined channel")
}

func TestMemberBot_BuySub(t *testing.T) {
	e := newMemberEnv()
	ch := e.seedChannel(-100777888, "کانال VIP")
	plan := e.seedPlan("ماهانه", 50000, 30)
	e.nats.Clear()

	result := e.handleBuySub(3001, plan.ID, ch.ChatID)

	if result != "subscribed" {
		t.Fatalf("buy result = %q", result)
	}

	// تأیید عضویت
	if !e.checker.IsMember(ch.ChatID, 3001) {
		t.Error("should be member after purchase")
	}

	// تأیید اشتراک
	var u MemberUser
	for _, raw := range e.db.List("member_users") {
		var tmp MemberUser
		if unmarshalJSON(raw, &tmp) == nil && tmp.TelegramID == 3001 {
			u = tmp
		}
	}
	if !u.HasActiveSub() {
		t.Error("subscription should be active")
	}
	daysLeft := int(time.Until(*u.SubExpiresAt).Hours() / 24)
	if daysLeft < 29 {
		t.Errorf("days left = %d, want ≥29", daysLeft)
	}

	// NATS
	if len(e.nats.Events("membership.joined")) != 1 {
		t.Error("membership.joined not published")
	}

	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "ماهانه") {
		t.Errorf("expected plan name in msg, got: %s", msg.Text)
	}
	t.Logf("✅ Subscribed: %d days, channel joined", daysLeft)
}

func TestMemberBot_ExpiredRemoval(t *testing.T) {
	e := newMemberEnv()
	ch := e.seedChannel(-100999000, "کانال تست")

	// کاربر با اشتراک منقضی
	past := time.Now().Add(-time.Hour)
	expiredUser := MemberUser{
		ID: "mu_exp", TelegramID: 4001,
		SubExpiresAt: &past, IsMember: true,
	}
	e.db.Insert("member_users", "mu_exp", expiredUser)
	e.checker.SetMember(ch.ChatID, 4001, true)

	// کاربر با اشتراک فعال
	e.seedMemberUser(4002, 10)
	e.checker.SetMember(ch.ChatID, 4002, true)

	e.nats.Clear()
	expired := e.handleExpiredCheck(ch.ChatID)

	if len(expired) != 1 {
		t.Errorf("expired count = %d, want 1", len(expired))
	}
	if expired[0] != 4001 {
		t.Errorf("expired user = %d, want 4001", expired[0])
	}

	// کاربر منقضی از کانال حذف شد
	if e.checker.IsMember(ch.ChatID, 4001) {
		t.Error("expired user should be removed from channel")
	}
	// کاربر فعال باقی‌ماند
	if !e.checker.IsMember(ch.ChatID, 4002) {
		t.Error("active user should remain in channel")
	}

	// NATS
	if len(e.nats.Events("membership.left")) != 1 {
		t.Error("membership.left not published")
	}
	leftEvent := e.nats.Events("membership.left")[0]
	if leftEvent.Data["reason"] != "expired" {
		t.Errorf("reason = %v, want expired", leftEvent.Data["reason"])
	}
	t.Logf("✅ Expired removal: %d removed, active users kept", len(expired))
}

func TestMemberBot_MultiChannel(t *testing.T) {
	e := newMemberEnv()
	ch1 := e.seedChannel(-100111, "کانال ۱")
	ch2 := e.seedChannel(-100222, "کانال ۲")
	e.seedMemberUser(5001, 30)
	e.nats.Clear()

	// عضویت در ۲ کانال
	r1 := e.handleJoinCheck(5001, ch1.ChatID)
	r2 := e.handleJoinCheck(5001, ch2.ChatID)

	if r1 != "joined" || r2 != "joined" {
		t.Errorf("r1=%s, r2=%s", r1, r2)
	}
	if !e.checker.IsMember(ch1.ChatID, 5001) || !e.checker.IsMember(ch2.ChatID, 5001) {
		t.Error("should be member in both channels")
	}

	events := e.nats.Events("membership.joined")
	if len(events) != 2 {
		t.Errorf("expected 2 join events, got %d", len(events))
	}
	t.Logf("✅ Multi-channel: joined %d channels", len(events))
}
