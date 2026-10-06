# Distinguish a blocking COMMENT in the pr-review template

Status: todo (non-blocking PR review comment, triaged on 2026-10-06; not scheduled)

## Problem

The Event column cannot tell a blocking fallback COMMENT from a plain COMMENT.

## Suggested work

Use the values `REQUEST_CHANGES / COMMENT (CHANGES REQUESTED) / COMMENT`.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/55#discussion_r4200574919
- Issue: https://github.com/redjhawk/pricetracker/issues/62
