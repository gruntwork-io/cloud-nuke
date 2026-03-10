#!/usr/bin/env bash
#
# Validates naming conventions for cloud-nuke resources and config keys.
#
# Resource type names (ResourceTypeName in <provider>/resources/**/*.go):
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

# Discover all provider resource directories (aws/resources, gcp/resources, etc.)
RESOURCE_DIRS=()
for dir in "$ROOT"/*/resources; do
  [[ -d "$dir" ]] && RESOURCE_DIRS+=("$dir")
done

if [[ ${#RESOURCE_DIRS[@]} -eq 0 ]]; then
  echo "  ERROR: No resource directories found"
  exit 1
fi

echo "  Scanning: ${RESOURCE_DIRS[*]}"

# Extract all ResourceTypeName values (skip comments)
NAMES=$(grep -rh 'ResourceTypeName:' "${RESOURCE_DIRS[@]}" --include='*.go' \
  | grep -v '^\s*//' \
  | sed -n 's/.*ResourceTypeName:[[:space:]]*"\([^"]*\)".*/\1/p') || true

if [[ -z "$NAMES" ]]; then
  echo "  ERROR: No ResourceTypeName values found"
  exit 1
fi

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

if [[ -z "$CONFIG_BLOCK" ]]; then
  echo "  ERROR: Could not extract Config struct from config/config.go"
  exit 1
fi

PASCAL_PATTERN='^[A-Z][A-Za-z0-9]*$'
TAG_COUNT=0
while IFS= read -r line; do
  # Extract yaml tag (strip ,inline etc.)
  tag=$(echo "$line" | sed -n 's/.*yaml:"\([^",]*\).*/\1/p')
  [[ -z "$tag" ]] && continue
  TAG_COUNT=$((TAG_COUNT + 1))

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

echo "  Checked $TAG_COUNT YAML tags"

if [[ $ERRORS -gt 0 ]]; then
  echo ""
  echo "FAILED: $ERRORS naming convention violation(s) found."
  exit 1
fi

echo ""
echo "PASSED: All naming conventions are valid."
