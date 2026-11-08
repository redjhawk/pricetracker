# GitHub release script: wrap git tag errors

Status: todo (non-blocking PR #100 review comment, triaged on 2026-11-08; not scheduled)

## Problem

Line 53 runs `git tag -a` unwrapped; without a git identity it fails with git's raw message (already accepted as meeting FR-GHREL-07 in RV-05).

## Suggested work

Wrap it as `git tag ... || fail "...configure user.name/user.email..."`.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/100#discussion_r4507310362
- Triage: `doc/changes/2026-11-08-1821-github-release/pr-triage.md`
- Issue: https://github.com/redjhawk/pricetracker/issues/104
