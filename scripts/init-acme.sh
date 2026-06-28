#!/usr/bin/env sh
set -e
ACME_FILE="${ACME_FILE:-/letsencrypt/acme.json}"
if [ ! -f "$ACME_FILE" ]; then
  touch "$ACME_FILE"
fi
chmod 600 "$ACME_FILE"
echo "acme.json initialized at $ACME_FILE with mode 600"
