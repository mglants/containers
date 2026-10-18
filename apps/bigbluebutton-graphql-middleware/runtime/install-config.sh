#!/bin/sh
# Build-time only: native config paths point into the runtime user's writable /tmp.
set -eu
role="$1"
base=/usr/share/bbb-container/originals
mkdir -p "$base"
case "$role" in
 web) entries='/etc/bigbluebutton/bbb-web.properties web.properties
/etc/bigbluebutton/turn-stun-servers.xml turn.xml' ;;
 apps) entries='/etc/bigbluebutton/bbb-apps-akka.conf apps.conf
/etc/bigbluebutton/bbb-html5.yml client.json' ;;
 notes) entries='/etc/bigbluebutton/bbb-shared-notes-server.yml notes.json' ;;
 annotations) entries='/usr/local/bigbluebutton/bbb-export-annotations/config/settings.json annotations.json' ;;
 graphql-middleware) entries='/etc/bigbluebutton/bbb-graphql-middleware.yml middleware.json' ;;
 signaling) entries='/etc/bigbluebutton/bbb-webrtc-sfu/production.json signaling.json' ;;
 recorder) entries='/etc/bbb-webrtc-recorder/bbb-webrtc-recorder.yml recorder.json' ;;
 recordings) entries='/etc/bigbluebutton/recording/recording.yml recording.json
/etc/bigbluebutton/bbb-web.properties web.properties' ;;
 sip) entries='/etc/bigbluebutton/livekit-sip.yaml sip.json' ;;
 frontend) entries='/etc/nginx/nginx.conf nginx.conf' ;;
 plugins) entries='/usr/share/nginx/html/live-transcription/manifest.json plugin.json' ;;
 *) exit 2 ;;
esac
printf '%s\n' "$entries" | while read -r path name; do
 if [ -f "$path" ]; then cp "$path" "$base/$name"; fi
 mkdir -p "$(dirname "$path")"
 ln -sf "/tmp/bbb-config/$name" "$path"
done
