package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("tuned:v2.26.0")
	helpers.RequireCommandSucceeds(t, image, nil, "tuned", "--version")
}
