# license-service (سرویس جدید — ساخته‌شده در جلسه‌ی ۲۰۲۶-۰۷-۰۲)

## این سرویس چیست
یک سرویس مرکزی تازه که هر `instance_id` (=BotID هر ربات ساخته‌شده) را به یک لایسنس امضاشده و «چسبیده به سرور» متصل می‌کند — هدف: تشخیص کپی/کلون‌شدنِ container یک ربات مشتری روی سروری خارج از کنترل `agentmanager`.

## مسئولیت‌ها
1. **صدور (`license.issue`)**: وقتی `botmanager` یک `BotInstance` تازه می‌سازد، بلافاصله یک لایسنس صادر می‌کند که BotID را به همان `ServerID`ای که رویش deploy شده «می‌چسباند». نتیجه (JWT امضاشده) به‌عنوان env var `LICENSE_TOKEN` به container ربات تزریق می‌شود.
2. **بررسی (`license.verify`)**: خودِ هر ربات هر ۶ ساعت با این subject چک می‌کند «من هنوز روی همان سروری‌ام که لایسنسم برایش صادر شده؟». اگر از یک `ServerID` دیگر check-in شود (یعنی کسی image/دیتا را کپی کرده)، `CloneFlagCount` بالا می‌رود و رویداد `license.clone_detected` منتشر می‌شود — ولی لایسنس **خودکار باطل نمی‌شود** (fail-open عمدی، تا یک جابه‌جایی مشروع یا مشکل شبکه‌ی گذرا، مشتری واقعی را قطع نکند).
3. **ابطال (`license.revoke`)**: فقط با یک فراخوانی دستی از سرویس‌های مرکزی مورد اعتماد.

## ارتباطات
- NATS request/reply: `license.issue`, `license.verify`, `license.revoke` (queue group `license-workers`).
- رویداد pub/sub: `license.clone_detected`.
- PostgreSQL مستقل خودش — فقط جدول `License` (BotID، InstanceID، OwnerID، KnownServerID، TokenHash، Status، CloneFlagCount، ...).
- احراز هویت `issue`/`revoke`: همان الگوی `SERVICE_HMAC_SECRET` که برای رفع باگ botpay ساخته شد — فقط `botmanager`/`agentmanager` مجازند. `verify` را خودِ container ربات صدا می‌زند که هرگز این راز مادر را نمی‌بیند؛ به‌جایش باید دقیقاً همان توکنی که در issue گرفته را ارائه بدهد (سرور هش آن را مقایسه می‌کند).

## یکپارچه‌سازی با بقیه‌ی پلتفرم
- `botmanager/internal/tgbot/user/wizard.go` (`Provision`) و `admin/admin_svctest.go` — بعد از ساخت instance، `license.issue` صدا زده می‌شود و `LICENSE_TOKEN`+`SERVER_ID` به `DeployCommand.EnvVars` اضافه می‌شود.
- هر ربات محصول (`uploader-bot` از طریق `shared-core/engine`؛ `vpn-bot`, `archive-bot`, `member-bot` مستقیم در `cmd/bot/main.go` چون از `engine` استفاده نمی‌کنند) یک `licenseclient.RunLicenseLoop` در پس‌زمینه اجرا می‌کند.

## ایرادها و نکات
- مثل بقیه‌ی سرویس‌های مرکزی، `AutoMigrate` مستقل خودش را دارد — همان الگوی «migration drift» که در کل پلتفرم شناخته‌شده است.
- فعلاً **هیچ رابط ادمینی برای دیدن رویداد `license.clone_detected` وجود ندارد** — این رویداد منتشر می‌شود ولی هیچ سرویسی (فعلاً) به آن گوش نمی‌دهد. قدم بعدی طبیعی: یک subscriber در `botmanager` که به ادمین/مالک پیام بدهد.
- چون این سرویس کاملاً تازه است، هنوز در محیط واقعی تست نشده — پیشنهاد می‌شود قبل از فعال‌سازی روی مشتریان واقعی، مسیر issue→deploy→verify یک‌بار به‌صورت end-to-end دستی تست شود.
- `LICENSE_SIGNING_SECRET` باید جدا از `SERVICE_HMAC_SECRET`/`ENCRYPTION_KEY` نگه داشته شود (طراحی عمدی) — در `.env` این سرویس یک مقدار نمونه قرار داده شده که باید قبل از استقرار واقعی rotate شود.
