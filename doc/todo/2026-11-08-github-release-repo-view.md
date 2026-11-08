# GitHub release script: simplify repository resolution

Status: todo (non-blocking PR #100 review comment, triaged on 2026-11-08; not scheduled)

## Problem

Line 29 passes the origin URL to `gh repo view`; with no argument gh already resolves the current repository, and URL parsing differs across remote URL forms.

## Suggested work

Use `gh repo view --json nameWithOwner --jq .nameWithOwner` without the URL argument and re-test with SSH and HTTPS remotes.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/100#discussion_r4507310354
- Triage: `doc/changes/2026-11-08-1821-github-release/pr-triage.md`
- Issue: https://github.com/redjhawk/pricetracker/issues/102
