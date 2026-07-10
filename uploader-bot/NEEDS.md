# uploader-bot — چه چیزی نیاز داریم (بررسی ۲۰۲۶-۰۷-۰۴)

## وضعیت فعلی (خلاصه‌ی واقعی، نه قدیمی)

بررسی تازه انجام شد (نه بر اساس مستندات قبلی). سرویس شامل ۶۲ فایل Go است:
`internal/models` (۶ فایل)، `internal/store` (۱۸ فایل، روی MongoDB)،
`internal/tgbot` (۲۶ فایل)، `internal/payment` (زرین‌پال + زیبال)، `internal/core`
و `internal/util`. این با ادعای CLAUDE.md («۲۸ قابلیت کامل: قفل کانال، رمز،
آلبوم، اشتراک، حذف خودکار، گزارش/لایک، بکاپ، چند ادمین») همخوانی دارد — این
واقعاً بزرگ‌ترین و کامل‌ترین سرویس از ۴ سرویس بررسی‌شده است.

نکات مثبت مهم که در کد پیدا شد:
- گیت امنیتی مرکزی deny-by-default در `internal/tgbot/callbacks.go:38-53` —
  هر callback که در `publicActions` نباشد باید `h.isAdmin(c)` را رد کند.
  این الگوی خوبی است و اکثر مسیرهای ادمین را پوشش می‌دهد.
- سیستم دسترسی دانه‌ریز (`PermUpload`, `PermUsers`, ...) در
  `internal/models/user.go:40-61` و `internal/tgbot/admin_perms.go` پیاده‌سازی
  شده — چند سطح ادمین با دسترسی جزئی واقعاً کار می‌کند، نه فقط یک owner ساده.
- کامنت صادقانه در `internal/store/admin.go:52-57` نشان می‌دهد یک باگ واقعی
  (تفسیر نادرست خطای Mongo به عنوان «ادمین بدون دسترسی») قبلاً پیدا و رفع شده.
- هیچ TODO/FIXME/HACK در کل سرویس وجود ندارد (grep تمیز).

نکته‌ی منفی: **صفر فایل تست** (`find uploader-bot -name "*_test.go"` چیزی
برنمی‌گرداند) — با وجود ۶۲ فایل و منطق پولی (زرین‌پال/زیبال)، هیچ تست
واحدی نوشته نشده.

## چیزی که واقعاً کم است (با file:line برای هر ادعا)

1. **افزایش دسترسی ادمین (privilege escalation) — شکاف امنیتی واقعی.**
   `internal/tgbot/callbacks.go:91-98` مسیرهای `aperm` (منوی دسترسی)،
   `aperm_t` (toggle دسترسی)، `admin_add`، `admin_del` را فقط پشت گیت عمومی
   `h.isAdmin(c)` (خط ۵۱) قرار می‌دهد — نه پشت `adminCan(ctx, c, models.PermAdmins)`.
   در مقابل، مسیر UI معتبر (`internal/tgbot/admin_menu.go:237`,
   `panelSection`) قبل از نمایش منوی «ادمین‌ها» چک می‌کند
   `sectionPerm("admins") == PermAdmins` و `adminCan` را صدا می‌زند.
   نتیجه: **هر ادمینی با فقط یک دسترسی جزئی (مثلاً فقط `PermUpload`) می‌تواند
   با ساختن دستی callback data به شکل `aperm_t:<targetTelegramID>:admins`
   (بدون نیاز به عبور از منوی admins) به خودش یا هر ادمین دیگری دسترسی
   `PermAdmins` بدهد** — چون تنها چک واقعی در مسیر پردازش callback همان
   `isAdmin` عمومی (`bot.go:365-371`) است، نه چک دسترسی دانه‌ریز.
   پیشنهاد: در `adminTogglePerm`/`adminPermsMenu`/`adminAskAdmin`/
   `adminRemoveAdmin` (تعریف در `internal/tgbot/admin_perms.go:77,105` و
   `internal/tgbot/handlers.go:898,904`) قبل از اجرا `adminCan(ctx, c,
   models.PermAdmins)` را هم چک کنید، نه فقط `isAdmin`.

2. **صفر فایل تست در کل سرویس.** با وجود منطق پرداخت آنلاین
   (`internal/payment/zarinpal.go`, `internal/payment/zibal.go`) و منطق
   کیف‌پول/اشتراک (`internal/tgbot/pay_online.go`)، هیچ `_test.go` وجود ندارد.
   حداقل تست‌های واحد برای `models.Admin.Has` (`internal/models/user.go:52`)
   و `sectionPerm`/`adminCan` (`internal/tgbot/admin_perms.go:13,27`) به‌خاطر
   حساسیت امنیتی بند ۱ توصیه می‌شود.

3. **NATS surface این سرویس فقط تنظیمات عمومی کلید/مقدار است، نه CRUD
   واقعی.** `internal/tgbot/nats_config.go:34-41` فقط به
   `uploader.settings.<botID>` و `config.updated` گوش می‌دهد — یعنی از راه دور
   فقط می‌شود یک `key=value` ساده تزریق کرد، نه لیست/ساخت/حذف کد، پوشه،
   قفل کانال یا بکاپ. جزئیات کامل در بخش بعد.

## این‌ها را در سرویس‌های دیگر هم نوشتم (اگر مرتبط بود)

یک یادداشت متقابل به `apimanager/NEEDS.md` اضافه شد: مدل‌های غنی این سرویس
(`internal/models/models.go` → `Code`, `Folder`, `ForceJoinChannel`,
`Backup`؛ تعریف در `internal/store/code.go`, `folder.go`, `channel.go`,
`backup.go`) کاندیدای خوبی برای CRUD واقعی apimanager هستند، نه فقط
تنظیمات کلید/مقدار فعلی. جزئیات آنجا نوشته شده تا صاحب apimanager تصمیم
بگیرد.
