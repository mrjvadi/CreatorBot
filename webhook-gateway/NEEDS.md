# webhook-gateway — چه چیزی نیاز داریم (بررسی ۲۰۲۶-۰۷-۰۴)

## وضعیت فعلی
۶ فایل، ~۷۲۰ خط. دریافت/forward webhook تلگرام؛ رفع‌شده ۲۰۲۶-۰۷-۰۲: باگ auth (دو RouterGroup جدا از
هم که باعث می‌شد `/internal/*` هیچ‌وقت واقعاً پشت `InternalAuth` نباشد) و وصل‌نبودن
`WebhookRateLimit` — رجوع `docs/security-audit-2026-07-02.md` بخش ۳. `log.AttachNATS` هم وصل شده.

## چیزی که واقعاً کم است
1. **بدون تست unit برای مسیر اصلی forward** — فقط `internal/middleware/ratelimit_test.go` هست؛
   خودِ `handleWebhook`/ثبت داینامیک ربات تست ندارد، دقیقاً همان جایی که باگ auth قبلی پیدا شد.
2. **ثبت داینامیک ربات از NATS (`gateway.register`) بدون auth سرویس** — کسی با دسترسی NATS می‌تواند
   webhook هر ربات دلخواهی را hijack کند با فرستادن یک پیام جعلی؛ همان الگوی `ServiceHMACSecret` که
   برای `pay.*`/`license.*` ساخته شد، این‌جا هنوز پیاده نشده.
3. **بدون Kubernetes manifest.**
