package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestComponent(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/bigbluebutton-bootstrap:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", `set -eu
test "$(id -u)" -ne 0
bbb-runtime bootstrap --render-only
hasura version --skip-update-check
psql --version
test -s /usr/share/bbb-graphql-server/bbb_schema.sql
`)
}
