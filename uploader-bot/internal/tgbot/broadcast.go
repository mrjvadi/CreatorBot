package tgbot

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/util"
)

// bcJobMsg یک کار ارسال همگانی (کپی یا فوروارد یک پیام منبع).
type bcJobMsg struct {
	Mode       string // "copy" یا "forward"
	SrcChat    int64  // چت منبع (پیوی ادمین)
	SrcMsg     int    // شناسه‌ی پیام منبع
	Preview    string // خلاصه‌ی متن/کپشن برای نمایش وضعیت
	Pin        bool
	AutoDelete bool
	DelHours   int
	Initiator  int64
}

// حداقل فاصله‌ی مجاز بین ارسال‌ها (تلگرام ~۳۰ پیام/ثانیه به کاربران مختلف).
const bcMinDelayMS = 30
const bcDefaultDelayMS = 50

// enqueueBroadcast یک کار همگانی را به صف می‌فرستد (بدون بلاک‌کردن).
func (h *Handler) enqueueBroadcast(job bcJobMsg) bool {
	if h.bcJobs == nil {
		return false
	}
	select {
	case h.bcJobs <- job:
		return true
	default:
		return false // صف پر است
	}
}

// adminBroadcastMenu انتخاب نوع ارسال همگانی (کپی/فوروارد).
func (h *Handler) adminBroadcastMenu(ctx context.Context, c tele.Context) error {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("📨 ارسال همگانی (کپی)", "bc_copy")),
		kb.Row(kb.Data("↪️ فوروارد همگانی", "bc_forward")),
	)
	return c.Send("📢 نوع ارسال همگانی را انتخاب کنید:\n\n«کپی» بدون برچسب فورواردشده می‌فرستد؛ «فوروارد» با منبع.", kb)
}

// askBroadcastContent از ادمین می‌خواهد محتوای همگانی (هر نوع) را بفرستد.
func (h *Handler) askBroadcastContent(ctx context.Context, c tele.Context, mode string) error {
	h.SetStepData(ctx, c.Sender().ID, stepBroadcast, "bc_mode", mode)
	return c.Send("✏️ پیام همگانی را بفرستید (متن، عکس، ویدیو، فایل و …):", kbCancelOnly())
}

// finalizeBroadcast پیام منبع را گرفته و کار همگانی را به صف می‌فرستد.
func (h *Handler) finalizeBroadcast(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	st := h.GetState(ctx, uid)
	mode := st.Data["bc_mode"]
	if mode == "" {
		mode = "copy"
	}
	h.ClearState(ctx, uid)

	msg := c.Message()
	preview := msg.Text
	if preview == "" {
		preview = msg.Caption
	}
	if preview == "" {
		preview = "[رسانه]"
	}

	job := bcJobMsg{
		Mode:       mode,
		SrcChat:    msg.Chat.ID,
		SrcMsg:     msg.ID,
		Preview:    shortCap(preview),
		Pin:        h.Store.GetSetting(ctx, models.SettingBroadcastPin) == "true",
		AutoDelete: h.Store.GetSetting(ctx, models.SettingBroadcastAutoDelete) == "true",
		DelHours:   h.GetSettingInt(ctx, models.SettingBroadcastDeleteHours, 24),
		Initiator:  uid,
	}
	if !h.enqueueBroadcast(job) {
		return c.Send("⚠️ صف ارسال همگانی پر است. کمی بعد دوباره تلاش کنید.", kbAdmin())
	}
	out := "📢 ارسال همگانی در صف قرار گرفت و در پس‌زمینه انجام می‌شود.\nگزارش نتیجه پس از پایان ارسال می‌شود."
	if job.AutoDelete {
		out += fmt.Sprintf("\n🗑 بعد از %d ساعت از پیوی همه حذف می‌شود.", job.DelHours)
	}
	return c.Send(out, kbAdmin())
}

// broadcastWorker تک‌worker که کارهای همگانی را یکی‌یکی پردازش می‌کند
// تا نرخ کلی کنترل‌شده بماند و روی سرعت ربات اثر نگذارد.
func (h *Handler) broadcastWorker() {
	for job := range h.bcJobs {
		h.runBroadcast(context.Background(), job)
	}
}

// runBroadcast پیام را با محدودیت نرخ و رعایت قوانین تلگرام ارسال می‌کند
// و وضعیت زنده را در دیتابیس ثبت می‌کند.
func (h *Handler) runBroadcast(ctx context.Context, job bcJobMsg) {
	users, _, err := h.Store.ListUsers(ctx, 1, 1_000_000)
	h.LogErr("runBroadcast: list users", err)

	// تعداد قابل ارسال (به‌جز بلاک‌شده‌ها در دیتابیس)
	total := 0
	for _, u := range users {
		if !u.IsBlocked {
			total++
		}
	}
	code := genBcCode()
	jobID := h.Store.CreateBroadcastJob(ctx, code, job.Mode, job.Preview, total)

	// فاصله‌ی بین ارسال‌ها (قابل تنظیم، با حداقل ایمن)
	delayMS := h.GetSettingInt(ctx, models.SettingBroadcastDelayMS, bcDefaultDelayMS)
	if delayMS < bcMinDelayMS {
		delayMS = bcMinDelayMS
	}
	delay := time.Duration(delayMS) * time.Millisecond

	var deleteAt time.Time
	if job.AutoDelete {
		hrs := job.DelHours
		if hrs <= 0 {
			hrs = 24
		}
		deleteAt = time.Now().Add(time.Duration(hrs) * time.Hour)
	}

	sent, failed, blocked := 0, 0, 0
	canceled := false
	i := 0
	for _, u := range users {
		if u.IsBlocked {
			continue
		}
		msg, err := h.bcSendOne(job, u.TelegramID)
		if err != nil {
			if isBlockedErr(err) {
				blocked++
			} else {
				failed++
			}
		} else {
			sent++
			if msg != nil {
				if job.Pin {
					// پیام بازگشتی از Copy/Forward ممکن است Chat نداشته باشد؛
					// برای Pin پیام را با ChatID مقصد می‌سازیم. پین‌نشدن یک پیام
					// تکی نباید کل ارسال همگانی را متوقف کند — فقط لاگ می‌شود.
					h.LogErr("runBroadcast: pin", h.Bot.Pin(&tele.Message{ID: msg.ID, Chat: &tele.Chat{ID: u.TelegramID}}, tele.Silent))
				}
				if job.AutoDelete {
					h.Store.AddBroadcastMsg(ctx, code, u.TelegramID, msg.ID, deleteAt)
				}
			}
		}
		i++
		if i%25 == 0 {
			h.Store.UpdateBroadcastJob(ctx, jobID, sent, failed, blocked, false)
			// بررسی درخواست توقف
			j, jErr := h.Store.GetBroadcastJob(ctx, jobID)
			h.LogErr("runBroadcast: check cancel", jErr)
			if j != nil && j.Canceled {
				canceled = true
				break
			}
		}
		time.Sleep(delay)
	}
	h.Store.UpdateBroadcastJob(ctx, jobID, sent, failed, blocked, true)

	if job.Initiator != 0 {
		head := "📢 ارسال همگانی تمام شد."
		if canceled {
			head = "⛔️ ارسال همگانی متوقف شد."
		}
		summary := fmt.Sprintf("%s\n🔖 کد: <code>%s</code>\n✅ موفق: %d\n🚫 بلاک: %d\n❌ ناموفق: %d",
			head, code, sent, blocked, failed)
		if job.AutoDelete {
			summary += fmt.Sprintf("\n🗑 حذف خودکار بعد از %d ساعت", job.DelHours)
		}
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data("📊 وضعیت/کنترل", "bcjob:"+code)))
		if _, err := h.Bot.Send(&tele.User{ID: job.Initiator}, summary, tele.ModeHTML, kb); err != nil {
			h.LogErr("runBroadcast: notify initiator", err)
		}
	}
}

// genBcCode کد کوتاهِ یکتای نسبی برای یک همگانی می‌سازد.
func genBcCode() string {
	const al = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 5)
	for i := range b {
		b[i] = al[rand.Intn(len(al))]
	}
	return string(b)
}

// bcSendOne پیام منبع را به یک کاربر کپی یا فوروارد می‌کند (با مدیریت FloodWait).
func (h *Handler) bcSendOne(job bcJobMsg, uid int64) (*tele.Message, error) {
	to := &tele.User{ID: uid}
	src := &tele.Message{ID: job.SrcMsg, Chat: &tele.Chat{ID: job.SrcChat}}

	send := func() (*tele.Message, error) {
		if job.Mode == "forward" {
			return h.Bot.Forward(to, src)
		}
		return h.Bot.Copy(to, src)
	}

	msg, err := send()
	if err == nil {
		return msg, nil
	}
	// رعایت محدودیت تلگرام (429): صبر به‌اندازه‌ی RetryAfter و تلاش مجدد.
	var fld *tele.FloodError
	if errors.As(err, &fld) && fld.RetryAfter > 0 {
		time.Sleep(time.Duration(fld.RetryAfter+1) * time.Second)
		return send()
	}
	return nil, err
}

// isBlockedErr تشخیص می‌دهد خطا به‌خاطر بلاک‌شدن/غیرفعال‌شدن کاربر است.
func isBlockedErr(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "blocked") ||
		strings.Contains(s, "deactivated") ||
		strings.Contains(s, "user is deactivated") ||
		strings.Contains(s, "chat not found") ||
		strings.Contains(s, "bot can't initiate")
}

// jobStatusText وضعیت متنی یک کار را برمی‌گرداند.
func jobStatusText(j *models.BroadcastJob) string {
	switch {
	case j.Canceled:
		return "⛔️ متوقف‌شده"
	case j.Done:
		return "✅ تمام‌شده"
	default:
		return "⏳ در حال ارسال"
	}
}

// adminBroadcastStatus فهرست آخرین همگانی‌ها را با دکمهٔ ورود به هرکدام نشان می‌دهد.
func (h *Handler) adminBroadcastStatus(ctx context.Context, c tele.Context) error {
	jobs, err := h.Store.ListBroadcastJobs(ctx, 8)
	h.LogErr("adminBroadcastStatus", err)
	if len(jobs) == 0 {
		return c.Edit("📭 هنوز هیچ ارسال همگانی‌ای انجام نشده.", kbBackHome())
	}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for i := range jobs {
		j := jobs[i]
		label := fmt.Sprintf("%s • %s • %d/%d", j.Code, jobStatusText(&j), j.Sent, j.Total)
		rows = append(rows, kb.Row(kb.Data(label, "bcjob:"+j.Code)))
	}
	rows = append(rows, kb.Row(kb.Data("🔄 بروزرسانی", "p:bcstat"), kb.Data(btnBackLabel, "p:home")))
	kb.Inline(rows...)
	return c.Edit("📊 <b>ارسال‌های همگانی اخیر</b>\nبرای جزئیات/کنترل، روی هرکدام بزنید:", tele.ModeHTML, kb)
}

// adminBroadcastJobView جزئیات و کنترل یک همگانی خاص (با کد).
func (h *Handler) adminBroadcastJobView(ctx context.Context, c tele.Context, code string) error {
	j, err := h.Store.FindBroadcastJobByCode(ctx, code)
	h.LogErr("FindBroadcastJobByCode", err)
	if j == nil {
		return c.Edit("❌ همگانی با این کد یافت نشد.", kbBackHome())
	}
	pending := h.Store.CountBroadcastMsgsByCode(ctx, code)
	text := fmt.Sprintf(
		"📢 <b>همگانی %s</b>\n🔖 کد: <code>%s</code>\n🧭 نوع: %s\n📝 %s\n\n"+
			"وضعیت: %s\n📨 ارسال‌شده: %d/%d\n🚫 بلاک: %d\n❌ ارور: %d\n⏰ باقی‌مانده: %d\n🗑 پیام‌های قابل حذف: %d",
		j.Code, j.Code, modeLabel(j.Mode), util.EscapeHTML(j.Preview),
		jobStatusText(j), j.Sent, j.Total, j.Blocked, j.Failed, j.Remaining(), pending)

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	rows = append(rows, kb.Row(kb.Data("🔄 بروزرسانی", "bcjob:"+code)))
	if !j.Done && !j.Canceled {
		rows = append(rows, kb.Row(kb.Data("⛔️ توقف ارسال", "bccancel:"+code)))
	}
	if pending > 0 {
		rows = append(rows, kb.Row(kb.Data("🗑 حذف همهٔ پیام‌ها از پیوی‌ها", "bcdelnow:"+code)))
	}
	rows = append(rows, kb.Row(kb.Data(btnBackLabel, "p:bcstat")))
	kb.Inline(rows...)
	return c.Edit(text, tele.ModeHTML, kb)
}

func modeLabel(m string) string {
	if m == "forward" {
		return "فوروارد"
	}
	return "کپی"
}

// adminBroadcastCancel یک همگانیِ در حال اجرا را متوقف می‌کند.
func (h *Handler) adminBroadcastCancel(ctx context.Context, c tele.Context, code string) error {
	j, err := h.Store.FindBroadcastJobByCode(ctx, code)
	h.LogErr("FindBroadcastJobByCode", err)
	if j == nil {
		return c.Respond(&tele.CallbackResponse{Text: "یافت نشد"})
	}
	h.Store.SetBroadcastJobCanceled(ctx, j.ID)
	h.LogErr("adminBroadcastCancel: respond", c.Respond(&tele.CallbackResponse{Text: "⛔️ درخواست توقف ثبت شد"}))
	return h.adminBroadcastJobView(ctx, c, code)
}

// adminBroadcastDeleteNow پیام‌های یک همگانی را همین حالا از پیوی همه حذف می‌کند.
func (h *Handler) adminBroadcastDeleteNow(ctx context.Context, c tele.Context, code string) error {
	h.LogErr("adminBroadcastDeleteNow: respond", c.Respond(&tele.CallbackResponse{Text: "🗑 حذف در پس‌زمینه شروع شد"}))
	go h.deleteBroadcastByCode(context.Background(), code)
	return h.adminBroadcastJobView(ctx, c, code)
}

func (h *Handler) deleteBroadcastByCode(ctx context.Context, code string) {
	for {
		msgs, err := h.Store.BroadcastMsgsByCode(ctx, code, 200)
		if err != nil || len(msgs) == 0 {
			return
		}
		for _, m := range msgs {
			err := h.Bot.Delete(&tele.Message{ID: m.MsgID, Chat: &tele.Chat{ID: m.ChatID}})
			var fld *tele.FloodError
			if err != nil && errors.As(err, &fld) && fld.RetryAfter > 0 {
				time.Sleep(time.Duration(fld.RetryAfter+1) * time.Second)
			}
			h.Store.DeleteBroadcastMsgRecord(ctx, m.ID)
			time.Sleep(40 * time.Millisecond)
		}
	}
}

// broadcastSweeper به‌صورت دوره‌ای پیام‌های همگانیِ سررسیده را از پیوی همه حذف می‌کند.
func (h *Handler) broadcastSweeper(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.sweepBroadcasts(ctx)
		}
	}
}

func (h *Handler) sweepBroadcasts(ctx context.Context) {
	msgs, err := h.Store.DueBroadcastMsgs(ctx, 200)
	if err != nil || len(msgs) == 0 {
		return
	}
	for _, m := range msgs {
		err := h.Bot.Delete(&tele.Message{ID: m.MsgID, Chat: &tele.Chat{ID: m.ChatID}})
		if err != nil {
			// رعایت محدودیت در حذف انبوه
			var fld *tele.FloodError
			if errors.As(err, &fld) && fld.RetryAfter > 0 {
				time.Sleep(time.Duration(fld.RetryAfter+1) * time.Second)
			} else {
				h.Log.Error("broadcast delete", ports.F("err", err))
			}
		}
		h.Store.DeleteBroadcastMsgRecord(ctx, m.ID)
		time.Sleep(40 * time.Millisecond)
	}
}
