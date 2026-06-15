# Changelog — CreatorBot V3

## [Unreleased] — Sprint 1-9

### Added
- Self-service bot provisioning — کاربر بدون دخالت ادمین ربات می‌سازد
- Double-entry ledger در botpay
- Prometheus metrics در همه سرویس‌ها
- Grafana + Loki در docker-compose
- Audit log برای همه عملیات حساس
- Secret rotation با dual-key grace period
- Rate limiting (token bucket) در webhook-gateway
- Health check endpoints در همه سرویس‌ها
- Kubernetes manifests (16 manifest)
- Config versioning با rollback
- VPN adapters: Hiddify، X-UI، MarzNeshin
- E2E test suite
- Migration system (golang-migrate)
- Panic recovery در همه handler ها
- Context timeout در عملیات طولانی
- DB connection pool با lifecycle management

### Fixed
- uuid=text bug در PostgreSQL JOIN queries
- telebot v4 API: Photo/ForwardFrom/InlineResult
- NATS Authorization Violation در startup
- Duplicate ApprovePayment در member-bot
- Bot auto-state در admin list handlers
- Format string مشکلات در i18n

### Architecture
- NATS JetStream جایگزین Centrifugo WebSocket
- Bot engines مستقیم به DB وصل می‌شوند (نه apimanager)
- instance_id = bot_<BotID> برای persistence
- fraud-engine از request/reply NATS
