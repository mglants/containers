package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/s4cmd:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/usr/bin/s4cmd", "--help")
}
