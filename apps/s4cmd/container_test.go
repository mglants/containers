package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("s4cmd:2.1.0")
	helpers.RequireCommandSucceeds(t, image, nil, "/usr/bin/s4cmd", "--help")
}
