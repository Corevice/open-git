#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
POLICY_FILE="$SCRIPT_DIR/license-policy.yml"

if [[ ! -f "$POLICY_FILE" ]]; then
  echo "license policy file not found: $POLICY_FILE" >&2
  exit 1
fi

mapfile -t ALLOWED_EXAMPLES < <(
  sed -n 's/^brand_allowed_examples:[[:space:]]*\[\(.*\)\]/\1/p' "$POLICY_FILE" \
    | tr ',' '\n' \
    | sed 's/^[[:space:]]*"//;s/"[[:space:]]*$//;s/^"//;s/"$//'
)

MATCHES=$(
  grep -rn \
    --include="*.ts" \
    --include="*.tsx" \
    --include="*.go" \
    -E '("GitHub"|"Octocat")' \
    --exclude-dir=node_modules \
    --exclude-dir=.git \
    --exclude-dir=vendor \
    --exclude-dir=__tests__ \
    "$REPO_ROOT" 2>/dev/null || true
)

if [[ -z "$MATCHES" ]]; then
  exit 0
fi

VIOLATIONS=()
while IFS= read -r line; do
  [[ -z "$line" ]] && continue

  allowed=false
  for example in "${ALLOWED_EXAMPLES[@]}"; do
    example="${example#"${example%%[![:space:]]*}"}"
    example="${example%"${example##*[![:space:]]}"}"
    [[ -z "$example" ]] && continue
    if [[ "$line" == *"$example"* ]]; then
      allowed=true
      break
    fi
  done

  if [[ "$allowed" == false ]]; then
    VIOLATIONS+=("$line")
  fi
done <<< "$MATCHES"

if [[ ${#VIOLATIONS[@]} -gt 0 ]]; then
  echo "Forbidden brand terms found outside allowed contexts:" >&2
  printf '%s\n' "${VIOLATIONS[@]}" >&2
  exit 1
fi

exit 0
