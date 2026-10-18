package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestComponent(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/bigbluebutton-plugins:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", `set -eu
test "$(id -u)" -ne 0
bbb-runtime plugins --render-only
nginx -t
test -s /usr/share/nginx/html/live-transcription/LiveTranscriptionPlugin.js
`)
	helpers.RequireHTTPEndpoint(t, image, helpers.HTTPTestConfig{Port: "8080", Path: "/live-transcription/LiveTranscriptionPlugin.js"}, &helpers.ContainerConfig{Env: map[string]string{
		"BBB_CONFIG_MODE": "env", "DOMAIN": "meet.example.test",
	}})
}
