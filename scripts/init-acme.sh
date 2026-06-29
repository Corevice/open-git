#!/usr/bin/env sh
set -e
if [ -z "${DOMAIN:-}" ]; then
  echo 'ERROR: DOMAIN env var is required' >&2
  exit 1
fi
ACME_FILE="${ACME_FILE:-/letsencrypt/acme.json}"
if [ ! -f "$ACME_FILE" ]; then
  touch "$ACME_FILE"
fi
chmod 600 "$ACME_FILE"
echo "acme.json initialized at $ACME_FILE with mode 600"
