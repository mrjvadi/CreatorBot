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

# ۱. botpay — اول باید بالا بیاد چون بقیه به آن وابسته‌اند
start_service "botpay"      "$ROOT/botpay"      "./cmd/..."
sleep 2

# ۲. سرویس‌های پشتیبان (بدون ربات)
start_service "fraud-engine"    "$ROOT/fraud-engine"      "./cmd/..."
start_service "revenue-service" "$ROOT/revenue-service"   "./cmd/..."
start_service "community-service" "$ROOT/community-service" "./cmd/..."
sleep 1

# ۳. member-bot — زیرساخت چک عضویت (قبل از botmanager)
start_service "member-bot"  "$ROOT/member-bot"  "./cmd/bot/..."
sleep 1

# ۴. ads-bot
start_service "ads-bot"     "$ROOT/ads-bot"     "./cmd/..."
sleep 1

# ۵. uploader-bot
start_service "uploader-bot" "$ROOT/uploader-bot" "./cmd/bot/..."
sleep 1

# ۶. agentmanager — مدیریت deploy container های کاربران
start_service "agentmanager" "$ROOT/agentmanager" "./cmd/..."
sleep 1

# ۷. botmanager — آخرین (به همه وابسته است)
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
