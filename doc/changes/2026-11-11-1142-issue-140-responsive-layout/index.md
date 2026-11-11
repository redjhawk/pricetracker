# Change: responsive layout (issue #140)

Started: 2026-11-11 11:42 UTC\
Branch: `ai-dev/issue-140-20261111-1142`\
Stage: 4 — development done; next: review

## Scope

Make every screen and modal (header and menu, login, tracked items list with platform tabs, item details, administrator page, Add item, Delete, Settings, Information, Add user modals, purchase goal and AI review sections) usable at phone widths (360 px reference, 320 px minimum) with no page-level horizontal scrolling and no loss of information or actions; desktop unchanged. Frontend-only expected; no API change expected.

## User decisions

- Issue #140 request (owner). No open functional questions.

## Artifacts

- Functional: [responsive-layout/functional.md](../../specifications/responsive-layout/functional.md) (FR-RESPONSIVE-001 to 013)
- Technical: [responsive-layout/technical.md](../../specifications/responsive-layout/technical.md) (TS-RESPONSIVE-001 to 010)
- API: `API_SPECIFICATION.md` unchanged. This is a frontend-only CSS and markup change: no endpoint, field or status is affected, and the backend is not affected.
- Implementation: [implementation.md](implementation.md)
- Summary: [FUNCTIONAL_SPECIFICATIONS.md](../../FUNCTIONAL_SPECIFICATIONS.md) FR-36 and section 15
