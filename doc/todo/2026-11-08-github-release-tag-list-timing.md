# GitHub release script: read remote tags after the build

Status: todo (non-blocking PR #100 review comment, triaged on 2026-11-08; not scheduled)

## Problem

Line 33 lists remote tags before the long build; a tag created meanwhile is missed. The later `git push` then fails with a clear message and no release is created, so impact is limited.

## Suggested work

Move the `git ls-remote` call after the build (keeping the early reachability check, or doing it once after the build).

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/100#discussion_r4507310358
- Triage: `doc/changes/2026-11-08-1821-github-release/pr-triage.md`
- Issue: https://github.com/redjhawk/pricetracker/issues/103
