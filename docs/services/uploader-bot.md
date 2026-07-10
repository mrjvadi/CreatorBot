# uploader-bot

## این سرویس چیست
کامل‌ترین ربات محصول پلتفرم — فروش/توزیع فایل با کد دریافت. طبق CLAUDE.md پروژه حدود ۲۸ قابلیت دارد: قفل کانال، رمز عبور، آلبوم/پوشه، اشتراک پولی، حذف خودکار ضدفیلتر، لایک/دیس‌لایک و گزارش، بکاپ، چند ادمین با سطح دسترسی.

## مسئولیت‌ها
- تحویل فایل با کد (`internal/tgbot/user_uploads.go`, `delivery_gate.go`) — شامل چک عضویت اجباری، رمز، محدودیت دانلود.
- پنل کامل مدیریت (`internal/tgbot/admin_menu.go`, `admin_panel.go`, `admin_perms.go`) — کدها، پوشه‌ها، تنظیمات، چند ادمین با پرمیشن جدا.
- «قفل‌ها» (`locks_panel.go`) — قفل کانال/گروه/لینک/**ربات دوم**. قفل نوع «ربات» یک توکن تلگرام جدا از ادمین می‌گیرد که برای چک عضویت استفاده می‌شود.
- broadcast/ارسال همگانی با صف پس‌زمینه و پاکسازی خودکار پیام‌ها (`broadcast.go`).
- بکاپ/ریستور (`backup_file.go`)، آمار، بازخورد (لایک/دیس‌لایک/گزارش).
- جستجو در کدها (`internal/store/code.go`'s `SearchCodes`) با regex Mongo — هم از منوی جستجوی داخل ربات، هم از inline query عمومی تلگرام.

## ارتباطات
- تنها ربات (به‌همراه `admanager-bot`) که از `shared-core/engine` استفاده می‌کند — یعنی اتصال مستقیم Postgres (فیلتر bot_id) + MongoDB (فیلتر instance_id) + Redis + heartbeat/license-loop روی NATS به‌صورت خودکار.
- داده‌ی اصلی (کد، فایل، پوشه، تنظیمات) در MongoDB؛ instance_id از توکن استخراج می‌شود.

## ایرادها و نکات
- **بررسی و تأیید شد (نه ایراد)**: یک عامل تحقیقاتی قبلی ادعا کرده بود `onCallback` در `internal/tgbot/callbacks.go` هیچ چک ادمینی ندارد. غلط بود — کد از قبل یک whitelist صریح (`publicActions`: `check_join`, `gate`, `sub_buy`, `sub_pay`, `pay_verify`, `folder_open`, `code_resend`, `react_like/dislike`, `report`, `noop`) دارد و هر اکشن دیگری deny-by-default پشت `h.isAdmin(c)` است. نیازی به تغییر نیست.
- **رفع شد (۲۰۲۶-۰۷-۰۲)**: توکن قفل نوع «ربات» (`ForceJoinChannel.BotToken`) برخلاف هر توکن دیگری در کل پروژه، بدون رمزنگاری در MongoDB ذخیره می‌شد. رفع شد: یک `EncryptKey` تا این ربات رشته‌کشی شد (از env تا `core.App`) و توکن قبل از ذخیره با AES-256-GCM رمزنگاری می‌شود؛ اگر کلید تنظیم نشده باشد ذخیره رد می‌شود (fail-closed، نه fallback به متن‌خام).
- **باقی‌مانده، رفع نشد**: قفل رمزدار فایل (`internal/tgbot/misc_features.go`'s `userCheckPassword`) هیچ محدودیت تلاش ندارد — یک رمز عددی کوتاه در عرض چند دقیقه با تلاش‌های پیاپی قابل حدس زدن است. نیاز به rate-limit/lockout روی (uid, code_id) در Redis دارد.
- **باقی‌مانده، رفع نشد**: `internal/store/code.go`'s `SearchCodes` یک `primitive.Regex{Pattern: query}` مستقیم از ورودی کاربر می‌سازد، بدون escape کاراکترهای متا — هم از جستجوی داخل ربات، هم از inline query (که پیش از هرگونه احراز هویت/عضویت قابل‌دسترس است) قابل رسیدن است. یک regex بدخیم (nested quantifiers) می‌تواند CPU سرور Mongo مشترک را مصرف کند (ReDoS با هزینه‌ی کم برای مهاجم).
