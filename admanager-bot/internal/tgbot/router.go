package tgbot

import (
	"strings"

	tele "gopkg.in/telebot.v4"
)

// onCallback مدیریت inline keyboard callbacks (همه ادمین‌محور).
func (h *Handler) onCallback(c tele.Context) error {
	if !h.isAdmin(c) {
		return c.Respond(&tele.CallbackResponse{Text: "⛔️"})
	}

	data := c.Callback().Data
	defer func() { _ = c.Respond() }()

	if len(data) > 0 && data[0] == '\f' {
		data = data[1:]
	}

	parts := strings.SplitN(data, ":", 2)
	action := parts[0]
	arg := ""
	if len(parts) == 2 {
		arg = parts[1]
	}

	switch action {

	// ── ناوبری ───────────────────────────────────────────────
	case "home":
		// کیبورد اصلی reply است، پس پیام inline قبلی را حذف و پیام تازه می‌فرستیم.
		_ = c.Delete()
		return c.Send("منوی اصلی 👇", kbAdminMain())

	// ── کانال‌ها ─────────────────────────────────────────────
	case "ch_list":
		return h.adminChannelsList(c)
	case "ch_add":
		return h.adminChannelAddStart(c)
	case "ch_view":
		return h.adminChannelView(c, arg)
	case "ch_toggle":
		return h.adminChannelToggle(c, arg)
	case "ch_del":
		return h.adminChannelDelete(c, arg)
	case "ch_tags":
		return h.adminChannelTagsView(c, arg)
	case "ch_tagtoggle":
		return h.adminChannelTagToggle(c, arg)
	case "ch_refresh":
		return h.adminChannelRefresh(c, arg)

	// ── برچسب‌ها ─────────────────────────────────────────────
	case "tag_list":
		return h.adminTagsList(c)
	case "tag_add":
		return h.adminTagAddStart(c)
	case "tag_del":
		return h.adminTagDelete(c, arg)
	case "tag_noop":
		return c.Respond()

	// ── کمپین‌ها ─────────────────────────────────────────────
	case "cmp_list":
		return h.campaignsList(c, arg)
	case "cmp_view":
		return h.campaignView(c, arg)
	case "cmp_start":
		return h.campaignSetStatus(c, arg, "start")
	case "cmp_pause":
		return h.campaignSetStatus(c, arg, "pause")
	case "cmp_resume":
		return h.campaignSetStatus(c, arg, "resume")
	case "cmp_cancel":
		return h.campaignSetStatus(c, arg, "cancel")
	case "cmp_del":
		return h.campaignDelete(c, arg)
	case "cmp_ename":
		return h.campaignEditNameStart(c, arg)
	case "cmp_esched":
		return h.campaignEditSchedStart(c, arg)
	case "cmp_tags":
		return h.campaignTagsView(c, arg)
	case "cmp_tagtoggle":
		return h.campaignTagToggle(c, arg)
	case "cmp_chans":
		return h.campaignChannelsView(c, arg)
	case "cmp_chantoggle":
		return h.campaignChannelToggle(c, arg)

	// ── تبلیغ‌ها ──────────────────────────────────────────────
	case "ad_list":
		return h.adList(c, arg)
	case "ad_new":
		return h.adNewMenu(c, arg)
	case "ad_new_std":
		return h.adNewStandalone(c, arg)
	case "ad_new_atc":
		return h.adAttachPickList(c, arg)
	case "ad_atc_pick":
		return h.adAttachPick(c, arg)
	case "ad_new_sim":
		return h.adNewSimple(c, arg)
	case "ad_done":
		return h.adDone(c)
	case "ad_view":
		return h.adView(c, arg)
	case "ad_del":
		return h.adDelete(c, arg)
	case "ad_preview":
		return h.adPreview(c, arg)
	case "ad_ename":
		return h.adEditNameStart(c, arg)
	case "ad_emain":
		return h.adEditMainStart(c, arg)
	case "ad_replies":
		return h.adRepliesView(c, arg)
	case "ad_reply_add":
		return h.adReplyAddStart(c, arg)
	case "ad_reply_del":
		return h.adReplyDelete(c, arg)
	case "ad_reply_up":
		return h.adReplyMove(c, arg, -1)
	case "ad_reply_down":
		return h.adReplyMove(c, arg, 1)
	case "ad_reply_noop":
		return c.Respond()
	case "ad_settings":
		return h.adSettingsView(c, arg)
	case "ad_tg_keep":
		return h.adToggleField(c, "keep")
	case "ad_tg_pin":
		return h.adToggleField(c, "pin")
	case "ad_tg_delprev":
		return h.adToggleField(c, "delprev")

	// ── قالب‌ها ───────────────────────────────────────────────
	case "tpl_list":
		return h.templatesList(c)
	case "tpl_new":
		return h.templateNewStart(c)
	case "tpl_view":
		return h.templateView(c, arg)
	case "tpl_del":
		return h.templateDelete(c, arg)
	case "tpl_use":
		return h.templateUse(c, arg)

	// ── آمار ─────────────────────────────────────────────────
	case "stats":
		return h.adminStats(c)

	// ── زمان‌بندی ────────────────────────────────────────────
	case "sch_day":
		return h.scheduleDay(c, arg)

	// ── تنظیمات ──────────────────────────────────────────────
	case "set_reminder":
		return h.settingsReminderStart(c)
	}

	return c.Respond(&tele.CallbackResponse{Text: "⚠️ نامعلوم"})
}
