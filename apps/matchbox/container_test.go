package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/matchbox:rolling")
	helpers.RequireHTTPEndpoint(t, image, helpers.HTTPTestConfig{Port: "8080", Path: "/"}, &helpers.ContainerConfig{Env: map[string]string{"MATCHBOX_ADDRESS": "0.0.0.0:8080"}})
}
