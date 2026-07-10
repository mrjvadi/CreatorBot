# تحلیل شکاف‌ها و پیشنهادها — CreatorBotV3 (۴ تیر ۱۴۰۵ / 2026-07-04)

## ⚠️ اصلاحیه‌ی مهم قبل از هر چیز

در بررسی این دور مشخص شد **`apimanager` و یک فرانت‌اند وب واقعی (`apimanager/web`)** بین جلسه‌ی قبلی و
همین حالا به‌طور قابل‌توجهی گسترش پیدا کرده‌اند — نه توسط من، احتمالاً توسط شما یا یک session دیگر
موازی. یعنی دو فایلی که همین جلسه‌ی پیش نوشتم (`apimanager/PROJECT_UNDERSTANDING.md` و
`apimanager/API_DESIGN.md`) الان **قدیمی‌اند**: من آن‌جا apimanager را «کم‌استفاده، فقط auth+instance
پایه» توصیف کرده بودم، ولی الان `apimanager/internal/handler/handler.go` از ۲۱۰ خط به **۱۵۲۵ خط** و
از ۱۵ روت به **بیش از ۴۰ روت** رسیده (مدیریت کاربران، پلن‌ها با محدودیت هر نوع ربات، سرورها، تمپلیت‌ها،
پرداخت‌ها، کدهای تخفیف، و یک مکانیزم تنظیمات schema-driven برای هر instance) — و یک فرانت‌اند
React+TypeScript+Vite کامل (`apimanager/web`) با پنل کاربر و پنل ادمین این‌ها را مصرف می‌کند. بخش
بزرگی از چیزی که در `API_DESIGN.md` به‌عنوان «Platform Admin API» پیشنهاد داده بودم، **از قبل ساخته
شده**. اگر بخواهید، آن دو فایل را در یک پاس جدا به‌روزرسانی می‌کنم — این‌جا فقط برای صداقت گزارش، جلوی
گمراه‌کننده‌بودن مستندات قبلی را می‌گیرم.

بقیه‌ی این فایل، تحلیل واقعیِ به‌روزشده‌ی «چه چیزی ساخته نشده» در کل پروژه است، با شواهد مستقیم از کد.

---

## ۱. چیزهایی که واقعاً ناتمام‌اند (با سند/شاهد مستقیم)

### ۱.۱ زنجیره‌ی source-service ↔ botmanager نصفه است
`shared/PENDING_CHANGES.md` (که خودش یک فایل خوب و به‌روز است — پیشنهاد می‌کنم قبل از هر کاری این فایل
چک شود) دقیقاً این را مستند کرده: `botmanager/internal/sourceworker/responder.go` طرف botmanager یک
قرارداد کامل با `source-service` را جواب می‌دهد (register/update/heartbeat)، و یک پنل ادمین
(«System → 🛰 Source Workers») برای مدیریت worker ها دارد. **ولی** طرف واقعیِ «چه کسی task را می‌سازد و
به کدام owner/chat مربوط است» هنوز نوشته نشده — `handleUpdate` فعلاً فقط ServiceKey را چک، لاگ، و ack
می‌کند. یعنی even اگر `source-service`'s MTProto worker (که خودش هم stub است — بخش ۱.۲) کامل شود، هنوز
یک حلقه‌ی گم‌شده در وسط هست.

### ۱.۲ source-service — هنوز واقعاً stub است
تأیید شده در جلسه‌ی قبل: تمام handler های HTTP آن مقدار ثابت `"TODO"` برمی‌گردانند؛ `userbot.Start()`
فقط لاگ «TODO: implement gotd/td» می‌زند. لایه‌ی auth ساده‌اش (`X-API-Key`) درست است، ولی صفر منطق
واقعی پشتش هست.

### ۱.۳ apimanager: تنظیمات ذخیره می‌شود ولی روی ربات واقعی اعمال نمی‌شود
`PUT /api/v1/instances/:id/settings` (handler.go:873) مقادیر را طبق schema تمپلیت اعتبارسنجی و در
`UpdateInstanceEnvOverrides` ذخیره می‌کند — ولی پاسخش صریحاً `"applied": false` است. یعنی هیچ‌جا این
override واقعاً به container زنده (از طریق یک `DeployCommand` جدید به `agentmanager`، یا حداقل یک
restart) پوش نمی‌شود. این یک فیچر نصفه‌کاره است: کاربر تنظیمات را «ذخیره» می‌کند ولی هیچ اتفاقی نمی‌افتد.

### ۱.۴ apimanager بدون CORS
خودِ README فرانت‌اند صریحاً این را می‌گوید و کد هم تأییدش می‌کند (هیچ middleware ای به نام cors در کل
`apimanager` نیست): اگر فرانت و بک‌اند هم‌مبدأ deploy نشوند، مرورگر همه‌ی درخواست‌ها را بلاک می‌کند. این
یک مانع production واقعی است، نه یک نکته‌ی جزئی.

### ۱.۵ کد مرده‌ی مستندشده در shared-core (خودِ پروژه این را قبلاً پیدا کرده)
طبق `shared/PENDING_CHANGES.md`: `shared-core/docstore/uploader.go`'s `CodeStore`/`FileStore` و
`shared-core/documents/uploader.go`'s `Code`/`File`/`CodeUsage` هیچ‌جا import نمی‌شوند (uploader-bot
مدل‌های خودش را دارد که جایگزینشان کرده) — امن برای حذف. `shared-core/schema` package هم (برای
جداسازی فیزیکی DB هر instance) نوشته شده ولی هیچ‌جا صدا زده نمی‌شود — عمداً scaffolding برای هدف
بلندمدتی است که هنوز شروع نشده.

### ۱.۶ `apimanager/internal/handler/agent_auth.go`'s `BotAuth` — handler یتیم
تعریف شده (`agent_auth.go:76`) ولی در `cmd/main.go` به هیچ route ای وصل نیست — یا باید تکمیل/وصل شود،
یا حذف شود تا کد مرده نماند.

---

## ۲. لایه‌ی «Bot Owner / End Customer API» که در `API_DESIGN.md` طراحی شد ولی هنوز ساخته نشده

مکانیزم تنظیمات فعلی apimanager (بخش ۱.۳) generic و schema-driven است — خوب برای «چند تا فیلد ساده‌ی
env-var-مانند»، ولی **جایگزین** چیزهایی که در `API_DESIGN.md` بخش ۴ طراحی شده نیست:
- هیچ endpoint ای برای CRUD واقعی روی داده‌ی داخل هر ربات نیست (لیست/حذف تک‌تک کدهای uploader-bot،
  مدیریت پنل‌های vpn-bot، دسته‌های archive-bot، قفل‌های member-bot).
- هیچ «End Customer API» عمومی نیست (خرید VPN از وب، آنلاک کد فایل از وب، ...) — این بخش کلاً هنوز
  صفر است، نه فقط ناقص.
- بدون این، مسیر لازم (بخش ۲ در `API_DESIGN.md`: یک NATS responder «تنظیمات» در هر ربات محصول) هنوز
  در هیچ رباتی پیاده نشده.

---

## ۳. جاهایی که Web UI برای دو سرویس تازه‌ساخته‌شده وجود ندارد

`license-service` و `log-collector` (که همین چند جلسه پیش ساختیم) **هیچ صفحه‌ای در `apimanager/web`
ندارند** — نه دیدن هشدار clone یک لایسنس، نه مرور لاگ‌های جمع‌آوری‌شده (با اینکه `log-collector` خودش
یک `GET /logs` دارد، هیچ صفحه‌ای در فرانت آن را صدا نمی‌زند؛ apimanager هم هیچ proxy ای بهش ندارد).

---

## ۴. تست و کیفیت

بررسی مستقیم (`find . -name "*_test.go"`): فقط ۱۱ فایل تست unit در کل مونوریپو + ۷ فایل integration/e2e
در `tests/`. سرویس‌هایی که **صفر تست unit** دارند: `agentmanager`, `apimanager`, `archive-bot`,
`community-service`, `license-service`, `log-collector`, `vpn-bot`, `ads-bot`, `admanager-bot`,
`source-service`, و **`uploader-bot`** (۶۲ فایل Go، بزرگ‌ترین ربات محصول، فقط تست integration خارجی
دارد، نه unit test داخلی). سرویس‌هایی که تست دارند: `botmanager`, `botpay`, `fraud-engine`,
`member-bot`, `revenue-service`, `shared`, `shared-core`, `webhook-gateway`.

هیچ CI/CD ای در ریپو نیست (نه `.github/workflows`، نه `.gitlab-ci.yml`). فقط `deploy/Makefile` هست که
`build-all` اش فقط ۸ سرویس را می‌سازد (`botmanager apimanager botpay agentmanager webhook-gateway
revenue-service community-service fraud-engine`) — یعنی حتی این هم قدیمی شده و شامل `uploader-bot,
vpn-bot, archive-bot, member-bot, ads-bot, admanager-bot, source-service, license-service,
log-collector` نمی‌شود.

---

## ۵. زیرساخت استقرار (Deployment)

- **Dockerfile**: تقریباً همه‌جا هست (بعضی در ریشه‌ی سرویس، بعضی در `<service>/deploy/Dockerfile`) —
  فقط **`ads-bot` و `admanager-bot` هیچ Dockerfile ای ندارند** (هماهنگ با این‌که طبق CLAUDE.md فعلاً با
  `go run` تست می‌شوند، نه container).
- **Kubernetes manifest** (`deploy/k8s/services/`): فقط ۷ سرویس دارند (apimanager, botmanager, botpay,
  community-service, fraud-engine, revenue-service, webhook-gateway). **`agentmanager` خودش هم منیفست
  K8s ندارد** — نکته‌ی مهم برای بحث قبلی‌مان درباره‌ی مهاجرت به K8s: لایه‌ی مرکزی تقریباً آماده است ولی
  حتی خودِ agentmanager (که قرار است orchestrator باشد) هنوز روی K8s تعریف نشده، چه برسد به ربات‌های
  داینامیک مشتری.
- Observability (Prometheus + Loki/promtail + Centrifugo) در `deploy/` تنظیم شده و به نظر کامل می‌رسد.

---

## ۶. پیشنهادهای من (اولویت‌بندی‌شده)

1. **اول از همه: مستندات را همگام کن.** بگو بروم `PROJECT_UNDERSTANDING.md`/`API_DESIGN.md` را با
   وضعیت واقعی apimanager به‌روز کنم — الان گمراه‌کننده‌اند برای هرکسی (یا هر مدلی) که بعداً بخواندشان.
2. **اعمال واقعیِ تنظیمات (بخش ۱.۳)** — این کوتاه‌ترین راه بین «چیزی که کاربر فکر می‌کند کار کرده» و
   «چیزی که واقعاً اتفاق افتاده» است؛ یک باگ اعتماد کاربر است، نه فقط یک فیچر ناقص.
3. **CORS در apimanager** — بدون این، فرانت‌اندی که همین الان آماده است اصلاً در production کار نمی‌کند.
4. **جمع کردن `build-all` در Makefile برای همه‌ی سرویس‌ها** — کار کمی است، فوراً جلوی «سرویس جدید ساختیم
   ولی یادمان رفت جایی اضافه‌اش کنیم» را می‌گیرد (دقیقاً همان کلاس مشکلی که در امنیت هم چند بار دیدیم:
   migration drift، allowlist فراموش‌شده).
5. **تست unit برای uploader-bot** — بزرگ‌ترین و پرقابلیت‌ترین سرویس، صفر تست داخلی. حتی پوشش جزئی
   (مثلاً روی منطق قفل/رمز که قبلاً هم نقطه‌ضعف امنیتی داشت) ارزش بالایی دارد.
6. **حذف کد مرده‌ی مستندشده در shared-core** (بخش ۱.۵) — کار کم‌ریسک و سریع، فقط چون قبلاً تصمیم
   گرفته و مستند شده، نگه‌داشتنش بی‌دلیل است.
7. **صفحه‌ی وب برای license-service/log-collector** — حالا که فرانت‌اند وجود دارد، وصل‌کردن این دو
   سرویس به آن (حداقل یک صفحه‌ی «لایسنس‌ها/هشدارهای کلون» و یک صفحه‌ی «لاگ‌ها») کار نسبتاً کوچکی است و
   بلافاصله این دو سرویس را «قابل‌استفاده‌ی واقعی» می‌کند به‌جای این‌که فقط API داشته باشند.
8. **CI حداقلی** — حتی یک GitHub Actions ساده که روی هر PR فقط `gofmt -l` و `go vet ./...` بزند (بدون
  نیاز به کل build که در محیط من ممکن نبود ولی روی ماشین شما با toolchain درست باید کار کند) جلوی خیلی
  از رگرسیون‌های کوچک را می‌گیرد.
9. **بخش End Customer API** (بخش ۲) را آخر بگذار — تا وقتی مشخص نشده کاربر نهایی هر ربات دقیقاً با چه
   چیزی احراز هویت می‌شود (تصمیمی که در `API_DESIGN.md` بند ۵.۴ باز گذاشته شده بود)، ساختن این لایه
   زودتر از موعد است.
