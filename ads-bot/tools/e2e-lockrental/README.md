# e2e-lockrental

تست end-to-end مدل اقتصادی «اجاره‌ی قفل کانال» بدون تلگرام.

## چه چیزی را تست می‌کند
کل چرخه‌ی حیات یک `LockRentalCampaign` را با **متدهای واقعی `ads-bot/internal/store`**
و **botpay واقعی** (NATS + HMAC) می‌سنجد:

1. seed چند `FreeBotSlot` آزاد + ساخت کمپین `pending_review`
2. تأیید: کسر بودجه از خریدار (`pay.deduct`) → `active` → اتصال slot ها
3. join کاربر: `TryRecordJoinReward` (رزرو با تأخیر) + تست idempotency
4. fraud: `ReversePendingRewardByUser` → reward = `reversed`، بودجه برمی‌گردد
5. settlement: `settle_at` را به گذشته می‌بریم → `pay.credit` به کاربر → `settled`
6. پایان کمپین: `end_at` گذشته → `MarkRentalDoneIfFinished` → `done` + آزادسازی slot

## چرا این‌جا (نه در tools/ ریشه)
ابزار باید `ads-bot/internal/store` را import کند؛ قانون `internal/` گو اجازه‌ی import
از یک ماژول دیگر را نمی‌دهد، پس ابزار **داخل ماژول ads-bot** قرار دارد. زیر `tools/`
است نه `cmd/` تا `go run ./cmd/...` در `run.sh` دست‌نخورده بماند.

## پیش‌نیاز
- NATS بالا (`nats://localhost:4222`)
- botpay بالا و به NATS وصل (برای `pay.credit`/`pay.deduct`)
- Postgres دیتابیس `adsbot` (پیش‌فرض پورت 5434)
- `SERVICE_HMAC_SECRET` یکسان با botpay و ads-bot

نیازی به اجرای خودِ ads-bot نیست؛ ابزار منطق store-levelِ handler/scheduler را
بازتولید می‌کند. با `-emit-nats` رویدادها روی core NATS هم منتشر می‌شوند تا اگر
نمونه‌ای از ads-bot در حال اجرا باشد آن هم exercise شود (idempotency آن را بی‌خطر می‌کند).

## اجرا
```bash
cd ads-bot/tools/e2e-lockrental
go run . \
  -hmac "$SERVICE_HMAC_SECRET" \
  -dsn 'postgres://botuser:PASS@127.0.0.1:5434/adsbot?sslmode=disable'
```
خروجی مرحله‌به‌مرحله `PASS`/`FAIL` است. با `-cleanup=false` داده‌های تست باقی می‌مانند.
