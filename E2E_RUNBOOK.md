# E2E Runbook — CreatorBot V3

راهنمای اجرای تست‌های end-to-end. سه سطح تست داریم که هر کدام لایه‌ی متفاوتی را می‌سنجند.

---

## وضعیت زیرساخت روی این ماشین (۲۰۲۶-۰۷-۱۴)

| مؤلفه | وضعیت | جزئیات |
|---|---|---|
| Postgres مرکزی | ✅ بالا | `127.0.0.1:5432` |
| Postgres `adsbot` | ✅ بالا | `0.0.0.0:5434` |
| NATS | ✅ بالا | `127.0.0.1:4222` (`nats`/`nats_secret`) |
| Redis | ✅ بالا | `6379` و `6381` (ads-bot DB=4) |
| MongoDB | ✅ بالا | `127.0.0.1:27017` |
| سرویس‌های go (botpay و…) | ❌ خاموش | نه `.run.pids` نه پروسه‌ای |
| local-bot-api (`141.95.210.17:8081`) | ❌ غیرقابل‌دسترس | `curl` → `000` |

**نتیجه‌ی مهم:** خودِ زیرساخت داده بالاست، ولی `botpay` (و هر ربات تلگرام) در startup با
`tele.NewBot` به local-bot-api در `141.95.210.17:8081` وصل می‌شود و چون این سرور در
دسترس نیست، `getMe` شکست می‌خورد و سرویس `log.Fatal` می‌کند. پس اجرای واقعیِ تست‌هایی که
به `pay.*` نیاز دارند (harness اجاره‌ی قفل و e2e-provision) در این محیط ممکن نیست تا
یکی از این‌ها فراهم شود:
1. یک local-bot-api قابل‌دسترس (تغییر URL در `botpay/cmd/main.go:158` و ربات‌های دیگر)، یا
2. اجازه‌ی اتصال مستقیم به Bot API رسمی تلگرام (حذف `URL` از tele.Settings).

هر دو تست از نظر کد **کامپایل و آماده‌اند** (`go build` سبز)؛ فقط اجرای زنده به bot API
قابل‌دسترس گره خورده است.

---

## تست ۱ — harness اجاره‌ی قفل کانال (ads-bot)

کل مدل اقتصادی lock-rental را بدون تلگرام می‌سنجد. رجوع
`ads-bot/tools/e2e-lockrental/README.md`.

**پیش‌نیاز:** NATS + **botpay زنده** + Postgres `adsbot`(5434) + `SERVICE_HMAC_SECRET`.

```bash
cd ads-bot/tools/e2e-lockrental
go run . \
  -hmac "$SERVICE_HMAC_SECRET" \
  -dsn 'postgres://botuser:PASS@127.0.0.1:5434/adsbot?sslmode=disable'
```

انتظار: `PASS` برای seed → approve → joins → idempotency → fraud reversal →
settlement → completion، و در پایان `✅ کل چرخه‌ی اجاره‌ی قفل تأیید شد`.

نیازی به اجرای خودِ ads-bot نیست (ابزار منطق store-level را بازتولید می‌کند). با
`-emit-nats` رویدادها روی core NATS هم منتشر می‌شوند تا یک ads-bot زنده هم exercise شود.

---

## تست ۲ — e2e-provision (زنجیره‌ی ساخت ربات، بدون تلگرام)

`pay.credit/balance/deduct` → `license.issue` → `DeployCommand` → انتظار
`agent.<serverID>.result`. رجوع `tools/e2e-provision/main.go`.

**پیش‌نیاز:** botpay، license-service، image-registry، agentmanager زنده + image
لوکال `uploader:dev`.

```bash
cd tools/e2e-provision
go run . -hmac "$SERVICE_HMAC_SECRET" -bot-token "<UPLOADER_BOT_TOKEN>" \
  -server-id cbe9f282-06a4-4c23-83b2-fe52b8ff9e17
```

این ابزار قبلاً (۲۰۲۶-۰۷-۱۰) کل حلقه‌ی botpay-HMAC/license/deploy را تأیید کرده است.

---

## تست ۳ — مسیر A: خرید پلن و ساخت ربات از پنل تلگرام botmanager

**فقط دستی** — به کلیک واقعی روی پنل تلگرام نیاز دارد و نمی‌توان autonomous اجرا کرد.

### گام ۱ — بالا آوردن سرویس‌ها
```bash
./run.sh          # ترتیب: botpay → image-registry/license → fraud/revenue/community
                  # → member-bot → ads-bot → agentmanager → apimanager → botmanager
./run.sh stop     # توقف
```
سلامت‌سنجی از روی `.logs/*.log`: دنبال `botmanager started`, `heartbeat sent`,
`success:true` بگرد. (توجه: `botpay.log` ممکن است `TON poll skipped (transient)` بدهد —
toncenter بدون key؛ مانع `pay.deduct` نیست، فقط تشخیص واریز on-chain.)

### گام ۲ — پیش‌نیاز داده‌ای (از پنل ادمین، نه SQL)
wizard خرید به این‌ها در DB نیاز دارد؛ همه از پنل ادمین botmanager (کاربر `OWNER_ID`)
ساخته می‌شوند:
- **Server** با تگ مناسب (`AdminServerAdd`) — پلن رایگان سروری با تگ `free` می‌خواهد.
- **BotTemplate** فعال برای نوع سرویس (`AdminTemplateAdd` / `AdminAddFreeTemplate`).
- **Plan** مرتبط با آن نوع سرویس (`AdminPlanAdd`).

(روی این ماشین طبق `botmanager.log`، سرور `cbe9f282-…` و یک uploader template قبلاً
seed شده‌اند.)

### گام ۳ — اجرای تست از تلگرام
1. در ربات botmanager `/start` → منوی خرید → انتخاب نوع سرویس → پلن → تأیید.
2. پول از کیف‌پول کسر می‌شود (`pay.deduct` → botpay).
3. `BotInstance` رکورد و `service.creation.requested`/`agent.<serverID>.deploy` به
   agentmanager می‌رود.
4. agentmanager container واقعی می‌سازد → ربات بالا می‌آید → `agent.<serverID>.result`.

### نقاط verify
- رکورد `bot_instances` با `status=running`.
- `docker logs <container>` بدون restart loop.
- `servers.is_online=true` از heartbeat.
- در شکست هر مرحله → refund خودکار به خریدار (چک `wallet`/`transactions` در botpay).
