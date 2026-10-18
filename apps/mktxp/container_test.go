package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/mktxp:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "python3", "/usr/local/bin/mktxp", "--help")
}
