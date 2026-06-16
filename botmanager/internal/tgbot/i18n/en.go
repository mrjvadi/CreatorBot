package i18n

var en = map[Key]string{
	KeyError:     "❌ An error occurred. Please try again.",
	KeyCancelled: "✅ Operation cancelled.",
	KeyDone:      "✅ Done.",
	KeyBack:      "🔙 Back",
	KeyCancel:    "❌ Cancel",
	KeyConfirm:   "✅ Confirm",
	KeyLoading:   "⏳ Loading...",
	KeyNotFound:  "❌ Not found.",
	KeyNoAccess:  "⛔ Access denied.",

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

	KeyMenuUsers:     "👥 Users",
	KeyMenuCampaigns: "📢 Campaigns",
	KeyMenuFinance:   "💰 Finance",
	KeyMenuFraud:     "🚨 Fraud",
	KeyMenuStats:     "📈 Statistics",
	KeyMenuSystem:    "⚙️ System",
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

	KeySubActiveNoBot: "✅ <b>Plan %s is active!</b>\n\n🤖 Capacity: %d bots\n📅 Expires: %s\n\nGo to «My Services» to create your bot.",
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
	KeyAdminTemplates:    "📦 <b>%s</b> — %s:%s",
	KeyServersTitle:      "🖥 <b>Servers</b> (%d servers)",
	KeyServersEmpty:      "🖥 <b>Servers</b>\n\nNo servers registered.",
	KeyServerAddError:    "❌ Failed to add server.",
	KeyServerDuplicate:   "❌ A server with this IP already exists.",
	KeyUsersTitle:        "👥 <b>Users</b> (%d total)",
	KeyUsersEmpty:        "👥 <b>Users</b>\n\nNo users found.",
	KeyAdminUserSummary:  "👑 %d owners | 🛡 %d admins | 👤 %d regular | 🚫 %d blocked",
	KeyUserBlocked:       "🚫 User <b>%s</b> blocked.",
	KeyUserUnblocked:     "✅ User <b>%s</b> unblocked.",
	KeyUserMadeAdmin:     "🛡 User <b>%s</b> made admin.",
	KeyUserMadeUser:      "👤 User <b>%s</b> made regular user.",
	KeyBlocked:           "🚫 Blocked",
	KeyStatsBotsLine:     "🟢 %d | 🔴 %d | 🟡 %d | ⚠️ %d",
	KeyStatsServersLine:  "🟢 %d | 🔴 %d",
	KeyStatsUsersLine:    "🛡 %d | 🚫 %d",
	KeyBotTypeVPN:        "🌐 VPN",
	KeyBotTypeUploader:   "📤 Uploader",
	KeyBotTypeMember:     "🔒 Membership Lock",
	KeyBotTypeArchive:    "📦 Archive",
	KeyNoPlan:            "❌ No active subscription.",
	KeySelectLang:        "🌍 Select your preferred language:",
	KeyBtnLimit:          "🔢 %d times",
}
