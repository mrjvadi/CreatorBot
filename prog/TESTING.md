# تست — CreatorBot V3

آخرین به‌روزرسانی: ۲۰۲۶-۰۷-۱۴. راهنمای اجرا در `E2E_RUNBOOK.md`.

## ابزارهای تست E2E (بدون تلگرام)

### tools/e2e-provision
زنجیره‌ی ساخت ربات: `pay.credit/balance/deduct` → `license.issue` → `DeployCommand` →
انتظار `agent.<serverID>.result`. تأییدشده ۲۰۲۶-۰۷-۱۰.
پیش‌نیاز: botpay، license-service، image-registry، agentmanager زنده + image `uploader:dev`.

### ads-bot/tools/e2e-lockrental (جدید ۲۰۲۶-۰۷-۱۴)
کل چرخه‌ی اجاره‌ی قفل با متدهای واقعی `ads-bot/internal/store` + botpay واقعی:
seed → approve (کسر بودجه) → join (رزرو، pending) → idempotency → fraud (reversed) →
settlement (settle_at→گذشته → pay.credit → settled) → completion (done + آزادسازی slot).
- داخل ماژول ads-bot (قانون `internal/`)، زیر `tools/` نه `cmd/` (تا run.sh دست‌نخورده).
- دو کاربر تست: member1 (مسیر settlement)، member2 (مسیر fraud).
- `-emit-nats` علاوه بر مسیر مستقیم، رویدادها را روی core NATS هم منتشر می‌کند.
- `-cleanup` (پیش‌فرض true) داده‌های تست را پاک می‌کند.
- پیش‌نیاز: NATS + **botpay زنده** + Postgres `adsbot`(5434) + `SERVICE_HMAC_SECRET`.

## تست‌های integration (mock telegram)
`tests/integration/` (ماژول جدا) با `framework_test.go`/`mock.go`:
`member_bot_test.go`, `botmanager_test.go`, `uploader_bot_test.go`, `vpn_bot_test.go`؛
و `tests/e2e/e2e_test.go`. ads-bot/lock-rental را پوشش نمی‌دهند.

## مسیر A — پنل تلگرام (فقط دستی)
خرید پلن از ربات botmanager → provision → container واقعی. نیاز به:
Server + BotTemplate + Plan در DB (از پنل ادمین، نه SQL) + سرویس‌های زنده. رجوع
`E2E_RUNBOOK.md` بخش تست ۳.

## وضعیت اجرای زنده روی این ماشین (۲۰۲۶-۰۷-۱۴)
زیرساخت داده بالا (pg 5432/5434، nats 4222، redis 6379/6381، mongo 27017) ولی سرویس‌های go
خاموش و local-bot-api (`141.95.210.17:8081`) در دسترس نیست → botpay startup را کامل نمی‌کند
→ اجرای زنده‌ی e2e-provision و e2e-lockrental موکول به bot API قابل‌دسترس. هر دو `go build` سبز.
