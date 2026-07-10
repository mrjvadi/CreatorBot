package tgbot

import (
	"context"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
)

// onCallback inline keyboard callbacks.
func (h *Handler) onCallback(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
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

	if isAdminOnlyAction(action) && !h.IsAdmin(c) {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyNoAccess), ShowAlert: true})
	}

	switch action {

	case "lang":
		return h.SetLanguage(ctx, c, arg)
	case "back_main":
		_ = c.Respond()
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyDone))
	// ── کیف پول ───────────────────────────────────────────
	case "wallet_topup":
		return h.WalletTopupStart(ctx, c)
	case "topup_amt":
		return h.WalletTopupAmount(ctx, c, arg)
	case "topup_custom":
		return h.WalletTopupCustom(ctx, c)
	case "wallet_history":
		return h.WalletHistory(ctx, c)
	case "wallet_home":
		_ = c.Respond()
		return h.UserWallet(ctx, c)

	// ── پشتیبانی / اطلاعات ───────────────────────────────
	case "user_support_inline":
		_ = c.Respond()
		return h.UserSupportInline(ctx, c)
	case "about_platform":
		_ = c.Respond()
		return h.AboutPlatform(ctx, c)

	// ── ارسال همگانی ──────────────────────────────────────
	case "bc_text":
		return h.BroadcastStartText(ctx, c)
	case "bc_confirm":
		return h.BroadcastConfirm(ctx, c)
	case "bc_forward":
		return h.BroadcastForwardStart(ctx, c)
	case "bc_fwd_confirm":
		return h.BroadcastForwardConfirm(ctx, c)
	case "bc_filtered":
		return h.BroadcastFilteredMenu(ctx, c)
	case "bc_filter":
		return h.BroadcastFilterSelect(ctx, c, arg)

	// ── کیف پول — بررسی پرداخت ───────────────────────────
	// وضعیتِ خودِ تراکنش (با کُد فاکتور) به‌صورت اعلان نشان داده می‌شود؛
	// پیامِ فاکتور و دکمه بسته نمی‌شود تا هر بار قابل بررسی باشد. (arg = code)
	case "topup_check":
		if h.Pay == nil {
			return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyError), ShowAlert: true})
		}
		st, err := h.Pay.InvoiceStatus(ctx, uid, arg)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyTxCheckFailed), ShowAlert: true})
		}
		return c.Respond(h.TxStatusAlert(ctx, uid, st))

	// ── ادمین — سیستم ────────────────────────────────────
	case "admin_sys_info":
		return h.AdminSysInfo(ctx, c)
	case "admin_sys_plans":
		_ = c.Respond()
		return h.AdminPlansList(ctx, c)
	case "admin_sys_servers":
		_ = c.Respond()
		return h.AdminServersList(ctx, c)
	case "admin_sys_templates":
		_ = c.Respond()
		return h.AdminTemplatesList(ctx, c)
	case "admin_sys_sourceworkers":
		_ = c.Respond()
		return h.AdminSourceWorkersList(ctx, c)
	case "admin_sys_promo":
		_ = c.Respond()
		return h.AdminPromoList(ctx, c)
	case "admin_sys_member", "admin_sys_nats", "admin_sys_db", "admin_sys_metrics":
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyComingSoon)})

	// ── ادمین — کدهای پروموشن ─────────────────────────────
	case "admin_promo_add":
		return h.AdminPromoStart(ctx, c)
	case "admin_promo_del":
		defer func() { _ = c.Respond() }()
		return h.AdminPromoDeleteConfirm(ctx, c, arg)
	case "admin_promo_del_do":
		defer func() { _ = c.Respond() }()
		return h.AdminPromoDelete(ctx, c, arg)
	case "admin_promo_toggle":
		return h.AdminPromoToggle(ctx, c, arg)

	// ── ادمین — source-service workerها ──────────────────
	case "admin_sw_add":
		return h.AdminSourceWorkerStart(ctx, c)
	case "admin_sw_del":
		defer func() { _ = c.Respond() }()
		return h.AdminSourceWorkerDeleteConfirm(ctx, c, arg)
	case "admin_sw_del_do":
		defer func() { _ = c.Respond() }()
		return h.AdminSourceWorkerDelete(ctx, c, arg)
	case "admin_sw_toggle":
		return h.AdminSourceWorkerToggle(ctx, c, arg)

	// ── ادمین — عملیاتِ روی بات‌ها (از لیستِ AdminBotsList) ─
	case "admin_bot_stop":
		return h.AdminBotStop(ctx, c, arg)
	case "admin_bot_start":
		return h.AdminBotStart(ctx, c, arg)
	case "admin_bot_del":
		return h.AdminBotDeleteConfirm(ctx, c, arg)
	case "admin_bot_del_do":
		return h.AdminBotDelete(ctx, c, arg)

	// ── کاربر — redeem کدِ پروموشن ────────────────────────
	case "redeem_promo":
		return h.PromoRedeemStart(ctx, c)

	// ── متفرقه (stub) ─────────────────────────────────────
	case "sys_notif":
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyComingSoon)})

	case "sys_lang":
		_ = c.Respond()
		return h.UserLanguageMenu(ctx, c)

	// ── ادمین — افزودن اعتبار ──────────────────────────
	case "add_credit":
		targetTid, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyError)})
		}
		_ = c.Respond()
		return h.AdminCreditStart(ctx, c, targetTid)

	// ── پلن‌ها ────────────────────────────────────────────
	case "show_plans":
		return h.UserPlans(ctx, c)
	case "select_plan":
		return h.UserSelectPlan(ctx, c, arg)
	case "buy_plan":
		return h.ExecutePlanPurchase(ctx, c, arg)
	case "start_free":
		return h.UserStartFree(ctx, c, arg)
	case "check_plan":
		sub := strings.SplitN(arg, ":", 2)
		if len(sub) == 2 {
			return h.CheckPlanAfterDeposit(ctx, c, sub[0], sub[1])
		}
	case "plan_current":
		return h.UserPlans(ctx, c)

	// ── سرویس‌های من ─────────────────────────────────────
	case "my_bots":
		return h.UserBotsList(ctx, c)
	case "svc_create":
		return h.WizardSelectType(ctx, c)
	case "svc_type":
		return h.WizardSelectTag(ctx, c, arg)
	case "svc_tag":
		return h.WizardSelectPlan(ctx, c, arg)
	case "wizard_plan":
		return h.WizardEnterToken(ctx, c, arg)
	case "wizard_pay":
		return h.WizardPay(ctx, c)
	case "wizard_create":
		return h.WizardCreateFree(ctx, c)
	case "svc_status":
		return h.InstanceStatus(ctx, c, arg)

	// ── عملیات روی سرویس ─────────────────────────────────
	case "bot_stop":
		return h.InstanceAction(ctx, c, arg, "stop")
	case "bot_start":
		return h.InstanceAction(ctx, c, arg, "start")
	case "bot_restart":
		return h.InstanceAction(ctx, c, arg, "restart")
	case "bot_delete":
		// نمایش تأیید قبل از حذف
		defer func() { _ = c.Respond() }()
		inst, err := h.Store.FindInstance(ctx, arg)
		if err != nil || inst == nil {
			return c.Edit(h.T(ctx, uid, i18n.KeyInstanceNotFound))
		}
		kb := &tele.ReplyMarkup{}
		kb.Inline(
			kb.Row(
				kb.Data(h.Btn(ctx, uid, i18n.KeyBtnConfirmDelete), "confirm_delete:"+arg),
				kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCancel), "cancel"),
			),
		)
		return c.Edit(
			h.T(ctx, uid, i18n.KeyDeleteConfirm, inst.ContainerName),
			tele.ModeHTML, kb,
		)
	case "confirm_delete":
		return h.InstanceAction(ctx, c, arg, "delete")
	case "svc_settings":
		return h.InstanceSettings(ctx, c, arg)
	case "svc_renew":
		return h.InstanceRenewConfirm(ctx, c, arg)
	case "svc_renew_do":
		return h.InstanceRenewExecute(ctx, c, arg)
	case "svc_stats":
		return h.InstanceStats(ctx, c, arg)

	// ── راهنما ────────────────────────────────────────────
	case "how_to_build":
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnGotIt), "cancel")))
		return c.Edit(h.T(ctx, uid, i18n.KeyHowToBuild), tele.ModeHTML, kb)

	// ── ادمین — کاربران ───────────────────────────────────
	case "block_user":
		return h.AdminUserAction(ctx, c, arg, "block")
	case "unblock_user":
		return h.AdminUserAction(ctx, c, arg, "unblock")
	case "make_admin":
		return h.AdminUserAction(ctx, c, arg, "make_admin")
	case "make_user":
		return h.AdminUserAction(ctx, c, arg, "make_user")
	case "admin_users":
		return h.AdminUsersList(ctx, c)
	case "start_plan_add":
		_ = c.Respond()
		return h.AdminStartAddPlan(ctx, c)

	// ── ویرایش پلن با دکمه‌های ➕➖ ─────────────────────────
	case "plan_edit":
		return h.AdminPlanEdit(ctx, c, arg)
	case "admin_plans_back":
		_ = c.Respond()
		return h.AdminPlansList(ctx, c)
	case "plim_up":
		// arg: planID:botType
		parts2 := strings.SplitN(arg, ":", 2)
		if len(parts2) != 2 {
			return nil
		}
		return h.AdminPlanLimitChange(ctx, c, parts2[0], parts2[1], +1)
	case "plim_dn":
		parts2 := strings.SplitN(arg, ":", 2)
		if len(parts2) != 2 {
			return nil
		}
		return h.AdminPlanLimitChange(ctx, c, parts2[0], parts2[1], -1)
	case "pmb_up":
		return h.AdminPlanMaxBotsChange(ctx, c, arg, +1)
	case "pmb_dn":
		return h.AdminPlanMaxBotsChange(ctx, c, arg, -1)

	case "add_server":
		return h.AdminServerStart(ctx, c)
	case "admin_srv_del":
		return h.AdminServerDeleteConfirm(ctx, c, arg)
	case "admin_srv_del_do":
		defer func() { _ = c.Respond() }()
		return h.AdminServerDelete(ctx, c, arg)
	case "create_tmpl":
		// نوعِ سرویس به‌صورت متن آزاد وارد می‌شود (انواع پویا).
		h.SetStep(ctx, uid, stepTmplType)
		return c.Edit(h.T(ctx, uid, i18n.KeyTemplateAskType),
			tele.ModeHTML, h.KbBackCancel(ctx, uid))
	case "tmpl_test":
		return h.AdminTestStart(ctx, c, arg)
	case "tmpl_schema":
		return h.AdminTemplateSchemaEdit(ctx, c, arg)

	// ── لغو ───────────────────────────────────────────────
	case "cancel":
		h.ClearState(ctx, uid)
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyCancelled))
	}

	return nil
}

// adminOnlyActions لیستِ نامِ کامل اکشن‌هایی که فقط ادمین/مالک مجاز به اجرای
// آن‌هاست. هر اکشنِ جدیدِ مختصِ ادمین باید اینجا (یا با پیشوندِ admin_) اضافه شود.
var adminOnlyActions = map[string]bool{
	"block_user":       true,
	"unblock_user":     true,
	"make_admin":       true,
	"make_user":        true,
	"admin_users":      true,
	"add_credit":       true,
	"add_server":       true,
	"plan_edit":        true,
	"admin_plans_back": true,
	"plim_up":          true,
	"plim_dn":          true,
	"pmb_up":           true,
	"pmb_dn":           true,
	"start_plan_add":   true,
	"create_tmpl":      true,
	"tmpl_test":        true,
	"tmpl_schema":      true,
	// قبلاً این‌ها اینجا نبودند — یعنی نظری فقط با ساختنِ دستیِ callback_data
	// درست (که بدون کلیک روی دکمه‌ی واقعیِ ادمین عملاً امکان‌پذیر نیست، ولی
	// دفاعِ لایه‌ی دومِ درستی نبود) قابلِ صداکردن بودند. حالا مثلِ بقیه‌ی
	// اکشن‌های ادمین صریحاً گارد می‌شوند.
	"bc_text":        true,
	"bc_confirm":     true,
	"bc_forward":     true,
	"bc_fwd_confirm": true,
	"bc_filtered":    true,
	"bc_filter":      true,
}

// isAdminOnlyAction بررسی می‌کند آیا نامِ اکشن (پیش از جداسازیِ arg) مختصِ
// ادمین است. اکشن‌هایی با پیشوندِ "admin_" همیشه مختصِ ادمین‌اند؛ بقیه از
// لیستِ adminOnlyActions خوانده می‌شوند.
func isAdminOnlyAction(action string) bool {
	if strings.HasPrefix(action, "admin_") {
		return true
	}
	return adminOnlyActions[action]
}
