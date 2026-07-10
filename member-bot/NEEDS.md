# member-bot — چه چیزی نیاز داریم (بررسی ۲۰۲۶-۰۷-۰۴)

## وضعیت فعلی (خلاصه‌ی واقعی، نه قدیمی)

بررسی تازه انجام شد. این سرویس بسیار پیچیده‌تر از یک «چک‌کننده‌ی ساده‌ی
عضویت» است: ۲۰ فایل Go، ~۲۴۱۹ خط، شامل صف Redis Stream
(`internal/worker`: pool، rate limiter، reclaimer)، load balancer بین چند
check-bot (`internal/dispatcher/balancer.go`)، NATS responder برای
`member.check` (`internal/memberresponder`)، و یک HTTP API جدا برای قفل‌ها
(`internal/lock/server.go`). این با ادعای CLAUDE.md («زیرساخت متمرکز چک
عضویت با کش») همخوانی دارد و حتی فراتر از آن — معماری صف/pool واقعاً
production-grade طراحی شده.

نکات مثبت مهم:
- **این تنها سرویس از ۴ سرویس بررسی‌شده با فایل تست است**:
  `internal/dispatcher/balancer_test.go` (۲ تست واحد برای
  `selectLeastLoaded`).
- دو رفع باگ امنیتی خودمستندشده در کد (کامنت‌های صادقانه، نشانه‌ی ممیزی
  واقعی قبلی):
  - `internal/tgbot/owner.go:165-186` (`canManageLock`) — قبلاً
    `pauseLock`/`deleteLock` هیچ چک مالکیتی نداشتند؛ حالا مالک واقعی یا
    ادمین پلتفرم چک می‌شود.
  - `internal/tgbot/admin.go:90-101` (`approvePayment`) — قبلاً هیچ چک
    ادمین نبود؛ حالا `isAdmin` چک می‌شود.
- Reclaimer واقعاً در `dispatcher.Start` راه‌اندازی می‌شود
  (`internal/dispatcher/dispatcher.go:61-62`) — نه یک قابلیت نوشته‌شده و
  فراموش‌شده.
- `main.go:73-75` کامنت صادقانه دارد درباره‌ی یک باگ رفع‌شده (اتصال NATS
  قبلاً فقط در حالت webhook برقرار می‌شد، یعنی در polling
  `member.check`/license کار نمی‌کرد).

## چیزی که واقعاً کم است (با file:line برای هر ادعا)

1. **افزودن Check Bot جدید از تلگرام، در سرویس در حال اجرا اثر نمی‌کند
   (نیاز به ری‌استارت کامل دارد).** `internal/dispatcher/dispatcher.go:69-80`
   متد `Dispatcher.AddBot` را تعریف می‌کند که واقعاً worker جدید می‌سازد و
   با `d.pool.Add(ctx, w)` (`internal/worker/pool.go:34-43`) به pool در حال
   اجرا اضافه می‌کند — این زیرساخت کاملاً کار می‌کند. ولی
   `internal/tgbot/owner.go:257-297` (`handleBotToken`، جایی که ادمین توکن
   ربات جدید را در `/addbot` وارد می‌کند) فقط `h.store.CreateCheckBot(ctx,
   bot)` را صدا می‌زند و **هرگز `dispatcher.AddBot` را فراخوانی نمی‌کند** —
   چون `tgbot.Handler` اصلاً رفرنسی به `*dispatcher.Dispatcher` ندارد
   (`internal/tgbot/handler.go` قابل بررسی؛ در `cmd/bot/main.go:109-146`،
   `tgbot.NewHandler` و `dispatcher.New` کاملاً مستقل از هم ساخته می‌شوند و
   به هم پاس داده نمی‌شوند). نتیجه: بات جدید در Postgres ذخیره می‌شود ولی
   تا ری‌استارت کامل سرویس، هیچ worker‌ای برایش اجرا نمی‌شود و
   `syncLoop` (خط ۱۳۷-۱۵۶) فقط عضویت بات‌های از قبل بارگذاری‌شده را
   sync می‌کند، نه بات‌های تازه.
   رفع سریع: به `tgbot.Handler` یک رفرنس به dispatcher بدهید و در
   `handleBotToken` بعد از `CreateCheckBot` مستقیماً `dispatcher.AddBot`
   را هم صدا بزنید.

2. **مسیر «رزرو/پرداخت اجاره‌ی قفل» فقط نصفه پیاده شده — پول واقعاً جابه‌جا
   نمی‌شود.** مدل `Payment` (`internal/models/models.go:81-88`) و متدهای
   `CreatePayment`/`ApprovePayment`/`FindPendingPayments`
   (`internal/store/store.go:121-134`) وجود دارند، ولی:
   - **`CreatePayment` هرگز در `internal/tgbot` صدا زده نمی‌شود** — هیچ
     UI یا handler‌ای برای «مالک قفل، هزینه‌ی اجاره را پرداخت می‌کند»
     پیدا نشد؛ در نتیجه هیچ ردیف `Payment`ای اصلاً هرگز ساخته نمی‌شود.
   - حتی مسیر تنها موجود، `approvePayment`
     (`internal/tgbot/admin.go:92-101`)، فقط `status` را به `"confirmed"`
     تغییر می‌دهد و **هرگز `store.UpdateBalance` را صدا نمی‌زند** —
     یعنی حتی اگر یک ردیف Payment به هر طریقی ساخته شود، تأیید آن هرگز
     `Owner.Balance` را افزایش نمی‌دهد.
   - `store.UpdateBalance` (`internal/store/store.go:136-140`) در کل
     `internal/tgbot` هیچ فراخوانی ندارد — یک تابع کاملاً یتیم.
   جمع‌بندی: مکانیزم بالانس/پرداخت محلی این سرویس (تومان، در Postgres خودش)
   کاملاً جدا از قانون بنیادی پروژه («همه‌ی عملیات پولی از طریق
   `natspayclient` با `botpay`» — طبق CLAUDE.md بخش ۵) است و حتی در حد خودش
   هم کامل وصل نشده. باید تصمیم گرفته شود: یا این مسیر تومانیِ محلی حذف و
   به `pay.deduct`/`pay.credit` روی `botpay` وصل شود (مطابق معماری کلی
   پروژه)، یا دست‌کم زنجیره‌ی محلی (`CreatePayment` → `approvePayment` →
   `UpdateBalance`) کامل شود.

3. **مدل `MemberVerification` تعریف و migrate می‌شود ولی هرگز خوانده یا
   نوشته نمی‌شود.** تعریف در `internal/models/models.go:72-79`، migrate در
   `cmd/bot/main.go:61` (`&models.MemberVerification{}`) — ولی
   `grep -rn "MemberVerification" member-bot --include=*.go` فقط همین دو
   خط را برمی‌گرداند. یعنی یک جدول واقعی در Postgres ساخته می‌شود
   (`member_verifications`) که همیشه خالی می‌ماند — به نظر می‌رسد قرار
   بوده تاریخچه‌ی هر چک عضویت (کدام بات، چه زمانی، نتیجه) ثبت شود اما این
   بخش هرگز نوشته نشده.

## این‌ها را در سرویس‌های دیگر هم نوشتم (اگر مرتبط بود)

یادداشتی به `apimanager/NEEDS.md` اضافه شد: لیست/مدیریت قفل‌های کانال
(`internal/models/models.go:40-51`, مدل `Lock`) و check-bot‌ها
(`internal/models/models.go:54-61`, مدل `CheckBot`) کاندیدای مناسبی برای
apimanager هستند — مخصوصاً چون مالکان قفل ممکن است بخواهند از وب،
وضعیت قفل‌های پولی خود را بدون تلگرام ببینند. توجه: تا وقتی بند ۲ (پرداخت
جابه‌جا نمی‌شود) رفع نشود، بهتر است apimanager صرفاً «نمایش» را اضافه کند،
نه عملیات پرداخت را.
