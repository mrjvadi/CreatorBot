package i18n

var fa = map[Key]string{
	// ── سیستم ────────────────────────────────────────────
	KeyError:                 "❌ خطایی رخ داد. لطفاً دوباره تلاش کنید.",
	KeyCancelled:             "✅ عملیات لغو شد.",
	KeyDone:                  "✅ انجام شد.",
	KeyBack:                  "🔙 بازگشت",
	KeyCancel:                "❌ لغو",
	KeyConfirm:               "✅ تأیید",
	KeyLoading:               "⏳ در حال بارگذاری...",
	KeyNotFound:              "❌ یافت نشد.",
	KeyComingSoon:            "🚧 این بخش به‌زودی فعال می‌شود.",
	KeyAccountTitle:          "👤 <b>پروفایل کاربری شما</b>\n🆔 شناسه: <code>%d</code>\n💰 موجودی: <b>%.2f TON</b>\n🎁 اعتبار: <b>%.2f TON</b>\n💵 مجموع: <b>%.2f TON</b>\n🌟 وضعیت: %s\n\nبرای ساخت ربات بیشتر و دسترسی به امکانات ویژه، حساب خود را شارژ کنید.",
	KeyAccountStatusStandard: "🙂 عادی",
	KeyAccountStatusVIP:      "🌟 ویژه (VIP)",
	KeyLanguageSelect:        "🌐 <b>انتخاب زبان پلتفرم</b>\n🇮🇷 لطفاً زبان مورد نظر خود را انتخاب کنید:\n🇬🇧 Please select your preferred language:",
	KeyBroadcastMenu:         "📢 <b>ماژول ارسال همگانی</b>\nپیام شما در صف قرار گرفته و به‌سرعت به همه کاربران هدف ارسال می‌شود:",
	KeySystemMenu:            "⚙️ <b>تنظیمات کلی سیستم</b>\nترجیحات اصلی ربات و تنظیمات مرکز فرماندهی را اینجا پیکربندی کنید:",
	KeyNoAccess:              "⛔ دسترسی ندارید.",

	// ── منوی اصلی کاربر ──────────────────────────────────
	KeyMenuWallet:        "💰 کیف پول",
	KeyMenuServices:      "🤖 سرویس‌های من",
	KeyMenuCommunities:   "🏘 کامیونیتی‌ها",
	KeyMenuAds:           "📢 تبلیغات",
	KeyMenuEarnings:      "📊 درآمدها",
	KeyMenuPlans:         "💎 پلن‌ها",
	KeyMenuNotifications: "🔔 اعلان‌ها",
	KeyMenuSettings:      "⚙️ تنظیمات",
	KeyMenuHelp:          "❓ راهنما",
	KeyMenuSupport:       "💬 پشتیبانی",
	KeyMenuMyBots:        "🤖 ربات‌های من",
	KeyMenuCreateBot:     "🚀 ساخت ربات جدید",
	KeyMenuAccount:       "💳 حساب و موجودی",
	KeyMenuLanguage:      "🌐 تغییر زبان",
	KeyMenuTutorials:     "📚 راهنما و آموزش",

	// ── منوی اصلی ادمین ──────────────────────────────────
	KeyMenuUsers:     "👥 کاربران",
	KeyMenuCampaigns: "📢 کمپین‌ها",
	KeyMenuFinance:   "💰 مالی",
	KeyMenuFraud:     "🚨 تقلب",
	KeyMenuStats:     "📈 آمار",
	KeyMenuSystem:    "⚙️ تنظیمات سیستم",
	KeyMenuBroadcast: "📢 پیام همگانی",
	KeyMenuExitAdmin: "🚪 خروج از پنل ادمین",
	KeyMenuBots:      "🤖 ربات‌ها",
	KeyMenuLinks:     "🔗 لینک‌ها",
	KeyMenuServers:   "🖥 سرورها",
	KeyMenuTemplates: "📦 تمپلیت‌ها",

	// ── خوش‌آمد ──────────────────────────────────────────
	KeyWelcomeUser: `👋 سلام <b>%s</b>!

به CreatorBot خوش آمدید.
با این پلتفرم می‌توانید:
• ربات‌های تلگرام بسازید و مدیریت کنید
• درآمد کسب کنید
• کامیونیتی‌ها را مدیریت کنید
• تبلیغات هدفمند اجرا کنید

از منوی پایین شروع کنید 👇`,

	KeyWelcomeAdmin: `👑 سلام <b>%s</b>!

به پنل مدیریت CreatorBot خوش آمدید.
از منوی پایین به بخش‌های مختلف دسترسی دارید 👇`,

	KeyHelpText: `📚 <b>راهنمای CreatorBot</b>

💰 <b>کیف پول</b> — موجودی، واریز، برداشت
🤖 <b>سرویس‌های من</b> — ایجاد و مدیریت ربات‌ها
🏘 <b>کامیونیتی‌ها</b> — ثبت و مدیریت گروه‌ها و کانال‌ها
📢 <b>تبلیغات</b> — ایجاد کمپین تبلیغاتی
📊 <b>درآمدها</b> — مشاهده گزارش درآمد
💎 <b>پلن‌ها</b> — ارتقا و مدیریت اشتراک
🔔 <b>اعلان‌ها</b> — تنظیم اعلان‌ها
⚙️ <b>تنظیمات</b> — زبان، امنیت، پشتیبانی

❓ سؤال دارید؟ از بخش پشتیبانی کمک بگیرید.`,

	KeyHelpAdmin: `👑 <b>پنل ادمین CreatorBot</b>

👥 <b>کاربران</b> — مدیریت و جستجوی کاربران
🤖 <b>سرویس‌ها</b> — مدیریت ربات‌های کاربران
🏘 <b>کامیونیتی‌ها</b> — نظارت بر گروه‌ها و کانال‌ها
📢 <b>کمپین‌ها</b> — مدیریت تبلیغات
💰 <b>مالی</b> — گزارش‌های مالی و برداشت‌ها
🚨 <b>تقلب</b> — نظارت و مقابله با تقلب
📈 <b>آمار</b> — آمار کلی پلتفرم
⚙️ <b>سیستم</b> — پلن‌ها، سرورها، تنظیمات`,

	// ── کیف پول ──────────────────────────────────────────
	KeyWalletHome: `💰 <b>کیف پول</b>

💎 موجودی TON: <b>%.4f</b>
🎁 اعتبار: <b>%.4f</b>
💵 کل قابل برداشت: <b>%.4f</b>`,

	KeyWalletDeposit:    "📥 <b>واریز</b>\n\nروش واریز را انتخاب کنید:",
	KeyWalletWithdraw:   "📤 <b>برداشت</b>\n\nروش برداشت را انتخاب کنید:",
	KeyWalletTransfer:   "🔄 <b>انتقال داخلی</b>\n\nشناسه تلگرام گیرنده را وارد کنید:",
	KeyWalletHistory:    "📜 <b>تاریخچه تراکنش‌ها</b>",
	KeyWalletRewards:    "🎁 <b>پاداش‌ها</b>\n\nپاداش‌های کسب‌شده شما:",
	KeyWalletLowBalance: "❌ موجودی کافی نیست.\n\n💡 برای شارژ کیف پول از دکمه واریز استفاده کنید.",

	// ── سرویس‌ها ─────────────────────────────────────────
	KeyServicesHome: "🤖 <b>سرویس‌های من</b>\n\n%d سرویس فعال",
	KeyServicesEmpty: `🤖 <b>سرویس‌های من</b>

هنوز سرویسی ندارید.
با کلیک روی «ایجاد سرویس» اولین ربات خود را بسازید! 🚀`,

	KeyServiceCreate:     "🆕 <b>ایجاد سرویس جدید</b>",
	KeyServiceSelectType: "🤖 <b>نوع سرویس را انتخاب کنید:</b>",
	KeyServiceSelectTag:  "🏷 <b>نسخه (تگ) سرویس را انتخاب کنید:</b>\n\nسرویس: <b>%s</b>",
	KeyServiceSelectPlan: "💎 <b>پلن را انتخاب کنید:</b>\n\nسرویس: %s",
	KeyServiceEnterToken: `🔑 <b>توکن ربات</b>

پلن: <b>%s</b> — <b>%.2f TON</b>

توکن ربات خود را از @BotFather دریافت کرده و ارسال کنید:

<code>1234567890:ABCDefghijklmnop...</code>

📌 نحوه دریافت توکن:
1. @BotFather را در تلگرام باز کنید
2. /newbot را ارسال کنید
3. نام و یوزرنیم ربات را وارد کنید
4. توکن دریافتی را اینجا ارسال کنید`,

	KeyServiceConfirm: `✅ <b>تأیید ایجاد سرویس</b>

🤖 نوع: <b>%s</b>
🏷 تگ: <b>%s</b>
💎 پلن: <b>%s</b>
💰 قیمت: <b>%.2f TON</b>

آیا مطمئن هستید؟`,

	KeyServiceCreating: "⏳ <b>در حال راه‌اندازی سرویس...</b>\n\nلطفاً صبر کنید.",
	KeyServiceCreated: `🎉 <b>سرویس با موفقیت ایجاد شد!</b>

🤖 نوع: <b>%s</b>
💎 پلن: <b>%s</b>
📦 وضعیت: <b>در حال راه‌اندازی</b>

⏱ معمولاً ظرف ۲ تا ۳ دقیقه آماده می‌شود.
از منوی «سرویس‌های من» وضعیت را پیگیری کنید.`,

	KeyServiceFailed:       "❌ <b>راه‌اندازی سرویس ناموفق بود.</b>\n\nمبلغ پرداخت‌شده به کیف پول شما بازگشت داده شد.",
	KeyServiceNoCapacity:   "❌ <b>ظرفیت تکمیل است.</b>\n\nبا پلن فعلی نمی‌توانید سرویس جدید ایجاد کنید.\n\n💡 برای افزایش ظرفیت پلن خود را ارتقا دهید.",
	KeyServiceInvalidToken: "❌ <b>توکن نامعتبر است.</b>\n\nمثال صحیح: <code>1234567890:ABCDefgh...</code>",
	KeyServiceDuplicate:    "❌ <b>این ربات قبلاً ثبت شده است.</b>\n\nهر ربات فقط یک بار قابل استفاده است.",

	// ── پلن‌ها ────────────────────────────────────────────
	KeyPlansHome: `💎 <b>پلن‌ها</b>

پلن مناسب خود را انتخاب کنید:`,

	KeyPlanCurrent: `💎 <b>پلن فعلی شما</b>

📦 پلن: <b>%s</b>
🤖 ربات‌ها: <b>%d / %d</b>
📅 انقضا: <b>%s</b>
⏰ روزهای باقی‌مانده: <b>%d روز</b>`,

	KeyPlanNone:    "❌ <b>اشتراک فعالی ندارید.</b>\n\nبرای استفاده از امکانات پلتفرم یک پلن انتخاب کنید.",
	KeyPlanExpired: "⚠️ <b>اشتراک شما منقضی شده است.</b>\n\nبرای ادامه استفاده پلن خود را تمدید کنید.",
	KeyPlanUpgrade: "🚀 <b>ارتقای پلن</b>\n\nپلن مقصد را انتخاب کنید:",
	KeyPlanBuyTitle: `💎 <b>%s</b>

⏱ مدت: <b>%d روز</b>
🤖 حداکثر ربات: <b>%d</b>
💰 قیمت: <b>%.2f TON</b>`,

	KeyPlanBought:   "🎉 <b>پلن %s با موفقیت فعال شد!</b>\n\nاز منوی «سرویس‌های من» ربات خود را بسازید.",
	KeyNoFreePlan:   "❌ در حال حاضر پلن رایگان موجود نیست.",
	KeyFreePlanDone: "✅ <b>پلن رایگان فعال شد!</b>\n\nاکنون می‌توانید اولین ربات خود را بسازید.",

	// ── کامیونیتی‌ها ─────────────────────────────────────
	KeyCommHome: `🏘 <b>کامیونیتی‌ها</b>

گروه‌ها و کانال‌های خود را مدیریت کنید و از تبلیغات درآمد کسب کنید.`,

	KeyCommEmpty: `🏘 <b>کامیونیتی‌ها</b>

هنوز کامیونیتی ثبت نکرده‌اید.

با ثبت گروه یا کانال خود می‌توانید از نمایش تبلیغات درآمد کسب کنید! 💰`,

	KeyCommRegister: "➕ <b>ثبت کامیونیتی جدید</b>\n\nنوع کامیونیتی را انتخاب کنید:",
	KeyCommVerify:   "🔍 <b>تأیید کامیونیتی</b>\n\nربات را به کامیونیتی اضافه کرده و ادمین کنید، سپس تأیید را بزنید.",

	// ── تبلیغات ──────────────────────────────────────────
	KeyAdsHome: `📢 <b>تبلیغات</b>

کمپین‌های تبلیغاتی خود را مدیریت کنید.`,

	KeyAdsEmpty: `📢 <b>تبلیغات</b>

هنوز کمپینی ایجاد نکرده‌اید.

با ایجاد کمپین، تبلیغ خود را در کانال‌های هدف نمایش دهید! 📣`,

	KeyAdsCreate: "➕ <b>کمپین جدید</b>\n\nنام کمپین را وارد کنید:",

	// ── درآمدها ──────────────────────────────────────────
	KeyEarningsHome: `📊 <b>درآمدها</b>

💰 کل درآمد: <b>%.4f TON</b>
📅 امروز: <b>%.4f TON</b>
📆 این ماه: <b>%.4f TON</b>`,

	KeyEarningsEmpty: "📊 <b>درآمدها</b>\n\nهنوز درآمدی ثبت نشده است.\n\n💡 با فعال کردن کامیونیتی‌ها و سرویس‌ها درآمد کسب کنید.",

	// ── تنظیمات ──────────────────────────────────────────
	KeySettingsHome: "⚙️ <b>تنظیمات</b>\n\nاز بخش‌های زیر تنظیمات خود را مدیریت کنید:",
	KeyLangChanged:  "✅ زبان با موفقیت تغییر کرد.",
	KeyLangSelect:   "🌍 <b>انتخاب زبان</b>\n\nزبان مورد نظر خود را انتخاب کنید:",

	// ── اعلان‌ها ──────────────────────────────────────────
	KeyNotificationsHome: "🔔 <b>اعلان‌ها</b>\n\nنوع اعلان‌های مورد نظر را انتخاب کنید:",

	// ── پشتیبانی ─────────────────────────────────────────
	KeySupportText: `💬 <b>پشتیبانی CreatorBot</b>

📚 مستندات: t.me/CreatorBotDocs
💬 پشتیبانی: @CreatorBotSupport
🐞 گزارش مشکل: @CreatorBotBug

⏰ ساعت پاسخگویی: ۹ صبح تا ۱۰ شب`,

	// ── ادمین — کاربران ──────────────────────────────────
	KeyAdminUsersTitle: `👥 <b>کاربران</b> (%d نفر)

👑 %d مالک | 🛡 %d ادمین | 👤 %d عادی | 🚫 %d مسدود`,

	KeyAdminUserDetail: `👤 <b>اطلاعات کاربر</b>

🏷 نام: <b>%s</b>
🔗 یوزرنیم: %s
🆔 شناسه: <code>%d</code>
👑 نقش: <b>%s</b>
🚫 وضعیت: %s
🤖 ربات‌های فعال: %d`,

	KeyAdminUserBlocked:   "🚫 کاربر <b>%s</b> مسدود شد.",
	KeyAdminUserUnblocked: "✅ کاربر <b>%s</b> از حالت مسدود خارج شد.",

	// ── ادمین — ربات‌ها ───────────────────────────────────
	KeyBotsEmpty: "🤖 <b>هیچ ربات فعالی وجود ندارد.</b>",

	// ── ادمین — سرور ─────────────────────────────────────
	KeyServerAskName: "🖥 <b>سرور جدید</b>\n\nنام سرور را وارد کنید:\n<i>مثال: server-eu-1</i>",
	KeyServerAskIP:   "🌐 آدرس IP سرور را وارد کنید:\n<i>مثال: 192.168.1.1</i>",
	KeyServerAdded:   "✅ سرور <b>%s</b> با موفقیت اضافه شد.",

	// ── ادمین — تمپلیت ───────────────────────────────────
	KeyTemplateAskType:  "📦 <b>تمپلیت جدید</b>\n\nنوع سرویس را انتخاب کنید:",
	KeyTemplateAskImage: "🐳 نام Docker image را وارد کنید:\n<i>مثال: creatorbot/vpn-bot</i>",
	KeyTemplateAskTag:   "🏷 تگ image را وارد کنید:\n<i>مثال: latest یا v1.2.3</i>",
	KeyTmplFreeAdded:    "✅ تمپلیت رایگان با موفقیت تعریف شد.",
	KeyTmplFreeExists:   "⚠️ تمپلیت رایگان برای این نوع قبلاً وجود دارد.",

	// ── ادمین — پلن ──────────────────────────────────────
	KeyPlanAskName:  "💎 <b>پلن جدید</b>\n\nنام پلن را وارد کنید:",
	KeyPlanAskPrice: "💰 قیمت پلن را به TON وارد کنید:\n<i>مثال: 5.0</i>",
	KeyPlanAskDays:  "📅 مدت پلن را به روز وارد کنید:\n<i>مثال: 30</i>",
	KeyPlanAskBots:  "🤖 حداکثر تعداد ربات را وارد کنید:\n<i>مثال: 3</i>",
	KeyPlanAdded:    "✅ پلن <b>%s</b> با موفقیت اضافه شد.",

	// ── ادمین — لینک ─────────────────────────────────────
	KeyLinkAskType:  "🔗 <b>لینک دعوت جدید</b>\n\nنوع سرویس را انتخاب کنید:",
	KeyLinkAskLimit: "🔢 تعداد دفعات استفاده را انتخاب کنید:",
	KeyLinkCreated:  "✅ لینک دعوت ایجاد شد:\n\n<code>%s</code>\n\nتعداد استفاده: %d بار",

	// ── آمار ─────────────────────────────────────────────
	KeyStatsTitle: `📈 <b>آمار سیستم</b>
⏰ %s

🤖 <b>ربات‌ها</b> (%d کل)
🟢 فعال: %d | 🔴 متوقف: %d | 🟡 در انتظار: %d | ⚠️ خطا: %d

🖥 <b>سرورها</b> (%d کل)
🟢 آنلاین: %d | 🔴 آفلاین: %d

👥 <b>کاربران</b> (%d کل)
🛡 ادمین: %d | 🚫 مسدود: %d

📦 %d پلن | 💰 %d کیف پول`,

	// ── ساب‌اسکریپشن ─────────────────────────────────────
	KeySubActiveNoBot: "✅ <b>پلن %s فعال است!</b>\n\nهنوز ربات فعالی ندارید.\nبرای ساخت ربات روی دکمه زیر کلیک کنید:",

	KeyBuildWithLink: `🔗 <b>لینک دعوت معتبر</b>

سرویس: <b>%s</b>
برای ادامه، نوع سرویس خود را تأیید کنید:`,

	// ── wizard ──────────────────────────────────────────
	KeyWizardInvalidLink: "❌ <b>لینک دعوت نامعتبر است.</b>\n\nلطفاً لینک صحیح را از ادمین دریافت کنید.",
	KeyWizardExpiredLink: "⏰ <b>لینک دعوت منقضی شده است.</b>\n\nلطفاً لینک جدید از ادمین دریافت کنید.",
	KeyWizardUsedLink:    "❌ <b>این لینک دعوت قبلاً استفاده شده است.</b>",
	KeyWizardAskToken:    "🔑 <b>توکن ربات</b>\n\nتوکن ربات خود را از @BotFather دریافت کرده و ارسال کنید:\n\n<code>1234567890:ABCDefgh...</code>",

	// ── How-to ──────────────────────────────────────────
	KeyHowToBuild: `📘 <b>نحوه ساخت ربات</b>

1️⃣ به @BotFather در تلگرام بروید
2️⃣ دستور /newbot را ارسال کنید
3️⃣ نام ربات را وارد کنید
4️⃣ یوزرنیم ربات را انتخاب کنید (باید به bot ختم شود)
5️⃣ توکن دریافتی را کپی کنید
6️⃣ توکن را اینجا ارسال کنید ✅`,

	KeyHowToBuildDone: "✅ متوجه شدم، ادامه می‌دهم",

	// ── دکمه‌ها ───────────────────────────────────────────
	KeyBtnYesBuild:  "✅ بله، ربات دارم",
	KeyBtnCancel:    "❌ لغو",
	KeyBtnBack:      "🔙 بازگشت",
	KeyBtnLimit1:    "1️⃣ یک بار",
	KeyBtnLimit3:    "3️⃣ سه بار",
	KeyBtnLimit5:    "5️⃣ پنج بار",
	KeyBtnLimit10:   "🔟 ده بار",
	KeyBtnLimitNo:   "♾️ نامحدود",
	KeyBtnBlock:     "🚫 مسدود کردن",
	KeyBtnUnblock:   "✅ رفع مسدودیت",
	KeyBtnMakeAdmin: "🛡 ادمین کردن",
	KeyBtnMakeUser:  "👤 کاربر عادی",
	// ── ادمین — ربات‌ها
	KeyBotsTitle:       "🤖 <b>ربات‌ها</b> (%d کل)",
	KeyAdminBotSummary: "%s <b>%s</b> — %s",
	KeyBotNotFound:     "❌ ربات یافت نشد.",
	KeyBotStopped:      "⏹ ربات <b>%s</b> متوقف شد.",
	KeyBotStarted:      "▶️ ربات <b>%s</b> شروع به کار کرد.",
	KeyBotDeleted:      "🗑 ربات <b>%s</b> حذف شد.",
	KeyBotActionFailed: "⚠️ این عملیات با خطا مواجه شد؛ سرور شاید آفلاین باشد. کمی بعد دوباره امتحان کن یا لاگ سرور را چک کن.",

	// ── ادمین — لینک‌ها
	KeyLinksTitle:      "🔗 <b>لینک‌های دعوت</b> (%d لینک)",
	KeyLinksEmpty:      "🔗 <b>لینک‌های دعوت</b>\n\nهیچ لینکی وجود ندارد.",
	KeyLinkAskLabel:    "🏷 یک برچسب برای این لینک وارد کنید (یا skip ارسال کنید):",
	KeyLinkCreateError: "❌ خطا در ایجاد لینک.",
	KeyAdminLinkStats:  "🔗 <code>%s</code>\n📦 %s | 🔢 %d/%s | ⏰ %s",
	KeyAdminLinkLimitX: "%d بار",

	// ── ادمین — پلن‌ها
	KeyPlansTitle:        "💎 <b>پلن‌ها</b> (%d پلن)",
	KeyPlansEmpty:        "💎 <b>پلن‌ها</b>\n\nهیچ پلنی تعریف نشده است.",
	KeyPlansNoTemplate:   "❌ ابتدا یک تمپلیت تعریف کنید.",
	KeyPlanAskTemplate:   "📦 تمپلیت سرویس را انتخاب کنید:",
	KeyPlanInvalidNumber: "❌ عدد معتبر وارد کنید.",
	KeyPlanTmplNotFound:  "❌ تمپلیت یافت نشد.",
	KeyPlanAddError:      "❌ خطا در ایجاد پلن.",
	KeyAdminPlanLine:     "💎 <b>%s</b> — %.1f TON | %d روز | %d ربات",
	KeyAdminPlanFree:     "🆓 رایگان",

	// ── ادمین — تمپلیت‌ها
	KeyTemplatesTitle:   "📦 <b>تمپلیت‌ها</b> (%d تمپلیت)",
	KeyTemplatesEmpty:   "📦 <b>تمپلیت‌ها</b>\n\nهیچ تمپلیتی تعریف نشده است.",
	KeyTemplateAskName:  "📦 <b>تمپلیت جدید</b>\n\nنام تمپلیت را وارد کنید:",
	KeyTemplateAdded:    "✅ تمپلیت <b>%s</b> اضافه شد.",
	KeyTemplateAddError: "❌ خطا در ایجاد تمپلیت.",

	// ── دکمه‌ها و تستِ سرویس ──
	KeyBtnTest:             "🧪 تست",
	KeyBtnNewTemplate:      "➕ تمپلیت جدید",
	KeyBtnAddCredit:        "💰 افزودن اعتبار",
	KeyBtnBackToList:       "🔙 بازگشت به لیست",
	KeyBtnAddServer:        "➕ افزودن سرور",
	KeyBtnViewWallet:       "💰 مشاهده کیف پول",
	KeyBtnConfirmDelete:    "🗑 بله، حذف شود",
	KeyBtnGotIt:            "✅ متوجه شدم",
	KeyBalanceUpdated:      "✅ <b>موجودی به‌روز شد</b>\n\n💎 موجودی کل: <b>%.4f TON</b>",
	KeyBalanceAlert:        "💎 موجودی کل: %.4f TON",
	KeyPaymentPendingAlert: "⏳ پرداخت هنوز تأیید نشده.\n💳 موجودی: %.4f TON | نیاز: %.2f TON\n⚠️ کمبود: %.4f TON\nچند لحظه بعد دوباره بررسی کنید.",
	KeyTxPending:           "⏳ تراکنش هنوز دریافت نشده. چند لحظه بعد دوباره بررسی کنید.",
	KeyTxPaid:              "✅ تراکنش دریافت شد!",
	KeyTxPartial:           "🔸 بخشی از مبلغ دریافت شد (%.4f از %.4f TON). منتظر باقی‌مانده.",
	KeyTxExpired:           "❌ این فاکتور منقضی شده. لطفاً فاکتور جدید بسازید.",
	KeyTxNotFound:          "❓ تراکنشی با این کد یافت نشد.",
	KeyTxCheckFailed:       "⚠️ بررسی وضعیت ممکن نشد. کمی بعد دوباره تلاش کنید.",
	KeyBtnDepositTON:       "📥 واریز TON",
	KeyBtnHistory:          "📜 تاریخچه",
	KeyBtnRedeemPromo:      "🎁 استفاده از کد تخفیف",
	KeyBtnNewDeposit:       "📥 واریز جدید",
	KeyBtnCheckPayment:     "🔄 بررسی پرداخت",
	KeyBtnBcText:           "💬 ارسال متنی",
	KeyBtnBcForward:        "🔄 فوروارد همگانی",
	KeyBtnBcFiltered:       "🎯 ارسال فیلترشده",
	KeyBtnCreateFree:       "✅ ایجاد رایگان",
	KeyBtnPayCreate:        "✅ پرداخت و ایجاد",

	// ── عمومی / وضعیت ──
	KeyFree:           "رایگان",
	KeyDaysCount:      "%d روز",
	KeyForever:        "∞ ابدی",
	KeyStatusActive:   "✅ فعال",
	KeyStatusInactive: "⛔ غیرفعال",
	KeyErrShort:       "❌ خطا",
	KeyErrSave:        "❌ خطا در ذخیره",

	// ── ادمین — پلن (ادیتور) ──
	KeyPlanNotFound:       "❌ پلن پیدا نشد.",
	KeyBtnNewPlan:         "➕ پلن جدید",
	KeyBtnBackToPlans:     "🔙 بازگشت به پلن‌ها",
	KeyBtnEditPlan:        "⚙️ ویرایش: %s",
	KeyBtnTotalCap:        "🤖 سقف کلی:  %d",
	KeyAdminPlanRow:       "%s 💎 <b>%s</b> — %s TON | %d روز | سقف %d ربات",
	KeyPlanEditTitle:      "⚙️ <b>ویرایش پلن: %s</b>\n💰 %s  |  ⏳ %s  |  %s\n\n📊 <b>محدودیت ربات‌ها:</b>\n<i>(هر تغییر بلافاصله ذخیره می‌شود)</i>",
	KeyAvailableTemplates: "📦 <b>تمپلیت‌های موجود:</b>",
	KeyPlanTmplChosen:     "تمپلیت: <b>%s</b>\n\n%s",
	KeyPlanLimitsPrompt:   "حالا محدودیت هر نوع ربات را وارد کنید.\n\nفرمت: <code>نوع=تعداد</code> جداشده با کاما\nمثال: <code>uploader=2,vpn=1</code>\n\nانواع: %s\nیا فقط یک عدد بفرستید تا برای همه انواع اعمال شود.",
	KeyPlanLimitsInvalid:  "فرمت نامعتبر. مثال: <code>uploader=2,vpn=1</code>",
	KeyPlanLimitsSaved:    "✅ <b>محدودیت‌ها ثبت شد</b>\n\n%s\n\nمجموع: %d ربات",

	// ── کاربر — پلن‌ها (UI) ──
	KeyDurationForever:          "برای همیشه",
	KeyBtnMyBots:                "🤖 ربات‌های من",
	KeyBtnTopupWallet:           "💎 شارژ کیف پول",
	KeyBtnRecheck:               "🔄 بررسی مجدد",
	KeyBtnClose:                 "❌ بستن",
	KeyBtnBuyWith:               "✅ خرید با %.2f TON",
	KeyPlansUnavailable:         "در حال حاضر پلنی موجود نیست. بعداً دوباره بررسی کنید.",
	KeyPlansAvailableTitle:      "<b>💎 پلن‌های موجود</b>\n\n",
	KeyPlanRemDays:              " — %d روز مانده",
	KeyPlanExpiredShort:         " — منقضی شده",
	KeyPlanActiveYours:          "✅ <b>پلن فعال شما:</b> %s%s\n\n",
	KeyPlanRow:                  "<b>%s</b>\n💰 %s  |  🤖 %d ربات  |  ⏳ %s\n\n",
	KeyPlansClickToBuy:          "برای خرید روی پلن مورد نظر کلیک کنید:",
	KeyPlanLabelFree:            "🆓 %s — رایگان",
	KeyPlanLabelPaid:            "💎 %s — %.2f TON",
	KeyPlanAlreadyActive:        "✅ این پلن در حال حاضر برای شما فعال است.",
	KeyPlanDetail:               "<b>💎 %s</b>\n\n🤖 تعداد ربات: %d عدد\n⏳ مدت: %s\n💰 قیمت: <b>%.2f TON</b>\n\n",
	KeyWalletBalanceLine:        "💳 موجودی کیف پول شما: <b>%.4f TON</b>\n",
	KeyBalanceEnough:            "\n✅ موجودی شما کافی است!",
	KeyBalanceShortfall:         "\n⚠️ کمبود موجودی: <b>%.4f TON</b>",
	KeyDepositAddrCode:          "\n\n💎 آدرس: <code>%s</code>\n🏷 کد (حتماً در comment وارد کنید): <code>%s</code>",
	KeyPayServiceUnavailable:    "\n⚠️ سرویس پرداخت موقتاً در دسترس نیست.",
	KeyFreePlanActivated:        "🎉 <b>پلن رایگان فعال شد!</b>\n\n✅ %d ربات — %s\n\nحالا می‌توانید سرویس خود را بسازید.",
	KeyPlanPurchaseDesc:         "خرید پلن %s",
	KeyPurchaseSuccess:          "🎉 <b>خرید موفق!</b>\n\n✅ پلن <b>%s</b> فعال شد\n🤖 %d ربات در اختیار شماست\n\nحالا می‌توانید سرویس خود را بسازید.",
	KeyPurchaseActivationFailed: "😥 مبلغ از کیف پولت کسر شد ولی فعال‌سازی پلن با یک خطای فنی مواجه شد.\n\n💰 مبلغ برایت برگردانده شد — نگران نباش، چیزی از دست نرفته.\nلطفاً دوباره تلاش کن؛ اگر باز هم مشکل داشتی، با پشتیبانی در تماس باش.",
	KeyPaymentNotConfirmed:      "⏳ پرداخت هنوز تأیید نشده.\n\n💳 موجودی فعلی: <b>%.4f TON</b>\n💰 نیاز: <b>%.2f TON</b>\n⚠️ کمبود: %.4f TON\n\nچند دقیقه صبر کنید و دوباره بررسی کنید.",

	// ── UX — ویزارد و badgeها ──
	KeyWizardStep:      "🧭 مرحله %d از %d",
	KeyBadgePopular:    "🔥 محبوب",
	KeyBadgeNewest:     "🆕 جدید",
	KeyBtnCustomAmount: "✏️ مبلغ دلخواه",
	KeyBtnRenew:        "🔄 تمدید / ارتقا",

	// ── تمدید سرویس و یادآور انقضا ──
	KeyRenewConfirm:    "🔄 <b>تمدید سرویس</b>\n\n📛 <code>%s</code>\n💎 پلن: <b>%s</b>\n💰 هزینه تمدید: <b>%.2f TON</b>\n\nاز کیف پولت کسر و سرویس تمدید می‌شود. تأیید می‌کنی؟",
	KeyBtnConfirmRenew: "✅ تأیید و پرداخت",
	KeyRenewDone:       "🎉 <b>سرویس تمدید شد!</b>\n\n📛 <code>%s</code>\n⏰ %s",
	KeyRenewNoPlan:     "❌ این سرویس پلن مشخصی برای تمدید ندارد. لطفاً از «پلن‌ها» اقدام کن.",
	KeyExpiryReminder:  "⏰ <b>یادآوری انقضا</b>\n\n📛 سرویس <code>%s</code> تا <b>%d روز</b> دیگر منقضی می‌شود.\nبرای جلوگیری از قطع شدن، همین حالا تمدید کن. 🔄",

	// ── کاربر — سرویس‌های من (UI) ──
	KeyBtnStats:           "📊 آمار",
	KeyBtnSettings:        "⚙️ تنظیمات",
	KeyBtnRestart:         "🔄 ری‌استارت",
	KeyBtnStop:            "⏸ توقف",
	KeyBtnStart:           "▶️ شروع",
	KeyBtnDeleteSvc:       "🗑 حذف سرویس",
	KeyBtnDelete:          "🗑 حذف",
	KeyBtnCheckStatus:     "🔄 بررسی وضعیت",
	KeyBtnRetry:           "🔄 تلاش مجدد",
	KeyBtnCreateSvc:       "➕ ایجاد سرویس",
	KeyBtnCreateNewSvc:    "➕ ایجاد سرویس جدید",
	KeyBtnStartFree:       "🆓 شروع رایگان",
	KeyBtnViewPlans:       "💎 مشاهده پلن‌ها",
	KeyBtnUpgradePlan:     "💎 ارتقای پلن",
	KeyMyServicesHeader:   "<b>🤖 سرویس‌های من</b> (%d سرویس)\n",
	KeySvcNameLine:        "📛 نام: <code>%s</code>\n",
	KeySvcStatusLine:      "%s وضعیت: <b>%s</b>\n",
	KeySvcExpiredNL:       "⏰ <b>منقضی شده</b>\n",
	KeySvcHoursLeft:       "⚠️ %d ساعت تا انقضا\n",
	KeySvcDaysLeft:        "⏰ %d روز مانده\n",
	KeyWelcomeNoService:   "👋 سلام <b>%s</b>!\n\nبا CreatorBot می‌توانید ربات تلگرام اختصاصی بسازید.\n\n🆓 <b>پلن رایگان:</b>\nیک ربات رایگان برای همیشه\n\n💎 <b>پلن‌های پولی:</b>\nچند ربات — با امکانات بیشتر\n\nبرای شروع روی «شروع رایگان» کلیک کنید:",
	KeyNeedPlanFirst:      "برای ساخت ربات باید ابتدا یک پلن داشته باشید.\n\nیک ربات رایگان می‌توانید داشته باشید:",
	KeyMaxBotsReached:     "❌ به حداکثر ربات رسیده‌اید.\n\n🤖 %d از %d ربات استفاده شده\n\nبرای ساخت ربات بیشتر پلن خود را ارتقا دهید.",
	KeyStatusRunning:      "در حال اجرا",
	KeyStatusStopped:      "متوقف",
	KeyStatusStarting:     "در حال راه‌اندازی...",
	KeyStatusErrContact:   "خطا — با پشتیبانی تماس بگیرید",
	KeyTypeNotAllowed:     "❌ پلن فعلی شما اجازه ساخت ربات <b>%s</b> را نمی‌دهد.\n\nبرای دسترسی، پلن خود را ارتقا دهید.",
	KeyMaxBotsReachedType: "❌ به حداکثر ربات <b>%s</b> رسیده‌اید (%d از %d).\n\nبرای ساخت بیشتر، پلن خود را ارتقا دهید.",
	KeyActionStopSent:     "⏹ چشم، داریم سرویس رو متوقف می‌کنیم…",
	KeyActionStartSent:    "▶️ باشه، داریم سرویس رو روشن می‌کنیم…",
	KeyActionRestartSent:  "🔄 در حالِ ری‌استارتِ سرویس…",
	KeyActionDeleteSent:   "🗑 درخواستِ حذف ثبت شد — تا چند لحظه‌ی دیگه سرویس پاک می‌شه.",
	KeySvcStatusShort:     "%s وضعیت سرویس: <b>%s</b>",
	KeySvcStatsDetail:     "📊 <b>آمار سرویس</b>\n\n📛 نام: <code>%s</code>\n%s وضعیت: <b>%s</b>",
	KeyServiceGeneric:     "سرویس",
	KeyUnknown:            "نامشخص",
	KeyExpiredLabel:       "⏰ <b>منقضی شده</b>",
	KeyDaysUntilExpiry:    "⏰ %d روز مانده تا انقضا",
	KeyPlanLine:           "💎 پلن: <b>%s</b>",
	KeyAdminTestAskToken: "🧪 <b>دپلوی تستی</b>\n\nسرویس: <b>%s</b> | تگ: <b>%s</b>\n\n" +
		"توکن ربات تست را از @BotFather بفرستید (بدون پلن و پرداخت دپلوی می‌شود):",
	KeyAdminTestDeployed: "🧪 <b>سرویس تستی در حال راه‌اندازی است.</b>\n\n" +
		"📛 کانتینر: <code>%s</code>\nظرف چند دقیقه آماده می‌شود.",
	KeyAdminTemplates: "📦 <b>%s</b> — %s:%s",

	// ── ادمین — سرورها
	KeyServersTitle:        "🖥 <b>سرورها</b> (%d سرور)",
	KeyServersEmpty:        "🖥 <b>سرورها</b>\n\nهیچ سروری ثبت نشده است.",
	KeyServerAddError:      "❌ خطا در افزودن سرور.",
	KeyServerDuplicate:     "❌ سروری با این IP قبلاً ثبت شده است.",
	KeyServerDeleteConfirm: "🗑 این سرور از لیستِ deploy حذف می‌شود (رباتِ در حالِ اجرا رویش دست‌نخورده می‌ماند). مطمئنی؟",
	KeyServerDeletedMsg:    "🗑 سرور حذف شد.",

	// ── ادمین — کاربران
	KeyUsersTitle:         "👥 <b>کاربران</b> (%d نفر)",
	KeyUsersEmpty:         "👥 <b>کاربران</b>\n\nهیچ کاربری یافت نشد.",
	KeyUsersSearchPrompt:  "🔍 برای دیدنِ جزئیاتِ یک کاربر، شناسه‌ی عددیِ تلگرامش (TelegramID) رو همین‌جا بفرست:",
	KeyUsersSearchInvalid: "❌ این یک شناسه‌ی عددیِ معتبر نیست. دوباره امتحان کن (فقط عدد، بدون @):",
	KeyAdminUserSummary:   "👑 %d مالک | 🛡 %d ادمین | 👤 %d عادی | 🚫 %d مسدود",
	KeyUserBlocked:        "🚫 کاربر <b>%s</b> مسدود شد.",
	KeyUserUnblocked:      "✅ کاربر <b>%s</b> رفع مسدودیت شد.",
	KeyUserMadeAdmin:      "🛡 کاربر <b>%s</b> ادمین شد.",
	KeyUserMadeUser:       "👤 کاربر <b>%s</b> به کاربر عادی تبدیل شد.",
	KeyBlocked:            "🚫 مسدود",

	// ── آمار
	KeyStatsBotsLine:    "🟢 %d | 🔴 %d | 🟡 %d | ⚠️ %d",
	KeyStatsServersLine: "🟢 %d | 🔴 %d",
	KeyStatsUsersLine:   "🛡 %d | 🚫 %d",

	// ── نوع ربات
	KeyBotTypeVPN:      "🌐 VPN",
	KeyBotTypeUploader: "📤 آپلودر",
	KeyBotTypeMember:   "🔒 قفل ممبرشیپ",
	KeyBotTypeArchive:  "📦 آرشیو",

	// ── متفرقه
	KeyNoPlan:     "❌ اشتراک فعالی ندارید.",
	KeySelectLang: "🌍 زبان مورد نظر خود را انتخاب کنید:",
	KeyBtnLimit:   "🔢 %d بار",

	// ── wizard — خطاها ───────────────────────────────────
	KeyWizardNoPlan:      "❌ پلنی برای این نوع سرویس یافت نشد.",
	KeyWizardRestart:     "❌ لطفاً از ابتدا شروع کنید.",
	KeyWizardNoServer:    "❌ هیچ سروری در دسترس نیست. با مدیر تماس بگیرید.",
	KeyWizardNoTemplate:  "❌ قالب سرویس پیدا نشد. با مدیر تماس بگیرید.",
	KeyWizardCreateError: "❌ خطا در ایجاد سرویس. لطفاً دوباره تلاش کنید.",
	KeyWizardDeployError: "❌ خطا در ارسال دستور راه‌اندازی. مبلغ به کیف پول بازگشت داده شد.",
	KeyWizardIncomplete:  "❌ اطلاعات ناقص است. از ابتدا شروع کنید.",
	KeyWizardLowBalance:  "❌ <b>موجودی کافی نیست.</b>\n\n💡 برای شارژ کیف پول از منوی «کیف پول» استفاده کنید.",
	KeyInstanceNotFound:  "❌ سرویس یافت نشد.",
	KeyInstanceNoAccess:  "⛔ دسترسی به این سرویس را ندارید.",

	// ── ادمین — افزودن اعتبار ────────────────────────────
	KeyAdminCreditAsk:     "💰 <b>افزودن اعتبار</b>\n\nکاربر: <code>%d</code>\n\nمقدار اعتبار (به TON) را وارد کنید:\n<i>مثال: 1.5 یا 10</i>",
	KeyAdminCreditDone:    "✅ <b>%.4f TON</b> به کیف پول کاربر <code>%d</code> اضافه شد.",
	KeyAdminCreditError:   "❌ خطا در افزودن اعتبار. لطفاً دوباره تلاش کنید.",
	KeyAdminCreditInvalid: "❌ مقدار نامعتبر. یک عدد مثبت وارد کنید (مثال: 2.5)",

	// ── کیف پول — صفحه اصلی ─────────────────────────────
	KeyWalletTitle: `💰 <b>کیف پول</b>

💎 موجودی TON: <b>%.4f</b>
🎁 اعتبار هدیه: <b>%.4f</b>
💵 کل موجودی: <b>%.4f</b>`,

	// ── تنظیمات ──────────────────────────────────────────
	KeySettingsLanguage: "🌐 تغییر زبان",
	KeySettingsSupport:  "💬 پشتیبانی",
	KeySettingsAbout:    "ℹ️ درباره پلتفرم",

	// ── سرویس — تنظیمات ──────────────────────────────────
	KeySvcSettingsDetail: `⚙️ <b>تنظیمات سرویس</b>

📛 نام: <code>%s</code>
🤖 نوع: <b>%s</b>
%s وضعیت: <b>%s</b>
🖥 سرور: <code>%s</code>
%s`,

	// ── حذف — تأیید ──────────────────────────────────────
	KeyDeleteConfirm: "⚠️ <b>تأیید حذف</b>\n\nسرویس <code>%s</code> برای همیشه حذف می‌شود.\n\nاین عمل غیرقابل بازگشت است. مطمئن هستید؟",
	KeyDeleteDone:    "🗑 سرویس <code>%s</code> حذف شد.",

	// ── واریز کیف پول ────────────────────────────────────
	KeyWalletTopupAsk:     "💰 <b>شارژ کیف پول</b>\n\nمقدار TON مورد نظر برای واریز را وارد کنید:\n<i>مثال: 5.5</i>",
	KeyWalletTopupInvoice: "📥 <b>اطلاعات واریز TON</b>\n\n💰 مبلغ: <b>%.4f TON</b>\n\n📬 آدرس واریز:\n<code>%s</code>\n\n🏷 کد پیگیری (حتماً در comment تراکنش وارد کنید):\n<code>%s</code>\n\n⏰ این invoice تا ۲۴ ساعت معتبر است.",
	KeyWalletTopupInvalid: "❌ مقدار نامعتبر. یک عدد مثبت به TON وارد کنید (مثال: 5.5)",

	// ── تاریخچه تراکنش‌ها ────────────────────────────────
	KeyWalletHistoryNote: "📜 <b>تاریخچه تراکنش‌ها</b>\n\n💎 موجودی TON: <b>%.4f</b>\n🎁 اعتبار هدیه: <b>%.4f</b>\n💵 کل: <b>%.4f</b>\n\n📊 تاریخچه کامل تراکنش‌ها به‌زودی اضافه می‌شود.",

	// ── پشتیبانی و اطلاعات (inline) ──────────────────────
	KeySupportInline: `💬 <b>پشتیبانی CreatorBot</b>

برای دریافت کمک از راه‌های زیر استفاده کنید:

📚 مستندات: @CreatorBotDocs
💬 پشتیبانی: @CreatorBotSupport
🐞 گزارش باگ: @CreatorBotBug

⏰ پاسخگویی: ۹ صبح تا ۱۰ شب`,

	KeyAboutPlatform: `ℹ️ <b>درباره CreatorBot</b>

🤖 <b>CreatorBot</b> یک پلتفرم PaaS برای ساخت ربات تلگرام بدون کدنویسی است.

✨ <b>امکانات:</b>
• ربات آپلودر فایل
• ربات VPN و فروش اشتراک
• ربات قفل ممبرشیپ
• ربات آرشیو با جستجوی فارسی
• سیستم کیف پول TON یکپارچه
• تبلیغات هدفمند و درآمد گروه‌ها

💰 <b>واحد پرداخت:</b> TON Blockchain
🔒 <b>امنیت:</b> احراز هویت پیشرفته بین سرویس‌ها

📌 نسخه: ۳.۰`,

	// ── ادمین — ارسال همگانی ─────────────────────────────
	KeyBroadcastAskText:        "📢 <b>ارسال همگانی</b>\n\nمتن پیامی که می‌خواهید به همه کاربران ارسال شود را بنویسید:",
	KeyBroadcastPreview:        "👁 <b>پیش‌نمایش پیام</b>\n\n─────────────\n%s\n─────────────\n\n📤 این پیام به <b>%d</b> کاربر ارسال خواهد شد.\n\nتأیید می‌کنید؟",
	KeyBroadcastDone:           "✅ <b>ارسال همگانی تمام شد.</b>\n\n📤 ارسال شده: %d\n❌ خطا: %d",
	KeyBroadcastStarted:        "🚀 <b>ارسال همگانی شروع شد.</b>\n\nپس از اتمام، گزارش نتیجه برایتان ارسال می‌شود.",
	KeyBroadcastConfirm:        "✅ تأیید و ارسال",
	KeyBroadcastForwardAsk:     "🔄 پیامی که می‌خوای برای همه فوروارد بشه رو همین‌جا بفرست یا فوروارد کن (متن، عکس، ویدیو، فایل — هرچی):",
	KeyBroadcastForwardPreview: "🔄 این پیام به‌صورت فوروارد برای <b>%d</b> کاربر ارسال خواهد شد.\n\nتأیید می‌کنی؟",
	KeyBroadcastEmptyAudience:  "🤷 با این فیلتر هیچ کاربری پیدا نشد — چیزی ارسال نشد.",
	KeyBcFilterTitle:           "🎯 <b>ارسالِ فیلترشده</b>\n\nپیام به چه کسانی ارسال بشه؟",
	KeyBcFilterAll:             "👥 همه‌ی کاربران",
	KeyBcFilterNoPlan:          "🆓 بدون پلن فعال",
	KeyBcFilterPlan:            "💎 کاربرانِ پلن «%s»",

	// ── ادمین — سیستم ────────────────────────────────────
	KeyAdminSysInfo: `⚙️ <b>وضعیت سیستم</b>

🤖 پلن‌ها: %d
🖥 سرورها: %d (🟢 %d آنلاین)
📦 تمپلیت‌ها: %d
👥 کاربران: %d`,

	// ── کدهای پروموشن ─────────────────────────────────────
	KeyMenuPromoCodes:     "🎁 کدهای پروموشن",
	KeyBtnAddPromo:        "➕ کد جدید",
	KeyPromoAsk:           "🎁 کد پروموشنت رو بفرست تا اعتبارش رو برات فعال کنیم:",
	KeyPromoNotFound:      "❌ همچین کدی پیدا نشد. حروف رو دوباره چک کن.",
	KeyPromoAlreadyUsed:   "🤏 این کد رو قبلاً استفاده کرده‌ای — هر کد فقط یک‌بار برای هر کاربر قابل استفاده‌ست.",
	KeyPromoExpiredOrFull: "⌛️ این کد یا منقضی شده یا ظرفیتش تمام شده.",
	KeyPromoCreditFailed:  "😥 کد <code>%s</code> ثبت شد ولی شارژ کیف پول با خطا مواجه شد. با پشتیبانی تماس بگیر و همین کد رو بگو تا دستی رسیدگی بشه.",
	KeyPromoRedeemed:      "🎉 قبول شد! <b>%.2f TON</b> به اعتبار کیف پولت اضافه شد.",
	KeyPromoAdminTitle:    "🎁 <b>کدهای پروموشن</b>",
	KeyPromoAdminEmpty:    "هنوز هیچ کدی ساخته نشده.",
	KeyPromoAskCode:       "🔤 متنِ کد رو بنویس (مثلاً WELCOME50):",
	KeyPromoAskAmount:     "💰 چند TON اعتبار به هر استفاده‌کننده داده بشه؟",
	KeyPromoAskMaxUses:    "🔢 حداکثر چند نفر می‌تونن استفاده کنن؟ (۰ = نامحدود)",
	KeyPromoAskDays:       "📅 چند روز دیگه منقضی بشه؟ (۰ = بدون انقضا)",
	KeyPromoCreateError:   "❌ ساخت کد با خطا مواجه شد — مقادیر رو چک کن.",
	KeyPromoDuplicate:     "❌ کدی با همین متن قبلاً ساخته شده.",
	KeyPromoCreated:       "✅ <b>کد ساخته شد.</b>\n\n🔤 کد: <code>%s</code>\n💰 اعتبار: %.2f TON\n🔢 سقف استفاده: %d\n📅 انقضا: %d روز دیگه",
	KeyPromoDeleteConfirm: "آیا از حذف این کد مطمئنی؟",
	KeyPromoDeleted:       "🗑 کد حذف شد.",

	// ── ادمین — source-service worker ────────────────────
	KeyMenuSourceWorkers:  "🛰 Source Workerها",
	KeyBtnAddSourceWorker: "➕ افزودن Worker",
	KeyBtnDeleteSW:        "🗑 حذف",
	KeyBtnToggleSW:        "🔁 فعال/غیرفعال",
	KeySWTitle:            "🛰 <b>Source-service Workerها</b>\n\nهر ردیف یک لایسنس/اکانت تلگرام است که یک worker با آن source.worker.register می‌زند.",
	KeySWEmpty:            "هنوز هیچ workerـی ثبت نشده.",
	KeySWAskAppID:         "🔢 <code>app_id</code> اپلیکیشن تلگرام (از my.telegram.org) را وارد کنید:",
	KeySWAskAppHash:       "🔑 <code>app_hash</code> را وارد کنید:",
	KeySWAskPhone:         "📱 شماره تلفن اکانت را وارد کنید (با کد کشور، مثلاً +989123456789):",
	KeySWAskLabel:         "📝 یک برچسب برای شناسایی این worker بنویسید (اختیاری — برای رد شدن «-» بفرستید):",
	KeySWInvalidAppID:     "❌ app_id باید یک عدد باشد. دوباره وارد کنید:",
	KeySWCreated:          "✅ <b>Worker ساخته شد.</b>\n\nاین اطلاعات را در تنظیمات همان source-service worker وارد کنید (فقط همین یک‌بار کامل نمایش داده می‌شود):\n\n🏷 برچسب: %s\n🆔 Worker ID: <code>%s</code>\n🔑 License Key: <code>%s</code>",
	KeySWCreateError:      "❌ ساخت worker با خطا مواجه شد.",
	KeySWDeleted:          "🗑 worker حذف شد.",
	KeySWToggledOn:        "✅ worker فعال شد.",
	KeySWToggledOff:       "⛔️ worker غیرفعال شد.",
	KeySWNotFound:         "❌ worker یافت نشد.",
	KeySWDeleteConfirm:    "آیا از حذف این worker مطمئنید؟ (اگر هنوز فعال است، register/heartbeatهای بعدی‌اش رد خواهد شد.)",

	// ── wizard — تنظیمات اختصاصی ─────────────────────────
	KeyWizardConfigField: "⚙️ <b>تنظیمات اختصاصی (%d از %d)</b>\n\n📝 <b>%s</b>\n📌 پیش‌فرض: <code>%s</code>\n\nمقدار دلخواه را ارسال کنید یا /skip برای استفاده از مقدار پیش‌فرض:",
	KeyWizardConfigDone:  "✅ تنظیمات ثبت شد — ربات دارد ساخته می‌شود...",

	// ── ادمین — ConfigSchema قالب ─────────────────────────
	KeyTmplAskSchema:     "📋 آرایه‌ی JSON فیلدهای قابل‌تنظیم این قالب را بفرست:\n\n<code>[{\"key\":\"CHANNEL_ID\",\"label\":\"آیدی کانال\",\"default\":\"0\",\"required\":false}]</code>\n\nبرای حذف کامل schema فقط <code>[]</code> بفرست.",
	KeyTmplSchemaSet:     "✅ Schema قالب ذخیره شد.",
	KeyTmplSchemaInvalid: "❌ JSON نامعتبر است — دوباره امتحان کنید.",
	KeyBtnEditSchema:     "⚙️ فیلدها",
}
