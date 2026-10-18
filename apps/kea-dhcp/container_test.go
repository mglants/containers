package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("kea-dhcp:2.7.7")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", "kea-dhcp4 -V; kea-dhcp6 -V; kea-ctrl-agent -V")
}
