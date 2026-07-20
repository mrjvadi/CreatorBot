#!/bin/bash
# CreatorBot V3 — Local Test Runner
# اجرا: chmod +x run.sh && ./run.sh
# توقف:  ./run.sh stop

set -e
ROOT="$(cd "$(dirname "$0")" && pwd)"
LOG_DIR="$ROOT/.logs"
PID_FILE="$ROOT/.run.pids"

mkdir -p "$LOG_DIR"

# ─── stop ────────────────────────────────────────────────────
if [ "$1" = "stop" ]; then
  if [ ! -f "$PID_FILE" ]; then echo "هیچ سرویسی در حال اجرا نیست."; exit 0; fi
  echo "⏹  در حال توقف سرویس‌ها..."
  while IFS= read -r line; do
    name="${line%%:*}"; pid="${line##*:}"
    kill "$pid" 2>/dev/null && echo "  ✓ $name ($pid)" || echo "  – $name ($pid) قبلاً متوقف شده"
  done < "$PID_FILE"
  rm -f "$PID_FILE"
  echo "✅ همه سرویس‌ها متوقف شدند."
  exit 0
fi

# ─── start check ─────────────────────────────────────────────
if [ -f "$PID_FILE" ]; then
  echo "⚠️  سرویس‌ها قبلاً در حال اجرا هستند. ابتدا: ./run.sh stop"
  exit 1
fi

# ─── helpers ─────────────────────────────────────────────────
start_service() {
  local name="$1"
  local dir="$2"
  local main_pkg="$3"
  echo -n "  ▶ $name ... "
  (
    cd "$dir"
    go run "$main_pkg" >> "$LOG_DIR/$name.log" 2>&1
  ) &
  local pid=$!
  echo "$name:$pid" >> "$PID_FILE"
  echo "PID $pid  (لاگ: .logs/$name.log)"
}

# ─── start ───────────────────────────────────────────────────
echo ""
echo "╔══════════════════════════════════════╗"
echo "║   CreatorBot V3  —  Local Testing   ║"
echo "╚══════════════════════════════════════╝"
echo ""
echo "📦 در حال راه‌اندازی سرویس‌ها..."
echo ""

# ۱. log-collector — قبل از همه، تا لاگ‌های startup بقیه‌ی سرویس‌ها هم
#    (که با log.AttachNATS به آن وصل می‌شوند) از همون اول جمع بشه
start_service "log-collector" "$ROOT/log-collector" "./cmd/..."
sleep 1

# ۲. botpay — باید زود بالا بیاد چون بقیه به آن وابسته‌اند
start_service "botpay"      "$ROOT/botpay"      "./cmd/..."
sleep 2

# ۳. image-registry و license-service — agentmanager (چک image، fail-closed)
#    و botmanager (صدور لایسنس) به این دو وابسته‌اند
start_service "image-registry"  "$ROOT/image-registry"  "./cmd/..."
start_service "license-service" "$ROOT/license-service" "./cmd/..."
sleep 1

# ۴. سرویس‌های پشتیبان (بدون ربات)
start_service "fraud-engine"    "$ROOT/fraud-engine"      "./cmd/..."
start_service "revenue-service" "$ROOT/revenue-service"   "./cmd/..."
start_service "community-service" "$ROOT/community-service" "./cmd/..."
sleep 1

# ۵. webhook-gateway — دروازه‌ی webhook تلگرام (لازم اگر هر رباتی BOT_MODE=webhook باشد)
start_service "webhook-gateway" "$ROOT/webhook-gateway" "./cmd/..."
sleep 1

# ۶. member-bot — زیرساخت چک عضویت (قبل از botmanager)
start_service "member-bot"  "$ROOT/member-bot"  "./cmd/bot/..."
sleep 1

# ۷. ads-bot
start_service "ads-bot"     "$ROOT/ads-bot"     "./cmd/..."
sleep 1

# ۸. agentmanager — مدیریت deploy container های کاربران
#    (uploader-bot دیگر این‌جا اجرا نمی‌شود — به‌صورت container داینامیک
#    توسط agentmanager ساخته می‌شود؛ برای dev دستی: cd uploader-bot && go run ./cmd/bot)
start_service "agentmanager" "$ROOT/agentmanager" "./cmd/..."
sleep 1

# ۹. apimanager — دروازه‌ی HTTP (قبل از botmanager؛ به NATS/DB مشترک وصل است)
start_service "apimanager"  "$ROOT/apimanager"  "./cmd/..."
sleep 1

# ۱۰. botmanager — آخرین (به همه وابسته است)
start_service "botmanager"  "$ROOT/botmanager"  "./cmd/..."

echo ""
echo "✅ همه سرویس‌ها در پس‌زمینه شروع شدند."
echo ""
echo "📋 لاگ‌ها:"
echo "   tail -f .logs/<نام>.log"
echo "   tail -f .logs/*.log  (همه با هم)"
echo ""
echo "⏹  توقف:  ./run.sh stop"
echo ""

# ─── انتظار برای Ctrl+C ──────────────────────────────────────
trap '
  echo ""
  echo "⏹  دریافت سیگنال — در حال توقف..."
  bash "$ROOT/run.sh" stop
  exit 0
' INT TERM

# نگه داشتن اسکریپت تا Ctrl+C
wait
