package i18n

var fa = map[Key]string{
	// ── عمومی ─────────────────────────────────────────────
	KeyCancel:    "❌ انصراف",
	KeyCancelled: "لغو شد.",
	KeyBack:      "🔙 بازگشت",
	KeyConfirm:   "✅ تأیید",
	KeyError:     "خطایی رخ داد. لطفاً دوباره تلاش کنید.",
	KeyNotFound:  "مورد درخواستی یافت نشد.",
	KeySaved:     "✅ با موفقیت ذخیره شد.",
	KeyDeleted:   "🗑 حذف شد.",

	// ── start / welcome ───────────────────────────────────
	KeyWelcomeAdmin: "سلام %s\nخوش اومدی به پنل مدیریت CreatorBot 👑",
	KeyWelcomeUser:  "سلام %s 👋\nبا CreatorBot می‌تونی ربات تلگرام اختصاصی بسازی.",

	// ── زبان ──────────────────────────────────────────────
	KeySelectLang:  "زبان خود را انتخاب کنید:\nSelect your language:",
	KeyLangChanged: "✅ زبان به فارسی تغییر کرد.",

	// ── منوی ادمین ────────────────────────────────────────
	KeyMenuBots:      "🤖 ربات‌ها",
	KeyMenuLinks:     "🔗 لینک‌های دعوت",
	KeyMenuServers:   "🖥 سرورها",
	KeyMenuTemplates: "📦 تمپلیت‌ها",
	KeyMenuPlans:     "💰 پلن‌ها",
	KeyMenuUsers:     "👥 کاربران",
	KeyMenuStats:     "📊 آمار",

	// ── منوی کاربر ────────────────────────────────────────
	KeyMenuMyBots:  "🤖 ربات‌های من",
	KeyMenuSupport: "📞 پشتیبانی",
	KeyMenuHelp:    "❓ راهنما",

	// ── سرور ──────────────────────────────────────────────
	KeyServersTitle:    "<b>🖥 سرورها</b>",
	KeyServersEmpty:    "هیچ سروری ثبت نشده.",
	KeyServerAskName:   "نام سرور را وارد کنید:\nمثال: <code>server-de1</code>",
	KeyServerAskIP:     "آدرس IP سرور را وارد کنید:\nمثال: <code>1.2.3.4</code>",
	KeyServerAdded:     "✅ <b>سرور ثبت شد</b>\n\nنام: %s\nIP: <code>%s</code>\nID: <code>%s</code>",
	KeyServerDuplicate: "این IP قبلاً ثبت شده است.",
	KeyServerAddError:  "خطا در ثبت سرور.",

	// ── تمپلیت ────────────────────────────────────────────
	KeyTemplatesTitle:   "<b>📦 تمپلیت‌ها</b>",
	KeyTemplatesEmpty:   "هیچ تمپلیتی وجود ندارد.",
	KeyTemplateAskType:  "نوع ربات را انتخاب کنید:",
	KeyTemplateAskImage: "نام image Docker را وارد کنید:\nمثال: <code>registry.io/mybot</code>",
	KeyTemplateAskTag:   "تگ image را وارد کنید:\nمثال: <code>latest</code> یا <code>v1.2.0</code>",
	KeyTemplateAskName:  "یک نام برای این تمپلیت وارد کنید:\nمثال: <code>uploader-v2</code>",
	KeyTemplateAdded:    "✅ <b>تمپلیت ثبت شد</b>\n\nنام: <b>%s</b>\nنوع: %s\nImage: <code>%s:%s</code>\nID: <code>%s</code>",
	KeyTemplateAddError: "خطا در ثبت تمپلیت.",

	// ── پلن ───────────────────────────────────────────────
	KeyPlansTitle:        "<b>💰 پلن‌ها</b>",
	KeyPlansEmpty:        "هیچ پلنی وجود ندارد.",
	KeyPlansNoTemplate:   "⚠️ ابتدا باید یک تمپلیت اضافه کنید.",
	KeyPlanAskTemplate:   "برای افزودن پلن جدید، ID تمپلیت را ارسال کنید:",
	KeyPlanTmplNotFound:  "تمپلیت یافت نشد. ID را دوباره بررسی کنید.",
	KeyPlanAskName:       "نام پلن را وارد کنید:\nمثال: ماهانه",
	KeyPlanAskDays:       "مدت پلن را به روز وارد کنید:\nمثال: <b>30</b>",
	KeyPlanAskPrice:      "قیمت پلن را به تومان وارد کنید:\nمثال: <b>150000</b>",
	KeyPlanInvalidNumber: "عدد معتبر وارد کنید.",
	KeyPlanAdded:         "✅ <b>پلن ثبت شد</b>\n\nنام: <b>%s</b>\nتمپلیت: %s\nمدت: %d روز\nقیمت: <b>%.0f تومان</b>\nID: <code>%s</code>",
	KeyPlanAddError:      "خطا در ثبت پلن.",

	// ── لینک دعوت ─────────────────────────────────────────
	KeyLinksTitle:      "<b>🔗 لینک‌های دعوت</b>\n\nبا این لینک‌ها کاربران می‌توانند ربات بسازند.",
	KeyLinksEmpty:      "هیچ لینکی ندارید.",
	KeyLinkAskType:     "برای ساخت لینک جدید، نوع ربات را انتخاب کنید:",
	KeyLinkAskLimit:    "محدودیت استفاده از لینک را انتخاب کنید:",
	KeyLinkAskLabel:    "یک یادداشت خصوصی بنویسید (مثلاً: «برای علی»)\nاین یادداشت فقط برای شما نمایش داده می‌شود.\n\nبرای بدون یادداشت عدد <b>0</b> بزنید.",
	KeyLinkCreated:     "✅ <b>لینک دعوت ساخته شد</b>\n\nنوع: %s %s%s\nمحدودیت: %s\n\n🔗 لینک:\n<code>%s</code>\n\nاین لینک را برای کاربر مورد نظر ارسال کنید.",
	KeyLinkCreateError: "خطا در ساخت لینک.",

	// ── ربات‌ها (ادمین) ────────────────────────────────────
	KeyBotsTitle:    "<b>🤖 همه ربات‌ها (%d)</b>",
	KeyBotsEmpty:    "هیچ ربات فعالی وجود ندارد.\n\nاز «🔗 لینک‌های دعوت» یک لینک بسازید و به کاربر بدهید.",
	KeyBotStopped:   "⏹ ربات <code>%s</code> متوقف شد.",
	KeyBotStarted:   "▶️ دستور راه‌اندازی برای <code>%s</code> ارسال شد.",
	KeyBotDeleted:   "🗑 ربات <b>%s</b> حذف شد.",
	KeyBotNotFound:  "ربات یافت نشد.",

	// ── کاربران ───────────────────────────────────────────
	KeyUsersTitle:   "<b>👥 کاربران (%d)</b>",
	KeyUsersEmpty:   "هیچ کاربری ثبت‌نام نکرده است.",
	KeyUserBlocked:  "🚫 کاربر بلاک شد.",
	KeyUserUnblocked: "✅ کاربر آنبلاک شد.",
	KeyUserMadeAdmin: "🛡 کاربر به Admin تبدیل شد.",
	KeyUserMadeUser:  "👤 نقش کاربر به User تغییر کرد.",

	// ── آمار ──────────────────────────────────────────────
	KeyStatsTitle: "<b>📊 آمار سیستم</b>",

	// ── ربات‌های کاربر ────────────────────────────────────
	KeyMyBotsTitle: "<b>🤖 ربات‌های شما (%d)</b>",
	KeyMyBotsEmpty: "<b>🤖 ربات‌های شما</b>\n\nهنوز هیچ ربات فعالی ندارید.\n\nبرای خرید یا دریافت لینک ساخت ربات با پشتیبانی تماس بگیرید.",
	KeySupportText: "<b>📞 پشتیبانی</b>\n\nبرای ارتباط با تیم پشتیبانی:\n@support_username\n\nساعت پاسخگویی: ۹ الی ۲۱",
	KeyHelpText: "<b>❓ راهنما</b>\n\n<b>ربات‌ها</b> — مدیریت همه ربات‌ها\n<b>لینک‌ها</b> — ساخت لینک دعوت\n<b>سرورها</b> — ثبت سرور جدید\n<b>تمپلیت‌ها</b> — image های Docker\n<b>پلن‌ها</b> — پلن‌های خرید\n<b>کاربران</b> — مدیریت کاربران\n<b>آمار</b> — آمار سیستم\n\nبرای لغو: /cancel\n\n<b>کاربران:</b>\nبا CreatorBot می‌توانید ربات‌های تلگرام اختصاصی بسازید:\n\n📤 <b>Uploader</b> — ارسال فایل با کد\n🔒 <b>VPN</b> — فروش اشتراک VPN\n📂 <b>Archive</b> — آرشیو و جستجوی فایل\n👥 <b>Member</b> — قفل ممبر کانال\n\nبرای خرید با پشتیبانی تماس بگیرید.",

	// ── wizard ────────────────────────────────────────────
	KeyWizardInvalidLink:   "❌ این لینک معتبر نیست.",
	KeyWizardExpiredLink:   "❌ این لینک منقضی شده است.",
	KeyWizardUsedLink:      "❌ این لینک قبلاً استفاده شده است.",
	KeyWizardConfirm:       "<b>🔗 لینک دعوت معتبر</b>\n\n%s <b>%s Bot</b>\n\n%s\n\nآیا می‌خواهید ادامه دهید؟",
	KeyWizardAskToken:      "توکن ربات را از @BotFather دریافت کرده و اینجا ارسال کنید.\n\n⚠️ توکن را با کسی به اشتراک نگذارید.",
	KeyWizardInvalidToken:  "❌ فرمت توکن نامعتبر است.\n\nتوکن باید از @BotFather باشد و شبیه این باشد:\n<code>123456789:AABB...</code>",
	KeyWizardAlreadyExists: "⚠️ این ربات قبلاً ثبت شده است.\n\nID: <code>%s</code>\nوضعیت: %s",
	KeyWizardNoServer:      "⚠️ متأسفانه در حال حاضر هیچ سروری در دسترس نیست.\nلطفاً بعداً دوباره تلاش کنید یا با پشتیبانی تماس بگیرید.",
	KeyWizardNoTemplate:    "⚠️ تمپلیت این نوع ربات تنظیم نشده است.\nبا پشتیبانی تماس بگیرید.",
	KeyWizardDeployError:   "⚠️ <b>ربات ثبت شد ولی deploy ناموفق بود</b>\n\nادمین به‌زودی بررسی خواهد کرد.\nID: <code>%s</code>",
	KeyWizardSuccess:       "🎉 <b>ربات شما ساخته شد!</b>\n\n%s <b>%s Bot</b>\nسرور: %s\nوضعیت: 🟡 در حال راه‌اندازی\n\nمعمولاً ظرف ۱-۲ دقیقه فعال می‌شود.\n\nبرای مشاهده وضعیت: <b>🤖 ربات‌های من</b>",

	// ── نوع ربات ──────────────────────────────────────────
	KeyBotTypeUploader: "📤 Uploader",
	KeyBotTypeVPN:      "🔒 VPN",
	KeyBotTypeArchive:  "📂 Archive",
	KeyBotTypeMember:   "👥 Member",

	KeyBotDescUploader: "ربات آپلودر فایل — با کد فایل بفرست",
	KeyBotDescVPN:      "ربات فروش VPN — اشتراک بفروش",
	KeyBotDescArchive:  "ربات آرشیو فایل — جستجو و دسته‌بندی",
	KeyBotDescMember:   "ربات قفل ممبر — چک عضویت کانال",


	// ── آمار ادمین ───────────────────────────────────────────
	KeyStatsBotsLine:    "🤖 ربات‌ها (%d جمع)\n🟢 فعال: %d  🔴 متوقف: %d  🟡 راه‌اندازی: %d  ⚠️ خطا: %d",
	KeyStatsServersLine: "🖥 سرورها (%d جمع)\n🟢 آنلاین: %d  🔴 آفلاین: %d",
	KeyStatsUsersLine:   "👥 کاربران (%d جمع)\n🛡 ادمین: %d  🚫 بلاک: %d",

	// ── پلن‌ها ────────────────────────────────────────────────
	KeyPlansAvailable:    "<b>💎 پلن‌های موجود</b>",
	KeyPlansFree:         "🆓 رایگان",
	KeyPlansDays:         "%d روز",
	KeyPlansEternal:      "ابدی",
	KeyPlansSelectPrompt: "ID پلن مورد نظر را ارسال کنید:",

	// ── کیف پول ──────────────────────────────────────────────
	KeyBalanceLine:  "💳 موجودی: <b>%.4f TON</b>",
	KeyCreditLine:   " (🎁 %.4f اعتبار)",
	KeyPlanLine:     "📋 پلن: <b>%s</b>\n🤖 %d/%d ربات\n%s",
	KeyExpiredSub:   "❌ منقضی",
	KeyEternalSub:   "♾ ابدی",
	KeyDaysLeft:     "⏰ %d روز مانده",

	// ── خرید پلن ─────────────────────────────────────────────
	KeyNoPlans:         "هیچ پلنی موجود نیست.",
	KeyBuyConfirm:      "<b>تأیید خرید</b>\n\n📋 پلن: <b>%s</b>\n💰 قیمت: <b>%.2f TON</b>\n💳 موجودی شما: %.4f TON\n\nآیا تأیید می‌کنید؟",
	KeyBuySuccess:      "✅ <b>پلن %s فعال شد!</b>\n\n🤖 %d ربات\nحالا می‌توانید ربات بسازید.",
	KeyInsufficientBal: "❌ موجودی کافی نیست.",
	KeyNeedDeposit:     "<b>💎 خرید پلن %s</b>\n\n💰 قیمت: %.2f TON\n💳 موجودی شما: %.4f TON\n📥 نیاز به واریز: <b>%.4f TON</b>\n\nکد: <code>%s</code>\n\n۱. روی «واریز» کلیک کنید\n۲. مبلغ را پرداخت کنید\n۳. «واریز کردم» بزنید",
	KeyDepositDone:     "✅ <b>پرداخت تأیید شد!</b>\n\n📋 پلن: %s\n🤖 %d ربات\nحالا می‌توانید ربات بسازید.",
	KeyDepositPending:  "⏳ موجودی هنوز کافی نیست.\n\n💳 موجودی: %.4f TON\n💰 نیاز: %.2f TON\n\nچند دقیقه صبر کنید و دوباره امتحان کنید.",
	KeySubExists:       "شما اشتراک فعال دارید.",
	KeyFreePlanActive:  "🎉 <b>پلن رایگان فعال شد!</b>\n\n📋 %s\n🤖 %d ربات\n⏳ %s\n\nحالا می‌توانید ربات بسازید.",
	KeyCapacityFull:    "به حداکثر ربات رسیده‌اید (%d/%d).\n\n💎 برای ربات بیشتر، پلن بهتری بخرید.",
	KeyNoPlan:          "برای ساخت ربات باید پلن خریداری کنید.",

	// ── بلاک ─────────────────────────────────────────────────
	KeyBlocked: "⛔️ دسترسی شما محدود شده است.",


	// ── ادمین — پلن ──────────────────────────────────────────
	KeyAdminPlanLine:   "• <b>%s</b>%s — %d روز — <b>%.2f TON</b> — %d ربات\n  ID: <code>%s</code>",
	KeyAdminPlanFree:   " 🆓",
	KeyAdminPlanAdded:  "✅ <b>پلن ثبت شد</b>\n\nنام: <b>%s</b>%s\nتمپلیت: %s\nمدت: %d روز\nقیمت: <b>%.2f TON</b>\nحداکثر ربات: %d\nID: <code>%s</code>",
	KeyAdminTemplates:  "<b>تمپلیت‌ها:</b>",

	// ── ادمین — ربات‌ها ────────────────────────────────────────
	KeyAdminBotSummary:  "🟢 %d  🔴 %d  🟡 %d  ⚠️ %d",
	KeyAdminLinkStats:   "✅ فعال: %d  |  ❌ منقضی: %d",
	KeyAdminLinkLimitX:  "%d×",

	// ── ادمین — کاربران ───────────────────────────────────────
	KeyAdminUserSummary: "👑 %d  🛡 %d  👤 %d  🚫 %d",
	KeyAdminUserDetail:  "<b>👤 %s</b>%s\nTID: <code>%d</code>\nنقش: %s\nبلاک: %s\nربات‌ها: %d",
	KeyAdminUserBlocked: "🚫",

	// ── راهنمای ساخت ربات ────────────────────────────────────
	KeyHowToBuild: "<b>🔗 چطور ربات بسازم؟</b>\n\n" +
		"۱. به ادمین پیام بدید و بگید می‌خواید ربات بسازید\n" +
		"۲. ادمین یک لینک دعوت برایتان می‌فرستد\n" +
		"۳. لینک را باز کنید\n" +
		"۴. از @BotFather یک ربات بسازید و توکن را اینجا بفرستید\n" +
		"۵. تمام! ربات شما در عرض ۲ دقیقه آماده است.",
	KeyHowToBuildDone: "✅ متوجه شدم",
	KeyNoFreePlan:     "پلن رایگان موجود نیست.",

	// ── تمپلیت رایگان ────────────────────────────────────────
	KeyTmplFreeAdded:  "✅ <b>تمپلیت رایگان ثبت شد</b>\n\nنام: <b>%s</b>\nنوع: %s\nImage: <code>%s:%s</code>\nID: <code>%s</code>\n\nحالا می‌توانید پلن رایگان بسازید.",
	KeyTmplFreeExists: "⚠️ یک تمپلیت رایگان از نوع <b>%s</b> قبلاً وجود دارد.\nID: <code>%s</code>",


	// ── وضعیت اشتراک بدون ربات ───────────────────────────────
	KeySubActiveNoBot:   "🎉 <b>اشتراک %s فعال است</b>\n\nهنوز ربات نساخته‌اید.\n\nبرای ساخت ربات، از ادمین یک <b>لینک دعوت</b> دریافت کنید.\nلینک را باز کنید و مراحل را دنبال کنید.",
	KeyBuildWithLink:    "🔗 ساخت ربات با لینک دعوت",

	// ── دکمه‌ها ───────────────────────────────────────────
	KeyBtnYesBuild:  "✅ بله، بساز",
	KeyBtnCancel:    "❌ انصراف",
	KeyBtnBack:      "🔙 بازگشت",
	KeyBtnLimit1:    "1️⃣  یک‌بار",
	KeyBtnLimit3:    "3️⃣  سه‌بار",
	KeyBtnLimit5:    "5️⃣  پنج‌بار",
	KeyBtnLimit10:   "🔟 ده‌بار",
	KeyBtnLimitNo:   "♾  نامحدود",
	KeyBtnBlock:     "🚫 بلاک",
	KeyBtnUnblock:   "✅ آنبلاک",
	KeyBtnMakeAdmin: "🛡 تبدیل به Admin",
	KeyBtnMakeUser:  "👤 تبدیل به User",
}
