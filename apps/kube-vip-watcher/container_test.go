package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("kube-vip-watcher:v0.11")
	helpers.RequireCommandSucceeds(t, image, nil, "python", "-c", "import ast; ast.parse(open(\"/app/kube-vip-watcher.py\").read())")
}
