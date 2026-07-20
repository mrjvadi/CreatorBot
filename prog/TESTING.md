# وضعیت تست CreatorBot V3

آخرین بازبینی: ۲۰۲۶-۰۷-۱۵.

## فرمان مرجع

    ./scripts/test-all.sh

این فرمان همه‌ی moduleهای دارای go.mod را پیدا و go test ./... را در هر module اجرا می‌کند. CI همچنین go vet ./... را برای هر module و baseline بدهی gofmt را بررسی می‌کند.

## معنی سبز بودن CI

CI فعلی compile و تست‌های موجود را تأیید می‌کند؛ پوشش کامل محصول را ثابت نمی‌کند. moduleهای source، uploader، apimanager، ads، community، image، license و log هنوز تست کامل production path ندارند.

## تست‌های موجود

- botmanager: helper ویزارد؛ مسیرهای parity همچنان نیازمند integration test زنده NATS هستند.
- apimanager: تست helperهای credential/lifecycle اضافه شد؛ handlerهای DB/NATS هنوز integration test ندارند.
- botpay: یک فایل store/model، بدون concurrency مالی.
- revenue: محاسبات/helper، بدون DB claim race.
- agentmanager: Docker/env policy، بدون queue delivery/ack.
- fraud: فرمول‌های scorer، بدون processor/NATS/Mongo.
- member: balancer dispatcher.
- admanager: parse/schedule مدل.
- shared: VPN adapters، configstore و rotation.
- shared-core: store helper؛ baseline اکنون build می‌شود.
- license-service: allowlist هویت سرویس برای callerهای مجاز، کلید نادرست و caller ناشناخته تست می‌شود؛ issue/revoke زنده NATS پوشش ندارد.
- webhook: فقط rate limiter.

## simulationهای tests/integration

فایل‌های botmanager/member/uploader/vpn در tests/integration مدل، store و handler مستقل خودشان را تعریف می‌کنند و production packageها را import نمی‌کنند. این‌ها specification simulation هستند، نه integration test کد فعلی. سبز ماندنشان تغییر wiring، auth یا schema production را تضمین نمی‌کند.

## E2E خارج از CI

tests/e2e/e2e_test.go خارج از هر go.mod است و توسط scripts/test-all.sh یا CI کشف نمی‌شود. همچنین botpay HTTP health/balance قدیمی را انتظار دارد، درحالی‌که API مالی فعلی NATS request/reply است. تا بازنویسی و module مستقل، این suite مرجع معتبر نیست.

## ابزارهای دستی

### tools/e2e-provision

pay credit/balance/deduct → license issue → deploy publish → انتظار agent result. نیازمند NATS، botpay، license، image-registry، agentmanager و image local واقعی است.

### ads-bot/tools/e2e-lockrental

store واقعی ads و botpay واقعی را برای reserve، duplicate join، fraud reversal، settlement و completion اجرا می‌کند. handler/subscriber production را end-to-end assert نمی‌کند و unit test محسوب نمی‌شود.

## اولویت تست بعدی

1. PostgreSQL concurrency: credit idempotency، earning claim، slot allocation.
2. contract test مشترک producer/consumer برای earning.created.
3. crash/retry بین payout legها و terminal state community/revenue.
4. source HMAC، freshness، replay و دو tenant هم‌نام cache.
5. deploy unauthorized/replay/policy و queue saturation.
6. Telegram webhook secret header و registration lifecycle.
7. license expiry/revoke/clone و image admission/tar validation.
8. provisioning/refund یکسان برای botmanager و apimanager.
9. انتقال E2E واقعی به module و job جدا با service containers.

## production proof

اجرای live Telegram، Docker و دیتابیس production در این بازبینی انجام نشد. قابلیت‌هایی مثل source MTProto، webhook واقعی، providerهای VPN و settlement مالی تا زمان اجرای fixture واقعی باید implemented/wired تلقی شوند، نه prod-proven.
