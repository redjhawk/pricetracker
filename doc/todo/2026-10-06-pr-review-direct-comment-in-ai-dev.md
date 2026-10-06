# Submit COMMENT directly in ai-dev PR reviews

Status: todo (non-blocking PR review comment, triaged on 2026-10-06; not scheduled)

## Problem

In ai-dev runs, REQUEST_CHANGES always fails with 422 (the PR has the same author), and the reviewer still tries it first.

## Suggested work

In ai-dev runs, submit COMMENT with the `CHANGES REQUESTED:` prefix directly. Elsewhere, try REQUEST_CHANGES and fall back on a 422.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200574899
- Issue: https://github.com/redjhawk/pricetracker/issues/59
