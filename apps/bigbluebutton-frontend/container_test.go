package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestComponent(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/bigbluebutton-frontend:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", `set -eu
test "$(id -u)" -ne 0
bbb-runtime frontend --render-only
nginx -t
test -s /usr/share/bigbluebutton/html5-client/index.html
test -s /var/bigbluebutton/playback/presentation/2.3/index.html
`)
}
