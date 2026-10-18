package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("gammu-telegram:v0.0.3")
	helpers.RequireCommandSucceeds(t, image, nil, "gammu-smsd", "--version")
}
