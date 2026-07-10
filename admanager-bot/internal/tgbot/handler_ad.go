// handler_ad.go — مدیریت تبلیغ‌ها (پست اصلی + ریپلی‌های متوالی) با Copy.
package tgbot

import (
	"context"
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

// ── لیست تبلیغ‌های یک کمپین ──────────────────────────────────────

func (h *Handler) adList(c tele.Context, campaignID string) error {
	ctx := context.Background()
	ads, _ := h.store.ListAdsByCampaign(ctx, campaignID)

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, ad := range ads {
		label := fmt.Sprintf("%s%s (%d ریپلی)", fixedBadge(ad), ad.Name, len(ad.Replies))
		rows = append(rows, kb.Row(cbBtn(kb, label, "ad_view:"+ad.ID)))
	}
	rows = append(rows, kb.Row(cbBtn(kb, "➕ افزودن محتوای جدید", "ad_new:"+campaignID)))
	rows = append(rows, kb.Row(cbBtn(kb, "🔙 بازگشت", "cmp_view:"+campaignID)))
	kb.Inline(rows...)

	text := "📣 <b>تبلیغ‌های کمپین</b>\nهر تبلیغ یک پست اصلی دارد که می‌تواند چند ریپلیِ متوالی هم داشته باشد."
	if len(ads) == 0 {
		text += "\n\nهنوز تبلیغی برای این کمپین ندارید."
	}
	return c.Edit(text, tele.ModeHTML, kb)
}

// fixedBadge نشانه‌ی کوچک جلوی نامِ تبلیغ‌های «ثابت»/«پین» در لیست‌ها.
func fixedBadge(ad models.Advertisement) string {
	switch {
	case ad.KeepAsLastMessage && ad.PinMessage:
		return "📌🔽 "
	case ad.PinMessage:
		return "📌 "
	case ad.KeepAsLastMessage:
		return "🔽 "
	}
	return "📣 "
}

// ── ➕ افزودن محتوای جدید: پرسیدن نقشِ محتوا (بخش ۲.۳) ─────────────

// adNewMenu از ادمین می‌پرسد محتوای تازه چه نقشی دارد: تبلیغ مستقل جدید،
// ریپلی روی یک تبلیغ موجود، یا یک پست ساده بدون ریپلی.
func (h *Handler) adNewMenu(c tele.Context, campaignID string) error {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(cbBtn(kb, "🆕 تبلیغ مستقل جدید", "ad_new_std:"+campaignID)),
		kb.Row(cbBtn(kb, "↩️ ریپلی روی تبلیغ موجود", "ad_new_atc:"+campaignID)),
		kb.Row(cbBtn(kb, "📨 ارسال ساده (بدون ریپلی)", "ad_new_sim:"+campaignID)),
		kb.Row(cbBtn(kb, "🔙 بازگشت", "ad_list:"+campaignID)),
	)
	text := "➕ <b>افزودن محتوای جدید</b>\nاین محتوا چه نقشی داشته باشد؟\n\n" +
		"🆕 <b>تبلیغ مستقل جدید</b> — پست اصلیِ تازه، با ریپلی‌های خودش\n" +
		"↩️ <b>ریپلی روی تبلیغ موجود</b> — به زنجیره‌ی ریپلی‌های یکی از تبلیغ‌های همین کمپین اضافه شود\n" +
		"📨 <b>ارسال ساده</b> — فقط یک پست، بدون ریپلی"
	return c.Edit(text, tele.ModeHTML, kb)
}

// ── وضعیت مستقیم مشاهده‌ی تبلیغ ────────────────────────────────────

func (h *Handler) renderAdView(ad *models.Advertisement) (string, *tele.ReplyMarkup) {
	text := fmt.Sprintf(
		"📣 <b>%s</b>\n\n"+
			"تعداد ریپلی‌ها: %d\n"+
			"همیشه آخرین پیام کانال: %s\n"+
			"پین در کانال: %s\n"+
			"🆔 <code>%s</code>",
		ad.Name, len(ad.Replies), boolLabel(ad.KeepAsLastMessage), boolLabel(ad.PinMessage), ad.ID,
	)
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			cbBtn(kb, "✏️ نام", "ad_ename:"+ad.ID),
			cbBtn(kb, "🔄 پست اصلی", "ad_emain:"+ad.ID),
		),
		kb.Row(cbBtn(kb, fmt.Sprintf("📝 مدیریت ریپلی‌ها (%d)", len(ad.Replies)), "ad_replies:"+ad.ID)),
		kb.Row(cbBtn(kb, "⚙️ تنظیمات ثابت/پین", "ad_settings:"+ad.ID)),
		kb.Row(cbBtn(kb, "👁 پیش‌نمایش", "ad_preview:"+ad.ID)),
		kb.Row(cbBtn(kb, "🗑 حذف", "ad_del:"+ad.ID)),
		kb.Row(cbBtn(kb, "🔙 بازگشت", "ad_list:"+ad.CampaignID)),
	)
	return text, kb
}

func (h *Handler) adView(c tele.Context, id string) error {
	ctx := context.Background()
	ad, err := h.store.FindAd(ctx, id)
	if err != nil || ad == nil {
		return c.Edit("تبلیغ پیدا نشد.")
	}
	text, kb := h.renderAdView(ad)
	return c.Edit(text, tele.ModeHTML, kb)
}

// sendAdView مثل adView ولی به‌جای ویرایش پیام inline، پیام تازه می‌فرستد
// (برای وقتی که از یک مرحله‌ی متنی برمی‌گردیم، نه از یک callback).
func (h *Handler) sendAdView(c tele.Context, id string) error {
	ctx := context.Background()
	ad, err := h.store.FindAd(ctx, id)
	if err != nil || ad == nil {
		return c.Send("تبلیغ پیدا نشد.", kbAdminMain())
	}
	text, kb := h.renderAdView(ad)
	return c.Send(text, tele.ModeHTML, kb)
}

func boolLabel(b bool) string {
	if b {
		return "✅ فعال"
	}
	return "◻️ غیرفعال"
}

// ── wizard ساخت «تبلیغ مستقل جدید» / «پست ساده» ───────────────────

func (h *Handler) adNewStandalone(c tele.Context, campaignID string) error {
	ctx := context.Background()
	st := userState{Step: stepAdName, Data: map[string]string{"campaign_id": campaignID, "mode": "standalone"}}
	h.setState(ctx, c.Sender().ID, st)
	return c.Edit("یک نام داخلی برای این تبلیغ بفرستید:")
}

func (h *Handler) adNewSimple(c tele.Context, campaignID string) error {
	ctx := context.Background()
	st := userState{Step: stepAdName, Data: map[string]string{"campaign_id": campaignID, "mode": "simple"}}
	h.setState(ctx, c.Sender().ID, st)
	return c.Edit("یک نام داخلی برای این پستِ ساده بفرستید:")
}

func (h *Handler) handleAdName(ctx context.Context, c tele.Context, st userState, text string) error {
	name := trimText(text)
	if name == "" {
		return c.Send("نام نمی‌تواند خالی باشد:", kbCancelOnly())
	}
	st.Data["name"] = name
	h.setState(ctx, c.Sender().ID, userState{Step: stepAdMain, Data: st.Data})
	return c.Send(
		"✅ نام ثبت شد.\n\n📌 حالا «پست اصلی» را بفرستید.\nهر نوع و از هر منبعی باشد (متن، عکس، ویدئو، فوروارد و…) عیناً همان ارسال می‌شود.",
		kbCancelOnly(),
	)
}

// handleAdMain پست اصلی را ثبت و تبلیغ را می‌سازد؛ اگر حالت «ساده» باشد
// همین‌جا تمام می‌شود، وگرنه وارد مرحله‌ی افزودن ریپلی می‌شود.
func (h *Handler) handleAdMain(ctx context.Context, c tele.Context, st userState) error {
	uid := c.Sender().ID
	msg := c.Message()
	if msg == nil {
		return c.Send("پیام نامعتبر است. دوباره بفرستید:", kbCancelOnly())
	}
	ad := &models.Advertisement{
		CampaignID:    st.Data["campaign_id"],
		Name:          st.Data["name"],
		SourceChatID:  c.Chat().ID,
		MainMessageID: msg.ID,
	}
	if err := h.store.CreateAd(ctx, ad); err != nil {
		h.log.Error("create ad", portsF("err", err))
		h.clearState(ctx, uid)
		return c.Send("❌ خطا در ذخیره‌ی تبلیغ.", kbAdminMain())
	}

	if st.Data["mode"] == "simple" {
		h.clearState(ctx, uid)
		return c.Send(fmt.Sprintf("🎉 پستِ ساده «%s» ذخیره شد (بدون ریپلی).", ad.Name), kbAdminMain())
	}

	st.Data["ad_id"] = ad.ID
	h.setState(ctx, uid, userState{Step: stepAdReplies, Data: st.Data})

	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(cbBtn(kb, "✅ تمام شد", "ad_done")))
	_ = c.Send(
		"📌 پست اصلی ثبت شد.\n\nحالا ریپلی‌ها را یکی‌یکی بفرستید — بعد از هرکدام مدت‌زمان نمایشش را (به دقیقه) می‌پرسم.\nوقتی تمام شد دکمه‌ی زیر را بزنید:",
		kb,
	)
	return nil
}

// ── افزودن ریپلی (هم برای تبلیغ تازه‌ساز، هم برای «ریپلی روی تبلیغ موجود») ──

// adAttachPickList لیست تبلیغ‌های موجود کمپین برای افزودن ریپلیِ بعدی.
func (h *Handler) adAttachPickList(c tele.Context, campaignID string) error {
	ctx := context.Background()
	ads, _ := h.store.ListAdsByCampaign(ctx, campaignID)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, ad := range ads {
		rows = append(rows, kb.Row(cbBtn(kb, "📣 "+ad.Name, "ad_atc_pick:"+ad.ID)))
	}
	rows = append(rows, kb.Row(cbBtn(kb, "🔙 بازگشت", "ad_new:"+campaignID)))
	kb.Inline(rows...)

	text := "↩️ <b>افزودن ریپلی به کدام تبلیغ؟</b>"
	if len(ads) == 0 {
		text = "هنوز تبلیغی در این کمپین نیست که بشود رویش ریپلی گذاشت.\nابتدا از «🆕 تبلیغ مستقل جدید» یکی بسازید."
	}
	return c.Edit(text, tele.ModeHTML, kb)
}

func (h *Handler) adAttachPick(c tele.Context, adID string) error {
	ctx := context.Background()
	ad, err := h.store.FindAd(ctx, adID)
	if err != nil || ad == nil {
		return c.Respond(&tele.CallbackResponse{Text: "تبلیغ پیدا نشد"})
	}
	h.setState(ctx, c.Sender().ID, userState{
		Step: stepAdReplies,
		Data: map[string]string{"ad_id": adID, "campaign_id": ad.CampaignID},
	})
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(cbBtn(kb, "✅ تمام شد", "ad_done")))
	return c.Edit(fmt.Sprintf("↩️ ریپلیِ بعدی برای «%s» را بفرستید:", ad.Name), kb)
}

// handleAdReplies پیام دریافتی را به‌عنوان یک ریپلیِ در انتظار ثبت می‌کند
// و مدت‌زمان نمایشش را می‌پرسد.
func (h *Handler) handleAdReplies(ctx context.Context, c tele.Context, st userState) error {
	msg := c.Message()
	if msg == nil {
		return nil
	}
	if t := trimText(c.Text()); t == "/done" || t == "تمام" || t == "پایان" {
		return h.adFinish(ctx, c, st)
	}
	st.Data["pending_reply_msg_id"] = strconv.Itoa(msg.ID)
	h.setState(ctx, c.Sender().ID, userState{Step: stepAdReplyMinutes, Data: st.Data})
	return c.Send("⏱ این ریپلی چند دقیقه نمایش داده شود؟ فقط عدد بفرستید (مثل 5):", kbCancelOnly())
}

// handleAdReplyMinutes مدت‌زمان دریافتی را با پیامِ در انتظار ذخیره می‌کند.
func (h *Handler) handleAdReplyMinutes(ctx context.Context, c tele.Context, st userState, text string) error {
	dur, err := strconv.Atoi(trimText(models.NormalizeDigits(text)))
	if err != nil || dur < 1 {
		return c.Send("❌ یک عدد معتبر (دقیقه، حداقل ۱) بفرستید:", kbCancelOnly())
	}
	msgID, _ := strconv.Atoi(st.Data["pending_reply_msg_id"])
	adID := st.Data["ad_id"]
	if err := h.store.AppendAdReply(ctx, adID, msgID, dur); err != nil {
		h.log.Error("append reply", portsF("err", err))
		return c.Send("❌ خطا در افزودن ریپلی.", kbCancelOnly())
	}
	delete(st.Data, "pending_reply_msg_id")
	h.setState(ctx, c.Sender().ID, userState{Step: stepAdReplies, Data: st.Data})

	ad, _ := h.store.FindAd(ctx, adID)
	n := 0
	if ad != nil {
		n = len(ad.Replies)
	}
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(cbBtn(kb, "✅ تمام شد", "ad_done")))
	return c.Send(fmt.Sprintf("➕ ریپلی %d اضافه شد (%d دقیقه). بعدی را بفرستید یا «تمام شد» را بزنید.", n, dur), kb)
}

// adFinish پایان wizard افزودن ریپلی.
func (h *Handler) adFinish(ctx context.Context, c tele.Context, st userState) error {
	uid := c.Sender().ID
	adID := st.Data["ad_id"]
	h.clearState(ctx, uid)
	ad, _ := h.store.FindAd(ctx, adID)
	name := ""
	if ad != nil {
		name = ad.Name
	}
	return c.Send(fmt.Sprintf("🎉 تبلیغ «%s» ذخیره شد.", name), kbAdminMain())
}

// adDone از طریق دکمه‌ی inline پایان می‌دهد.
func (h *Handler) adDone(c tele.Context) error {
	ctx := context.Background()
	st := h.getState(ctx, c.Sender().ID)
	if st.Step != stepAdReplies {
		return c.Respond()
	}
	_ = c.Respond(&tele.CallbackResponse{Text: "ذخیره شد"})
	return h.adFinish(ctx, c, st)
}

// ── ویرایش کامل تبلیغ (بخش ۲.۵) ────────────────────────────────────

func (h *Handler) adEditNameStart(c tele.Context, id string) error {
	ctx := context.Background()
	h.setState(ctx, c.Sender().ID, userState{Step: stepAdEditName, Data: map[string]string{"ad_id": id}})
	return c.Edit("نام جدید تبلیغ را بفرستید:")
}

func (h *Handler) handleAdEditName(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	name := trimText(text)
	if name == "" {
		return c.Send("نام نمی‌تواند خالی باشد:", kbCancelOnly())
	}
	id := st.Data["ad_id"]
	h.clearState(ctx, uid)
	if err := h.store.UpdateAd(ctx, id, bson.D{{Key: "name", Value: name}}); err != nil {
		h.log.Error("edit ad name", portsF("err", err))
		return c.Send("❌ خطا در ویرایش.", kbAdminMain())
	}
	_ = c.Send("✅ نام به‌روزرسانی شد.")
	return h.sendAdView(c, id)
}

func (h *Handler) adEditMainStart(c tele.Context, id string) error {
	ctx := context.Background()
	h.setState(ctx, c.Sender().ID, userState{Step: stepAdEditMain, Data: map[string]string{"ad_id": id}})
	return c.Edit("پست اصلیِ جدید را بفرستید (جایگزین پست فعلی می‌شود):")
}

func (h *Handler) handleAdEditMain(ctx context.Context, c tele.Context, st userState) error {
	uid := c.Sender().ID
	msg := c.Message()
	if msg == nil {
		return c.Send("پیام نامعتبر است. دوباره بفرستید:", kbCancelOnly())
	}
	id := st.Data["ad_id"]
	h.clearState(ctx, uid)
	if err := h.store.ReplaceAdMain(ctx, id, msg.ID); err != nil {
		h.log.Error("replace ad main", portsF("err", err))
		return c.Send("❌ خطا در ویرایش.", kbAdminMain())
	}
	_ = c.Send("✅ پست اصلی جایگزین شد.")
	return h.sendAdView(c, id)
}

// ── مدیریت ریپلی‌ها: حذف / جابه‌جایی / افزودن ─────────────────────

func (h *Handler) adRepliesView(c tele.Context, id string) error {
	ctx := context.Background()
	ad, err := h.store.FindAd(ctx, id)
	if err != nil || ad == nil {
		return c.Edit("تبلیغ پیدا نشد.")
	}
	h.setCtx(ctx, c.Sender().ID, "ad", id)

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for i, r := range ad.Replies {
		rows = append(rows, kb.Row(
			cbBtn(kb, fmt.Sprintf("%d) %d دقیقه", i+1, r.DurationMinutes), "ad_reply_noop"),
			cbBtn(kb, "⬆️", "ad_reply_up:"+strconv.Itoa(i)),
			cbBtn(kb, "⬇️", "ad_reply_down:"+strconv.Itoa(i)),
			cbBtn(kb, "🗑", "ad_reply_del:"+strconv.Itoa(i)),
		))
	}
	rows = append(rows, kb.Row(cbBtn(kb, "➕ افزودن ریپلی", "ad_reply_add:"+id)))
	rows = append(rows, kb.Row(cbBtn(kb, "🔙 بازگشت", "ad_view:"+id)))
	kb.Inline(rows...)

	text := "📝 <b>ریپلی‌های این تبلیغ</b>\nترتیبِ نمایش از بالا به پایین است؛ هرکدام مدت‌زمانِ خودش را دارد."
	if len(ad.Replies) == 0 {
		text += "\n\nهنوز ریپلی‌ای ندارد."
	}
	return c.Edit(text, tele.ModeHTML, kb)
}

func (h *Handler) adReplyAddStart(c tele.Context, id string) error {
	ctx := context.Background()
	ad, err := h.store.FindAd(ctx, id)
	if err != nil || ad == nil {
		return c.Respond(&tele.CallbackResponse{Text: "تبلیغ پیدا نشد"})
	}
	h.setState(ctx, c.Sender().ID, userState{
		Step: stepAdReplies,
		Data: map[string]string{"ad_id": id, "campaign_id": ad.CampaignID},
	})
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(cbBtn(kb, "✅ تمام شد", "ad_done")))
	return c.Edit(fmt.Sprintf("➕ ریپلیِ جدید برای «%s» را بفرستید:", ad.Name), kb)
}

func (h *Handler) adReplyDelete(c tele.Context, idxStr string) error {
	ctx := context.Background()
	id := h.getCtx(ctx, c.Sender().ID, "ad")
	ad, err := h.store.FindAd(ctx, id)
	if err != nil || ad == nil {
		return c.Respond(&tele.CallbackResponse{Text: "تبلیغ پیدا نشد"})
	}
	idx, e := strconv.Atoi(idxStr)
	if e != nil || idx < 0 || idx >= len(ad.Replies) {
		return c.Respond()
	}
	next := append(append([]models.AdReply{}, ad.Replies[:idx]...), ad.Replies[idx+1:]...)
	_ = h.store.ReplaceAdReplies(ctx, id, next)
	return h.adRepliesView(c, id)
}

func (h *Handler) adReplyMove(c tele.Context, idxStr string, dir int) error {
	ctx := context.Background()
	id := h.getCtx(ctx, c.Sender().ID, "ad")
	ad, err := h.store.FindAd(ctx, id)
	if err != nil || ad == nil {
		return c.Respond(&tele.CallbackResponse{Text: "تبلیغ پیدا نشد"})
	}
	idx, e := strconv.Atoi(idxStr)
	target := idx + dir
	if e != nil || idx < 0 || idx >= len(ad.Replies) || target < 0 || target >= len(ad.Replies) {
		return c.Respond()
	}
	next := append([]models.AdReply{}, ad.Replies...)
	next[idx], next[target] = next[target], next[idx]
	_ = h.store.ReplaceAdReplies(ctx, id, next)
	return h.adRepliesView(c, id)
}

// ── تنظیمات ثابت/پین (بخش ۲.۴) ──────────────────────────────────────

func (h *Handler) adSettingsView(c tele.Context, id string) error {
	ctx := context.Background()
	ad, err := h.store.FindAd(ctx, id)
	if err != nil || ad == nil {
		return c.Edit("تبلیغ پیدا نشد.")
	}
	h.setCtx(ctx, c.Sender().ID, "ad", id)

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(cbBtn(kb, toggleLabel("همیشه آخرین پیام کانال بماند", ad.KeepAsLastMessage), "ad_tg_keep")),
		kb.Row(cbBtn(kb, toggleLabel("پین در کانال", ad.PinMessage), "ad_tg_pin")),
		kb.Row(cbBtn(kb, toggleLabel("حذف نسخه‌ی قبلی هنگام بازارسال", ad.DeletePreviousOnRepost), "ad_tg_delprev")),
		kb.Row(cbBtn(kb, "🔙 بازگشت", "ad_view:"+id)),
	)
	text := "⚙️ <b>تنظیمات ثابت/پین</b>\n\n" +
		"🔽 <b>همیشه آخرین پیام بماند</b> — با هر پستِ دیگری که بعدش در کانال بیاید، این تبلیغ دوباره فرستاده می‌شود تا جدیدترین پیامِ کانال بماند.\n" +
		"📌 <b>پین در کانال</b> — با Pin بومی تلگرام سنجاق می‌شود (ربات باید دسترسی Pin در آن کانال داشته باشد).\n" +
		"🔁 <b>حذف نسخه‌ی قبلی هنگام بازارسال</b> — وقتی به‌خاطر «آخرین پیام بماند» دوباره فرستاده می‌شود، نسخه‌ی قبلی‌اش در کانال پاک شود یا باقی بماند."
	return c.Edit(text, tele.ModeHTML, kb)
}

func toggleLabel(label string, on bool) string {
	if on {
		return "✅ " + label
	}
	return "◻️ " + label
}

func (h *Handler) adToggleField(c tele.Context, field string) error {
	ctx := context.Background()
	id := h.getCtx(ctx, c.Sender().ID, "ad")
	ad, err := h.store.FindAd(ctx, id)
	if err != nil || ad == nil {
		return c.Respond(&tele.CallbackResponse{Text: "تبلیغ پیدا نشد"})
	}
	var key string
	var next bool
	switch field {
	case "keep":
		key, next = "keep_as_last_message", !ad.KeepAsLastMessage
	case "pin":
		key, next = "pin_message", !ad.PinMessage
	case "delprev":
		key, next = "delete_previous_on_repost", !ad.DeletePreviousOnRepost
	default:
		return c.Respond()
	}
	_ = h.store.UpdateAd(ctx, id, bson.D{{Key: key, Value: next}})
	return h.adSettingsView(c, id)
}

// ── حذف / پیش‌نمایش ──────────────────────────────────────────────

func (h *Handler) adDelete(c tele.Context, id string) error {
	ctx := context.Background()
	ad, _ := h.store.FindAd(ctx, id)
	if err := h.store.DeleteAd(ctx, id); err != nil {
		h.log.Error("delete ad", portsF("err", err))
	}
	cid := ""
	if ad != nil {
		cid = ad.CampaignID
	}
	return h.adList(c, cid)
}

func (h *Handler) adPreview(c tele.Context, id string) error {
	ctx := context.Background()
	ad, err := h.store.FindAd(ctx, id)
	if err != nil || ad == nil {
		return c.Respond(&tele.CallbackResponse{Text: "تبلیغ پیدا نشد"})
	}
	if ad.SourceChatID == 0 || ad.MainMessageID == 0 {
		return c.Respond(&tele.CallbackResponse{
			Text:      "این تبلیغ ناقص است (با نسخه‌ی قبلی ساخته شده). لطفاً حذف و دوباره بسازید.",
			ShowAlert: true,
		})
	}
	if _, err := h.postAd(c.Sender(), ad); err != nil {
		h.log.Error("ad preview", portsF("err", err))
		return c.Respond(&tele.CallbackResponse{Text: "خطا در پیش‌نمایش: " + err.Error(), ShowAlert: true})
	}
	return c.Respond(&tele.CallbackResponse{Text: "پیش‌نمایش ارسال شد"})
}

// postAd پست اصلی را Copy می‌کند و ریپلی‌ها را به‌ترتیب (همه با هم، فقط
// برای پیش‌نمایش) روی آن می‌گذارد. شناسه‌ی همه‌ی پیام‌های ارسال‌شده در
// گیرنده را برمی‌گرداند.
func (h *Handler) postAd(to tele.Recipient, ad *models.Advertisement) ([]int, error) {
	return copyAd(h.bot, to, ad)
}

// ── helpers ──────────────────────────────────────────────────────

// copyAd منطق مشترک ارسال یک تبلیغ (اصلی + همه‌ی ریپلی‌ها پشت سر هم) با
// Copy — فقط برای پیش‌نمایش استفاده می‌شود؛ رفتار واقعیِ چرخه‌ی متوالی
// (یک ریپلی در هر لحظه) در internal/scheduler پیاده شده است.
func copyAd(bot *tele.Bot, to tele.Recipient, ad *models.Advertisement) ([]int, error) {
	src := &tele.Chat{ID: ad.SourceChatID}
	main, err := bot.Copy(to, &tele.Message{ID: ad.MainMessageID, Chat: src})
	if err != nil {
		return nil, err
	}
	posted := []int{main.ID}
	for _, r := range ad.Replies {
		m, e := bot.Copy(to, &tele.Message{ID: r.MessageID, Chat: src},
			&tele.SendOptions{ReplyTo: &tele.Message{ID: main.ID}})
		if e != nil {
			continue // یک ریپلی شکست‌خورده نباید کل پیش‌نمایش را خراب کند
		}
		posted = append(posted, m.ID)
	}
	return posted, nil
}
