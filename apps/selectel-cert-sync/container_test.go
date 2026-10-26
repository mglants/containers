package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/selectel-cert-sync:rolling")

	t.Run("certificate sync tests", func(t *testing.T) {
		helpers.RequireCommandSucceeds(t, image, nil, "python", "-m", "unittest", "discover", "-s", "/app", "-q")
	})
	t.Run("CLI through container entrypoint", func(t *testing.T) {
		helpers.RequireCommandSucceeds(t, image, nil, "/usr/local/bin/container-entrypoint", "--help")
	})
}
