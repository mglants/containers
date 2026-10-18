package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("cni-plugins:v1.5.1")
	helpers.RequireFileExists(t, image, "/plugins/macvlan")
}
