package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestComponent(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/bigbluebutton-stt:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", `set -eu
test "$(id -u)" -ne 0
bbb-runtime stt --render-only
python3 main.py --help
python3 -c "from gladia_stt_agent import GladiaSttAgent; import redis, livekit.agents"
`)
}
