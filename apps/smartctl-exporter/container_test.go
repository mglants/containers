package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/smartctl-exporter:rolling")
	helpers.RequireHTTPEndpoint(t, image, helpers.HTTPTestConfig{Port: "9633", Path: "/metrics"}, nil)
}
