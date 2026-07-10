# apimanager

## این سرویس چیست
دروازه‌ی HTTP بیرونی پلتفرم — برای زمانی طراحی شده که یک وب‌سایت/اپ موبایل بخواهد بدون تلگرام با پلتفرم حرف بزند. طبق کامنت‌های خودِ کد («apimanager دیگر در مسیر hot path نیست») و CLAUDE.md پروژه، در حال حاضر کم‌استفاده است — ربات‌ها مستقیم به DB وصل می‌شوند، نه از طریق این سرویس.

## مسئولیت‌ها
- `POST /api/v1/auth/telegram` — ورود با Telegram Login Widget (امضای HMAC را با `verifyTelegramAuth` چک می‌کند، fail-closed اگر توکن ربات تنظیم نشده باشد، و `auth_date` را هم چک انقضا می‌کند)، صدور JWT access/refresh.
- `POST /api/v1/auth/refresh` — تمدید توکن.
- `POST /api/v1/agent/auth` و `agent.*` (پشت `AgentKeyAuth`) — احراز/heartbeat/نتیجه از سمت agentmanager (یک مسیر HTTP موازیِ همان چیزی که عمدتاً روی NATS انجام می‌شود).
- روت‌های کاربر (پشت `JWTAuth`): `GET /me`, `instances` CRUD، `plans`.
- روت‌های ادمین (پشت `JWTAuth` + `RequireRole("admin","owner")`): آمار، مدیریت سرور/تمپلیت.
- گوش‌دادن به `agent.*.heartbeat`/`agent.*.result` روی NATS برای به‌روزرسانی وضعیت سرور/instance در DB مشترک.

## ارتباطات
- HTTP روی پورت ۸۰۸۰ (پیش‌فرض)، متریک روی ۹۰۹۰.
- NATS: subscribe `agent.*.heartbeat`, `agent.*.result`.
- PostgreSQL مشترک (`shared-core/store`، همان مدل‌های `botmanager`).
- Rate limit ساده: ۶۰ درخواست در دقیقه به‌ازای IP روی مسیرهای کاربر.

## ایرادها و نکات
- **بررسی و تأیید شد (نه ایراد)**: یک عامل تحقیقاتی قبلی ادعا کرده بود `TelegramAuth` امضای HMAC ویجت لاگین تلگرام را هرگز چک نمی‌کند. این ادعا **غلط** بود — کد در `internal/handler/handler.go` واقعاً `verifyTelegramAuth(fields, req.Hash, h.telegramBotTok)` را صدا می‌زند، با رفتار fail-closed اگر توکن ربات پیکربندی نشده باشد، و انقضای `auth_date` را هم چک می‌کند. نیازی به تغییر نیست.
- **موازی‌کاری غیرضروری با NATS**: مسیر `agent.*` هم روی HTTP (پشت `AgentKeyAuth`) و هم روی NATS (subscribe مستقیم در همین سرویس، بدون auth چون NATS ACL ندارد) پیاده شده — دو مسیر برای یک کار، که نگه‌داری را سخت‌تر می‌کند و مسیر NATسی‌اش همان ضعف عمومی نبودِ ACL را دارد.
- این سرویس طبق طراحی «برای آینده» است و هنوز مصرف‌کننده‌ی واقعی (وب/اپ) ندارد — یعنی روت‌هایش کمتر از بقیه‌ی سرویس‌ها در عمل تست شده‌اند؛ قبل از افتتاح این API به مصرف‌کننده‌ی بیرونی واقعی، یک بازبینی امنیتی جداگانه (خصوصاً JWT secret rotation و rate limit روی auth endpoints) توصیه می‌شود.
