#!/usr/bin/env bash

set -euo pipefail

# Download cloud-nuke result artifacts from CircleCI for a single repo.
# Paginates through scheduled pipelines up to 7 days back, following the
# chain: pipeline → workflow → job → artifact.
#
# Required environment variables:
#   CIRCLECI_TOKEN  CircleCI personal API token
#   CIRCLECI_API    CircleCI API base URL
#   GH_ORG          GitHub organization name
#
# Usage: collect-artifacts.sh <repo-name>

repo=$1
OUTPUT_DIR="/tmp/scan/$repo"
mkdir -p "$OUTPUT_DIR"

NEXT=""
IDX=0

# Cap at 10 pages (~200 pipelines). The NEXT token and 7-day cutoff
# control actual termination; this is a safety bound.
for _ in $(seq 0 9); do
  URL="${CIRCLECI_API}/project/gh/${GH_ORG}/$repo/pipeline"
  [ -n "$NEXT" ] && URL="$URL?page-token=$NEXT"
  RESP=$(curl -sf -H "Circle-Token: $CIRCLECI_TOKEN" "$URL" || echo '{}')

  for pid in $(echo "$RESP" | jq -r '.items[]? | select(.trigger.type == "scheduled_pipeline") | .id'); do
    WID=$(curl -sf -H "Circle-Token: $CIRCLECI_TOKEN" \
      "${CIRCLECI_API}/pipeline/$pid/workflow" | \
      jq -r '.items[]? | select(.name == "nuke-leaked-resources") | .id // empty' | head -1)
    [ -z "$WID" ] && continue

    # Include both successful and failed jobs — failed runs still have artifacts.
    JNUM=$(curl -sf -H "Circle-Token: $CIRCLECI_TOKEN" \
      "${CIRCLECI_API}/workflow/$WID/job" | \
      jq -r '.items[]? | select(.name == "cloud_nuke_cleanup" and (.status == "success" or .status == "failed")) | .job_number // empty' | head -1)
    [ -z "$JNUM" ] && continue

    AURL=$(curl -sf -H "Circle-Token: $CIRCLECI_TOKEN" \
      "${CIRCLECI_API}/project/gh/${GH_ORG}/$repo/$JNUM/artifacts" | \
      jq -r '.items[]? | select(.path == "cloud-nuke-results.json") | .url // empty' | head -1)
    [ -z "$AURL" ] && continue

    curl -sf -H "Circle-Token: $CIRCLECI_TOKEN" -L "$AURL" \
      -o "$OUTPUT_DIR/run-$IDX.json" || continue
    IDX=$((IDX + 1))
  done

  NEXT=$(echo "$RESP" | jq -r '.next_page_token // empty')
  [ -z "$NEXT" ] && break

  # Stop paginating once we've gone past 7 days
  OLDEST=$(echo "$RESP" | jq -r '.items[-1].created_at // empty')
  if [ -n "$OLDEST" ]; then
    OLDEST_TS=$(date -d "$OLDEST" +%s 2>/dev/null || echo 0)
    CUTOFF_TS=$(date -d "7 days ago" +%s 2>/dev/null || echo 0)
    [ "$OLDEST_TS" -lt "$CUTOFF_TS" ] 2>/dev/null && break
  fi
done

echo "$repo: $IDX runs collected"
