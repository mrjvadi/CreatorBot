// Package i18n سیستم چندزبانگی botmanager.
package i18n

type Key string

const (
	// ── سیستم ────────────────────────────────────────────
	KeyError     Key = "error"
	KeyCancelled Key = "cancelled"
	KeyDone      Key = "done"
	KeyBack      Key = "back"
	KeyCancel    Key = "cancel"
	KeyConfirm   Key = "confirm"
	KeyLoading   Key = "loading"
	KeyNotFound  Key = "not_found"
	KeyNoAccess  Key = "no_access"
	KeyComingSoon Key = "coming_soon"
	KeyAccountTitle   Key = "account_title"
	KeyLanguageSelect Key = "language_select"
	KeyBroadcastMenu  Key = "broadcast_menu"
	KeySystemMenu     Key = "system_menu"

	// ── منوی اصلی کاربر ──────────────────────────────────
	KeyMenuWallet        Key = "menu_wallet"
	KeyMenuServices      Key = "menu_services"
	KeyMenuCommunities   Key = "menu_communities"
	KeyMenuAds           Key = "menu_ads"
	KeyMenuEarnings      Key = "menu_earnings"
	KeyMenuPlans         Key = "menu_plans"
	KeyMenuNotifications Key = "menu_notifications"
	KeyMenuSettings      Key = "menu_settings"
	KeyMenuHelp          Key = "menu_help"
	KeyMenuSupport       Key = "menu_support"
	KeyMenuMyBots        Key = "menu_my_bots"
	KeyMenuCreateBot     Key = "menu_create_bot"
	KeyMenuAccount       Key = "menu_account"
	KeyMenuLanguage      Key = "menu_language"
	KeyMenuTutorials     Key = "menu_tutorials"

	// ── منوی اصلی ادمین ──────────────────────────────────
	KeyMenuUsers     Key = "menu_users"
	KeyMenuCampaigns Key = "menu_campaigns"
	KeyMenuFinance   Key = "menu_finance"
	KeyMenuFraud     Key = "menu_fraud"
	KeyMenuStats     Key = "menu_stats"
	KeyMenuSystem    Key = "menu_system"
	KeyMenuBroadcast Key = "menu_broadcast"
	KeyMenuExitAdmin Key = "menu_exit_admin"
	// ادمین هم از KeyMenuServices و KeyMenuCommunities استفاده می‌کند

	// ── قدیمی (backward compat) ───────────────────────────
	KeyMenuBots      Key = "menu_bots"
	KeyMenuLinks     Key = "menu_links"
	KeyMenuServers   Key = "menu_servers"
	KeyMenuTemplates Key = "menu_templates"

	// ── خوش‌آمد و /start ──────────────────────────────────
	KeyWelcomeUser  Key = "welcome_user"
	KeyWelcomeAdmin Key = "welcome_admin"
	KeyHelpText     Key = "help_text"
	KeyHelpAdmin    Key = "help_admin"

	// ── کیف پول ──────────────────────────────────────────
	KeyWalletHome       Key = "wallet_home"
	KeyWalletDeposit    Key = "wallet_deposit"
	KeyWalletWithdraw   Key = "wallet_withdraw"
	KeyWalletTransfer   Key = "wallet_transfer"
	KeyWalletHistory    Key = "wallet_history"
	KeyWalletRewards    Key = "wallet_rewards"
	KeyWalletLowBalance Key = "wallet_low_balance"

	// ── سرویس‌ها ─────────────────────────────────────────
	KeyServicesHome      Key = "services_home"
	KeyServicesEmpty     Key = "services_empty"
	KeyServiceCreate     Key = "service_create"
	KeyServiceSelectType Key = "service_select_type"
	KeyServiceSelectPlan Key = "service_select_plan"
	KeyServiceEnterToken Key = "service_enter_token"
	KeyServiceConfirm    Key = "service_confirm"
	KeyServiceCreating   Key = "service_creating"
	KeyServiceCreated    Key = "service_created"
	KeyServiceFailed     Key = "service_failed"
	KeyServiceNoCapacity Key = "service_no_capacity"
	KeyServiceInvalidToken Key = "service_invalid_token"
	KeyServiceDuplicate  Key = "service_duplicate"

	// ── پلن‌ها ────────────────────────────────────────────
	KeyPlansHome    Key = "plans_home"
	KeyPlanCurrent  Key = "plan_current"
	KeyPlanNone     Key = "plan_none"
	KeyPlanExpired  Key = "plan_expired"
	KeyPlanUpgrade  Key = "plan_upgrade"
	KeyPlanBuyTitle Key = "plan_buy_title"
	KeyPlanBought   Key = "plan_bought"
	KeyNoFreePlan   Key = "no_free_plan"
	KeyFreePlanDone Key = "free_plan_done"

	// ── کامیونیتی‌ها ─────────────────────────────────────
	KeyCommHome     Key = "comm_home"
	KeyCommEmpty    Key = "comm_empty"
	KeyCommRegister Key = "comm_register"
	KeyCommVerify   Key = "comm_verify"

	// ── تبلیغات ──────────────────────────────────────────
	KeyAdsHome    Key = "ads_home"
	KeyAdsEmpty   Key = "ads_empty"
	KeyAdsCreate  Key = "ads_create"

	// ── درآمدها ──────────────────────────────────────────
	KeyEarningsHome  Key = "earnings_home"
	KeyEarningsEmpty Key = "earnings_empty"

	// ── تنظیمات ──────────────────────────────────────────
	KeySettingsHome Key = "settings_home"
	KeyLangChanged  Key = "lang_changed"
	KeyLangSelect   Key = "lang_select"

	// ── اعلان‌ها ──────────────────────────────────────────
	KeyNotificationsHome Key = "notifications_home"

	// ── پشتیبانی ─────────────────────────────────────────
	KeySupportText Key = "support_text"

	// ── ادمین — کاربران ──────────────────────────────────
	KeyAdminUsersTitle  Key = "admin_users_title"
	KeyAdminUserDetail  Key = "admin_user_detail"
	KeyAdminUserBlocked Key = "admin_user_blocked"
	KeyAdminUserUnblocked Key = "admin_user_unblocked"

	// ── ادمین — ربات‌ها ────────────────────────────────────
	KeyBotsEmpty Key = "bots_empty"

	// ── ادمین — سرور ────────────────────────────────────
	KeyServerAskName Key = "server_ask_name"
	KeyServerAskIP   Key = "server_ask_ip"
	KeyServerAdded   Key = "server_added"

	// ── ادمین — تمپلیت ────────────────────────────────────
	KeyTemplateAskType  Key = "tmpl_ask_type"
	KeyTemplateAskImage Key = "tmpl_ask_image"
	KeyTemplateAskTag   Key = "tmpl_ask_tag"
	KeyTmplFreeAdded    Key = "tmpl_free_added"
	KeyTmplFreeExists   Key = "tmpl_free_exists"

	// ── ادمین — پلن ──────────────────────────────────────
	KeyPlanAskName  Key = "plan_ask_name"
	KeyPlanAskPrice Key = "plan_ask_price"
	KeyPlanAskDays  Key = "plan_ask_days"
	KeyPlanAskBots  Key = "plan_ask_bots"
	KeyPlanAdded    Key = "plan_added"

	// ── ادمین — لینک دعوت ────────────────────────────────
	KeyLinkAskType  Key = "link_ask_type"
	KeyLinkAskLimit Key = "link_ask_limit"
	KeyLinkCreated  Key = "link_created"

	// ── آمار ────────────────────────────────────────────
	KeyStatsTitle Key = "stats_title"

	// ── ساب‌اسکریپشن ────────────────────────────────────
	KeySubActiveNoBot Key = "sub_active_no_bot"
	KeyBuildWithLink  Key = "build_with_link"

	// ── wizard ──────────────────────────────────────────
	KeyWizardInvalidLink Key = "wizard_invalid_link"
	KeyWizardExpiredLink Key = "wizard_expired_link"
	KeyWizardUsedLink    Key = "wizard_used_link"
	KeyWizardAskToken    Key = "wizard_ask_token"

	// ── How-to ──────────────────────────────────────────
	KeyHowToBuild     Key = "how_to_build"
	KeyHowToBuildDone Key = "how_to_build_done"

	// ── دکمه‌ها ───────────────────────────────────────────
	KeyBtnYesBuild  Key = "btn_yes_build"
	KeyBtnCancel    Key = "btn_cancel"
	KeyBtnBack      Key = "btn_back"
	KeyBtnLimit1    Key = "btn_limit_1"
	KeyBtnLimit3    Key = "btn_limit_3"
	KeyBtnLimit5    Key = "btn_limit_5"
	KeyBtnLimit10   Key = "btn_limit_10"
	KeyBtnLimitNo   Key = "btn_limit_no"
	KeyBtnBlock     Key = "btn_block"
	KeyBtnUnblock   Key = "btn_unblock"
	KeyBtnMakeAdmin Key = "btn_make_admin"
	KeyBtnMakeUser  Key = "btn_make_user"
	// ── ادمین — ربات‌ها ───────────────────────────────────
	KeyBotsTitle        Key = "bots_title"
	KeyAdminBotSummary  Key = "admin_bot_summary"
	KeyBotNotFound      Key = "bot_not_found"
	KeyBotStopped       Key = "bot_stopped"
	KeyBotStarted       Key = "bot_started"
	KeyBotDeleted       Key = "bot_deleted"

	// ── ادمین — لینک‌ها ───────────────────────────────────
	KeyLinksTitle       Key = "links_title"
	KeyLinksEmpty       Key = "links_empty"
	KeyLinkAskLabel     Key = "link_ask_label"
	KeyLinkCreateError  Key = "link_create_error"
	KeyAdminLinkStats   Key = "admin_link_stats"
	KeyAdminLinkLimitX  Key = "admin_link_limit_x"

	// ── ادمین — پلن‌ها ───────────────────────────────────
	KeyPlansTitle       Key = "plans_title"
	KeyPlansEmpty       Key = "plans_empty"
	KeyPlansNoTemplate  Key = "plans_no_template"
	KeyPlanAskTemplate  Key = "plan_ask_template"
	KeyPlanInvalidNumber Key = "plan_invalid_number"
	KeyPlanTmplNotFound Key = "plan_tmpl_not_found"
	KeyPlanAddError     Key = "plan_add_error"
	KeyAdminPlanLine    Key = "admin_plan_line"
	KeyAdminPlanFree    Key = "admin_plan_free"

	// ── ادمین — تمپلیت‌ها ─────────────────────────────────
	KeyTemplatesTitle   Key = "templates_title"
	KeyTemplatesEmpty   Key = "templates_empty"
	KeyTemplateAskName  Key = "template_ask_name"
	KeyTemplateAdded    Key = "template_added"
	KeyTemplateAddError Key = "template_add_error"
	KeyAdminTemplates   Key = "admin_templates"

	// ── ادمین — سرورها ────────────────────────────────────
	KeyServersTitle     Key = "servers_title"
	KeyServersEmpty     Key = "servers_empty"
	KeyServerAddError   Key = "server_add_error"
	KeyServerDuplicate  Key = "server_duplicate"

	// ── ادمین — کاربران ───────────────────────────────────
	KeyUsersTitle       Key = "users_title"
	KeyUsersEmpty       Key = "users_empty"
	KeyAdminUserSummary Key = "admin_user_summary"
	KeyUserBlocked      Key = "user_blocked"
	KeyUserUnblocked    Key = "user_unblocked"
	KeyUserMadeAdmin    Key = "user_made_admin"
	KeyUserMadeUser     Key = "user_made_user"
	KeyBlocked          Key = "blocked"

	// ── آمار ─────────────────────────────────────────────
	KeyStatsBotsLine    Key = "stats_bots_line"
	KeyStatsServersLine Key = "stats_servers_line"
	KeyStatsUsersLine   Key = "stats_users_line"

	// ── نوع ربات ─────────────────────────────────────────
	KeyBotTypeVPN       Key = "bot_type_vpn"
	KeyBotTypeUploader  Key = "bot_type_uploader"
	KeyBotTypeMember    Key = "bot_type_member"
	KeyBotTypeArchive   Key = "bot_type_archive"

	// ── متفرقه ───────────────────────────────────────────
	KeyNoPlan           Key = "no_plan"
	KeySelectLang       Key = "select_lang"
	KeyBtnLimit         Key = "btn_limit"

)
