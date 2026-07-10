# vpn-bot — چه چیزی نیاز داریم (بررسی ۲۰۲۶-۰۷-۰۴)

## وضعیت فعلی (خلاصه‌ی واقعی، نه قدیمی)

بررسی تازه انجام شد. سرویس شامل ۱۳ فایل Go و ~۲۱۶۵ خط است — کوچک‌تر و
ساده‌تر از uploader-bot، ولی معماری منسجمی دارد: `internal/vpn` (factory
برای پنل‌های VPN)، `internal/payment` (gateway کارت‌به‌کارت)،
`internal/scheduler` (اطلاع انقضا/غیرفعال‌سازی/همگام‌سازی مصرف)،
`internal/store` (GORM/Postgres)، `internal/tgbot`.

نکات مثبت:
- گیت ادمین به سبک allow-list (نه deny-by-default مثل uploader-bot) ولی هر
  مسیر ادمین (`panel_add`, `panel_test_all`, `ptype`, `panel_toggle`,
  `panel_del`, `approve_pay`, `reject_pay`) واقعاً `h.isAdmin(c)` را چک
  می‌کند — در `internal/tgbot/handler.go:191-244`. هیچ مسیر ادمینی بدون چک
  پیدا نشد.
- سه گیت‌وی پرداخت واقعی جدا (زرین‌پال، NowPayments، کارت‌به‌کارت) در
  `cmd/bot/main.go:135-145` واقعاً به آداپتورهای `shared/pkg/adapters/zarinpal`
  و `shared/pkg/adapters/nowpayments` وصل‌اند — نه فقط استاب.
- کامنت صادقانه‌ی «FIX 20: call Login() right after creation» و
  «FIX 16: start scheduler» در `cmd/bot/main.go:82,158` نشان‌دهنده‌ی رفع
  باگ‌های واقعی قبلی است.
- تنها ۱ گیت‌وی پرداخت هم‌زمان قابل‌فعال‌سازی است (`cfg.PaymentGateway`،
  یک env var) — این عمداً است، نه باگ؛ ولی روی این نکته در بند ۲ دقت شود.

## چیزی که واقعاً کم است (با file:line برای هر ادعا)

1. **factory چهار پنل VPN (marzban/marzneshin/hiddify/xui) کاملاً نوشته
   شده ولی در نقطه‌ی ورودی استفاده نمی‌شود — یعنی فقط Marzban واقعاً کار
   می‌کند.** `internal/vpn/factory.go:22-53` (`vpn.NewPanel`) از هر ۴ نوع
   پشتیبانی می‌کند (`SupportedPanels()` در خط ۵۶-۵۸ هم هر ۴ تا را لیست
   می‌کند)، ولی `cmd/bot/main.go:84-89` این factory را اصلاً صدا نمی‌زند:
   ```go
   switch cfg.PanelType {
   case "marzban":
       panel = marzban.New(cfg.PanelURL, cfg.PanelUser, cfg.PanelPass)
   default:
       log.Fatal("unknown PANEL_TYPE", ports.F("type", cfg.PanelType))
   }
   ```
   یعنی اگر کسی `PANEL_TYPE=marzneshin` یا `hiddify` یا `xui` تنظیم کند،
   سرویس بلافاصله با `log.Fatal` کرش می‌کند — با اینکه آداپتورهای واقعی آن‌ها
   (`shared/pkg/adapters/marzneshin`, `hiddify`, `xui`) در factory وارد
   و پیاده‌سازی شده‌اند. برخلاف ادعای CLAUDE.md («اتصال به
   Marzban/Marzneshin/Hiddify/XUI»)، فعلاً فقط Marzban واقعاً وصل‌شدنی است.
   رفع سریع: در `cmd/bot/main.go` به‌جای switch دستی، `vpn.NewPanel(cfg.PanelType,
   ...)` را صدا بزنید.

2. **کد تخفیف (DiscountCode) کاملاً ساخته شده ولی هرگز در مسیر خرید
   استفاده نمی‌شود.** مدل در `internal/models/models.go:78-85`، متدهای
   store در `internal/store/store.go:232-251`
   (`CreateDiscountCode`, `FindDiscountCode`, `UseDiscountCode`)، و UI ساخت
   کد تخفیف برای ادمین در `internal/tgbot/admin.go:284-313`
   (`handleDiscountInput`) — همه وجود دارند. ولی در کل مسیر خرید
   (`internal/tgbot/user.go:20-136`: `onBuy`, `onPlanSelected`,
   `onGatewaySelected`, `handlePaymentInput`) **هیچ‌جا `FindDiscountCode`
   یا `UseDiscountCode` صدا زده نمی‌شود** و هیچ مرحله‌ای برای وارد کردن کد
   تخفیف از کاربر وجود ندارد. یعنی ادمین می‌تواند کد تخفیف بسازد ولی هیچ
   کاربری هرگز نمی‌تواند از آن استفاده کند — یک قابلیت کاملاً یتیم.

3. **فیلدهای reseller (`ResellerID`, `Discount`) در مدل `User` تعریف شده‌اند
   ولی در کل کد استفاده نمی‌شوند.** `internal/models/models.go:31-32`. هیچ
   handler یا متد store‌ای این دو فیلد را می‌خواند یا می‌نویسد
   (grep کامل روی `internal` چیزی برنگرداند) — زیرساخت reseller طراحی‌شده
   ولی پیاده‌سازی نشده.

4. **UI انتخاب گیت‌وی سه دکمه نشان می‌دهد ولی فقط یکی واقعاً فعال است.**
   `internal/tgbot/keyboards.go:74-83` (`kbPaymentGateway`) همیشه سه دکمه
   «کارت به کارت»، «زرین‌پال»، «NowPayments» را نشان می‌دهد، ولی
   `cmd/bot/main.go:135-145` فقط یک `gateway` واحد بر اساس
   `cfg.PaymentGateway` می‌سازد. اگر ادمین `PAYMENT_GATEWAY=card` تنظیم کرده
   باشد، انتخاب «زرین‌پال» در `onGatewaySelected`
   (`internal/tgbot/user.go:115-119`) هنوز `h.gateway.CreatePayment` (که در
   واقع کارت است) را صدا می‌زند و پیام گمراه‌کننده «لینک پرداخت آنلاین»
   نشان می‌دهد. باید کیبورد را بر اساس گیت‌وی واقعاً فعال فیلتر کرد.

5. **صفر فایل تست** — هیچ `_test.go` در کل سرویس (برخلاف member-bot که
   حداقل یک تست دارد).

6. **خط تکراری بی‌اثر (dead code جزئی).** `internal/tgbot/admin.go:36-37`
   دو بار پشت‌سرهم `return h.adminPanels(ctx, c)` نوشته شده — خط ۳۷ کد
   مرده است (بعد از `return` هرگز اجرا نمی‌شود). بی‌خطر ولی نشانه‌ی
   copy-paste است، بهتر است پاک شود.

## این‌ها را در سرویس‌های دیگر هم نوشتم (اگر مرتبط بود)

یادداشت متقابل به `apimanager/NEEDS.md` اضافه شد: مدیریت پنل‌های VPN
(`internal/models/models.go:36-46` مدل `Panel`، و
`internal/tgbot/admin_panel.go` برای CRUD فعلی فقط از داخل خود ربات)
کاندیدای واضحی برای یک صفحه‌ی مدیریتی در وب apimanager است — مثلاً افزودن/
غیرفعال‌کردن پنل VPN بدون نیاز به وارد شدن به تلگرام. جزئیات آنجا نوشته شد.
