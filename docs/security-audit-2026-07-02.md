# گزارش امنیتی CreatorBotV3 — ۲ تیر ۱۴۰۵ (2026-07-02)

## خلاصه

این گزارش نتیجه‌ی یک بررسی امنیتی کامل روی هر ۱۸ سرویس پلتفرم است، به همراه سرویس جدید `license-service` (ضدکپی/ضدکلون instance_id). رویکرد کار: شبیه‌سازی نگاه یک مهاجم که فقط به همان چیزهایی دسترسی دارد که در دنیای واقعی هم می‌تواند داشته باشد — یک کلاینت NATS، یک اکانت تلگرام معمولی در هر ربات، یا شل داخل container ربات خودش (چون هر مشتری صاحب container ربات خودش است).

مهم‌ترین یافته: **یک باگ واحد که عملاً کل کیف‌پول پلتفرم را قابل تخلیه می‌کرد** — این و ۷ مشکل واقعیِ دیگر پیدا و در همین جلسه رفع شدند. جزئیات کامل هر کدام، همراه با نام فایل و شماره‌خط، پایین آمده.

نکته‌ی صداقت: سه مورد از یافته‌های گزارش‌شده توسط عامل‌های تحقیق (که برای سرعت، بررسی موازی انجام دادند) در بازبینیِ مستقیم من نادرست از آب درآمدند — کد در واقع از قبل درست بود. این‌ها صریحاً در بخش «یافته‌های نادرست» مشخص شده‌اند تا وقت شما روی چیزی که نیاز به تغییر ندارد تلف نشود.

---

## ۱. مسیر حمله‌ی اصلی: تخلیه‌ی کیف‌پول پلتفرم (Critical — رفع شد)

### داستان حمله

هر سرویس مرکزی (botmanager، ads-bot، ...) برای حرف زدن با `botpay` یک `PayRequest` می‌فرستد که شامل `service_id` (مثلاً `"botmanager"`) و `service_key` است. طبق کامنت خود کد، `service_key` قرار بود «کلید احراز هویت سرویس» باشد.

اما در `botpay/internal/payresponder/responder.go`، تابع `authorize()` این‌طور بود:

```go
func (r *Responder) authorize(ctx context.Context, serviceID string) bool {
    return r.wallet.Store().ValidateServiceID(ctx, serviceID)
}
```

و `ValidateServiceID` (در `botpay/internal/store/store.go`) فقط چک می‌کرد که `service_id` یکی از رشته‌های عمومی `"botmanager"`, `"apimanager"`, `"botpay"`, `"ads-bot"` باشد (یا `bot_<BotID>` با یک instance فعال) — **`service_key` هرگز خوانده یا مقایسه نمی‌شد.**

از آنجا که همه‌ی سرویس‌ها (و همه‌ی container های داینامیک ربات مشتری‌ها) یک `NATS_USERNAME`/`NATS_PASSWORD` مشترک دارند، این یعنی:

1. مهاجم فقط باید به NATS وصل شود (با همان یک پسورد مشترک که در همه‌ی `.env` ها هست).
2. یک پیام به subject `pay.credit` بفرستد با payload:
   ```json
   {"service_id": "botmanager", "telegram_id": <آی‌دی تلگرام خودش>, "amount_ton": 999999}
   ```
3. چون `"botmanager"` یک رشته‌ی کاملاً عمومی و قابل‌مشاهده در همین ریپازیتوری است، هیچ رازی لازم نبود.
4. `botpay` بدون چک واقعی، ۹۹۹٬۹۹۹ TON به کیف‌پول مهاجم اعتبار می‌داد.

همین مسیر برای `pay.deduct` (کسر از حساب هر کاربر دیگری) و `pay.transfer` هم باز بود.

### رفع

یک HMAC واقعی اضافه شد (`shared/pkg/auth.ComputeServiceKey`/`ValidateServiceKey`): هر سرویس مرکزی، کلید خودش را از یک راز مشترک (`SERVICE_HMAC_SECRET`، فقط در دست سرویس‌های مرکزی مورد اعتماد) و نام خودش می‌سازد؛ `botpay` همان محاسبه را انجام می‌دهد و مقایسه می‌کند. اگر رازها مطابقت نداشته باشند، رد می‌شود — حتی اگر `service_id` درست باشد. رفتار «fail closed»: اگر `SERVICE_HMAC_SECRET` تنظیم نشده باشد، همه‌ی درخواست‌ها رد می‌شوند (نه اینکه دوباره به حالت قدیمی برگردد).

علاوه بر این، دو باگ مرتبط هم در همان مسیر رفع شد:
- **مبلغ منفی**: هیچ‌جا چک نمی‌شد `amount_ton > 0`؛ یک «کسر» با مبلغ منفی عملاً اعتبار اضافه می‌کرد. حالا `validAmount()` مقدار را مثبت، متناهی، و زیر یک سقف معقول (۱ میلیون TON) نگه می‌دارد.
- **بدون idempotency**: `IdempotencyKey` گرفته می‌شد ولی هیچ‌جا چک نمی‌شد؛ یک retry شبکه‌ای (یا replay عمدی) می‌توانست یک پرداخت را دوبار انجام دهد. حالا `Deduct` قبل از کسر، یک تراکنش قبلی با همان `(service_id, ref)` را چک می‌کند.

فایل‌های تغییریافته: `shared/pkg/auth/auth.go`، `botpay/internal/payresponder/responder.go`، `botpay/internal/store/wallet_repo.go`، `botpay/internal/store/transfer.go`، `botpay/cmd/main.go`، `botmanager/cmd/main.go`، `ads-bot/cmd/main.go`.

---

## ۲. مسیر حمله‌ی دوم: حذف کل پلتفرم با یک پیام NATS (Critical — رفع شد)

### داستان حمله

`agentmanager` (که Docker container های واقعی را می‌سازد) وقتی دستور `stop`/`remove`/`restart` از NATS می‌گرفت، `container_id` را بدون هیچ چکی مستقیم به Docker SDK پاس می‌داد:

```go
case protocol.MsgRemove:
    out, execErr = dockerClient.Remove(ctx, cmd.ContainerID)
```

مهاجمی که (طبق مسیر بالا) به NATS دسترسی دارد، فقط کافی بود بفرستد:
```json
{"type": "remove", "container_id": "postgres"}
```
و `agentmanager` با `Force: true` آن container را حذف می‌کرد — حتی اگر در حال اجرا بود. همین برای `botpay`, `nats`, `agentmanager` خودش هم صادق بود. یک پیام = خرابی کل پلتفرم.

### رفع

یک label به نام `creatorbot.managed=true` روی هر container ای که خود `agentmanager` می‌سازد گذاشته می‌شود. قبل از هر `Stop`/`Remove`/`Restart`، ابتدا `ContainerInspect` چک می‌کند این label وجود دارد؛ اگر نه، عملیات رد می‌شود. یعنی حتی با دسترسی کامل به NATS، مهاجم فقط می‌تواند container های خودِ مشتری‌ها (که با همین مسیر ساخته شده‌اند) را دستکاری کند، نه زیرساخت پلتفرم را.

فایل تغییریافته: `agentmanager/internal/docker/client.go`.

---

## ۳. مسیر حمله‌ی سوم: ربودن webhook هر رباتی (Critical — رفع شد)

### داستان حمله

`webhook-gateway` قرار بود روت‌های `/internal/register`, `/internal/unregister`, `/internal/bots`, `/internal/stats` را پشت یک `InternalAuth` (کلید API) بگذارد. در `cmd/main.go`:

```go
engine.Group("/internal").Use(middleware.InternalAuth(cfg.InternalKey))
r.Register(engine)
```

و داخل `r.Register`، دوباره یک گروه دیگر (`engine.Group("/internal")`) ساخته می‌شد و روت‌ها رویش ثبت می‌شدند — بدون هیچ middleware ای، چون این دو `Group()` دو شیء کاملاً جدا از هم بودند (ویژگی gin که این‌طور کار می‌کند). نتیجه: **هیچ احراز هویتی روی این روت‌ها اجرا نمی‌شد.**

مهاجم فقط با یک `curl` به پورت HTTP این سرویس می‌توانست:
```
POST /internal/register {"token":"...", "bot_id": <هر ربات دلخواه>, "nats_subject": "webhook.<subject دلخواه>"}
```
و ترافیک webhook هر رباتی را به جای دلخواه خودش هدایت کند، یا با `/internal/unregister` تحویل webhook هر رباتی را قطع کند — بدون هیچ کلیدی.

### رفع

`Router.Register` حالا خودش `internalKey` می‌گیرد و مستقیم روی همان گروهی که route ها را ثبت می‌کند middleware را اعمال می‌کند — دیگر دو گروه جدا وجود ندارد. همچنین `WebhookRateLimit` که تعریف شده بود ولی هیچ‌جا وصل نشده بود، روی مسیر `/webhook/:token` وصل شد (جلوی این را می‌گیرد که یک ربات بدرفتار کل بودجه‌ی نرخ‌محدودی مشترک را مصرف کند).

فایل‌های تغییریافته: `webhook-gateway/cmd/main.go`، `webhook-gateway/internal/router/router.go`.

---

## ۴. توکن ربات‌های قفل، ذخیره‌ی متن‌خام در Mongo (High — رفع شد)

`uploader-bot` امکان «قفل با ربات» دارد: ادمین یک توکن ربات دوم وارد می‌کند تا برای چک عضویت استفاده شود (`ForceJoinChannel.BotToken`). برخلاف `BotInstance.BotToken` (اصلی، در shared-core) و `CheckBot.Token` (در member-bot) که هر دو صراحتاً با AES-256-GCM رمزنگاری‌شده ذخیره می‌شوند، این یکی به‌صورت متن‌خام در MongoDB ذخیره می‌شد. یعنی هرکسی که به Mongo دسترسی پیدا کند (backup لو رفته، دسترسی دیتابیس مشترک، ...) یک توکن ربات زنده و کارکردی به دست می‌آورد.

رفع شد: یک `EncryptKey` تا اینجا هرگز به `uploader-bot` نرسیده بود؛ از env (`ENCRYPTION_KEY`) تا `Handler` رشته‌کشی شد و توکن قبل از ذخیره با `auth.Encrypt` رمزنگاری می‌شود؛ اگر کلید تنظیم نشده باشد، ذخیره‌سازی رد می‌شود (نه اینکه ساکت متن‌خام ذخیره کند).

فایل‌های تغییریافته: `uploader-bot/internal/tgbot/locks_panel.go`، `uploader-bot/internal/core/app.go`، `uploader-bot/internal/tgbot/bot.go`، `uploader-bot/cmd/bot/main.go`.

---

## ۵. مسیرهای حمله‌ی پیدا و رفع‌شده در ربات‌های دیگر

| سرویس | مشکل | تأثیر | وضعیت |
|---|---|---|---|
| vpn-bot | `onCallback` هیچ چک ادمین روی مدیریت پنل/تأیید پرداخت نداشت | هر کاربر می‌توانست به خودش موجودی/اشتراک رایگان بدهد، یا پنل VPN دلخواه خودش را وارد پلتفرم کند | رفع شد |
| vpn-bot | IDOR در `link:`/`qr:` — بدون چک مالکیت | هر کاربری که UUID اشتراک کاربر دیگری را می‌دید (که در همان کیبورد خودش است) به لینک VPN زنده‌ی او دسترسی پیدا می‌کرد | رفع شد |
| member-bot | `onCallback` هیچ چک مالکیت/ادمین روی توقف/حذف قفل و تأیید پرداخت نداشت | هر کاربری با دانستن UUID یک قفل، می‌توانست قفل پولی کاربر دیگر را حذف کند | رفع شد |
| ads-bot | `deleteCampaign`/`pauseCampaign` بدون چک مالکیت؛ بازپرداخت به مهاجم | مهاجم کمپین رقیب را حذف و بودجه‌ی باقی‌مانده‌ی آن را به کیف‌پول خودش می‌ریخت | رفع شد |
| ads-bot | `verify_ch`/`reject_ch` بدون چک ادمین | هر کاربری می‌توانست کانال هر کسی (حتی خودش، بدون بررسی کیفیت) را verified کند | رفع شد |
| revenue-service | `earning.created` بدون auth/idempotency/اعتبارسنجی مبلغ؛ روت `/earn` هر کلید غیرخالی را می‌پذیرفت | پرداخت جعلی/دوباره روی رویداد replay-شده یا کلید نشت‌کرده | رفع شد |
| community-service | `campaign.revenue.generated` بدون اعتبارسنجی مبلغ/idempotency | همان کلاس مشکل بالا، یک قدم زودتر در زنجیره | رفع شد |

---

## ۶. یافته‌های نادرست (رفع نشد چون از قبل درست بود)

سه مورد از یافته‌های اولیه‌ی عامل‌های تحقیقاتی، در بازبینی مستقیم من (خواندن کامل فایل واقعی، نه فقط دیف) نادرست تشخیص داده شدند:

- **ادعا: `botmanager`'s onCallback هیچ چک ادمین ندارد.** غلط — `botmanager/internal/tgbot/router.go` از قبل یک گیت deny-by-default دارد (`isAdminOnlyAction` + یک map صریح، چک‌شده قبل از switch) که همه‌ی اکشن‌های حساس را می‌پوشاند.
- **ادعا: `apimanager` امضای HMAC ویجت لاگین تلگرام را چک نمی‌کند.** غلط — `apimanager/internal/handler/handler.go`'s `TelegramAuth` از قبل `verifyTelegramAuth` را با رفتار fail-closed و چک انقضای `auth_date` صدا می‌زند.
- **ادعا: `uploader-bot`'s onCallback هیچ چک ادمین ندارد.** غلط — `uploader-bot/internal/tgbot/callbacks.go` از قبل یک whitelist صریح از اکشن‌های عمومی + `isAdmin` روی بقیه دارد.

نیازی به کاری روی این سه مورد نیست.

---

## ۷. آنچه مستند شد ولی در این جلسه رفع نشد (برای اقدام بعدی)

- **بزرگ‌ترین مشکل ریشه‌ای باقی‌مانده**: NATS هیچ ACL روی سطح subject ندارد — همه‌ی ۱۸ سرویس یک username/password مشترک دارند. رفع اصلی این باید در سطح تنظیمات NATS (accounts/permissions per-service) انجام شود؛ خارج از دامنه‌ی این جلسه بود ولی مهم‌ترین قدم بعدی امنیتی پلتفرم است.
- همه‌ی فایل‌های `.env` واقعی (شامل توکن ربات‌ها، پسورد DB، `ENCRYPTION_KEY`، ...) در گیت commit شده‌اند و `.gitignore` ریشه‌ای وجود ندارد — باید rotate و از تاریخچه‌ی گیت پاک شوند.
- `CreateWithdraw` یک race شرطی روی `frozen` دارد (بدون row-lock هم‌زمان با چک موجودی).
- قفل فایل uploader-bot (رمز عددی کوتاه) هیچ محدودیت تلاش ندارد — قابل brute-force.
- جستجوی inline در uploader-bot یک regex خام از ورودی کاربر می‌سازد (ReDoS، قابل‌دسترس پیش از احراز هویت).

---

## ۸. سرویس جدید: license-service

سرویس جدید `license-service/` ساخته و به `docker-compose.yml`/`go.work` اضافه شد. کارش:

1. وقتی `botmanager` یک instance تازه می‌سازد، یک لایسنس امضاشده برای `instance_id` (=BotID) صادر می‌کند که به `ServerID` همان استقرار «چسبیده» است (`license.issue`، فقط با `SERVICE_HMAC_SECRET` معتبر).
2. توکن به‌عنوان `LICENSE_TOKEN` (و `SERVER_ID`) به container ربات تزریق می‌شود.
3. خودِ ربات هر ۶ ساعت با `license.verify` چک می‌کند هنوز روی همان سرور است.
4. اگر همان `instance_id` از یک `ServerID` دیگر check-in کند (یعنی کسی container را کپی و جای دیگری اجرا کرده)، یک رویداد `license.clone_detected` منتشر می‌شود — ولی لایسنس **خودکار باطل نمی‌شود** (fail-open، برای اینکه یک مشکل شبکه‌ی گذرا مشتری واقعی را قطع نکند). ابطال واقعی فقط با `license.revoke` دستی انجام می‌شود.

این یعنی همین حالا زیرساخت آماده است؛ قدم بعدی (اختیاری) این است که پنل ادمین `botmanager` رویداد `license.clone_detected` را subscribe کند و به ادمین/مالک اطلاع بدهد — این بخش پیاده نشد چون به رابط ادمین جدید نیاز داشت و خارج از دامنه‌ی این جلسه بود.

---

## ۹. وضعیت وریفای

هر ۲۸+ فایل تغییریافته با `gofmt` بررسی نحوی شد (بدون خطا). اجرای کامل `go build ./...` در sandbox این نشست ممکن نشد چون `go.work` نسخه‌ی ۱٫۲۵٫۰ Go می‌خواهد و شبکه‌ی sandbox اجازه‌ی دانلود آن toolchain را نداد — پیشنهاد می‌شود قبل از deploy واقعی، یک `go build ./...` (یا CI) کامل روی این تغییرات اجرا شود.
