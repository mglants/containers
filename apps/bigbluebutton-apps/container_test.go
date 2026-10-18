package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestComponent(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/bigbluebutton-apps:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", `set -eu
test "$(id -u)" -ne 0
bbb-runtime apps --render-only
java -version
test -x bin/bbb-apps-akka
test -s /etc/bigbluebutton/bbb-apps-akka.conf
`)
}
