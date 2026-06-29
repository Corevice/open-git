#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIOS=(
  rest-api-read.js
  rest-api-write.js
  graphql.js
  auth.js
  git-clone.js
)

FAILED=0

for script in "${SCENARIOS[@]}"; do
  echo "Running smoke test: ${script}"
  if k6 run --vus 2 --duration 30s "${SCRIPT_DIR}/${script}"; then
    echo "PASS: ${script}"
  else
    echo "FAIL: ${script}"
    FAILED=1
  fi
done

if [ "${FAILED}" -ne 0 ]; then
  echo "Smoke test suite: FAIL"
  exit 1
fi

echo "Smoke test suite: PASS"
