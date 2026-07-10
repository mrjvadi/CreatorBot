# community-service

## این سرویس چیست
تقسیم درآمد بین گروه‌ها/کانال‌ها با لینک دعوت قابل‌ردیابی — حلقه‌ی واسط بین `ads-bot` (که می‌گوید یک کمپین چقدر درآمد داشته) و `revenue-service` (که کمیسیون نهایی را حساب و پرداخت می‌کند).

## مسئولیت‌ها
- ثبت `Community` و اعضای آن با لینک دعوت اختصاصی (`InviteHash`) برای ردیابی منبع عضویت (`organic` در برابر `invite_link`).
- دریافت `campaign.revenue.generated` از `ads-bot`، ساخت `CommunityRevenue`، و توزیع بین اعضای فعال (`CommunityDistribution`) بر اساس امتیاز فعالیت (`CalcActivityScore` — پیام/ریپلای/ری‌اکشن/روزهای فعال، در MongoDB).
- انتشار `earning.created` برای `revenue-service` تا سهم واقعی TON پرداخت شود.
- امتیاز کیفیت کانال (`UpdateQualityScore`) بر اساس رویداد `community.score.updated` (از `fraud-engine`).

## ارتباطات
- NATS: subscribe `membership.joined`, `membership.left`, `community.activity.updated`, `campaign.revenue.generated`, `community.score.updated`؛ publish `earning.created`.
- PostgreSQL (روابط/امتیاز) + MongoDB (فعالیت اعضا).

## ایرادها و نکات
- **رفع شد (۲۰۲۶-۰۷-۰۲)**: هندلر `campaign.revenue.generated` (`internal/engine/nats_handler.go`) هیچ اعتبارسنجی مبلغ یا idempotency نداشت — چون این subject هم auth ندارد، هر کلاینت NATS می‌توانست مبلغ منفی/صفر بفرستد یا یک رویداد قبلی را replay کند و یک توزیع درآمد جدید و واقعی بسازد. رفع شد: مبلغ باید مثبت/متناهی باشد، و یک متد جدید `FindRevenueByCampaignCommunity` قبل از ساخت رکورد جدید چک می‌کند همین جفت (کمپین، کامیونیتی) قبلاً پردازش نشده باشد.
- **باقی‌مانده، low severity**: این فقط یک قدم از زنجیره را می‌بندد؛ خودِ subject همچنان بدون auth است (بخشی از ضعف عمومی نبودِ ACL در NATS، نه چیزی که بشود فقط این‌جا رفع کرد بدون تغییر پروتکل مشترک `pay.*`-مانند در همه‌ی سرویس‌های درگیر پول).
