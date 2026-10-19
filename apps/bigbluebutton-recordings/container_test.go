package main

import (
	"testing"

	helpers "github.com/mglants/containers/tests"
)

func TestComponent(t *testing.T) {
	image := helpers.GetTestImage("ghcr.io/mglants/bigbluebutton-recordings:rolling")
	helpers.RequireCommandSucceeds(t, image, nil, "/bin/sh", "-ec", `set -eu
test "$(id -u)" -ne 0
bbb-runtime recordings --render-only
bundle check
bundle exec rake -T
ffmpeg -version
ffmpeg -hide_banner -loglevel error -f lavfi -i color=c=black:s=64x64:r=1 -frames:v 1 -c:v libvpx -an /tmp/mkclean-input.webm
mkclean --quiet /tmp/mkclean-input.webm /tmp/mkclean-output.webm
ffprobe -v error /tmp/mkclean-output.webm
bbb-presentation-video --help
python3 -c "import perfect_freehand, bbb_presentation_video"
`)
	helpers.RequireCommandSucceeds(t, image, &helpers.ContainerConfig{Env: map[string]string{
		"BBB_CONFIG_MODE": "env", "BBB_PUBLIC_URL": "https://bbb.example.test", "SHARED_SECRET": "test-only",
		"RECORDING_FORMATS": "presentation,video,notes",
	}}, "/bin/sh", "-ec", `set -eu
bbb-runtime recordings --render-only
id=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-1000
raw=/var/bigbluebutton/recording/raw/$id
mkdir -p "$raw/notes" /var/bigbluebutton/recording/status/processed /var/bigbluebutton/recording/status/published
printf '%s\n' '%PDF-1.4' 'Synthetic notes publication test' '%%EOF' > "$raw/notes/notes.pdf"
printf '%s\n' '<recording><metadata meetingId="notes-test" meetingName="Synthetic notes check"/><event timestamp="1000" eventname="ParticipantJoinEvent"><userId>test-user</userId></event><event timestamp="2000" eventname="EndMeetingEvent"/></recording>' > "$raw/events.xml"
cd scripts
bundle exec ruby process/notes.rb -m "$id"
bundle exec ruby publish/notes.rb -m "$id-notes"
file=/var/bigbluebutton/published/notes/$id
cmp "$raw/notes/notes.pdf" "$file/notes.pdf"
grep -q "https://bbb.example.test/notes/$id/notes.pdf" "$file/metadata.xml"
`)
}
