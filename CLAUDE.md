# CreatorBot V3 — معرفی کامل و عمیق پروژه

## فهرست
1. این پروژه چیست
2. معماری کلی و قانون‌های بنیادی
3. نقشه‌ی کامل ۱۸ سرویس
4. مدل کامل داده — همه‌ی جدول‌ها
5. نقشه‌ی کامل ارتباطات NATS
6. جریان‌های اصلی (سفر کاربر، سفر پول)
7. آنچه ساخته و کامل شده
8. آنچه کم داریم / شکاف‌های واقعی
9. وضعیت Deployment

---

## ۱. این پروژه چیست

یک **پلتفرم PaaS برای ساخت ربات تلگرام بدون کدنویسی**. منطق کسب‌وکار سه لایه دارد:

- **لایه‌ی فروش/ساخت** — کاربر ربات خودش را از فروشگاه پلتفرم می‌خرد و می‌سازد.
- **لایه‌ی محصول** — خود ربات‌های ساخته‌شده (آپلودر فایل، VPN، آرشیو، قفل عضویت).
- **لایه‌ی رشد/درآمد** — تبلیغات، اجاره‌ی قفل کانال روی ربات‌های رایگان پلتفرم،
  و تقسیم درآمد بین گروه‌ها.

شبیه این فکر کنید: Shopify (ساخت فروشگاه بدون کد) + Stripe (کیف‌پول داخلی) +
Google Ads (تبلیغات) — همه برای دنیای ربات‌های تلگرام، با واحد پول TON.

---

## ۲. معماری کلی و قانون‌های بنیادی

### اصل پایه: جدایی کامل با پیام‌رسان

```
سرویس A  ──(NATS)──▶  سرویس B
```

هیچ سرویسی مجاز نیست مستقیم به دیتابیس سرویس دیگر کوئری بزند. ارتباط فقط با
**NATS** انجام می‌شود، به دو شکل:

| الگو | کاربرد | مثال |
|---|---|---|
| **Request/Reply** (سؤال-جواب فوری) | وقتی همین الان به یک پاسخ نیاز دارید | «موجودی این کاربر چقدر است؟» |
| **Publish/Subscribe** (رویداد، بی‌نام‌گیرنده) | وقتی فقط می‌خواهید اطلاع بدهید چیزی اتفاق افتاد | «این کاربر عضو شد» |

این الگو در عمل پروژه این‌طور دیده می‌شود:
- همه‌ی عملیات **پولی** (`pay.balance`, `pay.deduct`, `pay.invoice.create`,...)
  Request/Reply هستند — یک کلاینت مشترک (`shared-core/natspayclient`) همه‌جا
  استفاده می‌شود، سرور آن `botpay` است.
- همه‌ی **رویدادها** (`membership.joined`, `fraud.detected`,
  `campaign.revenue.generated`, `earning.created`) Publish/Subscribe هستند —
  چند سرویس می‌توانند به یک رویداد گوش بدهند.

### وضعیت فعلی دیتابیس (به‌روزشده ۲۰۲۶-۰۷-۰۶)

سرویس‌های مرکزی همچنان روی **یک instance/سرور Postgres** هستند، ولی از
۲۰۲۶-۰۷-۰۶ هرکدام **دیتابیس مخصوص خودشان** را دارند (نه یک دیتابیس مشترک
`creatorbot` مثل قبل): `botpay` → `botpay`, `ads-bot` → `adsbot`,
`community-service` → `community`, `revenue-service` → `revenue`,
`license-service` → `license`, `image-registry` → `imageregistry`. این دقیقاً
همان جداسازیِ منطقی‌ای است که پیش‌تر «مسیر بلندمدت» نامیده می‌شد.

استثنای عمدی: `botmanager` و `apimanager` هنوز یک دیتابیس مشترک (`botmanager`)
دارند — چون این دو، دو رابط (ربات تلگرام + HTTP API) روی دقیقاً همان مدل‌های
`shared-core` (`User`, `BotInstance`, `Plan`, ...) هستند، نه دو مالک داده‌ی
مستقل؛ جدا کردن این دو یعنی apimanager دیگر هیچ instance/کاربر/پلنی نمی‌بیند.

جداسازی فیزیکی کامل (سرور/instance جدا برای هرکدام، نه فقط دیتابیس جدا روی
همان سرور) هنوز انجام نشده — این قدم بعدی است اگر لازم شود. رجوع
`deploy/migrations/000_create_databases.sql` و `docker-compose.yml` برای
پیاده‌سازی دقیق.

علاوه بر PostgreSQL:
- **MongoDB** — برای داده‌های مخصوص هر ربات کاربر (تنظیمات، آمار) با
  جداسازی به‌وسیله‌ی `instance_id`.
- **Redis** — فقط کش (نه منبع حقیقت). مثلاً موجودی کیف‌پول کش می‌شود، ولی فقط
  `botpay` حق نوشتن روی آن را دارد.
- **NATS JetStream** — هم پیام‌رسانی، هم صف‌های پایدار (مثل صف چک عضویت در
  member-bot).

### مدل deployment ربات‌های کاربر

`uploader-bot`, `vpn-bot`, `member-bot`, `archive-bot` و `ads-bot` **در
docker-compose اصلی پلتفرم نیستند** — این‌ها به‌صورت داینامیک توسط
`agentmanager` (هنگام درخواست کاربر در botmanager) به شکل container جدا روی
سرورهای پلتفرم ساخته می‌شوند. فقط سرویس‌های مرکزی پلتفرم
(postgres, mongo, redis, nats, botmanager, apimanager, botpay, agentmanager,
webhook-gateway, revenue-service, community-service, fraud-engine، +
مانیتورینگ) در docker-compose ثابت هستند.

---

## ۳. نقشه‌ی کامل ۱۸ سرویس

### لایه‌ی مرکزی (ثابت، همیشه روشن)

| سرویس | مسئولیت | نکته‌ی کلیدی |
|---|---|---|
| **botmanager** | ربات فروش اصلی؛ پنل کاربر و پنل ادمین کامل | تنها سرویسی که مدل‌هایش از `shared-core` می‌آید، نه مدل اختصاصی |
| **apimanager** | دروازه‌ی HTTP بیرونی (برای آینده — وب/اپ) | فعلاً کم‌استفاده؛ هدف نهایی: ترجمه‌ی HTTP↔NATS |
| **agentmanager** | روی هر سرور؛ واقعاً Docker container کاربران را می‌سازد/مدیریت می‌کند | با `docker` CLI کار می‌کند (نه SDK) از طریق یک `docker-socket-proxy` محدودشده، نه دسترسی مستقیم به socket |
| **webhook-gateway** | وقتی پلتفرم در حالت webhook (نه polling) باشد، آپدیت‌های تلگرام را می‌گیرد و به NATS منتقل می‌کند | هر ربات می‌تواند مستقل polling یا webhook انتخاب کند |
| **botpay** | کیف‌پول مرکزی (TON)؛ تنها نویسنده‌ی موجودی در کل پلتفرم | لجر دوطرفه + **زنجیره‌ی هش‌شده شبیه بلاکچین** (تشخیص دستکاری) |
| **dbmigrate** | CLI متمرکز migration ورژن‌دار برای هر ۱۱ سرویس Postgres دار (نه یک سرویس همیشه‌روشن) | baseline از AutoMigrate واقعی هر سرویس؛ `up`/`status`/`mark`/`new` — رجوع `dbmigrate/README.md` |

### ربات‌های قابل‌ساخت (محصول نهایی، داینامیک)

| سرویس | مسئولیت |
|---|---|
| **uploader-bot** | فروش فایل با کد دریافت — کامل‌ترین (۲۸ قابلیت: قفل کانال، رمز، آلبوم، اشتراک، حذف خودکار ضدفیلتر، گزارش/لایک، بکاپ،...) |
| **vpn-bot** | فروش اشتراک VPN؛ اتصال به Marzban/Marzneshin/Hiddify/XUI؛ پرداخت کارت/زرین‌پال/NowPayments |
| **archive-bot** | آرشیو فایل با جستجوی فارسی fuzzy (pg_trgm) |
| **member-bot** | **زیرساخت داخلی**، نه ربات کاربرپسند — چک متمرکز عضویت کانال با کش، تا هر ربات مجبور نباشد خودش ادمین هر کانالی شود |

### لایه‌ی تبلیغات و درآمد (جدیدترین و پیچیده‌ترین)

| سرویس | مسئولیت |
|---|---|
| **ads-bot** | دو سیستم در یک سرویس: (۱) تبلیغات CPJ کلاسیک، (۲) اجاره‌ی قفل کانال روی ربات‌های رایگان پلتفرم (مدل سه‌طرفه‌ی پولی) |
| **community-service** | تقسیم درآمد بین گروه‌ها با لینک دعوت قابل‌ردیابی |
| **fraud-engine** | امتیاز کیفیت کاربر/گروه؛ تشخیص الگوهای تقلب |
| **revenue-service** | قوانین کلی کمیسیون و واریز نهایی |

### سرویس‌های پیاده‌شده اما هنوز تست‌نشده در Production

| سرویس | وضعیت |
|---|---|
| **source-service** | پیاده‌شده با gotd/td v0.159 — MTProto client کامل، rules engine، watch/forward، NATS task dispatch، worker registration. هنوز تست E2E با اکانت تلگرام واقعی نشده. |

---

## ۴. مدل کامل داده — همه‌ی جدول‌ها (به‌تفکیک سرویس)

این لیست مستقیماً از کد استخراج شده (نه تخمینی):

**shared-core** (پایه‌ی پلتفرم، استفاده‌شده توسط botmanager):
`User`, `Server`, `BotTemplate`, `BotInstance`, `Plan`, `PlanBotLimit`,
`Subscription`, `Payment`, `InviteLink`, `DeployJob`, `AuditLog`

**botpay** (کیف‌پول):
`Wallet`, `Transaction`, `Invoice`, `WithdrawRequest`, `LedgerEntry` (زنجیره‌ی هش‌شده)

**uploader-bot**:
`User`, `SubPlan`, `Payment`, `Folder`, `Code`, `File`, `CodeFile`,
`ForceJoinChannel`, `PreviewChannel`, `Backup`, `Setting`, `DownloadLog`, `Admin`

**vpn-bot**:
`User`, `Panel`, `Plan`, `Subscription`, `DiscountCode`, `Payment`, `Setting`

**member-bot**:
`Owner`, `Lock`, `CheckBot`, `BotChannelMembership`, `MemberVerification`,
`Payment`, `Setting`

**archive-bot**:
`User`, `Category`, `File`, `Setting`

**ads-bot**:
`AdConfig`, `ChannelCategory`, `Publisher`, `AdChannel`, `MemberAnalysis`,
`Campaign`, `Impression` (سیستم CPJ قدیمی) +
`LockRentalCampaign`, `FreeBotSlot`, `RentalJoinReward`, `FreeBotOwnerReward`
(سیستم اجاره‌ی قفل، جدید)

**community-service**:
`Community`, `CampaignParticipant`, `CommunityRevenue`, `CommunityDistribution`

**fraud-engine**:
`UserProfile`, `UserProfileHistory`, `UserMembership`, `UserActivity`,
`UserScoreSnapshot`, `ScoreBreakdown`, `CommunityScoreSnapshot`,
`CommunityBreakdown`, `CommunityStatistics`, `FraudEvent`

**revenue-service**:
`RevenueRule`, `Earning`, `PlatformWallet`

---

## ۵. نقشه‌ی کامل ارتباطات NATS

### Request/Reply (پولی — همه از طریق `natspayclient` ↔ `botpay`)

| Subject | کاربرد |
|---|---|
| `pay.balance` | گرفتن موجودی |
| `pay.authorize` | آیا این سرویس مجاز به دسترسی به حساب کاربر است؟ |
| `pay.deduct` | کسر (پرداخت) |
| `pay.credit` | افزودن اعتبار (پاداش، استرداد) |
| `pay.transfer` | انتقال بین دو کاربر |
| `pay.invoice.create` | ساخت رسید واریز TON |
| `member.check` | (مشابه، ولی برای member-bot) آیا کاربر عضو این کانال است؟ |

### Publish/Subscribe (رویدادها)

| Subject | فرستنده | گیرنده(ها) |
|---|---|---|
| `service.creation.requested` | botmanager | agentmanager (deploy واقعی) |
| `agent.<serverID>.deploy` | botmanager | agentmanager |
| `config.updated` | botmanager (تغییر تنظیمات) | (broadcast به ربات‌های فرعی) |
| `membership.joined` | member-bot | fraud-engine، community-service، **ads-bot** |
| `membership.left` | member-bot | fraud-engine |
| `freebot.created` | botmanager (وقتی instance رایگان ساخته شد) | ads-bot |
| `fraud.detected` | fraud-engine | ads-bot (برای لغو پاداش معلق) |
| `campaign.revenue.generated` | ads-bot | community-service |
| `earning.created` | ads-bot، community-service | revenue-service |
| `wallet.updated` | botpay | همه‌ی کلاینت‌ها (باطل‌کردن کش Redis) |

نکته‌ی مهم: زنجیره‌ی درآمد تبلیغات این‌طور به هم وصل است:

```
ads-bot ──(campaign.revenue.generated)──▶ community-service
ads-bot یا community-service ──(earning.created)──▶ revenue-service
fraud-engine ──(fraud.detected)──▶ ads-bot (لغو پاداش در صورت تقلب)
```

---

## ۶. جریان‌های اصلی

### جریان ۱: ساخت ربات توسط کاربر

```
کاربر در botmanager پلن می‌خرد
  → پول از کیف‌پول او در botpay کسر می‌شود (pay.deduct)
  → یک BotInstance رکورد می‌شود
  → NATS: "این سرویس را بساز" به agentmanager می‌رود
  → agentmanager روی یکی از سرورها Docker container واقعی بالا می‌آورد
  → ربات تازه (مثلاً uploader-bot) بالا می‌آید، به DB مشترک وصل می‌شود
  → اگر هر مرحله شکست بخورد → پول خودکار به خریدار برمی‌گردد (refund)
```

### جریان ۲: اجاره‌ی قفل کانال روی ربات‌های رایگان (مدل سه‌طرفه‌ی پولی)

این بخش به‌طور کامل در این مرحله از پروژه طراحی و کدنویسی شده:

```
۱. خریدار در ads-bot درخواست اجاره می‌دهد (کانال + بودجه + پاداش هر عضو)
۲. درخواست برای تأیید به ادمین اصلی پلتفرم می‌رود (نه هر ادمین معمولی)
۳. تأیید → بودجه همان لحظه از کیف‌پول خریدار کسر می‌شود
        → چند ربات رایگان به این کمپین وصل و در اختیار خریدار قرار می‌گیرند
۴. خریدار ربات‌ها را در کانال خودش ادمین می‌کند → قفل‌کردن شروع می‌شود
۵. کاربر واقعی عضو می‌شود → member-bot تشخیص می‌دهد → membership.joined
۶. ads-bot می‌گیرد:
     - پاداش به کاربر "رزرو" می‌شود (نه فوری) — ۲۴ ساعت تأخیر برای ضد تقلب
     - سهم owner ربات رایگان هم به همین شکل رزرو می‌شود
۷. بعد از ۲۴ ساعت (اگر fraud-engine چیزی نگفته) → واریز واقعی انجام می‌شود
۸. اگر fraud-engine قبل از تسویه تقلب تشخیص دهد → پاداش لغو می‌شود، پول
   هرگز واریز نمی‌شود، بودجه به کمپین برمی‌گردد
۹. وقتی بودجه تمام شود یا زمان کمپین بگذرد → کمپین به پایان می‌رسد →
   اطلاع‌رسانی به خریدار و همه‌ی owner های ربات‌های رایگان متصل
```

نکته‌ی کلیدی طراحی: **هیچ پولی فوری منتقل نمی‌شود** — همیشه یک دوره‌ی
escrow (در انتظار) هست تا فرصت تشخیص تقلب باشد.

---

## ۷. آنچه ساخته و کامل شده

- ✅ ساخت ربات و جریان خرید پلن کامل (با refund خودکار در شکست)
- ✅ uploader-bot با ۲۸ قابلیت کامل (قفل کانال، رمز، آلبوم، اشتراک، حذف خودکار، گزارش/لایک، بکاپ، چند ادمین)
- ✅ سیستم پرداخت متمرکز با NATS request/reply (جایگزین HTTP قدیمی)
- ✅ احراز هویت پویا بین سرویس‌ها (بدون کلید ثابت در env برای هر ربات)
- ✅ کش Redis برای موجودی با باطل‌سازی خودکار با رویداد
- ✅ لجر مالی با زنجیره‌ی هش‌شده (دستکاری قابل‌کشف) + پایشگر دوره‌ای (chainguard)
- ✅ سوییچ polling/webhook به‌ازای هر ربات با یک env
- ✅ زیرساخت متمرکز چک عضویت با کش (member-bot) به‌جای نیاز هر ربات به ادمین‌شدن در هر کانال
- ✅ کل مدل اقتصادی اجاره‌ی قفل کانال — تأیید ادمین، کسر/رزرو/تسویه با تأخیر، اتصال fraud-engine، lifecycle کمپین

---

## ۸. آنچه کم داریم / شکاف‌های واقعی (صادقانه)

این بخش از بررسی دقیق کد نوشته شده، نه فرضی:

### شکاف‌های زیرساختی
- (رفع‌شده ۲۰۲۶-۰۷-۱۰ با **dbmigrate**) migration ناهماهنگ بین سرویس‌ها:
  قبلاً هر سرویس فقط با `AutoMigrate` در startup خودش schema می‌ساخت —
  بدون ورژن، بدون تاریخچه، و دوبار باعث خطای «جدول وجود ندارد» شد.
  حالا سرویس `dbmigrate/` (رجوع README خودش) برای هر ۱۱ سرویس Postgres دار
  migration ورژن‌دار SQL دارد (baseline نسخه‌ی ۱ از AutoMigrate واقعی هر
  سرویس تولید شده) با دستورهای `up`/`status`/`mark`/`new`، جدول
  `schema_migrations` با checksum در هر دیتابیس، و ساخت خودکار دیتابیس.
  AutoMigrate در startup سرویس‌ها عمداً حذف نشده (additive و بی‌خطر است)؛
  تغییرهای عمدی schema از این به بعد به‌صورت نسخه در dbmigrate ثبت می‌شوند.
  ربات‌های Mongo (uploader-bot, fraud-engine, log-collector, admanager-bot)
  schema ندارند و خارج از این سیستم‌اند.
- (رفع‌شده ۲۰۲۶-۰۷-۱۰) **لیست سرویس‌های مجاز در botpay** دیگر هاردکود نیست:
  `authorize()` در payresponder اکنون HMAC را کافی می‌داند برای سرویس‌های
  مرکزی (هر سرویسی که `SERVICE_HMAC_SECRET` داشته باشد بدون تغییر کد مجاز
  است). فقط ربات‌های مشتری (`bot_<BotID>`) هنوز علاوه بر HMAC نیاز به
  DB check دارند (instance باید فعال باشد). قبلاً فراموش کردن اضافه‌کردن
  `ads-bot` به یک switch ثابت باعث قطعی واقعی شد.

### شکاف‌های تأیید‌نشده
- چند فراخوانی API تلگرام (`update.InviteLink`, `ChatByUsername`) با
  Bot API رسمی نوشته شده‌اند ولی محیط توسعه دسترسی به سورس کتابخونه
  نداشت — هنوز با اجرای واقعی تأیید نشده‌اند.
- بخش اجاره‌ی قفل کانال تازه به‌طور کامل طراحی شده؛ تست end-to-end با
  ربات واقعی و دیتابیس واقعی هنوز کامل انجام نشده.

### امنیتی (شناسایی و تا حدی رفع‌شده)
- یک نقص امنیتی واقعی در ads-bot پیدا و رفع شد: تأیید/رد کمپین تبلیغاتی
  قبلاً هیچ چک نمی‌کرد فرستنده‌ی callback واقعاً ادمین است.
- secret های محیطی (`.env`) در نقطه‌ای از تاریخچه‌ی گیت کامیت شده بودند؛
  باید rotate شوند (صرف اضافه‌کردن `.gitignore` کافی نیست).
- `agentmanager` قبلاً مستقیم به `docker.sock` دسترسی کامل داشت (یعنی
  هر آسیب‌پذیری در آن معادل دسترسی root به host بود) — این محدود شد
  با یک `docker-socket-proxy` که فقط endpointهای لازم را باز می‌گذارد.
- (رفع‌شده ۲۰۲۶-۰۷-۱۰) **webhook-gateway `gateway.register` بدون auth**: هر کسی با
  دسترسی NATS می‌توانست webhook هر ربات دلخواهی را hijack کند. رفع: `SERVICE_HMAC_SECRET`
  به Config اضافه شد؛ اگر تنظیم شده باشد، ارسال‌کننده باید `service_id`+`service_key`
  معتبر داشته باشد (همان الگوی HMAC بقیه‌ی سرویس‌ها). اگر secret خالی باشد، backward
  compatible (بدون چک).
- (رفع‌شده ۲۰۲۶-۰۷-۱۰) **privilege escalation در uploader-bot**: هر ادمین با هر دسترسی
  می‌توانست با ارسال مستقیم callback `aperm_t:<id>:admins` به خودش دسترسی `PermAdmins` بدهد —
  چون `adminTogglePerm`/`adminPermsMenu`/`adminAskAdmin`/`adminRemoveAdmin` فقط `isAdmin` عمومی
  داشتند نه `adminCan(ctx, c, PermAdmins)`. رفع: check دانه‌ریز اضافه شد به هر چهار تابع.
- (رفع‌شده ۲۰۲۶-۰۷-۱۰) **TOCTOU race در `botpay/CreateWithdraw`**: چک
  موجودی (`HasEnough`) و قفل‌کردن (`frozen += amount`) در دو مرحله‌ی جدا
  بودند — دو درخواست هم‌زمان می‌توانستند هر دو رد شوند و `frozen` را
  بیشتر از موجودی واقعی بالا ببرند. رفع: `SELECT FOR UPDATE` داخل همان
  تراکنش + re-check قبل از increment. خطای `store.ErrInsufficientBalance`
  تعریف شد و wallet آن را با `wallet.ErrInsufficientBalance` wrap می‌کند
  تا `errors.Is` در responder همچنان کار کند.
- (رفع‌شده ۲۰۲۶-۰۷-۱۴) **fraud-engine fail-open auth**: `authMiddleware` وقتی
  `ADMIN_KEY` خالی بود، هدر غایب `""` را با کلید خالی برابر می‌دید و همه‌ی
  `/admin/*` باز می‌ماند. رفع: fail-closed (کلید خالی → همیشه 401) +
  `crypto/subtle.ConstantTimeCompare` + `log.Fatal` در startup اگر کلید نباشد.
- (رفع‌شده ۲۰۲۶-۰۷-۱۴) **vpn-bot double-spend race**: `confirmBuyWithBalance`
  چک `balance < price` و `UpdateBalance(-price)` را جدا انجام می‌داد → دو کلیک
  هم‌زمان دو اشتراک می‌ساخت. رفع: متد اتمیک `DeductBalanceIfEnough`
  (`UPDATE ... WHERE balance >= amount` + بررسی `RowsAffected`).
- (رفع‌شده ۲۰۲۶-۰۷-۱۴) **vpn-bot payment بدون dedup**: `verifyOnlinePayment`
  هر کلیک «پرداخت کردم» را یک subscription جدید می‌کرد. رفع: `ClaimOnlinePayment`
  با `ON CONFLICT DO NOTHING` روی ایندکس partial یکتای `(gateway, ref_code)`.
- (رفع‌شده ۲۰۲۶-۰۷-۱۴) **secret rotation + بازنویسی history**: همه‌ی secret های
  اپلیکیشنی در `.env` ها rotate شدند (HMAC/ENCRYPTION/JWT/admin pairs/… هماهنگ)،
  همه‌ی `.env` ها untrack و `**/.env` به gitignore اضافه شد، و کل git history با
  `filter-branch` از `.env` پاک و force-push شد. پسوردهای زیرساخت و توکن‌های خارجی
  نیاز به اقدام دستی دارند — رجوع `SECRETS_ROTATION.md`. (backup:
  `../CreatorBotV3-backup.git`.)

### تست
- (اضافه‌شده ۲۰۲۶-۰۷-۱۴) `ads-bot/tools/e2e-lockrental/` — تست E2E کامل مدل اقتصادی
  اجاره‌ی قفل بدون تلگرام (با store واقعی + botpay واقعی). رجوع `E2E_RUNBOOK.md`.
  مستند کامل پروژه در `prog/`، تاریخچه در `CHANGELOG.md`.

### معماری ناتمام
- `apimanager` (دروازه‌ی HTTP بیرونی برای وب/اپ) ساخته شده ولی هنوز
  در عمل استفاده‌ی کاملی ندارد — برای زمانی است که پلتفرم بخواهد رابط
  وب/اپ بدهد.
- `source-service` (رفع‌شده ۲۰۲۶-۰۷-۱۰) پیاده شده — MTProto client (gotd/td v0.159),
  rules engine, channel watch/forward, NATS task dispatch. هنوز E2E با اکانت
  واقعی تلگرام تأیید نشده.
- (رفع جزئی ۲۰۲۶-۰۷-۰۶) دیتابیس‌های منطقی سرویس‌های مرکزی از هم جدا شدند
  (هرکدام دیتابیس خودش، رجوع بخش ۲) — ولی جداسازی فیزیکیِ کامل (سرور/instance
  جدا برای هرکدام، نه فقط دیتابیس جدا روی همان سرور Postgres) هنوز انجام نشده.
- (رفع‌شده ۲۰۲۶-۰۷-۱۰) **member-bot: Publisher هرگز ساخته نمی‌شد** —
  `events.Publisher` تعریف شده بود و handler join/leave/activity داشت ولی
  در `cmd/bot/main.go` هرگز instantiate یا register نمی‌شد. یعنی
  `membership.joined`/`membership.left` برای fraud-engine و community-service
  و `community.activity.updated` برای امتیازدهی فعالیت هرگز publish نمی‌شدند.
  رفع: publisher ساخته و با `pub.Register(rawBot)` ثبت می‌شود. همچنین
  `ActivityPublisher` interface به `Handler` اضافه شد تا پیام‌های گروهی
  از طریق `onText` هم activity را track کنند.
- (رفع‌شده ۲۰۲۶-۰۷-۱۰) community-service: چهار باگ واقعی رفع شد:
  (۱) nil pointer دو تابع MongoDB وقتی mongo=nil بود → guard اضافه شد،
  (۲) IncrementMemberCount هیچ‌وقت صدا نمی‌شد → در HandleJoin اضافه شد،
  (۳) کامنت اشتباه «MongoDB» برای DecrementMemberCount که روی Postgres کار می‌کرد،
  (۴) `community.reward.created` که هیچ subscriber ای نداشت → حالا `earning.created`
  با نوع `member_reward` publish می‌شود که revenue-service آن را پردازش می‌کند.

---

## ۹. وضعیت Deployment

`docker-compose.yml` این سرویس‌های ثابت را مدیریت می‌کند:
`postgres`, `mongo`, `redis`, `nats`, `botmanager`, `apimanager`, `botpay`,
`docker-socket-proxy`, `agentmanager`, `webhook-gateway`, `revenue-service`,
`community-service`, `fraud-engine`, و پشته‌ی مانیتورینگ
(`loki`, `promtail`, `prometheus`, `grafana`).

ربات‌های کاربر (`uploader-bot`, `vpn-bot`, `member-bot`, `archive-bot`) و
`ads-bot` در این فایل نیستند — این‌ها به‌صورت container جدا و داینامیک
ساخته/اجرا می‌شوند (یا توسط `agentmanager` برای ربات‌های واقعی کاربران،
یا دستی برای توسعه/تست مثل `ads-bot` که در حال حاضر مستقیم با
`go run cmd/*.go` تست می‌شود).