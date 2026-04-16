#!/usr/bin/env bash

set -euo pipefail

# Aggregate cloud-nuke scan results and post a Slack summary.
#
# Expects:
#   /tmp/scan/<repo>/run-*.json       — per-repo CircleCI artifacts
#   /tmp/guardrail/<run-id>/*/nuke-*.json — account-wide GHA nuke artifacts
#
# Required environment variables:
#   WEBHOOK_URL  Slack incoming webhook URL
#   REPOS        Space-separated list of repo names
#   RUN_URL      GitHub Actions run URL for the "View Run" link

if [ -z "$WEBHOOK_URL" ]; then
  echo "::error::SLACK_WEBHOOK_URL secret is not set"
  exit 1
fi

NL=$'\n'
REPO_SUMMARY=$(mktemp)
TOTAL_FOUND=0
TOTAL_DELETED=0
TOTAL_WARNED=0
TOTAL_FAILED=0
TOTAL_RUNS=0
CLEAN_REPOS=""
NO_DATA_REPOS=""

# ── Collect per-repo unique resource IDs for cross-repo dedup ──

mkdir -p /tmp/ids
for repo in $REPOS; do
  for json in "/tmp/scan/$repo"/run-*.json; do
    [ -f "$json" ] || continue
    jq -r '.resources[]? | "\(.resource_type):\(.identifier)"' "$json" 2>/dev/null
  done | sort -u > "/tmp/ids/$repo.txt"
done

cat /tmp/ids/*.txt | sort | uniq -d > /tmp/shared_orphans.txt

# ── Aggregate per-repo stats ──

for repo in $REPOS; do
  REPO_RUN_COUNT=$(find "/tmp/scan/$repo" -name "run-*.json" 2>/dev/null | wc -l | tr -d ' ')
  TOTAL_RUNS=$((TOTAL_RUNS + REPO_RUN_COUNT))

  if [ "$REPO_RUN_COUNT" -eq 0 ]; then
    NO_DATA_REPOS="${NO_DATA_REPOS:+$NO_DATA_REPOS, }$repo"
    continue
  fi

  REPO_FOUND=0
  REPO_DELETED=0
  REPO_WARNED=0
  REPO_FAILED=0
  REPO_ACTIVE_RUNS=0

  for json in "/tmp/scan/$repo"/run-*.json; do
    [ -f "$json" ] || continue

    F=$(jq '(.summary.found // .summary.total_resources // 0)' "$json" 2>/dev/null)
    [ "$F" -eq 0 ] && continue
    REPO_ACTIVE_RUNS=$((REPO_ACTIVE_RUNS + 1))

    D=$(jq '(.summary.deleted // 0)' "$json" 2>/dev/null)
    W=$(jq '(.summary.warned // 0)' "$json" 2>/dev/null)
    FL=$(jq '(.summary.failed // 0)' "$json" 2>/dev/null)
    REPO_FOUND=$((REPO_FOUND + F))
    REPO_DELETED=$((REPO_DELETED + D))
    REPO_WARNED=$((REPO_WARNED + W))
    REPO_FAILED=$((REPO_FAILED + FL))
  done

  if [ "$REPO_FOUND" -eq 0 ]; then
    CLEAN_REPOS="${CLEAN_REPOS:+$CLEAN_REPOS, }$repo"
    continue
  fi

  UNIQUE_TO_REPO=$(comm -23 "/tmp/ids/$repo.txt" /tmp/shared_orphans.txt | wc -l | tr -d ' ')
  if [ "$UNIQUE_TO_REPO" -eq 0 ]; then
    CLEAN_REPOS="${CLEAN_REPOS:+$CLEAN_REPOS, }$repo"
    continue
  fi

  TOTAL_FOUND=$((TOTAL_FOUND + REPO_FOUND))
  TOTAL_DELETED=$((TOTAL_DELETED + REPO_DELETED))
  TOTAL_WARNED=$((TOTAL_WARNED + REPO_WARNED))
  TOTAL_FAILED=$((TOTAL_FAILED + REPO_FAILED))

  # Full type breakdown for logs
  FULL_TYPES=$(find "/tmp/scan/$repo" -name "run-*.json" \
    -exec jq -r '.resources[]? | select(.status != null) | "\(.resource_type) (\(.status))"' {} \; 2>/dev/null | \
    sort | uniq -c | sort -rn | sed 's/^ *//' | \
    awk '{printf "%s%s", (NR>1 ? ", " : ""), $0}')

  # Top 3 types for Slack
  TYPE_LINES=$(find "/tmp/scan/$repo" -name "run-*.json" \
    -exec jq -r '.resources[]? | select(.status != null) | .resource_type' {} \; 2>/dev/null | \
    sort | uniq -c | sort -rn)
  TYPE_COUNT=$(echo "$TYPE_LINES" | wc -l | tr -d ' ')
  SHORT_TYPES=$(echo "$TYPE_LINES" | head -3 | sed 's/^ *//' | \
    awk '{printf "%s%s", (NR>1 ? ", " : ""), $0}')
  [ "$TYPE_COUNT" -gt 3 ] && SHORT_TYPES="${SHORT_TYPES}, +$((TYPE_COUNT - 3)) more"

  # tab-separated: found \t active \t total_runs \t repo \t short_types \t full_types
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$REPO_FOUND" "$REPO_ACTIVE_RUNS" "$REPO_RUN_COUNT" "$repo" "$SHORT_TYPES" "$FULL_TYPES" >> "$REPO_SUMMARY"
done

# ── Aggregate account-wide guardrail stats ──

GR_FOUND=0
GR_DELETED=0
GR_WARNED=0
GR_ACTIVE=0

# Count actual nuke runs (top-level directories), not per-region JSON files.
# Each run downloads as /tmp/guardrail/<run-id>/PhxDevOps-<region>/nuke-<region>.json
GR_RUNS=$(find /tmp/guardrail -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l | tr -d ' ')

for run_dir in /tmp/guardrail/*/; do
  [ -d "$run_dir" ] || continue
  RUN_FOUND=0

  for json in "$run_dir"/*/*.json; do
    [ -f "$json" ] || continue

    F=$(jq '(.summary.found // 0)' "$json" 2>/dev/null)
    GR_FOUND=$((GR_FOUND + F))
    RUN_FOUND=$((RUN_FOUND + F))

    GR_DELETED=$((GR_DELETED + $(jq '(.summary.deleted // 0)' "$json" 2>/dev/null)))
    GR_WARNED=$((GR_WARNED + $(jq '(.summary.warned // 0)' "$json" 2>/dev/null)))
  done

  [ "$RUN_FOUND" -gt 0 ] && GR_ACTIVE=$((GR_ACTIVE + 1))
done

GR_FULL_TYPES=""
GR_SHORT_TYPES=""
if [ "$GR_FOUND" -gt 0 ]; then
  GR_TYPE_LINES=$(find /tmp/guardrail -name "*.json" \
    -exec jq -r '.resources[]? | select(.status != null) | .resource_type' {} \; 2>/dev/null | \
    sort | uniq -c | sort -rn)
  GR_TYPE_COUNT=$(echo "$GR_TYPE_LINES" | wc -l | tr -d ' ')

  GR_FULL_TYPES=$(echo "$GR_TYPE_LINES" | sed 's/^ *//' | \
    awk '{printf "%s%s", (NR>1 ? ", " : ""), $0}')

  GR_SHORT_TYPES=$(echo "$GR_TYPE_LINES" | head -5 | sed 's/^ *//' | \
    awk '{printf "%s%s", (NR>1 ? ", " : ""), $0}')
  [ "$GR_TYPE_COUNT" -gt 5 ] && GR_SHORT_TYPES="${GR_SHORT_TYPES}, +$((GR_TYPE_COUNT - 5)) more"
fi

# ══════════════════════════════════════════════════════════════════
# GHA Logs — full details
# ══════════════════════════════════════════════════════════════════

echo "============================================"
echo "  WEEKLY RESIDUAL RESOURCE REPORT (FULL)"
echo "============================================"
echo ""
echo "--- Per-repo cleanup (${TOTAL_RUNS} runs across $(echo "$REPOS" | wc -w | tr -d ' ') repos) ---"
echo "Total: ${TOTAL_FOUND} residuals (deleted ${TOTAL_DELETED}, warned ${TOTAL_WARNED}, failed ${TOTAL_FAILED})"
echo ""

if [ -s "$REPO_SUMMARY" ]; then
  printf '%-40s %8s %10s  %s\n' "REPO" "FOUND" "RUNS" "RESOURCE TYPES"
  printf '%-40s %8s %10s  %s\n' "----" "-----" "----" "--------------"
  sort -t$'\t' -k1 -rn "$REPO_SUMMARY" | while IFS=$'\t' read -r FOUND ACTIVE TOTAL_R REPO SHORT FULL; do
    printf '%-40s %8s %6s/%-3s  %s\n' "$REPO" "$FOUND" "$ACTIVE" "$TOTAL_R" "$FULL"
  done
fi

echo ""
if [ -n "$CLEAN_REPOS" ]; then
  echo "Clean: $CLEAN_REPOS"
fi
if [ -n "$NO_DATA_REPOS" ]; then
  echo "No data: $NO_DATA_REPOS"
fi

echo ""
echo "--- Account-wide guardrail (PhxDevOps, ${GR_RUNS} runs) ---"
if [ "$GR_FOUND" -eq 0 ]; then
  echo "Nothing caught."
else
  echo "Caught ${GR_FOUND} resources in ${GR_ACTIVE}/${GR_RUNS} runs (deleted ${GR_DELETED}, warned ${GR_WARNED})"
  [ -n "$GR_FULL_TYPES" ] && echo "Types: ${GR_FULL_TYPES}"
fi

SHARED_COUNT=$(wc -l < /tmp/shared_orphans.txt | tr -d ' ')
if [ "$SHARED_COUNT" -gt 0 ]; then
  echo ""
  echo "--- Shared orphans ($SHARED_COUNT resources seen across multiple repos) ---"
  cat /tmp/shared_orphans.txt
fi

echo ""
echo "============================================"

# ══════════════════════════════════════════════════════════════════
# Slack — concise summary
# ══════════════════════════════════════════════════════════════════

if [ "$TOTAL_FAILED" -gt 0 ]; then
  COLOR="danger"
elif [ "$TOTAL_FOUND" -gt 0 ]; then
  COLOR="warning"
else
  COLOR="good"
fi

MSG=":mag: *Weekly Residual Resource Report*"

# Per-repo section
MSG="${MSG}${NL}${NL}*Per-repo cleanup* (${TOTAL_RUNS} runs across $(echo "$REPOS" | wc -w | tr -d ' ') repos)"
if [ "$TOTAL_FOUND" -eq 0 ]; then
  MSG="${MSG}${NL}All repos clean."
else
  MSG="${MSG}${NL}Total: ${TOTAL_FOUND} residuals (deleted ${TOTAL_DELETED}, warned ${TOTAL_WARNED}, failed ${TOTAL_FAILED})"
  MSG="${MSG}${NL}"
  sort -t$'\t' -k1 -rn "$REPO_SUMMARY" | while IFS=$'\t' read -r FOUND ACTIVE TOTAL_R REPO SHORT FULL; do
    echo "• *${REPO}*: ${FOUND} in ${ACTIVE}/${TOTAL_R} runs — ${SHORT}"
  done | while IFS= read -r line; do
    MSG="${MSG}${NL}${line}"
  done
fi

if [ -n "$NO_DATA_REPOS" ]; then
  MSG="${MSG}${NL}:question: No data: ${NO_DATA_REPOS}"
fi
if [ -n "$CLEAN_REPOS" ]; then
  MSG="${MSG}${NL}:white_check_mark: Clean: ${CLEAN_REPOS}"
fi

# Guardrail section
MSG="${MSG}${NL}${NL}*Account-wide guardrail* (PhxDevOps, ${GR_RUNS} runs)"
if [ "$GR_FOUND" -eq 0 ]; then
  MSG="${MSG}${NL}Nothing caught."
else
  MSG="${MSG}${NL}Caught ${GR_FOUND} untagged/missed resources in ${GR_ACTIVE}/${GR_RUNS} runs (deleted ${GR_DELETED}, warned ${GR_WARNED})"
  [ -n "$GR_SHORT_TYPES" ] && MSG="${MSG}${NL}Top types: ${GR_SHORT_TYPES}"
fi

MSG="${MSG}${NL}${NL}<${RUN_URL}|View full report>"
rm -f "$REPO_SUMMARY" /tmp/shared_orphans.txt
rm -rf /tmp/ids

PAYLOAD=$(jq -n \
  --arg color "$COLOR" \
  --arg text "$MSG" \
  '{attachments: [{color: $color, blocks: [{type: "section", text: {type: "mrkdwn", text: $text}}]}]}')

curl -sS -X POST "$WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD"
