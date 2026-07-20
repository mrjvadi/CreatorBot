// lockrental.go — اجاره‌ی قفل کانال روی ربات‌های رایگان پلتفرم.
//
// جریان کامل (طبق توضیح کاربر):
//  1. خریدار در ads-bot درخواست اجاره می‌دهد (کانال هدف + بودجه + پاداش هر عضو)
//  2. درخواست به ادمین اصلی پلتفرم (OWNER_ID) می‌رود — نه ادمین معمولی
//  3. بعد از تأیید: بودجه از کیف پول خریدار کسر می‌شود (botpay) و چند ربات
//     رایگان به این کمپین وصل می‌شوند (FreeBotSlot.RentalID) و "در اختیار"
//     خریدار قرار می‌گیرند
//  4. وقتی خریدار ربات را در کانال خودش ادمین کرد، IsChannelAdminConfirmed=true
//     می‌شود و از همان لحظه آن ربات شروع به قفل‌کردن برای این تبلیغ می‌کند
//  5. چک عضویت با member-bot انجام می‌شود (فاز ۳ این نقشه‌راه)
package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/google/uuid"
	"github.com/mrjvadi/creatorbot/ads-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// تعداد پیش‌فرض ربات‌هایی که به هر کمپین اجاره‌ای تخصیص می‌یابد.
// (می‌تواند بعداً بر اساس بودجه/تقاضا دینامیک شود.)
const defaultSlotsPerRental = 3

// ── شروع درخواست اجاره ────────────────────────────────────────

func (h *Handler) onRentLock(c tele.Context) error {
	ctx := context.Background()
	h.setStep(ctx, c.Sender().ID, stepRentChannel)
	return c.Send(
		"<b>🔒 اجاره‌ی قفل کانال روی ربات‌های رایگان</b>\n\n"+
			"آیدی عددی یا یوزرنیم کانالی که می‌خواهید کاربران عضوش شوند را وارد کنید:\n"+
			"(ربات‌های رایگان پلتفرم بعد از تأیید، کاربران را به این کانال هدایت می‌کنند)",
		tele.ModeHTML, kbCancel(),
	)
}

func (h *Handler) handleRentChannel(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	text = strings.TrimSpace(text)
	if text == "" {
		return c.Send("❌ کانال نامعتبر است.")
	}

	chatID, username, err := h.resolveChannel(text)
	if err != nil {
		return c.Send("❌ کانال پیدا نشد. مطمئن شوید ربات در کانال عضو/ادمین است و دوباره وارد کنید:")
	}

	h.setStep(ctx, uid, stepRentBudget,
		"channel_id", strconv.FormatInt(chatID, 10),
		"channel_username", username,
	)
	return c.Send("💰 بودجه‌ی کل (TON) را وارد کنید:\nاز کیف پول شما کسر و برای کاربرانی که از طریق ربات‌های رایگان عضو شوند پرداخت می‌شود.")
}

// resolveChannel ورودی کاربر (آیدی عددی یا @username) را به chatID واقعی
// تلگرام تبدیل می‌کند — لازم برای این‌که بعدا بشود membership.joined را
// به این کمپین خاص نسبت داد (فاز ۶: پرداخت per-join).
func (h *Handler) resolveChannel(input string) (chatID int64, username string, err error) {
	if id, perr := strconv.ParseInt(input, 10, 64); perr == nil {
		chat, gerr := h.bot.ChatByID(id)
		if gerr != nil {
			return 0, "", gerr
		}
		return chat.ID, chat.Username, nil
	}
	uname := strings.TrimPrefix(input, "@")
	chat, gerr := h.bot.ChatByUsername("@" + uname)
	if gerr != nil {
		return 0, "", gerr
	}
	return chat.ID, chat.Username, nil
}

func (h *Handler) handleRentBudget(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	budget, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || budget <= 0 {
		return c.Send("❌ عدد نامعتبر. دوباره وارد کنید:")
	}
	h.setStep(ctx, uid, stepRentReward,
		"channel_id", st.Data["channel_id"],
		"channel_username", st.Data["channel_username"],
		"budget", text,
	)
	return c.Send("🎁 به ازای هر عضو واقعی، چقدر (TON) به کاربر پرداخت شود؟")
}

func (h *Handler) handleRentReward(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	reward, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || reward <= 0 {
		return c.Send("❌ عدد نامعتبر. دوباره وارد کنید:")
	}
	budget, _ := strconv.ParseFloat(st.Data["budget"], 64)
	channelID, _ := strconv.ParseInt(st.Data["channel_id"], 10, 64)
	h.clearState(ctx, uid)

	rental := &store.LockRentalCampaign{
		ID:                        uuid.New(),
		BuyerTelegramID:           uid,
		TargetChannelID:           channelID,
		TargetChannelUsername:     st.Data["channel_username"],
		Status:                    store.RentalPendingReview,
		RewardPerJoinTON:          reward,
		Budget:                    budget,
		FreeBotOwnerRewardPercent: 5, // پیش‌فرض ۵٪ کل بودجه بین owner های ربات‌های رایگان
	}
	if err := h.store.CreateLockRental(ctx, rental); err != nil {
		h.log.Error("create lock rental", ports.F("err", err))
		return c.Send("❌ خطا در ثبت درخواست.")
	}

	h.notifyAdminRental(ctx, rental)
	return c.Send(fmt.Sprintf(
		"✅ درخواست شما ثبت شد.\n\n"+
			"📢 کانال: %s\n💰 بودجه: %.2f TON\n🎁 پاداش هر عضو: %.2f TON\n\n"+
			"⏳ منتظر تأیید ادمین اصلی پلتفرم بمانید.",
		rental.TargetChannelUsername, budget, reward,
	))
}

// notifyAdminRental درخواست را برای تأیید به ادمین اصلی پلتفرم می‌فرستد.
// طبق طراحی، این پیام فقط به OWNER_ID می‌رود نه به هر ادمین.
func (h *Handler) notifyAdminRental(ctx context.Context, r *store.LockRentalCampaign) {
	if h.ownerID == 0 {
		return
	}
	admin := &tele.Chat{ID: h.ownerID}
	text := fmt.Sprintf(
		"🔒 <b>درخواست اجاره‌ی قفل کانال</b>\n\n"+
			"👤 خریدار: <code>%d</code>\n📢 کانال: %s\n"+
			"💰 بودجه: %.2f TON\n🎁 پاداش هر عضو: %.2f TON\n\n🆔 <code>%s</code>",
		r.BuyerTelegramID, r.TargetChannelUsername, r.Budget, r.RewardPerJoinTON, r.ID,
	)
	_, _ = h.bot.Send(admin, text, tele.ModeHTML, kbRentReview(r.ID.String()))
}

// ── تأیید / رد توسط ادمین اصلی ─────────────────────────────────

// approveLockRental: کسر بودجه از کیف پول خریدار (با metadata شفاف) سپس
// تخصیص N ربات رایگان به این کمپین — این ربات‌ها از این لحظه "در اختیار" خریدار هستند.
func (h *Handler) approveLockRental(ctx context.Context, c tele.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Edit("❌ شناسه نامعتبر.")
	}
	rental, err := h.store.FindLockRental(ctx, id)
	if err != nil || rental == nil {
		return c.Edit("❌ یافت نشد.")
	}
	if rental.Status != store.RentalPendingReview {
		return c.Edit("⚠️ این درخواست قبلاً بررسی شده است.")
	}

	// ── کسر بودجه از کیف پول خریدار (شفاف، با metadata) ─────
	if h.pay != nil {
		meta := fmt.Sprintf(`{"type":"lock_rental","rental_id":%q,"channel":%q}`,
			rental.ID.String(), rental.TargetChannelUsername)
		_, err := h.pay.DeductWithMeta(ctx, rental.BuyerTelegramID, rental.Budget,
			"lock_rental:"+rental.ID.String(), rental.ID.String(), rental.ID.String(), meta)
		if err != nil {
			if natspayclient.IsInsufficientBalance(err) {
				return c.Edit("❌ موجودی خریدار کافی نیست. درخواست رد شد.")
			}
			h.log.Error("rental budget deduct failed", ports.F("err", err))
			return c.Edit("❌ خطا در کسر بودجه.")
		}
	}

	if err := h.store.ApproveLockRental(ctx, id, c.Sender().ID); err != nil {
		return c.Edit("❌ خطا در تأیید.")
	}

	// ── تخصیص ربات‌های رایگان به این کمپین ────────────────
	slots, err := h.store.AssignSlotsToRental(ctx, id, rental.BuyerTelegramID, defaultSlotsPerRental)
	if err != nil {
		h.log.Error("assign slots failed", ports.F("err", err))
	}

	// ── سهم owner های واقعی ربات‌های رایگان از کل بودجه ────
	// (جمعی تقسیم می‌شود بین تعداد slot هایی که واقعاً تخصیص یافتند،
	// نه per-join — طبق طراحی کاربر)
	h.payFreeBotOwners(ctx, rental, slots)

	// اطلاع به خریدار: ربات‌ها در اختیارش قرار گرفت
	buyer := &tele.Chat{ID: rental.BuyerTelegramID}
	_, _ = h.bot.Send(buyer, fmt.Sprintf(
		"🎉 <b>درخواست اجاره‌ی قفل تأیید شد!</b>\n\n"+
			"📦 %d ربات رایگان در اختیار شما قرار گرفت.\n\n"+
			"⚠️ مرحله‌ی بعد: این ربات‌ها را در کانال <b>%s</b> ادمین کنید "+
			"تا قفل‌کردن کاربران برای این تبلیغ شروع شود.",
		len(slots), rental.TargetChannelUsername,
	), tele.ModeHTML)

	return c.Edit(fmt.Sprintf("✅ تأیید شد. %d ربات تخصیص یافت.", len(slots)))
}

// ── پرداخت per-join لحظه‌ای (فاز ۶) ──────────────────────────

// HandleFraudDetected وقتی fraud-engine یک کاربر را برای فعالیت مشکوک در
// یک کانال خاص علامت می‌زند صدا زده می‌شود — اگر آن کاربر یک پاداش
// per-join هنوز تسویه‌نشده برای آن کانال داشته باشد، لغو می‌شود (پول هرگز
// واریز نمی‌شود و بودجه‌ی رزروشده به کمپین برمی‌گردد).
func (h *Handler) HandleFraudDetected(ctx context.Context, telegramID, channelID int64) {
	if err := h.store.ReversePendingRewardByUser(ctx, channelID, telegramID); err != nil {
		h.log.Error("reverse pending reward failed",
			ports.F("user", telegramID), ports.F("channel", channelID), ports.F("err", err))
		return
	}
	h.log.Info("checked/reversed pending reward after fraud detection",
		ports.F("user", telegramID), ports.F("channel", channelID))
}

// HandleMembershipJoined وقتی یک کاربر واقعاً عضو یک کانال هدف می‌شود
// (رویداد membership.joined از member-bot) صدا زده می‌شود. اگر آن کانال
// به یک کمپین اجاره‌ای فعال وصل باشد، پاداش per-join را به کاربر واریز
// و از بودجه‌ی باقی‌مانده کسر می‌کند.
func (h *Handler) HandleMembershipJoined(ctx context.Context, telegramID, channelID int64) {
	rental, err := h.store.FindActiveRentalByChannel(ctx, channelID)
	if err != nil {
		h.log.Error("find active rental failed", ports.F("err", err))
		return
	}
	if rental == nil {
		return // این کانال به هیچ کمپین اجاره‌ای وصل نیست — کار این هندلر نیست
	}
	if !rental.IsActive() {
		return // بودجه تمام شده یا منقضی شده
	}

	// idempotency در سطح دیتابیس: هر کاربر برای یک کمپین فقط یک بار
	// پاداش می‌گیرد (جلوگیری از پاداش مکرر در صورت event تکراری از NATS
	// یا چند instance هم‌زمان ads-bot)
	firstTime, err := h.store.TryRecordJoinReward(ctx, rental.ID, telegramID, rental.RewardPerJoinTON)
	if err != nil {
		h.log.Error("record join reward failed", ports.F("err", err))
		return
	}
	if !firstTime {
		return // قبلاً برای این کاربر در این کمپین ثبت شده
	}

	// ── کسر از بودجه‌ی کمپین همین لحظه (طبق طراحی کاربر) ────
	// توجه: این فقط Spent کمپین را بالا می‌برد — یعنی "هزینه از حساب
	// خریدار همان لحظه کسر شده محسوب می‌شود" (خریدار قبلاً کل بودجه را
	// موقع تأیید پرداخته بود؛ این فقط مصرف آن بودجه را ثبت می‌کند).
	// واریز واقعی به کیف پول کاربر، طبق طراحی، با تأخیر (RewardSettlementDelay)
	// در scheduler انجام می‌شود — نه همین‌جا.
	if err := h.store.AddRentalJoinCount(ctx, rental.ID, 1, rental.RewardPerJoinTON); err != nil {
		h.log.Error("rental join count update failed", ports.F("err", err))
	}

	h.log.Info("join reward reserved (pending settlement)",
		ports.F("user", telegramID), ports.F("rental_id", rental.ID.String()),
		ports.F("amount", rental.RewardPerJoinTON))

	_, _ = h.bot.Send(&tele.User{ID: telegramID}, fmt.Sprintf(
		"🎉 پاداش عضویت ثبت شد!\n\n"+
			"💵 %.4f TON\n⏳ این پاداش پس از %d ساعت به کیف پول شما واریز می‌شود.",
		rental.RewardPerJoinTON, int(store.RewardSettlementDelay.Hours()),
	))

	// ── بررسی اتمام کمپین (بودجه تمام شد) ──────────────────
	h.checkCampaignCompletion(ctx, rental.ID)
}

// checkCampaignCompletion بررسی می‌کند آیا کمپین به‌تازگی به پایان رسیده
// (بودجه تمام شده یا منقضی شده) و اگر بله، به خریدار و owner های ربات‌های
// رایگان متصل اطلاع می‌دهد. اتمیک: حتی اگر چند join هم‌زمان این شرط را
// true کنند، فقط یکی واقعا اعلام می‌فرستد (MarkRentalDoneIfFinished).
func (h *Handler) checkCampaignCompletion(ctx context.Context, rentalID uuid.UUID) {
	justFinished, err := h.store.MarkRentalDoneIfFinished(ctx, rentalID)
	if err != nil {
		h.log.Error("mark rental done check failed", ports.F("err", err))
		return
	}
	if !justFinished {
		return
	}

	rental, err := h.store.FindLockRental(ctx, rentalID)
	if err != nil || rental == nil {
		return
	}

	h.log.Info("rental campaign finished", ports.F("rental_id", rentalID.String()))

	_, _ = h.bot.Send(&tele.User{ID: rental.BuyerTelegramID}, fmt.Sprintf(
		"🏁 <b>کمپین اجاره‌ی قفل به پایان رسید!</b>\n\n"+
			"📢 کانال: %s\n💰 مصرف‌شده: %.4f از %.4f TON\n👥 تعداد عضو واقعی: %d\n\n"+
			"ربات‌های رایگان دیگر برای این تبلیغ کاربر جذب نمی‌کنند.",
		rental.TargetChannelUsername, rental.Spent, rental.Budget, rental.RealJoins,
	), tele.ModeHTML)

	slots, err := h.store.ListSlotsByRental(ctx, rentalID)
	if err != nil {
		h.log.Error("list slots for finished rental failed", ports.F("err", err))
		return
	}
	for _, slot := range slots {
		ownerTgID, err := h.store.ResolveSlotOwnerTelegramID(ctx, slot.BotInstanceID)
		if err != nil || ownerTgID == 0 {
			continue
		}
		_, _ = h.bot.Send(&tele.User{ID: ownerTgID}, fmt.Sprintf(
			"ℹ️ تبلیغی که ربات رایگان شما برایش کار می‌کرد به پایان رسید.\n"+
				"ربات شما به‌زودی به حالت آزاد یا تبلیغ بعدی برمی‌گردد.",
		))
		// آزاد کردن slot برای کمپین بعدی
		if err := h.store.ReleaseSlot(ctx, slot.ID); err != nil {
			h.log.Error("release slot after rental finished failed",
				ports.F("slot_id", slot.ID), ports.F("err", err))
		}
	}
}

// RunSettlementScheduler هر چند دقیقه پاداش‌های pending که مهلتشان رسیده
// را تسویه می‌کند — واریز واقعی به کیف پول کاربر، با تأخیر طبق طراحی
// (RewardSettlementDelay) تا فرصت تشخیص تقلب باشد.
func (h *Handler) RunSettlementScheduler(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	h.log.Info("join-reward settlement scheduler started")

	// یک اجرای فوری در شروع تا منتظر اولین tick نمانیم
	h.settleDueRewards(ctx)
	h.settleDueOwnerRewards(ctx)
	h.checkExpiredRentals(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.settleDueRewards(ctx)
			h.settleDueOwnerRewards(ctx)
			h.checkExpiredRentals(ctx)
		}
	}
}

// checkExpiredRentals کمپین‌هایی که با گذشت زمان (نه اتمام بودجه) باید
// تمام شوند را پیدا و به پایان می‌رساند — لازم چون اگر هیچ join جدیدی
// نیاید، checkCampaignCompletion هیچ‌وقت trigger نمی‌شود.
func (h *Handler) checkExpiredRentals(ctx context.Context) {
	expired, err := h.store.FindExpiredActiveRentals(ctx)
	if err != nil {
		h.log.Error("find expired rentals failed", ports.F("err", err))
		return
	}
	for _, r := range expired {
		h.checkCampaignCompletion(ctx, r.ID)
	}
}

func (h *Handler) settleDueRewards(ctx context.Context) {
	if h.pay == nil {
		return
	}
	due, err := h.store.FindDueRewards(ctx, 200)
	if err != nil {
		h.log.Error("find due rewards failed", ports.F("err", err))
		return
	}

	for _, r := range due {
		meta := fmt.Sprintf(`{"type":"join_reward_settlement","rental_id":%q,"reward_id":%q}`,
			r.RentalID.String(), r.ID.String())
		ref := "join_reward:" + r.RentalID.String() + ":" + r.ID.String()

		if err := h.pay.Credit(ctx, r.TelegramID, r.AmountTON, ref, meta); err != nil {
			h.log.Error("settlement credit failed",
				ports.F("reward_id", r.ID.String()), ports.F("err", err))
			continue // دفعه‌ی بعد دوباره تلاش می‌شود (هنوز pending مانده)
		}
		if err := h.store.SettleReward(ctx, r.ID); err != nil {
			h.log.Error("mark settled failed", ports.F("reward_id", r.ID.String()), ports.F("err", err))
			continue
		}

		_, _ = h.bot.Send(&tele.User{ID: r.TelegramID}, fmt.Sprintf(
			"💰 پاداش عضویت شما واریز شد!\n\n💵 %.4f TON به کیف پول شما اضافه شد.", r.AmountTON,
		))
	}

	if len(due) > 0 {
		h.log.Info("join rewards settled", ports.F("count", len(due)))
	}
}

// settleDueOwnerRewards مشابه settleDueRewards ولی برای سهم owner های
// ربات‌های رایگان.
func (h *Handler) settleDueOwnerRewards(ctx context.Context) {
	if h.pay == nil {
		return
	}
	due, err := h.store.FindDueOwnerRewards(ctx, 200)
	if err != nil {
		h.log.Error("find due owner rewards failed", ports.F("err", err))
		return
	}

	for _, r := range due {
		meta := fmt.Sprintf(`{"type":"freebot_owner_reward_settlement","rental_id":%q,"slot_id":%q}`,
			r.RentalID.String(), r.SlotID.String())
		ref := "freebot_reward:" + r.RentalID.String() + ":" + r.SlotID.String()

		if err := h.pay.Credit(ctx, r.OwnerTelegramID, r.AmountTON, ref, meta); err != nil {
			h.log.Error("owner settlement credit failed",
				ports.F("reward_id", r.ID.String()), ports.F("err", err))
			continue
		}
		if err := h.store.SettleOwnerReward(ctx, r.ID); err != nil {
			h.log.Error("mark owner reward settled failed",
				ports.F("reward_id", r.ID.String()), ports.F("err", err))
			continue
		}

		_, _ = h.bot.Send(&tele.User{ID: r.OwnerTelegramID}, fmt.Sprintf(
			"💰 درآمد ربات رایگان شما واریز شد!\n\n💵 %.4f TON به کیف پول شما اضافه شد.", r.AmountTON,
		))
	}

	if len(due) > 0 {
		h.log.Info("owner rewards settled", ports.F("count", len(due)))
	}
}

// payFreeBotOwners سهم owner های واقعی ربات‌های رایگان را از کل بودجه‌ی
// کمپین محاسبه و رزرو می‌کند (طبق همان مدل escrow per-join: کسر همان لحظه
// از بودجه‌ی کمپین، واریز واقعی به owner با تأخیر RewardSettlementDelay).
func (h *Handler) payFreeBotOwners(ctx context.Context, rental *store.LockRentalCampaign, slots []store.FreeBotSlot) {
	if len(slots) == 0 || rental.FreeBotOwnerRewardPercent <= 0 {
		return
	}

	totalPool := rental.Budget * rental.FreeBotOwnerRewardPercent / 100
	perOwner := totalPool / float64(len(slots))
	if perOwner <= 0 {
		return
	}

	// کسر یکجای کل پول owner ها از بودجه‌ی کمپین — یک بار، نه per-slot
	// (جلوگیری از کسر چندباره/ناهماهنگ اگر چند owner داشته باشیم)
	if err := h.store.IncrementRentalSpentOnly(ctx, rental.ID, totalPool); err != nil {
		h.log.Error("increment rental spent for owner pool failed", ports.F("err", err))
	}

	reservedCount := 0
	for _, slot := range slots {
		ownerTgID, err := h.store.ResolveSlotOwnerTelegramID(ctx, slot.BotInstanceID)
		if err != nil || ownerTgID == 0 {
			h.log.Warn("free bot owner not resolved — reward skipped",
				ports.F("slot_id", slot.ID), ports.F("err", err))
			continue
		}

		firstTime, err := h.store.TryRecordOwnerReward(ctx, rental.ID, slot.ID, ownerTgID, perOwner)
		if err != nil {
			h.log.Error("record owner reward failed", ports.F("err", err))
			continue
		}
		if !firstTime {
			continue // قبلاً برای این slot در این کمپین رزرو شده
		}
		reservedCount++

		// اطلاع به owner ربات رایگان (پول هنوز واریز نشده — فقط رزرو شده)
		_, _ = h.bot.Send(&tele.Chat{ID: ownerTgID}, fmt.Sprintf(
			"💰 <b>درآمد از ربات رایگان شما!</b>\n\n"+
				"ربات شما برای یک تبلیغ اجاره‌ای استفاده شد.\n"+
				"💵 سهم شما: <b>%.4f TON</b>\n"+
				"⏳ پس از %d ساعت به کیف پول شما واریز می‌شود.",
			perOwner, int(store.RewardSettlementDelay.Hours()),
		), tele.ModeHTML)
	}

	h.log.Info("free bot owner rewards reserved (pending settlement)",
		ports.F("rental_id", rental.ID.String()),
		ports.F("reserved", reservedCount), ports.F("total_pool", totalPool))
}

func (h *Handler) rejectLockRental(ctx context.Context, c tele.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Edit("❌ شناسه نامعتبر.")
	}
	if err := h.store.RejectLockRental(ctx, id, c.Sender().ID, "رد شده توسط ادمین"); err != nil {
		return c.Edit("❌ خطا.")
	}

	rental, _ := h.store.FindLockRental(ctx, id)
	if rental != nil {
		buyer := &tele.Chat{ID: rental.BuyerTelegramID}
		_, _ = h.bot.Send(buyer, "❌ درخواست اجاره‌ی قفل کانال شما رد شد.")
	}
	return c.Edit("❌ رد شد.")
}

// ── تأیید ادمین‌شدن ربات در کانال خریدار ────────────────────────

// ConfirmChannelAdminByBotID وقتی bot فرعی (مثلا uploader) تشخیص داد که در
// کانال هدف ادمین شده، این را صدا می‌زند. از این لحظه قفل‌کردن شروع می‌شود.
// (این تابع از طریق NATS responder در فاز بعد صدا زده خواهد شد.)
func (h *Handler) ConfirmChannelAdminByBotID(ctx context.Context, botID int64) error {
	slot, err := h.store.FindFreeBotSlotByBotID(ctx, botID)
	if err != nil || slot == nil {
		return fmt.Errorf("slot not found for bot_id=%d", botID)
	}
	if slot.IsFree() {
		return fmt.Errorf("bot_id=%d is not assigned to any rental", botID)
	}
	return h.store.ConfirmChannelAdmin(ctx, slot.ID)
}

// GetBotStatus برمی‌گرداند که آیا این bot (با BotID تلگرامیِ خودش) الان به
// یک کمپینِ اجاره‌ی قفلِ فعال وصل است — پاسخ به bot فرعیِ رایگانی که هنگام
// start و به‌صورت دوره‌ای می‌پرسد (رجوع protocol.SubjBotStatusCheck). این
// جایگزینِ کوئریِ مستقیمِ Postgres/bot_instances.lock_mode شد که با قطعِ
// Postgres از ربات‌های محصول از کار افتاده بود.
func (h *Handler) GetBotStatus(ctx context.Context, botID int64) (inCampaign bool, campaignID string, err error) {
	slot, err := h.store.FindFreeBotSlotByBotID(ctx, botID)
	if err != nil {
		return false, "", err
	}
	if slot == nil || slot.IsFree() {
		return false, "", nil
	}
	rental, err := h.store.FindLockRental(ctx, *slot.RentalID)
	if err != nil {
		return false, "", err
	}
	if rental == nil || !rental.IsActive() {
		return false, "", nil
	}
	return true, rental.ID.String(), nil
}
