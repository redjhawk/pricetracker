# Technical specification handoff

Status: ready. Date: 2026-10-03. Agent: `/root/leboncoin_session_technical`.

Read repository AGENTS.md, staged workflow, technical-specifier role and all four required project skills. Applied the Go backend/architecture guidance; frontend is unaffected. Read both ready functional subjects, the approved API, current configuration/service/collector/release scripts, and the prior investigation including its native visible-browser prototype.

Deliverables:

- [Capture design](../../specifications/leboncoin-session-capture/technical.md), covering FR-LBC-CAP-001–009.
- [Collection design](../../specifications/leboncoin-session-collection/technical.md), covering FR-LBC-COL-001–010.
- [API preservation assessment](api-step.md).

The implementation stays in a desktop Node helper and narrow Go configuration/session/collector integration. A visible directly spawned browser with private profile and local CDP supports manual verification. No browser is required on the Pi. The shared JSON exports only a verified scoped datadome cookie and expiry/capture metadata. The collector never overwrites the operator import: updates/deletions persist in a private sidecar tied to its SHA-256 fingerprint, with context-cancellable serialization so in-flight responses cannot overwrite newer operator renewal. Only a verified matching final listing permits persistence of replacement cookie values; 403/challenge values do not replace valid state. An explicit correctly scoped deletion is persisted regardless of HTTP status, so revoked cookies do not reappear after restart.

The developer must record focused failing Go/Node tests before implementation, then actual passing checks and production ARMv6 build. Independent review, adjudication, CLI/browser/application QA and coordinator commits remain subsequent gates. The technical stage performed source/document inspection only; it did not run tests, implement code, deploy, or verify remote cookie portability.

No new HTTP contract, dependency, SQLite migration, UI work or unrelated refactoring is proposed. No unresolved product choice blocks handoff. The coordinator retains ownership of the index and later stage records.
