# Workflow: restore a cheap base check before each part merge

Status: todo (non-blocking PR review comment, triaged on 2026-10-06; not scheduled)

## Problem

The check that each part targets the feature branch before it merges was removed. It is still true for a clean run, but a resumed or partially failed run cannot rely on it.

## Suggested work

Before each part merge, check `gh pr view <n> --json baseRefName` and retarget if the base is wrong, or add one clause saying why the base is guaranteed.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/71#discussion_r4200844251
- Issue: https://github.com/redjhawk/pricetracker/issues/73
