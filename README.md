# CreatorBot V3

پلتفرم SaaS برای ساخت و مدیریت ربات‌های تلگرام — کاملاً self-service.

## معماری

```
Telegram
   │ webhook
   ▼
webhook-gateway (:8090)
   │
   ├── botmanager     ← ربات اصلی فروش
   ├── botpay (:8087) ← کیف پول TON
   └── bot engines    ← ربات‌های کاربران
         │
   apimanager (:8080)
         │ NATS JetStream
         │
   agentmanager → Docker
         │
   ┌─────┼──────────────┐
   │     │              │
fraud  revenue  community-service
(:8092) (:8088)  (:8093)
```

## سرویس‌ها

| سرویس | پورت | توضیح |
|-------|------|-------|
| botmanager | - | ربات اصلی — فروش و مدیریت |
| apimanager | 8080 | REST API + JWT auth |
| botpay | 8087 | کیف پول TON |
| agentmanager | - | Docker orchestration |
| webhook-gateway | 8090 | دریافت webhook تلگرام |
| revenue-service | 8088 | تقسیم درآمد |
| community-service | 8093 | مدیریت کامیونیتی |
| fraud-engine | 8092 | تشخیص تقلب |

## نوع‌های ربات پشتیبانی‌شده

- 🌐 **VPN** — Marzban، MarzNeshin، Hiddify، 3x-ui
- 📤 **Uploader** — آپلود و مدیریت فایل
- 🔒 **Member** — قفل ممبرشیپ کانال/گروه
- 📦 **Archive** — آرشیو فایل با جستجو

## راه‌اندازی سریع

```bash
# ۱. کپی env
cp .env.example .env
# ویرایش .env با مقادیر واقعی

# ۲. راه‌اندازی services
docker compose up -d

# ۳. migration
make install-migrate
make migrate-up

# ۴. بررسی وضعیت
docker compose ps
curl http://localhost:8080/health
```

## Kubernetes

```bash
# ویرایش secrets
vim deploy/k8s/base/secret.yaml

# deploy
kubectl apply -k deploy/k8s/

# بررسی
kubectl get pods -n creatorbot
```

## تست

```bash
# unit tests
go test ./...

# integration tests (نیاز به service های واقعی)
E2E=true go test ./tests/e2e/...

# VPN adapter tests
go test ./shared/pkg/adapters/...
```

## متریک‌ها

- Prometheus: `http://localhost:9090/metrics` (هر سرویس)
- Grafana: `http://localhost:3000`
- Loki logs: از طریق Grafana

## قوانین معماری

1. bot ها هرگز `apimanager` را صدا نمی‌زنند — مستقیم به DB وصل می‌شوند
2. `instance_id = bot_<BotID>` از توکن تلگرام
3. fraud-engine از NATS request/reply
4. همه event ها از NATS JetStream
5. هر تراکنش مالی double-entry ledger دارد

## NATS Events

| Event | Publisher | Subscriber |
|-------|-----------|------------|
| `service.creation.*` | botmanager | agentmanager |
| `agent.*.deploy` | apimanager | agentmanager |
| `plan.upgraded` | botmanager | همه سرویس‌ها |
| `wallet.deposit` | botpay | revenue-service |
| `campaign.revenue.generated` | ads-bot | community-service |
| `config.updated` | botmanager | همه سرویس‌ها |
| `fraud.*.score.*` | fraud-engine | botmanager |
