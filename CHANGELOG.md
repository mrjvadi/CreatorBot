# Changelog — CreatorBot V3

---
## [2026-07-10] — Sprint: یکپارچه‌سازی botmanager+apimanager + Env Schema wizard

### فاز ۱: اصلاح تداخل heartbeat/result بین botmanager و apimanager

#### shared-core/agentlistener/ — پکیج جدید
- `HandleHeartbeat`: RecordHeartbeat (CPU/RAM/containers JSON) + loop container status
- `HandleResult`: Deploy→Running / Failure→Error / Stop→Stopped / Remove→DeleteInstance
- **هر دو** botmanager و apimanager از این پکیج استفاده می‌کنند (منطق مشترک)

#### botmanager/cmd/main.go
- QueueSubscribe queue group از `"botmanager"` به `"managers"` تغییر کرد
- heartbeat/result inline handlers حذف شدند → `agentlistener.Handle*` جایگزین
- `MarkStaleServersOffline` ticker اضافه شد (هر ۲۰ ثانیه، آستانه ۶۰ ثانیه)
- import: `encoding/json` و `protocol` و `models` حذف شدند، `agentlistener` اضافه شد

#### apimanager/cmd/main.go
- `nc.Subscribe("agent.*.heartbeat", ...)` و `nc.Subscribe("agent.*.result", ...)` حذف شدند
- توابع `handleHeartbeat`/`handleResult` → delegate به `agentlistener.Handle*`
- HTTP fallback endpoints (`/agent/heartbeat`, `/agent/result`) دست‌نخورده ماندند

**باگ رفع‌شده:** Remove موفق → botmanager آن را `StatusRunning` می‌کرد (اشتباه). حالا `DeleteInstance` صدا می‌شود.

---
### فاز ۲: Env Schema — تعریف فیلد برای image + ویرایش توسط کاربر

#### shared-core/models/models.go
- `ConfigField struct` اضافه شد (`Key/Label/Default/Required`)
- `BotTemplate.ParseConfigSchema() []ConfigField`
- `BotInstance.ParseEnvOverrides() map[string]string`
- `BotInstance.SetEnvOverrides(map[string]string)`
- import: `encoding/json` اضافه شد

#### shared-core/store/store.go
- `UpdateTemplateSchema(ctx, id, schema string) error` — فقط config_schema را آپدیت می‌کند

#### botmanager/internal/tgbot/state/state.go
- `StepWizardConfig = "wiz:config"` — کاربر فیلدهای ConfigSchema را پر می‌کند
- `StepTmplSchemaJSON = "tmpl:schema:json"` — ادمین JSON schema می‌فرستد

#### botmanager/internal/tgbot/i18n/
- `keys.go`: `KeyWizardConfigField`, `KeyWizardConfigDone`, `KeyTmplAskSchema`, `KeyTmplSchemaSet`, `KeyTmplSchemaInvalid`, `KeyBtnEditSchema` اضافه شدند
- `fa.go` و `en.go`: ترجمه‌های متناظر

#### botmanager/internal/tgbot/admin/admin_tmpl.go
- `AdminTemplatesList`: دکمه `KeyBtnEditSchema` برای هر template اضافه شد
- `AdminTemplateSchemaEdit(ctx, c, tmplID)` — شروع ویرایش schema
- `AdminTemplateSchemaSet(ctx, c, tmplID, jsonText)` — تأیید + ذخیره schema

#### botmanager/internal/tgbot/user/wizard.go
- constant: `wkCfgIdx`, `wkCfgValues` اضافه شدند
- `WizardFinish`: اگر template دارای ConfigSchema باشد → `WizardConfigStart` به‌جای confirm
- `wizardShowConfigField(ctx, c, uid, fields, idx)` — نمایش فیلد با label + default
- `wizardShowConfirm(ctx, c, uid, data, plan)` — تأیید نهایی (extracted از WizardFinish)
- `WizardConfigValue(ctx, c, st, text)` — handler state `wiz:config`: ذخیره مقدار + پیشروی
- `parseCfgValues(jsonStr)` — parse helper
- `Provision(...)`: اضافه شد `extraEnv map[string]string` — مقادیر ConfigSchema به EnvVars merge می‌شوند
- `WizardPay`/`WizardCreateFree`: `parseCfgValues(data[wkCfgValues])` → `Provision`

#### botmanager/internal/tgbot/router.go
- callback `tmpl_schema` → `AdminTemplateSchemaEdit` اضافه شد (admin-only)

#### botmanager/internal/tgbot/router_text.go
- `case stepWizardConfig:` → `WizardConfigValue`
- `case stepTmplSchemaJSON:` → `AdminTemplateSchemaSet`

#### botmanager/internal/tgbot/state.go (alias file)
- `stepWizardConfig` و `stepTmplSchemaJSON` alias اضافه شدند



---
## [2026-07-10] — Sprint: fail-closed license برای همه ربات‌ها + per-service-type base env

### vpn-bot/cmd/bot/main.go
- اضافه شد `EncryptKey string` به Config (فیلد گمشده‌ای که NewHandler نیاز داشت)
- اضافه شد `licenseclient.RequireValid` در startup — fail-closed

### archive-bot/cmd/bot/main.go
- اضافه شد `licenseclient.RequireValid` در startup — fail-closed

### agentmanager — per-service-type base env
- `DeployDefaults`: فیلد جدید `TypeEnvDir` — دایرکتوری با فایل‌های per-type
- `Deploy()`: merge ۳ لایه: BaseEnv → TypeEnv (`<dir>/<image>.env`) → cmd.EnvVars
- `envfile.go`: اضافه شد `mergeEnvMaps()` و `parseEnvFileIfExists()`
- `cmd/main.go`: Config جدید `BOT_ENV_DIR`
- `.env`: `BOT_ENV_DIR=./botenv` اضافه شد
- `botenv/uploader.env`, `botenv/vpn-bot.env`, `botenv/archive-bot.env` ساخته شدند
- `botenv/` در `.gitignore` (روی سرور جدید دستی ساخته شود)

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
