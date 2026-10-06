# Correct stale commit-step records and pending links

Status: todo (non-blocking PR review comment, triaged on 2026-10-06; not scheduled)

## Problem

The commit-step.md files in the workflow-pr-reviewer and workflow-pr-triage change folders describe the wrong PRs and bases, and their index.md files link to files that are added later.

## Suggested work

Record the real stack (#55 and #57 on feature/workflow-pr-review), and mark links as pending until the files exist.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200750027 , https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200750039 , https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200750403 , https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200750409
- Issue: https://github.com/redjhawk/pricetracker/issues/64
