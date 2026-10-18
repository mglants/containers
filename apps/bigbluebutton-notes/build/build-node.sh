#!/usr/bin/env bash
set -euo pipefail
component="$1"
mkdir -p /out
case "$component" in
  notes)
    cd /sources/bbb/bbb-shared-notes-server
    npm ci --no-audit --no-fund
    npm run build
    npm prune --omit=dev --no-audit --no-fund
    cp -a dist/. /out/
    mv /out/index.js /out/bbb-shared-notes-server.js
    cp -a node_modules package.json package-lock.json assets config blocknote_schema.sql /out/
    ;;
  *) echo "Unknown component: $component" >&2; exit 2 ;;
esac
