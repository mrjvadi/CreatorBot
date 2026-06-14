package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/google/uuid"
	"github.com/mrjvadi/creatorbot/ads-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ── ساخت کمپین ────────────────────────────────────────────

func (h *Handler) onNewCampaign(c tele.Context) error {
	ctx := context.Background()
	h.setStep(ctx, c.Sender().ID, stepCampName)
	return c.Send(
		"<b>➕ کمپین جدید</b>\n\nنام کمپین را وارد کنید:",
		tele.ModeHTML, kbCancel(),
	)
}

func (h *Handler) handleCampName(ctx context.Context, c tele.Context, st wizardState, text string) error {
	if len(text) < 2 {
		return c.Send("نام خیلی کوتاه است.")
	}
	h.setStep(ctx, c.Sender().ID, stepCampMedia, "name", text)
	return c.Send(
		"محتوای تبلیغ را ارسال کنید:\n\n"+
			"📸 عکس\n🎥 ویدیو\n📝 یا فقط متن (ادامه با دکمه رد کردن)",
		kbSkipCancel(),
	)
}

func (h *Handler) handleCampaignMedia(ctx context.Context, c tele.Context, st wizardState) error {
	uid := c.Sender().ID
	m := c.Message()
	var fileID, mediaType string

	switch {
	case m.Photo != nil:
		fileID = m.Photo.FileID
		mediaType = "photo"
	case m.Video != nil:
		fileID = m.Video.FileID
		mediaType = "video"
	}

	h.setStep(ctx, uid, stepCampCaption,
		"name", st.Data["name"],
		"file_id", fileID,
		"media_type", mediaType,
	)
	return c.Send("متن/caption تبلیغ:", kbSkipCancel())
}

func (h *Handler) handleCampCaption(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	caption := ""
	if text != btnSkip {
		caption = text
	}
	h.setStep(ctx, uid, stepCampButton,
		"name", st.Data["name"],
		"file_id", st.Data["file_id"],
		"media_type", st.Data["media_type"],
		"caption", caption,
	)
	return c.Send("متن دکمه لینک (اختیاری):\nمثال: <code>🔗 بیشتر بدانید</code>",
		tele.ModeHTML, kbSkipCancel())
}

func (h *Handler) handleCampButton(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	btnText := ""
	if text != btnSkip {
		btnText = text
		h.setStep(ctx, uid, stepCampURL,
			"name", st.Data["name"],
			"file_id", st.Data["file_id"],
			"media_type", st.Data["media_type"],
			"caption", st.Data["caption"],
			"btn_text", btnText,
		)
		return c.Send("لینک دکمه:", kbCancel())
	}
	// بدون دکمه → برو به بودجه
	h.setStep(ctx, uid, stepCampBudget,
		"name", st.Data["name"],
		"file_id", st.Data["file_id"],
		"media_type", st.Data["media_type"],
		"caption", st.Data["caption"],
	)
	return c.Send("بودجه کل (TON):\nمثال: <code>5.0</code>", tele.ModeHTML, kbCancel())
}

func (h *Handler) handleCampURL(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	if !strings.HasPrefix(text, "http") {
		return c.Send("❌ لینک نامعتبر. باید با http یا https شروع شود.")
	}
	h.setStep(ctx, uid, stepCampBudget,
		"name", st.Data["name"],
		"file_id", st.Data["file_id"],
		"media_type", st.Data["media_type"],
		"caption", st.Data["caption"],
		"btn_text", st.Data["btn_text"],
		"btn_url", text,
	)
	return c.Send("بودجه کل (TON):\nمثال: <code>5.0</code>", tele.ModeHTML, kbCancel())
}

func (h *Handler) handleCampBudget(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	budget, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || budget <= 0 {
		return c.Send("❌ مبلغ نامعتبر.")
	}
	// بررسی موجودی
	pub, _ := h.store.FindPublisher(ctx, uid)
	if pub != nil && pub.Balance < budget {
		return c.Send(fmt.Sprintf(
			"❌ موجودی کافی نیست.\nموجودی: %.4f TON | نیاز: %.4f TON",
			pub.Balance, budget,
		))
	}
	h.setStep(ctx, uid, stepCampCPJ,
		"name", st.Data["name"],
		"file_id", st.Data["file_id"],
		"media_type", st.Data["media_type"],
		"caption", st.Data["caption"],
		"btn_text", st.Data["btn_text"],
		"btn_url", st.Data["btn_url"],
		"budget", text,
	)
	return c.Send(
		"Cost Per Join (CPJ) — هزینه جذب هر عضو جدید (TON):\n"+
			"مثال: <code>0.01</code> یعنی به ازای هر عضو ۰.۰۱ TON",
		tele.ModeHTML, kbCancel(),
	)
}

func (h *Handler) handleCampCPJ(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	cpj, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || cpj <= 0 {
		return c.Send("❌ مقدار نامعتبر.")
	}
	budget, _ := strconv.ParseFloat(st.Data["budget"], 64)

	pub, _ := h.store.FindPublisher(ctx, uid)
	if pub == nil {
		return c.Send(h.t("error"))
	}

	camp := &store.Campaign{
		PublisherID: pub.ID,
		Name:        st.Data["name"],
		MediaFileID: st.Data["file_id"],
		MediaType:   st.Data["media_type"],
		Caption:     st.Data["caption"],
		ButtonText:  st.Data["btn_text"],
		ButtonURL:   st.Data["btn_url"],
		Budget:      budget,
		CPJ:         cpj,
		Status:      store.CampaignPending,
		TargetCount: int(budget / cpj),
	}

	if err := h.store.CreateCampaign(ctx, camp); err != nil {
		h.log.Error("createCampaign", ports.F("err", err))
		return c.Send("❌ خطا در ساخت کمپین.")
	}

	// block بودجه از موجودی
	h.store.UpdateBalance(ctx, pub.ID, -budget)

	// اطلاع به ادمین
	h.notifyAdmin(ctx, camp)

	return c.Send(
		fmt.Sprintf(
			"✅ <b>کمپین ارسال شد</b>\n\n"+
				"📌 %s\n"+
				"💰 بودجه: %.2f TON\n"+
				"🎯 CPJ: %.3f TON\n"+
				"👥 هدف: %d عضو\n\n"+
				"⏳ منتظر تأیید ادمین...",
			camp.Name, camp.Budget, camp.CPJ, camp.TargetCount,
		),
		tele.ModeHTML, kbMain(),
	)
}

// ── لیست کمپین‌ها ─────────────────────────────────────────

func (h *Handler) onMyCampaigns(c tele.Context) error {
	ctx := context.Background()
	pub, _ := h.store.FindPublisher(ctx, c.Sender().ID)
	if pub == nil {
		return c.Send("ابتدا /start بزنید.")
	}

	camps, _ := h.store.FindCampaignsByPublisher(ctx, pub.ID)
	if len(camps) == 0 {
		return c.Send("هیچ کمپینی ندارید.", kbMain())
	}

	for _, camp := range camps {
		cp := camp
		err := c.Send(fmtCampaign(cp), tele.ModeHTML, kbCampaignActions(cp.ID.String()))
		if err != nil {
			h.log.Error("onMyCampaigns send", ports.F("err", err))
		}
	}
	return nil
}

func (h *Handler) pauseCampaign(ctx context.Context, c tele.Context, campIDStr string) error {
	campID, _ := uuid.Parse(campIDStr)
	camp, _ := h.store.FindCampaign(ctx, campID)
	if camp == nil { return c.Edit("❌ یافت نشد.") }
	if camp.Status == store.CampaignActive {
		camp.Status = store.CampaignPaused
	} else {
		camp.Status = store.CampaignActive
	}
	h.store.UpdateCampaign(ctx, camp)
	return c.Edit(fmtCampaign(*camp), tele.ModeHTML, kbCampaignActions(campIDStr))
}

func (h *Handler) deleteCampaign(ctx context.Context, c tele.Context, campIDStr string) error {
	campID, _ := uuid.Parse(campIDStr)
	camp, _ := h.store.FindCampaign(ctx, campID)
	if camp == nil { return c.Edit("❌ یافت نشد.") }

	// بازگشت بودجه باقی‌مانده
	pub, _ := h.store.FindPublisher(ctx, c.Sender().ID)
	if pub != nil {
		h.store.UpdateBalance(ctx, pub.ID, camp.RemainingBudget())
	}

	camp.Status = store.CampaignDone
	h.store.UpdateCampaign(ctx, camp)
	return c.Edit("🗑 کمپین متوقف و بودجه باقی‌مانده برگشت داده شد.")
}

func (h *Handler) t(key string) string {
	m := map[string]string{"error": "❌ خطا. دوباره امتحان کنید."}
	if v, ok := m[key]; ok { return v }
	return key
}

func (h *Handler) notifyAdmin(ctx context.Context, camp *store.Campaign) {
	if h.ownerID == 0 { return }
	admin := &tele.Chat{ID: h.ownerID}
	text := fmt.Sprintf(
		"📣 <b>کمپین جدید</b>\n\n📌 %s\n💰 %.2f TON | CPJ: %.3f\n🆔 <code>%s</code>",
		camp.Name, camp.Budget, camp.CPJ, camp.ID,
	)
	h.bot.Send(admin, text, tele.ModeHTML, kbReview(camp.ID.String()))
}
