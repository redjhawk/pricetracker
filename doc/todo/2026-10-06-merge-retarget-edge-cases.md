# Workflow: state merge-retarget edge cases (last part, failed merge after retarget)

Status: todo (non-blocking PR review comment, triaged on 2026-10-06; not scheduled)

## Problem

The retarget step does not say to skip it when there is no next open part. It also does not say what state remains when part k's merge fails after part k+1 was retargeted: k+1 then targets the feature branch, and its diff includes part k's commits.

## Suggested work

In pr-review-triage.md, WORKFLOW.md and TS-WORKFLOW-TRIAGE-005, change the step to: "If there is a next open part, first retarget it; if the merge then fails, record that the next part was already retargeted, then stop."

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/71#discussion_r4200844245
- Issue: https://github.com/redjhawk/pricetracker/issues/72
