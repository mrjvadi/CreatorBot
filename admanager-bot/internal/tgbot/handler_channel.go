// handler_channel.go — مدیریت کانال‌ها و برچسب‌ها (فاز ۲).
package tgbot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

var (
	errChannelNotFound = errors.New("❌ کانال پیدا نشد. یوزرنیم را درست بفرستید یا یک پیام از کانال forward کنید.")
	errChannelOnly     = errors.New("❌ این یک کانال نیست.")
)

// ── لیست کانال‌ها ────────────────────────────────────────────────

// renderChannels متن و کیبورد لیست کانال‌ها را می‌سازد.
func (h *Handler) renderChannels(ctx context.Context) (string, *tele.ReplyMarkup) {
	list, err := h.store.ListChannels(ctx, "", 1, listPageSize)
	if err != nil {
		h.log.Error("list channels", portsF("err", err))
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, ch := range list {
		label := fmt.Sprintf("%s %s (%d)", channelStatusIcon(ch.Status), ch.Title, ch.MemberCount)
		rows = append(rows, kb.Row(cbBtn(kb, label, "ch_view:"+ch.ID)))
	}
	rows = append(rows, kb.Row(
		cbBtn(kb, "➕ افزودن کانال", "ch_add"),
		cbBtn(kb, "🏷 برچسب‌ها", "tag_list"),
	))
	kb.Inline(rows...)

	text := "📡 <b>کانال‌های شما</b>\nکانال‌هایی که ربات در آن‌ها ادمین است و می‌تواند تبلیغ پخش کند."
	if len(list) == 0 {
		text += "\n\nهنوز کانالی اضافه نکرده‌اید — با «➕ افزودن کانال» شروع کنید."
	}
	return text, kb
}

// channelsHome ورود از کیبورد اصلی (پیام تازه).
func (h *Handler) channelsHome(c tele.Context) error {
	text, kb := h.renderChannels(context.Background())
	return c.Send(text, tele.ModeHTML, kb)
}

// adminChannelsList ورود از callback (ویرایش پیام).
func (h *Handler) adminChannelsList(c tele.Context) error {
	text, kb := h.renderChannels(context.Background())
	return c.Edit(text, tele.ModeHTML, kb)
}

// ── مشاهده‌ی کانال ───────────────────────────────────────────────

func (h *Handler) adminChannelView(c tele.Context, id string) error {
	ctx := context.Background()
	ch, err := h.store.FindChannel(ctx, id)
	if err != nil || ch == nil {
		return c.Edit("کانال پیدا نشد.")
	}

	tags, _ := h.store.ListTags(ctx)
	tagNames := tagLabels(tags, ch.TagIDs)

	text := fmt.Sprintf(
		"📡 <b>%s</b>\n\n"+
			"یوزرنیم: %s\n"+
			"تعداد اعضا: %d\n"+
			"وضعیت: %s\n"+
			"برچسب‌ها: %s\n"+
			"🆔 <code>%s</code>",
		ch.Title, fmtUsername(ch.Username), ch.MemberCount,
		channelStatusLabel(ch.Status), emptyDash(tagNames), ch.ID,
	)

	kb := &tele.ReplyMarkup{}
	toggle := cbBtn(kb, "⏸ غیرفعال‌سازی", "ch_toggle:"+ch.ID)
	if ch.Status != models.ChannelActive {
		toggle = cbBtn(kb, "▶️ فعال‌سازی", "ch_toggle:"+ch.ID)
	}
	kb.Inline(
		kb.Row(
			cbBtn(kb, "🏷 برچسب‌ها", "ch_tags:"+ch.ID),
			cbBtn(kb, "🔄 به‌روزرسانی آمار", "ch_refresh:"+ch.ID),
		),
		kb.Row(toggle),
		kb.Row(cbBtn(kb, "🗑 حذف", "ch_del:"+ch.ID)),
		kb.Row(cbBtn(kb, "🔙 بازگشت", "ch_list")),
	)
	return c.Edit(text, tele.ModeHTML, kb)
}

// ── اختصاص برچسب به کانال ────────────────────────────────────────

func (h *Handler) adminChannelTagsView(c tele.Context, id string) error {
	ctx := context.Background()
	ch, err := h.store.FindChannel(ctx, id)
	if err != nil || ch == nil {
		return c.Edit("کانال پیدا نشد.")
	}
	h.setCtx(ctx, c.Sender().ID, "ch", id)
	tags, _ := h.store.ListTags(ctx)
	selected := map[string]bool{}
	for _, t := range ch.TagIDs {
		selected[t] = true
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, t := range tags {
		mark := "▫️"
		if selected[t.ID] {
			mark = "✅"
		}
		rows = append(rows, kb.Row(cbBtn(kb, mark+" "+t.Name, "ch_tagtoggle:"+t.ID)))
	}
	rows = append(rows, kb.Row(cbBtn(kb, "🔙 بازگشت", "ch_view:"+id)))
	kb.Inline(rows...)

	text := "🏷 <b>برچسب‌های این کانال</b>\nبا برچسب‌گذاری، کانال در کمپین‌هایی که همان برچسب را هدف گرفته‌اند نمایش داده می‌شود."
	if len(tags) == 0 {
		text = "هیچ برچسبی نیست. ابتدا از «📡 کانال‌ها → 🏷 برچسب‌ها» یک برچسب بسازید."
	}
	return c.Edit(text, tele.ModeHTML, kb)
}

func (h *Handler) adminChannelTagToggle(c tele.Context, tID string) error {
	ctx := context.Background()
	chID := h.getCtx(ctx, c.Sender().ID, "ch")
	ch, err := h.store.FindChannel(ctx, chID)
	if err != nil || ch == nil {
		return c.Respond(&tele.CallbackResponse{Text: "کانال پیدا نشد"})
	}
	found := false
	var next []string
	for _, t := range ch.TagIDs {
		if t == tID {
			found = true
			continue
		}
		next = append(next, t)
	}
	if !found {
		next = append(next, tID)
	}
	_ = h.store.UpdateChannel(ctx, chID, bson.D{{Key: "tag_ids", Value: next}})
	return h.adminChannelTagsView(c, chID)
}

// adminChannelRefresh تعداد اعضای کانال را از تلگرام تازه می‌کند.
func (h *Handler) adminChannelRefresh(c tele.Context, id string) error {
	ctx := context.Background()
	ch, err := h.store.FindChannel(ctx, id)
	if err != nil || ch == nil {
		return c.Respond(&tele.CallbackResponse{Text: "کانال پیدا نشد"})
	}
	if n, e := h.bot.Len(&tele.Chat{ID: ch.TelegramID}); e == nil {
		_ = h.store.UpdateChannelStats(ctx, id, n, ch.AvgViews, ch.EngageRate)
	}
	return h.adminChannelView(c, id)
}

func (h *Handler) adminChannelToggle(c tele.Context, id string) error {
	ctx := context.Background()
	ch, err := h.store.FindChannel(ctx, id)
	if err != nil || ch == nil {
		return c.Respond(&tele.CallbackResponse{Text: "کانال پیدا نشد"})
	}
	next := models.ChannelActive
	if ch.Status == models.ChannelActive {
		next = models.ChannelInactive
	}
	if err := h.store.UpdateChannelStatus(ctx, id, next); err != nil {
		h.log.Error("toggle channel", portsF("err", err))
	}
	return h.adminChannelView(c, id)
}

func (h *Handler) adminChannelDelete(c tele.Context, id string) error {
	ctx := context.Background()
	if err := h.store.DeleteChannel(ctx, id); err != nil {
		h.log.Error("delete channel", portsF("err", err))
	}
	h.audit(ctx, c, models.AuditChannelRemove, "channel", id, "حذف کانال")
	return h.adminChannelsList(c)
}

// ── افزودن کانال ─────────────────────────────────────────────────

func (h *Handler) adminChannelAddStart(c tele.Context) error {
	ctx := context.Background()
	h.setStep(ctx, c.Sender().ID, stepChannelAdd)
	return c.Edit(
		"➕ <b>افزودن کانال</b>\n\n"+
			"یکی از این دو کار را انجام دهید:\n"+
			"• یک پیام از کانال را <b>forward</b> کنید (برای کانال‌های بدون یوزرنیم)\n"+
			"• یا یوزرنیم کانال را بفرستید (مثل <code>@mychannel</code>)\n\n"+
			"⚠️ ربات باید از قبل ادمین آن کانال باشد.",
		tele.ModeHTML,
	)
}

// handleChannelAdd کانال را از روی پیام forward‌شده یا یوزرنیم ثبت می‌کند.
func (h *Handler) handleChannelAdd(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	// ابتدا حالت forward را بررسی می‌کنیم (کانال بدون یوزرنیم).
	chat, err := h.resolveChannelInput(c, text)
	if err != nil {
		return c.Send(err.Error(), kbAdminMain())
	}

	// بررسی ادمین بودن ربات
	member, merr := h.bot.ChatMemberOf(chat, h.bot.Me)
	if merr != nil || (member.Role != tele.Administrator && member.Role != tele.Creator) {
		return c.Send("❌ ربات ادمین این کانال نیست. ابتدا ربات را ادمین کنید.", kbAdminMain())
	}

	if existing, _ := h.store.FindChannelByTelegramID(ctx, chat.ID); existing != nil {
		return c.Send("این کانال قبلاً اضافه شده است.", kbAdminMain())
	}

	memberCount, _ := h.bot.Len(chat)

	ch := &models.Channel{
		TelegramID:  chat.ID,
		Username:    strings.TrimPrefix(chat.Username, "@"),
		Title:       chat.Title,
		MemberCount: memberCount,
	}
	if err := h.store.CreateChannel(ctx, ch); err != nil {
		h.log.Error("create channel", portsF("err", err))
		return c.Send("❌ خطا در ثبت کانال.", kbAdminMain())
	}
	// چون ابزار ادمین‌محور است، کانال بلافاصله فعال می‌شود.
	_ = h.store.UpdateChannelStatus(ctx, ch.ID, models.ChannelActive)
	h.audit(ctx, c, models.AuditChannelAdd, "channel", ch.ID, "افزودن کانال "+ch.Title)

	return c.Send(
		fmt.Sprintf("✅ کانال «%s» با %d عضو اضافه و فعال شد.", ch.Title, memberCount),
		kbAdminMain(),
	)
}

// resolveChannelInput کانال را از پیام forward‌شده یا متن یوزرنیم تشخیص می‌دهد.
func (h *Handler) resolveChannelInput(c tele.Context, text string) (*tele.Chat, error) {
	if msg := c.Message(); msg != nil && msg.OriginalChat != nil {
		if msg.OriginalChat.Type != tele.ChatChannel {
			return nil, errChannelOnly
		}
		return msg.OriginalChat, nil
	}

	uname := strings.TrimSpace(text)
	if uname == "" {
		return nil, errChannelNotFound
	}
	if !strings.HasPrefix(uname, "@") {
		uname = "@" + uname
	}
	chat, err := h.bot.ChatByUsername(uname)
	if err != nil || chat == nil {
		return nil, errChannelNotFound
	}
	if chat.Type != tele.ChatChannel {
		return nil, errChannelOnly
	}
	return chat, nil
}

// ── برچسب‌ها ─────────────────────────────────────────────────────

func (h *Handler) adminTagsList(c tele.Context) error {
	ctx := context.Background()
	tags, _ := h.store.ListTags(ctx)

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, t := range tags {
		rows = append(rows, kb.Row(
			cbBtn(kb, "🏷 "+t.Name, "tag_noop"),
			cbBtn(kb, "🗑", "tag_del:"+t.ID),
		))
	}
	rows = append(rows,
		kb.Row(cbBtn(kb, "➕ برچسب جدید", "tag_add")),
		kb.Row(cbBtn(kb, "🔙 بازگشت", "ch_list")),
	)
	kb.Inline(rows...)

	text := "🏷 <b>برچسب‌ها</b>\nبرای دسته‌بندی کانال‌ها و هدف‌گذاریِ کمپین‌ها روی یک دسته (مثلاً «موسیقی») استفاده می‌شوند."
	if len(tags) == 0 {
		text += "\n\nهنوز برچسبی نساخته‌اید."
	}
	return c.Edit(text, tele.ModeHTML, kb)
}

func (h *Handler) adminTagAddStart(c tele.Context) error {
	ctx := context.Background()
	h.setStep(ctx, c.Sender().ID, stepTagAdd)
	return c.Edit("➕ نام برچسبِ جدید را بفرستید (مثلاً «موسیقی»):", tele.ModeHTML)
}

func (h *Handler) handleTagAdd(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)
	name := strings.TrimSpace(text)
	if name == "" {
		return c.Send("نام برچسب خالی است.", kbAdminMain())
	}
	tag := &models.Tag{Name: name, Slug: slugify(name)}
	if err := h.store.CreateTag(ctx, tag); err != nil {
		h.log.Error("create tag", portsF("err", err))
		return c.Send("❌ خطا در ساخت برچسب.", kbAdminMain())
	}
	return c.Send(fmt.Sprintf("✅ برچسب «%s» ساخته شد.", name), kbAdminMain())
}

func (h *Handler) adminTagDelete(c tele.Context, id string) error {
	ctx := context.Background()
	// حذف نرم با غیرفعال‌سازی کافی نیست؛ از DeleteTag استفاده می‌کنیم اگر بود،
	// در غیر این صورت غیرفعال. اینجا غیرفعال‌سازی منطقی است.
	_ = h.store.SetTagInactive(ctx, id)
	return h.adminTagsList(c)
}

// ── helpers ──────────────────────────────────────────────────────

func channelStatusIcon(s models.ChannelStatus) string {
	switch s {
	case models.ChannelActive:
		return "🟢"
	case models.ChannelInactive:
		return "⚪️"
	default:
		return "🟡"
	}
}

func channelStatusLabel(s models.ChannelStatus) string {
	switch s {
	case models.ChannelActive:
		return "🟢 فعال"
	case models.ChannelInactive:
		return "⚪️ غیرفعال"
	default:
		return "🟡 در انتظار"
	}
}

func fmtUsername(u string) string {
	if u == "" {
		return "—"
	}
	return "@" + u
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func tagLabels(tags []models.Tag, ids []string) string {
	idset := map[string]bool{}
	for _, id := range ids {
		idset[id] = true
	}
	var names []string
	for _, t := range tags {
		if idset[t.ID] {
			names = append(names, t.Name)
		}
	}
	return strings.Join(names, "، ")
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
