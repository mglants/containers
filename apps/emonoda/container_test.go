package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("emonoda:v2.1.37")
	helpers.RequireCommandSucceeds(t, image, nil, "python", "-c", "import emonoda")
}
