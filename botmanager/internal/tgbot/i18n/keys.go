// Package i18n سیستم چندزبانگی botmanager.
package i18n

type Key string

const (
	// ── سیستم ────────────────────────────────────────────
	KeyError                 Key = "error"
	KeyCancelled             Key = "cancelled"
	KeyDone                  Key = "done"
	KeyBack                  Key = "back"
	KeyCancel                Key = "cancel"
	KeyConfirm               Key = "confirm"
	KeyLoading               Key = "loading"
	KeyNotFound              Key = "not_found"
	KeyNoAccess              Key = "no_access"
	KeyComingSoon            Key = "coming_soon"
	KeyAccountTitle          Key = "account_title"
	KeyAccountStatusStandard Key = "account_status_standard"
	KeyAccountStatusVIP      Key = "account_status_vip"
	KeyLanguageSelect        Key = "language_select"
	KeyBroadcastMenu         Key = "broadcast_menu"
	KeySystemMenu            Key = "system_menu"

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
	KeyServicesHome        Key = "services_home"
	KeyServicesEmpty       Key = "services_empty"
	KeyServiceCreate       Key = "service_create"
	KeyServiceSelectType   Key = "service_select_type"
	KeyServiceSelectTag    Key = "service_select_tag"
	KeyServiceSelectPlan   Key = "service_select_plan"
	KeyServiceEnterToken   Key = "service_enter_token"
	KeyServiceConfirm      Key = "service_confirm"
	KeyServiceCreating     Key = "service_creating"
	KeyServiceCreated      Key = "service_created"
	KeyServiceFailed       Key = "service_failed"
	KeyServiceNoCapacity   Key = "service_no_capacity"
	KeyServiceInvalidToken Key = "service_invalid_token"
	KeyServiceDuplicate    Key = "service_duplicate"

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
	KeyAdsHome   Key = "ads_home"
	KeyAdsEmpty  Key = "ads_empty"
	KeyAdsCreate Key = "ads_create"

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
	KeyAdminUsersTitle    Key = "admin_users_title"
	KeyAdminUserDetail    Key = "admin_user_detail"
	KeyAdminUserBlocked   Key = "admin_user_blocked"
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
	KeyBtnYesBuild    Key = "btn_yes_build"
	KeyBtnCancel      Key = "btn_cancel"
	KeyBtnBack        Key = "btn_back"
	KeyBtnLimit1      Key = "btn_limit_1"
	KeyBtnLimit3      Key = "btn_limit_3"
	KeyBtnLimit5      Key = "btn_limit_5"
	KeyBtnLimit10     Key = "btn_limit_10"
	KeyBtnLimitNo     Key = "btn_limit_no"
	KeyBtnBlock       Key = "btn_block"
	KeyBtnUnblock     Key = "btn_unblock"
	KeyBtnMakeAdmin   Key = "btn_make_admin"
	KeyBtnMakeUser    Key = "btn_make_user"
	KeyBtnTest        Key = "btn_test"
	KeyBtnNewTemplate Key = "btn_new_template"

	// ── دکمه‌ها (UI مشترک) ───────────────────────────────
	KeyBtnAddCredit        Key = "btn_add_credit"
	KeyBtnBackToList       Key = "btn_back_to_list"
	KeyBtnAddServer        Key = "btn_add_server"
	KeyBtnViewWallet       Key = "btn_view_wallet"
	KeyBtnConfirmDelete    Key = "btn_confirm_delete"
	KeyBtnGotIt            Key = "btn_got_it"
	KeyBalanceUpdated      Key = "balance_updated"
	KeyBalanceAlert        Key = "balance_alert"
	KeyPaymentPendingAlert Key = "payment_pending_alert"
	KeyTxPending           Key = "tx_pending"
	KeyTxPaid              Key = "tx_paid"
	KeyTxPartial           Key = "tx_partial"
	KeyTxExpired           Key = "tx_expired"
	KeyTxNotFound          Key = "tx_not_found"
	KeyTxCheckFailed       Key = "tx_check_failed"
	KeyBtnDepositTON       Key = "btn_deposit_ton"
	KeyBtnHistory          Key = "btn_history"
	KeyBtnRedeemPromo      Key = "btn_redeem_promo"
	KeyBtnNewDeposit       Key = "btn_new_deposit"
	KeyBtnCheckPayment     Key = "btn_check_payment"
	KeyBtnBcText           Key = "btn_bc_text"
	KeyBtnBcForward        Key = "btn_bc_forward"
	KeyBtnBcFiltered       Key = "btn_bc_filtered"
	KeyBtnCreateFree       Key = "btn_create_free"
	KeyBtnPayCreate        Key = "btn_pay_create"

	// ── عمومی / وضعیت ────────────────────────────────────
	KeyFree           Key = "free"
	KeyDaysCount      Key = "days_count"
	KeyForever        Key = "forever"
	KeyStatusActive   Key = "status_active"
	KeyStatusInactive Key = "status_inactive"
	KeyErrShort       Key = "err_short"
	KeyErrSave        Key = "err_save"

	// ── ادمین — پلن (ادیتور) ─────────────────────────────
	KeyPlanNotFound       Key = "plan_not_found"
	KeyBtnNewPlan         Key = "btn_new_plan"
	KeyBtnBackToPlans     Key = "btn_back_to_plans"
	KeyBtnEditPlan        Key = "btn_edit_plan"
	KeyBtnTotalCap        Key = "btn_total_cap"
	KeyAdminPlanRow       Key = "admin_plan_row"
	KeyPlanEditTitle      Key = "plan_edit_title"
	KeyAvailableTemplates Key = "available_templates"
	KeyPlanTmplChosen     Key = "plan_tmpl_chosen"
	KeyPlanLimitsPrompt   Key = "plan_limits_prompt"
	KeyPlanLimitsInvalid  Key = "plan_limits_invalid"
	KeyPlanLimitsSaved    Key = "plan_limits_saved"

	// ── کاربر — پلن‌ها (UI) ──────────────────────────────
	KeyDurationForever          Key = "duration_forever"
	KeyBtnMyBots                Key = "btn_my_bots"
	KeyBtnTopupWallet           Key = "btn_topup_wallet"
	KeyBtnRecheck               Key = "btn_recheck"
	KeyBtnClose                 Key = "btn_close"
	KeyBtnBuyWith               Key = "btn_buy_with"
	KeyPlansUnavailable         Key = "plans_unavailable"
	KeyPlansAvailableTitle      Key = "plans_available_title"
	KeyPlanRemDays              Key = "plan_rem_days"
	KeyPlanExpiredShort         Key = "plan_expired_short"
	KeyPlanActiveYours          Key = "plan_active_yours"
	KeyPlanRow                  Key = "plan_row"
	KeyPlansClickToBuy          Key = "plans_click_to_buy"
	KeyPlanLabelFree            Key = "plan_label_free"
	KeyPlanLabelPaid            Key = "plan_label_paid"
	KeyPlanAlreadyActive        Key = "plan_already_active"
	KeyPlanDetail               Key = "plan_detail"
	KeyWalletBalanceLine        Key = "wallet_balance_line"
	KeyBalanceEnough            Key = "balance_enough"
	KeyBalanceShortfall         Key = "balance_shortfall"
	KeyDepositAddrCode          Key = "deposit_addr_code"
	KeyPayServiceUnavailable    Key = "pay_service_unavailable"
	KeyFreePlanActivated        Key = "free_plan_activated"
	KeyPlanPurchaseDesc         Key = "plan_purchase_desc"
	KeyPurchaseSuccess          Key = "purchase_success"
	KeyPaymentNotConfirmed      Key = "payment_not_confirmed"
	KeyPurchaseActivationFailed Key = "purchase_activation_failed"

	// ── UX — ویزارد و badgeها ────────────────────────────
	KeyWizardStep      Key = "wizard_step"
	KeyBadgePopular    Key = "badge_popular"
	KeyBadgeNewest     Key = "badge_newest"
	KeyBtnCustomAmount Key = "btn_custom_amount"
	KeyBtnRenew        Key = "btn_renew"

	// ── تمدید سرویس و یادآور انقضا ───────────────────────
	KeyRenewConfirm    Key = "renew_confirm"
	KeyBtnConfirmRenew Key = "btn_confirm_renew"
	KeyRenewDone       Key = "renew_done"
	KeyRenewNoPlan     Key = "renew_no_plan"
	KeyExpiryReminder  Key = "expiry_reminder"

	// ── کاربر — سرویس‌های من (UI) ────────────────────────
	KeyBtnStats           Key = "btn_stats"
	KeyBtnSettings        Key = "btn_settings"
	KeyBtnRestart         Key = "btn_restart"
	KeyBtnStop            Key = "btn_stop"
	KeyBtnStart           Key = "btn_start"
	KeyBtnDeleteSvc       Key = "btn_delete_svc"
	KeyBtnDelete          Key = "btn_delete"
	KeyBtnCheckStatus     Key = "btn_check_status"
	KeyBtnRetry           Key = "btn_retry"
	KeyBtnCreateSvc       Key = "btn_create_svc"
	KeyBtnCreateNewSvc    Key = "btn_create_new_svc"
	KeyBtnStartFree       Key = "btn_start_free"
	KeyBtnViewPlans       Key = "btn_view_plans"
	KeyBtnUpgradePlan     Key = "btn_upgrade_plan"
	KeyMyServicesHeader   Key = "my_services_header"
	KeySvcNameLine        Key = "svc_name_line"
	KeySvcStatusLine      Key = "svc_status_line"
	KeySvcExpiredNL       Key = "svc_expired_nl"
	KeySvcHoursLeft       Key = "svc_hours_left"
	KeySvcDaysLeft        Key = "svc_days_left"
	KeyWelcomeNoService   Key = "welcome_no_service"
	KeyNeedPlanFirst      Key = "need_plan_first"
	KeyMaxBotsReached     Key = "max_bots_reached"
	KeyStatusRunning      Key = "status_running"
	KeyStatusStopped      Key = "status_stopped"
	KeyStatusStarting     Key = "status_starting"
	KeyStatusErrContact   Key = "status_err_contact"
	KeyTypeNotAllowed     Key = "type_not_allowed"
	KeyMaxBotsReachedType Key = "max_bots_reached_type"
	KeyActionStopSent     Key = "action_stop_sent"
	KeyActionStartSent    Key = "action_start_sent"
	KeyActionRestartSent  Key = "action_restart_sent"
	KeyActionDeleteSent   Key = "action_delete_sent"
	KeySvcStatusShort     Key = "svc_status_short"
	KeySvcStatsDetail     Key = "svc_stats_detail"
	KeyServiceGeneric     Key = "service_generic"
	KeyUnknown            Key = "unknown"
	KeyExpiredLabel       Key = "expired_label"
	KeyDaysUntilExpiry    Key = "days_until_expiry"
	KeyPlanLine           Key = "plan_line"

	// ── ادمین — تست سرویس ────────────────────────────────
	KeyAdminTestAskToken Key = "admin_test_ask_token"
	KeyAdminTestDeployed Key = "admin_test_deployed"
	// ── ادمین — ربات‌ها ───────────────────────────────────
	KeyBotsTitle       Key = "bots_title"
	KeyAdminBotSummary Key = "admin_bot_summary"
	KeyBotNotFound     Key = "bot_not_found"
	KeyBotStopped      Key = "bot_stopped"
	KeyBotStarted      Key = "bot_started"
	KeyBotDeleted      Key = "bot_deleted"
	KeyBotActionFailed Key = "bot_action_failed"

	// ── ادمین — لینک‌ها ───────────────────────────────────
	KeyLinksTitle      Key = "links_title"
	KeyLinksEmpty      Key = "links_empty"
	KeyLinkAskLabel    Key = "link_ask_label"
	KeyLinkCreateError Key = "link_create_error"
	KeyAdminLinkStats  Key = "admin_link_stats"
	KeyAdminLinkLimitX Key = "admin_link_limit_x"

	// ── ادمین — پلن‌ها ───────────────────────────────────
	KeyPlansTitle        Key = "plans_title"
	KeyPlansEmpty        Key = "plans_empty"
	KeyPlansNoTemplate   Key = "plans_no_template"
	KeyPlanAskTemplate   Key = "plan_ask_template"
	KeyPlanInvalidNumber Key = "plan_invalid_number"
	KeyPlanTmplNotFound  Key = "plan_tmpl_not_found"
	KeyPlanAddError      Key = "plan_add_error"
	KeyAdminPlanLine     Key = "admin_plan_line"
	KeyAdminPlanFree     Key = "admin_plan_free"

	// ── ادمین — تمپلیت‌ها ─────────────────────────────────
	KeyTemplatesTitle   Key = "templates_title"
	KeyTemplatesEmpty   Key = "templates_empty"
	KeyTemplateAskName  Key = "template_ask_name"
	KeyTemplateAdded    Key = "template_added"
	KeyTemplateAddError Key = "template_add_error"
	KeyAdminTemplates   Key = "admin_templates"

	// ── ادمین — سرورها ────────────────────────────────────
	KeyServersTitle        Key = "servers_title"
	KeyServersEmpty        Key = "servers_empty"
	KeyServerAddError      Key = "server_add_error"
	KeyServerDuplicate     Key = "server_duplicate"
	KeyServerDeleteConfirm Key = "server_delete_confirm"
	KeyServerDeletedMsg    Key = "server_deleted_msg"

	// ── ادمین — کاربران ───────────────────────────────────
	KeyUsersTitle         Key = "users_title"
	KeyUsersEmpty         Key = "users_empty"
	KeyUsersSearchPrompt  Key = "users_search_prompt"
	KeyUsersSearchInvalid Key = "users_search_invalid"
	KeyAdminUserSummary   Key = "admin_user_summary"
	KeyUserBlocked        Key = "user_blocked"
	KeyUserUnblocked      Key = "user_unblocked"
	KeyUserMadeAdmin      Key = "user_made_admin"
	KeyUserMadeUser       Key = "user_made_user"
	KeyBlocked            Key = "blocked"

	// ── آمار ─────────────────────────────────────────────
	KeyStatsBotsLine    Key = "stats_bots_line"
	KeyStatsServersLine Key = "stats_servers_line"
	KeyStatsUsersLine   Key = "stats_users_line"

	// ── نوع ربات ─────────────────────────────────────────
	KeyBotTypeVPN      Key = "bot_type_vpn"
	KeyBotTypeUploader Key = "bot_type_uploader"
	KeyBotTypeMember   Key = "bot_type_member"
	KeyBotTypeArchive  Key = "bot_type_archive"

	// ── متفرقه ───────────────────────────────────────────
	KeyNoPlan     Key = "no_plan"
	KeySelectLang Key = "select_lang"
	KeyBtnLimit   Key = "btn_limit"

	// ── ادمین — افزودن اعتبار ────────────────────────────
	KeyAdminCreditAsk     Key = "admin_credit_ask"
	KeyAdminCreditDone    Key = "admin_credit_done"
	KeyAdminCreditError   Key = "admin_credit_error"
	KeyAdminCreditInvalid Key = "admin_credit_invalid"

	// ── wizard — خطاها ───────────────────────────────────
	KeyWizardNoPlan      Key = "wizard_no_plan"
	KeyWizardRestart     Key = "wizard_restart"
	KeyWizardNoServer    Key = "wizard_no_server"
	KeyWizardNoTemplate  Key = "wizard_no_template"
	KeyWizardCreateError Key = "wizard_create_error"
	KeyWizardDeployError Key = "wizard_deploy_error"
	KeyWizardIncomplete  Key = "wizard_incomplete"
	KeyWizardLowBalance  Key = "wizard_low_balance"
	KeyInstanceNotFound  Key = "instance_not_found"
	KeyInstanceNoAccess  Key = "instance_no_access"

	// ── کیف پول — صفحه اصلی ───────────────────────────────
	KeyWalletTitle Key = "wallet_title"

	// ── تنظیمات ───────────────────────────────────────────
	KeySettingsLanguage Key = "settings_language"
	KeySettingsSupport  Key = "settings_support"
	KeySettingsAbout    Key = "settings_about"

	// ── سرویس — تنظیمات ──────────────────────────────────
	KeySvcSettingsDetail Key = "svc_settings_detail"

	// ── حذف — تأیید ──────────────────────────────────────
	KeyDeleteConfirm Key = "delete_confirm"
	KeyDeleteDone    Key = "delete_done"

	// ── واریز کیف پول ────────────────────────────────────
	KeyWalletTopupAsk     Key = "wallet_topup_ask"
	KeyWalletTopupInvoice Key = "wallet_topup_invoice"
	KeyWalletTopupInvalid Key = "wallet_topup_invalid"

	// ── تاریخچه (stub) ───────────────────────────────────
	KeyWalletHistoryNote Key = "wallet_history_note"

	// ── پشتیبانی و اطلاعات (inline) ──────────────────────
	KeySupportInline Key = "support_inline"
	KeyAboutPlatform Key = "about_platform"

	// ── ادمین — ارسال همگانی ─────────────────────────────
	KeyBroadcastAskText        Key = "broadcast_ask_text"
	KeyBroadcastPreview        Key = "broadcast_preview"
	KeyBroadcastDone           Key = "broadcast_done"
	KeyBroadcastStarted        Key = "broadcast_started"
	KeyBroadcastConfirm        Key = "bc_confirm"
	KeyBroadcastForwardAsk     Key = "broadcast_forward_ask"
	KeyBroadcastForwardPreview Key = "broadcast_forward_preview"
	KeyBroadcastEmptyAudience  Key = "broadcast_empty_audience"
	KeyBcFilterTitle           Key = "bc_filter_title"
	KeyBcFilterAll             Key = "bc_filter_all"
	KeyBcFilterNoPlan          Key = "bc_filter_no_plan"
	KeyBcFilterPlan            Key = "bc_filter_plan"

	// ── ادمین — سیستم ────────────────────────────────────
	KeyAdminSysInfo Key = "admin_sys_info"

	// ── کدهای پروموشن ─────────────────────────────────────
	KeyMenuPromoCodes     Key = "menu_promo_codes"
	KeyBtnAddPromo        Key = "btn_add_promo"
	KeyPromoAsk           Key = "promo_ask"
	KeyPromoNotFound      Key = "promo_not_found"
	KeyPromoAlreadyUsed   Key = "promo_already_used"
	KeyPromoExpiredOrFull Key = "promo_expired_or_full"
	KeyPromoCreditFailed  Key = "promo_credit_failed"
	KeyPromoRedeemed      Key = "promo_redeemed"
	KeyPromoAdminTitle    Key = "promo_admin_title"
	KeyPromoAdminEmpty    Key = "promo_admin_empty"
	KeyPromoAskCode       Key = "promo_ask_code"
	KeyPromoAskAmount     Key = "promo_ask_amount"
	KeyPromoAskMaxUses    Key = "promo_ask_max_uses"
	KeyPromoAskDays       Key = "promo_ask_days"
	KeyPromoCreateError   Key = "promo_create_error"
	KeyPromoDuplicate     Key = "promo_duplicate"
	KeyPromoCreated       Key = "promo_created"
	KeyPromoDeleteConfirm Key = "promo_delete_confirm"
	KeyPromoDeleted       Key = "promo_deleted"

	// ── ادمین — source-service worker ────────────────────
	KeyMenuSourceWorkers  Key = "menu_source_workers"
	KeyBtnAddSourceWorker Key = "btn_add_source_worker"
	KeyBtnDeleteSW        Key = "btn_delete_sw"
	KeyBtnToggleSW        Key = "btn_toggle_sw"
	KeySWTitle            Key = "sw_title"
	KeySWEmpty            Key = "sw_empty"
	KeySWAskAppID         Key = "sw_ask_app_id"
	KeySWAskAppHash       Key = "sw_ask_app_hash"
	KeySWAskPhone         Key = "sw_ask_phone"
	KeySWAskLabel         Key = "sw_ask_label"
	KeySWInvalidAppID     Key = "sw_invalid_app_id"
	KeySWCreated          Key = "sw_created"
	KeySWCreateError      Key = "sw_create_error"
	KeySWDeleted          Key = "sw_deleted"
	KeySWToggledOn        Key = "sw_toggled_on"
	KeySWToggledOff       Key = "sw_toggled_off"
	KeySWNotFound         Key = "sw_not_found"
	KeySWDeleteConfirm    Key = "sw_delete_confirm"

	// ── wizard — تنظیمات اختصاصی (ConfigSchema) ──────────
	KeyWizardConfigField Key = "wiz_config_field"
	KeyWizardConfigDone  Key = "wiz_config_done"

	// ── ادمین — ویرایش ConfigSchema قالب ─────────────────
	KeyTmplAskSchema    Key = "tmpl_ask_schema"
	KeyTmplSchemaSet    Key = "tmpl_schema_set"
	KeyTmplSchemaInvalid Key = "tmpl_schema_invalid"
	KeyBtnEditSchema    Key = "btn_edit_schema"
)
