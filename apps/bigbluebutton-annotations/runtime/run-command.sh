#!/bin/sh
# Container process limits replace the outer systemd unit. This timeout alone
# does not reproduce upstream's per-command filesystem/network sandbox.
set -eu
if [ "$#" -lt 2 ]; then
    echo "Usage: $0 timeout-seconds command [arguments...]" >&2
    exit 2
fi
# BBB web passes values such as 60s; the notes service passes plain seconds.
duration="${1%s}"
case "$duration" in
    ''|*[!0-9]*) echo "Invalid timeout: $1" >&2; exit 2 ;;
esac
[ "$duration" -gt 0 ] || exit 2
shift
exec timeout --signal=TERM --kill-after=5 "${duration}s" "$@"
