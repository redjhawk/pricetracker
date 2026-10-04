#!/usr/bin/env bash
# Counts the changed lines (added + deleted) a pull request would contain, excluding generated files,
# and fails when they exceed the limit in AGENTS.md (hard ceiling 500; target 450±50 when splitting).
# Usage: scripts/pr-size.sh [base] [head]   (defaults: origin/master, HEAD)
set -euo pipefail

base="${1:-origin/master}"
head="${2:-HEAD}"
limit="${PR_LINE_LIMIT:-500}"

# Generated files do not count towards the limit
generated='^(package-lock\.json|go\.sum|dist/.*|test-results/.*|playwright-report/.*)$'

total=$(git diff --numstat "$base...$head" \
  | awk -v gen="$generated" '$3 !~ gen && $1 != "-" { sum += $1 + $2 } END { print sum + 0 }')

echo "Changed lines from $base to $head (generated files excluded): $total (limit $limit)"
if [ "$total" -gt "$limit" ]; then
  echo "Too large: split this into stacked pull requests of about 450 changed lines each." >&2
  exit 1
fi
