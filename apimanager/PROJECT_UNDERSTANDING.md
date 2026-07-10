# CONTEXT PROMPT — CreatorBotV3 / apimanager

> پرامتِ زمینه، مثل `botmanager/PROJECT_UNDERSTANDING.md`. هدف این فایل این است که هرکس (یا هر مدلی) روی
> `apimanager` کار می‌کند، بدون نیاز به خواندن کل ریپازیتوری، تصویر کامل پلتفرم + همه‌ی قابلیت‌های واقعیِ
> هر سرویس + جایگاه دقیق این سرویس در آن را داشته باشد. آزادانه ادیت کن و کل فایل را به‌روز نگه دار.

---

## ۱. این پروژه دقیقاً چیست

**CreatorBotV3** یک PaaS برای ساخت ربات تلگرام بدون کدنویسی است، با سه لایه:
- **لایه‌ی فروش** (`botmanager`) — کاربر ربات خودش را می‌خرد/می‌سازد.
- **لایه‌ی محصول** (`uploader-bot`, `vpn-bot`, `archive-bot`, `member-bot`) — خودِ ربات‌های ساخته‌شده.
- **لایه‌ی رشد/درآمد** (`ads-bot`, `admanager-bot`, `community-service`, `fraud-engine`, `revenue-service`) — تبلیغات، اجاره‌ی قفل کانال، تقسیم درآمد.

شبیه Shopify (ساخت بدون کد) + Stripe (کیف‌پول داخلی TON) + Google Ads، برای دنیای ربات‌های تلگرام.

### قانون بنیادی معماری
هیچ سرویسی مستقیم به DB سرویس دیگر کوئری نمی‌زند. ارتباط فقط با **NATS**: Request/Reply برای چیزهایی
که به پاسخ فوری نیاز دارند (پول، لایسنس، عضویت)، Publish/Sub برای رویدادها. اکثر سرویس‌های مرکزی روی
یک PostgreSQL مشترک نشسته‌اند (تصمیم آگاهانه برای سادگی فعلی، نه هدف نهایی) — ولی هرکدام فقط جدول‌های
خودش را می‌بیند.

---

## ۲. apimanager دقیقاً کجای این تصویر است

`apimanager` **دروازه‌ی HTTP بیرونی** پلتفرم است — برای روزی که یک وب‌سایت یا اپ موبایل بخواهد بدون
تلگرام با پلتفرم حرف بزند. طبق کامنت خودِ `shared-core/engine`: *«apimanager دیگر در مسیر hot path
نیست»* — یعنی خودِ ربات‌های محصول (uploader-bot و بقیه) مستقیم به DB وصل می‌شوند، نه از طریق این سرویس.
پس این سرویس امروز **کم‌استفاده** است؛ برای آینده نگه داشته شده.

### مسئولیت‌های واقعی امروز (`apimanager/cmd/main.go` + `internal/handler`, `internal/middleware`)
- `POST /api/v1/auth/telegram` — ورود با Telegram Login Widget. امضای HMAC را با `verifyTelegramAuth`
  چک می‌کند (fail-closed اگر `BOT_TOKEN` تنظیم نشده)، `auth_date` را هم expiry می‌زند، بعد JWT
  access/refresh صادر می‌کند. **این مسیر قبلاً به‌اشتباه در یک گزارش موقتی «ناامن» گزارش شده بود؛ بعد از
  خواندن مستقیم کد تأیید شد که کاملاً درست پیاده شده — به آن دست نزن مگر واقعاً چیزی عوض شده باشد.**
- `POST /api/v1/auth/refresh` — تمدید access token از refresh token.
- `POST /api/v1/agent/auth` + `agent.*` (پشت `middleware.AgentKeyAuth(cfg.AgentAPIKey)`) — یک مسیر HTTP
  موازی برای همان چیزی که عمدتاً روی NATS انجام می‌شود (heartbeat/نتیجه‌ی agentmanager).
- کاربر (پشت `middleware.JWTAuth`): `GET /me`, `instances` CRUD (`start/stop/restart/delete/logs`),
  `GET /plans`.
- ادمین (پشت `JWTAuth` + `middleware.RequireRole("admin","owner")`): `GET /admin/stats`, مدیریت
  `servers`/`templates`.
- Rate limit ساده: ۶۰ درخواست در دقیقه به‌ازای IP روی مسیرهای کاربر (`middleware.RateLimit`).
- گوش‌دادن مستقیم به `agent.*.heartbeat`/`agent.*.result` روی NATS (بدون رفتن از مسیر HTTP) برای
  به‌روزرسانی وضعیت سرور/instance در DB مشترک.
- (اضافه‌شده ۲۰۲۶-۰۷-۰۲) `log.AttachNATS(nc, "apimanager")` — هر `Warn`/`Error`/`Fatal` این سرویس هم
  الان روی `logs.events` منتشر می‌شود؛ رجوع به بخش ۷.

### مدل‌ها و DB
از `shared-core/models`/`shared-core/store` استفاده می‌کند — همان مدل‌های `botmanager`
(`User, Server, BotTemplate, BotInstance, Plan, PlanBotLimit, Subscription, Payment, InviteLink,
DeployJob, AuditLog`). Migration با `pg.Conn().AutoMigrate(models.AllModels()...)`.

---

## ۳. فهرست کامل قابلیت‌های پروژه، به‌تفکیک سرویس

این بخش خلاصه‌ی جدولی نیست — همه‌ی قابلیت‌های واقعی‌ای است که از خواندن مستقیم کد هر سرویس (نه فقط
مستندات) استخراج شده. اگر می‌خواهی بدانی «آیا فلان قابلیت وجود دارد»، اول این‌جا را نگاه کن.

### ۳.۱ botpay — کیف‌پول مرکزی TON
- کیف‌پول per-user با دو بخش جدا: `TONBalance` (واقعی، قابل‌برداشت) و `Credit` (پاداش/استرداد، غیرقابل‌برداشت مستقیم).
- واریز TON با **یک آدرس مشترک پلتفرم** + کد comment یکتا به‌ازای هر invoice (نه لینک پرداخت آنلاین)؛ `ton.Watcher` بلاک‌چین را poll می‌کند.
- پشتیبانی از واریز جزئی (partial payment) — یک invoice می‌تواند در چند تراکنش کامل شود.
- کسر/اعتبار/انتقال داخلی P2P بین دو کاربر، با قفل ردیف (`SELECT FOR UPDATE`) ضد race.
- برداشت با حداقل مبلغ ثابت + کارمزد شبکه‌ی ثابت.
- تاریخچه‌ی کامل تراکنش هر کیف‌پول.
- **زنجیره‌ی هش‌شده (Ledger)** شبیه بلاک‌چین داخلی برای تشخیص دستکاری دیتابیس + `chainguard` که دوره‌ای این زنجیره را پایش می‌کند و به ادمین هشدار می‌دهد.
- **Consensus Engine** — چند "worker" محلی قبل از تأیید نهایی واریز/کسر رأی می‌دهند (لایه‌ی دفاعی مستقل از احراز هویت سرویس).
- ربات تلگرام اختصاصی برای نمایش موجودی/تاریخچه/درخواست برداشت به‌خودِ کاربر.
- اعلان فوری push به کاربر هنگام واریز تأییدشده.
- API کامل NATS request/reply برای بقیه‌ی سرویس‌ها (`pay.balance/authorize/deduct/credit/transfer/invoice.create/invoice.status`).
- (۲۰۲۶-۰۷-۰۲) احراز هویت واقعی سرویس با HMAC (`SERVICE_HMAC_SECRET`)، اعتبارسنجی مبلغ (مثبت/متناهی/سقف‌دار)، و idempotency روی کسر — قبلاً هیچ‌کدام واقعی نبودند (رجوع بخش ۵).

### ۳.۲ botmanager — ربات فروش اصلی
- ثبت‌نام خودکار کاربر با نقش (`user`/`admin`/`owner`).
- **ویزارد ساخت ربات**: انتخاب نوع سرویس (کاملاً پویا از DB، نه هاردکد) → انتخاب تگ/نسخه‌ی همان سرویس → انتخاب پلن → وارد کردن توکن ربات → تأیید → پرداخت (یا مسیر رایگان) → provisioning.
- **Provisioning کامل**: انتخاب کم‌بارترین سرور آنلاین → ساخت رکورد instance → کسر پول از `botpay` → صدور لایسنس از `license-service` (۲۰۲۶-۰۷-۰۲) → ارسال `DeployCommand` به `agentmanager` → **refund خودکار** در صورت شکست هر مرحله.
- مدیریت کیف‌پول کاربر: دکمه‌های شارژ سریع (مبالغ پیش‌فرض) + مبلغ دلخواه، مشاهده‌ی تاریخچه.
- «سرویس‌های من»: لیست ربات‌های ساخته‌شده با وضعیت زنده (running/stopped/pending/error)، عملیات stop/start/restart/delete (با تأییدیه‌ی دوم قبل از حذف)، مشاهده‌ی تنظیمات، تمدید اشتراک (کسر خودکار + افزایش `ExpiresAt`).
- **یادآور انقضای خودکار** — job پس‌زمینه هر ۶ ساعت، بازه‌ی ۷۲ ساعت مانده به انقضا، dedupe در Redis تا اسپم نشود.
- پنل ادمین کامل: مدیریت کاربران (مسدود/آزاد، ارتقا/تنزل نقش، افزودن اعتبار دستی)، مدیریت پلن‌ها (ساخت/ویرایش با دکمه‌های ➕➖ برای محدودیت هر نوع ربات جداگانه)، مدیریت سرورها (افزودن سرور جدید)، مدیریت تمپلیت‌ها (تعریف سرویس/تگ‌های تازه، کاملاً data-driven)، **تست دیپلوی داخلی** برای ادمین (بدون پرداخت واقعی، تگ مخفی `test`)، ارسال همگانی (broadcast)، آمار سیستم.
- چندزبانه‌ی کامل (fa/en) — هیچ متن نمایشی هاردکد نیست، همه از i18n.
- badge خودکار روی پلن‌ها (🆕 جدیدترین تگ، 🔥 پلن میانی به‌عنوان anchor pricing).

### ۳.۳ uploader-bot — فروش فایل با کد (کامل‌ترین ربات محصول، ~۲۸ قابلیت)
- تحویل فایل با کد دریافت، با گیت عضویت اجباری (force-join) قبل از تحویل.
- رمز عبور اختیاری روی هر کد (قابل تنظیم/حذف جداگانه).
- محدودیت تعداد دانلود هر کاربر روی هر کد.
- پوشه‌بندی چندسطحی (پوشه/زیرپوشه) برای سازمان‌دهی فایل‌ها.
- آلبوم — چند فایل زیر یک کد واحد.
- اشتراک پولی (Sub Plan) برای دسترسی نامحدود بدون نیاز به کد جداگانه.
- **چند نوع قفل هم‌زمان** روی هر کد/کانال: قفل کانال، قفل گروه، قفل لینک دعوت، قفل با **ربات دوم** (توکن مجزا، رمزنگاری‌شده در Mongo)؛ هرکدام اجباری یا اختیاری، با سقف تعداد عضو قابل تنظیم.
- حذف خودکار پیام تحویلی بعد از مدت مشخص (ضد گزارش/ضدفیلتر).
- قفل فوروارد (forward_lock) — جلوگیری از فوروارد فایل تحویلی.
- سیستم لایک/دیس‌لایک + گزارش تخلف روی هر فایل، با شمارنده‌ی قابل‌مشاهده.
- بازدید/دانلود «فیک» قابل‌تنظیم توسط ادمین (برای نمایش محبوبیت اولیه).
- کاور اختصاصی برای هر کد/فایل.
- پیش‌نمایش/اسلایدشو محتوا قبل از دریافت.
- ترتیب‌دهی دستی فایل‌ها (جابه‌جایی بالا/پایین) و جابه‌جایی بین پوشه‌ها.
- **جستجوی فارسی fuzzy** در کدها — هم از منوی داخلی، هم از inline query عمومی تلگرام.
- broadcast/ارسال همگانی (کپی یا فوروارد) با صف پس‌زمینه و حذف خودکار پیام‌های broadcast بعد از مدت.
- چند ادمین با **سطح دسترسی جداگانه به‌ازای هرکدام** (permission granular، نه یک نقش یکسان).
- صف تأیید محتوا — ادمین محتوای آپلودشده‌ی کاربر را approve/reject می‌کند قبل از انتشار.
- بکاپ/ریستور کامل داده‌های ربات.
- ابزارهای انبوه (bulk): روشن/خاموش کردن قفل فوروارد روی همه‌ی فایل‌ها یک‌جا، فعال/غیرفعال حذف خودکار روی همه، **حذف کامل همه‌ی محتوا** (با تأییدیه).
- تنظیمات عمومی قابل تغییر توسط ادمین از پنل.
- پرداخت آنلاین + مسیر تأیید دستی پرداخت توسط ادمین.
- مدیریت کانال‌های پیش‌نمایش/تبلیغاتی جداگانه.
- مسدود/آزادکردن کاربر خاص، ریست شمارنده‌ی دانلود یک کاربر.

### ۳.۴ vpn-bot — فروش اشتراک VPN
- خرید پلن با اتصال زنده به پنل Marzban (ساختار آماده برای Marzneshin/Hiddify/XUI هم هست، فعلاً فقط Marzban واقعاً وایر شده).
- **سه درگاه پرداخت**: زرین‌پال (آنلاین)، NowPayments (کریپتو)، کارت‌به‌کارت با تأیید دستی ادمین.
- ارسال لینک اتصال + QR Code اشتراک.
- تمدید اشتراک.
- مدیریت چند پنل VPN از پنل ادمین: افزودن (با تست ورود زنده هنگام ثبت)، ویرایش، حذف، toggle فعال/غیرفعال، تست همه‌ی پنل‌ها یک‌جا. رمز پنل با AES-256-GCM رمزنگاری‌شده در DB.
- کدهای تخفیف — مدل و منطق آماده در DB، ولی فعلاً به هیچ ورودی کاربر وصل نیست (فیچر نیمه‌کاره/مرده).
- زمان‌بند دوره‌ای برای چک/عملیات پس‌زمینه‌ی مرتبط با اشتراک‌ها.

### ۳.۵ archive-bot — آرشیو فایل
- آپلود و دسته‌بندی فایل با `Category`.
- **جستجوی فارسی fuzzy** با `pg_trgm` (اکستنشن PostgreSQL) + GIN index روی عنوان/تگ/توضیحات.
- حذف فایل توسط ادمین.
- ساده‌ترین ربات محصول از نظر سطح حمله — بدون پرداخت، بدون قفل کانال، بدون توکن ثانویه.

### ۳.۶ member-bot — زیرساخت داخلی چک عضویت
- پاسخ متمرکز به «آیا کاربر X عضو کانال Y هست؟» (`member.check`، با کش) — بقیه‌ی ربات‌ها مجبور نیستند خودشان در هر کانالی ادمین شوند.
- استخر «check bot» — چند ربات فرعی که واقعاً `getChatMember` می‌زنند؛ `dispatcher`/`balancer` بار را بین‌شان پخش می‌کند؛ توکن‌ها AES-256-GCM رمزنگاری‌شده.
- مدیریت `Lock` (مدل قدیمی‌تر قفل کانال پولی، پیش از انتقال اجاره‌ی قفل به `ads-bot`) — توقف/حذف با چک مالکیت (رفع‌شده ۲۰۲۶-۰۷-۰۲).
- تأیید دستی پرداخت توسط ادمین پلتفرم (فقط ادمین، رفع‌شده ۲۰۲۶-۰۷-۰۲).
- HTTP API داخلی برای عملیات قفل (پشت کلید API).

### ۳.۷ ads-bot — دو سیستم اقتصادی در یک ربات
**سیستم CPJ کلاسیک**: کمپین ساده، publisher/تبلیغ‌دهنده، شمارش impression، تحلیل کیفیت عضو (member analysis).
**سیستم اجاره‌ی قفل کانال (جدیدتر و پیچیده‌تر)**:
- خریدار بودجه و پاداش هر عضو را تعیین می‌کند → تأیید **فقط توسط ادمین اصلی پلتفرم** (نه هر ادمین) → بودجه فوری کسر می‌شود.
- چند ربات رایگان پلتفرم (که با `LockMode=free` ساخته شده‌اند) به کمپین متصل می‌شوند؛ خریدار آن‌ها را در کانال خودش ادمین می‌کند.
- عضویت واقعی (از `member-bot`) → پاداش عضو و سهم owner ربات رایگان **رزرو** می‌شود، نه فوری — **تأخیر ۲۴ ساعته‌ی ضدتقلب** با timestamp واقعی سرور.
- اگر `fraud-engine` قبل از سررسید تقلب تشخیص دهد، پاداش لغو و بودجه به کمپین برمی‌گردد.
- idempotency پاداش با `uniqueIndex` واقعی DB (نه فقط منطق اپلیکیشن).
- توقف/حذف کمپین با بازگشت بودجه‌ی باقی‌مانده **به مالک واقعی کمپین** (رفع‌شده ۲۰۲۶-۰۷-۰۲ — قبلاً هرکسی می‌توانست کمپین دیگری را حذف و بودجه‌اش را بدزدد).
- تأیید/رد کیفیت کانال قبل از ورود به استخر تبلیغاتی (فقط ادمین، رفع‌شده ۲۰۲۶-۰۷-۰۲).

### ۳.۸ admanager-bot — ابزار ادمین‌محور تبلیغ (خارج از CLAUDE.md اصلی)
- بدون کاربر نهایی/پرداخت/کیف‌پول — فقط صاحب/ادمین کانال با `OWNER_ID`.
- مدیریت کانال‌های خودِ ادمین، ساخت کمپین (گروه‌بندی تبلیغ + زمان‌بندی + هدف‌گذاری کانال، بدون workflow مالی)، قالب‌های کمپین قابل‌استفاده‌ی مجدد، پاسخ خودکار، زمان‌بند پست، آمار.
- رابطه‌اش با `ads-bot` باید توسط صاحب پروژه روشن شود (آیا جایگزین بخشی از آن است؟).

### ۳.۹ community-service — تقسیم درآمد گروه‌ها
- ثبت کامیونیتی + لینک دعوت اختصاصی قابل‌ردیابی؛ تشخیص منبع عضویت (`organic` در برابر `invite_link`).
- توزیع درآمد کمپین بین اعضای فعال بر اساس **امتیاز فعالیت** (پیام + ریپلای + ری‌اکشن + روزهای فعال، در MongoDB).
- امتیاز کیفیت کانال (از `fraud-engine`).
- اعتبارسنجی مبلغ + idempotency روی `campaign.revenue.generated` (رفع‌شده ۲۰۲۶-۰۷-۰۲).

### ۳.۱۰ fraud-engine — امتیازدهی کیفیت/تقلب
- امتیاز اعتماد هر کاربر (۰-۱۰۰ + برچسب high_risk/suspicious/normal/trusted) بر اساس رفتار (عضویت/ترک/تبلیغ/تکمیل).
- امتیاز کیفیت هر کامیونیتی/کانال.
- تاریخچه‌ی کامل تغییرات پروفایل.
- ثبت و انتشار `FraudEvent` هنگام تشخیص الگوی مشکوک (مصرف‌کننده: `ads-bot`).
- API عمومی خواندن امتیاز (بدون auth، عمدی) + API ادمین برای جزئیات/recalculate.

### ۳.۱۱ revenue-service — قوانین کمیسیون و واریز نهایی
- قوانین کمیسیون قابل‌تنظیم به تفکیک نوع درآمد (`RevenueRule`)، با seed پیش‌فرض.
- تقسیم هر `Earning` بین owner و کیف‌پول پلتفرم؛ صف پردازش pending earnings (worker پس‌زمینه).
- اعتبارسنجی مبلغ + idempotency روی `RefID` + سخت‌گیری کلید API روی `/earn` (رفع‌شده ۲۰۲۶-۰۷-۰۲).
- ⚠️ **باگ شناخته‌شده، هنوز رفع نشده**: برای واریز واقعی از یک HTTP client به `botpay` استفاده می‌کند (`/api/v1/pay/deduct`, `/api/v1/pay/credit/add`) که در `botpay` دیگر وجود ندارد (REST API آن‌جا کلاً حذف شده، فقط NATS مانده). یعنی این مسیر پرداخت احتمالاً همیشه fail می‌شود — نیاز به سوییچ به `natspayclient`/`pay.*` دارد.

### ۳.۱۲ agentmanager — اجرای واقعی Docker
- `deploy`/`stop`/`remove`/`restart` واقعی container از روی دستور NATS.
- Whitelist اجباری image (بدون آن هیچ deploy ای مجاز نیست)، بدون pull از اینترنت (فقط image محلی).
- سخت‌گیری امنیتی هر container: `no-new-privileges`، drop همه‌ی capability (فقط لیست صریح دوباره اضافه)، سقف CPU/RAM/تعداد پردازه (ضد fork-bomb)، امکان rootfs فقط‌خواندنی + tmpfs برای `/tmp`.
- Heartbeat دوره‌ای وضعیت container ها به `botmanager`/`apimanager`.
- اتصال به Docker از طریق `docker-socket-proxy` محدود (فقط CONTAINERS/IMAGES/POST باز است).
- (۲۰۲۶-۰۷-۰۲) label `creatorbot.managed` — قبل از هر stop/remove/restart چک می‌شود، جلوی حذف زیرساخت پلتفرم با یک پیام NATS جعلی را می‌گیرد.

### ۳.۱۳ webhook-gateway — دریافت webhook تلگرام
- دریافت آپدیت‌های تلگرام و forward به NATS (`webhook.<bot_id>`) برای هر رباتی که در حالت webhook (نه polling) است.
- ثبت/حذف داینامیک ربات (هم HTTP هم NATS).
- Rate limit سراسری + per-bot/IP (وصل‌شده ۲۰۲۶-۰۷-۰۲ — قبلاً تعریف‌شده ولی وصل نبود).
- مشاهده‌ی وضعیت/آمار/health.

### ۳.۱۴ license-service (جدید، ۲۰۲۶-۰۷-۰۲) — ضدکپی/ضدکلون
- صدور لایسنس امضاشده برای هر `instance_id` (=BotID) هنگام deploy، چسبیده به `ServerID` استقرار.
- بررسی دوره‌ای (هر ۶ ساعت از خودِ ربات) که هنوز روی همان سرور است.
- تشخیص clone (check-in از سرور غیرمنتظره) → رویداد هشدار، **بدون ابطال خودکار** (fail-open).
- ابطال دستی لایسنس توسط سرویس‌های مرکزی مجاز.

### ۳.۱۵ log-collector (جدید، ۲۰۲۶-۰۷-۰۲) — جمع‌آوری لاگ
- هر سرویس، لاگ‌های Warn/Error/Fatal را (نه Debug/Info) خودکار روی `logs.events` منتشر می‌کند (`shared/pkg/logger.AttachNATS`).
- ذخیره در MongoDB با ایندکس روی زمان/سرویس/سطح.
- API کوئری HTTP (`GET /logs?service=&level=&q=&from=&to=`) پشت کلید API.
- هشدار به سوپرگروه فوروم تلگرام — **هر سرویس یک topic اختصاصی خودش را می‌گیرد** (اولین لاگ آن سرویس topic را می‌سازد).

### ۳.۱۶ source-service — ناتمام (stub)
- قرار است با MTProto (`gotd/td`) به‌عنوان یک «یوزربات» واقعی، فایل از کانال منبع به کانال تحویل فوروارد کند.
- امروز: تمام handler ها مقدار ثابت `"TODO"` برمی‌گردانند؛ منطق واقعی صفر است.
- هشدار صریح در خودِ کد: نقض قوانین تلگرام (ToS)، فقط برای آرشیو شخصی توصیه شده.

### ۳.۱۷ کتابخانه‌های مشترک (shared / shared-core)
- `shared`: رمزنگاری AES-256-GCM + JWT + HMAC سرویس (`auth`)، آداپتورهای Postgres/Mongo/Redis/NATS/webhook/marzban/zarinpal/nowpayments، `config` (بارگذاری env)، `logger` (zap + NATS sink)، `metrics` (Prometheus مشترک).
- `shared-core`: مدل‌های مرکزی `botmanager`، `protocol` (مرکز تعریف همه‌ی NATS subjects/پیام‌ها)، `engine` (موتور کامل هر bot container — DB مستقیم + heartbeat + license loop)، `natspayclient`، `licenseclient`، `docstore`/`configstore` (داده‌ی per-bot در Mongo)، `schema` (مدیریت schema امن).

---

## ۴. نقشه‌ی کامل NATS (تا امروز)

### Request/Reply
| Subject | مسئول پاسخ | کاربرد |
|---|---|---|
| `pay.balance`, `pay.authorize`, `pay.deduct`, `pay.credit`, `pay.transfer`, `pay.invoice.create`, `pay.invoice.status` | botpay | همه‌ی عملیات پولی؛ auth با `ServiceID`+`ServiceKey` (HMAC از `SERVICE_HMAC_SECRET`) |
| `license.issue`, `license.verify`, `license.revoke` | license-service | صدور/چک/ابطال لایسنس؛ issue/revoke فقط با ServiceKey، verify با تطبیق خودِ توکن |
| `member.check` | member-bot | چک عضویت کانال، با کش |

### Publish/Subscribe
| Subject | فرستنده | گیرنده |
|---|---|---|
| `deploy.<server_id>` | botmanager/apimanager | agentmanager |
| `agent.<server_id>.heartbeat`, `agent.<server_id>.result` | agentmanager | botmanager, apimanager |
| `service.creation.requested/started/completed/failed` | botmanager | (رصد provisioning) |
| `freebot.created` | botmanager | ads-bot |
| `membership.joined`, `membership.left` | member-bot | fraud-engine, community-service, ads-bot |
| `fraud.detected` | fraud-engine | ads-bot |
| `campaign.revenue.generated` | ads-bot | community-service |
| `earning.created` | ads-bot, community-service | revenue-service |
| `wallet.updated` | botpay | همه (باطل‌کردن کش Redis) |
| `license.clone_detected` | license-service | (فعلاً هیچ subscriber ای ندارد — قدم بعدی طبیعی: botmanager) |
| `logs.events` | همه‌ی سرویس‌ها (از طریق `log.AttachNATS`) | log-collector |

---

## ۵. وضعیت امنیتی — چیزهایی که باید بدانی قبل از دست‌زدن به auth/pay/license

گزارش کامل: `docs/security-audit-2026-07-02.md`. خلاصه‌ی چیزی که مستقیم به apimanager مربوط است:

- **تأیید شد، دست نزن**: `TelegramAuth` در apimanager از قبل درست امضای HMAC ویجت لاگین را چک می‌کند
  (`verifyTelegramAuth`, fail-closed، expiry). یک گزارش قدیمی‌تر این را اشتباهاً «ناامن» خوانده بود.
- **الگوی auth سرویس‌به‌سرویس** که در همین جلسه برای رفع یک باگ critical در `botpay` ساخته شد
  (`shared/pkg/auth.ComputeServiceKey`/`ValidateServiceKey`, env `SERVICE_HMAC_SECRET`) الان الگوی
  استانداردِ پلتفرم است — هر subject پولی/حساس جدید باید همین را استفاده کند.
- **ضعف ریشه‌ای هنوز باز**: NATS هیچ ACL سطح-subject ندارد — همه‌ی سرویس‌ها یک `NATS_USERNAME`/
  `NATS_PASSWORD` مشترک دارند. رفع کامل یعنی تنظیم accounts/permissions واقعی NATS.
- `apimanager` خودش `AgentKeyAuth`, `JWTAuth`, `RequireRole` را درست پیاده کرده — الگوی خوبی برای هر
  endpoint HTTP جدید.

---

## ۶. راه‌اندازی apimanager (`cmd/main.go`)

env لازم: `POSTGRES_DSN, NATS_URL, NATS_USERNAME, NATS_PASSWORD, PORT (پیش‌فرض 8080),
JWT_ACCESS_SECRET, JWT_REFRESH_SECRET, ENCRYPTION_KEY, AGENT_API_KEY, BOT_TOKEN`.

زنجیره‌ی start: Postgres + AutoMigrate(models.AllModels()) → NATS (+`AttachNATS` روی logger) →
`sharedocker.NewManager(nc)` → `handler.New(...)` → subscribe `agent.*.heartbeat`/`agent.*.result` →
gin routes → metrics روی `:9090` → `ListenAndServe` روی `:PORT`.

---

## ۷. شکاف‌های شناخته‌شده‌ی کل پلتفرم (نه فقط apimanager)

- **migration drift**: `botmanager`/`apimanager` از `AllModels()` متمرکز استفاده می‌کنند؛ `botpay`,
  `ads-bot`, `community-service`, `revenue-service`, `license-service`, `fraud-engine` هرکدام
  `AutoMigrate` مستقل دارند؛ `uploader-bot, vpn-bot, member-bot, archive-bot` اصلاً migration خودکار ندارند.
- **botpay ServiceKey allowlist هاردکد**: سرویس مجاز جدید باید دستی در کد `botpay` اضافه شود.
- **revenue-service ↔ botpay ناهماهنگ** (بخش ۳.۱۱) — هنوز رفع نشده، اولویت بالا.
- **admanager-bot در CLAUDE.md ریشه نیست** — رابطه‌اش با `ads-bot` باید روشن شود.
- **NATS بدون ACL سطح-subject** — بزرگ‌ترین شکاف امنیتی باقی‌مانده.

---

## ۸. اگر می‌خواهی روی apimanager کار کنی، این‌ها را رعایت کن

- هر endpoint جدید که پول/داده‌ی حساس جابه‌جا می‌کند باید از الگوی `ServiceID`+`ServiceKey` یا
  `JWTAuth`/`RequireRole` موجود استفاده کند.
- هر subject NATS تازه باید در `shared-core/protocol/subjects.go` تعریف شود (مرکز واحد)، نه رشته‌ی خام.
- بعد از هر تغییر `gofmt` بزن؛ build کامل این‌جا ممکن نیست (sandbox به toolchain `go1.25.0` که
  `go.work` می‌خواهد دسترسی شبکه ندارد) — یک `go build ./...` واقعی قبل از merge اجرا کن.
- `log.Warn`/`Error`/`Fatal` الان خودکار به `logs.events` هم می‌رود — پیام/فیلدها را برای خواننده‌ی
  انسانی (Mongo/تلگرام) هم قابل‌فهم بنویس.

## ۹. طراحی API کامل + فرانت‌اند وب (`API_DESIGN.md`, `web/`) — ۲۰۲۶-۰۷-۰۳

`apimanager/API_DESIGN.md` سند طراحی (نه کد) برای تبدیل apimanager از یک HTTP gateway نیمه‌کاره به API
کامل پلتفرم است — سه سطح دسترسی (Platform Admin / Bot Owner / End Customer)، و ۵ فاز پیشنهادی
پیاده‌سازی (فاز ۱ = Platform Admin کامل با DB مشترک موجود، بدون نیاز به تغییر ربات‌های محصول؛
فازهای ۲-۵ نیاز به `internal/settingsresponder` جدید در uploader-bot/vpn-bot/archive-bot دارند تا
تنظیمات هر ربات از طریق NATS request/reply در دسترس apimanager باشد).

**وضعیت پیاده‌سازی (به‌روز شد بعد از وصل‌شدن `shared`/`shared-core` به همین workspace در ادامه‌ی همین
روز):** تقریباً کل فاز ۱ (Platform Admin) به‌جز چند مورد مشخص:

پیاده‌شده:
- `GET /admin/users?search=&role=`، `GET /admin/users/:id` (کاربر + instance ها + اشتراک).
- `POST /admin/users/:id/role`، `/block`، `/unblock` (متدهای `store.SetUserRole`/`SetUserBlocked` از
  قبل وجود داشتند).
- `POST /admin/users/:id/credit` — از مسیر رسمی `natspayclient.Credit` → NATS `pay.credit` → botpay رد
  می‌شود؛ **مستقیم `User.Balance` را نمی‌نویسد** (با این‌که آن فیلد روی مدل هست) چون طبق قانون بنیادی
  پلتفرم فقط botpay نویسنده‌ی موجودی است (بخش ۵).
- `GET/POST/PATCH/DELETE /admin/plans` + `PATCH /admin/plans/:id/limits` — دو متد `UpdatePlan`/
  `DeletePlan`/`ListAllPlans` تازه به `store.go` اضافه شدند (الگوی دقیقاً مثل متدهای مشابه موجود).
- `PATCH/DELETE /admin/templates/:id` — همین‌طور `UpdateTemplate`/`DeleteTemplate` تازه اضافه شدند.
- `GET /admin/servers/:id/instances`.
- همه‌ی این‌ها یک `AuditLog` (best-effort) هم ثبت می‌کنند.

هنوز پیاده نشده (تصمیم آگاهانه، نه فراموشی):
- **پروکسی log-collector/license-service** — `licenseclient.Verify` به توکن لایسنس خودِ instance نیاز
  دارد که apimanager هیچ‌جا ذخیره‌اش نمی‌کند؛ قبل از این باید یک فیلد/مسیر جدید برای نگه‌داشتن آن طراحی شود.
- **broadcast ادمین** — نیاز به صف/job پس‌زمینه دارد (سند صراحتاً می‌گوید synchronous نباشد)، خارج از
  scope یک افزودن سریع.
- **test-deploy تمپلیت** — به ساخت یک instance واقعی (ولو تستی) نزدیک می‌شود؛ ریسک بیشتری داشت که
  بدون هماهنگی بیشتر پیاده شود.

**یافته‌ی مهم جانبی**: هیچ‌کدام از struct های `shared-core/models` تا امروز `json` tag نداشتند — یعنی
هر endpoint ای که مستقیم یک struct مدل برمی‌گرداند (نه یک `gin.H` دستی)، در واقع PascalCase (نام فیلد Go)
سریالایز می‌کرد، نه snake_case (که همه‌ی پاسخ‌های دستی apimanager از آن استفاده می‌کنند). این حالا با
افزودن `json:"snake_case"` روی `Base`/`User`/`Server`/`BotTemplate`/`BotInstance`/`Plan`/`PlanBotLimit`/
`Subscription` رفع شد؛ `BotToken` و `EnvOverrides` هم عمداً `json:"-"` شدند (نباید هیچ‌وقت در پاسخ HTTP
دیده شوند). اگر جای دیگری هم مدل‌ها را مستقیم JSON می‌کند (مثلاً یک endpoint دیگر در آینده)، حالا فرمتش
تغییر کرده — قبلش هوشیار باش.

فرانت‌اند وب (`apimanager/web/`) برای همه‌ی endpoint های بالا UI دارد: صفحه‌ی «کاربران» (جزئیات + تغییر
نقش + مسدودسازی + افزودن اعتبار)، صفحه‌ی جدید «پلن‌ها» (CRUD کامل + محدودیت هر نوع ربات)، صفحه‌ی
«قالب‌ها» (ویرایش/حذف)، صفحه‌ی «سرورها» (مشاهده‌ی instance ها). جزئیات کامل در `web/README.md`.

**محدودیت تأیید**: `go build ./...` در این sandbox هنوز کامل قابل‌اجرا نیست — پراکسی ماژول Go برای دو
پکیج مشخص (`klauspost/compress`, `gabriel-vasile/mimetype`) پیوسته ۴۰۳ برمی‌گرداند. همه‌ی تغییرات Go با
`gofmt -e` (پارس صحیح) + خواندن مستقیم سورس هر متد/فیلد استفاده‌شده تأیید شدند، نه با کامپایل واقعی —
حتماً قبل از merge یک `go build ./...` واقعی (لوکال یا CI) اجرا شود.

## نکات/اصلاحات اضافه (اینجا بنویس)
<…>
