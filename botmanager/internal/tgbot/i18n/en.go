package i18n

var en = map[Key]string{
	KeyError:                 "❌ An error occurred. Please try again.",
	KeyCancelled:             "✅ Operation cancelled.",
	KeyDone:                  "✅ Done.",
	KeyBack:                  "🔙 Back",
	KeyCancel:                "❌ Cancel",
	KeyConfirm:               "✅ Confirm",
	KeyLoading:               "⏳ Loading...",
	KeyNotFound:              "❌ Not found.",
	KeyComingSoon:            "🚧 This section will be available soon.",
	KeyAccountTitle:          "👤 <b>Your User Profile</b>\n🆔 ID: <code>%d</code>\n💰 Balance: <b>%.2f TON</b>\n🎁 Credit: <b>%.2f TON</b>\n💵 Total: <b>%.2f TON</b>\n🌟 Status: %s\n\nTop up your account to create more bots and access premium features.",
	KeyAccountStatusStandard: "🙂 Standard",
	KeyAccountStatusVIP:      "🌟 VIP",
	KeyLanguageSelect:        "🌐 <b>Language Selection</b>\n🇬🇧 Please select your preferred language:\n🇮🇷 لطفاً زبان مورد نظر خود را انتخاب کنید:",
	KeyBroadcastMenu:         "📢 <b>Broadcast Module</b>\nYour message will be queued and sent to all target users rapidly:",
	KeySystemMenu:            "⚙️ <b>Global System Settings</b>\nConfigure main bot preferences and admin center settings here:",
	KeyNoAccess:              "⛔ Access denied.",

	KeyMenuWallet:        "💰 Wallet",
	KeyMenuServices:      "🤖 My Services",
	KeyMenuCommunities:   "🏘 Communities",
	KeyMenuAds:           "📢 Advertisements",
	KeyMenuEarnings:      "📊 Earnings",
	KeyMenuPlans:         "💎 Plans",
	KeyMenuNotifications: "🔔 Notifications",
	KeyMenuSettings:      "⚙️ Settings",
	KeyMenuHelp:          "❓ Help",
	KeyMenuSupport:       "💬 Support",
	KeyMenuMyBots:        "🤖 My Bots",
	KeyMenuCreateBot:     "🚀 Create New Bot",
	KeyMenuAccount:       "💳 Account & Balance",
	KeyMenuLanguage:      "🌐 Language | تغییر زبان",
	KeyMenuTutorials:     "📚 Help & Tutorials",

	KeyMenuUsers:     "👥 Users",
	KeyMenuCampaigns: "📢 Campaigns",
	KeyMenuFinance:   "💰 Finance",
	KeyMenuFraud:     "🚨 Fraud",
	KeyMenuStats:     "📈 Statistics",
	KeyMenuSystem:    "⚙️ System Settings",
	KeyMenuBroadcast: "📢 Mass Broadcast",
	KeyMenuExitAdmin: "🚪 Exit Admin Panel",
	KeyMenuBots:      "🤖 Bots",
	KeyMenuLinks:     "🔗 Links",
	KeyMenuServers:   "🖥 Servers",
	KeyMenuTemplates: "📦 Templates",

	KeyWelcomeUser: `👋 Hello <b>%s</b>!

Welcome to CreatorBot.
With this platform you can:
• Create and manage Telegram bots
• Earn revenue
• Manage communities
• Run targeted advertising campaigns

Use the menu below to get started 👇`,

	KeyWelcomeAdmin: `👑 Hello <b>%s</b>!

Welcome to the CreatorBot Admin Panel.
Use the menu below to access all sections 👇`,

	KeyHelpText: `📚 <b>CreatorBot Help</b>

💰 <b>Wallet</b> — Balance, deposit, withdraw
🤖 <b>My Services</b> — Create and manage bots
🏘 <b>Communities</b> — Register groups and channels
📢 <b>Advertisements</b> — Create ad campaigns
📊 <b>Earnings</b> — Revenue reports
💎 <b>Plans</b> — Upgrade subscription
🔔 <b>Notifications</b> — Configure alerts
⚙️ <b>Settings</b> — Language, security, support

❓ Need help? Contact our support team.`,

	KeyHelpAdmin: `👑 <b>CreatorBot Admin Panel</b>

👥 <b>Users</b> — Manage and search users
🤖 <b>Services</b> — Manage user bots
🏘 <b>Communities</b> — Monitor groups and channels
📢 <b>Campaigns</b> — Manage advertising
💰 <b>Finance</b> — Financial reports and withdrawals
🚨 <b>Fraud</b> — Monitor and combat fraud
📈 <b>Statistics</b> — Platform-wide stats
⚙️ <b>System</b> — Plans, servers, settings`,

	KeyWalletHome: `💰 <b>Wallet</b>

💎 TON Balance: <b>%.4f</b>
🎁 Credit: <b>%.4f</b>
💵 Total Available: <b>%.4f</b>`,

	KeyWalletDeposit:    "📥 <b>Deposit</b>\n\nSelect a deposit method:",
	KeyWalletWithdraw:   "📤 <b>Withdraw</b>\n\nSelect a withdrawal method:",
	KeyWalletTransfer:   "🔄 <b>Internal Transfer</b>\n\nEnter the recipient's Telegram ID:",
	KeyWalletHistory:    "📜 <b>Transaction History</b>",
	KeyWalletRewards:    "🎁 <b>Rewards</b>\n\nYour earned rewards:",
	KeyWalletLowBalance: "❌ Insufficient balance.\n\n💡 Use the Deposit button to top up your wallet.",

	KeyServicesHome:  "🤖 <b>My Services</b>\n\n%d active services",
	KeyServicesEmpty: "🤖 <b>My Services</b>\n\nYou don't have any services yet.\nClick «Create Service» to build your first bot! 🚀",

	KeyServiceCreate:     "🆕 <b>Create New Service</b>",
	KeyServiceSelectType: "🤖 <b>Select Service Type:</b>",
	KeyServiceSelectTag:  "🏷 <b>Select service version (tag):</b>\n\nService: <b>%s</b>",
	KeyServiceSelectPlan: "💎 <b>Select a Plan:</b>\n\nService: %s",
	KeyServiceEnterToken: `🔑 <b>Bot Token</b>

Plan: <b>%s</b> — <b>%.2f TON</b>

Get your bot token from @BotFather and send it here:

<code>1234567890:ABCDefghijklmnop...</code>

📌 How to get a token:
1. Open @BotFather in Telegram
2. Send /newbot
3. Enter your bot's name and username
4. Copy and send the token here`,

	KeyServiceConfirm: `✅ <b>Confirm Service Creation</b>

🤖 Type: <b>%s</b>
🏷 Tag: <b>%s</b>
💎 Plan: <b>%s</b>
💰 Price: <b>%.2f TON</b>

Are you sure?`,

	KeyServiceCreating: "⏳ <b>Setting up your service...</b>\n\nPlease wait.",
	KeyServiceCreated: `🎉 <b>Service successfully created!</b>

🤖 Type: <b>%s</b>
💎 Plan: <b>%s</b>
📦 Status: <b>Setting up</b>

⏱ Usually ready within 2-3 minutes.
Track the status in «My Services».`,

	KeyServiceFailed:       "❌ <b>Service setup failed.</b>\n\nYour payment has been refunded to your wallet.",
	KeyServiceNoCapacity:   "❌ <b>Capacity reached.</b>\n\nYour current plan doesn't allow more services.\n\n💡 Upgrade your plan to increase capacity.",
	KeyServiceInvalidToken: "❌ <b>Invalid token.</b>\n\nExample: <code>1234567890:ABCDefgh...</code>",
	KeyServiceDuplicate:    "❌ <b>This bot is already registered.</b>\n\nEach bot can only be used once.",

	KeyPlansHome:    "💎 <b>Plans</b>\n\nChoose a plan that suits you:",
	KeyPlanCurrent:  "💎 <b>Your Current Plan</b>\n\n📦 Plan: <b>%s</b>\n🤖 Bots: <b>%d / %d</b>\n📅 Expires: <b>%s</b>\n⏰ Days remaining: <b>%d</b>",
	KeyPlanNone:     "❌ <b>No active subscription.</b>\n\nSelect a plan to use the platform.",
	KeyPlanExpired:  "⚠️ <b>Your subscription has expired.</b>\n\nRenew your plan to continue.",
	KeyPlanUpgrade:  "🚀 <b>Upgrade Plan</b>\n\nSelect a target plan:",
	KeyPlanBuyTitle: "💎 <b>%s</b>\n\n⏱ Duration: <b>%d days</b>\n🤖 Max bots: <b>%d</b>\n💰 Price: <b>%.2f TON</b>",
	KeyPlanBought:   "🎉 <b>Plan %s activated successfully!</b>\n\nGo to «My Services» to create your bot.",
	KeyNoFreePlan:   "❌ No free plan available at this time.",
	KeyFreePlanDone: "✅ <b>Free plan activated!</b>\n\nYou can now create your first bot.",

	KeyCommHome:     "🏘 <b>Communities</b>\n\nManage your groups and channels and earn from advertising.",
	KeyCommEmpty:    "🏘 <b>Communities</b>\n\nYou haven't registered any communities yet.\n\nRegister a group or channel to start earning from ads! 💰",
	KeyCommRegister: "➕ <b>Register New Community</b>\n\nSelect the community type:",
	KeyCommVerify:   "🔍 <b>Verify Community</b>\n\nAdd the bot to your community as admin, then click Verify.",

	KeyAdsHome:   "📢 <b>Advertisements</b>\n\nManage your advertising campaigns.",
	KeyAdsEmpty:  "📢 <b>Advertisements</b>\n\nNo campaigns yet.\n\nCreate a campaign to show your ad in target channels! 📣",
	KeyAdsCreate: "➕ <b>New Campaign</b>\n\nEnter the campaign name:",

	KeyEarningsHome:  "📊 <b>Earnings</b>\n\n💰 Total: <b>%.4f TON</b>\n📅 Today: <b>%.4f TON</b>\n📆 This Month: <b>%.4f TON</b>",
	KeyEarningsEmpty: "📊 <b>Earnings</b>\n\nNo earnings recorded yet.\n\n💡 Activate communities and services to start earning.",

	KeySettingsHome: "⚙️ <b>Settings</b>\n\nManage your preferences below:",
	KeyLangChanged:  "✅ Language changed successfully.",
	KeyLangSelect:   "🌍 <b>Select Language</b>\n\nChoose your preferred language:",

	KeyNotificationsHome: "🔔 <b>Notifications</b>\n\nSelect which notifications you want to receive:",

	KeySupportText: "💬 <b>CreatorBot Support</b>\n\n📚 Docs: t.me/CreatorBotDocs\n💬 Support: @CreatorBotSupport\n🐞 Bug Report: @CreatorBotBug\n\n⏰ Response hours: 9 AM – 10 PM",

	KeyAdminUsersTitle:    "👥 <b>Users</b> (%d total)\n\n👑 %d owner | 🛡 %d admin | 👤 %d regular | 🚫 %d blocked",
	KeyAdminUserDetail:    "👤 <b>User Info</b>\n\n🏷 Name: <b>%s</b>\n🔗 Username: %s\n🆔 ID: <code>%d</code>\n👑 Role: <b>%s</b>\n🚫 Status: %s\n🤖 Active bots: %d",
	KeyAdminUserBlocked:   "🚫 User <b>%s</b> has been blocked.",
	KeyAdminUserUnblocked: "✅ User <b>%s</b> has been unblocked.",

	KeyBotsEmpty:     "🤖 <b>No active bots found.</b>",
	KeyServerAskName: "🖥 <b>New Server</b>\n\nEnter the server name:\n<i>Example: server-eu-1</i>",
	KeyServerAskIP:   "🌐 Enter the server IP address:\n<i>Example: 192.168.1.1</i>",
	KeyServerAdded:   "✅ Server <b>%s</b> added successfully.",

	KeyTemplateAskType:  "📦 <b>New Template</b>\n\nSelect the service type:",
	KeyTemplateAskImage: "🐳 Enter the Docker image name:\n<i>Example: creatorbot/vpn-bot</i>",
	KeyTemplateAskTag:   "🏷 Enter the image tag:\n<i>Example: latest or v1.2.3</i>",
	KeyTmplFreeAdded:    "✅ Free template defined successfully.",
	KeyTmplFreeExists:   "⚠️ A free template for this type already exists.",

	KeyPlanAskName:  "💎 <b>New Plan</b>\n\nEnter the plan name:",
	KeyPlanAskPrice: "💰 Enter the plan price in TON:\n<i>Example: 5.0</i>",
	KeyPlanAskDays:  "📅 Enter the plan duration in days:\n<i>Example: 30</i>",
	KeyPlanAskBots:  "🤖 Enter the maximum number of bots:\n<i>Example: 3</i>",
	KeyPlanAdded:    "✅ Plan <b>%s</b> added successfully.",

	KeyLinkAskType:  "🔗 <b>New Invite Link</b>\n\nSelect the service type:",
	KeyLinkAskLimit: "🔢 Select the number of uses:",
	KeyLinkCreated:  "✅ Invite link created:\n\n<code>%s</code>\n\nUsage limit: %d",

	KeyStatsTitle: "📈 <b>System Statistics</b>\n⏰ %s\n\n🤖 <b>Bots</b> (%d total)\n🟢 Running: %d | 🔴 Stopped: %d | 🟡 Pending: %d | ⚠️ Error: %d\n\n🖥 <b>Servers</b> (%d total)\n🟢 Online: %d | 🔴 Offline: %d\n\n👥 <b>Users</b> (%d total)\n🛡 Admin: %d | 🚫 Blocked: %d\n\n📦 %d plans | 💰 %d wallets",

	KeySubActiveNoBot: "✅ <b>Plan %s is active!</b>\n\nYou have no active bots yet.\nClick below to create one:",
	KeyBuildWithLink:  "🔗 <b>Valid Invite Link</b>\n\nService: <b>%s</b>\nConfirm to continue:",

	KeyWizardInvalidLink: "❌ <b>Invalid invite link.</b>\n\nPlease get a valid link from the admin.",
	KeyWizardExpiredLink: "⏰ <b>This invite link has expired.</b>\n\nPlease request a new link from the admin.",
	KeyWizardUsedLink:    "❌ <b>This invite link has already been used.</b>",
	KeyWizardAskToken:    "🔑 <b>Bot Token</b>\n\nGet your bot token from @BotFather and send it here:\n\n<code>1234567890:ABCDefgh...</code>",

	KeyHowToBuild: `📘 <b>How to Create a Bot</b>

1️⃣ Open @BotFather in Telegram
2️⃣ Send /newbot
3️⃣ Enter your bot's name
4️⃣ Choose a username (must end with "bot")
5️⃣ Copy the token you receive
6️⃣ Send the token here ✅`,

	KeyHowToBuildDone: "✅ Got it, let's continue",

	KeyBtnYesBuild:       "✅ Yes, I have a bot",
	KeyBtnCancel:         "❌ Cancel",
	KeyBtnBack:           "🔙 Back",
	KeyBtnLimit1:         "1️⃣ Once",
	KeyBtnLimit3:         "3️⃣ Three times",
	KeyBtnLimit5:         "5️⃣ Five times",
	KeyBtnLimit10:        "🔟 Ten times",
	KeyBtnLimitNo:        "♾️ Unlimited",
	KeyBtnBlock:          "🚫 Block",
	KeyBtnUnblock:        "✅ Unblock",
	KeyBtnMakeAdmin:      "🛡 Make Admin",
	KeyBtnMakeUser:       "👤 Make Regular User",
	KeyBotsTitle:         "🤖 <b>Bots</b> (%d total)",
	KeyAdminBotSummary:   "%s <b>%s</b> — %s",
	KeyBotNotFound:       "❌ Bot not found.",
	KeyBotStopped:        "⏹ Bot <b>%s</b> stopped.",
	KeyBotStarted:        "▶️ Bot <b>%s</b> started.",
	KeyBotDeleted:        "🗑 Bot <b>%s</b> deleted.",
	KeyBotActionFailed:   "⚠️ That action failed — the server might be offline. Try again shortly or check the server logs.",
	KeyLinksTitle:        "🔗 <b>Invite Links</b> (%d links)",
	KeyLinksEmpty:        "🔗 <b>Invite Links</b>\n\nNo links found.",
	KeyLinkAskLabel:      "🏷 Enter a label for this link (or send 'skip'):",
	KeyLinkCreateError:   "❌ Failed to create link.",
	KeyAdminLinkStats:    "🔗 <code>%s</code>\n📦 %s | 🔢 %d/%s | ⏰ %s",
	KeyAdminLinkLimitX:   "%d times",
	KeyPlansTitle:        "💎 <b>Plans</b> (%d plans)",
	KeyPlansEmpty:        "💎 <b>Plans</b>\n\nNo plans defined.",
	KeyPlansNoTemplate:   "❌ Define a template first.",
	KeyPlanAskTemplate:   "📦 Select a service template:",
	KeyPlanInvalidNumber: "❌ Enter a valid number.",
	KeyPlanTmplNotFound:  "❌ Template not found.",
	KeyPlanAddError:      "❌ Failed to create plan.",
	KeyAdminPlanLine:     "💎 <b>%s</b> — %.1f TON | %d days | %d bots",
	KeyAdminPlanFree:     "🆓 Free",
	KeyTemplatesTitle:    "📦 <b>Templates</b> (%d templates)",
	KeyTemplatesEmpty:    "📦 <b>Templates</b>\n\nNo templates defined.",
	KeyTemplateAskName:   "📦 <b>New Template</b>\n\nEnter template name:",
	KeyTemplateAdded:     "✅ Template <b>%s</b> added.",
	KeyTemplateAddError:  "❌ Failed to create template.",

	// ── buttons & service test ──
	KeyBtnTest:             "🧪 Test",
	KeyBtnNewTemplate:      "➕ New Template",
	KeyBtnAddCredit:        "💰 Add Credit",
	KeyBtnBackToList:       "🔙 Back to List",
	KeyBtnAddServer:        "➕ Add Server",
	KeyBtnViewWallet:       "💰 View Wallet",
	KeyBtnConfirmDelete:    "🗑 Yes, delete",
	KeyBtnGotIt:            "✅ Got it",
	KeyBalanceUpdated:      "✅ <b>Balance updated</b>\n\n💎 Total balance: <b>%.4f TON</b>",
	KeyBalanceAlert:        "💎 Total balance: %.4f TON",
	KeyPaymentPendingAlert: "⏳ Payment not confirmed yet.\n💳 Balance: %.4f TON | Required: %.2f TON\n⚠️ Short: %.4f TON\nTry again in a moment.",
	KeyTxPending:           "⏳ Transaction not received yet. Check again shortly.",
	KeyTxPaid:              "✅ Transaction received!",
	KeyTxPartial:           "🔸 Partially received (%.4f of %.4f TON). Awaiting the rest.",
	KeyTxExpired:           "❌ This invoice has expired. Please create a new one.",
	KeyTxNotFound:          "❓ No transaction found for this code.",
	KeyTxCheckFailed:       "⚠️ Couldn't check status. Try again shortly.",
	KeyBtnDepositTON:       "📥 Deposit TON",
	KeyBtnHistory:          "📜 History",
	KeyBtnRedeemPromo:      "🎁 Redeem promo code",
	KeyBtnNewDeposit:       "📥 New deposit",
	KeyBtnCheckPayment:     "🔄 Check payment",
	KeyBtnBcText:           "💬 Text broadcast",
	KeyBtnBcForward:        "🔄 Forward broadcast",
	KeyBtnBcFiltered:       "🎯 Filtered broadcast",
	KeyBtnCreateFree:       "✅ Create free",
	KeyBtnPayCreate:        "✅ Pay & create",

	// ── general / status ──
	KeyFree:           "Free",
	KeyDaysCount:      "%d days",
	KeyForever:        "∞ Forever",
	KeyStatusActive:   "✅ Active",
	KeyStatusInactive: "⛔ Inactive",
	KeyErrShort:       "❌ Error",
	KeyErrSave:        "❌ Save error",

	// ── admin — plan (editor) ──
	KeyPlanNotFound:       "❌ Plan not found.",
	KeyBtnNewPlan:         "➕ New Plan",
	KeyBtnBackToPlans:     "🔙 Back to Plans",
	KeyBtnEditPlan:        "⚙️ Edit: %s",
	KeyBtnTotalCap:        "🤖 Total cap:  %d",
	KeyAdminPlanRow:       "%s 💎 <b>%s</b> — %s TON | %d days | cap %d bots",
	KeyPlanEditTitle:      "⚙️ <b>Edit Plan: %s</b>\n💰 %s  |  ⏳ %s  |  %s\n\n📊 <b>Bot limits:</b>\n<i>(each change is saved instantly)</i>",
	KeyAvailableTemplates: "📦 <b>Available templates:</b>",
	KeyPlanTmplChosen:     "Template: <b>%s</b>\n\n%s",
	KeyPlanLimitsPrompt:   "Now enter the limit per bot type.\n\nFormat: <code>type=count</code> comma-separated\nExample: <code>uploader=2,vpn=1</code>\n\nTypes: %s\nOr send a single number to apply to all types.",
	KeyPlanLimitsInvalid:  "Invalid format. Example: <code>uploader=2,vpn=1</code>",
	KeyPlanLimitsSaved:    "✅ <b>Limits saved</b>\n\n%s\n\nTotal: %d bots",

	// ── user — plans (UI) ──
	KeyDurationForever:          "Forever",
	KeyBtnMyBots:                "🤖 My Bots",
	KeyBtnTopupWallet:           "💎 Top up wallet",
	KeyBtnRecheck:               "🔄 Re-check",
	KeyBtnClose:                 "❌ Close",
	KeyBtnBuyWith:               "✅ Buy for %.2f TON",
	KeyPlansUnavailable:         "No plans available right now. Please check back later.",
	KeyPlansAvailableTitle:      "<b>💎 Available Plans</b>\n\n",
	KeyPlanRemDays:              " — %d days left",
	KeyPlanExpiredShort:         " — expired",
	KeyPlanActiveYours:          "✅ <b>Your active plan:</b> %s%s\n\n",
	KeyPlanRow:                  "<b>%s</b>\n💰 %s  |  🤖 %d bots  |  ⏳ %s\n\n",
	KeyPlansClickToBuy:          "Tap the plan you want to purchase:",
	KeyPlanLabelFree:            "🆓 %s — Free",
	KeyPlanLabelPaid:            "💎 %s — %.2f TON",
	KeyPlanAlreadyActive:        "✅ This plan is already active for you.",
	KeyPlanDetail:               "<b>💎 %s</b>\n\n🤖 Bots: %d\n⏳ Duration: %s\n💰 Price: <b>%.2f TON</b>\n\n",
	KeyWalletBalanceLine:        "💳 Your wallet balance: <b>%.4f TON</b>\n",
	KeyBalanceEnough:            "\n✅ Your balance is sufficient!",
	KeyBalanceShortfall:         "\n⚠️ Shortfall: <b>%.4f TON</b>",
	KeyDepositAddrCode:          "\n\n💎 Address: <code>%s</code>\n🏷 Code (be sure to include in comment): <code>%s</code>",
	KeyPayServiceUnavailable:    "\n⚠️ Payment service temporarily unavailable.",
	KeyFreePlanActivated:        "🎉 <b>Free plan activated!</b>\n\n✅ %d bots — %s\n\nYou can now create your service.",
	KeyPlanPurchaseDesc:         "Plan purchase %s",
	KeyPurchaseSuccess:          "🎉 <b>Purchase successful!</b>\n\n✅ Plan <b>%s</b> activated\n🤖 %d bots available\n\nYou can now create your service.",
	KeyPurchaseActivationFailed: "😥 The amount was deducted from your wallet, but activating the plan hit a technical error.\n\n💰 We've refunded you — nothing was lost.\nPlease try again; if it keeps happening, reach out to support.",
	KeyPaymentNotConfirmed:      "⏳ Payment not confirmed yet.\n\n💳 Current balance: <b>%.4f TON</b>\n💰 Required: <b>%.2f TON</b>\n⚠️ Shortfall: %.4f TON\n\nWait a few minutes and check again.",

	// ── UX — wizard & badges ──
	KeyWizardStep:      "🧭 Step %d of %d",
	KeyBadgePopular:    "🔥 Popular",
	KeyBadgeNewest:     "🆕 New",
	KeyBtnCustomAmount: "✏️ Custom amount",
	KeyBtnRenew:        "🔄 Renew / Upgrade",

	// ── service renewal & expiry reminder ──
	KeyRenewConfirm:    "🔄 <b>Renew Service</b>\n\n📛 <code>%s</code>\n💎 Plan: <b>%s</b>\n💰 Renewal cost: <b>%.2f TON</b>\n\nIt will be charged from your wallet and the service renewed. Confirm?",
	KeyBtnConfirmRenew: "✅ Confirm & pay",
	KeyRenewDone:       "🎉 <b>Service renewed!</b>\n\n📛 <code>%s</code>\n⏰ %s",
	KeyRenewNoPlan:     "❌ This service has no plan to renew. Please use «Plans».",
	KeyExpiryReminder:  "⏰ <b>Expiry reminder</b>\n\n📛 Service <code>%s</code> expires in <b>%d days</b>.\nRenew now to avoid interruption. 🔄",

	// ── user — my services (UI) ──
	KeyBtnStats:           "📊 Stats",
	KeyBtnSettings:        "⚙️ Settings",
	KeyBtnRestart:         "🔄 Restart",
	KeyBtnStop:            "⏸ Stop",
	KeyBtnStart:           "▶️ Start",
	KeyBtnDeleteSvc:       "🗑 Delete service",
	KeyBtnDelete:          "🗑 Delete",
	KeyBtnCheckStatus:     "🔄 Check status",
	KeyBtnRetry:           "🔄 Retry",
	KeyBtnCreateSvc:       "➕ Create service",
	KeyBtnCreateNewSvc:    "➕ Create new service",
	KeyBtnStartFree:       "🆓 Start free",
	KeyBtnViewPlans:       "💎 View plans",
	KeyBtnUpgradePlan:     "💎 Upgrade plan",
	KeyMyServicesHeader:   "<b>🤖 My Services</b> (%d services)\n",
	KeySvcNameLine:        "📛 Name: <code>%s</code>\n",
	KeySvcStatusLine:      "%s Status: <b>%s</b>\n",
	KeySvcExpiredNL:       "⏰ <b>Expired</b>\n",
	KeySvcHoursLeft:       "⚠️ %d hours until expiry\n",
	KeySvcDaysLeft:        "⏰ %d days left\n",
	KeyWelcomeNoService:   "👋 Hello <b>%s</b>!\n\nWith CreatorBot you can build your own Telegram bot.\n\n🆓 <b>Free plan:</b>\nOne free bot, forever\n\n💎 <b>Paid plans:</b>\nMultiple bots — with more features\n\nTap «Start free» to begin:",
	KeyNeedPlanFirst:      "To create a bot you first need a plan.\n\nYou can have one free bot:",
	KeyMaxBotsReached:     "❌ You've reached the bot limit.\n\n🤖 %d of %d bots used\n\nUpgrade your plan to create more bots.",
	KeyStatusRunning:      "Running",
	KeyStatusStopped:      "Stopped",
	KeyStatusStarting:     "Starting...",
	KeyStatusErrContact:   "Error — contact support",
	KeyTypeNotAllowed:     "❌ Your current plan doesn't allow creating <b>%s</b> bots.\n\nUpgrade your plan for access.",
	KeyMaxBotsReachedType: "❌ You've reached the <b>%s</b> bot limit (%d of %d).\n\nUpgrade your plan to create more.",
	KeyActionStopSent:     "⏹ Got it, stopping your service…",
	KeyActionStartSent:    "▶️ On it, starting your service…",
	KeyActionRestartSent:  "🔄 Restarting your service…",
	KeyActionDeleteSent:   "🗑 Delete request received — your service will be gone in a moment.",
	KeySvcStatusShort:     "%s Service status: <b>%s</b>",
	KeySvcStatsDetail:     "📊 <b>Service Stats</b>\n\n📛 Name: <code>%s</code>\n%s Status: <b>%s</b>",
	KeyServiceGeneric:     "Service",
	KeyUnknown:            "Unknown",
	KeyExpiredLabel:       "⏰ <b>Expired</b>",
	KeyDaysUntilExpiry:    "⏰ %d days until expiry",
	KeyPlanLine:           "💎 Plan: <b>%s</b>",
	KeyAdminTestAskToken: "🧪 <b>Test Deploy</b>\n\nService: <b>%s</b> | Tag: <b>%s</b>\n\n" +
		"Send the test bot token from @BotFather (deploys with no plan/payment):",
	KeyAdminTestDeployed: "🧪 <b>Test service is starting.</b>\n\n" +
		"📛 Container: <code>%s</code>\nReady in a few minutes.",
	KeyAdminTemplates:      "📦 <b>%s</b> — %s:%s",
	KeyServersTitle:        "🖥 <b>Servers</b> (%d servers)",
	KeyServersEmpty:        "🖥 <b>Servers</b>\n\nNo servers registered.",
	KeyServerAddError:      "❌ Failed to add server.",
	KeyServerDuplicate:     "❌ A server with this IP already exists.",
	KeyServerDeleteConfirm: "🗑 This server will be removed from the deploy pool (bots already running on it are untouched). Sure?",
	KeyServerDeletedMsg:    "🗑 Server deleted.",
	KeyUsersTitle:          "👥 <b>Users</b> (%d total)",
	KeyUsersEmpty:          "👥 <b>Users</b>\n\nNo users found.",
	KeyUsersSearchPrompt:   "🔍 To see a user's details, send their numeric Telegram ID here:",
	KeyUsersSearchInvalid:  "❌ That's not a valid numeric ID. Try again (digits only, no @):",
	KeyAdminUserSummary:    "👑 %d owners | 🛡 %d admins | 👤 %d regular | 🚫 %d blocked",
	KeyUserBlocked:         "🚫 User <b>%s</b> blocked.",
	KeyUserUnblocked:       "✅ User <b>%s</b> unblocked.",
	KeyUserMadeAdmin:       "🛡 User <b>%s</b> made admin.",
	KeyUserMadeUser:        "👤 User <b>%s</b> made regular user.",
	KeyBlocked:             "🚫 Blocked",
	KeyStatsBotsLine:       "🟢 %d | 🔴 %d | 🟡 %d | ⚠️ %d",
	KeyStatsServersLine:    "🟢 %d | 🔴 %d",
	KeyStatsUsersLine:      "🛡 %d | 🚫 %d",
	KeyBotTypeVPN:          "🌐 VPN",
	KeyBotTypeUploader:     "📤 Uploader",
	KeyBotTypeMember:       "🔒 Membership Lock",
	KeyBotTypeArchive:      "📦 Archive",
	KeyNoPlan:              "❌ No active subscription.",
	KeySelectLang:          "🌍 Select your preferred language:",
	KeyBtnLimit:            "🔢 %d times",

	// ── Wizard — Errors ─────────────────────────────────
	KeyWizardNoPlan:      "❌ No plan found for this service type.",
	KeyWizardRestart:     "❌ Please start over.",
	KeyWizardNoServer:    "❌ No servers available. Contact admin.",
	KeyWizardNoTemplate:  "❌ Service template not found. Contact admin.",
	KeyWizardCreateError: "❌ Failed to create service. Please try again.",
	KeyWizardDeployError: "❌ Deploy command failed. Your payment has been refunded.",
	KeyWizardIncomplete:  "❌ Incomplete information. Please start over.",
	KeyWizardLowBalance:  "❌ <b>Insufficient balance.</b>\n\n💡 Top up your wallet from the Wallet menu.",
	KeyInstanceNotFound:  "❌ Service not found.",
	KeyInstanceNoAccess:  "⛔ You don't have access to this service.",

	// ── Admin — Add Credit ───────────────────────────────
	KeyAdminCreditAsk:     "💰 <b>Add Credit</b>\n\nUser: <code>%d</code>\n\nEnter amount (in TON):\n<i>e.g. 1.5 or 10</i>",
	KeyAdminCreditDone:    "✅ <b>%.4f TON</b> added to wallet of user <code>%d</code>.",
	KeyAdminCreditError:   "❌ Failed to add credit. Please try again.",
	KeyAdminCreditInvalid: "❌ Invalid amount. Enter a positive number (e.g. 2.5)",

	// ── Wallet ──────────────────────────────────────────
	KeyWalletTitle: "💰 <b>Wallet</b>\n\n💎 TON Balance: <b>%.4f</b>\n🎁 Gift Credit: <b>%.4f</b>\n💵 Total: <b>%.4f</b>",

	// ── Settings ────────────────────────────────────────
	KeySettingsLanguage: "🌐 Change Language",
	KeySettingsSupport:  "💬 Support",
	KeySettingsAbout:    "ℹ️ About",

	// ── Service Settings ────────────────────────────────
	KeySvcSettingsDetail: "⚙️ <b>Service Settings</b>\n\n📛 Name: <code>%s</code>\n🤖 Type: <b>%s</b>\n%s Status: <b>%s</b>\n🖥 Server: <code>%s</code>\n%s",

	// ── Delete Confirm ───────────────────────────────────
	KeyDeleteConfirm: "⚠️ <b>Confirm Delete</b>\n\nService <code>%s</code> will be permanently deleted.\n\nThis action cannot be undone. Are you sure?",
	KeyDeleteDone:    "🗑 Service <code>%s</code> deleted.",

	// ── Wallet Top-up ────────────────────────────────────
	KeyWalletTopupAsk:     "💰 <b>Top Up Wallet</b>\n\nEnter the TON amount you want to deposit:\n<i>e.g. 5.5</i>",
	KeyWalletTopupInvoice: "📥 <b>TON Deposit Details</b>\n\n💰 Amount: <b>%.4f TON</b>\n\n📬 Deposit Address:\n<code>%s</code>\n\n🏷 Reference Code (must include in transaction comment):\n<code>%s</code>\n\n⏰ This invoice is valid for 24 hours.",
	KeyWalletTopupInvalid: "❌ Invalid amount. Enter a positive number in TON (e.g. 5.5)",

	// ── Wallet History ───────────────────────────────────
	KeyWalletHistoryNote: "📜 <b>Transaction History</b>\n\n💎 TON Balance: <b>%.4f</b>\n🎁 Gift Credit: <b>%.4f</b>\n💵 Total: <b>%.4f</b>\n\n📊 Full transaction history coming soon.",

	// ── Support & About (inline) ─────────────────────────
	KeySupportInline: "💬 <b>CreatorBot Support</b>\n\n📚 Docs: @CreatorBotDocs\n💬 Support: @CreatorBotSupport\n🐞 Bug reports: @CreatorBotBug\n\n⏰ Response hours: 9 AM – 10 PM",
	KeyAboutPlatform: "ℹ️ <b>About CreatorBot</b>\n\n🤖 No-code Telegram bot platform\n\n✨ Features: Uploader, VPN, Member lock, Archive\n💰 Payment: TON Blockchain\n📌 Version: 3.0",

	// ── Admin — Broadcast ────────────────────────────────
	KeyBroadcastAskText:        "📢 <b>Broadcast</b>\n\nType the message to send to all users:",
	KeyBroadcastPreview:        "👁 <b>Message Preview</b>\n\n─────────────\n%s\n─────────────\n\n📤 Will be sent to <b>%d</b> users.\n\nConfirm?",
	KeyBroadcastDone:           "✅ <b>Broadcast complete.</b>\n\n📤 Sent: %d\n❌ Failed: %d",
	KeyBroadcastStarted:        "🚀 <b>Broadcast started.</b>\n\nYou'll get a summary once it finishes.",
	KeyBroadcastConfirm:        "✅ Confirm & Send",
	KeyBroadcastForwardAsk:     "🔄 Send or forward the message you want broadcast to everyone (text, photo, video, file — anything):",
	KeyBroadcastForwardPreview: "🔄 This message will be forwarded to <b>%d</b> users.\n\nConfirm?",
	KeyBroadcastEmptyAudience:  "🤷 No users match that filter — nothing was sent.",
	KeyBcFilterTitle:           "🎯 <b>Filtered Broadcast</b>\n\nWho should get this message?",
	KeyBcFilterAll:             "👥 All users",
	KeyBcFilterNoPlan:          "🆓 No active plan",
	KeyBcFilterPlan:            "💎 Users on «%s»",

	// ── Admin — System ───────────────────────────────────
	KeyAdminSysInfo: "⚙️ <b>System Status</b>\n\n🤖 Plans: %d\n🖥 Servers: %d (🟢 %d online)\n📦 Templates: %d\n👥 Users: %d",

	// ── Promo codes ───────────────────────────────────────
	KeyMenuPromoCodes:     "🎁 Promo Codes",
	KeyBtnAddPromo:        "➕ New Code",
	KeyPromoAsk:           "🎁 Send your promo code and we'll credit your wallet:",
	KeyPromoNotFound:      "❌ Couldn't find that code. Double-check the spelling.",
	KeyPromoAlreadyUsed:   "🤏 You've already used this code — each code works once per user.",
	KeyPromoExpiredOrFull: "⌛️ This code has either expired or reached its usage limit.",
	KeyPromoCreditFailed:  "😥 Code <code>%s</code> was accepted, but crediting your wallet failed. Contact support with this code and we'll sort it out.",
	KeyPromoRedeemed:      "🎉 Nice! <b>%.2f TON</b> credit added to your wallet.",
	KeyPromoAdminTitle:    "🎁 <b>Promo Codes</b>",
	KeyPromoAdminEmpty:    "No codes created yet.",
	KeyPromoAskCode:       "🔤 Enter the code text (e.g. WELCOME50):",
	KeyPromoAskAmount:     "💰 How much TON credit per redemption?",
	KeyPromoAskMaxUses:    "🔢 Max number of redemptions? (0 = unlimited)",
	KeyPromoAskDays:       "📅 Expires in how many days? (0 = never)",
	KeyPromoCreateError:   "❌ Failed to create the code — check the values.",
	KeyPromoDuplicate:     "❌ A code with that text already exists.",
	KeyPromoCreated:       "✅ <b>Code created.</b>\n\n🔤 Code: <code>%s</code>\n💰 Credit: %.2f TON\n🔢 Max uses: %d\n📅 Expires in: %d days",
	KeyPromoDeleteConfirm: "Delete this code?",
	KeyPromoDeleted:       "🗑 Code deleted.",

	// ── Admin — source-service worker ────────────────────
	KeyMenuSourceWorkers:  "🛰 Source Workers",
	KeyBtnAddSourceWorker: "➕ Add Worker",
	KeyBtnDeleteSW:        "🗑 Delete",
	KeyBtnToggleSW:        "🔁 Enable/Disable",
	KeySWTitle:            "🛰 <b>Source-service Workers</b>\n\nEach row is a license/Telegram account a worker activates via source.worker.register.",
	KeySWEmpty:            "No workers registered yet.",
	KeySWAskAppID:         "🔢 Enter the Telegram <code>app_id</code> (from my.telegram.org):",
	KeySWAskAppHash:       "🔑 Enter the <code>app_hash</code>:",
	KeySWAskPhone:         "📱 Enter the account's phone number (with country code, e.g. +989123456789):",
	KeySWAskLabel:         "📝 Enter a label to identify this worker (optional — send \"-\" to skip):",
	KeySWInvalidAppID:     "❌ app_id must be a number. Try again:",
	KeySWCreated:          "✅ <b>Worker created.</b>\n\nEnter these in that source-service worker's config (shown in full only this once):\n\n🏷 Label: %s\n🆔 Worker ID: <code>%s</code>\n🔑 License Key: <code>%s</code>",
	KeySWCreateError:      "❌ Failed to create worker.",
	KeySWDeleted:          "🗑 Worker deleted.",
	KeySWToggledOn:        "✅ Worker enabled.",
	KeySWToggledOff:       "⛔️ Worker disabled.",
	KeySWNotFound:         "❌ Worker not found.",
	KeySWDeleteConfirm:    "Delete this worker? (If still active, its future register/heartbeat calls will be rejected.)",

	// ── wizard — custom config ─────────────────────────────
	KeyWizardConfigField: "⚙️ <b>Custom settings (%d of %d)</b>\n\n📝 <b>%s</b>\n📌 Default: <code>%s</code>\n\nSend your value or /skip to use the default:",
	KeyWizardConfigDone:  "✅ Settings saved — your bot is being created...",

	// ── admin — template ConfigSchema ─────────────────────
	KeyTmplAskSchema:     "📋 Send the JSON array of configurable fields for this template:\n\n<code>[{\"key\":\"CHANNEL_ID\",\"label\":\"Channel ID\",\"default\":\"0\",\"required\":false}]</code>\n\nTo remove the schema entirely, just send <code>[]</code>.",
	KeyTmplSchemaSet:     "✅ Template schema saved.",
	KeyTmplSchemaInvalid: "❌ Invalid JSON — please try again.",
	KeyBtnEditSchema:     "⚙️ Fields",
}
