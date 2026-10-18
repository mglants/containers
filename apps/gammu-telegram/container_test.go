package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/gammu-telegram:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "gammu-smsd", "--version")
}
