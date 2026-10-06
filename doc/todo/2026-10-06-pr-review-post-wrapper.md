# Post PR reviews through a numeric-checked wrapper

Status: todo (non-blocking PR review comment, triaged on 2026-10-06; not scheduled)

## Problem

The ai-dev allowlist glob `gh api repos/redjhawk/pricetracker/pulls/*/reviews --method POST --input *` can match extra arguments, so it is not a strict boundary.

## Suggested work

Add `scripts/post-pr-review.sh <n> <file>` that checks `^[0-9]+$`, and allow only `Bash(scripts/post-pr-review.sh:*)`.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200574885
- Issue: https://github.com/redjhawk/pricetracker/issues/58
