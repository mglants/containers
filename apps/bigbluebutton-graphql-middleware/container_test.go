package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestComponent(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/bigbluebutton-graphql-middleware:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/usr/local/bin/bbb-runtime", "graphql-middleware", "--render-only")
	helpers.RequireCommandSucceeds(t, image, &helpers.ContainerConfig{Env: map[string]string{
		"BBB_CONFIG_MODE": "env", "DOMAIN": "meet.example.test", "SHARED_SECRET": "test-secret", "LIVEKIT_KEY": "test-key", "LIVEKIT_SECRET": "test-secret",
	}}, "/usr/local/bin/bbb-runtime", "graphql-middleware", "--render-only")
	helpers.RequireFileExists(t, image, "/usr/local/bin/bbb-graphql-middleware")
}
