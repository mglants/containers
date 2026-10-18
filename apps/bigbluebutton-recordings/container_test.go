package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestComponent(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/bigbluebutton-recordings:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", `set -eu
test "$(id -u)" -ne 0
bbb-runtime recordings --render-only
bundle check
ffmpeg -version
bbb-presentation-video --help
python3 -c "import perfect_freehand, bbb_presentation_video"
`)
}
