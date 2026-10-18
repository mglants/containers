package main

import (
	helpers "github.com/mglants/containers/tests"
	"testing"
)

func TestImage(t *testing.T) {
	image := helpers.GetTestImage("rpi-dnsmasq:v1.52")
	helpers.RequireFileExists(t, image, "/var/lib/tftpboot/ipxe-arm64.efi")
}
