# Reuse the issue's change folder when ai-dev resumes

Status: todo (non-blocking PR review comment, triaged on 2026-10-06; not scheduled)

## Problem

A resumed ai-dev run can create a second `doc/changes/*-issue-<n>-*` folder, which breaks FR-WORKFLOW-DATE-003.

## Suggested work

In a separate PR, tell the ai-dev prompt: if `doc/changes/*-issue-<n>-*` exists, reuse it and never create a second one.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/54#discussion_r4200575319
- Issue: https://github.com/redjhawk/pricetracker/issues/67
