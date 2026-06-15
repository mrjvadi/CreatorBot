# CreatorBot V3 — Makefile

DB_URL ?= $(POSTGRES_DSN)
MIGRATE = migrate -path deploy/migrations -database "$(DB_URL)"

.PHONY: migrate-up migrate-down migrate-status db-reset

## اجرای همه migration ها
migrate-up:
	$(MIGRATE) up

## rollback یک migration
migrate-down:
	$(MIGRATE) down 1

## وضعیت migration ها
migrate-status:
	$(MIGRATE) version

## reset کامل DB (فقط development)
db-reset:
	$(MIGRATE) drop -f
	$(MIGRATE) up

## نصب golang-migrate
install-migrate:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

## build همه سرویس‌ها
build-all:
	@for svc in botmanager apimanager botpay agentmanager webhook-gateway \
	            revenue-service community-service fraud-engine; do \
		echo "Building $$svc..."; \
		go build -o bin/$$svc ./$$svc/cmd/...; \
	done

## docker compose up
up:
	docker compose up -d

## docker compose down
down:
	docker compose down

## لاگ همه سرویس‌ها
logs:
	docker compose logs -f --tail=50
