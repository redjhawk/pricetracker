# Define a triage round and one stop rule

Status: todo (non-blocking PR review comment, triaged on 2026-10-06; not scheduled)

## Problem

The 5-round loop guard is worded three ways, and "the same comment" cannot be tracked across reviews because each review has new comment ids.

## Suggested work

Define a round as one fix push plus re-review plus triage, and use "stop after 5 rounds if any blocking comment remains" everywhere.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/57#discussion_r4200750388
- Issue: https://github.com/redjhawk/pricetracker/issues/65
