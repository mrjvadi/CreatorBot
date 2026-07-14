# وضعیت امنیتی سرویس‌به‌سرویس — CreatorBot V3

آخرین به‌روزرسانی: ۲۰۲۶-۰۷-۱۴.

## رفع‌شده‌ها

| سرویس | مشکل | رفع | تاریخ |
|---|---|---|---|
| uploader-bot | privilege escalation — هر ادمین با callback `aperm_t:<id>:admins` به خودش `PermAdmins` می‌داد | `adminCan(ctx,c,PermAdmins)` به ۴ تابع | 07-10 |
| botpay | TOCTOU race در `CreateWithdraw` (چک و freeze جدا) | `SELECT FOR UPDATE` + re-check | 07-10 |
| botpay | allowlist هاردکد سرویس‌ها | HMAC کافی برای named services؛ فقط `bot_*` نیاز DB check | 07-10 |
| member-bot | Publisher هرگز register نمی‌شد → رویدادها منتشر نمی‌شدند | ساخت + `pub.Register(rawBot)` | 07-10 |
| community-service | ۴ باگ (nil mongo، IncrementMemberCount، کامنت، subscriber گمشده) | guard + fix + `earning.created` | 07-10 |
| webhook-gateway | `gateway.register` بدون auth → hijack webhook | `SERVICE_HMAC_SECRET` + service_id/key | 07-10 |
| ads-bot | تأیید/رد کمپین بدون چک ادمین‌بودن فرستنده callback | admin check | (قبل‌تر) |
| **fraud-engine** | **fail-open auth** — `ADMIN_KEY` خالی → همه‌ی `/admin/*` باز | fail-closed + `ConstantTimeCompare` + `log.Fatal` در startup | **07-14** |
| **vpn-bot** | **double-spend race** در `confirmBuyWithBalance` (چک/کسر غیراتمیک) | `DeductBalanceIfEnough` اتمیک (`WHERE balance>=amount`) | **07-14** |
| **vpn-bot** | **نبود dedup** در `verifyOnlinePayment` — کلیک تکراری = چند اشتراک | `ClaimOnlinePayment` + ایندکس partial یکتا روی `(gateway, ref_code)` | **07-14** |
| **archive-bot** | باگ کوچک `botUsername` ست‌نشده (hint خراب) | ست در `NewHandler` | **07-14** |

جزئیات فایل/خط در `CHANGELOG.md` (ورودی ۲۰۲۶-۰۷-۱۴).

## بدهی امنیتی باز

### بحرانی — Secret leakage
همه‌ی ۱۹ فایل `.env` با مقدار واقعی در git tracked و روی remote عمومی
`github.com/mrjvadi/CreatorBot.git` push شده‌اند. `ENCRYPTION_KEY` ریشه از اولین کامیت تا
HEAD تغییر نکرده. **rotation کامل + بازنویسی history اجباری** (کارگاه C، تصمیم کاربر:
کامل شامل ENCRYPTION_KEY + `git filter-repo` + force-push). چون قبلاً روی GitHub رفته،
توکن‌های خارجی (BotFather، toncenter، پنل VPN) باید **باطل** شوند، نه فقط عوض.

### hotspot های source-service (audit عمیق‌تر لازم، بدون تغییر کد فعلاً)
- `internal/userbot/run_bot_command.go` — مرز authorization: هرکس روی subject pool
  publish کند، دستور دلخواه به هر `BotUsername` اجرا می‌شود و فایل جواب آرشیو می‌شود.
  باید بررسی شود چه کسی می‌تواند trigger کند.
- `internal/telegram/dbsession.go` — session کامل MTProto در DB ذخیره می‌شود؛ لو رفتن =
  takeover کامل اکانت. رمزنگاری ذخیره باید بررسی شود.
- `internal/telegram/client.go` — شماره تلفن لاگ می‌شود.

### سایر hotspot ها (از audit ۲۰۲۶-۰۷-۱۴، اولویت پایین‌تر)
- vpn-bot `admin.go` — دو `return` پشت‌سرهم (unreachable) و دستورهای تبلیغ‌شده‌ی
  `/block /unblock /addbalance` که در Register هندل نشده‌اند (nonfunctional، نه امنیتی).
- fraud-engine — endpoint های `/score/*` عمداً public؛ NATS score handlers auth سطح‌پیام
  ندارند (به creds NATS تکیه دارند).
