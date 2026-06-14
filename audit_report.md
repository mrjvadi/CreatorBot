# CreatorBot V3 — Full Architecture Audit Report

## 1. Service Status Matrix

| Service | Status | Coverage |
|---------|--------|----------|
| botmanager | ✅ Implemented | 85% — Stop/Start/Restart UI missing |
| botpay | ✅ Implemented | 95% — Double-entry ledger missing |
| apimanager | ✅ Implemented | 80% — Rate limiting, tests missing |
| agentmanager | ✅ Implemented | 90% — K8s, metrics missing |
| webhook-gateway | ✅ Implemented | 95% |
| fraud-engine | ✅ Implemented | 90% |
| revenue-service | ⚠️ Partial | 70% — Channel split, community integration missing |
| community-service | ⚠️ Partial | 75% — CampaignAttribution moved elsewhere |
| ads-bot | ⚠️ Partial | 75% — campaign.revenue.generated not published |
| member-bot | ✅ Implemented | 85% — Multi-bot scaling missing |
| uploader-bot | ✅ Implemented | 90% |
| vpn-bot | ✅ Implemented | 85% — 3 adapters missing |
| archive-bot | ✅ Implemented | 90% |
| configstore | ✅ Implemented | 80% — Versioning missing |

---

## 2. Critical Gaps (باید قبل از production رفع شود)

### GAP-1: NATS Events ناقص
مشکل: چندین event که سرویس‌ها subscribe می‌کنند هرگز publish نمی‌شوند.

| Event | Publisher | Subscriber |
|-------|-----------|------------|
| config.updated | ❌ هیچ‌کس | configstore |
| community.revenue.generated | ❌ هیچ‌کس | community-service |
| campaign.revenue.generated | ❌ هیچ‌کس | community-service |
| wallet.deposit / wallet.withdraw | ❌ هیچ‌کس | — |

راه‌حل: botpay باید wallet.deposit publish کند. ads-bot باید campaign.revenue.generated publish کند. configstore باید config.updated publish کند.

---

### GAP-2: Revenue-Service — Channel split پیاده نشده
مشکل: revenue-service قانون channel 90/10 ندارد — فقط subscription و lock دارد.
community-service این منطق را دارد ولی revenue-service با آن integrate نشده.

راه‌حل: SeedDefaultRules باید channel_revenue rule اضافه کند.

---

### GAP-3: Campaign Attribution → Community-Service
مشکل: ads-bot وقتی join ثبت می‌کند، به community-service خبر نمی‌دهد.
بنابراین community-service نمی‌داند که این join از campaign Y بوده.

راه‌حل: ads-bot باید بعد از RecordJoin یک event publish کند.

---

### GAP-4: botmanager — Stop/Start/Restart UI
مشکل: apimanager این endpoint‌ها را دارد ولی botmanager UI ندارد.

---

### GAP-5: plan.upgraded event برای quota cache
مشکل: وقتی کاربر پلن ارتقا می‌دهد، quota در Redis آپدیت نمی‌شود.
تا restart بعدی، کاربر ممکن است ظرفیت قدیمی را ببیند.

---

## 3. Architecture Diagram

```
TELEGRAM
   │ webhook
   ▼
webhook-gateway (:8090)
   │ NATS webhook.{bot_id}
   ├── botmanager     → apimanager (:8080)
   ├── botpay         │    │ NATS agent.*.deploy
   └── ads-bot        │    ▼
                      │ agentmanager → Docker
                      │
                    NATS JetStream (:4222)
                      │
        ┌─────────────┼──────────────┐
        │             │              │
   fraud-engine  revenue-service  community-service
   (:8092)       (:8088)          (:8093)
        │             │              │
        └─────────────┴──────────────┘
                      │
                   botpay (:8087)
                   [wallet ops]
```

---

## 4. Database Analysis

### PostgreSQL
- ✅ UUID PKs با gen_random_uuid()
- ✅ Soft delete (DeletedAt)
- ✅ Unique constraints
- ❌ Explicit FK constraints (فقط GORM implicit)
- ❌ Audit log tables
- ❌ Migration system (فقط AutoMigrate)

### MongoDB
- ✅ همه collections با instance_id فیلتر می‌کنند (tenant isolation)
- ✅ fraud-engine با database جداگانه
- ❌ Index documentation
- ❌ Schema validation

### Redis
- ✅ TTL روی همه wizard states
- ⚠️ Redis به عنوان job queue استفاده می‌شود (member-bot stream) — acceptable
- ❌ Distributed lock pattern مستند نشده

---

## 5. NATS Event Analysis

### Implemented ✅
- membership.joined / left
- earning.created
- fraud.detected
- community.reward.created
- gateway.register / unregister
- agent.*.deploy / heartbeat / result

### Missing ❌
- config.updated (publish)
- wallet.deposit / wallet.withdraw
- campaign.revenue.generated (publish from ads-bot)
- community.revenue.generated (publish)
- plan.upgraded

---

## 6. Security Analysis

| Item | Status |
|------|--------|
| Token encryption (AES-256-GCM) | ✅ |
| JWT Auth | ✅ |
| Financial Consensus (4 workers) | ✅ |
| TON TX deduplication | ✅ |
| User blocking | ✅ |
| API rate limiting | ❌ |
| Audit logs | ❌ |
| Admin action logging | ❌ |
| Secret rotation | ❌ |

---

## 7. Financial Analysis

| Item | Status |
|------|--------|
| Row-level locking | ✅ |
| Atomic DB transactions | ✅ |
| Balance hold/freeze | ✅ |
| TON dedup (seenTx) | ✅ |
| Consensus workers | ✅ |
| Double-entry ledger | ❌ |
| Balance reconciliation | ❌ |
| Withdrawal audit trail | ⚠️ Partial |

---

## 8. Missing Features (Priority Order)

### 🔴 P0 — قبل از هر production test
1. config.updated publish (از botmanager/apimanager)
2. ads-bot: campaign.revenue.generated publish
3. revenue-service: channel 90/10 rule
4. botpay: wallet.deposit/withdraw publish

### 🟠 P1 — هفته اول
5. botmanager: Stop/Start/Restart UI
6. plan.upgraded NATS event + quota reset
7. Redis rate limiting در apimanager/webhook-gateway

### 🟡 P2 — هفته دوم
8. Audit log table (PostgreSQL)
9. Migration system (golang-migrate یا goose)
10. member-bot: multi-bot channel assignment
11. Health check endpoints همه سرویس‌ها

### 🟢 P3 — آینده
12. Double-entry accounting
13. Config versioning
14. VPN adapters (Hiddify, X-UI, MarzNeshin)
15. Kubernetes manifests
16. Prometheus metrics
17. Centralized logging (Loki)

---

## 9. Technical Debt

| Item | Severity | Description |
|------|----------|-------------|
| GORM AutoMigrate | Medium | باید با migration tool جایگزین شود |
| Hardcoded Persian strings | Low | user_bot.go هنوز 93 string دارد |
| source-service | Low | فقط stub — حذف یا پیاده‌سازی |
| No tests | High | هیچ unit/integration test ندارد |
| Error handling inconsistent | Medium | برخی جاها err نادیده گرفته می‌شود |

---

## 10. Production Readiness Score

| Category | Score | Notes |
|----------|-------|-------|
| Core functionality | 78/100 | GAP-1,2,3 رفع نشده |
| Security | 70/100 | Rate limit، audit log ندارد |
| Financial integrity | 75/100 | Double-entry ندارد |
| Reliability | 65/100 | Test ندارد، مهاجرت DB ندارد |
| Observability | 40/100 | فقط structured log |
| **Overall** | **66/100** | |

---

## 11. Recommended Roadmap

### Sprint 1 (این هفته) — P0 fixes
- [ ] config.updated publish در botmanager
- [ ] wallet.deposit publish در botpay
- [ ] campaign.revenue.generated در ads-bot
- [ ] channel revenue rule در revenue-service

### Sprint 2 — P1 features
- [ ] Stop/Start/Restart UI در botmanager
- [ ] plan.upgraded event
- [ ] Rate limiting middleware

### Sprint 3 — Quality
- [ ] Unit tests برای scorer، engine، store
- [ ] Migration system
- [ ] Audit log

### Sprint 4 — Scale
- [ ] Multi-bot member assignment
- [ ] Health checks + Prometheus
- [ ] VPN adapters
