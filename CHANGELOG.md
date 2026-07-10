# Changelog — CreatorBot V3

---
## [2026-07-10] — Sprint: botpay allowlist + revenue-service NATS migration

### botpay/internal/store/store.go
- `ValidateServiceID` allowlist: اضافه شد `community-service`, `fraud-engine`, `revenue-service`
- قبلاً این سه سرویس اگر pay.credit/deduct می‌زدند، botpay آن‌ها را رد می‌کرد

### revenue-service — HTTP → NATS
- `cmd/main.go`: `botpayClient` HTTP struct حذف شد → `natspayAdapter` با `natspayclient.Client`
- Config: حذف `BOTPAY_URL/BOTPAY_API_KEY/BOTPAY_ADMIN_KEY` → اضافه `SERVICE_HMAC_SECRET`
- `go.mod`: `shared-core` به عنوان dependency اضافه شد
- `.env`: آدرس HTTP botpay حذف شد، `SERVICE_HMAC_SECRET` اضافه شد

---
## [2026-07-10] — Sprint: License Fail-Closed + E2E Integration

### تغییرات این سشن (کامیت fdfb693)

#### shared-core/licenseclient/client.go — تابع جدید
- `RequireValid(ctx, nc, botID, token, serverID) error` اضافه شد
- fail-closed: token خالی / NATS قطع / verify رد شده → error (نه warning)
- `RunLicenseLoop` (هر ۶ ساعت) همچنان fail-open باقی ماند

#### uploader-bot/cmd/bot/main.go
- بعد از `engine.New()` و قبل از `rawBot.Start()`:
  `licenseclient.RequireValid` با timeout 20s صدا زده می‌شود
- اگر fail کند → `log.Fatal` → container اجرا نمی‌شود

---
## [2026-07-10] — Sprint: E2E Integration Chain (7 Services)

### سرویس‌های تغییرداده‌شده

#### shared/pkg/config/config.go — باگ بحرانی fix شد
- `bindEnvs()` اضافه شد تا همه فیلدهای mapstructure صریحاً BindEnv شوند
- **بدون این fix:** container بدون .env همه config را خالی می‌گرفت (BOT_TOKEN خالی → crash-loop)

#### agentmanager/internal/docker/client.go + envfile.go
- `DeployDefaults{BaseEnv, DefaultNetwork}` اضافه شد
- `Deploy()`: env نهایی = merge(BaseEnv از botenv.env، cmd.EnvVars) — overlay برنده
- اگر `cmd.NetworkName == ""` → DefaultNetwork (`deploy_backend`)

#### agentmanager/cmd/main.go + .env
- Config: `BOT_BASE_ENV_FILE=./botenv.env`, `DEFAULT_NETWORK=deploy_backend`
- `agentmanager/botenv.env` در .gitignore — روی سرور جدید دستی بساز

#### botmanager/internal/tgbot/user/wizard.go و admin_svctest.go
- `"OWNER_ID": fmt.Sprint(u.TelegramID)` به EnvVars اضافه شد
- (قبلاً فقط OWNER_TELEGRAM بود؛ uploader-bot OWNER_ID می‌خواند)

#### shared-core/engine/engine.go
- `loadInstanceInfo()` — PlanID/LockMode از جدول bot_instances خوانده می‌شود
- `InstanceInfo` struct با `IsFreeLock()` / `IsRentedLock()`

#### tools/e2e-provision/main.go — ابزار جدید
- E2E بدون تلگرام: pay.credit/balance/deduct → license.issue → DeployCommand → ResultMsg
- اجرا: `go run . -hmac <SECRET> -bot-token <TOKEN> -server-id <UUID>`

#### run.sh
- اضافه: image-registry، license-service، apimanager
- حذف: standalone uploader-bot (حالا فقط container داینامیک)

#### botpay/.env
- اضافه: `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB`

#### image-registry/.env
- `SEED_CALLER_CIDR`: `172.16.0.0/12` → `127.0.0.1/32`

### شکاف‌های باقی‌مانده
- botpay allowlist هاردکد: community-service و fraud-engine هنوز در آن نیستند
- revenue-service هنوز HTTP قدیمی botpay را صدا می‌زند (باید NATS شود)
- botenv.env فعلاً DSN فقط برای uploader_bot دارد (vpn-bot نیاز به DSN جدا دارد)

---
## [2026-07-10] — Sprint: dbmigrate (migration ورژن‌دار)

### dbmigrate/ — ماژول جدید
- CLI: `up / status / mark / new / list` با `-dsn` و `-service`
- جدول `schema_migrations` با checksum در هر دیتابیس
- baseline SQL از AutoMigrate واقعی هر ۱۱ سرویس Postgres‌دار
- `migrations/botmanager/0001` تا `0004`:
  - 0002: fix `users_telegram_id_key` → `idx_users_telegram_id`
  - 0003: حذف legacy unique constraints روی servers/bot_instances/invite_links
  - 0004: تبدیل uuid→text برای FK ستون‌های bot_instances/plans/subscriptions

---
## [Unreleased] — Sprint 1-9

### Added
- سرویس `dbmigrate` — migration ورژن‌دار SQL برای هر ۱۱ سرویس Postgres دار
  (baseline از AutoMigrate واقعی هر سرویس؛ up/status/mark/new؛
  جدول schema_migrations با checksum؛ ساخت خودکار دیتابیس) — رجوع dbmigrate/README.md
- Self-service bot provisioning — کاربر بدون دخالت ادمین ربات می‌سازد
- Double-entry ledger در botpay
- Prometheus metrics در همه سرویس‌ها
- Grafana + Loki در docker-compose
- Audit log برای همه عملیات حساس
- Secret rotation با dual-key grace period
- Rate limiting (token bucket) در webhook-gateway
- Health check endpoints در همه سرویس‌ها
- Kubernetes manifests (16 manifest)
- Config versioning با rollback
- VPN adapters: Hiddify، X-UI، MarzNeshin
- E2E test suite
- Migration system (golang-migrate)
- Panic recovery در همه handler ها
- Context timeout در عملیات طولانی
- DB connection pool با lifecycle management

### Fixed
- uuid=text bug در PostgreSQL JOIN queries
- telebot v4 API: Photo/ForwardFrom/InlineResult
- NATS Authorization Violation در startup
- Duplicate ApprovePayment در member-bot
- Bot auto-state در admin list handlers
- Format string مشکلات در i18n

### Architecture
- NATS JetStream جایگزین Centrifugo WebSocket
- Bot engines مستقیم به DB وصل می‌شوند (نه apimanager)
- instance_id = bot_<BotID> برای persistence
- fraud-engine از request/reply NATS
