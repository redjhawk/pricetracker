# Manual LeBoncoin sessions

Stage: complete (commit stage recorded in [commit-step.md](commit-step.md)). Coordinator: `/root`; continued on 2026-10-03 by a follow-up coordinator session.

User decision: on 2026-10-03 the user answered **“Yes, manual verification is acceptable”** to completing the slider in a browser on their computer, securely transferring the verified session to PriceFollower on the Raspberry Pi, and repeating verification when rejected. This authorizes the manual-session integration; it does not assert indefinite validity, remote access, or a new HTTP API contract.

Scope: a desktop capture helper and protected session reuse by the existing Go collector, with documented transfer/renewal. Keep the Pi's pure-Go deployment and existing application interface/API. Follow the [investigation evidence](../2026-10-03-0854-leboncoin-403-investigation/report.md); the prior documentation commit alone did not change collection behavior.

## Stage records

1. [Functional handoff](functional-step.md), [capture requirements](../../specifications/leboncoin-session-capture/functional.md), [collection requirements](../../specifications/leboncoin-session-collection/functional.md).
2. [Technical handoff](technical-step.md), [capture design](../../specifications/leboncoin-session-capture/technical.md), [collection design](../../specifications/leboncoin-session-collection/technical.md).
3. [API assessment](api-step.md).
4. [Implementation](implementation.md).
5. [Independent review](review.md).
6. [Independent decisions](decisions.md).
7. [QA and URL catalog](qa.md).
8. [Commit](commit-step.md).

Functional agent: `/root/leboncoin_session_functional`; both subjects ready. Technical agent: `/root/leboncoin_session_technical`; both designs and API preservation assessment ready. No unrelated refactoring, new dependency, public API or frontend change is proposed.

Implementation is split by disjoint file ownership: `/root/leboncoin_session_backend_developer` owns Go configuration, collector/session handling and Go tests; `/root/leboncoin_session_capture_developer` owns the desktop helper, its Node tests and operator guide. Both must record behavioral RED before full implementation. The coordinator combines their handoffs before independent review.

## Completion handoff (2026-10-03)

The follow-up coordinator verified the inherited state (Go tests, race tests, vet and the 13 helper tests passed), then ran the remaining gates with separate agents: reviewer recheck of REV-LBC-001 (resolved), adjudication, fixture-based interface QA, and live QA. QA found QA-LBC-F01: Chrome 149 ignored the `about:blank#…` startup marker, so the helper could never start. That was adjudicated critical, fixed with a `data:text/plain,…` marker, rechecked by the reviewer and retested. Live capture and live session collection then succeeded on the desktop host (HTTP 200 with the session versus 403 without). Raspberry Pi portability (QA-DEF-001) and capture after a human solves a slider (QA-DEF-002) are recorded as noncritical deferrals in [decisions.md](decisions.md). See [QA](qa.md) and [review](review.md).
