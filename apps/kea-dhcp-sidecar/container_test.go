package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/kea-dhcp-sidecar:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", "sh -n /app/generate_config.sh; kubectl version --client=true")
}
