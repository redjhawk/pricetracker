# Drop dead gh pr edit from the ai-dev allowlist

Status: todo (non-blocking PR review comment, triaged on 2026-10-07; not scheduled)

## Problem

`Bash(gh pr edit:*)` is still allowed, but the prompt, WORKFLOW.md and technical.md say it fails (Projects classic GraphQL error) and use the REST PATCH instead.

## Suggested work

Remove the entry from --allowedTools and record the removal in technical.md.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/76#discussion_r4201072237
- Issue: https://github.com/redjhawk/pricetracker/issues/77
