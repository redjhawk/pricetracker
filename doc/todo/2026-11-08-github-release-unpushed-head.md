# GitHub release script: decide whether unpushed commits may be released

Status: todo (non-blocking PR #100 review comment, triaged on 2026-11-08; not scheduled)

## Problem

`scripts/create-github-release.sh` tags HEAD without checking that it is on origin, so pushing the tag uploads a commit no branch contains. This is specified behavior (FR-GHREL-03 releases the current commit; RV-04 rejected a restriction).

## Suggested work

If the user wants it, add a functional requirement (e.g. HEAD must be contained in an origin branch, or in origin/master) and then a check before tagging. Needs a functional decision from the user first.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/100#discussion_r4507310350
- Triage: `doc/changes/2026-11-08-1821-github-release/pr-triage.md`
- Issue: https://github.com/redjhawk/pricetracker/issues/101
