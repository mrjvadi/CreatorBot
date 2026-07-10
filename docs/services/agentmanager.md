# agentmanager

## این سرویس چیست
سرویسی که روی هر سرور فیزیکی پلتفرم اجرا می‌شود و واقعاً Docker container های ربات‌های مشتری را می‌سازد/متوقف/حذف می‌کند. تنها سرویسی که مستقیم با Docker daemon حرف می‌زند (از طریق SDK، نه اجرای دستور shell).

## مسئولیت‌ها
- دریافت `DeployCommand` از NATS (subject `deploy.<server_id>`) و اجرای `deploy`/`stop`/`remove`/`restart`.
- **Whitelist اجباری image**: فقط image هایی که prefix شان در `ALLOWED_IMAGES` باشد قابل اجرا هستند؛ هیچ‌وقت از رجیستری pull نمی‌کند (image باید از قبل روی سرور موجود باشد) — جلوی اجرای image دستکاری‌شده از بیرون را می‌گیرد.
- سخت‌گیری امنیتی پیش‌فرض هر container: `no-new-privileges`, drop همه‌ی capability ها (فقط با لیست صریح دوباره اضافه می‌شوند)، محدودیت CPU/حافظه/تعداد پردازه (ضد fork-bomb)، امکان read-only rootfs + tmpfs برای `/tmp`.
- Heartbeat دوره‌ای به `apimanager`/`botmanager` با لیست container های در حال اجرا روی این سرور.
- به Docker daemon نه مستقیم بلکه از طریق یک `docker-socket-proxy` وصل می‌شود که فقط endpoint های لازم (CONTAINERS, IMAGES, POST) را باز گذاشته؛ EXEC/SWARM/PLUGINS بسته‌اند — یعنی حتی اگر خودِ agentmanager هک شود، دسترسی exec داخل container های دیگر یا کنترل کامل daemon ندارد.

## ارتباطات
- NATS: subscribe `deploy.<server_id>` (JetStream stream `DEPLOY`)، publish `agent.<server_id>.heartbeat` و `agent.<server_id>.result` (stream `AGENT`).
- Docker: از طریق `DOCKER_HOST=tcp://docker-socket-proxy:2375`.
- HTTP: `/health` روی پورت ۸۰۹۶، متریک روی ۹۰۹۳.

## ایرادها و نکات
- **بحرانی، رفع شد در این جلسه**: `Stop`/`Remove`/`Restart` قبلاً `container_id` را از NATS message بدون هیچ اعتبارسنجی مستقیم به Docker پاس می‌دادند — یعنی هرکس به NATS دسترسی داشت می‌توانست با یک پیام (`{"type":"remove","container_id":"postgres"}`) هر container ای روی هر سرور، از جمله زیرساخت خودِ پلتفرم، را force-remove کند. رفع شد با یک label (`creatorbot.managed=true`) که فقط روی container هایی گذاشته می‌شود که خودِ `agentmanager` ساخته؛ حالا قبل از هر عملیات مخرب‌پذیر، این label چک می‌شود.
- Whitelist مربوط به `Deploy` است؛ اگر `ALLOWED_IMAGES` خالی باشد، به‌درستی هیچ deploy ای مجاز نیست (fail-safe) — این رفتار از قبل درست بود.
- Heartbeat اطلاعات کامل container ها (نام، image، وضعیت) را روی یک subject بدون ACL منتشر می‌کند — یک مهاجم با دسترسی NATS می‌تواند این را subscribe کند و از توپولوژی/نام‌گذاری داخلی پلتفرم مطلع شود (شناسایی، نه دسترسی مستقیم). ریشه‌اش همان نبودِ ACL سطح-subject در NATS است که در گزارش امنیتی به‌عنوان بزرگ‌ترین مشکل باقی‌مانده مستند شده.
