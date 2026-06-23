# CONTEXT PROMPT — CreatorBotV3 / botmanager

> پرامتِ زمینه. هر خط یک واقعیت/قرارداد پروژه است. آزادانه ادیت کن و کل فایل را بده.

---

## نقش تو (دستور به مدل)

روی مونوریپوی Go «CreatorBotV3» کار می‌کنی. ماژول هدف `botmanager` (ربات تلگرام،
telebot v4) که control-plane پلتفرم است. `shared` و `shared-core` کتابخانه‌های مشترک
و قابل ویرایش‌اند. **قوانین سخت:**
- هیچ متنِ hardcode در کد نباشد؛ همه‌ی رشته‌های نمایش‌داده‌شده از `i18n` بیایند.
- انواع سرویس/ربات نباید در کد hardcode باشند؛ کاملاً data-driven از دیتابیس.
- بعد از هر تغییر `gofmt` بزن و یادآوری کن `go build ./...` سمت کاربر اجرا شود
  (اینجا دیپندنسی‌ها دانلود نمی‌شوند، پس build اینجا ممکن نیست).

---

## معماری کلی

- `botmanager` ربات تلگرامی است که کاربر با آن ربات اختصاصی می‌سازد/مدیریت می‌کند و
  ادمین پلتفرم را مدیریت می‌کند.
- ربات‌های کاربر به‌صورت کانتینر Docker روی سرورهای **agent** اجرا می‌شوند.
- ارتباط بین سرویس‌ها: NATS | داده: Postgres (gorm) | state/کش: Redis.

### botmanager فقط به این‌ها وابسته است
- **agent** (اجرای کانتینرها روی سرور + heartbeat/نتیجه روی NATS)
- **موجودی ولت** (از طریق botpay روی NATS)
- خودِ ربات **botmanager**

> ❌ هیچ مفهوم community / company / کمپین / تقلب در این سرویس لازم نیست — این‌ها در
> سرویس‌های دیگرِ پلتفرم جداگانه نوشته شده‌اند. این بخش‌ها باید از botmanager حذف شوند.

---

## مدل داده‌ی واقعی (تأییدشده از shared-core/models/models.go)

- `BotTemplate`: `Name, Type (string آزاد), ImageName, ImageTag, Description, IsActive, IsFree`.
  → **«سرویس» = `Type`** و **«تگ» = `ImageTag`**. چند template با یک `Type` و
  `ImageTag`های مختلف = همان «سرویس با چند تگ». `Type` آزاد است (نه enum) → پویا.
- `Plan`: `TemplateID` **deprecated/null** است؛ پلن دیگر به تمپلیت گره نمی‌خورد.
  محدودیت‌ها در `PlanBotLimit` (per-type: مثلا VPN=5, Uploader=3)؛ `MaxBots` فقط fallback.
- `BotInstance`: `OwnerID, TemplateID, ServerID, BotToken, ContainerName, ContainerID,
  BotID, DBSchema, Status, PlanID, LockMode, ExpiresAt`. Status: running/stopped/pending/
  error/deleted.
- پس برای «انواع پویا» فقط کافی است همه‌جا به‌جای switchِ hardcode، از
  `ListTemplates`/گروه‌بندی بر اساس `Type` و `ImageTag` استفاده شود.

## سرویس‌ها و تگ‌ها (مدل اصلی — مهم)

- «سرویس» = یک نوع ربات (مثل uploader). انواع سرویس **پویا** هستند: باید بشود بعداً یک
  سرویسِ ربات جدید را بالا آورد **بدون تغییر کد** botmanager. هیچ‌جا لیست ثابتِ
  vpn/uploader/... در کد نباشد.
- هر سرویس **چند تگ (نسخه/ورژن)** دارد. کاربر تگ موردنظرش را انتخاب و نصب می‌کند.
- برای هر سرویس **یک پنل** داریم، نه یک پنل به‌ازای هر تگ. مثال: سرویس uploader یک پنل
  دارد و کاربر داخل همان پنل نسخه‌ی تگی که می‌خواهد را نصب می‌کند.
- منبع حقیقت سرویس‌ها و تگ‌ها: دیتابیس (templates/services). UI باید از همان ساخته شود.

### تست سرویس
- باید بشود یک سرویس با تگ `test` ساخت و آن را تست کرد (مسیر تستِ سرویس‌ها لازم است).

---

## حذف لینک دعوت
- کل فیچر InviteLink باید حذف شود (هندلرها، state، کیبوردها، کلیدهای i18n، مدل اگر فقط
  همین‌جا استفاده شده، و مسیر `/start <token>`).

---

## راه‌اندازی (cmd/main.go)

- env: `BOT_TOKEN, OWNER_ID, LOCAL_BOT_API, POSTGRES_DSN, REDIS_*, NATS_*,
  ENCRYPTION_KEY, TON_*, BOTPAY_*`.
- زنجیره: Postgres+migrate → Redis → NATS → docker manager (NATS) → telebot → store →
  ton → natspayclient.
- subscriptions: `agent.*.heartbeat` و `agent.*.result`.

---

## جریان ساخت ربات (wizard)

`svc_create` → انتخاب **سرویس** (پویا از DB) → انتخاب **تگ** → انتخاب پلن → توکن →
تأیید → پرداخت/رایگان → `provision`.
`provision`: timeout → `SelectLeastLoadedServer` (فقط آنلاین) → template مطابق سرویس+تگ →
نام کانتینر → instance (pending) → رویداد NATS → JWT → `DeployCommand` →
`DeploySubject(serverID)`. در خطا → refund + failed.

---

## پنل ادمین (بعد از پاکسازی)

- نگه‌داشتنی: کاربران، ربات‌ها، سرورها، آمار، ارسال همگانی، **سرویس‌ها/تگ‌ها**، پلن‌ها، ولت.
- حذف‌شدنی: لینک‌ها، مالی، تقلب، کمپین، کامیونیتی.

---

## پرداخت
`natspayclient` با botpay: Balance/Deduct/DeductForService/RefundService/Credit/
CreateInvoice/SubscribeWalletUpdates. TON.

## سرورها
`Server.IsOnline` پیش‌فرض false؛ فقط heartbeat agent آن را true می‌کند؛
`SelectLeastLoadedServer` فقط آنلاین را انتخاب می‌کند.

---

## ساختار کد هدف (دسته‌بندی)

کد باید تمیز دسته‌بندی شود؛ مثلاً:
- `internal/handlers/` (هندلرهای تلگرام: user، admin، wizard، ...)
- `internal/keyboards/`
- `internal/state/`
- `internal/i18n/`
- `internal/services/` (منطق سرویس/تگ پویا)
> ساختار نهایی را با هم نهایی می‌کنیم.

---

## قراردادها
- بدون hardcode متن؛ همه از i18n.
- انواع سرویس پویا از DB.
- کوئری با id خالی/UUID صفر زده نشود (گارد در store موجود است).
- بدون شکستن build؛ gofmt + یادآوری build.

## وضعیت ریفکتور (به‌روزرسانی مداوم)

- ✅ فاز ۱ — حذف شد: کل InviteLink (admin_link.go، stepهای لینک، wizardPending،
  wizardStart، payloadِ /start، kbLinkLimit، linkLimitFromText، genToken، fmtLink،
  دکمه‌های منوی Links)؛ و بخش‌های community/finance/fraud/campaign (هندلرها، دکمه‌های
  منو، caseهای stub در router). نکته: مدل/متد InviteLink در shared-core **دست‌نخورده**
  ماند (ممکن است سرویس‌های دیگر استفاده کنند). کلیدهای i18nِ بی‌استفاده فعلاً مانده‌اند،
  در فاز ۴ پاک می‌شوند.
- ✅ فاز ۲ — سرویس/تگ پویا:
  - store جدید (shared-core): `ListServiceTypes`, `ListTemplatesByType`, `FindTemplateByTypeAndTag`.
  - جریان کاربر: سرویس (پویا از DB) → تگ (ImageTag) → پلن → توکن → تأیید → provision
    با `FindTemplateByTypeAndTag`. wizard state کلید `wkTag` گرفت.
  - ادمینِ تمپلیت: نوع سرویس حالا **متن آزاد** است (انواع پویا) — حذف
    kbServiceCreate/kbAdminBotType/kbBotType/botTypeFromText/serviceTypeLabel/planBotTypes/admin_type.
  - ادیتور محدودیت پلن و validation حالا انواع را از `ListServiceTypes` می‌خوانند.
  - کلید جدید i18n: `KeyServiceSelectTag`.
  - باقی‌مانده برای فاز ۴: switchِ نمایشِ نوع در userBotsList/instanceSettings هنوز
    vpn/uploader/... را hardcode دارد (برای نوع ناشناخته «سرویس» نشان می‌دهد — graceful).
- ✅ فاز ۳ — تست سرویس:
  - تگ `test` (ثابت `testTag`) از کاربر عادی در منوی تگ مخفی است؛ فقط ادمین می‌بیند/نصب می‌کند.
  - دپلوی تستی ادمین: در لیست تمپلیت‌ها هر تمپلیت دکمه‌ی 🧪 تست دارد →
    `tmpl_test:<id>` → درخواست توکن → `admin_test.go` بدون پلن/پرداخت instance را دپلوی می‌کند.
  - state جدید `stepAdminTestToken`؛ کلیدهای i18n: KeyBtnTest, KeyBtnNewTemplate,
    KeyAdminTestAskToken, KeyAdminTestDeployed.
- ✅ فاز ۴ — i18n کامل:
  - همه‌ی متن‌های نمایشی در همه‌ی فایل‌ها به i18n منتقل شد (keyboards, router,
    router_text, menu_extra, admin_server, admin_plan, admin_tmpl, wizard,
    user_plans, user_bot). ده‌ها کلید جدید در keys/fa/en اضافه شد.
  - switchهای نوعِ hardcode در userBotsList/instanceSettings حذف و پویا شد
    (نوع = tmpl.Type، آیکن = botTypeEmoji graceful). statusLabel به متد i18n تبدیل شد.
  - استثناها (عمدی): «فارسی/English» (language endonyms)؛ دو «رایگان» در admin_tmpl.go
    که منطقِ تشخیص ورودی‌اند نه متن نمایشی.
  - ⚠️ چون build محلی ممکن نیست، رشته‌های فرمت (%s/%d) باید با اجرای واقعی تست شوند.
- 🚧 بازطراحی UX (تصمیم‌ها: لحن صمیمی‌حرفه‌ای، badge خودکار، یادآور انقضا همین فاز):
  - سند طراحی: `UX_REDESIGN.md`.
  - ✅ ویزارد: نشانگر «مرحله X از ۴»، رد خودکار وقتی تک‌تگ، badge «🆕 جدید» (جدیدترین
    تگ، order = created_at desc) و «🔥 محبوب» (پلن میانی، anchor pricing).
  - ✅ کیف‌پول: مبالغ سریع شارژ (1/5/10 TON) + مبلغ دلخواه (helper مشترک walletCreateInvoice).
  - ✅ تمدید سرویس واقعی (`service_renew.go`): کسر از botpay (NATS) → تمدید ExpiresAt
    (store.UpdateInstanceExpiry) → اگر خاموش بود start به agent. دکمه‌ی تمدید در کارت تنظیمات.
  - ✅ یادآور انقضا (`reminders.go`): job پس‌زمینه (هر ۶ ساعت، بازه‌ی ۷۲ ساعت)، dedupe در Redis،
    پیام تمدید تک‌تپ؛ از main.go با `h.StartExpiryReminders(ctx)` استارت می‌شود.
    store جدید: `UpdateInstanceExpiry`, `ListInstancesExpiringBetween`.
  - ⏳ مانده (اختیاری/فاز بعد): نمایش صرفه‌جویی پلن‌های بلندمدت، صفحه‌بندی لیست‌های بلند، معرفی دوستان.
- ✅ فاز ۵ — دسته‌بندی به subpackageها (با محدودیت Go: متدهای یک type باید کنار هم باشند):
  - `internal/tgbot/i18n` (از قبل)، `internal/tgbot/state` (typeها/ثابت‌های Step + UserState)،
    `internal/tgbot/format` (توابع محضِ فرمت/آیکن: StatusIcon/StatusEmoji/BotTypeEmoji/Fmt*).
  - `Handler` و همه‌ی متدهایش الزاماً در `tgbot` ماندند (Go اجازه‌ی split متدهای یک type
    بین پکیج‌ها را نمی‌دهد). در tgbot لایه‌ی نازکِ alias گذاشته شد (`type step = state.Step`،
    `var statusIcon = format.StatusIcon`، ...) تا call siteها بدون churn بمانند و ریسک build صفر.
  - فایل‌های هندلر همچنان کوچک و موضوع‌محورند (user_*, admin_*, router*, wizard, service_renew,
    reminders, ...).

- 🚧 جداسازیِ واقعیِ admin/user به package (build‌گیت‌دار، چون اینجا compiler نیست):
  - ✅ گام ۱: package `internal/tgbot/core` ساخته شد — `Deps` (وابستگی‌ها) + همه‌ی helperهای
    مشترکِ exported (T/Btn/F/BotTypeLabel، Auth: LoadUser/IsOwner/IsAdmin/IsInAdminMode/
    SetAdminMode/GetOrCreateUser، State: GetState/SetState/SetStep/ClearState، AuditLog،
    کیبوردهای مشترک: B/KbUser/KbUserFull/KbAdmin/KbUserActions/KbBack/KbBackCancel/KbCancel/
    KbLanguage، SendMain، IsCancel). فعلاً import نشده، پس build فعلی را نمی‌شکند.
  - ⏳ گام ۲: `tgbot.Handler` به `struct{ *core.Deps }` تبدیل شود؛ تعریف‌های تکراری از
    bot.go/helpers.go/state.go/keyboards.go حذف و ارجاعات `h.store→h.Store`, `h.t(→h.T(`, ...
    با sed به‌روز شوند. (نیازمند build تو)
  - ✅ گام ۲ (انجام شد): `tgbot.Handler` حالا `*core.Deps` را embed می‌کند؛ تعریف‌های تکراری
    حذف شد؛ keyboards.go حذف شد (به core رفت)؛ همه‌ی ارجاعات به نام‌های exported (h.Store,
    h.T, h.KbAdmin, ...) با sed به‌روز شد. به‌علاوه **همه‌ی متدهای admin/user هم Exported شدند**
    (AdminUsersList, UserBotsList, WizardSelectType, ...) و call siteها (router) به‌روز شدند.
    رفع دو باگ: admin_test.go→admin_svctest.go (نام _test.go باعث می‌شد build عادی نادیده‌اش بگیرد)؛
    و h.bot→h.Bot.
  - ✅ گام ۳/۴/۵ (انجام شد): جداسازیِ کاملِ package‌ها:
    - `internal/tgbot/admin` (package admin، type `Admin struct{*core.Deps}`): admin_bot,
      admin_plan, admin_server, admin_stats, admin_tmpl, admin_svctest, admin_user,
      admin_user_action (AdminUserAction)، admin_menu (credit/broadcast/system از menu_extra).
    - `internal/tgbot/user` (package user، type `User struct{*core.Deps}`): user_bot, user_plans,
      wizard (+wizard_test), service_renew, reminders, user_menu (account/wallet/settings/lang).
    - `internal/tgbot` فقط routing: bot.go (`Handler struct{ *core.Deps; *admin.Admin; *user.User }`
      + NewHandler که هر سه را با deps مشترک می‌سازد)، router.go، router_text.go، state.go (aliasها).
    - متدها promote می‌شوند، پس router بدون qualifier `h.AdminX`/`h.UserX` را صدا می‌زند.
    - بررسی: gofmt تمیز، syntax سالم، importهای بلااستفاده صفر، هیچ ارجاع admin↔user متقابل،
      هر پوشه یک package. **اما build نهایی باید سمت کاربر تأیید شود.**
  - (تاریخچه‌ی قبلی — قبل از انجام، برای مرجع):
    نقشه‌ی دقیق برای ادامه:
    1) `internal/tgbot/admin` (package admin، type `Admin struct{ *core.Deps }`): فایل‌های
       admin_bot, admin_plan, admin_server, admin_stats, admin_tmpl, admin_svctest, admin_user،
       + AdminUserAction (از user_bot)، + بخش admin از menu_extra (AdminCreditStart/Execute,
       AdminBroadcastMenu, BroadcastStartText/Execute/Confirm, RunBroadcast(+broadcastRate),
       AdminSystemMenu, AdminSysInfo).
    2) `internal/tgbot/user` (package user، type `User struct{ *core.Deps }`): user_bot (منهای
       AdminUserAction)، user_plans، wizard، service_renew، reminders، + بخش user از menu_extra.
    3) tgbot.Handler = `struct{ *core.Deps; *admin.Admin; *user.User }` (نام typeها متفاوت تا
       تداخل embed نشود)؛ router بدون تغییر کار می‌کند چون متدها promote می‌شوند (h.AdminX/h.UserX).
    4) در فایل‌های admin/user: `fmtX→format.FmtX`، `statusIcon→format.StatusIcon`،
       `stepX→state.StepX`، `userState→state.UserState`؛ و importها اصلاح شوند (با build).
    5) state.go و helpers.go (aliasها) در tgbot می‌مانند چون router/handleStep ازشان استفاده می‌کند.

## نکات/اصلاحات اضافه (اینجا بنویس)
<…>
