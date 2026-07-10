# botmanager — چه چیزی نیاز داریم (بررسی ۲۰۲۶-۰۷-۰۴)

> این سرویس از آخرین باری که مستند شد (`apimanager/PROJECT_UNDERSTANDING.md`) رشد کرده: از ~۳۰ به ۳۶
> فایل / ~۷۱۷۰ خط. دو قابلیت تازه که آن‌جا اصلاً ذکر نشده بودند: **کدهای پروموشن** و **مدیریت
> source-service worker ها**. این فایل هم آن‌ها را مستند می‌کند، هم چیزهایی که واقعاً کم است.

## قابلیت‌های تازه‌ی کشف‌شده (باید به مستندات مرکزی هم اضافه شوند)

- **کدهای پروموشن** (`internal/tgbot/admin/admin_promo.go` + `user/promo.go`): ادمین کد پرومو
  می‌سازد/لیست/فعال‌غیرفعال/حذف می‌کند؛ کاربر با یک دستور آن را redeem می‌کند. مدل در
  `shared-core/models` (`PromoCode` یا مشابه). دقیقاً همان چیزی که در `apimanager`'s
  `GET/POST/PATCH/DELETE /admin/promo-codes` و فرانت `AdminPromoCodes.tsx` هم پیاده شده — یعنی این
  فیچر الان از هم مسیر تلگرام هم مسیر وب در دسترس است.
- **مدیریت Source Workers** (`internal/tgbot/admin/admin_sourceworker.go`): CRUD روی
  `shared-core/models.SourceWorkerConfig` — لیست/toggle/حذف/افزودن worker (که هر کدام یک اکانت واقعی
  تلگرام برای `source-service` است، رجوع به `source-service/NEEDS.md`). پاسخ‌دهی واقعی NATS این
  قرارداد در `internal/sourceworker/responder.go` است (پکیج جدا، چون به `tele.Context` نیاز ندارد).

## چیزی که واقعاً کم است

1. **حلقه‌ی گم‌شده‌ی گزارش‌دهی source-worker (بحرانی برای آن فیچر، مستند در `shared/PENDING_CHANGES.md`
   و `source-service/NEEDS.md`)**: `internal/sourceworker/responder.go`'s `handleUpdate` یک
   correlation ID از source-service می‌گیرد ولی هیچ جدولی برای نگاشت آن به یک owner/chat در botmanager
   ندارد — فعلاً فقط لاگ می‌شود. تا این ساخته نشود، نتیجه‌ی کارهای چندمرحله‌ای worker ها (مثلاً
   «فایل را از فلان ربات بگیر») هرگز به کاربری که درخواستش داده نمی‌رسد.
   **پیشنهاد عملی**: یک جدول ساده `SourceWorkerTask{ID (=correlation id), OwnerTelegramID, ChatID,
   CreatedAt}` در `shared-core/models` + یک متد `CreateSourceWorkerTask`/`FindSourceWorkerTask` در
   `shared-core/store`؛ جایی که وظیفه اول صادر می‌شود (هرجا آن باشد — این‌جا هم پیدا نشد که کجاست؛
   خودش هم بخشی از این کار ناتمام است) یک رکورد بسازد، و `handleUpdate` با همان ID جستجو و پیام را به
   `ChatID` بفرستد.
2. **مسیر صدور اصلیِ task برای source-worker هنوز مشخص نیست** — یعنی حتی مشخص نیست چه چیزی در
   botmanager قرار است اول یک task به یک worker بدهد (از کجا در پنل ادمین/کاربر این‌کار آغاز می‌شود؟).
   این باید قبل از نیاز شماره ۱ روشن شود، وگرنه جدول correlation-id بدون یک producer واقعی بی‌فایده است.
3. **بدون CORS مرتبط نیست به این سرویس** (مال apimanager است، رجوع `apimanager/NEEDS.md`) — ولی چون
   `botmanager` تنها منبع حقیقتِ `BotInstance`/`Plan`/... است که هم پنل تلگرام هم پنل وب از آن
   می‌خوانند، هر schema جدیدی که این‌جا اضافه می‌شود (مثل `PromoCode`, `SourceWorkerConfig`) باید در
   `apimanager`'s API هم منعکس شود تا وب از عقب نماند — فعلاً هماهنگ است (بررسی شد، `apimanager` هر دو
   را دارد)، ولی این هماهنگی نیاز به انضباط مداوم دارد، نه یک تضمین ساختاری.
4. **تنها یک فایل تست** (`internal/tgbot/user/wizard_test.go`) برای کل این سرویس بزرگ (۳۶ فایل) —
   خصوصاً منطق مالی/provisioning (`Provision` در wizard.go) که پول واقعی جابه‌جا می‌کند، تست بیشتری
   نیاز دارد.

## به‌روزرسانی ۲۰۲۶-۰۷-۰۶: جداسازی دیتابیس
دیتابیس `botmanager` دیگر با `botpay`/`ads-bot`/`community-service`/`revenue-service`/
`license-service`/`image-registry` مشترک نیست — هرکدام حالا دیتابیس مخصوص خودش را دارد (همان سرور
Postgres، رجوع `deploy/migrations/000_create_databases.sql`). `botmanager` فقط با `apimanager` همان
دیتابیس (`botmanager`) را مشترک نگه داشت — عمدی، چون هر دو دقیقاً همان مدل‌های `shared-core` را
می‌خوانند/می‌نویسند (رجوع بند بالای همین بخش درباره‌ی هماهنگی schema با apimanager).

## این‌ها را در سرویس‌های دیگر هم نوشتم
- `source-service/NEEDS.md` — همان نیاز شماره ۱ (طرف دیگر همین قرارداد).
