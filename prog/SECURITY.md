# وضعیت امنیتی CreatorBot V3

آخرین به‌روزرسانی: ۲۰۲۶-۰۷-۲۰.

## مرزهای تثبیت‌شده

- botpay برای requestهای مالی HMAC service identity دارد؛ instanceهای bot_* علاوه بر آن در DB validate می‌شوند.
- credit و deduct با ref سرویس idempotent شده‌اند و wallet mutation داخل transaction/lock انجام می‌شود.
- source task/file اکنون HMAC، tenant، freshness پنج‌دقیقه‌ای و nonce یک‌بارمصرف fail-closed دارد.
- webhook register/unregister NATS با HMAC محافظت و secret در startup اجباری است.
- fraud admin API در نبود key بالا نمی‌آید و مقایسه constant-time است.
- agentmanager image allowlist مرکزی، managed-label، drop capabilities و resource policy دارد.
- فایل‌های env از tracking/history خارج شده‌اند؛ rotation provider/infrastructure طبق runbook هنوز عملیاتی است.
- IDOR عملیات instance در apimanager بسته شد؛ lifecycle و logs اکنون OwnerID را بررسی می‌کنند.
- block و role در هر درخواست JWT از DB تازه می‌شوند و refresh claim قدیمی را بازتولید نمی‌کند.
- BotToken جدید در هر دو manager با AES-GCM ذخیره می‌شود؛ migration رکورد plaintext قدیمی را opportunistic ارتقا می‌دهد.
- license issue/revoke فقط برای allowlist صریح `agentmanager`، `botmanager` و `apimanager` با service key امضاشده فعال است.
- license-service یک بایپسِ **عمدی و off-by-default** دارد (`TEST_LICENSE_SECRET`، ۲۰۲۶-۰۷-۱۷): اگر
  تنظیم شود، هر instance که همان مقدار را در `LICENSE_TOKEN` بفرستد بدون رکورد License واقعی تأیید
  می‌شود (برای هر BotID). fail-closed (پیش‌فرض خالی)، مقایسه constant-time، هر مصرف log می‌شود و
  startup هم warning صریح می‌دهد. **هرگز در deploymentی که مشتری واقعی هم دارد فعال نشود** — اگر لو
  برود، هر کسی می‌تواند حفاظت ضدِ کلونِ هر instance ای را دور بزند. باید مثل بقیه‌ی secretها rotate/فقط
  در محیط dev نگه داشته شود.
- (رفع‌شده env-level، ۲۰۲۶-۰۷-۲۰) **member-bot Lock HTTP API fail-open**: `LOCK_API_SECRET` هیچ‌جا
  در `member-bot/.env` تنظیم نبود؛ `authMiddleware` وقتی `apiKey` خالی است، درخواستِ بدونِ هدرِ
  `X-API-Key` را هم رد نمی‌کند. مقدارِ واقعی (که در `.env` ریشه‌ی legacy پروژه بود ولی هیچ‌وقت به
  `member-bot/.env` کپی نشده بود) اعمال شد؛ فایل‌های تصمیم/fail-closed واقعی در کد هنوز باز است
  (رجوع بدهیِ باز پایین).
- (کشف‌شده و rotate‌شده، ۲۰۲۶-۰۷-۲۰) **`license-service/TEST_LICENSE_SECRET` به‌عنوان یک secret
  واقعی (نه placeholder) در `.env.example` کامیت شده بود** — و همان مقدار زیرِ نامِ اشتباهِ
  `LICENSE_SIGNING_SECRET` در `uploader-bot/.env.example` هم تکرار شده بود. rotate شد.

## بدهی امنیتی باز با اولویت بالا

### member-bot Lock API — نیاز به fail-closed واقعی در کد (نه فقط env)

فعلاً فقط env درست شد. `member-bot/internal/lock/server.go`'s `authMiddleware` هنوز اگر
`apiKey` خالی برسد هیچ `log.Fatal`ای در startup نمی‌زند — برخلافِ الگویی که fraud-engine
(۲۰۲۶-۰۷-۱۴) و webhook-gateway/`SERVICE_HMAC_SECRET` دنبال می‌کنند. باید همان الگو این‌جا هم
اضافه شود.

### agentmanager deploy envelope — بخش اصالت رفع شد (۲۰۲۶-۰۷-۱۵، Plan 001)

DeployCommand حالا احراز هویت درون‌پیام دارد: چهار فیلد envelope
(`service_id/service_key/issued_at/nonce`)، یک `Verifier` در
`agentmanager/internal/queue/authz.go` با HMAC + پنجره‌ی تازگی + nonce store، و رد شدن
دستور پیش از enqueue در `worker.go`. `agentmanager` بدون `SERVICE_HMAC_SECRET` fail-closed
است. همه‌ی publisher های deploy (botmanager wizard/svctest، apimanager handler،
e2e-provision) از مسیر امضاشده‌ی `shared-core/docker.Manager` می‌روند.

**باقی‌مانده (پلن جدا):** override های امتیازساز `DeploySettings` (`CapAdd`،
`ReadonlyRootfs=false`، `NetworkName`) هنوز از سمت publisher قابل‌تعیین‌اند؛ باید
`SecurityPolicy` سرور authoritative شود و این‌ها را allowlist/رد کند. ACL سطح-subject NATS
و binding صریح tenant/instance هم هنوز باز است.

### Telegram webhook authenticity

public route فقط token موجود در URL را lookup می‌کند. setWebhook secret token مستقل تنظیم نمی‌کند و gateway header مخصوص Telegram را validate نمی‌کند. secret per-bot، مقایسه constant-time و rotation tokenهایی که در URL log شده‌اند لازم است.

### source file cache tenant isolation

PostgreSQL primary key مدل BotFileCache و Redis key شامل tenant نیستند. tenant باید بخشی از unique/primary key و cache namespace شود و namespace قدیمی invalidate گردد.

### Core NATS economic events

membership.joined، fraud.detected، earning.created و چند event اقتصادی دیگر application-level auth یا durability کامل ندارند. compromise credential یا subscriber downtime می‌تواند event جعلی یا گمشده بسازد. transition به JetStream فقط همراه event ID، dedup، durable consumer و replay policy امن است.

### session و PII source-service

MTProto session takeover کامل اکانت است؛ در DB با AES-GCM نگه‌داری می‌شود ولی key lifecycle/rotation و restore production-proven نیست. شماره تلفن در چند log و row key دیده می‌شود و باید masking/retention مشخص داشته باشد.

## بدهی امنیتی متوسط

- image-registry admission و tar validation تست ندارد؛ agent artifact download هنوز wired نیست.
- license clone behavior عمداً fail-open است و باید در threat model ثبت بماند.
- log-collector روی Core NATS است و audit trail قطعی نیست؛ redaction token/session/authorization باید تست شود.
- internal eventهای fraud و community به credential NATS اعتماد می‌کنند.
- secretهای خارجی و پسوردهای زیرساختی نیازمند revoke/rotation هماهنگ production هستند.

## موارد قبلاً رفع‌شده

جزئیات زمانی در CHANGELOG.md: uploader privilege escalation، botpay withdraw TOCTOU، member publisher wiring، community nil Mongo/subscriber، webhook control auth، fraud fail-open admin، VPN balance race/payment dedup، archive username و idempotency جدید credit/revenue/slot.
