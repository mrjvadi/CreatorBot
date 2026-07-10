# ads-bot — چه چیزی نیاز داریم (بررسی ۲۰۲۶-۰۷-۰۴)

## وضعیت فعلی (خلاصه‌ی واقعی، نه قدیمی)

بازبینی کامل ۱۲ فایل Go (~۳۳۶۰ خط) نشان می‌دهد ads-bot یک سرویس دومنظوره و
تقریباً کامل است:

1. **CPJ کلاسیک** (`Campaign`, `AdChannel`, `Impression`, `Publisher`) —
   ناشر بودجه تعیین می‌کند، تبلیغ روی کانال‌های تأییدشده پخش می‌شود، هزینه‌ی
   هر join واقعی از بودجه کم می‌شود.
2. **اجاره‌ی قفل کانال روی ربات‌های رایگان** (`LockRentalCampaign`,
   `FreeBotSlot`, `RentalJoinReward`, `FreeBotOwnerReward`) — این بخش کاملاً
   end-to-end پیاده‌سازی شده، نه فقط طراحی‌شده: تأیید ادمین اصلی، کسر بودجه
   با `natspayclient.DeductWithMeta`، تخصیص slot، escrow ۲۴ساعته با
   `RunSettlementScheduler`، لغو پاداش با `fraud.detected`، و پایان خودکار
   کمپین (`checkExpiredRentals`, `checkCampaignCompletion`).

نکته‌ی مثبت مهم: چرخه‌ی «ربات فرعی می‌گوید در کانال خریدار ادمین شدم» کامل و
واقعاً سیم‌کشی‌شده است — `uploader-bot/internal/tgbot/mychatmember.go:41` با
`memberclient.ConfirmChannelAdmin` صدا می‌زند، که به NATS subject
`protocol.SubjConfirmChannelAdmin` می‌رود، و `ads-bot/cmd/main.go:171-182`
واقعاً به آن پاسخ می‌دهد (`ConfirmChannelAdminByBotID`). این برخلاف چیزی است
که کامنت خودِ `lockrental.go:514-515` می‌گوید ("فاز بعد صدا زده خواهد شد") —
آن کامنت قدیمی/نادرست است، فاز بعد همین الان پیاده شده.

امنیت callback: هر عملیات ادمین‌محور (`approve`, `reject`, `verify_ch`,
`reject_ch`, `rent_approve`, `rent_reject`) در `handler.go:157-206` جداگانه
`isAdmin(c)` چک می‌کند — این همان نقصی است که طبق CLAUDE.md قبلاً رفع شده؛
تأیید شد که رفع باقی مانده. عملیات مالکیت‌محور (`camp_pause`, `camp_del` در
`campaign.go:225-264`) هم مالکیت واقعی کمپین را با
`camp.PublisherID != pub.ID` چک می‌کنند.

## چیزی که واقعاً کم است (با file:line برای هر ادعا)

1. **کل زیرساخت تحلیل کیفیت کانال/کاربر (`Analyzer`) کد مرده است.**
   `ads-bot/internal/engine/analyzer.go` (۱۶۹ خط) شامل `NewAnalyzer`,
   `AnalyzeChannel`, `AnalyzeMember`, `ComputeEffectiveCPJ` است، ولی هیچ‌کدام
   در کل سرویس صدا زده نمی‌شوند (`grep -rn "AnalyzeChannel\|AnalyzeMember\|
   ComputeEffectiveCPJ\|NewAnalyzer" ads-bot --include="*.go" | grep -v
   analyzer.go` → صفر نتیجه). تأیید کانال در عمل کاملاً دستی است
   (`tgbot/channel.go:116-136` — فقط پیام به ادمین با دکمه‌ی تأیید/رد، بدون
   امتیاز خودکار). یعنی مدل `MemberAnalysis` در دیتابیس migrate می‌شود
   (`store/models.go:228`) ولی هرگز پر نمی‌شود.
2. **صفر فایل تست.** `find ads-bot -name "*_test.go"` هیچ نتیجه‌ای برنمی‌گرداند؛
   منطق مالی حساس (کسر بودجه، رزرو پاداش، idempotency تسویه) بدون تست است.
3. **`ReportJoin`/attribution mismatch با member-bot** (این باگ در خود
   member-bot است نه ads-bot، ولی مستقیماً روی داده‌ای اثر می‌گذارد که
   ads-bot از fraud-engine می‌خواند) — به بخش «در سرویس‌های دیگر» مراجعه کنید.

## این‌ها را در سرویس‌های دیگر هم نوشتم (اگر مرتبط بود)

- **community-service/NEEDS.md** — `community.activity.updated` هرگز از
  هیچ ربات واقعی publish نمی‌شود (نه فقط برای ads-bot، بلکه چون
  `member-bot/internal/events/publisher.go:153` تابع `PublishActivity` را
  تعریف می‌کند ولی هیچ‌جا صدا نمی‌زند)، پس مسیر امتیاز فعالیت اعضا هم برای
  fraud-engine هم برای community-service کاملاً غیرفعال است.
- **fraud-engine/NEEDS.md** — `member-bot/internal/events/publisher.go:120-123`
  با کامنت صریح خودش تأیید می‌کند که پارامتر چهارم `ReportJoin` (که
  fraudclient آن را `campaign_id` می‌نامد) در واقع `invite_hash` است، نه
  campaign_id واقعیِ کمپین ads-bot. یعنی هر منطق آینده که بخواهد fraud را
  به کمپین مشخصی از ads-bot ربط بدهد، با این mismatch روبه‌رو می‌شود.

## به‌روزرسانی ۲۰۲۶-۰۷-۰۶: جداسازی دیتابیس
`ads-bot` حالا دیتابیس مخصوص خودش (`adsbot`) را دارد — دیگر با بقیه‌ی سرویس‌های مرکزی مشترک نیست
(`ads-bot/.env`). توجه: چون `ads-bot` در `docker-compose.yml` نیست (طبق CLAUDE.md فعلاً دستی با
`go run` تست می‌شود)، این تغییر فقط در `.env` محلی اعمال شد — قبل از هر اجرای بعدی مطمئن شوید
دیتابیس `adsbot` واقعاً روی سرور Postgres شما ساخته شده (یا با اجرای مجدد
`deploy/migrations/000_create_databases.sql`، یا دستی).
