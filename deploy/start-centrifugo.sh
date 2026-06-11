#!/usr/bin/env bash
# راه‌اندازی Centrifugo با env vars
# اجرا: bash deploy/start-centrifugo.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: .env file not found at $ENV_FILE"
  exit 1
fi

# load env
set -a; source "$ENV_FILE"; set +a

# راه‌اندازی centrifugo با env vars مستقیم
# centrifugo v5 از CENTRIFUGO_ prefix برای override همه config fields پشتیبانی می‌کنه
exec centrifugo \
  --config="$SCRIPT_DIR/centrifugo.json" \
  --token_hmac_secret_key="${CENTRIFUGO_TOKEN}" \
  --api_key="${CENTRIFUGO_API_KEY}" \
  --admin_password="${CENTRIFUGO_API_KEY}" \
  --admin_secret="${CENTRIFUGO_API_KEY}" \
  --address="127.0.0.1" \
  --port=8001
