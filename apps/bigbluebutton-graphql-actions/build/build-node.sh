#!/usr/bin/env bash
set -euo pipefail
component="$1"
mkdir -p /out
case "$component" in
  graphql-actions)
    cd /sources/bbb/bbb-graphql-actions
    npm ci --no-audit --no-fund
    npm run build
    npm prune --omit=dev --no-audit --no-fund
    cp -a dist/. /out/
    mv /out/index.js /out/bbb-graphql-actions.js
    cp -a node_modules package.json package-lock.json /out/
    ;;
  *) echo "Unknown component: $component" >&2; exit 2 ;;
esac
