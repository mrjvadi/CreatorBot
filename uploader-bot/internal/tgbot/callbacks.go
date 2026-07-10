package tgbot

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"
)

// onCallback مسیریابی همه‌ی دکمه‌های شیشه‌ای (callback). برای خوانایی از bot.go جدا شده.
func (h *Handler) onCallback(c tele.Context) error {
	ctx := context.Background()
	data := strings.TrimPrefix(c.Callback().Data, "\f")
	defer c.Respond()

	parts := strings.SplitN(data, ":", 3)
	action := parts[0]
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}
	arg2 := ""
	if len(parts) > 2 {
		arg2 = parts[2]
	}

	// ── پنل مدیریت inline (p / ps / pt / pv) ─────────────────
	if action == "p" || action == "ps" || action == "pt" || action == "pv" {
		if _, err := h.handlePanel(ctx, c, action, arg, arg2); err != nil {
			return err
		}
		return nil
	}

	// ── گیت امنیتی: اکشن‌های عمومی که نیاز به دسترسی ادمین ندارند ──
	// هر اکشنی که در این لیست نباشد، ادمین‌محور فرض می‌شود و نیاز به
	// h.isAdmin(c) دارد؛ در غیر این صورت دسترسی رد می‌شود (deny-by-default).
	publicActions := map[string]bool{
		"check_join":    true,
		"gate":          true,
		"sub_buy":       true,
		"sub_pay":       true,
		"pay_verify":    true,
		"folder_open":   true,
		"code_resend":   true,
		"react_like":    true,
		"react_dislike": true,
		"report":        true,
		"noop":          true,
	}
	if !publicActions[action] && !h.isAdmin(c) {
		return c.Respond(&tele.CallbackResponse{Text: "⛔️ دسترسی ندارید"})
	}

	switch action {
	// ── ادمین ────────────────────────────────────────────────
	case "admin_code_del":
		return h.adminDeleteCode(ctx, c, arg)
	case "admin_code_edit":
		return h.adminEditCodeMenu(ctx, c, arg)
	case "admin_code_set_forward":
		return h.adminToggleCodeProp(ctx, c, arg, "forward_lock")
	case "admin_code_set_delete":
		return h.adminSetAutoDelete(ctx, c, arg, arg2)
	case "admin_code_set_sub":
		return h.adminToggleCodeProp(ctx, c, arg, "sub_required")
	case "admin_code_set_channel":
		return h.adminToggleCodeProp(ctx, c, arg, "channel_lock")
	case "admin_folder_open":
		return h.adminFolderOpen(ctx, c, arg)
	case "admin_folder_del":
		return h.adminFolderDelete(ctx, c, arg)
	case "admin_sub_del":
		return h.adminSubPlanDelete(ctx, c, arg)
	case "ch_add":
		return h.adminAskChannel(ctx, c)
	case "lk":
		return h.lockDetail(ctx, c, arg)
	case "lk_add":
		return h.lockAskAdd(ctx, c)
	case "lk_mode":
		return h.lockToggleMode(ctx, c, arg)
	case "lk_cap":
		return h.lockAskCap(ctx, c, arg)
	case "lk_del":
		return h.lockDelete(ctx, c, arg)
	case "lk_leave":
		return h.lockToggleLeave(ctx, c)
	case "plan_add":
		return h.adminAskPlan(ctx, c)
	case "aperm":
		return h.adminPermsMenu(ctx, c, arg)
	case "aperm_t":
		return h.adminTogglePerm(ctx, c, arg, arg2)
	case "admin_add":
		return h.adminAskAdmin(ctx, c)
	case "admin_del":
		return h.adminRemoveAdmin(ctx, c, arg)
	case "admin_ch_del":
		return h.adminForceJoinDelete(ctx, c, arg)
	case "admin_user_block":
		return h.adminToggleBlock(ctx, c, arg, true)
	case "admin_user_unblock":
		return h.adminToggleBlock(ctx, c, arg, false)
	case "admin_pay_confirm":
		return h.adminConfirmPayment(ctx, c, arg)
	case "admin_pay_reject":
		return h.adminRejectPayment(ctx, c, arg)

	// ── تنظیمات کد رسانه ─────────────────────────────────────
	case "code_delete":
		return h.adminDeleteCode(ctx, c, arg)
	case "code_list":
		return h.adminListCodes(ctx, c)
	case "slide":
		return h.adminSlideshow(ctx, c, arg)
	case "code_toggle_forward":
		return h.adminToggleCodeProp(ctx, c, arg, "forward_lock")
	case "code_toggle_antidl":
		return h.adminToggleAutoDelete(ctx, c, arg)
	case "code_set_password":
		h.SetStepData(ctx, c.Sender().ID, stepSetPassword, "code_id", arg)
		return c.Send("🔐 رمز جدید را بفرستید (برای حذف: 0):", kbCancelOnly())
	case "code_set_limit":
		h.SetStepData(ctx, c.Sender().ID, stepSetLimit, "code_id", arg)
		return c.Send("📥 محدودیت دانلود هر کاربر را بفرستید (0=نامحدود):", kbCancelOnly())
	case "code_edit_caption":
		h.SetStepData(ctx, c.Sender().ID, stepEditCaption, "code_id", arg)
		return c.Send("✏️ کپشن جدید را بفرستید:", kbCancelOnly())
	case "code_send_preview":
		return h.adminSendPreview(ctx, c, arg)
	case "code_toggle_channel":
		return h.adminToggleCodeProp(ctx, c, arg, "channel_lock")
	case "code_toggle_sub":
		return h.adminToggleCodeProp(ctx, c, arg, "sub_required")
	case "code_toggle_seen":
		return h.adminToggleCodeProp(ctx, c, arg, "force_seen")
	case "code_toggle_react":
		return h.adminToggleCodeProp(ctx, c, arg, "force_react")
	case "code_set_likes":
		return h.adminCodeAskFake(ctx, c, arg, stepSetLikes, "تعداد لایک فیک")
	case "code_set_downloads":
		return h.adminCodeAskFake(ctx, c, arg, stepSetDownloads, "تعداد دانلود فیک")
	case "code_set_views":
		return h.adminCodeAskFake(ctx, c, arg, stepSetViews, "تعداد بازدید فیک")
	case "code_set_cover":
		return h.adminCodeAskCover(ctx, c, arg)
	case "code_move":
		return h.adminCodeMoveMenu(ctx, c, arg)

	// ── پوشه‌ها ───────────────────────────────────────────────
	case "folder_new":
		return h.adminNewFolder(ctx, c)
	case "folder_delete":
		return h.adminFolderDelete(ctx, c, arg)
	case "folder_newsub":
		return h.adminNewSubfolder(ctx, c, arg)
	case "afolder":
		return h.adminFolderBrowse(ctx, c, arg)
	case "code_moveto":
		return h.adminCodeMoveTo(ctx, c, arg, arg2)

	// ── پیش‌نمایش / تبلیغات / اشتراک کاربر ────────────────────
	case "admin_prev_del":
		return h.adminPreviewDelete(ctx, c, arg)
	case "admin_ad_del":
		return h.adminAdDelete(ctx, c, arg)
	case "admin_user_subm":
		return h.adminUserSubMenu(ctx, c, arg)
	case "admin_setsub":
		return h.adminSetUserSub(ctx, c, arg, arg2)
	case "admin_user_reset":
		return h.adminResetUserDownloads(ctx, c, arg)

	// ── کاربر / تحویل ─────────────────────────────────────────
	case "check_join":
		return h.onCheckJoin(ctx, c)
	case "gate":
		return h.gatePass(ctx, c, arg)
	case "bc_copy":
		return h.askBroadcastContent(ctx, c, "copy")
	case "bc_forward":
		return h.askBroadcastContent(ctx, c, "forward")
	case "bcjob":
		return h.adminBroadcastJobView(ctx, c, arg)
	case "bccancel":
		return h.adminBroadcastCancel(ctx, c, arg)
	case "bcdelnow":
		return h.adminBroadcastDeleteNow(ctx, c, arg)

	// ── ابزارهای انبوه ────────────────────────────────────────
	case "tools_fwd_on":
		return h.toolsForwardAll(ctx, c, true)
	case "tools_fwd_off":
		return h.toolsForwardAll(ctx, c, false)
	case "tools_ad_on":
		return h.toolsAutoDeleteAll(ctx, c, true)
	case "tools_ad_off":
		return h.toolsAutoDeleteAll(ctx, c, false)
	case "tools_delall":
		return h.toolsDeleteAllConfirm(ctx, c)
	case "tools_delall_yes":
		return h.toolsDeleteAll(ctx, c)

	// ── بکاپ/ریستور ───────────────────────────────────────────
	case "backup_export":
		return h.adminBackupExport(ctx, c)
	case "backup_restore":
		return h.adminBackupRestoreAsk(ctx, c)

	// ── صف تایید / اشتراک کاربر ───────────────────────────────
	case "code_approve":
		return h.adminApproveCode(ctx, c, arg)
	case "code_reject":
		return h.adminRejectCode(ctx, c, arg)
	case "sub_buy":
		return h.userBuySubPlan(ctx, c, arg)
	case "sub_pay":
		return h.userPaySub(ctx, c, arg, arg2)
	case "pay_verify":
		return h.payVerify(ctx, c, arg)
	case "folder_open":
		return h.userOpenFolder(ctx, c, arg)
	case "code_resend":
		user, err := h.Store.GetUser(ctx, c.Sender().ID)
		h.LogErr("code_resend: get user", err)
		return h.userDeliverCode(ctx, c, user, arg)
	case "react_like":
		return h.reactToggle(ctx, c, arg, 1)
	case "react_dislike":
		return h.reactToggle(ctx, c, arg, -1)
	case "noop":
		return c.Respond()
	case "code_order":
		return h.adminFilesOrder(ctx, c, arg)
	case "fmoveup":
		return h.adminFileMove(ctx, c, arg, arg2, -1)
	case "fmovedown":
		return h.adminFileMove(ctx, c, arg, arg2, 1)
	case "report":
		h.notifyAdminsReport(ctx, c, arg)
		return c.Respond(&tele.CallbackResponse{Text: "⚠️ گزارش شما ثبت شد. ممنون!"})
	}

	return nil
}
