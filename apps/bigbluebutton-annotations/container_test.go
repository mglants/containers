package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestComponent(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/bigbluebutton-annotations:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", `set -eu
test "$(id -u)" -ne 0
bbb-runtime annotations --render-only
node --check master.js; gs --version
`)
}
