#!/usr/bin/env bash
set -euo pipefail
component="$1"
mkdir -p /out
case "$component" in
  signaling)
    cd /sources/signaling
    npm ci --omit=dev --no-audit --no-fund
    cp config/default.example.yml config/default.yml
    cp -a . /out/
    ;;
  *) echo "Unknown component: $component" >&2; exit 2 ;;
esac
