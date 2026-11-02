package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/remnawave-s3-sync:rolling")

	t.Run("prints its version", func(t *testing.T) {
		helpers.RequireCommandSucceeds(t, image, nil, "/usr/local/bin/remnawave-s3-sync", "-version")
	})
}
