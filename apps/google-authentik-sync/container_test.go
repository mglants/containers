package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/google-authentik-sync:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "python", "-m", "unittest", "discover", "-s", "/app", "-q")
}
