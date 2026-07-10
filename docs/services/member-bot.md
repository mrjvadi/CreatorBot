# member-bot

## این سرویس چیست
زیرساخت داخلی پلتفرم برای چک متمرکز عضویت کانال — دیگر مستقیم توسط کاربر نهایی استفاده نمی‌شود (دستورات قدیمی‌اش مثل `/mylocks`/`/newlock` الان فقط یک پیام راهنما نشان می‌دهند و کاربر را به `ads-bot`/`/rentlock` ارجاع می‌دهند). هدف اصلی‌اش این است که بقیه‌ی ربات‌ها مجبور نباشند خودشان در هر کانالی ادمین شوند؛ به‌جایش از این سرویس می‌پرسند «کاربر X عضو کانال Y هست؟».

## مسئولیت‌ها
- `member.check` (NATS request/reply) — پاسخ به سؤال عضویت، با کش.
- استخر «check bot» ها (`CheckBot` — توکن AES-256-GCM رمزنگاری‌شده) که برای واقعاً زدن `getChatMember` استفاده می‌شوند؛ یک `dispatcher`/`balancer` کار را بین آن‌ها پخش می‌کند.
- مدیریت `Lock` (قفل کانال پولی، مربوط به مدل قدیمی‌تر پیش از اینکه اجاره‌ی قفل به `ads-bot` منتقل شود).
- HTTP API داخلی برای قفل (`internal/lock`، پشت `LOCK_API_SECRET`).

## ارتباطات
- `shared-core/engine` استفاده نمی‌شود — Postgres/Redis مستقیم.
- NATS: responder `member.check` (queue `memberbot-workers`)، publish `membership.joined`/`membership.left`.

## ایرادها و نکات
- **بحرانی، رفع شد (۲۰۲۶-۰۷-۰۲)**: `onCallback` در `internal/tgbot/handler.go` هیچ چک مالکیت/ادمین روی `lock_pause`, `lock_delete`, `approve_pay` نداشت. `internal/store/store.go`'s `ExpireLock`/`DeleteLock`/`ApprovePayment` هم فیلتر owner_id/admin نداشتند — هر کاربری که UUID یک قفل را می‌دانست (که در `/mylocks` خودِ مالک درج می‌شود) می‌توانست قفل پولی کاربر دیگری را حذف کند، یا هر پرداخت در انتظاری را خودش «تأیید» کند. رفع شد: یک تابع جدید `canManageLock` (در `internal/tgbot/owner.go`) قبل از توقف/حذف قفل، مالکیت واقعی یا ادمین‌بودن را چک می‌کند؛ `approvePayment` (در `internal/tgbot/admin.go`) هم حالا `isAdmin` را الزامی می‌کند.
- **رفع شد (جانبی)**: اتصال NATS این ربات قبلاً فقط در حالت webhook ساخته می‌شد — یعنی در حالت polling نه `memberresponder` (خودِ `member.check`، یعنی زیرساخت اصلی این سرویس!) نه license check-in کار می‌کردند. الان هر وقت `NATS_URL` تنظیم باشد در هر دو حالت وصل می‌شود.
- **تأیید شد، بدون مشکل**: `dispatcher`/`balancer` توکن رمزگشایی‌شده را فقط در حافظه نگه می‌دارند و هیچ‌جا در لاگ/URL چاپ نمی‌کنند.
- **باقی‌مانده، کم‌خطر**: `handleBotToken` قبل از رمزنگاری و ثبت یک توکن جدید در استخر check-bot، هیچ `getMe` واقعی برای تأیید معتبربودن توکن نمی‌زند (فقط فرمت را چک می‌کند) — یک توکن نامعتبر می‌تواند وارد استخر شود (مشکل در دسترسی‌پذیری، نه محرمانگی).
