#!/bin/sh
# Build-time only: native config paths point into the runtime user's writable /tmp.
set -eu
role="$1"
base=/usr/share/bbb-container/originals
mkdir -p "$base"
case "$role" in
 annotations) entries='/usr/local/bigbluebutton/bbb-export-annotations/config/settings.json annotations.json' ;;
 *) exit 2 ;;
esac
printf '%s\n' "$entries" | while read -r path name; do
 if [ -f "$path" ]; then cp "$path" "$base/$name"; fi
 mkdir -p "$(dirname "$path")"
 ln -sf "/tmp/bbb-config/$name" "$path"
done
