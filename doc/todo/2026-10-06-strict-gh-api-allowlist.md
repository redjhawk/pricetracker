# ai-dev: tighten the gh api allowlist globs (PATCH base, reviews POST)

Status: todo (non-blocking PR review comment, triaged on 2026-10-06; not scheduled)

## Problem

The `*` globs in the ai-dev allowlist entries `gh api .../pulls/* --method PATCH -f base=*` and `.../reviews --method POST --input *` also accept extra arguments, for example `-f state=closed`, so the run could close or edit any PR.

## Suggested work

Use a wrapper script that validates the PR number and branch name, or a narrower command, for both entries.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/71#discussion_r4200844256
- Issue: https://github.com/redjhawk/pricetracker/issues/74
