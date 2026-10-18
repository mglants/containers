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
bundle exec rake -T
ffmpeg -version
ffmpeg -hide_banner -loglevel error -f lavfi -i color=c=black:s=64x64:r=1 -frames:v 1 -c:v libvpx -an /tmp/mkclean-input.webm
mkclean --quiet /tmp/mkclean-input.webm /tmp/mkclean-output.webm
ffprobe -v error /tmp/mkclean-output.webm
bbb-presentation-video --help
python3 -c "import perfect_freehand, bbb_presentation_video"
`)
}
