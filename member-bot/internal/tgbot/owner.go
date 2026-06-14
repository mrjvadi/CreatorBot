package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/google/uuid"
	"github.com/mrjvadi/creatorbot/member-bot/internal/models"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ════════════════════════════════════════════════════════════
// قفل‌های من
// ════════════════════════════════════════════════════════════

func (h *Handler) onMyLocks(c tele.Context) error {
	ctx := context.Background()
	owner, _ := h.store.FindOwnerByTelegramID(ctx, c.Sender().ID)
	if owner == nil {
		return c.Send("ابتدا /start بزنید.")
	}

	locks, _ := h.store.FindLocksByOwnerID(ctx, owner.ID)
	if len(locks) == 0 {
		return c.Send("هیچ قفلی ندارید.\n\nبرای ساخت قفل جدید: /newlock", kbMain())
	}

	for _, l := range locks {
		err := c.Send(fmtLock(l), tele.ModeHTML, kbLockActions(l.ID.String()))
		if err != nil {
			h.log.Error("onMyLocks send", ports.F("err", err))
		}
	}
	return nil
}

// ════════════════════════════════════════════════════════════
// ساخت قفل جدید
// ════════════════════════════════════════════════════════════

func (h *Handler) onNewLock(c tele.Context) error {
	ctx := context.Background()
	h.setStep(ctx, c.Sender().ID, stepLockChannel)
	return c.Send(
		"<b>➕ قفل جدید</b>\n\n"+
			"یک پیام از کانال مورد نظر را forward کنید.\n\n"+
			"⚠️ ربات باید ادمین کانال باشد.",
		tele.ModeHTML, kbCancel(),
	)
}

// handleChannelForward پیام forward شده از کانال را پردازش می‌کند.
func (h *Handler) handleChannelForward(ctx context.Context, c tele.Context, st wizardState) error {
	uid := c.Sender().ID
	msg := c.Message()

	if msg.OriginalSender == nil && msg.OriginalChat == nil {
		return c.Send("لطفاً یک پیام از کانال forward کنید.")
	}

	var channelID int64
	var channelTitle string

	if msg.OriginalChat != nil {
		channelID = msg.OriginalChat.ID
		channelTitle = msg.OriginalChat.Title
	} else {
		return c.Send("پیام ارسالی از کانال نیست.")
	}

	// بررسی قفل تکراری
	existing, _ := h.store.FindLockByChannelID(ctx, channelID)
	if existing != nil {
		return c.Send(fmt.Sprintf(
			"⚠️ این کانال قبلاً قفل شده است.\nID: <code>%s</code>",
			existing.ID,
		), tele.ModeHTML, kbMain())
	}

	h.setStep(ctx, uid, stepLockDuration,
		"channel_id", strconv.FormatInt(channelID, 10),
		"channel_title", channelTitle,
	)

	return c.Send(
		fmt.Sprintf(
			"کانال: <b>%s</b>\nID: <code>%d</code>\n\n"+
				"مدت قفل را به روز وارد کنید:\nمثال: <code>30</code>",
			channelTitle, channelID,
		),
		tele.ModeHTML, kbCancel(),
	)
}

func (h *Handler) handleLockDuration(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	days, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || days <= 0 || days > 365 {
		return c.Send("عدد صحیح بین ۱ تا ۳۶۵ وارد کنید.")
	}
	h.setStep(ctx, uid, stepLockPrice,
		"channel_id", st.Data["channel_id"],
		"channel_title", st.Data["channel_title"],
		"duration_day", strconv.Itoa(days),
	)
	return c.Send(
		fmt.Sprintf(
			"مدت: <b>%d روز</b>\n\nقیمت روزانه را وارد کنید (تومان):\nمثال: <code>500</code>",
			days,
		),
		tele.ModeHTML, kbCancel(),
	)
}

func (h *Handler) handleLockPrice(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	price, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || price < 0 {
		return c.Send("مبلغ نامعتبر.")
	}

	owner, _ := h.store.FindOwnerByTelegramID(ctx, uid)
	if owner == nil {
		return c.Send(h.t("error"))
	}

	channelID, _ := strconv.ParseInt(st.Data["channel_id"], 10, 64)
	days, _ := strconv.Atoi(st.Data["duration_day"])

	lock := &models.Lock{
		OwnerID:      owner.ID,
		ChannelID:    channelID,
		ChannelTitle: st.Data["channel_title"],
		DurationDay:  days,
		PricePerDay:  price,
		Status:       models.LockActive,
		ExpiresAt:    time.Now().AddDate(0, 0, days),
	}
	if err := h.store.CreateLock(ctx, lock); err != nil {
		return c.Send("❌ خطا در ساخت قفل.")
	}

	return c.Send(
		fmt.Sprintf(
			"✅ <b>قفل ساخته شد!</b>\n\n"+
				"📢 کانال: %s\n"+
				"⏳ مدت: %d روز\n"+
				"💰 قیمت: %.0f تومان/روز\n"+
				"🆔 ID: <code>%s</code>\n\n"+
				"حالا bot های check خود را به کانال اضافه کنید.",
			lock.ChannelTitle, lock.DurationDay, lock.PricePerDay, lock.ID,
		),
		tele.ModeHTML, kbMain(),
	)
}

func (h *Handler) pauseLock(ctx context.Context, c tele.Context, lockIDStr string) error {
	lockID, _ := uuid.Parse(lockIDStr)
	h.store.ExpireLock(ctx, lockID)
	return c.Edit("⏸ قفل متوقف شد.")
}

func (h *Handler) deleteLock(ctx context.Context, c tele.Context, lockIDStr string) error {
	lockID, _ := uuid.Parse(lockIDStr)
	h.store.DeleteLock(ctx, lockID)
	return c.Edit("🗑 قفل حذف شد.")
}

// ════════════════════════════════════════════════════════════
// Check Bot ها
// ════════════════════════════════════════════════════════════

func (h *Handler) onMyBots(c tele.Context) error {
	ctx := context.Background()
	bots, err := h.store.FindActiveBots(ctx)
	if err != nil {
		return c.Send("❌ خطا.")
	}
	if len(bots) == 0 {
		return c.Send(
			"هیچ check bot ای وجود ندارد.\n\nبرای افزودن: /addbot",
			kbMain(),
		)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🤖 Check Bot ها (%d)</b>\n\n", len(bots)))
	for _, b := range bots {
		active := "✅"
		if !b.IsActive {
			active = "❌"
		}
		channels := len(b.Memberships)
		sb.WriteString(fmt.Sprintf(
			"%s @%s — %d کانال — %d req/s\n",
			active, b.Username, channels, b.RateLimit,
		))
	}
	return c.Send(sb.String(), tele.ModeHTML, kbMain())
}

func (h *Handler) onAddBot(c tele.Context) error {
	ctx := context.Background()
	h.setStep(ctx, c.Sender().ID, stepAddBot)
	return c.Send(
		"<b>🤖 افزودن Check Bot</b>\n\n"+
			"توکن bot جدید را وارد کنید:\n"+
			"(از @BotFather دریافت کنید)\n\n"+
			"⚠️ bot باید عضو کانال‌های قفل شده باشد.",
		tele.ModeHTML, kbCancel(),
	)
}

func (h *Handler) handleBotToken(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	token := strings.TrimSpace(text)
	if len(token) < 40 || !strings.Contains(token, ":") {
		return c.Send("❌ توکن نامعتبر.")
	}

	// استخراج bot username — تست توکن
	botID, err := models.BotIDFromToken(token)
	if err != nil {
		return c.Send("❌ فرمت توکن نادرست.")
	}

	// رمزنگاری توکن
	encToken, err := auth.Encrypt(token, h.encryptKey)
	if err != nil {
		return c.Send("❌ خطا در رمزنگاری.")
	}

	bot := &models.CheckBot{
		Token:     encToken,
		Username:  fmt.Sprintf("bot_%d", botID),
		IsActive:  true,
		RateLimit: 20,
	}
	if err := h.store.CreateCheckBot(ctx, bot); err != nil {
		return c.Send("❌ خطا در ثبت bot.")
	}

	return c.Send(
		fmt.Sprintf(
			"✅ <b>Check Bot افزوده شد</b>\n\n"+
				"🆔 ID: <code>%s</code>\n\n"+
				"Bot را به کانال‌های قفل خود اضافه کنید.",
			bot.ID,
		),
		tele.ModeHTML, kbMain(),
	)
}

// ════════════════════════════════════════════════════════════
// موجودی
// ════════════════════════════════════════════════════════════

func (h *Handler) onBalance(c tele.Context) error {
	ctx := context.Background()
	owner, _ := h.store.FindOwnerByTelegramID(ctx, c.Sender().ID)
	if owner == nil {
		return c.Send("ابتدا /start بزنید.")
	}
	return c.Send(
		fmt.Sprintf(
			"<b>💰 موجودی</b>\n\nموجودی: <b>%.0f تومان</b>",
			owner.Balance,
		),
		tele.ModeHTML, kbMain(),
	)
}

// ════════════════════════════════════════════════════════════
// helpers
// ════════════════════════════════════════════════════════════

func (h *Handler) createOwner(ctx context.Context, c tele.Context) (*models.Owner, error) {
	o := &models.Owner{
		TelegramID: c.Sender().ID,
		Username:   c.Sender().Username,
		FirstName:  c.Sender().FirstName,
	}
	return o, h.store.CreateOwner(ctx, o)
}

func (h *Handler) t(key string) string {
	texts := map[string]string{
		"error": "❌ خطا. دوباره امتحان کنید.",
	}
	if v, ok := texts[key]; ok {
		return v
	}
	return key
}
