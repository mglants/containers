package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/kea-dhcp:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", "kea-dhcp4 -V; kea-dhcp6 -V; kea-ctrl-agent -V")
}
