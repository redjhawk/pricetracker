# Responsive layout: check the admin table's inner scroller at tablet widths

Status: todo (deferred review finding REV-005, decided on 2026-11-11; not scheduled)

## Problem

The admin page tests at 700 and 1000 px in `tests/responsive-layout.spec.ts` only check that the page does not scroll sideways. They do not check Carbon's inner table scroller (`.cds--data-table-content`), which hid QA-F001 on the tracked-items list.

## Suggested work

Extend the QA-F001 inner-scroller assertion to the admin users table at 700, 800 and 1000 px.

## Source

- Review: `doc/changes/2026-11-11-1142-issue-140-responsive-layout/review.md` (REV-005)
- Decision: `doc/changes/2026-11-11-1142-issue-140-responsive-layout/decisions.md`
- Issue: not created; `gh issue create` was not permitted in the issue #140 run.
