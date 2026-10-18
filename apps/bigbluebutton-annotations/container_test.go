package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestComponent(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/bigbluebutton-annotations:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", "set -eu\ntest \"$(id -u)\" -ne 0\nrole=annotations\nbbb-runtime \"$role\" --render-only\ncase \"$role\" in\n  frontend)\n    nginx -t\n    test -s /usr/share/bigbluebutton/html5-client/index.html\n    test -s /var/bigbluebutton/playback/presentation/2.3/index.html ;;\n  web)\n    java -version\n    test -s WEB-INF/classes/bigbluebutton.properties\n    libreoffice --version ;;\n  apps)\n    java -version\n    test -x bin/bbb-apps-akka\n    test -s /etc/bigbluebutton/bbb-apps-akka.conf ;;\n  graphql-actions) node --check bbb-graphql-actions.js ;;\n  graphql-middleware) test -x /usr/local/bin/bbb-graphql-middleware ;;\n  notes) node --check bbb-shared-notes-server.js; test -s blocknote_schema.sql ;;\n  annotations) node --check master.js; gs --version ;;\n  signaling)\n    node --check server.js\n    node -e \"require('mediasoup'); require('@livekit/rtc-node'); require('livekit-server-sdk')\" ;;\n  bootstrap)\n    hasura version --skip-update-check\n    psql --version\n    test -s /usr/share/bbb-graphql-server/bbb_schema.sql ;;\n  recorder)\n    test -x /usr/local/bin/bbb-webrtc-recorder\n    test -s /etc/bbb-webrtc-recorder/bbb-webrtc-recorder.yml ;;\n  recordings)\n    bundle check\n    ffmpeg -version\n    bbb-presentation-video --help\n    python3 -c \"import perfect_freehand, bbb_presentation_video\" ;;\n  webhooks) node --check app.js; test -s config/default.yml ;;\n  plugins)\n    nginx -t\n    test -s /usr/share/nginx/html/live-transcription/LiveTranscriptionPlugin.js ;;\n  sip) livekit-sip --help ;;\n  stt)\n    python3 main.py --help\n    python3 -c \"from gladia_stt_agent import GladiaSttAgent; import redis, livekit.agents\" ;;\n  *) exit 2 ;;\nesac")
}
