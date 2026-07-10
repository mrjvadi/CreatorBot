# Source Service

این سرویس یک **ورکر** برای اجرای دستورهای تلگرامی (از طریق MTProto UserBot) هست: بهش می‌گی "از فلان کانال اعضا رو استخراج کن" یا "از فلان بات فایل رو بگیر، کپشن رو عوض کن و بفرست"، اونم اجرا می‌کنه و جواب رو برمی‌گردونه. می‌تونی چند نمونه از این سرویس رو هم‌زمان بالا بیاری؛ هر کدوم یه شماره/ID مجزا داره.

## وضعیت

⚠️ منطق واقعی MTProto (`internal/telegram`) نوشته شده و از `gotd/td` استفاده می‌کنه. لایه NATS/DB (`internal/bus`, `internal/worker`, `internal/store`, `internal/botmanager`) در برابر کد **واقعی** `shared` و `shared-core` بازبینی و اصلاح شده — دیگه فرضی نیست. یه نکته‌ی مهم از همین بازبینی: **gotd/td حداقل Go 1.25 لازم داره** (تأییدشده روی چند نسخه‌ی مختلفش، نه فقط حدس) — `go.mod` و `Dockerfile` به‌روز شدن؛ اگه لوکال Go قدیمی‌تری داری باید آپدیت کنی.

جزئیات محدودیت‌های باقی‌مونده (بخش‌هایی از gotd/td که هنوز کامپایل واقعی نشدن، و قرارداد `botmanager` که پیاده نشده) پایین‌تر هست.

## دسترسی و اعتماد

source-service یه ابزار داخلی برای **سرویس‌های اصلی** هست، نه یه BotInstance مشتری‌محور مثل uploader/vpn/archive/member (که هرکدوم توکن Bot API خودشون رو دارن و توسط agentmanager/botmanager به‌صورت مدیریت‌شده deploy می‌شن). یعنی:

- فقط سرویس‌های مرکزی معتمد (در حال حاضر: botmanager) باید بتونن بهش دستور بدن. این کنترل دسترسی دو لایه‌ست:
  - **سطح NATS/شبکه** (عملیاتی، نه کد): subject های `worker.*` (دستورها) باید توی permission های اکانت NATS محدود بشن به سرویس‌های اصلی — این سرویس خودش این رو enforce نمی‌کنه.
  - **سطح اپلیکیشن**: `source.worker.register` و `source.worker.update` (ارتباط با botmanager) یه `service_id`/`service_key` می‌خوان، دقیقاً همون الگویی که `license.issue`/`pay.credit` توی `shared-core/protocol` استفاده می‌کنن.
- اطلاع‌رسانی به بات‌های کاربرمحور (همونایی که کاربر نهایی باهاشون کار می‌کنه) خیلی کم پیش میاد و مستقیم نیست: خود ورکر هیچ‌وقت مستقیم به یه بات کاربری پیام نمی‌ده؛ نتیجه رو به botmanager گزارش می‌ده (`source.worker.update`) و **botmanager** تصمیم می‌گیره که آیا و چطور به بات مربوطه اطلاع بده.

## معماری: Worker + Task

### هویت ورکر (از طریق botmanager)

هر نمونه از این سرویس با یه `LICENSE_KEY` بالا میاد. موقع start:

1. با NATS request-reply به `source.worker.register` (تعریف‌شده توی `shared-core/protocol`, نه فقط داخل این سرویس) وصل می‌شه و `{service_id, service_key, license_key}` می‌فرسته.
2. botmanager جواب می‌ده: `{success, worker_id, telegram: {app_id, app_hash, phone, session_key}}`. `session_key` یه کلید AES-256 (base64) هست که session تلگرام همین اکانت باهاش رمزنگاری می‌شه.
3. سرویس همون `worker_id` رو به‌عنوان شماره خودش نگه می‌داره و باهاش subject های اختصاصی خودش رو می‌سازه.

> ⚠️ **هنوز پیاده نشده جای دیگه‌ای که من ببینم:** ساختار request/reply و شکل داده‌ها (`shared-core/protocol/source_worker.go`) رو خودم اضافه کردم چون توی `shared-core` چیزی برای source-service نبود، ولی دقیقاً با همون الگوی امن (`ServiceID`/`ServiceKey`) که `license.issue`/`pay.credit` توی همون پکیج استفاده می‌کنن. باید خود botmanager هم این پکیج رو import کنه و بهش پاسخ بده — این سمت دیگه‌ی قرارداده که هنوز نوشته نشده.

هر ورکر هر ۳۰ ثانیه یه heartbeat هم منتشر می‌کنه (`worker.<id>.heartbeat` و `source.worker.heartbeat` از `shared-core/protocol`).

### مسیریابی دستورها (هم مستقیم، هم استخری)

| Subject | نوع | کاربرد |
|---|---|---|
| `worker.<id>.tasks` | Request-Reply مستقیم | وقتی حتماً باید همون ورکر خاص کار رو انجام بده (مثلاً چون فقط اون روی اون اکانت/کانال لاگینه) |
| `worker.pool.tasks` | Request-Reply + Queue Group `"workers"` | وقتی برات فرقی نداره کدوم ورکر انجامش بده — هر کدوم آزاد بود می‌گیرتش |

هر دو subject از همون `internal/task.Registry` سرویس می‌گیرن، پس رفتار دستورها فرقی نمی‌کنه.

### فرمت دستور (Task)

Request روی هر دو subject بالا:

```json
{"id": "abc123", "type": "extract_members", "payload": {"channel": "@some_channel"}}
```

`id` اختیاریه ولی برای دستورهایی که جوابشون فوری نیست (مثل `run_bot_command`، پایین‌تر توضیح داده شده) لازمه: همون id رو می‌دی، و هروقت جواب واقعی آماده شد (حتی اگه چند مرحله طول بکشه)، با همون id به botmanager گزارش می‌شه — این‌جوری می‌فهمی جواب برای کدوم دستور بوده.

Reply همزمان (همیشه این شکل، حتی روی خطا؛ `id` همون چیزیه که فرستادی):

```json
{"id": "abc123", "ok": true, "type": "extract_members", "data": {...}}
{"id": "abc123", "ok": false, "type": "extract_members", "error": "..."}
```

### دستورهای فعلی

| type | payload | کار |
|---|---|---|
| `extract_members` | `{"channel": "@channel"}` | لیست اعضای کانال/سوپرگروه |
| `fetch_edit_send` | `{"bot_username", "command", "caption", "dest_username"}` | به بات پیام/دستور می‌فرسته، منتظر فایل جواب می‌مونه، دانلودش می‌کنه، کپشن رو عوض می‌کنه، به مقصد می‌فرسته |
| `forward_message` | `{"source_channel", "dest_channel", "message_id"}` | فوروارد مستقیم یه پیام بین دو چت |
| `watch_channel` | `{"source_channel", "dest_channel"}` | **بلادرنگ**: از این به بعد هر پستی که `source_channel` بذاره خودکار برای `dest_channel` فوروارد می‌شه. جواب یه `watch_id` برمی‌گردونه. |
| `list_watches` | `{}` | لیست channel-watch های زنده‌ی همین ورکر |
| `remove_watch` | `{"watch_id"}` | خاموش کردن یه channel-watch |
| `watch_nats` | `{"subject", "dest_channel"}` | **بلادرنگ**: از این به بعد هر پیامی که روی این NATS subject بیاد، متنش برای `dest_channel` (کانال/بات/کاربر تلگرام) فرستاده می‌شه. جواب یه `watch_id` برمی‌گردونه. |
| `list_nats_watches` | `{}` | لیست nats-watch های فعال همین ورکر |
| `remove_nats_watch` | `{"watch_id"}` | خاموش کردن یه nats-watch |
| `run_bot_command` | `{"bot_username", "command"}` | برو کامند رو داخل بات بزن، فایلی که جواب می‌ده رو بگیر، توی آرشیو خودمون ثبتش کن، و نتیجه رو با همون correlation id به botmanager گزارش بده |
| `create_rule` | `{"trigger", "conditions"?, "action"}` | **موتور قانون عمومی** — پایین‌تر کامل توضیح داده شده |
| `list_rules` | `{}` | لیست قانون‌های فعال همین ورکر |
| `delete_rule` | `{"rule_id"}` | خاموش کردن یه قانون |

### مانیتور بلادرنگ یک کانال (watch_channel)

با `watch_channel` می‌گی: «هر وقت فلان کانال پست گذاشت، بفرستش برای فلان‌جا». این poll نیست — روی همون stream آپدیت‌های زنده‌ی MTProto (`OnNewChannelMessage`) گوش می‌ده و همون لحظه فوروارد می‌کنه.

- قانون‌ها توی جدول `channel_watches` هم ذخیره می‌شن (`phone`, `source_channel`, `dest_channel`, `active`)، پس با ری‌استارت ورکر از بین نمی‌رن: به‌محض این‌که telegram client دوباره authorize شد (`Client.Ready()`)، `internal/userbot.RestoreWatches` همه‌ی watch های فعال همون شماره رو دوباره زنده می‌کنه.
- `remove_watch` هم قانون زنده رو خاموش می‌کنه هم رکورد دیتابیس رو `active=false` می‌کنه (soft delete، نه حذف کامل).
- الان فقط عمل «فوروارد» پشتیبانی می‌شه؛ اگه بعداً خواستی به‌جای فوروارد کار دیگه‌ای بشه (مثلاً فقط نوتیف بدون فوروارد کامل)، جاش داخل `internal/telegram/watch.go` (تابع `forwardWatchedPost`) مشخصه.
- هر ورکر فقط watch های همون اکانت/شماره خودش رو می‌بینه؛ چون Postgres بین چند ورکر مشترکه ولی query بر اساس `phone` فیلتر می‌شه.

### پل زدن NATS به تلگرام (watch_nats)

همون ایده‌ی `watch_channel`، ولی منبعش یه NATS subject دلخواهه به‌جای یه کانال تلگرام: «هر پیامی که روی این subject اومد، بفرستش برای فلان‌جا». مثال:

```json
{"type": "watch_nats", "payload": {"subject": "source.files.registered", "dest_channel": "@my_alerts_channel"}}
```

- پیاده‌سازی: `internal/userbot/watch_nats.go` روی `subject` یه `ports.NATS.Subscribe` واقعی می‌زنه (fire-and-forget، نه request-reply) و توی callback متن payload رو مستقیم به‌عنوان پیام متنی به `dest_channel` می‌فرسته (`Client.SendText`، جدید).
- قانون‌ها توی جدول `nats_watches` ذخیره می‌شن، دقیقاً مثل `channel_watches`، و با `internal/userbot.RestoreNatsWatches` بعد از ری‌استارت دوباره زنده می‌شن.
- الان فقط «متن خام پیام رو به‌عنوان پیام تلگرام بفرست» پشتیبانی می‌شه؛ اگه بعداً خواستی به‌جای فرستادن متن خام یه کار دیگه‌ای بشه (مثلاً parse کردن payload و صدا زدن یه دستور دیگه)، جاش داخل `startNatsWatch` (همون فایل) مشخصه — دقیقاً همون الگوی extensible که برای بقیه‌ی دستورها هم داریم.

### اجرای کامند توی یک بات و گزارش به botmanager (run_bot_command)

```json
{"id": "abc123", "type": "run_bot_command", "payload": {"bot_username": "@some_file_bot", "command": "/get 42"}}
```

این دستور: کامند رو به بات می‌فرسته، منتظر فایل جواب می‌مونه، دانلودش می‌کنه، توی آرشیو خودمون (`archive_files`، همون جدول قبلی) ثبتش می‌کنه، و بعد به `source.worker.update` (`shared-core/protocol` — همون الگوی generic که خودت توضیح دادی: یه subject با tag های مختلف، همه با یه `id` مشترک همبسته می‌شن) گزارش می‌ده:

```json
{"id": "abc123", "tags": {"action": "bot_file_ready", "bot_username": "@some_file_bot", "archive_file_id": "...", "file_name": "...", "mime_type": "...", "file_size": 12345}}
```

⚠️ **نکته فنی مهم درباره‌ی «file_id»:** یه Bot-API file_id واقعی رو فقط با توکن همون بات می‌شه ساخت (از طریق Bot API، نه MTProto). این ورکر با MTProto به‌عنوان یه اکانت کاربر عمل می‌کنه، نه با توکن بات، پس نمی‌تونه خودش یه file_id واقعی بسازه. به‌جاش فایل رو توی آرشیو خودمون ثبت می‌کنه و `archive_file_id` رو گزارش می‌ده — botmanager (که توکن بات‌ها رو داره) باید با `source.files.get` فایل رو بگیره، از طریق Bot API آپلودش کنه تا file_id واقعی بسازه، و بعد با `source.files.cache` (که از قبل داریم) کش‌اش کنه. اگه توقع داشتی خود ورکر مستقیم file_id بده، این نیاز به یه مسیر جدا (اکانت به‌عنوان بات، نه UserBot) داره — بگو تا اونم اضافه کنم.

### موتور قانون عمومی (create_rule) — «هر چیزی که بخوام»

`watch_channel` و `watch_nats` هرکدوم یه ترکیب ثابت و از قبل کدنویسی‌شده‌ان (فوروارد / ارسال متن خام). `create_rule` عمومیه: یه قانون از سه تیکه‌ی داده‌محور (نه کد) تشکیل می‌شه — **trigger** (چی باعث اجرا می‌شه)، **conditions** (اختیاری، فیلتر قبل از اجرا)، **action** (چیکار بکنه) — و ذخیره می‌شه توی جدول `rules`، بدون نیاز به کد جدید یا ری‌دیپلوی برای هر ترکیب جدید.

**Triggerها:**

| type | config | فعال می‌شه وقتی |
|---|---|---|
| `channel_post` | `{"channel": "@x"}` | یه پست جدید توی این کانال بیاد |
| `nats_message` | `{"subject": "x.y.z"}` | یه پیام روی این NATS subject بیاد |

**Conditionها** (لیست، همه باید true باشن؛ اگه خالی باشه یعنی همیشه true):

| type | value | true می‌شه وقتی |
|---|---|---|
| `text_contains` | `"کلمه"` | متن رویداد شامل این کلمه باشه |
| `text_regex` | `"regex"` | متن با این الگو match بشه |
| `sender_is` | `"شناسه"` | فرستنده (اگه قابل‌تشخیص باشه) همین باشه |

**Actionها:**

| type | config | کار |
|---|---|---|
| `forward` | `{"dest": "@y"}` | فوروارد مستقیم پیام تریگرکننده (فقط برای trigger از نوع `channel_post`) |
| `send_text` | `{"dest": "@y", "template": "..."}` | ارسال متن (با template) به مقصد |
| `run_task` | `{"task_type": "...", "payload_template": "..."}` | **هر task دیگه‌ای که این ورکر داره رو اجرا کن** — همون کاری که خودت خواستی: «برو داخل فلان بات بده» یعنی همین‌جا `task_type: "run_bot_command"` یا `task_type: "fetch_edit_send"` می‌ذاری |

`template`/`payload_template` یه Go `text/template` معمولیه (نه یه زبان اسکریپتی — نمی‌تونه کد اجرا کنه، فقط می‌تونه فیلدهای رویداد رو بخونه): `{{.Text}}`, `{{.Sender}}`, `{{.SourceChannel}}`, `{{.Subject}}`, `{{.MessageID}}`. برای `run_task` که خروجی باید JSON معتبر باشه، برای متن‌هایی که ممکنه quote یا newline داشته باشن از `{{.Text | json}}` استفاده کن نه `"{{.Text}}"` مستقیم — وگرنه JSON خراب می‌شه (این مورد رو توی همین محیط تست کردم، هم نسخه‌ی خراب هم نسخه‌ی درست).

**مثال دقیقاً مطابق چیزی که خودت گفتی** («اگه فلان کانال پست گذاشت، برو داخل فلان ربات بده»):

```json
{
  "type": "create_rule",
  "payload": {
    "trigger": {"type": "channel_post", "config": {"channel": "@source_channel"}},
    "action": {
      "type": "run_task",
      "config": {
        "task_type": "run_bot_command",
        "payload_template": "{\"bot_username\":\"@my_bot\",\"command\":\"/process {{.Text | json}}\"}"
      }
    }
  }
}
```

چون `run_task` مستقیم روی همون `task.Registry` که همه‌ی تسک‌های دیگه (`extract_members`, `fetch_edit_send`, `watch_channel`, ...) روش ثبت شدن dispatch می‌کنه، هر تسکی که الان هست یا بعداً اضافه بشه، خودکار به‌عنوان action یه قانون هم قابل‌استفاده‌ست — بدون این‌که موتور rule engine نیاز به تغییر داشته باشه.

قانون‌ها توی جدول `rules` ذخیره می‌شن (trigger/conditions/action به‌صورت JSON خام، پس نوع جدید trigger/condition/action نیاز به migration نداره — فقط `internal/rules` نیاز به یه `case` جدید داره) و با `internal/userbot.RestoreRules` بعد از ری‌استارت دوباره زنده می‌شن، دقیقاً مثل بقیه‌ی watch ها.

### اضافه کردن دستور جدید

هر دستور فایل مستقل خودش رو داره توی `internal/userbot/` (مثلاً `watch_channel.go`)، نه یه فایل مشترک برای همه. برای اضافه کردن یکی جدید:

```go
// internal/userbot/your_new_task.go
package userbot

func (u *Userbot) handleYourNewTask(ctx context.Context, id string, raw json.RawMessage) (any, error) {
    // parse raw, انجام کار (احتمالاً از طریق u.tg.*)، برگردون نتیجه یا error.
    // اگه جواب واقعی async می‌رسه، همون id رو به botmanager.Report بده.
}
```

و توی `internal/userbot/userbot.go`، داخل `Register`، یه خط اضافه کن:

```go
reg.Register("your_new_task", u.handleYourNewTask)
```

نیازی نیست چیزی توی `internal/bus` یا `cmd/service/main.go` تغییر کنه — `task.Registry` کاملاً از transport جداست.

## اجرای چند ورکر هم‌زمان

هر نمونه فقط با `LICENSE_KEY` فرق می‌کنه (نمونه در `deploy/docker-compose.yml`: `worker-1` و `worker-2`). ورکرها همه به یه NATS/Postgres/Redis مشترک وصلن.

## ذخیره session (چرا دیتابیس، نه فایل/volume)

session تلگرام (MTProto) هر اکانت **توی Postgres و رمزنگاری‌شده (AES-256-GCM)** ذخیره می‌شه، نه یه فایل روی volume داکر. دلیلش ساده‌ست: اگه container/volume از بین بره یا دوباره ساخته بشه، فایل session هم از بین می‌ره و باید دوباره با کد وارد شی؛ ولی دیتابیس معمولاً جای پایدارتری‌ه و از بین نمی‌ره.

- جدول: `telegram_sessions` (`phone` یکتا، `encrypted` = بایت‌های رمزنگاری‌شده). هیچ‌جا session به‌صورت plaintext ذخیره نمی‌شه.
- کلید رمزنگاری (`session_key`، یه کلید AES-256 به‌صورت base64) از همون جواب `botmanager.workers.register` میاد — یعنی مدیریت کلید هم دست botmanager‌ه، نه یه env ثابت روی هر سرور.
- پیاده‌سازی: `internal/telegram/dbsession.go` (پیاده‌سازی interface استاندارد `session.Storage` از gotd/td روی Postgres). منطق رمزنگاری/رمزگشایی رو جدا توی همین محیط با یه تست مستقل (encrypt→decrypt، رد کردن کلید غلط، رد کردن کلید با طول اشتباه) تأیید کردم — این بخش fully verified هست، برخلاف بقیه‌ی کدهای gotd/td که پایین توضیح داده شده.

### لاگین دستی / بازیابی اضطراری (بدون نیاز به بالا بودن یه ورکر واقعی)

مسیر اصلیِ لاگین همینه که بالا گفته شد: ورکر با botmanager هماهنگ می‌شه و کد رو از NATS می‌گیره. ولی برای تست یا بازیابی اضطراری (مثلاً session یه اکانت خراب شده و نمی‌خوای منتظر بمونی) یه ابزار جدا هم هست:

```bash
# یه کلید رمزنگاری بساز (یا از botmanager بگیر)
go run ./cmd/login --gen-key

# لاگین و ذخیره در همون دیتابیسی که ورکرها می‌خونن:
docker compose run --rm login --app-id 12345 --app-hash xxxxxxxx --phone +989120000000 \
  --postgres-dsn "$POSTGRES_DSN" --session-key "$SESSION_ENCRYPTION_KEY"

# یا فقط برای تست آفلاین سریع، بدون دیتابیس (فایل محلی):
go run ./cmd/login --app-id ... --app-hash ... --phone ... --file
```

چون این ابزار پیش‌فرض session رو دقیقاً به همون شکل رمزنگاری‌شده و توی همون جدول Postgres ذخیره می‌کنه که ورکرهای واقعی می‌خونن (کلید یکسان از طریق `--session-key` بده)، اگه یه اکانت مشکل پیدا کرد، همین ابزار برای ریکاوری سریع کافیه — نیازی نیست منتظر یه چرخه کامل botmanager بمونی.

## ثبت فایل آرشیو (باقی‌مانده از قبل)

سرویس همچنان یه registry ساده برای فایل‌های آرشیوشده داره (جدول‌های `archive_files` و `bot_file_caches` در Postgres، کش در Redis):

| Subject | Request | Reply |
|---|---|---|
| `source.files.register` | `{message_id, file_type, file_name, mime_type, file_size, caption}` | `{ok, file}` |
| `source.files.get` | `{uuid, bot_token_hash?}` | `{ok, file, cached_file_id?}` |
| `source.files.cache` | `{uuid, bot_token_hash, file_id}` | `{ok}` |

Event مرتبط: `source.files.registered` بعد از ثبت موفق.

## ENV های لازم

```env
POSTGRES_DSN=
REDIS_ADDR=
REDIS_PASSWORD=
REDIS_DB=
NATS_URL=nats://nats:4222

LICENSE_KEY=          # مخصوص همین نمونه — از botmanager می‌گیری
SERVICE_ID=source-service   # پیش‌فرض همینه، معمولاً لازم نیست ست کنی
SERVICE_KEY=          # اعتبار سرویس اصلی، برای source.worker.register/update

# پایین‌تر فقط برای cmd/login (تست/بازیابی دستی) لازمن، نه برای ورکر واقعی:
SESSION_ENCRYPTION_KEY=
SESSIONS_DIR=/app/sessions
```

دیگه به `TG_APP_ID` / `TG_APP_HASH` / `TG_PHONE` / `WORKER_ID` نیازی نیست — همه از botmanager با `LICENSE_KEY` میان (`session_key` هم همین‌طور).

### ورود اولیه به تلگرام (کد لاگین)

بار اول که یه ورکر با یه اکانت جدید بالا میاد، تلگرام یه کد می‌فرسته. چون این سرویس headless هست (بدون ترمینال تعاملی)، این کد از طریق NATS request-reply گرفته می‌شه:

- Subject: `worker.<id>.auth.code`
- هر چیزی (مثلاً یه پنل ادمین یا botmanager) که کد رو داره باید به این subject جواب بده: `{"code": "12345"}`

## محدودیت‌های شناخته‌شده این پیاده‌سازی

- **botmanager**: قرارداد `source.worker.register` / `source.worker.update` (تعریف‌شده توی `shared-core/protocol/source_worker.go`) از سمت source-service پیاده‌ست، ولی خود **botmanager هنوز این پکیج رو import و بهش پاسخ نمی‌ده** — این نیمه‌ی دیگه‌ی قراردادیه که جای دیگه‌ای نوشته نشده.
- **shared/shared-core**: دیگه فرضی نیست — `ports.NATS` وجود نداشت (به‌جاش `*natsclient.Client` واقعی از `shared/pkg/adapters/nats` استفاده شد، با متدهای واقعی `Request`/`Respond`/`QueueRespond`/`Subscribe`/`PublishCore`)، `ports.DB` نیاز به `.Conn()` داشت (`internal/store` اصلاح شد)، و `ports.Logger`/`Field`/`F` دقیقاً مطابق فرض قبلی از آب درومدن. جزئیات کامل توی حافظه‌ی cross-service ثبت شده.
- **gotd/td نیاز به Go 1.25+ داره** (تأییدشده، نه حدس) — `go.mod` و `Dockerfile` به‌روز شدن. کدهای `internal/telegram` بر اساس API واقعی gotd/td v0.159.0 نوشته شدن؛ به خاطر محدودیت دانلود toolchain جدید توی محیط من، build نهایی این بخش خاص رو خودت باید لوکال (با Go 1.25+) تأیید کنی. جاهایی که بیشتر احتمال داره اسم/شکل فیلد کمی فرق کنه: `internal/telegram/fetch_from_bot.go` (استخراج مدیا از پیام، آپلود/دانلود)، `internal/telegram/members.go` (variant های `ChannelParticipant`)، و `OnNewChannelMessage` روی `tg.UpdateDispatcher` (قیاسی از `OnNewMessage`).
- **fetch_edit_send** فقط کپشن رو "ادیت" می‌کنه؛ اگه منظور ادیت واقعی فایل (کراپ/واترمارک/…) باشه، جای اضافه کردنش داخل `FetchEditSend` بین دانلود و آپلود مشخص شده.
- ورکر همزمان فقط یک درخواست «منتظر جواب یک بات مشخص» رو پشتیبانی می‌کنه (`internal/telegram/waiter.go`، مشترک بین `fetch_edit_send` و `run_bot_command`) — برای همزمانی بیشتر باید اون waiter رو به لیست/صف تبدیل کرد.
- channel-watch/nats-watch فقط «فوروارد» یا «ارسال متن خام» رو پشتیبانی می‌کنن، نه ادیت/فیلتر محتوا قبل از ارسال — جای اضافه کردنش مشخصه (بالا گفته شد).
- **run_bot_command** فایل رو فقط توی آرشیو خودمون ثبت می‌کنه، نه اینکه خودش file_id واقعی بسازه — دلیلش فنی‌ه و بالا توضیح داده شده.
- **موتور قانون (`create_rule`)**: فقط دو trigger (`channel_post`, `nats_message`) و سه action (`forward`, `send_text`, `run_task`) و سه condition ساده داره. الگو extensible‌ه (یه `case` جدید توی `internal/rules`)، ولی چیزایی مثل «زمان‌بندی/schedule»، «OR/NOT توی condition ها»، یا trigger های دیگه (مثلاً پیام خصوصی/گروه، نه فقط کانال) هنوز نیستن.
- `internal/rules`، برخلاف بقیه‌ی این سرویس، فقط از stdlib (`text/template`, `regexp`) استفاده می‌کنه، هیچ وابستگی به gotd/td یا shared نداره جز `ports.NATS`/`ports.Logger` — پس این بخش رو واقعاً می‌تونستم توی همین محیط تست کنم و کردم (رندر template، escape کردن JSON با `| json`).

### گام بعدی پیشنهادی: پوشش بیشتر MTProto

موتور قانون همون‌قدر قدرتمنده که primitive های زیرش (`internal/telegram`) اجازه بدن. الان فقط extract members / fetch از بات / forward / send text / watch داریم. برای این‌که واقعاً «هر کاری» از طریق `run_task` یا مستقیم قابل‌انجام بشه، مرحله‌ی بعدی که خودت هم به‌عنوان اولویت دوم انتخاب کردی افزودن primitive های بیشتره: جوین/لفت کانال، میوت/آنمیوت، ری‌اکشن، ادیت/حذف پیام، دانلود/آپلود انواع مدیا (ویس، ویدیو، عکس با کیفیت‌های مختلف)، مدیریت اعضا (بن/کیک/ارتقا)، جستجو و گرفتن تاریخچه، نظرسنجی، و resolve کردن لینک‌های دعوت. هرکدوم از این‌ها یه متد جدید روی `telegram.Client` (فایل خودش، طبق همون قاعده‌ی فایل‌های کوچیک) + یه task جدید توی `internal/userbot`‌ه — همون الگوی فعلی، فقط بیشتر.

## توجه

استفاده از UserBot API ممکن است با شرایط خدمات تلگرام در تضاد باشد.
فقط برای archive/automation شخصی استفاده کنید.
