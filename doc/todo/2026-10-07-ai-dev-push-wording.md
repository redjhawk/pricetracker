# Make the ai-dev prompt's push instructions consistent

Status: todo (non-blocking PR review comment, triaged on 2026-10-07; not scheduled)

## Problem

"Publish a branch only with exactly `git push -u origin HEAD`" contradicts the allowed `git push origin HEAD` and `--force-with-lease` pushes.

## Suggested work

Reword to: publish a new branch with `git push -u origin HEAD`; later pushes may use `git push origin HEAD`, or `git push --force-with-lease origin HEAD` after a rebase.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/76#discussion_r4201072245
- Issue: https://github.com/redjhawk/pricetracker/issues/78
