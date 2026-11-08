# GitHub release script: comment GIT_TERMINAL_PROMPT

Status: todo (non-blocking PR #100 review comment, triaged on 2026-11-08; not scheduled)

## Problem

Line 6 exports `GIT_TERMINAL_PROMPT=0` without explaining that it makes git fail instead of prompting for credentials.

## Suggested work

Add a one-line comment.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/100#discussion_r4507310369
- Triage: `doc/changes/2026-11-08-1821-github-release/pr-triage.md`
- Issue: https://github.com/redjhawk/pricetracker/issues/105
