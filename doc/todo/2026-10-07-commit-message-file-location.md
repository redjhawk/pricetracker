# Keep ai-dev commit message files outside the work tree

Status: todo (non-blocking PR review comment, triaged on 2026-10-07; not scheduled)

## Problem

`git commit -F <file>` does not say where the file goes, so a file inside the repository could be staged by a later `git add`.

## Suggested work

Say to write the file outside the repository (for example under `$RUNNER_TEMP`), or to prefer `-m`.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/76#discussion_r4201072252
- Issue: https://github.com/redjhawk/pricetracker/issues/79
