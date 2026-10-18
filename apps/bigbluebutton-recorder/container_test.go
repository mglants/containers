package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestComponent(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/bigbluebutton-recorder:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", `set -eu
test "$(id -u)" -ne 0
bbb-runtime recorder --render-only
test -x /usr/local/bin/bbb-webrtc-recorder
test -s /etc/bbb-webrtc-recorder/bbb-webrtc-recorder.yml
`)
}
