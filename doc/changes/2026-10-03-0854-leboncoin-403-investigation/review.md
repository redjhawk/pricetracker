# Independent investigation review

Date: 2026-10-03. Reviewer: `/root/leboncoin_investigation_reviewer`. Scope: diagnostic evidence and documentation, not a product implementation. No live requests were made by this reviewer.

Reviewed base: `fe76586952fac12a5aa0c884172a5b8e16a38364`. Reviewed report SHA-256: `b957a3b936c3d2b1928123ac0b72dc0e4ea264fb67aa34fdc5b7e5f30a6544ec`; index SHA-256: `9d8a54e320356e3865edbe2bab0387a1c7aad487ee8b92b488782d76de2eef77`. Temporary Go source SHA-256: `34077baa6349cc90a0449ed5578fa488860594426f65c51cdcc5e5a33f17af22`; its repository-local and `/tmp` copies matched.

Read AGENTS.md, the reviewer role, all four required project skills, workflow, index, report, current collector, relevant configuration/service/API clauses, temporary browser/HTTP/TLS/Go probe sources, and the eight named JSON result artifacts. The working tree contained only the temporary probe and investigation documentation as untracked paths; no tracked production or dependency diff was present.

## Evidence assessment

- The browser summaries support fresh-session 403 responses, explicit distinction between Playwright launch and CDP instrumentation, and successful persisted-profile reload with matching ID, active status, and 2690000 cents after the reported user verification.
- The Go harness directly invokes `leboncoin.NewCollector(...).Collect(...)`. It does not implement a substitute parser. Collector success requires the response ID to match the fixed requested ID and the ad status to be active. The sanitized probe output itself has no separate ID field; ID agreement is established through this inspected success path and the browser JSON.
- The transport clone removes existing Cookie headers and injects only the selected `datadome` cookie. Every transport request must have HTTPS scheme, exact `www.leboncoin.fr` host, and no userinfo. Other destinations are rejected before transport execution. The offline case returns saved HTML and is correctly distinguished from live retrieval.
- The Go result transcription supports baseline 403 followed by cookie149/cookie154 200 and collector success. Its transcription provenance is explicitly disclosed. Independent QA should validate actual output fields rather than treating an exit code as an acquisition assertion.
- Local short-term session reuse is the supported conclusion. The report appropriately leaves server decision rules, lifetime, unattended renewal, other IP/device behavior, remote ARM execution, and scheduled durability unproven. These are limits of this diagnostic scope, not automatic completion blockers or evidence that a production feature exists.
- The unchanged API already classifies DataDome challenges as `request_error`. Separate product specifications or new API approval are unnecessary for the documentation-only investigation. Proposed cookie integration remains explicitly unapproved and unimplemented.

## Findings

### REV-001 — First manual run did extract listing data before the screenshot failure

Provisional severity: low. Requirement: accurate diagnostic reporting; no new product requirement. Location: [report.md](report.md), executed case LBC-10.

The report says the initial screenshot timeout “did not establish successful extraction in that first run.” However, `/tmp/pricefollower-leboncoin-diagnostic/manual-initial.json` records `snapshots[0].nextDataPresent: true` and an `ad` containing ID `3245888872`, status `active`, title, and `price_cents: 2690000`. Its error records a screenshot timeout after that snapshot. The probe pushes the extracted snapshot before attempting a screenshot.

Expected: distinguish successful initial data extraction from the subsequently interrupted screenshot/export/reload sequence. Actual: LBC-10 understates what the saved evidence establishes. Impact: minor factual inconsistency; it does not invalidate the later reload or Go successes. Suggested fix: state that the first run captured correct listing data but the screenshot timeout interrupted the remaining workflow; LBC-11 separately confirms restart/reload and successful cookie export. Independent adjudication is required before disposition.

## Handoff and limitations

No other findings. No production correction or API change is requested. This review inspected saved evidence and source; it did not repeat live access, witness the human interaction, execute the ARM binary, or verify long-term cookie viability. The report's cross-compilation claim remains coordinator execution evidence to be checked by QA. External reference pages were not independently re-fetched; the result does not rely on them as proof of the live acquisition cases.

## REV-001 correction recheck

Independently verified on 2026-10-03 against `manual-initial.json`: corrected LBC-10 now acknowledges the saved correct ID, active status, title and 2690000 cents before the screenshot timeout. It accurately distinguishes the interrupted export/reload workflow from the later LBC-11 restart/reload and cookie export. The recorded snapshot and timeout support the correction. REV-001 is resolved; no new finding arose from this wording change. No additional network requests were made.

Final reviewed report SHA-256: `e5cce9730b934848819b499c4afc85b72e608b8583547027d842a5f9e1a213ed`. Independent QA and the coordinator's commit stage remain separate completion gates.
