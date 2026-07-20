# CreatorBot V3 — مستند کامل پروژه

> منبع مرجع عمیق. برای خلاصه‌ی روزمره `CLAUDE.md`، برای تاریخچه `CHANGELOG.md`.
> آخرین به‌روزرسانی: ۲۰۲۶-۰۷-۱۵.

---

## ۱. این پروژه چیست

در ۲۰۲۶-۰۷-۱۵ control planeهای Telegram و HTTP هم‌سان شدند: lifecycle مشترک روی
`deploy.{server_id}`، BotToken رمز‌شده، provisioning کامل plan/license/env، خرید و
subscription اتمیک، renew/redeem/broadcast/source-worker/audit دوطرفه و migration تلگرام.
جزئیات و محدودیت‌ها در `MANAGER_PARITY.md` است.

پلتفرم **PaaS برای ساخت ربات تلگرام بدون کدنویسی** با واحد پول TON. سه لایه:
- **فروش/ساخت** — کاربر ربات خودش را از فروشگاه پلتفرم می‌خرد و می‌سازد.
- **محصول** — ربات‌های ساخته‌شده (آپلودر فایل، VPN، آرشیو، قفل عضویت).
- **رشد/درآمد** — تبلیغات، اجاره‌ی قفل کانال روی ربات‌های رایگان، تقسیم درآمد بین گروه‌ها.

قیاس ذهنی: Shopify + Stripe + Google Ads، برای دنیای ربات‌های تلگرام.

---

## ۲. قانون‌های بنیادی معماری

### جدایی کامل با پیام‌رسان
هیچ سرویسی مستقیم به DB سرویس دیگر کوئری نمی‌زند. ارتباط فقط با **NATS**:
- **Request/Reply** — پاسخ فوری لازم است (همه‌ی عملیات پولی: `pay.*` از طریق
  `shared-core/natspayclient` ↔ `botpay`).
- **Publish/Subscribe** — رویداد بی‌نام‌گیرنده (`membership.joined`, `fraud.detected`,
  `campaign.revenue.generated`, `earning.created`).

### دیتابیس‌ها (وضعیت ۲۰۲۶-۰۷-۰۶)
- **PostgreSQL** — هر سرویس مرکزی دیتابیس خودش را دارد روی همان instance:
  `botpay→botpay`, `ads-bot→adsbot`, `community-service→community`,
  `revenue-service→revenue`, `license-service→license`, `image-registry→imageregistry`.
  استثنا: `botmanager`+`apimanager` دیتابیس مشترک `botmanager` (دو رابط روی همان مدل‌های
  `shared-core`). جداسازی فیزیکی کامل (سرور جدا) هنوز انجام نشده.
- **MongoDB** — داده‌های مخصوص هر ربات کاربر، جداسازی با `instance_id`.
- **Redis** — فقط کش (نه منبع حقیقت)؛ فقط `botpay` روی کش موجودی می‌نویسد.
- **NATS JetStream** — پیام‌رسانی + صف‌های پایدار.

### deployment ربات‌های کاربر
`uploader-bot`, `vpn-bot`, `member-bot`, `archive-bot`, `ads-bot` در docker-compose اصلی
نیستند — `agentmanager` آن‌ها را داینامیک به‌صورت container جدا با Docker SDK می‌سازد.
اتصال daemon از `DOCKER_HOST` می‌آید و در deployment می‌تواند پشت docker-socket-proxy محدود باشد.

---

## ۳. نقشه‌ی سرویس‌ها

### لایه‌ی مرکزی (ثابت، همیشه روشن)
| سرویس | مسئولیت | نکته |
|---|---|---|
| **botmanager** | ربات فروش + پنل کاربر/ادمین | تنها سرویس با مدل‌های `shared-core` |
| **apimanager** | کنترل‌پلین HTTP فعال برای user/admin | peer وب botmanager؛ JWT، lifecycle، wallet، plan و proxy محصول |
| **agentmanager** | ساخت واقعی container کاربران روی هر سرور | Docker SDK؛ اتصال daemon از DOCKER_HOST |
| **webhook-gateway** | آپدیت‌های تلگرام (حالت webhook) → NATS | هر ربات مستقل polling/webhook |
| **botpay** | کیف‌پول مرکزی TON؛ تنها نویسنده‌ی موجودی | لجر دوطرفه + زنجیره‌ی هش‌شده (chainguard) |
| **dbmigrate** | CLI migration ورژن‌دار برای ۸ سرویس Postgres | `up/status/mark/new/list` |

### ربات‌های قابل‌ساخت (محصول، داینامیک)

(۲۰۲۶-۰۷-۱۷) هر چهار ربات این بخش دیگر Postgres ندارند — همه MongoDB، یک دیتابیسِ مشترک
به‌ازای نوعِ ربات + جداسازیِ instanceها با فیلدِ `instance_id` (رجوع CLAUDE.md بخش ۲). رجوع
CHANGELOG برای جزئیاتِ کاملِ هر دو فاز (migration اولیه + رفعِ نبودِ جداسازیِ tenant).

| سرویس | مسئولیت |
|---|---|
| **uploader-bot** | فروش فایل با کد؛ ۲۸ قابلیت (قفل کانال، رمز، آلبوم، اشتراک، حذف خودکار، گزارش/لایک، بکاپ، چند ادمین) |
| **vpn-bot** | فروش VPN؛ runtime فعلی Marzban؛ adapter سه provider دیگر هنوز در startup wired نیست |
| **archive-bot** | آرشیو فایل با جستجوی فارسی fuzzy (app-side trigram/Jaccard، معادل pg_trgm سابق) |
| **member-bot** | زیرساخت داخلی چک متمرکز عضویت کانال با کش (نه ربات کاربرپسند) |

### لایه‌ی تبلیغات و درآمد
| سرویس | مسئولیت |
|---|---|
| **ads-bot** | (۱) تبلیغات CPJ کلاسیک، (۲) اجاره‌ی قفل کانال روی ربات‌های رایگان (مدل سه‌طرفه) |
| **community-service** | تقسیم درآمد بین گروه‌ها با لینک دعوت قابل‌ردیابی |
| **fraud-engine** | امتیاز کیفیت کاربر/گروه؛ تشخیص تقلب (HTTP + NATS، Mongo) |
| **revenue-service** | قوانین کمیسیون و واریز نهایی |

### پیاده‌شده، تست‌نشده در production
| سرویس | وضعیت |
|---|---|
| **source-service** | MTProto client کامل (gotd/td v0.159)، rules engine، watch/forward، NATS task dispatch. هنوز E2E با اکانت واقعی نشده. حساس‌ترین سرویس (اکانت کامل تلگرام). |

---

## ۴. مدل کامل داده (به‌تفکیک سرویس)

- **shared-core:** `User`, `Server`, `BotTemplate`, `BotInstance`, `Plan`, `PlanBotLimit`,
  `Subscription`, `Payment`, `InviteLink`, `DeployJob`, `AuditLog`
- **botpay:** `Wallet`, `Transaction`, `Invoice`, `WithdrawRequest`, `LedgerEntry` (زنجیره‌ی هش‌شده)
- **uploader-bot:** `User`, `SubPlan`, `Payment`, `Folder`, `Code`, `File`, `CodeFile`,
  `ForceJoinChannel`, `PreviewChannel`, `Backup`, `Setting`, `DownloadLog`, `Admin`
- **vpn-bot (MongoDB):** `User`, `Panel`, `Plan`, `Subscription`, `DiscountCode`, `Payment`
  (`Setting` بلااستفاده بود، در migration به Mongo حذف شد)
- **member-bot (MongoDB):** `Owner`, `Lock`, `CheckBot` (با `Memberships` تودرتو), `Payment`
  (`MemberVerification`/`Setting` بلااستفاده بودند، حذف شدند)
- **archive-bot (MongoDB):** `User`, `Category`, `File` (با `Ngrams` برای جستجوی فازی؛ `Setting`
  بلااستفاده بود، حذف شد)
- **ads-bot:** CPJ قدیمی: `AdConfig`, `ChannelCategory`, `Publisher`, `AdChannel`,
  `MemberAnalysis`, `Campaign`, `Impression` — اجاره‌ی قفل (جدید): `LockRentalCampaign`,
  `FreeBotSlot`, `RentalJoinReward`, `FreeBotOwnerReward`
- **community-service:** `Community`, `CampaignParticipant`, `CommunityRevenue`, `CommunityDistribution`
- **fraud-engine:** `UserProfile`, `UserProfileHistory`, `UserMembership`, `UserActivity`,
  `UserScoreSnapshot`, `ScoreBreakdown`, `CommunityScoreSnapshot`, `CommunityBreakdown`,
  `CommunityStatistics`, `FraudEvent`
- **revenue-service:** `RevenueRule`, `Earning`, `PlatformWallet`

### جزئیات مدل اجاره‌ی قفل (ads-bot/internal/store/models.go)
- `LockRentalCampaign`: `BuyerTelegramID`, `TargetChannelID`, `Status`
  (`pending_review`→`active`→`done`/`rejected`), `RewardPerJoinTON`, `Budget`, `Spent`,
  `FreeBotOwnerRewardPercent` (پیش‌فرض ۵)، `StartAt`, `EndAt`. متدها: `RemainingBudget()`,
  `IsActive()`.
- `FreeBotSlot`: `BotInstanceID`(unique), `BotID`(unique), `RentalID`(nil=آزاد),
  `AssignedOwnerTelegramID`, `IsChannelAdminConfirmed`.
- `RentalJoinReward`: unique `(RentalID, TelegramID)`؛ `Status` (`pending`/`settled`/`reversed`),
  `SettleAt` (=CreatedAt+`RewardSettlementDelay`=۲۴h)، `SettledAt`.
- `FreeBotOwnerReward`: unique `(RentalID, SlotID)`؛ همان state machine تأخیری.
- ثابت‌ها (`internal/store/store.go`): `RewardSettlementDelay=24h`,
  `DefaultRentalDuration=30d`؛ ticker settlement=۵m در `internal/tgbot/lockrental.go`.

---

## ۵. نقشه‌ی NATS

### Request/Reply (پولی — `natspayclient` ↔ `botpay`)
`pay.balance`, `pay.authorize`, `pay.deduct`, `pay.credit`, `pay.transfer`,
`pay.invoice.create`, `member.check`. احراز هویت: HMAC با `SERVICE_HMAC_SECRET` و
`service_key = ComputeServiceKey(secret, service_id)`. برای named services فقط HMAC کافی است؛
`bot_<BotID>` علاوه بر HMAC نیاز به DB check (instance فعال) دارد.

### Publish/Subscribe (رویدادها)
| Subject | فرستنده | گیرنده(ها) | transport |
|---|---|---|---|
| `service.creation.requested` | botmanager | agentmanager | |
| `agent.<serverID>.deploy` | botmanager | agentmanager | |
| `agent.<serverID>.result` | agentmanager | botmanager/apimanager | |
| `config.updated` | botmanager | ربات‌های فرعی | |
| `membership.joined` | member-bot | fraud-engine، community-service، **ads-bot** | **core NATS** (نه JetStream) |
| `membership.left` | member-bot | fraud-engine | core |
| `freebot.created` | botmanager | ads-bot | |
| `fraud.detected` | fraud-engine | ads-bot (لغو پاداش pending) | core |
| `campaign.revenue.generated` | ads-bot (فقط CPJ، نه lock-rental) | community-service | |
| `earning.created` | ads-bot، community-service | revenue-service | |
| `wallet.updated` | botpay | همه‌ی کلاینت‌ها (باطل‌کردن کش) | |

**نکته‌ی مهم transport:** `membership.joined` و `fraud.detected` با `PublishCore` (core NATS)
منتشر و با `nc.Subscribe` مصرف می‌شوند — برای جعل در تست باید core publish کرد.
منبع struct: `shared-core/protocol/subjects.go:MembershipJoinedEvent`،
publisher: `member-bot/internal/events/publisher.go`.

**تصحیح مهم:** جریان lock-rental **هیچ** `campaign.revenue.generated`/`earning.created`
منتشر نمی‌کند؛ این دو فقط در مسیر CPJ (`ads-bot/internal/engine/engine.go:RecordJoin`) هستند.
خروجی NATS جریان lock-rental فقط `pay.deduct` (تأیید) و `pay.credit` (settlement) است.

---

## ۶. جریان‌های اصلی

### جریان ۱: ساخت ربات
```
کاربر پلن می‌خرد → pay.deduct (botpay) → BotInstance ثبت
→ NATS deploy به agentmanager → docker run روی سرور → ربات بالا می‌آید
→ license.verify (startup، fail-closed) → agent.result → botmanager DB update
→ شکست هر مرحله → refund خودکار
```

### جریان ۲: اجاره‌ی قفل کانال (سه‌طرفه، escrow)
```
۱. خریدار در ads-bot درخواست اجاره (/rentlock) → LockRentalCampaign (pending_review)
۲. تأیید توسط OWNER پلتفرم (نه هر ادمین) — callback rent_approve:<id>
۳. تأیید → pay.DeductWithMeta بودجه از خریدار → ApproveLockRental (active, end_at=+30d)
        → AssignSlotsToRental (تا ۳ FreeBotSlot آزاد وصل می‌شوند)
۴. خریدار ربات‌ها را در کانالش ادمین می‌کند → قفل شروع
۵. کاربر واقعی عضو می‌شود → member-bot → membership.joined (core NATS)
۶. ads-bot: TryRecordJoinReward (رزرو، pending، settle_at=+24h) + AddRentalJoinCount (Spent↑)
        + سهم owner ربات رایگان هم رزرو (payFreeBotOwners → TryRecordOwnerReward)
۷. بعد ۲۴h (scheduler هر ۵m): FindDueRewards → pay.credit → SettleReward (settled)
۸. اگر fraud.detected قبل از تسویه → ReversePendingRewardByUser → reversed، بودجه برمی‌گردد
۹. بودجه تمام/زمان گذشت → MarkRentalDoneIfFinished (done) → آزادسازی slot ها + اطلاع
```
اصل کلیدی: هیچ پولی فوری منتقل نمی‌شود؛ همیشه escrow ۲۴ساعته برای تشخیص تقلب.

فایل‌های کلیدی: `ads-bot/internal/tgbot/lockrental.go` (کل منطق)،
`ads-bot/internal/store/store.go` (DB)، `ads-bot/cmd/main.go` (wiring/subscribe/scheduler).

---

## ۷. وضعیت امنیتی (خلاصه — جزئیات در SECURITY.md)

سرویس‌های audit‌شده و رفع‌شده: uploader-bot (privilege escalation)، botpay (TOCTOU race در
CreateWithdraw، allowlist)، member-bot (publisher مرده)، community-service (۴ باگ)،
webhook-gateway (gateway.register بدون auth)، ads-bot (تأیید کمپین بدون admin check).

سشن ۲۰۲۶-۰۷-۱۴: fraud-engine (fail-open auth)، vpn-bot (double-spend race + payment dedup)،
archive-bot (botUsername) — رجوع SECURITY.md.

secretهای اپلیکیشنی rotate و فایل‌های `.env` از tracking و history حذف شده‌اند. توکن‌های خارجی و
پسوردهای زیرساختی که نیازمند هماهنگی production هستند مطابق `SECRETS_ROTATION.md` هنوز باید revoke/rotate شوند.

---

## ۸. تست (خلاصه — جزئیات در TESTING.md)

- `tools/e2e-provision/` — زنجیره‌ی ساخت ربات بدون تلگرام (تأییدشده ۲۰۲۶-۰۷-۱۰).
- `ads-bot/tools/e2e-lockrental/` — کل چرخه‌ی اجاره‌ی قفل بدون تلگرام (جدید، ۲۰۲۶-۰۷-۱۴).
- tests/integration specification simulation مستقل است؛ production packageها را اجرا نمی‌کند. tests/e2e نیز خارج از module/CI است.
- مسیر A (پنل تلگرام) — فقط دستی، رجوع `E2E_RUNBOOK.md`.

---

## ۹. Deployment

`docker-compose.yml`: postgres, mongo, redis, nats, botmanager, apimanager, botpay,
docker-socket-proxy, agentmanager, webhook-gateway, revenue-service, community-service,
fraud-engine + مانیتورینگ (loki, promtail, prometheus, grafana).

`run.sh` (dev): سرویس‌های مرکزی را با `go run` بالا می‌آورد (ترتیب: botpay →
image-registry/license → fraud/revenue/community → member-bot → ads-bot → agentmanager →
apimanager → botmanager). زیرساخت داده (pg/mongo/redis/nats) باید جدا بالا باشد.

**وابستگی مهم:** همه‌ی ربات‌ها در startup با `tele.NewBot` به local-bot-api
(`141.95.210.17:8081`، هاردکد در `botpay/cmd/main.go:158` و مشابه) وصل می‌شوند؛ اگر در
دسترس نباشد `getMe` شکست → `log.Fatal`. برای dev بدون آن سرور باید URL عوض یا حذف شود.

---

## ۱۰. شکاف‌های واقعی (صادقانه)

- **secret rotation بیرونی/زیرساختی** — حذف history انجام شده؛ revoke توکن‌های خارجی و rotation هماهنگ دیتابیس/NATS/Redis/Mongo باقی است.
- **local-bot-api هاردکد** — آدرس در چند فایل هاردکد است؛ باید به config منتقل شود.
- **source-service** — E2E با اکانت واقعی تلگرام نشده؛ hotspot های امنیتی (رجوع SECURITY.md).
- **apimanager** — کنترل‌پلین HTTP فعال است؛ شکاف اصلی آن تست صفر، handler بزرگ و تکمیل APIهای product-specific است.
- **جداسازی فیزیکی DB** — فقط منطقی جدا شده، نه سرور جدا.
- **مسیر A** — تست کامل خرید پلن از پنل تلگرام هنوز end-to-end انجام نشده.
- **RewardSettlementDelay/ticker هاردکد** — ۲۴h و ۵m از env قابل تنظیم نیستند.

---

## ۱۱. نتیجه‌ی بازبینی کامل کد — ۲۰۲۶-۰۷-۱۴

تمام ۳۹۸ فایل Go فعلی خوانده شد. repository شامل ۲۲ module Go و ۲۱ entry point main است. ماتریس دقیق فایل‌ها، wiring، datastore، تست و شکاف‌ها در services/SERVICE_REVIEW.md ثبت شده است.

تصحیح‌های کلیدی: apimanager کنترل‌پلین HTTP فعال است؛ agentmanager از Docker SDK استفاده می‌کند؛ image-registry هنوز artifact را به agent دانلود/load نمی‌کند؛ بیشتر eventهای دامنه Core NATS هستند؛ tests/integration شبیه‌سازی مستقل است؛ و شکاف‌های فوری مالی شامل ref per-join ads، schema community→revenue و terminal state بدون acknowledgment است.
