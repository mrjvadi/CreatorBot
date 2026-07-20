#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
while IFS= read -r mod; do
  dir="$(dirname "$mod")"
  echo "==> go test $dir/..."
  (cd "$root/$dir" && go test ./...)
done < <(cd "$root" && find . -name go.mod -not -path '*/vendor/*' | sort)
