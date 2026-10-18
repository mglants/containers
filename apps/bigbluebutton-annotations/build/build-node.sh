#!/usr/bin/env bash
set -euo pipefail
component="$1"
mkdir -p /out
case "$component" in
  annotations)
    cd /sources/bbb/bbb-export-annotations
    npm ci --omit=dev --no-audit --no-fund
    cp -a . /out/
    ;;
  *) echo "Unknown component: $component" >&2; exit 2 ;;
esac
