package i18n

var en = map[Key]string{
	// ── General ───────────────────────────────────────────
	KeyCancel:    "❌ Cancel",
	KeyCancelled: "Cancelled.",
	KeyBack:      "🔙 Back",
	KeyConfirm:   "✅ Confirm",
	KeyError:     "An error occurred. Please try again.",
	KeyNotFound:  "Not found.",
	KeySaved:     "✅ Saved successfully.",
	KeyDeleted:   "🗑 Deleted.",

	// ── Start / Welcome ───────────────────────────────────
	KeyWelcomeAdmin: "Hello %s\nWelcome to CreatorBot admin panel 👑",
	KeyWelcomeUser:  "Hello %s 👋\nWith CreatorBot you can build custom Telegram bots.",

	// ── Language ──────────────────────────────────────────
	KeySelectLang:  "زبان خود را انتخاب کنید:\nSelect your language:",
	KeyLangChanged: "✅ Language changed to English.",

	// ── Admin Menu ────────────────────────────────────────
	KeyMenuBots:      "🤖 Bots",
	KeyMenuLinks:     "🔗 Invite Links",
	KeyMenuServers:   "🖥 Servers",
	KeyMenuTemplates: "📦 Templates",
	KeyMenuPlans:     "💰 Plans",
	KeyMenuUsers:     "👥 Users",
	KeyMenuStats:     "📊 Statistics",

	// ── User Menu ─────────────────────────────────────────
	KeyMenuMyBots:  "🤖 My Bots",
	KeyMenuSupport: "📞 Support",
	KeyMenuHelp:    "❓ Help",

	// ── Server ────────────────────────────────────────────
	KeyServersTitle:    "<b>🖥 Servers</b>",
	KeyServersEmpty:    "No servers registered.",
	KeyServerAskName:   "Enter server name:\nExample: <code>server-de1</code>",
	KeyServerAskIP:     "Enter server IP address:\nExample: <code>1.2.3.4</code>",
	KeyServerAdded:     "✅ <b>Server added</b>\n\nName: %s\nIP: <code>%s</code>\nID: <code>%s</code>",
	KeyServerDuplicate: "This IP is already registered.",
	KeyServerAddError:  "Error adding server.",

	// ── Template ──────────────────────────────────────────
	KeyTemplatesTitle:   "<b>📦 Templates</b>",
	KeyTemplatesEmpty:   "No templates found.",
	KeyTemplateAskType:  "Select bot type:",
	KeyTemplateAskImage: "Enter Docker image name:\nExample: <code>registry.io/mybot</code>",
	KeyTemplateAskTag:   "Enter image tag:\nExample: <code>latest</code> or <code>v1.2.0</code>",
	KeyTemplateAskName:  "Enter a name for this template:\nExample: <code>uploader-v2</code>",
	KeyTemplateAdded:    "✅ <b>Template added</b>\n\nName: <b>%s</b>\nType: %s\nImage: <code>%s:%s</code>\nID: <code>%s</code>",
	KeyTemplateAddError: "Error adding template.",

	// ── Plan ──────────────────────────────────────────────
	KeyPlansTitle:        "<b>💰 Plans</b>",
	KeyPlansEmpty:        "No plans found.",
	KeyPlansNoTemplate:   "⚠️ You need to add a template first.",
	KeyPlanAskTemplate:   "Send the template ID to create a new plan:",
	KeyPlanTmplNotFound:  "Template not found. Please check the ID.",
	KeyPlanAskName:       "Enter plan name:\nExample: Monthly",
	KeyPlanAskDays:       "Enter plan duration in days:\nExample: <b>30</b>",
	KeyPlanAskPrice:      "Enter plan price:\nExample: <b>5</b>",
	KeyPlanInvalidNumber: "Please enter a valid number.",
	KeyPlanAdded:         "✅ <b>Plan added</b>\n\nName: <b>%s</b>\nTemplate: %s\nDuration: %d days\nPrice: <b>%.2f</b>\nID: <code>%s</code>",
	KeyPlanAddError:      "Error adding plan.",

	// ── Invite Link ───────────────────────────────────────
	KeyLinksTitle:      "<b>🔗 Invite Links</b>\n\nUsers can build bots using these links.",
	KeyLinksEmpty:      "No invite links yet.",
	KeyLinkAskType:     "Select bot type for the new link:",
	KeyLinkAskLimit:    "Select usage limit for this link:",
	KeyLinkAskLabel:    "Write a private note (e.g. «For John»)\nThis note is only visible to you.\n\nEnter <b>0</b> for no note.",
	KeyLinkCreated:     "✅ <b>Invite link created</b>\n\nType: %s %s%s\nLimit: %s\n\n🔗 Link:\n<code>%s</code>\n\nSend this link to the user.",
	KeyLinkCreateError: "Error creating invite link.",

	// ── Bots (Admin) ──────────────────────────────────────
	KeyBotsTitle:    "<b>🤖 All Bots (%d)</b>",
	KeyBotsEmpty:    "No bots found.\n\nCreate an invite link from «🔗 Invite Links» and share it with a user.",
	KeyBotStopped:   "⏹ Bot <code>%s</code> stopped.",
	KeyBotStarted:   "▶️ Start command sent for <code>%s</code>.",
	KeyBotDeleted:   "🗑 Bot <b>%s</b> deleted.",
	KeyBotNotFound:  "Bot not found.",

	// ── Users ─────────────────────────────────────────────
	KeyUsersTitle:    "<b>👥 Users (%d)</b>",
	KeyUsersEmpty:    "No users registered yet.",
	KeyUserBlocked:   "🚫 User blocked.",
	KeyUserUnblocked: "✅ User unblocked.",
	KeyUserMadeAdmin: "🛡 User promoted to Admin.",
	KeyUserMadeUser:  "👤 User role changed to User.",

	// ── Stats ─────────────────────────────────────────────
	KeyStatsTitle: "<b>📊 System Statistics</b>",

	// ── User Bots ─────────────────────────────────────────
	KeyMyBotsTitle: "<b>🤖 Your Bots (%d)</b>",
	KeyMyBotsEmpty: "<b>🤖 Your Bots</b>\n\nYou don't have any active bots yet.\n\nContact support to purchase or get an invite link.",
	KeySupportText: "<b>📞 Support</b>\n\nContact our support team:\n@support_username\n\nAvailable: 9am - 9pm",
	KeyHelpText:    "<b>❓ Help</b>\n\nWith CreatorBot you can build custom Telegram bots:\n\n📤 <b>Uploader</b> — Send files via code\n🔒 <b>VPN</b> — Sell VPN subscriptions\n📂 <b>Archive</b> — Archive and search files\n👥 <b>Member</b> — Channel membership lock\n\nContact support to purchase.",

	// ── Wizard ────────────────────────────────────────────
	KeyWizardInvalidLink:   "❌ This link is not valid.",
	KeyWizardExpiredLink:   "❌ This link has expired.",
	KeyWizardUsedLink:      "❌ This link has already been used.",
	KeyWizardConfirm:       "<b>🔗 Valid Invite Link</b>\n\n%s <b>%s Bot</b>\n\n%s\n\nDo you want to continue?",
	KeyWizardAskToken:      "Get a bot token from @BotFather and send it here.\n\n⚠️ Never share your token with anyone.",
	KeyWizardInvalidToken:  "❌ Invalid token format.\n\nThe token must be from @BotFather and look like:\n<code>123456789:AABB...</code>",
	KeyWizardAlreadyExists: "⚠️ This bot is already registered.\n\nID: <code>%s</code>\nStatus: %s",
	KeyWizardNoServer:      "⚠️ No servers are available at this time.\nPlease try again later or contact support.",
	KeyWizardNoTemplate:    "⚠️ No template configured for this bot type.\nPlease contact support.",
	KeyWizardDeployError:   "⚠️ <b>Bot registered but deploy failed</b>\n\nAdmin will investigate shortly.\nID: <code>%s</code>",
	KeyWizardSuccess:       "🎉 <b>Your bot is ready!</b>\n\n%s <b>%s Bot</b>\nServer: %s\nStatus: 🟡 Starting up\n\nUsually active within 1-2 minutes.\n\nCheck status: <b>🤖 My Bots</b>",

	// ── Bot Types ─────────────────────────────────────────
	KeyBotTypeUploader: "📤 Uploader",
	KeyBotTypeVPN:      "🔒 VPN",
	KeyBotTypeArchive:  "📂 Archive",
	KeyBotTypeMember:   "👥 Member",

	KeyBotDescUploader: "File uploader bot — send files via code",
	KeyBotDescVPN:      "VPN sales bot — sell subscriptions",
	KeyBotDescArchive:  "Archive bot — search and categorize files",
	KeyBotDescMember:   "Member lock bot — check channel membership",


	// ── Admin Stats ──────────────────────────────────────────
	KeyStatsBotsLine:    "🤖 Bots (%d total)\n🟢 Running: %d  🔴 Stopped: %d  🟡 Pending: %d  ⚠️ Error: %d",
	KeyStatsServersLine: "🖥 Servers (%d total)\n🟢 Online: %d  🔴 Offline: %d",
	KeyStatsUsersLine:   "👥 Users (%d total)\n🛡 Admin: %d  🚫 Blocked: %d",

	// ── Plans ─────────────────────────────────────────────────
	KeyPlansAvailable:    "<b>💎 Available Plans</b>",
	KeyPlansFree:         "🆓 Free",
	KeyPlansDays:         "%d days",
	KeyPlansEternal:      "Lifetime",
	KeyPlansSelectPrompt: "Send the plan ID you want:",

	// ── Wallet ────────────────────────────────────────────────
	KeyBalanceLine:  "💳 Balance: <b>%.4f TON</b>",
	KeyCreditLine:   " (🎁 %.4f credit)",
	KeyPlanLine:     "📋 Plan: <b>%s</b>\n🤖 %d/%d bots\n%s",
	KeyExpiredSub:   "❌ Expired",
	KeyEternalSub:   "♾ Lifetime",
	KeyDaysLeft:     "⏰ %d days left",

	// ── Purchase ──────────────────────────────────────────────
	KeyNoPlans:         "No plans available.",
	KeyBuyConfirm:      "<b>Confirm Purchase</b>\n\n📋 Plan: <b>%s</b>\n💰 Price: <b>%.2f TON</b>\n💳 Your balance: %.4f TON\n\nConfirm?",
	KeyBuySuccess:      "✅ <b>Plan %s activated!</b>\n\n🤖 %d bots\nYou can now build your bot.",
	KeyInsufficientBal: "❌ Insufficient balance.",
	KeyNeedDeposit:     "<b>💎 Buy Plan %s</b>\n\n💰 Price: %.2f TON\n💳 Your balance: %.4f TON\n📥 Need to deposit: <b>%.4f TON</b>\n\nCode: <code>%s</code>\n\n1. Click Deposit\n2. Pay the amount\n3. Click «Done»",
	KeyDepositDone:     "✅ <b>Payment confirmed!</b>\n\n📋 Plan: %s\n🤖 %d bots\nYou can now build your bot.",
	KeyDepositPending:  "⏳ Balance still insufficient.\n\n💳 Balance: %.4f TON\n💰 Need: %.2f TON\n\nWait a few minutes and try again.",
	KeySubExists:       "You already have an active subscription.",
	KeyFreePlanActive:  "🎉 <b>Free plan activated!</b>\n\n📋 %s\n🤖 %d bots\n⏳ %s\n\nYou can now build your bot.",
	KeyCapacityFull:    "Bot limit reached (%d/%d).\n\n💎 Upgrade your plan for more bots.",
	KeyNoPlan:          "You need to purchase a plan to build bots.",

	// ── Block ─────────────────────────────────────────────────
	KeyBlocked: "⛔️ Your access has been restricted.",


	// ── Admin — Plan ──────────────────────────────────────────
	KeyAdminPlanLine:   "• <b>%s</b>%s — %d days — <b>%.2f TON</b> — %d bots\n  ID: <code>%s</code>",
	KeyAdminPlanFree:   " 🆓",
	KeyAdminPlanAdded:  "✅ <b>Plan added</b>\n\nName: <b>%s</b>%s\nTemplate: %s\nDuration: %d days\nPrice: <b>%.2f TON</b>\nMax bots: %d\nID: <code>%s</code>",
	KeyAdminTemplates:  "<b>Templates:</b>",

	// ── Admin — Bots ──────────────────────────────────────────
	KeyAdminBotSummary:  "🟢 %d  🔴 %d  🟡 %d  ⚠️ %d",
	KeyAdminLinkStats:   "✅ Active: %d  |  ❌ Expired: %d",
	KeyAdminLinkLimitX:  "%d×",

	// ── Admin — Users ─────────────────────────────────────────
	KeyAdminUserSummary: "👑 %d  🛡 %d  👤 %d  🚫 %d",
	KeyAdminUserDetail:  "<b>👤 %s</b>%s\nTID: <code>%d</code>\nRole: %s\nBlocked: %s\nBots: %d",
	KeyAdminUserBlocked: "🚫",

	// ── How to build ──────────────────────────────────────────
	KeyHowToBuild: "<b>🔗 How to build a bot?</b>\n\n" +
		"1. Message the admin and say you want to build a bot\n" +
		"2. Admin will send you an invite link\n" +
		"3. Open the link\n" +
		"4. Create a bot on @BotFather and send the token here\n" +
		"5. Done! Your bot will be ready in 2 minutes.",
	KeyHowToBuildDone: "✅ Got it",
	KeyNoFreePlan:     "No free plan available.",

	// ── Free template ─────────────────────────────────────────
	KeyTmplFreeAdded:  "✅ <b>Free template added</b>\n\nName: <b>%s</b>\nType: %s\nImage: <code>%s:%s</code>\nID: <code>%s</code>\n\nNow you can create a free plan.",
	KeyTmplFreeExists: "⚠️ A free template of type <b>%s</b> already exists.\nID: <code>%s</code>",


	// ── Subscription active, no bot ─────────────────────────
	KeySubActiveNoBot:   "🎉 <b>Plan %s is active</b>\n\nYou haven't built a bot yet.\n\nTo build a bot, get an <b>invite link</b> from the admin.\nOpen the link and follow the steps.",
	KeyBuildWithLink:    "🔗 Build bot with invite link",

	// ── Buttons ───────────────────────────────────────────
	KeyBtnYesBuild:  "✅ Yes, build it",
	KeyBtnCancel:    "❌ Cancel",
	KeyBtnBack:      "🔙 Back",
	KeyBtnLimit1:    "1️⃣  Once",
	KeyBtnLimit3:    "3️⃣  3 times",
	KeyBtnLimit5:    "5️⃣  5 times",
	KeyBtnLimit10:   "🔟 10 times",
	KeyBtnLimitNo:   "♾  Unlimited",
	KeyBtnBlock:     "🚫 Block",
	KeyBtnUnblock:   "✅ Unblock",
	KeyBtnMakeAdmin: "🛡 Make Admin",
	KeyBtnMakeUser:  "👤 Make User",
}
