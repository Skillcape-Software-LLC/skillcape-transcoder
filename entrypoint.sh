#!/bin/sh
set -e

# Fix ownership of data directory (needed for mounted volumes)
chown -R appuser:appuser /data /config 2>/dev/null || true

# Run the application as appuser
exec gosu appuser server "$@"
