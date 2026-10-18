package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("rw-temporary-subscription-app:v2.0.0")
	helpers.RequireHTTPEndpoint(t, image, helpers.HTTPTestConfig{Port: "3001", Path: "/api/health"}, nil)
}
