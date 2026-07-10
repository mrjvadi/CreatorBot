# طراحی کامل API پلتفرم (apimanager) — پنل ادمین + پنل کاربر برای همه‌ی ربات‌ها

> این فایل یک **سند طراحی** است، نه کد. هدف: apimanager را از یک HTTP gateway نیمه‌کاره (فقط auth+instance
> پایه) به یک API کامل تبدیل کند که هرچه امروز فقط از داخل تلگرام (پنل ادمین botmanager + پنل ادمین/کاربر
> هر ربات محصول) قابل انجام است، از طریق HTTP هم قابل انجام باشد — برای یک پنل وب/اپ آینده.
> پیش‌نیاز خواندن: `apimanager/PROJECT_UNDERSTANDING.md` (نقشه‌ی کامل معماری + قابلیت‌های هر سرویس).

---

## ۱. سه سطح دسترسی (Persona)

همه‌ی endpoint های زیر باید دقیقاً در یکی از این سه سطح قرار بگیرند — این مهم‌ترین تصمیم طراحی است:

| سطح | یعنی کی | مثال در تلگرام امروز | توکن |
|---|---|---|---|
| **Platform Admin** | ادمین/owner خودِ پلتفرم (نقش `admin`/`owner` در `botmanager`) | پنل ادمین botmanager: کاربران، پلن‌ها، سرورها، تمپلیت‌ها | JWT با `role=admin/owner` (همان `middleware.RequireRole` موجود) |
| **Bot Owner** | مشتری‌ای که یک `BotInstance` خریده (مثلاً یک uploader-bot دارد) | پنل ادمین همان ربات (مثلاً منوی مدیریت uploader-bot خودش) | JWT کاربر + مالکیت instance چک شود (`instance.OwnerID == user.ID`) |
| **End Customer** | کسی که از رباتِ یک Bot Owner استفاده می‌کند (مثلاً کسی که از یک vpn-bot خاص VPN می‌خرد) | منوی کاربر همان ربات (نه ادمینش) | یا JWT (اگر پنل وب دارد) یا یک کد/توکن یک‌بارمصرف مرتبط با تلگرام‌آی‌دی همان کاربر در آن ربات |

نکته‌ی کلیدی: **Bot Owner و End Customer به دیتای همان یک instance محدودند** — یعنی authorization باید همیشه
اول چک کند «آیا این instance واقعاً مال این کاربر است؟» قبل از هر عملیات، دقیقاً مثل چیزی که در ممیزی امنیتی
۲۰۲۶-۰۷-۰۲ برای IDOR ها (`vpn-bot` لینک اشتراک، `member-bot` قفل‌ها) رفع شد — همان اشتباه نباید در API هم تکرار شود.

---

## ۲. مشکل معماری‌ای که باید حل شود قبل از نوشتن کد

طبق قانون بنیادی پروژه (`docs/security-audit-2026-07-02.md`, `apimanager/PROJECT_UNDERSTANDING.md`):
**هیچ سرویسی مستقیم به DB سرویس دیگر کوئری نمی‌زند.**

اما تنظیمات هر ربات (کدها/پوشه‌های uploader-bot، پنل‌های vpn-bot، دسته‌های archive-bot، قفل‌های
member-bot) در دیتابیس **خودِ همان ربات** است (اکثراً MongoDB، با `instance_id` جدا). `apimanager`
امروز فقط به DB مشترک `botmanager` (Postgres: `BotInstance`, `Plan`, ...) دسترسی دارد، نه به DB
داخلی هر ربات.

**راه‌حل**: هر ربات محصول باید یک NATS responder عمومی برای «تنظیمات» اضافه کند (اگر ندارد)، و
`apimanager` هر endpoint HTTP مربوط به تنظیمات یک ربات را به یک NATS request/reply روی همان ربات ترجمه
کند — دقیقاً همان الگویی که `pay.*` و `license.*` استفاده می‌کنند.

subject پیشنهادی (باید در `shared-core/protocol/subjects.go` تعریف شود، مطابق قرارداد پروژه):

```
settings.<bot_type>.<action>          مثال: settings.uploader.code.list
                                              settings.uploader.code.delete
                                              settings.vpn.panel.list
                                              settings.vpn.panel.create
```

هر request باید حداقل این‌ها را حمل کند: `instance_id` (تعیین می‌کند کدام instance)، `requester_role`
(`owner` یا `admin`، تعیین‌شده توسط خودِ apimanager بعد از چک مالکیت — **نه** از کلاینت HTTP گرفته شود)،
و بدنه‌ی مخصوص آن اکشن. پاسخ باید `{success, data, error}` باشد. **auth service-to-service همان الگوی
`ServiceID`+`ServiceKey` (`SERVICE_HMAC_SECRET`) باشد که برای رفع باگ botpay ساخته شد** — یعنی هر ربات
باید مطمئن شود این درخواست واقعاً از `apimanager` آمده، نه از یک کلاینت NATS دلخواه (همان درسی که از
باگ critical کیف‌پول گرفته شد: هرگز فقط به `instance_id`/`service_id` به‌عنوان راز اعتماد نکن).

این یعنی: قبل از پیاده‌سازی endpoint های بخش ۴ (پنل ادمین/کاربر هر ربات)، باید در هر ربات محصول یک
`internal/settingsresponder` (یا مشابه) اضافه شود. این کار مرحله‌به‌مرحله در بخش ۶ (نقشه‌ی راه) آمده.

---

## ۳. Platform Admin API — گسترش چیزی که از قبل هست

این‌ها معادل HTTP همان چیزی هستند که پنل ادمین `botmanager` امروز در تلگرام دارد
(`apimanager/PROJECT_UNDERSTANDING.md` بخش ۳.۲). چیزهایی که با ✅ مشخص شده از قبل وجود دارند؛ بقیه باید اضافه شوند.

### کاربران
- ✅ (باید اضافه شود) `GET /api/v1/admin/users` — لیست با فیلتر/جست‌وجو/صفحه‌بندی
- `GET /api/v1/admin/users/:id` — جزئیات یک کاربر + لیست instance هایش
- `POST /api/v1/admin/users/:id/block` / `POST /api/v1/admin/users/:id/unblock`
- `POST /api/v1/admin/users/:id/role` — تغییر نقش (`user`/`admin`/`owner`)
- `POST /api/v1/admin/users/:id/credit` — افزودن اعتبار دستی (بدنه: `amount_ton, reason`)

### پلن‌ها
- ✅ `GET /admin/... ` (لیست تمپلیت از قبل هست) → باید `GET/POST/PATCH/DELETE /api/v1/admin/plans` و
  `PATCH /api/v1/admin/plans/:id/limits` (معادل دکمه‌های ➕➖ محدودیت هر نوع ربات) اضافه شود.

### سرورها
- ✅ `GET /admin/servers`, `POST /admin/servers` از قبل هست.
- `DELETE /api/v1/admin/servers/:id`, `GET /api/v1/admin/servers/:id/instances` (چه instance هایی رویش هستند).

### تمپلیت‌ها (سرویس/تگ)
- ✅ `GET/POST /admin/templates` از قبل هست.
- `PATCH /api/v1/admin/templates/:id` (فعال/غیرفعال، تغییر image tag)، `DELETE`.
- `POST /api/v1/admin/templates/:id/test-deploy` — معادل تست دیپلوی داخلی ادمین در تلگرام.

### آمار و رصد
- ✅ `GET /admin/stats` از قبل هست — باید شامل آمار مالی (از `botpay` روی NATS)، آمار fraud (از
  `fraud-engine` HTTP)، و آمار لاگ (از `log-collector` HTTP) هم بشود، نه فقط شمارش instance/کاربر.
- `GET /api/v1/admin/logs` — proxy مستقیم به `log-collector`'s `GET /logs` (با همان فیلترها).
- `GET /api/v1/admin/licenses/:bot_id` — proxy به `license.verify`/وضعیت لایسنس (برای دیدن clone warning ها).

### ارسال همگانی
- `POST /api/v1/admin/broadcast` — بدنه: `text, target (all|role|plan)`. باید به یک صف/job تبدیل شود
  (نه synchronous)، مثل چیزی که `botmanager` با صف پس‌زمینه دارد.

---

## ۴. Bot Owner + End Customer API — به تفکیک نوع ربات

این بخش «تنظیماتی که برای هر رباتی هست، ادمینش (owner) و کاربرانش (end customer) بتوانند از پنل انجام
دهند» را دقیقاً پیاده می‌کند. مسیر پایه: `/api/v1/bots/:instance_id/...` — همه‌جا اول چک مالکیت
(`Bot Owner`) یا صرفاً هویت کاربر همان ربات (`End Customer`) انجام می‌شود.

### ۴.۱ uploader-bot
**Bot Owner (مدیریت):**
- `GET/POST/DELETE /api/v1/bots/:id/codes` — لیست/ساخت/حذف کد.
- `PATCH /api/v1/bots/:id/codes/:code_id` — رمز، محدودیت دانلود، قفل فوروارد، حذف خودکار، کاور.
- `GET/POST/DELETE /api/v1/bots/:id/folders` — پوشه‌بندی.
- `GET/POST/DELETE /api/v1/bots/:id/locks` — قفل کانال/گروه/لینک/ربات دوم.
- `GET/POST/DELETE /api/v1/bots/:id/admins` — چند ادمین با پرمیشن granular.
- `POST /api/v1/bots/:id/backup` (ساخت) / `POST /api/v1/bots/:id/restore` (آپلود بکاپ).
- `POST /api/v1/bots/:id/broadcast` — ارسال همگانی به کاربران آن ربات.
- `POST /api/v1/bots/:id/tools/bulk` — ابزارهای انبوه (خاموش/روشن قفل فوروارد همه، حذف کامل محتوا — با تأییدیه‌ی دوم اجباری در بدنه‌ی درخواست).
- `GET /api/v1/bots/:id/pending-content` + `POST .../approve` / `.../reject` — صف تأیید محتوا.

**End Customer (کاربر نهایی که فایل می‌خرد):**
- `GET /api/v1/bots/:id/public/codes/:code` — اطلاعات عمومی یک کد (بدون افشای رمز) + آیا نیاز به عضویت/رمز دارد.
- `POST /api/v1/bots/:id/public/codes/:code/unlock` — بدنه: `password?`. برگرداندن فایل یا خطا. **باید rate-limit روی (کاربر, کد) داشته باشد** — دقیقاً همان چیزی که در ممیزی امنیتی به‌عنوان مشکل باز (بروت‌فورس رمز) مستند شد؛ این‌جا فرصت خوبی است که در همان مرحله‌ی طراحی API رفعش کنیم، نه بعداً.
- `POST /api/v1/bots/:id/public/codes/:code/react` — لایک/دیس‌لایک/گزارش.

### ۴.۲ vpn-bot
**Bot Owner:**
- `GET/POST/PATCH/DELETE /api/v1/bots/:id/panels` — مدیریت پنل VPN (رمز هرگز در پاسخ GET برنگردد، فقط در نوشتن).
- `POST /api/v1/bots/:id/panels/:panel_id/test` — تست زنده‌ی اتصال.
- `GET/POST /api/v1/bots/:id/discount-codes` (فعلاً فیچر مرده در تلگرام — این‌جا API آماده می‌شود که وصل‌شدنش راحت‌تر باشد).
- `GET /api/v1/bots/:id/payments/pending` + `POST .../:payment_id/approve` / `.../reject` — تأیید دستی کارت‌به‌کارت.

**End Customer:**
- `GET /api/v1/bots/:id/public/plans` — پلن‌های قابل‌خرید این vpn-bot.
- `POST /api/v1/bots/:id/public/subscriptions` — خرید (بدنه: `plan_id, gateway`).
- `GET /api/v1/bots/:id/public/subscriptions/me` — اشتراک‌های خودِ کاربر (با چک مالکیت telegram_id — همان IDOR که در ممیزی رفع شد، این‌جا هم باید از اول درست باشد).
- `GET /api/v1/bots/:id/public/subscriptions/:sub_id/link` و `/qr` — فقط اگر `sub.UserID == caller.ID`.

### ۴.۳ archive-bot
**Bot Owner:** `GET/POST/DELETE /api/v1/bots/:id/categories`, `DELETE /api/v1/bots/:id/files/:file_id`.
**End Customer:** `GET /api/v1/bots/:id/public/search?q=` (باید از escape شدن regex مطمئن شویم — رجوع به ایراد ReDoS مستند‌شده‌ی uploader-bot، همین اشتباه در طراحی API archive-bot نباید تکرار شود).

### ۴.۴ member-bot
این سرویس زیرساخت داخلی است (نه چیزی که مشتری مستقیم بخرد) — API عمومی/کاربر لازم ندارد. تنها چیزی که
شاید لازم شود: `GET /api/v1/admin/member-checks/stats` در سطح Platform Admin برای رصد سلامت استخر check-bot ها.

### ۴.۵ ads-bot (لایه‌ی رشد — الگوی متفاوت، owner واقعی ندارد بلکه «خریدار»/«ناشر» دارد)
- Platform Admin: `GET/POST /api/v1/admin/lock-rentals/:id/approve` / `/reject` (فقط ادمین اصلی، مطابق قانونی که در ممیزی تأیید شد درست پیاده شده).
- خریدار (یک نقش جدید، نه لزوماً Bot Owner): `POST /api/v1/campaigns`, `GET /api/v1/campaigns/me`, `POST /api/v1/campaigns/:id/pause` / `/delete` (با چک `camp.PublisherID == caller.ID` — دقیقاً همان رفعِ ۲۰۲۶-۰۷-۰۲).

---

## ۵. الزامات فنی جدید (قبل از کدنویسی endpoint های بخش ۴)

1. تعریف subject های `settings.<bot_type>.*` در `shared-core/protocol/subjects.go` (مرکز واحد).
2. افزودن یک `internal/settingsresponder` به هرکدام از `uploader-bot`, `vpn-bot`, `archive-bot` که
   NATS request/reply این subject ها را جواب دهد، با auth `ServiceID="apimanager"` + `ServiceKey`
   (از همان `SERVICE_HMAC_SECRET`).
3. یک لایه‌ی «چک مالکیت» مشترک در `apimanager` (میان‌افزار `middleware.RequireBotOwnership(instanceID)`)
   که قبل از هر `/api/v1/bots/:id/...` بررسی کند `instance.OwnerID == جاری‌کاربر.ID` (یا نقش admin باشد) —
   یک‌بار نوشته شود، همه‌جا استفاده شود، تا IDOR ای که چند بار در این پروژه پیدا شد این‌جا تکرار نشود.
4. برای مسیرهای `public/*` (End Customer)، تصمیم بگیر: کاربر نهایی با چه چیزی لاگین می‌کند؟ (تلفن؟
   تلگرام Login Widget همان که `apimanager` از قبل دارد؟ یا یک deep-link توکن‌دار از خودِ ربات؟) —
   این تصمیم روی طراحی جدول‌بندی auth تأثیر مستقیم دارد و باید قبل از پیاده‌سازی بخش ۴.۱-۴.۳ با تیم
   محصول نهایی شود.
5. Rate limiting per-endpoint حساس (unlock کد، خرید VPN، تلاش رمز) — نه فقط rate limit سراسری فعلی
   `apimanager` (۶۰ req/min کلی)، بلکه per-user-per-resource.

---

## ۶. نقشه‌ی راه پیشنهادی پیاده‌سازی (فازبندی)

1. **فاز ۱ — Platform Admin کامل** (بخش ۳): همه از DB مشترک موجود (`shared-core/store`) قابل پیاده‌سازی
   است، بدون نیاز به تغییر هیچ ربات محصولی. کم‌ریسک‌ترین و سریع‌ترین فاز.
2. **فاز ۲ — زیرساخت settings.* روی یک ربات پایلوت** (پیشنهاد: `archive-bot`، چون ساده‌ترین است) — تا
  الگوی NATS request/reply + چک مالکیت را validate کند قبل از تکرار روی uploader-bot/vpn-bot پیچیده‌تر.
3. **فاز ۳ — uploader-bot + vpn-bot Bot Owner API** (بخش ۴.۱، ۴.۲ — نیمه‌ی مدیریتی).
4. **فاز ۴ — End Customer API** (نیمه‌ی عمومی بخش ۴.۱-۴.۳) — همراه با تصمیم auth کاربر نهایی (بند ۴ بالا)
   و rate-limit های اختصاصی.
5. **فاز ۵ — ads-bot** (بخش ۴.۵) — بعد از اینکه الگو روی بقیه جا افتاد، چون منطق escrow/تأیید ادمین
   ظریف‌تر است و اشتباه در آن مستقیماً پول جابه‌جا می‌کند.

## نکات/اصلاحات اضافه (اینجا بنویس)
<…>
