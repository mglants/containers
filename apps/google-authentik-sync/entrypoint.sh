#!/bin/sh
set -eu

# Preserve the sync CLI while allowing the repository's dgoss keepalive command.
case "${1:-}" in
  ""|-*) exec python /app/sync.py "$@" ;;
  *) exec "$@" ;;
esac
