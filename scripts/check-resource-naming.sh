#!/usr/bin/env bash
#
# Validates naming conventions for cloud-nuke resources and config keys.
#
# Resource type names (ResourceTypeName in aws/resources/*.go):
#   - Must be lowercase kebab-case: ^[a-z][a-z0-9]*(-[a-z0-9]+)*$
#   - Must be unique across all resources
#
# Config YAML tags on Config struct fields (in config/config.go):
#   - Must be PascalCase: ^[A-Z][A-Za-z0-9]*$
#   - Must match their Go field name (prevents drift like EC2DHCPOption vs EC2DhcpOption)
#   - Must be unique across all fields
#
set -euo pipefail

ERRORS=0
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# --- Check ResourceTypeName values ---
echo "Checking ResourceTypeName conventions..."

# Extract all ResourceTypeName values (skip comments)
NAMES=$(grep -rh 'ResourceTypeName:' "$ROOT/aws/resources/" --include='*.go' \
  | grep -v '^\s*//' \
  | sed -n 's/.*ResourceTypeName:[[:space:]]*"\([^"]*\)".*/\1/p')

KEBAB_PATTERN='^[a-z][a-z0-9]*(-[a-z0-9]+)*$'
while IFS= read -r name; do
  if ! [[ "$name" =~ $KEBAB_PATTERN ]]; then
    echo "  FAIL: ResourceTypeName \"$name\" is not kebab-case"
    ERRORS=$((ERRORS + 1))
  fi
done <<< "$NAMES"

DUPES=$(echo "$NAMES" | sort | uniq -d)
if [[ -n "$DUPES" ]]; then
  echo "  FAIL: Duplicate ResourceTypeName values: $DUPES"
  ERRORS=$((ERRORS + 1))
fi

echo "  Checked $(echo "$NAMES" | wc -l | tr -d ' ') resource types"

# --- Check Config struct YAML tags ---
echo "Checking Config struct YAML tag conventions..."

# Extract just the Config struct block
CONFIG_BLOCK=$(sed -n '/^type Config struct {$/,/^}$/p' "$ROOT/config/config.go" \
  | tail -n +2 \
  | sed '$d')

PASCAL_PATTERN='^[A-Z][A-Za-z0-9]*$'
while IFS= read -r line; do
  # Extract yaml tag (strip ,inline etc.)
  tag=$(echo "$line" | sed -n 's/.*yaml:"\([^",]*\).*/\1/p')
  [[ -z "$tag" ]] && continue

  # Check PascalCase
  if ! [[ "$tag" =~ $PASCAL_PATTERN ]]; then
    echo "  FAIL: YAML tag \"$tag\" is not PascalCase"
    ERRORS=$((ERRORS + 1))
  fi

  # Check tag matches Go field name
  field=$(echo "$line" | awk '{print $1}')
  if [[ "$tag" != "$field" ]]; then
    echo "  FAIL: YAML tag \"$tag\" does not match field name \"$field\""
    ERRORS=$((ERRORS + 1))
  fi
done <<< "$CONFIG_BLOCK"

YAML_DUPES=$(echo "$CONFIG_BLOCK" \
  | sed -n 's/.*yaml:"\([^",]*\).*/\1/p' \
  | sort | uniq -d)
if [[ -n "$YAML_DUPES" ]]; then
  echo "  FAIL: Duplicate YAML tags: $YAML_DUPES"
  ERRORS=$((ERRORS + 1))
fi

if [[ $ERRORS -gt 0 ]]; then
  echo ""
  echo "FAILED: $ERRORS naming convention violation(s) found."
  exit 1
fi

echo ""
echo "PASSED: All naming conventions are valid."
