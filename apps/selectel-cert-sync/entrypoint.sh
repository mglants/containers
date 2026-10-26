#!/bin/sh
set -eu

# Support the sync CLI and the build pipeline's container keepalive command.
case "${1:-}" in
  ""|-*) exec python /app/sync.py "$@" ;;
  *) exec "$@" ;;
esac
