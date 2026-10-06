# Write PR review JSON outside the work tree

Status: todo (non-blocking PR review comment, triaged on 2026-10-06; not scheduled)

## Problem

Nothing says where the `--input` JSON file goes, so stray files could be committed.

## Suggested work

Name `$RUNNER_TEMP` (or an ignored path) in the pr-reviewer role and the ai-dev prompt.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200574906
- Issue: https://github.com/redjhawk/pricetracker/issues/60
