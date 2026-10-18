package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("google-authentik-sync:0.1.5")
	helpers.RequireCommandSucceeds(t, image, nil, "python", "-m", "unittest", "discover", "-s", "/app", "-q")
}
