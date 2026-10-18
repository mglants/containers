package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("mktxp:v1.2.19")
	helpers.RequireCommandSucceeds(t, image, nil, "python3", "/usr/local/bin/mktxp", "--help")
}
