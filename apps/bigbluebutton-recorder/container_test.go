package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestComponent(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/bigbluebutton-recorder:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/usr/local/bin/bbb-runtime", "recorder", "--render-only")
	helpers.RequireCommandSucceeds(t, image, &helpers.ContainerConfig{Env: map[string]string{
		"BBB_CONFIG_MODE": "env", "DOMAIN": "meet.example.test", "SHARED_SECRET": "test-secret", "LIVEKIT_KEY": "test-key", "LIVEKIT_SECRET": "test-secret",
	}}, "/usr/local/bin/bbb-runtime", "recorder", "--render-only")
	helpers.RequireFileExists(t, image, "/usr/local/bin/bbb-webrtc-recorder")
	helpers.RequireCommandSucceeds(t, image, nil, "/usr/local/bin/bbb-webrtc-recorder", "--version")
	helpers.RequireCommandSucceeds(t, image, &helpers.ContainerConfig{Env: map[string]string{
		"BBB_CONFIG_MODE": "env", "DOMAIN": "meet.example.test", "SHARED_SECRET": "test-secret", "LIVEKIT_KEY": "test-key", "LIVEKIT_SECRET": "test-secret",
	}}, "/usr/local/bin/bbb-runtime", "recorder", "bbb-webrtc-recorder", "--config", "/etc/bbb-webrtc-recorder/bbb-webrtc-recorder.yml", "--dump", "recorder.directory")
}
