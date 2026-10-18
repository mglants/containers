package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("kea-dhcp-sidecar:v0.0.2")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", "sh -n /app/generate_config.sh; kubectl version --client=true")
}
