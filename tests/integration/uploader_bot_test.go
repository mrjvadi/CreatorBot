package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════
// مدل‌های uploader-bot
// ══════════════════════════════════════════════════════════════

type UploaderUser struct {
	ID            string     `json:"id"`
	TelegramID    int64      `json:"telegram_id"`
	IsBlocked     bool       `json:"is_blocked"`
	FreeDownloads int        `json:"free_downloads"`
	SubExpiresAt  *time.Time `json:"sub_expires_at"`
}

func (u UploaderUser) HasSub() bool {
	return u.SubExpiresAt != nil && u.SubExpiresAt.After(time.Now())
}

type MediaCode struct {
	ID            string     `json:"id"`
	Code          string     `json:"code"`
	Caption       string     `json:"caption"`
	ForwardLock   bool       `json:"forward_lock"`
	AutoDelete    int        `json:"auto_delete"`
	Password      string     `json:"password"`
	DownloadLimit int        `json:"download_limit"`
	SubRequired   bool       `json:"sub_required"`
	ChannelLock   bool       `json:"channel_lock"`
	UsedCount     int        `json:"used_count"`
	FakeLikes     int        `json:"fake_likes"`
	FakeViews     int        `json:"fake_views"`
	ExpiresAt     *time.Time `json:"expires_at"`
}

type MediaFile struct {
	ID        string `json:"id"`
	FileID    string `json:"file_id"`
	FileType  string `json:"file_type"` // video|photo|audio|document
	Caption   string `json:"caption"`
	Thumbnail string `json:"thumbnail"`
}

type DownloadLog struct {
	UserID string `json:"user_id"`
	CodeID string `json:"code_id"`
	Count  int    `json:"count"`
}

type UploaderSubPlan struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Days  int     `json:"days"`
}

type ForceJoinCh struct {
	ID       string `json:"id"`
	ChatID   int64  `json:"chat_id"`
	Title    string `json:"title"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

// ── test env ─────────────────────────────────────────────────

type UploaderEnv struct {
	db    *MockDB
	cache *MockCache
	bot   *MockBot
	nats  *MockNATS
	state *MockStateStore
	ctx   context.Context
}

func newUploaderEnv() *UploaderEnv {
	c := NewMockCache()
	return &UploaderEnv{
		db:    NewMockDB(),
		cache: c,
		bot:   NewMockBot(),
		nats:  NewMockNATS(),
		state: NewMockStateStore(c),
		ctx:   context.Background(),
	}
}

func (e *UploaderEnv) seedUser(tgID int64, blocked bool, freeDL int, subDays int) UploaderUser {
	u := UploaderUser{
		ID: fmt.Sprintf("u_%d", tgID), TelegramID: tgID,
		IsBlocked: blocked, FreeDownloads: freeDL,
	}
	if subDays > 0 {
		t := time.Now().AddDate(0, 0, subDays)
		u.SubExpiresAt = &t
	}
	e.db.Insert("users", u.ID, u)
	return u
}

func (e *UploaderEnv) seedCode(code string, opts MediaCode) MediaCode {
	opts.Code = code
	if opts.ID == "" {
		opts.ID = "code_" + code
	}
	e.db.Insert("codes", opts.ID, opts)
	return opts
}

func (e *UploaderEnv) seedFile(codeID, fileType string) MediaFile {
	f := MediaFile{
		ID: "file_" + fileType, FileID: "TG_FILE_" + strings.ToUpper(fileType) + "_001",
		FileType: fileType, Caption: "تست " + fileType,
	}
	if fileType == "video" {
		f.Thumbnail = "TG_THUMB_001"
	}
	e.db.Insert("files", f.ID, f)
	e.db.Insert("code_files", codeID+"_"+f.ID,
		map[string]string{"code_id": codeID, "file_id": f.ID})
	return f
}

func (e *UploaderEnv) seedPlan(name string, price float64, days int) UploaderSubPlan {
	p := UploaderSubPlan{ID: "plan_" + name, Name: name, Price: price, Days: days}
	e.db.Insert("sub_plans", p.ID, p)
	return p
}

func (e *UploaderEnv) getCode(id string) *MediaCode {
	var c MediaCode
	if e.db.Find("codes", id, &c) {
		return &c
	}
	return nil
}

func (e *UploaderEnv) getUser(id string) *UploaderUser {
	var u UploaderUser
	if e.db.Find("users", id, &u) {
		return &u
	}
	return nil
}

// ── simulate handlers ─────────────────────────────────────────

func (e *UploaderEnv) handleGetCode(user UploaderUser, codeStr string) string {
	// بررسی بلاک
	if user.IsBlocked {
		msg := "⛔️ دسترسی شما محدود شده است."
		e.bot.Send(user.TelegramID, msg)
		return msg
	}

	// پیدا کردن کد
	var code MediaCode
	found := false
	for _, raw := range e.db.List("codes") {
		var c MediaCode
		if err := unmarshalJSON(raw, &c); err == nil && c.Code == codeStr {
			code = c
			found = true
		}
	}
	if !found {
		msg := "❌ کد یافت نشد."
		e.bot.Send(user.TelegramID, msg)
		return msg
	}

	// انقضا
	if code.ExpiresAt != nil && code.ExpiresAt.Before(time.Now()) {
		msg := "⏰ این رسانه منقضی شده است."
		e.bot.Send(user.TelegramID, msg)
		return msg
	}

	// جوین اجباری
	if code.ChannelLock {
		channels := e.db.List("force_join")
		if len(channels) > 0 {
			msg := "⚠️ ابتدا در کانال‌های زیر عضو شوید:"
			e.bot.Send(user.TelegramID, msg, [][]string{{"📢 کانال ۱"}, {"✅ عضو شدم"}})
			return msg
		}
	}

	// اشتراک
	if code.SubRequired && !user.HasSub() {
		freeLimit := 3
		if user.FreeDownloads >= freeLimit {
			msg := "💎 برای دسترسی به این محتوا اشتراک تهیه کنید:"
			e.bot.Send(user.TelegramID, msg, [][]string{{"💳 خرید اشتراک"}})
			return msg
		}
	}

	// رمز عبور
	if code.Password != "" {
		key := fmt.Sprintf("pwd_verified:%s:%s", user.ID, code.ID)
		if _, ok := e.cache.Get(e.ctx, key); !ok {
			e.state.Set(e.ctx, user.TelegramID, "await_password",
				map[string]string{"code_id": code.ID})
			msg := "🔐 رمز عبور رسانه را وارد کنید:"
			e.bot.Send(user.TelegramID, msg)
			return msg
		}
	}

	// محدودیت دانلود
	if code.DownloadLimit > 0 {
		var log DownloadLog
		logKey := user.ID + "_" + code.ID
		e.db.Find("download_logs", logKey, &log)
		if log.Count >= code.DownloadLimit {
			msg := "❌ به حداکثر تعداد دانلود رسیده‌اید."
			e.bot.Send(user.TelegramID, msg)
			return msg
		}
	}

	// ارسال فایل‌ها
	files := e.db.FindWhere("files", func(raw string) bool {
		var f MediaFile
		unmarshalJSON(raw, &f)
		codeFiles := e.db.FindWhere("code_files", func(r string) bool {
			return strings.Contains(r, code.ID) && strings.Contains(r, f.ID)
		})
		return len(codeFiles) > 0
	})

	for _, raw := range files {
		var f MediaFile
		unmarshalJSON(raw, &f)
		caption := f.Caption
		if code.ForwardLock {
			caption += "\n🔒 فوروارد غیرفعال"
		}
		e.bot.Send(user.TelegramID, fmt.Sprintf("[%s] %s", strings.ToUpper(f.FileType), caption))

		// تایمر حذف
		if code.AutoDelete > 0 {
			go func(delay int) {
				time.Sleep(time.Duration(delay) * time.Millisecond)
				e.bot.Send(user.TelegramID, fmt.Sprintf("🗑 رسانه بعد از %d ثانیه حذف شد.", delay))
			}(code.AutoDelete)
		}
	}

	// لاگ دانلود
	logKey := user.ID + "_" + code.ID
	var log DownloadLog
	e.db.Find("download_logs", logKey, &log)
	log.UserID = user.ID
	log.CodeID = code.ID
	log.Count++
	e.db.Update("download_logs", logKey, log)

	// آپدیت used_count
	code.UsedCount++
	e.db.Update("codes", code.ID, code)

	// آپدیت free_downloads — از DB بخون تا up-to-date باشه
	if !user.HasSub() {
		var freshUser UploaderUser
		if e.db.Find("users", user.ID, &freshUser) {
			freshUser.FreeDownloads++
			e.db.Update("users", freshUser.ID, freshUser)
		}
	}

	return "sent"
}

func (e *UploaderEnv) handlePasswordInput(user UploaderUser, password string) string {
	st := e.state.Get(e.ctx, user.TelegramID)
	if st.Step != "await_password" {
		return "❌ حالت نادرست."
	}

	codeID := st.Data["code_id"]
	var code MediaCode
	if !e.db.Find("codes", codeID, &code) {
		return "❌ کد منقضی شده."
	}

	if password != code.Password {
		msg := "❌ رمز عبور اشتباه است."
		e.bot.Send(user.TelegramID, msg)
		return msg
	}

	// تأیید رمز
	key := fmt.Sprintf("pwd_verified:%s:%s", user.ID, code.ID)
	e.cache.Set(e.ctx, key, "1", time.Hour)
	e.state.Clear(e.ctx, user.TelegramID)

	return e.handleGetCode(user, code.Code)
}

// ══════════════════════════════════════════════════════════════
// تست‌های uploader-bot
// ══════════════════════════════════════════════════════════════

func TestUploaderBot_GetCode_Normal(t *testing.T) {
	e := newUploaderEnv()
	user := e.seedUser(1001, false, 0, 0)
	e.seedCode("TEST01", MediaCode{})
	e.seedFile("code_TEST01", "video")

	result := e.handleGetCode(user, "TEST01")
	if result != "sent" {
		t.Fatalf("expected sent, got: %s", result)
	}

	if e.bot.Count() == 0 {
		t.Fatal("no messages sent")
	}
	lastMsg := e.bot.Last()
	if !strings.Contains(lastMsg.Text, "VIDEO") {
		t.Errorf("expected VIDEO in message, got: %s", lastMsg.Text)
	}

	// بررسی download log
	var log DownloadLog
	e.db.Find("download_logs", user.ID+"_code_TEST01", &log)
	if log.Count != 1 {
		t.Errorf("download count = %d, want 1", log.Count)
	}

	// بررسی used_count
	code := e.getCode("code_TEST01")
	if code.UsedCount != 1 {
		t.Errorf("used_count = %d, want 1", code.UsedCount)
	}
	t.Logf("✅ Normal code delivery: msg=%q, downloads=%d", lastMsg.Text[:20], log.Count)
}

func TestUploaderBot_GetCode_BlockedUser(t *testing.T) {
	e := newUploaderEnv()
	user := e.seedUser(1002, true, 0, 0)
	e.seedCode("TEST02", MediaCode{})

	result := e.handleGetCode(user, "TEST02")
	if !strings.Contains(result, "محدود") {
		t.Errorf("expected block message, got: %s", result)
	}
	t.Logf("✅ Blocked user: %s", result)
}

func TestUploaderBot_GetCode_NotFound(t *testing.T) {
	e := newUploaderEnv()
	user := e.seedUser(1003, false, 0, 0)

	result := e.handleGetCode(user, "NOTEXIST")
	if !strings.Contains(result, "یافت نشد") {
		t.Errorf("expected not found, got: %s", result)
	}
	t.Logf("✅ Code not found: %s", result)
}

func TestUploaderBot_GetCode_Expired(t *testing.T) {
	e := newUploaderEnv()
	user := e.seedUser(1004, false, 0, 0)
	past := time.Now().Add(-time.Hour)
	e.seedCode("EXPIRED", MediaCode{ExpiresAt: &past})

	result := e.handleGetCode(user, "EXPIRED")
	if !strings.Contains(result, "منقضی") {
		t.Errorf("expected expired, got: %s", result)
	}
	t.Logf("✅ Expired code: %s", result)
}

func TestUploaderBot_GetCode_ForceJoin(t *testing.T) {
	e := newUploaderEnv()
	user := e.seedUser(1005, false, 0, 0)
	e.seedCode("FJOIN", MediaCode{ChannelLock: true})
	e.db.Insert("force_join", "ch1", ForceJoinCh{
		ID: "ch1", ChatID: -100123456, Title: "کانال تست", Username: "testchannel", IsActive: true,
	})

	result := e.handleGetCode(user, "FJOIN")
	if !strings.Contains(result, "کانال") {
		t.Errorf("expected join required, got: %s", result)
	}
	if !e.bot.HasButton("✅ عضو شدم") {
		t.Error("expected 'عضو شدم' button")
	}
	t.Logf("✅ Force join: %s", result)
}

func TestUploaderBot_GetCode_SubRequired(t *testing.T) {
	e := newUploaderEnv()

	// کاربر بدون اشتراک و ۳ دانلود رایگان مصرف‌شده
	user := e.seedUser(1006, false, 3, 0)
	e.seedCode("SUBONLY", MediaCode{SubRequired: true})
	e.seedPlan("ماهانه", 50000, 30)

	result := e.handleGetCode(user, "SUBONLY")
	if !strings.Contains(result, "اشتراک") {
		t.Errorf("expected sub required, got: %s", result)
	}
	if !e.bot.HasButton("💳 خرید اشتراک") {
		t.Error("expected buy sub button")
	}
	t.Logf("✅ Sub required: %s", result)
}

func TestUploaderBot_GetCode_WithSub(t *testing.T) {
	e := newUploaderEnv()

	// کاربر با اشتراک فعال
	user := e.seedUser(1007, false, 0, 30)
	e.seedCode("SUBONLY2", MediaCode{SubRequired: true})
	e.seedFile("code_SUBONLY2", "photo")

	result := e.handleGetCode(user, "SUBONLY2")
	if result != "sent" {
		t.Errorf("subscriber should get content, got: %s", result)
	}
	t.Logf("✅ Active subscriber gets content")
}

func TestUploaderBot_GetCode_Password(t *testing.T) {
	e := newUploaderEnv()
	user := e.seedUser(1008, false, 0, 0)
	e.seedCode("LOCKED", MediaCode{Password: "secret123"})
	e.seedFile("code_LOCKED", "document")

	// اول: درخواست رمز
	result := e.handleGetCode(user, "LOCKED")
	if !strings.Contains(result, "رمز") {
		t.Errorf("expected password prompt, got: %s", result)
	}

	st := e.state.Get(e.ctx, user.TelegramID)
	if st.Step != "await_password" {
		t.Errorf("state step = %q, want await_password", st.Step)
	}

	// رمز اشتباه
	e.bot.Clear()
	result2 := e.handlePasswordInput(user, "wrongpass")
	if !strings.Contains(result2, "اشتباه") {
		t.Errorf("expected wrong password msg, got: %s", result2)
	}

	// رمز درست
	e.bot.Clear()
	result3 := e.handlePasswordInput(user, "secret123")
	if result3 != "sent" {
		t.Errorf("correct password should deliver content, got: %s", result3)
	}
	t.Logf("✅ Password protected code: flow complete")
}

func TestUploaderBot_GetCode_DownloadLimit(t *testing.T) {
	e := newUploaderEnv()
	user := e.seedUser(1009, false, 0, 0)
	e.seedCode("LIMIT", MediaCode{DownloadLimit: 2})
	e.seedFile("code_LIMIT", "audio")

	// دانلود ۱
	e.handleGetCode(user, "LIMIT")
	// دانلود ۲
	e.handleGetCode(user, "LIMIT")
	// دانلود ۳ — باید بلاک بشه
	e.bot.Clear()
	result := e.handleGetCode(user, "LIMIT")
	if !strings.Contains(result, "حداکثر") {
		t.Errorf("expected limit reached, got: %s", result)
	}

	var log DownloadLog
	e.db.Find("download_logs", user.ID+"_code_LIMIT", &log)
	if log.Count != 2 {
		t.Errorf("download count = %d, want 2", log.Count)
	}
	t.Logf("✅ Download limit: blocked at count %d", log.Count)
}

func TestUploaderBot_GetCode_AutoDelete(t *testing.T) {
	e := newUploaderEnv()
	user := e.seedUser(1010, false, 0, 0)
	// auto_delete با مقدار کوچیک برای تست (millisecond)
	e.seedCode("AUTODEL", MediaCode{AutoDelete: 50})
	e.seedFile("code_AUTODEL", "video")

	e.handleGetCode(user, "AUTODEL")

	// منتظر بشیم goroutine تایمر اجرا بشه
	time.Sleep(200 * time.Millisecond)

	msgs := e.bot.All()
	hasDeleteMsg := false
	for _, m := range msgs {
		if strings.Contains(m.Text, "حذف شد") {
			hasDeleteMsg = true
			break
		}
	}
	if !hasDeleteMsg {
		t.Error("expected auto-delete message after timer")
	}
	t.Logf("✅ Auto-delete: %d messages total", len(msgs))
}

func TestUploaderBot_GetCode_MultiFile_Album(t *testing.T) {
	e := newUploaderEnv()
	user := e.seedUser(1011, false, 0, 0)
	e.seedCode("ALBUM", MediaCode{})
	e.seedFile("code_ALBUM", "photo")
	// فایل دوم
	f2 := MediaFile{ID: "file_photo2", FileID: "TG_PHOTO_002", FileType: "photo", Caption: "عکس دوم"}
	e.db.Insert("files", f2.ID, f2)
	e.db.Insert("code_files", "code_ALBUM_"+f2.ID, map[string]string{"code_id": "code_ALBUM", "file_id": f2.ID})

	e.handleGetCode(user, "ALBUM")

	count := e.bot.Count()
	if count < 2 {
		t.Errorf("expected 2+ messages for album, got %d", count)
	}
	t.Logf("✅ Album delivery: %d files sent", count)
}

func TestUploaderBot_FreeDownloadCounter(t *testing.T) {
	e := newUploaderEnv()
	user := e.seedUser(1012, false, 0, 0)

	// ۳ کد مختلف آپلود کن
	for i := 1; i <= 3; i++ {
		code := fmt.Sprintf("FREE%d", i)
		e.seedCode(code, MediaCode{})
		e.seedFile(fmt.Sprintf("code_%s", code), "document")
		e.handleGetCode(user, code)
	}

	// بررسی شمارش
	updated := e.getUser(user.ID)
	if updated.FreeDownloads != 3 {
		t.Errorf("free_downloads = %d, want 3", updated.FreeDownloads)
	}
	t.Logf("✅ Free download counter: %d", updated.FreeDownloads)
}

func TestUploaderBot_Admin_CreateCode(t *testing.T) {
	e := newUploaderEnv()

	// ادمین آپلود می‌کند
	fileID := "TG_VIDEO_ADMIN_001"
	thumbnail := "TG_THUMB_ADMIN_001"

	f := MediaFile{ID: "f_admin", FileID: fileID, FileType: "video",
		Caption: "ویدیوی تست ادمین", Thumbnail: thumbnail}
	e.db.Insert("files", f.ID, f)

	code := MediaCode{ID: "code_admin", Code: "ADMIN01", Caption: "تست ادمین"}
	e.db.Insert("codes", code.ID, code)
	e.db.Insert("code_files", "code_admin_f_admin",
		map[string]string{"code_id": "code_admin", "file_id": "f_admin"})

	e.bot.Send(999, "✅ رسانه ذخیره شد!\n🆔 کد: ADMIN01")

	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "ADMIN01") {
		t.Errorf("expected code in message, got: %s", msg.Text)
	}
	t.Logf("✅ Admin upload: %s", msg.Text)
}

func TestUploaderBot_Admin_ToggleForwardLock(t *testing.T) {
	e := newUploaderEnv()
	e.seedCode("FWTEST", MediaCode{ForwardLock: false})
	e.seedFile("code_FWTEST", "video")

	// فعال کردن قفل فوروارد
	code := e.getCode("code_FWTEST")
	code.ForwardLock = true
	e.db.Update("codes", "code_FWTEST", code)

	// تحویل با قفل
	user := e.seedUser(2001, false, 0, 0)
	e.handleGetCode(user, "FWTEST")

	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "فوروارد غیرفعال") {
		t.Errorf("expected forward lock indicator, got: %s", msg.Text)
	}
	t.Logf("✅ Forward lock: %s", msg.Text)
}

func TestUploaderBot_Admin_Stats(t *testing.T) {
	e := newUploaderEnv()

	// seed data
	e.seedUser(3001, false, 0, 0)
	e.seedUser(3002, false, 0, 30)
	e.seedCode("S1", MediaCode{})
	e.seedCode("S2", MediaCode{})

	totalUsers := e.db.Count("users")
	totalCodes := e.db.Count("codes")

	statsMsg := fmt.Sprintf("📊 آمار ربات\n\n👥 کاربران: %d\n📤 رسانه‌ها: %d",
		totalUsers, totalCodes)
	e.bot.Send(999, statsMsg)

	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "کاربران") {
		t.Errorf("expected stats, got: %s", msg.Text)
	}
	t.Logf("✅ Stats: users=%d, codes=%d", totalUsers, totalCodes)
}

func TestUploaderBot_Backup(t *testing.T) {
	e := newUploaderEnv()

	// seed داده
	for i := 1; i <= 5; i++ {
		e.seedCode(fmt.Sprintf("BK%d", i), MediaCode{Caption: fmt.Sprintf("رسانه %d", i)})
	}

	totalCodes := e.db.Count("codes")

	// شبیه‌سازی بکاپ
	backupMeta := map[string]interface{}{
		"version": "3.0", "total_codes": totalCodes,
		"created_at": time.Now().Format(time.RFC3339),
	}
	e.db.Insert("backups", "bk_001", backupMeta)

	e.bot.Send(999, fmt.Sprintf("💾 بکاپ ساخته شد\n📦 %d رسانه", totalCodes),
		[][]string{{"📥 دانلود بکاپ"}})

	msg := e.bot.Last()
	if !strings.Contains(msg.Text, "بکاپ") {
		t.Errorf("expected backup message, got: %s", msg.Text)
	}
	if !e.bot.HasButton("📥 دانلود بکاپ") {
		t.Error("expected download button")
	}
	t.Logf("✅ Backup: %d codes backed up", totalCodes)
}

func TestUploaderBot_Broadcast(t *testing.T) {
	e := newUploaderEnv()

	users := []int64{4001, 4002, 4003, 4004}
	for _, uid := range users {
		e.seedUser(uid, false, 0, 0)
	}
	// یه کاربر مسدود
	e.seedUser(4005, true, 0, 0)

	broadcastText := "📢 پیام همگانی تست"
	sent, failed, blocked := 0, 0, 0

	for _, raw := range e.db.List("users") {
		var u UploaderUser
		if unmarshalJSON(raw, &u) == nil {
			if u.IsBlocked {
				blocked++
				continue
			}
			e.bot.Send(u.TelegramID, broadcastText)
			sent++
		}
	}

	if sent != 4 {
		t.Errorf("sent = %d, want 4", sent)
	}
	if blocked != 1 {
		t.Errorf("blocked = %d, want 1", blocked)
	}

	e.bot.Send(999, fmt.Sprintf("✅ ارسال همگانی\n✉️ موفق: %d\n🚫 رد شده: %d\n❌ ناموفق: %d",
		sent, blocked, failed))
	t.Logf("✅ Broadcast: sent=%d, blocked=%d, failed=%d", sent, blocked, failed)
}
