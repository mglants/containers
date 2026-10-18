#!/usr/bin/env bash
# Same input/output/format/timeout interface as BBB's converter, without Docker.
# Run the container without network access when testing document conversion.
set -euo pipefail
if (( $# < 2 )); then
    echo "Usage: $0 input output [format] [timeout-seconds]" >&2
    exit 2
fi
input="$1"
output="$2"
format="${3:-pdf}"
duration="${4:-60}"
extension="${format%%:*}"
[[ "$extension" =~ ^[a-zA-Z0-9]+$ && "$duration" =~ ^[0-9]+$ ]] || exit 2
(( duration > 0 && duration <= 999 )) || exit 2
temporary="$(mktemp -d /tmp/bbb-office.XXXXXXXX)"
trap 'rm -rf "$temporary"' EXIT
cp -- "$input" "$temporary/file"
options=()
if [[ "$input" == *.html || "$input" == *.htm ]]; then
    options+=(--writer)
fi
timeout --signal=TERM --kill-after=5 "${duration}s" \
    libreoffice "-env:UserInstallation=file://$temporary/profile" --headless \
    "${options[@]}" --convert-to "$format" --outdir "$temporary" "$temporary/file"
test -s "$temporary/file.$extension"
cp -- "$temporary/file.$extension" "$output"
