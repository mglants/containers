package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/kube-vip-watcher:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "python", "-c", "import ast; ast.parse(open(\"/app/kube-vip-watcher.py\").read())")
}
