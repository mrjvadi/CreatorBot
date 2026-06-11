# CreatorBot V3

پلتفرم مدیریت ربات‌های تلگرام — Go 1.22 + go.work + Ports & Adapters

## ساختار

```
CreatorBot/
├── go.work                 ← workspace root (همه ماژول‌ها اینجا ثبت می‌شن)
├── shared/                 ← کد مشترک بین همه سرویس‌ها
│   └── pkg/
│       ├── ports/          ← ← ← Interface ها (قرارداد)
│       │   ├── db.go           DB interface
│       │   ├── cache.go        Cache interface
│       │   ├── bot.go          BotSender interface
│       │   ├── payment.go      PaymentGateway interface
│       │   ├── vpn_panel.go    VPNPanel interface
│       │   └── notifier.go     Notifier + Logger interfaces
│       ├── adapters/       ← ← ← پیاده‌سازی‌ها (قابل جایگزینی)
│       │   ├── postgres/       implements ports.DB
│       │   ├── redis/          implements ports.Cache
│       │   ├── telebot/        implements ports.BotSender
│       │   ├── zarinpal/       implements ports.PaymentGateway
│       │   ├── nowpayments/    implements ports.PaymentGateway
│       │   ├── marzban/        implements ports.VPNPanel
│       │   └── centrifugo/     implements ports.Notifier
│       ├── config/         generic env loader (viper)
│       ├── auth/           JWT + AES-256-GCM
│       └── logger/         implements ports.Logger (zap)
│
├── botmanager/             ← پلتفرم مرکزی
├── uploader-bot/           ← ربات آپلودر فایل
├── vpn-bot/                ← ربات فروش VPN
├── archive-bot/            ← ربات آرشیو با جستجوی فازی
├── member-bot/             ← ربات فروش ممبر + worker pool
└── source-service/         ← سرویس آرشیو مرجع (userbot)
```

## چطور یک سرویس را جایگزین کنید

### مثال: تغییر دیتابیس از PostgreSQL به MySQL

```go
// shared/pkg/adapters/mysql/mysql.go
package mysql

import (
    "gorm.io/driver/mysql"
    "github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type DB struct { db *gorm.DB }
var _ ports.DB = (*DB)(nil) // compile-time check

func New(dsn string) (*DB, error) { ... }
// implement Conn(), Ping(), Migrate(), Close()
```

سپس فقط در `main.go` سرویس مورد نظر:
```go
// قبل:
db, _ := postgres.New(postgres.Config{DSN: cfg.DSN})
// بعد:
db, _ := mysql.New(cfg.DSN)
```

**هیچ فایل دیگری نیاز به تغییر ندارد.**

---

### مثال: اضافه کردن پنل VPN جدید (مثلاً Hiddify)

```go
// shared/pkg/adapters/hiddify/hiddify.go
package hiddify

type Panel struct { ... }
var _ ports.VPNPanel = (*Panel)(nil)

func New(baseURL, apiKey string) *Panel { ... }
// implement Name(), Login(), CreateUser(), ...
```

سپس در `vpn-bot/cmd/bot/main.go`:
```go
case "hiddify":
    panel = hiddify.New(cfg.PanelURL, cfg.PanelKey)
```

---

### مثال: تغییر سیستم notification از Centrifugo به NATS

```go
// shared/pkg/adapters/nats/nats.go
package nats

type Notifier struct { ... }
var _ ports.Notifier = (*Notifier)(nil)

func New(url string) *Notifier { ... }
func (n *Notifier) Publish(ctx context.Context, channel string, payload any) error { ... }
```

فقط در `botmanager/cmd/bot/main.go`:
```go
// قبل:
var notifier ports.Notifier = centrifugo.New(...)
// بعد:
var notifier ports.Notifier = nats.New(cfg.NatsURL)
```

## راه‌اندازی

```bash
# کلون و اجرا
git clone ...
cd CreatorBot

# هر سرویس
cd uploader-bot
cp .env.example .env
# ویرایش .env
go run ./cmd/bot

# یا با docker
docker compose -f deploy/docker-compose.yml up -d
```

## استک فنی

| بخش | پیاده‌سازی پیش‌فرض | جایگزین‌پذیر با |
|-----|----------|---------|
| Database | PostgreSQL (GORM) | MySQL, SQLite, ... |
| Cache/Queue | Redis | Memcached, in-memory, ... |
| Telegram | telebot.v4 | telegram-bot-api, ... |
| پرداخت | Zarinpal / NowPayments | هر gateway دیگری |
| پنل VPN | Marzban | Hiddify, 3x-ui, ... |
| Notification | Centrifugo | NATS, gRPC, WebSocket |
| Logging | Zap | slog, logrus, ... |
