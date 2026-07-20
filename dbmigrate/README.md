# dbmigrate — سرویس متمرکز migration ورژن‌دار

این سرویس مشکل «migration ناهماهنگ بین سرویس‌ها» (CLAUDE.md بخش ۸) را حل
می‌کند: به‌جای این‌که هر سرویس فقط با AutoMigrate در startup خودش schema
بسازد (بدون ورژن، بدون تاریخچه)، حالا **هر سرویس Postgres دار یک پوشه‌ی
migration ورژن‌دار** دارد و با یک دستور می‌شود دیتابیس هر سرویس را روی هر
نسخه‌ای ساخت/به‌روز کرد.

## سرویس‌های تحت پوشش

همان ۸ سرویس Postgres دار پلتفرم (رجوع `internal/migrate/registry.go`):

| سرویس | دیتابیس | توضیح |
|---|---|---|
| botmanager | botmanager | مشترک با apimanager (`-service apimanager` هم همین را می‌گیرد) |
| botpay | botpay | |
| ads-bot | adsbot | |
| community-service | community | |
| revenue-service | revenue | |
| license-service | license | |
| image-registry | imageregistry | |
| source-service | source_svc | |

سرویس‌های Mongo (uploader-bot, vpn-bot, archive-bot, member-bot, fraud-engine,
log-collector, admanager-bot) این‌جا نیستند — schema ندارند، هر سرویس
ایندکس‌های خودش را در startup می‌سازد (`EnsureIndexes()`). vpn-bot/archive-bot/
member-bot تا ۲۰۲۶-۰۷-۱۷ Postgres داشتند — تاریخچه‌ی baseline در
`migrations/{vpn-bot,archive-bot,member-bot}/` نگه داشته شده (رجوع `RETIRED.md`
هرکدام)، ولی دیگر تحت پوشش این ابزار نیستند. agentmanager و webhook-gateway
دیتابیس ندارند.

## دستورها

```bash
export POSTGRES_DSN="postgres://botuser:pass@127.0.0.1:5434/botmanager?sslmode=disable"
# اسم دیتابیس داخل DSN مهم نیست — برای هر سرویس خودکار عوض می‌شود.

go run ./cmd list                                  # سرویس‌ها و آخرین نسخه‌ی هرکدام
go run ./cmd status -service all                   # هر دیتابیس روی چه نسخه‌ای است
go run ./cmd up -service ads-bot                   # تا آخرین نسخه (دیتابیس نبود؟ می‌سازد)
go run ./cmd up -service ads-bot -version 2        # دقیقاً تا نسخه‌ی ۲
go run ./cmd up -service all                       # همه‌ی سرویس‌ها تا آخرین نسخه
go run ./cmd mark -service botmanager -version 1   # ثبت بدون اجرا (پایین توضیح داده شده)
go run ./cmd new -service botpay -name add_x       # ساخت فایل نسخه‌ی بعدی
```

## اضافه‌کردن نسخه‌ی جدید (گردش کار اصلی)

وقتی مدل یک سرویس را عوض می‌کنید و می‌خواهید تغییر schema ورژن‌دار باشد:

```bash
cd dbmigrate
go run ./cmd new -service botpay -name add_refund_reason
# فایل migrations/botpay/0002_add_refund_reason.sql ساخته می‌شود — SQL را بنویسید
go run ./cmd up -service botpay -version 2
```

قواعد:
- **فایل اعمال‌شده را هرگز عوض نکنید** — checksum هر نسخه موقع اعمال در
  `schema_migrations` ثبت می‌شود و اگر فایل بعداً فرق کند، `up` با خطا
  می‌ایستد. اشتباه کردید؟ نسخه‌ی جدید بسازید که اصلاحش کند.
- هر فایل کامل در **یک تراکنش** اجرا می‌شود — یا کل نسخه اعمال می‌شود یا هیچ.
- down migration نداریم (عمداً) — برگشت به عقب یعنی نسخه‌ی جدیدِ جبرانی.
- فایل‌ها با `go:embed` داخل باینری‌اند؛ بعد از اضافه‌کردن فایل، build جدید لازم است.

## baseline (نسخه‌ی ۱) از کجا آمده؟

از حدس یا بازنویسی دستی نه — ۲۰۲۶-۰۷-۱۰ برای هر سرویس، AutoMigrate واقعی
خودش (همان لیست مدل‌هایی که در `cmd/main.go` هر سرویس است) روی یک دیتابیس
خالی اجرا و خروجی با `pg_dump --schema-only` گرفته شد. یعنی baseline دقیقاً
همان schema ای است که GORM از مدل‌های فعلی می‌سازد.

## mark چیست و کی لازم است؟

`mark` نسخه را **بدون اجرای SQL** به‌عنوان اعمال‌شده ثبت می‌کند. برای
دیتابیسی که schema اش از قبل وجود دارد (چون سرویس با AutoMigrate ساخته)
baseline را نباید اجرا کرد (جدول‌ها هستند و خطا می‌گیرید) — فقط ثبتش کنید:

```bash
go run ./cmd mark -service botmanager -version 1
```

(برای دیتابیس‌های dev فعلی این کار ۲۰۲۶-۰۷-۱۰ انجام شده — botmanager و
botpay mark شدند، بقیه که خالی بودند واقعاً با `up` ساخته شدند.)

## رابطه با AutoMigrate فعلی سرویس‌ها

AutoMigrate در startup سرویس‌ها فعلاً سر جایش است (additive و بی‌خطر است و
container های محصول به آن متکی‌اند). dbmigrate منبع حقیقتِ **ورژن‌دار**
schema است: تغییرهای عمدی schema را از این به بعد این‌جا به‌صورت نسخه ثبت
کنید — مخصوصاً تغییرهایی که AutoMigrate هرگز اعمال نمی‌کند (تغییر ایندکس
موجود، DROP ستون، تغییر constraint — رجوع کامنت `DBSchema` در
`shared-core/models/models.go` که دقیقاً همین دردسر بود).
