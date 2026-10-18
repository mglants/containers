#!/usr/bin/env bash
set -euo pipefail
component="$1"
mkdir -p /out
case "$component" in
  frontend)
    cd /sources/bbb/bigbluebutton-html5
    CI=true npm ci --no-audit --no-fund
    DISABLE_ESLINT_PLUGIN=true npm run build-safari
    npm run build
    python3 - <<'PY'
from pathlib import Path
import re
import os
dist = Path('dist')
hashes = [m.group(1) for p in dist.glob('bundle.*.js') if (m := re.fullmatch(r'bundle\.([a-f0-9]{20})\.js', p.name))]
if len(hashes) != 1:
    raise SystemExit(f'Expected one main client bundle, got {hashes}')
for p in list(dist.glob('*.safari.js')) + list(dist.glob('*.safari.js.map')):
    if hashes[0] not in p.name:
        prefix, suffix = p.name.split('.safari.js', 1)
        p.rename(p.with_name(f'{prefix}.{hashes[0]}.safari.js{suffix}'))
version = os.environ['VERSION']
for p in [dist/'index.html', dist/'stylesheets/fonts.css']:
    p.write_text(p.read_text().replace('?v=VERSION', '?v=' + version))
PY
    cp -a dist /out/client
    cd /sources/bbb/bbb-learning-dashboard
    npm ci --no-audit --no-fund
    npm run build
    cp -a build /out/dashboard
    ;;
  graphql-actions)
    cd /sources/bbb/bbb-graphql-actions
    npm ci --no-audit --no-fund
    npm run build
    npm prune --omit=dev --no-audit --no-fund
    cp -a dist/. /out/
    mv /out/index.js /out/bbb-graphql-actions.js
    cp -a node_modules package.json package-lock.json /out/
    ;;
  notes)
    cd /sources/bbb/bbb-shared-notes-server
    npm ci --no-audit --no-fund
    npm run build
    npm prune --omit=dev --no-audit --no-fund
    cp -a dist/. /out/
    mv /out/index.js /out/bbb-shared-notes-server.js
    cp -a node_modules package.json package-lock.json assets config blocknote_schema.sql /out/
    ;;
  annotations)
    cd /sources/bbb/bbb-export-annotations
    npm ci --omit=dev --no-audit --no-fund
    cp -a . /out/
    ;;
  signaling)
    cd /sources/signaling
    npm ci --omit=dev --no-audit --no-fund
    cp config/default.example.yml config/default.yml
    cp -a . /out/
    ;;
  *) echo "Unknown component: $component" >&2; exit 2 ;;
esac
