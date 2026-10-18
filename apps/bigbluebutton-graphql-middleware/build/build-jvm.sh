#!/usr/bin/env bash
set -euo pipefail
cd /source
case "$1" in
  common)
    java -version
    python3 - <<'PY'
from pathlib import Path
import os
revision = 'bbb' + os.environ['VERSION']
replacements = {old + '-SNAPSHOT': old + '-' + revision for old in ('0.0.22', '0.0.5')}
paths = [Path('bbb-common-message/build.sbt'), Path('bbb-common-web/build.sbt'),
         Path('bbb-common-web/project/Dependencies.scala'),
         Path('akka-bbb-apps/project/Dependencies.scala'), Path('bigbluebutton-web/build.gradle')]
for path in paths:
    text = path.read_text()
    changed = text
    for old, new in replacements.items():
        changed = changed.replace(old, new)
    if changed == text:
        raise SystemExit(f'Expected common-library reference missing: {path}')
    path.write_text(changed)
PY
    cd bbb-common-message
    sbt publish publishLocal
    cd ../bbb-common-web
    sbt publish publishLocal
    ;;
  apps)
    cd akka-bbb-apps
    sbt universal:packageBin
    mkdir -p /out
    unzip -q target/universal/bbb-apps-akka-*.zip -d /tmp/bbb-apps
    cp -a /tmp/bbb-apps/*/. /out/
    ;;
  web)
    cd bigbluebutton-web
    ./gradlew --no-daemon resolveDeps assemble
    mkdir -p /out/web /out/prescheck/lib
    unzip -q build/libs/bigbluebutton-0.10.0.war -d /out/web
    cd pres-checker
    ../gradlew --no-daemon resolveDeps jar
    cp -a lib/. /out/prescheck/lib/
    cp build/libs/bbb-pres-check-*.jar /out/prescheck/lib/
    cp run.sh /out/prescheck/prescheck.sh
    ;;
  *) echo "Unknown JVM build: $1" >&2; exit 2 ;;
esac
