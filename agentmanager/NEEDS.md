# agentmanager — چه چیزی نیاز داریم (بررسی ۲۰۲۶-۰۷-۰۴، به‌روزرسانی همان روز)

## رفع‌شده در همین بررسی (۲۰۲۶-۰۷-۰۴، پاس دوم)
- **ناسازگاری نسخه‌ی Go**: `go.mod` نیازمند `go 1.25.0` بود ولی هر دو Dockerfile
  (`Dockerfile` و `deploy/Dockerfile`) از `golang:1.22-alpine` استفاده می‌کردند —
  یعنی build یا شکست می‌خورد یا مجبور به دانلود خودکار toolchain از اینترنت
  می‌شد. هر دو به `golang:1.25-alpine` تغییر کردند.
- **پوشه‌ی خالی `internal/nats/`** حذف شد — بازمانده از قبل از مهاجرت به
  `shared/pkg/adapters/nats`؛ agentmanager همان پکیج مشترک را استفاده می‌کند.
- **تست unit اولیه اضافه شد**: `internal/docker/client_test.go` — پوشش
  `splitImageRef` (تگ ندارد، رجیستری با پورت، ref خالی) و `isImageAllowed`
  (مجاز، رد شده، و مهم‌تر از همه fail-closed روی خطای registry client).
  توجه: `verifyManaged` هنوز تست ندارد چون به یک Docker daemon واقعی وابسته
  است (`c.cli` از نوع concrete `*dockerclient.Client` است، نه interface) —
  برای تست آن باید یا یک Docker داخل CI باشد یا این وابستگی به یک interface
  کوچک تبدیل شود.
- **`.env.example` بازنویسی کامل شد** تا دقیقاً با `Config` واقعی در
  `cmd/main.go` مطابقت داشته باشد — ارجاعات قدیمی به `API_MANAGER_URL`,
  `AGENT_API_KEY`, `CENTRIFUGO_URL`, `CENTRIFUGO_API_KEY`,
  `CENTRIFUGO_WS_ENDPOINT` (که هیچ‌کدام در کد واقعی نیستند) حذف شدند و
  `NATS_URL`/`NATS_USERNAME`/`NATS_PASSWORD` جایگزین شد.

## وضعیت فعلی
`deploy`/`stop`/`remove`/`restart` واقعی روی Docker، سخت‌گیری امنیتی هر container، و (رفع‌شده
۲۰۲۶-۰۷-۰۲) چک label `creatorbot.managed` قبل از هر عملیات مخرب‌پذیر — رجوع
`docs/security-audit-2026-07-02.md` بخش ۲.

**تغییر بزرگ همین بررسی (۲۰۲۶-۰۷-۰۴)**: whitelist محلیِ image (`ALLOWED_IMAGES` در `.env`، prefix
matching داخل خودِ agentmanager) کاملاً حذف و با یک سرویس مرکزی جدید — `image-registry` — جایگزین شد.
حالا هر agentmanager قبل از هر `Deploy` از آن سرویس (`GET /v1/check`) می‌پرسد «آیا این image:tag مجاز
است؟»، و آن سرویس این تصمیم را بر اساس IP واقعیِ خودِ agentmanager (نه یک ادعای داخل payload) می‌گیرد.
جزئیات کامل طراحی/محدودیت‌ها: `image-registry/README.md`. فایل‌های تغییریافته:
`internal/docker/client.go` (`SecurityPolicy.AllowedImages` حذف شد، `isImageAllowed` حالا async و
fail-closed روی خطای شبکه هم هست)، `internal/registryclient/` (کلاینت HTTP تازه)، `cmd/main.go`.

## چیزی که واقعاً کم است
0. **(بحرانی — یافته‌ی بررسی ۲۰۲۶-۰۷-۰۵) دانلود+load خودکار image هنوز پیاده نشده.** `image-registry`
   حالا خودِ فایل image را هم ذخیره/توزیع می‌کند (`GET /v1/check` وقتی فایل موجود باشد `download_url`+
   `sha256` هم برمی‌گرداند) — ولی `agentmanager` این را مصرف نمی‌کند:
   `internal/registryclient/registryclient.go`'s `IsAllowed` فقط فیلد `allowed` را می‌خواند؛
   `internal/docker/client.go`'s `Deploy()` هنوز فقط `ImageInspectWithRaw` می‌زند و اگر image محلی
   نبود، خطا می‌دهد (باید دستی `docker load` شود). **کاری که لازم است:**
   - در `registryclient`: یک متد جدید (مثلاً `Fetch(ctx, name, tag) (*CheckResult, error)`) که کل پاسخ
     `/v1/check` (شامل `download_url`, `sha256`, `size`) را برگرداند، نه فقط `bool`.
   - یک متد `Download(ctx, downloadURL, destPath string) (sha256 string, err error)` که فایل را با
     `io.Copy` استریم کند (نه کل آن در حافظه) و هم‌زمان sha256 را حساب کند.
   - در `docker.Client`: قبل از `ImageInspectWithRaw`، اگر image محلی نبود و `has_file=true` بود،
     دانلود کند، `sha256` را چک کند (عدم تطابق = رد deploy، fail-closed)، و `docker load` بزند (از
     طریق Docker SDK: `ImageLoad(ctx, reader, ...)`، نه CLI — طبق همان قاعده‌ی کل این فایل که هیچ
     `exec.Command` نباشد).
   - تصمیم طراحی باز: فایل دانلودشده کجا موقت ذخیره شود (دیسک local سرور agentmanager) و کِی پاک شود؟
     (بعد از `docker load` موفق، یا نگه‌داشتن به‌عنوان کش برای instance های بعدی همان image؟)
1. **بدون Kubernetes manifest** — تنها ۷ سرویس در `deploy/k8s/services/` هستند و agentmanager یکی
   از آن‌ها نیست. با توجه به بحث قبلی درباره‌ی مهاجرت به K8s: این مهم‌ترین سرویس برای آن مهاجرت است
   (چون orchestrator است) ولی هنوز حتی شروع نشده. `image-registry` هم به همین دلیل باید هم‌زمان اضافه شود.
2. **تست unit ناقص** — (رفع جزئی ۲۰۲۶-۰۷-۰۴) `isImageAllowed`/`splitImageRef` حالا تست دارند؛
   `verifyManaged` و کل `internal/docker.Client` (Deploy/Stop/Remove/Restart/ListContainers) هنوز
   تست ندارند چون مستقیم به Docker SDK concrete client وابسته‌اند، نه یک interface قابل‌mock — برای
   تست کامل باید یا این وابستگی abstract شود یا از یک Docker daemon واقعی در CI استفاده شود.
3. **Heartbeat بدون ACL** — لیست کامل container های هر سرور روی یک subject بدون احراز هویت منتشر
   می‌شود (شناسایی/recon، نه دسترسی مستقیم) — بخشی از ضعف عمومی NATS، نه چیزی که فقط این‌جا رفع شود.
4. **`.env.example`** — (رفع‌شده ۲۰۲۶-۰۷-۰۴) بازنویسی کامل شد و حالا دقیقاً با `Config` واقعی مطابقت
   دارد.
5. **جایی برای `Makefile`'s `build-all`** — این سرویس در `deploy/Makefile`'s `build-all` هست، خوب؛
   `image-registry` باید همان‌جا هم اضافه شود (رجوع `image-registry/NEEDS.md`).

## این‌ها را در سرویس‌های دیگر هم نوشتم
- `image-registry/README.md` و `image-registry/NEEDS.md` — طرف دیگرِ همین تغییر.
