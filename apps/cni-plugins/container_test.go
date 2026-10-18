package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/cni-plugins:rolling")
	helpers.RequireFileExists(t, image, "/plugins/macvlan")
}
