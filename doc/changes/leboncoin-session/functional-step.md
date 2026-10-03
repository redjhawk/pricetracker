# Functional specification handoff

Status: ready\
Date: 2026-10-03\
Role: independent functional specification agent

## Requested subjects

1. [LeBoncoin session capture](../../specifications/leboncoin-session-capture/functional.md), FR-LBC-CAP-001 through FR-LBC-CAP-009: visible isolated desktop browser, human verification, verified listing identity, private minimal export, secure documented transfer/install, cancellation/failure, and repeatable renewal.
2. [LeBoncoin session collection](../../specifications/leboncoin-session-collection/functional.md), FR-LBC-COL-001 through FR-LBC-COL-010: optional protected file, existing Go collection paths, response updates and restart persistence, live renewal, preserved observations/error semantics, and secret-free diagnostics.

## Decisions and boundaries

The coordinator supplied the user's explicit reply, “Yes, manual verification is acceptable,” to the proposed computer-browser verification, secure transfer to the Raspberry Pi, and repeated verification when rejected. The coordinator selected a command-line desktop helper with documented secure transfer/install steps within that scope. The specifications are ready for technical design; ready does not claim approval of an unpresented API contract or specific implementation.

The existing API and React interface are preserved. There is no session endpoint, new screen, account login, automated challenge solving, mandatory Raspberry Pi browser, or unrelated refactoring. The listing `https://www.leboncoin.fr/ad/voitures/3245888872` is investigation evidence and a possible QA input, not the only supported target.

The [investigation](../leboncoin-403-investigation/report.md) demonstrated local accepted session reuse, not portability between networks or lasting access. Capture success, installation success, and server collection success are separate observable outcomes. Rejection requires the documented human renewal loop and must preserve existing prices.

## Preparation and verification

Read `AGENTS.md`, the staged workflow, the functional-specifier role, all four required project skills, the functional template, existing functional specifications, and the investigation report. Applied the Go backend guidance and unchanged frontend/API boundaries. Checked that each subject has scope, actor/prerequisite statements, stable requirement IDs, testable acceptance criteria, error/corner cases, and traceability.

No unresolved product question or refactoring decision blocks the technical handoff. Browser integration, file format, configuration, timeout, transfer commands, and persistence details remain technical design work. No application code was changed and no application checks or live requests were performed by this role.
