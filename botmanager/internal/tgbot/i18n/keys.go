// Package i18n سیستم چندزبانگی botmanager.
// هر متن یک Key دارد — هر زبان یک map از Key به متن.
// زبان کاربر در Redis ذخیره می‌شود.
package i18n

// Lang کد زبان.
type Lang string

const (
	FA Lang = "fa"
	EN Lang = "en"

	Default = FA
)

// Key نام هر متن.
type Key string

const (
	// ── عمومی ─────────────────────────────────────────────
	KeyCancel    Key = "cancel"
	KeyCancelled Key = "cancelled"
	KeyBack      Key = "back"
	KeyConfirm   Key = "confirm"
	KeyError     Key = "error"
	KeyNotFound  Key = "not_found"
	KeySaved     Key = "saved"
	KeyDeleted   Key = "deleted"

	// ── start / welcome ───────────────────────────────────
	KeyWelcomeAdmin Key = "welcome_admin"
	KeyWelcomeUser  Key = "welcome_user"

	// ── زبان ──────────────────────────────────────────────
	KeySelectLang   Key = "select_lang"
	KeyLangChanged  Key = "lang_changed"

	// ── منوی ادمین ────────────────────────────────────────
	KeyMenuBots      Key = "menu_bots"
	KeyMenuLinks     Key = "menu_links"
	KeyMenuServers   Key = "menu_servers"
	KeyMenuTemplates Key = "menu_templates"
	KeyMenuPlans     Key = "menu_plans"
	KeyMenuUsers     Key = "menu_users"
	KeyMenuStats     Key = "menu_stats"

	// ── منوی کاربر ────────────────────────────────────────
	KeyMenuMyBots  Key = "menu_my_bots"
	KeyMenuSupport Key = "menu_support"
	KeyMenuHelp    Key = "menu_help"

	// ── سرور ──────────────────────────────────────────────
	KeyServersTitle      Key = "servers_title"
	KeyServersEmpty      Key = "servers_empty"
	KeyServerAskName     Key = "server_ask_name"
	KeyServerAskIP       Key = "server_ask_ip"
	KeyServerAdded       Key = "server_added"
	KeyServerDuplicate   Key = "server_duplicate"
	KeyServerAddError    Key = "server_add_error"

	// ── تمپلیت ────────────────────────────────────────────
	KeyTemplatesTitle    Key = "templates_title"
	KeyTemplatesEmpty    Key = "templates_empty"
	KeyTemplateAskType   Key = "template_ask_type"
	KeyTemplateAskImage  Key = "template_ask_image"
	KeyTemplateAskTag    Key = "template_ask_tag"
	KeyTemplateAskName   Key = "template_ask_name"
	KeyTemplateAdded     Key = "template_added"
	KeyTemplateAddError  Key = "template_add_error"

	// ── پلن ───────────────────────────────────────────────
	KeyPlansTitle        Key = "plans_title"
	KeyPlansEmpty        Key = "plans_empty"
	KeyPlansNoTemplate   Key = "plans_no_template"
	KeyPlanAskTemplate   Key = "plan_ask_template"
	KeyPlanTmplNotFound  Key = "plan_tmpl_not_found"
	KeyPlanAskName       Key = "plan_ask_name"
	KeyPlanAskDays       Key = "plan_ask_days"
	KeyPlanAskPrice      Key = "plan_ask_price"
	KeyPlanInvalidNumber Key = "plan_invalid_number"
	KeyPlanAdded         Key = "plan_added"
	KeyPlanAddError      Key = "plan_add_error"

	// ── لینک دعوت ─────────────────────────────────────────
	KeyLinksTitle       Key = "links_title"
	KeyLinksEmpty       Key = "links_empty"
	KeyLinkAskType      Key = "link_ask_type"
	KeyLinkAskLimit     Key = "link_ask_limit"
	KeyLinkAskLabel     Key = "link_ask_label"
	KeyLinkCreated      Key = "link_created"
	KeyLinkCreateError  Key = "link_create_error"

	// ── ربات‌ها (ادمین) ────────────────────────────────────
	KeyBotsTitle        Key = "bots_title"
	KeyBotsEmpty        Key = "bots_empty"
	KeyBotStopped       Key = "bot_stopped"
	KeyBotStarted       Key = "bot_started"
	KeyBotDeleted       Key = "bot_deleted"
	KeyBotNotFound      Key = "bot_not_found"

	// ── کاربران ───────────────────────────────────────────
	KeyUsersTitle       Key = "users_title"
	KeyUsersEmpty       Key = "users_empty"
	KeyUserBlocked      Key = "user_blocked"
	KeyUserUnblocked    Key = "user_unblocked"
	KeyUserMadeAdmin    Key = "user_made_admin"
	KeyUserMadeUser     Key = "user_made_user"

	// ── آمار ──────────────────────────────────────────────
	KeyStatsTitle Key = "stats_title"

	// ── ربات‌های کاربر ────────────────────────────────────
	KeyMyBotsTitle  Key = "my_bots_title"
	KeyMyBotsEmpty  Key = "my_bots_empty"
	KeySupportText  Key = "support_text"
	KeyHelpText     Key = "help_text"

	// ── wizard ────────────────────────────────────────────
	KeyWizardInvalidLink    Key = "wizard_invalid_link"
	KeyWizardExpiredLink    Key = "wizard_expired_link"
	KeyWizardUsedLink       Key = "wizard_used_link"
	KeyWizardConfirm        Key = "wizard_confirm"
	KeyWizardAskToken       Key = "wizard_ask_token"
	KeyWizardInvalidToken   Key = "wizard_invalid_token"
	KeyWizardAlreadyExists  Key = "wizard_already_exists"
	KeyWizardNoServer       Key = "wizard_no_server"
	KeyWizardNoTemplate     Key = "wizard_no_template"
	KeyWizardDeployError    Key = "wizard_deploy_error"
	KeyWizardSuccess        Key = "wizard_success"

	// ── نوع ربات ──────────────────────────────────────────
	KeyBotTypeUploader Key = "bot_type_uploader"
	KeyBotTypeVPN      Key = "bot_type_vpn"
	KeyBotTypeArchive  Key = "bot_type_archive"
	KeyBotTypeMember   Key = "bot_type_member"

	KeyBotDescUploader Key = "bot_desc_uploader"
	KeyBotDescVPN      Key = "bot_desc_vpn"
	KeyBotDescArchive  Key = "bot_desc_archive"
	KeyBotDescMember   Key = "bot_desc_member"


	// ── آمار ادمین ───────────────────────────────────────────
	KeyStatsBotsLine    Key = "stats_bots_line"
	KeyStatsServersLine Key = "stats_servers_line"
	KeyStatsUsersLine   Key = "stats_users_line"

	// ── پلن‌های کاربر ────────────────────────────────────────
	KeyPlansAvailable   Key = "plans_available"
	KeyPlansFree        Key = "plans_free"
	KeyPlansDays        Key = "plans_days"
	KeyPlansEternal     Key = "plans_eternal"
	KeyPlansSelectPrompt Key = "plans_select_prompt"

	// ── کیف پول کاربر ────────────────────────────────────────
	KeyBalanceLine      Key = "balance_line"
	KeyCreditLine       Key = "credit_line"
	KeyPlanLine         Key = "plan_line"
	KeyExpiredSub       Key = "expired_sub"
	KeyEternalSub       Key = "eternal_sub"
	KeyDaysLeft         Key = "days_left"

	// ── خرید پلن ─────────────────────────────────────────────
	KeyNoPlans          Key = "no_plans"
	KeyBuyConfirm       Key = "buy_confirm"
	KeyBuySuccess       Key = "buy_success"
	KeyInsufficientBal  Key = "insufficient_balance"
	KeyNeedDeposit      Key = "need_deposit"
	KeyDepositDone      Key = "deposit_done"
	KeyDepositPending   Key = "deposit_pending"
	KeySubExists        Key = "sub_exists"
	KeyFreePlanActive   Key = "free_plan_active"
	KeyCapacityFull     Key = "capacity_full"
	KeyNoPlan           Key = "no_plan"

	// ── بلاک ─────────────────────────────────────────────────
	KeyBlocked          Key = "blocked"


	// ── ادمین — پلن ──────────────────────────────────────────
	KeyAdminPlanLine    Key = "admin_plan_line"
	KeyAdminPlanFree    Key = "admin_plan_free_badge"
	KeyAdminPlanAdded   Key = "admin_plan_added"
	KeyAdminTemplates   Key = "admin_templates_header"

	// ── ادمین — ربات‌ها ────────────────────────────────────────
	KeyAdminBotSummary  Key = "admin_bot_summary"
	KeyAdminLinkStats   Key = "admin_link_stats"
	KeyAdminLinkLimitX  Key = "admin_link_limit_x"

	// ── ادمین — کاربران ───────────────────────────────────────
	KeyAdminUserSummary Key = "admin_user_summary"
	KeyAdminUserDetail  Key = "admin_user_detail"
	KeyAdminUserBlocked Key = "admin_user_blocked_badge"

	// ── راهنمای ساخت ربات ────────────────────────────────────
	KeyHowToBuild       Key = "how_to_build"
	KeyHowToBuildDone   Key = "how_to_build_done"
	KeyNoFreePlan       Key = "no_free_plan"

	// ── تمپلیت رایگان ────────────────────────────────────────
	KeyTmplFreeAdded    Key = "tmpl_free_added"
	KeyTmplFreeExists   Key = "tmpl_free_exists"

	KeySubActiveNoBot   Key = "sub_active_no_bot"
	KeyBuildWithLink     Key = "build_with_link"

	// ── دکمه‌ها ───────────────────────────────────────────
	KeyBtnYesBuild   Key = "btn_yes_build"
	KeyBtnCancel     Key = "btn_cancel"
	KeyBtnBack       Key = "btn_back"
	KeyBtnLimit1     Key = "btn_limit_1"
	KeyBtnLimit3     Key = "btn_limit_3"
	KeyBtnLimit5     Key = "btn_limit_5"
	KeyBtnLimit10    Key = "btn_limit_10"
	KeyBtnLimitNo    Key = "btn_limit_no"
	KeyBtnBlock      Key = "btn_block"
	KeyBtnUnblock    Key = "btn_unblock"
	KeyBtnMakeAdmin  Key = "btn_make_admin"
	KeyBtnMakeUser   Key = "btn_make_user"
)
