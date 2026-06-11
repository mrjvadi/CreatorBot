#!/usr/bin/env bash
# ============================================================
# CreatorBot — Setup Script
# ============================================================
# این اسکریپت:
#   1. وابستگی‌ها رو بررسی می‌کنه (docker, git, ...)
#   2. فایل .env رو از مقادیر کاربر می‌سازه
#   3. کلیدها و رمزها رو تولید می‌کنه
#   4. centrifugo.json رو می‌سازه
#   5. docker compose رو اجرا می‌کنه
# ============================================================
set -euo pipefail

# ── Colors ──────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; NC='\033[0m'
BOLD='\033[1m'

info()    { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
error()   { echo -e "${RED}[ERR]${NC}  $*"; exit 1; }
prompt()  { echo -e "${CYAN}${BOLD}$*${NC}"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$SCRIPT_DIR/.env"

echo ""
echo -e "${BOLD}╔══════════════════════════════════════╗${NC}"
echo -e "${BOLD}║     CreatorBot — Setup Wizard        ║${NC}"
echo -e "${BOLD}╚══════════════════════════════════════╝${NC}"
echo ""

# ─────────────────────────────────────────────────────────────
# 1. Dependency checks
# ─────────────────────────────────────────────────────────────
info "بررسی وابستگی‌ها..."

check_cmd() {
  if ! command -v "$1" &>/dev/null; then
    error "$1 نصب نیست. لطفاً ابتدا نصب کنید."
  fi
  success "$1 موجود است"
}

check_cmd docker
check_cmd git

# Docker Compose v2
if ! docker compose version &>/dev/null; then
  error "docker compose (v2) نصب نیست."
fi
success "docker compose موجود است"

echo ""

# ─────────────────────────────────────────────────────────────
# 2. Check existing .env
# ─────────────────────────────────────────────────────────────
if [[ -f "$ENV_FILE" ]]; then
  warn "فایل .env از قبل وجود دارد."
  read -rp "آیا می‌خواهید آن را بازنویسی کنید؟ [y/N] " overwrite
  if [[ "$overwrite" != "y" && "$overwrite" != "Y" ]]; then
    info "Setup بدون تغییر .env ادامه می‌یابد."
    SKIP_ENV=true
  fi
fi

# ─────────────────────────────────────────────────────────────
# 3. Collect user input
# ─────────────────────────────────────────────────────────────
if [[ "${SKIP_ENV:-false}" != "true" ]]; then
  echo ""
  echo -e "${BOLD}── اطلاعات ربات ──────────────────────────────${NC}"
  prompt "توکن ربات اصلی (BotFather):"
  read -rp "> " BOT_TOKEN

  prompt "آیدی عددی ادمین (Owner ID):"
  read -rp "> " OWNER_ID

  echo ""
  echo -e "${BOLD}── دامنه و SSL ────────────────────────────────${NC}"
  prompt "دامنه اصلی سرور (مثال: example.com) — خالی بگذارید برای localhost:"
  read -rp "> " DOMAIN
  DOMAIN="${DOMAIN:-localhost}"

  prompt "ایمیل برای Let's Encrypt (ACME):"
  read -rp "> " ACME_EMAIL
  ACME_EMAIL="${ACME_EMAIL:-admin@${DOMAIN}}"

  echo ""
  echo -e "${BOLD}── پورت‌ها ──────────────────────────────────────${NC}"
  prompt "پورت PostgreSQL [5434]:"
  read -rp "> " POSTGRES_PORT
  POSTGRES_PORT="${POSTGRES_PORT:-5434}"

  prompt "پورت Redis [6381]:"
  read -rp "> " REDIS_PORT
  REDIS_PORT="${REDIS_PORT:-6381}"

  prompt "پورت API [8086]:"
  read -rp "> " API_PORT
  API_PORT="${API_PORT:-8086}"

  prompt "پورت Centrifugo [8001]:"
  read -rp "> " CENTRIFUGO_PORT
  CENTRIFUGO_PORT="${CENTRIFUGO_PORT:-8001}"

  prompt "پورت Lock API (member-bot) [8082]:"
  read -rp "> " LOCK_API_PORT
  LOCK_API_PORT="${LOCK_API_PORT:-8082}"

  echo ""
  echo -e "${BOLD}── VPN Panel ────────────────────────────────────${NC}"
  prompt "نوع پنل VPN [marzban]:"
  read -rp "> " PANEL_TYPE
  PANEL_TYPE="${PANEL_TYPE:-marzban}"

  prompt "آدرس پنل VPN (مثال: https://panel.example.com):"
  read -rp "> " PANEL_URL

  prompt "نام کاربری پنل:"
  read -rp "> " PANEL_USERNAME

  prompt "رمز عبور پنل:"
  read -rsp "> " PANEL_PASSWORD
  echo ""

  echo ""
  echo -e "${BOLD}── درگاه پرداخت ─────────────────────────────────${NC}"
  prompt "درگاه پرداخت [zarinpal/nowpayments/card]:"
  read -rp "> " PAYMENT_GATEWAY
  PAYMENT_GATEWAY="${PAYMENT_GATEWAY:-zarinpal}"

  ZARINPAL_MERCHANT=""
  NOWPAYMENTS_KEY=""
  CARD_NUMBER=""
  CARD_OWNER=""

  if [[ "$PAYMENT_GATEWAY" == "zarinpal" ]]; then
    prompt "Merchant ID زرین‌پال:"
    read -rp "> " ZARINPAL_MERCHANT
  elif [[ "$PAYMENT_GATEWAY" == "nowpayments" ]]; then
    prompt "API Key ناو‌پیمنتس:"
    read -rp "> " NOWPAYMENTS_KEY
  elif [[ "$PAYMENT_GATEWAY" == "card" ]]; then
    prompt "شماره کارت:"
    read -rp "> " CARD_NUMBER
    prompt "نام صاحب کارت:"
    read -rp "> " CARD_OWNER
  fi

  echo ""
  echo -e "${BOLD}── ربات‌های چک ممبر ─────────────────────────────${NC}"
  prompt "توکن ربات‌های چک ممبر (با کاما جدا کنید):"
  read -rp "> " CHECKER_BOT_TOKENS

  # ─────────────────────────────────────────────────────────────
  # 4. Generate secrets
  # ─────────────────────────────────────────────────────────────
  info "تولید کلیدهای امنیتی..."

  gen_hex()    { openssl rand -hex "$1"; }
  gen_pass()   { openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 32; }

  POSTGRES_PASSWORD=$(gen_hex 16)
  REDIS_PASSWORD=$(gen_hex 16)
  ENCRYPTION_KEY=$(gen_hex 32)
  JWT_ACCESS_SECRET=$(gen_hex 32)
  JWT_REFRESH_SECRET=$(gen_hex 32)
  LOCK_API_SECRET=$(gen_hex 24)
  CENTRIFUGO_API_KEY=$(gen_hex 16)
  CENTRIFUGO_TOKEN=$(gen_hex 16)
  GRAFANA_PASSWORD=$(gen_hex 16)
  SERVER_ID=$(python3 -c "import uuid; print(uuid.uuid4())" 2>/dev/null || cat /proc/sys/kernel/random/uuid 2>/dev/null || gen_hex 16)

  # Build DSN strings
  MASTER_DSN="postgres://botuser:${POSTGRES_PASSWORD}@postgres:5432/botmanager?sslmode=disable"
  UPLOADER_DSN="postgres://botuser:${POSTGRES_PASSWORD}@postgres:5432/uploader_bot?sslmode=disable"
  VPN_DSN="postgres://botuser:${POSTGRES_PASSWORD}@postgres:5432/vpn_bot?sslmode=disable"
  ARCHIVE_DSN="postgres://botuser:${POSTGRES_PASSWORD}@postgres:5432/archive_bot?sslmode=disable"
  MEMBER_DSN="postgres://botuser:${POSTGRES_PASSWORD}@postgres:5432/member_bot?sslmode=disable"
  SOURCE_DSN="postgres://botuser:${POSTGRES_PASSWORD}@postgres:5432/source_svc?sslmode=disable"

  # ─────────────────────────────────────────────────────────────
  # 5. Write .env
  # ─────────────────────────────────────────────────────────────
  info "نوشتن فایل .env..."

  cat > "$ENV_FILE" << ENVEOF
# ============================================================
# CreatorBot — Environment Configuration
# Generated by setup.sh on $(date)
# ============================================================

# ===== Domain =====
DOMAIN=${DOMAIN}
ACME_EMAIL=${ACME_EMAIL}

# ===== Bot =====
BOT_TOKEN=${BOT_TOKEN}
OWNER_ID=${OWNER_ID}

# ===== Database =====
MASTER_DSN=${MASTER_DSN}
UPLOADER_DSN=${UPLOADER_DSN}
VPN_DSN=${VPN_DSN}
ARCHIVE_DSN=${ARCHIVE_DSN}
MEMBER_DSN=${MEMBER_DSN}
SOURCE_DSN=${SOURCE_DSN}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_PORT=${POSTGRES_PORT}

# ===== Redis =====
REDIS_ADDR=redis:6379
REDIS_PASSWORD=${REDIS_PASSWORD}
REDIS_DB=0
REDIS_PORT=${REDIS_PORT}

# ===== API =====
API_ADDR=apimanager:${API_PORT}
API_PORT=${API_PORT}

# ===== Centrifugo =====
CENTRIFUGO_API_ENDPOINT=http://centrifugo:8000/api
CENTRIFUGO_API_KEY=${CENTRIFUGO_API_KEY}
CENTRIFUGO_WS_ENDPOINT=ws://centrifugo:8000/connection/websocket
CENTRIFUGO_TOKEN=${CENTRIFUGO_TOKEN}
CENTRIFUGO_PORT=${CENTRIFUGO_PORT}

# ===== Security =====
ENCRYPTION_KEY=${ENCRYPTION_KEY}
JWT_ACCESS_SECRET=${JWT_ACCESS_SECRET}
JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}

# ===== Lock API =====
LOCK_API_PORT=${LOCK_API_PORT}
LOCK_API_SECRET=${LOCK_API_SECRET}

# ===== VPN Panel =====
PANEL_TYPE=${PANEL_TYPE}
PANEL_URL=${PANEL_URL}
PANEL_USERNAME=${PANEL_USERNAME}
PANEL_PASSWORD=${PANEL_PASSWORD}

# ===== Payment =====
PAYMENT_GATEWAY=${PAYMENT_GATEWAY}
ZARINPAL_MERCHANT=${ZARINPAL_MERCHANT}
NOWPAYMENTS_KEY=${NOWPAYMENTS_KEY}
CARD_NUMBER=${CARD_NUMBER}
CARD_OWNER=${CARD_OWNER}

# ===== Grafana =====
GRAFANA_PASSWORD=${GRAFANA_PASSWORD}

# ===== Agent =====
SERVER_ID=${SERVER_ID}
HEARTBEAT_INTERVAL_SEC=5

# ===== Central Locks =====
CHECKER_BOT_TOKENS=${CHECKER_BOT_TOKENS}

# ===== Source Service =====
TG_APP_ID=
TG_APP_HASH=
TG_PHONE=
TG_SESSION_FILE=/app/sessions/session.json
TG_SOURCE_CHANNEL=
TG_DELIVERY_CHANNEL=
ENVEOF

  success ".env ساخته شد"

  # ─────────────────────────────────────────────────────────────
  # 6. Write centrifugo.json with real values
  # ─────────────────────────────────────────────────────────────
  cat > "$SCRIPT_DIR/centrifugo.json" << CEOF
{
  "token_hmac_secret_key": "${CENTRIFUGO_TOKEN}",
  "api_key": "${CENTRIFUGO_API_KEY}",
  "admin": true,
  "admin_password": "${CENTRIFUGO_API_KEY}",
  "admin_secret": "${CENTRIFUGO_API_KEY}",
  "allowed_origins": ["*"],
  "namespaces": [
    {
      "name": "server",
      "publish": false,
      "history_size": 10,
      "history_ttl": "60s"
    }
  ]
}
CEOF
  success "centrifugo.json ساخته شد"
fi

# ─────────────────────────────────────────────────────────────
# 7. Load env and start
# ─────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}── راه‌اندازی سرویس‌ها ──────────────────────────${NC}"
read -rp "آیا می‌خواهید docker compose را اجرا کنید؟ [Y/n] " start_now
if [[ "$start_now" == "n" || "$start_now" == "N" ]]; then
  echo ""
  info "برای راه‌اندازی دستی اجرا کنید:"
  echo "  cd deploy && docker compose --env-file .env up -d"
  exit 0
fi

info "ساخت و راه‌اندازی..."
cd "$SCRIPT_DIR"
docker compose --env-file .env up -d --build

echo ""
success "همه سرویس‌ها راه‌اندازی شدند!"
echo ""
echo -e "${BOLD}── اطلاعات دسترسی ──────────────────────────────${NC}"

# Load env for display
set -a; source "$ENV_FILE"; set +a

echo -e "  API:        ${CYAN}https://api.${DOMAIN}${NC}"
echo -e "  Grafana:    ${CYAN}https://grafana.${DOMAIN}${NC}  (admin / ${GRAFANA_PASSWORD:-***})"
echo -e "  Centrifugo: ${CYAN}https://centrifugo.${DOMAIN}${NC}"
echo ""
warn "فایل .env را در جای امنی نگهداری کنید — حاوی تمام کلیدهای محرمانه است!"
echo ""
