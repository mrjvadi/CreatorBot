# Changelog — CreatorBot V3

---
## [2026-07-14] — Sprint: audit fixes + E2E اجاره‌ی قفل + runbook + rotation

### Security — رفع باگ‌های audit (کارگاه A)

- **fraud-engine: رفع fail-open در احراز هویت admin.**
  - `internal/api/api.go` — `authMiddleware` اگر `adminKey` خالی باشد همیشه `401` می‌دهد
    (قبلاً هدر غایب `""` با کلید خالی برابر می‌شد → همه‌ی `/admin/*` باز). مقایسه با
    `crypto/subtle.ConstantTimeCompare`.
  - `cmd/main.go` — اگر `ADMIN_KEY` تنظیم نشده باشد `log.Fatal` (fail-closed در startup).
- **vpn-bot: رفع double-spend race در خرید با موجودی.**
  - `internal/store/store.go` — متد اتمیک `DeductBalanceIfEnough`
    (`UPDATE ... WHERE balance >= amount` + بررسی `RowsAffected`).
  - `internal/tgbot/user.go:confirmBuyWithBalance` — استفاده از متد اتمیک به‌جای
    check-then-`UpdateBalance`؛ دو کلیک هم‌زمان دیگر دو اشتراک نمی‌سازند.
- **vpn-bot: رفع نبود dedup در تأیید پرداخت آنلاین.**
  - `internal/store/store.go` — متد `ClaimOnlinePayment` با `ON CONFLICT DO NOTHING`.
  - `cmd/bot/main.go` — ایندکس partial یکتا `uq_payment_online_ref` روی
    `payments(gateway, ref_code)` برای گیت‌وی‌های آنلاین.
  - `internal/tgbot/user.go:verifyOnlinePayment` — پرداخت را بر اساس `refID` «claim» می‌کند؛
    کلیک تکراری «پرداخت کردم» دیگر اشتراک دوم نمی‌سازد.
- **archive-bot: رفع باگ کوچک `botUsername`.**
  - `internal/tgbot/handler.go` — `NewHandler` حالا `botUsername` را به struct ست می‌کند.
- (فقط مستند، بدون تغییر کد) **source-service hotspots** برای audit بعدی: مرز authorization در
  `internal/userbot/run_bot_command.go`، ذخیره‌ی session کامل MTProto در DB، لاگ‌شدن شماره تلفن.

### Added — تست E2E اجاره‌ی قفل کانال (کارگاه B)

- **`ads-bot/tools/e2e-lockrental/`** — ابزار تست end-to-end مدل اقتصادی lock-rental بدون تلگرام،
  با متدهای واقعی `ads-bot/internal/store` + botpay واقعی (NATS/HMAC): seed → approve → join →
  idempotency → fraud reversal → settlement → completion. داخل ماژول ads-bot و زیر `tools/`
  (نه `cmd/`) تا run.sh دست‌نخورده بماند. + `README.md`.

### Docs — runbook و آماده‌سازی مسیر A (کارگاه D)

- **`E2E_RUNBOOK.md`** — راهنمای سه سطح تست + وضعیت زیرساخت. زیرساخت داده بالاست ولی
  سرویس‌های go خاموش و local-bot-api (`141.95.210.17:8081`) در دسترس نیست → botpay startup
  را کامل نمی‌کند، پس اجرای زنده به bot API قابل‌دسترس موکول است (هر دو تست `go build` سبز).

### Security — چرخش secret ها + بازنویسی history (کارگاه C)

- **چرخش secret های اپلیکیشنی** در همه‌ی `.env` ها با `openssl rand -hex 32`، گروه‌های مشترک
  بین‌سرویسی هماهنگ (تأییدشده با hash): `SERVICE_HMAC_SECRET`، `ENCRYPTION_KEY`، `JWT_*`،
  `AGENT_API_KEY`، `INTERNAL_KEY`، `LOCK_API_SECRET`، `SERVICE_KEY_*`، `BOTPAY_API_KEY`،
  جفت‌های admin (fraud/image-registry/botpay)، `LICENSE_SIGNING_SECRET`،
  `SESSION_ENCRYPTION_KEY`، `LOG_API_KEY`، `GRAFANA_PASSWORD` و کلیدهای admin مستقل.
- **توقف ردیابی:** همه‌ی ۱۹ فایل `.env` با `git rm --cached` از git خارج شدند (روی دیسک با
  مقدار نو ماندند)؛ `.gitignore` → `**/.env` (فقط `.env.example` tracked).
- **بازنویسی history:** با `git filter-branch` همه‌ی `.env` از کل history حذف و
  `git push --force` روی `origin/main`. backup: `../CreatorBotV3-backup.git` (mirror).
- **دستی (در `SECRETS_ROTATION.md`):** پسوردهای زیرساخت (Postgres/Redis/NATS/Mongo — نیاز به
  تغییر سرور) و توکن‌های خارجی (BotFather/toncenter/پنل VPN — نیاز به revoke) عمداً env-only
  عوض نشدند چون stack را می‌شکست؛ با دستور دقیق مستند شدند.
- **`SECRETS_ROTATION.md`** (جدید) — runbook کامل چرخش.

### Process

- از این سشن هر تغییر در **CHANGELOG.md** و کل پروژه با جزئیات در **`prog/`** ثبت می‌شود
  (خواسته‌ی کاربر ۲۰۲۶-۰۷-۱۴).

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
