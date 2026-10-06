# Technical specification handoff

Status: ready (all five subjects); API contract API approved by the user 2026-10-03 (proposal revision 1 unchanged; see the change index and api-step.md); implementation authorized; R-1 approved ("Do it first"); expiry clarification accepted\
Date: 2026-10-03\
Role: independent technical specification agent

Read `AGENTS.md`, the workflow (sections 2–3), the technical-specifier role and template, the four project skills, the five ready functional files, both revision 1 technical files, `API_SPECIFICATION.md`, and the current code (`src/App.tsx`, `src/api/items.ts`, `src/components/AddItemModal.tsx`, `internal/httpapi/server.go`, `internal/store/store.go`, `internal/service/service.go`, `internal/leboncoin/{collector,session}.go`, `config/config.go`, `scripts/capture-leboncoin-session.mjs`, tests, `doc/leboncoin-session.md`, `README.md`).

## Deliverables

| Subject | Technical file | Technical IDs | Tiers |
| --- | --- | --- | --- |
| Application header menu | [technical](../../specifications/app-header-menu/technical.md) | TS-APP-MENU-001–004 | Frontend only; backend/API not affected |
| LeBoncoin session settings | [technical](../../specifications/leboncoin-session-settings/technical.md) | TS-LBC-SET-001–007 | Frontend, backend, API (new endpoints) |
| LeBoncoin session collection, revision 2 | [technical](../../specifications/leboncoin-session-collection/technical.md) | new TS-LBC-COL-006–011; superseded 001, 003, 004; 002 retained/amended; 005 retained | Backend; frontend not affected; API: message text only |
| LeBoncoin session capture, revision 2 | [technical](../../specifications/leboncoin-session-capture/technical.md) | new TS-LBC-CAP-006–012; superseded 004, 005 and the shared import format; 001–003 retained/amended | Desktop helper and docs; frontend/backend/API not affected |
| File-based session removal | [technical](../../specifications/leboncoin-session-file-removal/technical.md) | TS-LBC-RM-001–006 | Backend, helper, docs; frontend/API not affected |

API proposal: [api-step.md](api-step.md) and the “LeBoncoin session settings (approved 2026-10-03)” section of [API_SPECIFICATION.md](../../../API_SPECIFICATION.md).

## Design summary

- **Storage:** additive, idempotent migration creating single-row table `leboncoin_session` (`value`, `expires_at`, `revoked_at`, `revision`, `updated_at`, `last_attempt_at`, `last_attempt_outcome`). No import of old files.
- **Conflict detection (FR-LBC-SET-017):** integer `revision` returned by GET and required by PUT; stale → `409 SESSION_CHANGED`, nothing changed. Incremented by operator save/clear, value-changing renewal and revocation.
- **Collector integration:** the service reads the row at the start of each LeBoncoin attempt (no restart needed), passes the value to `Collector.CollectWithSession`, and persists renewal/revocation and the hint record in one transaction conditioned on the start `revision`, so an in-flight attempt can never overwrite a newer save or resurrect a cleared one. The collector stays free of persistence (architecture skill). The file gate is not needed. Renewal is stored only after the verified requested listing; deletion is honoured on any allowed response; expired/revoked values are never sent.
- **Input parsing:** server-side only (`leboncoin.ParseSessionInput`), 8192-byte input limit, 4096-byte value limit, cookie-string extraction rules and messages per FR-LBC-SET-005/006.
- **UI:** Carbon `OverflowMenu` with `UserAvatar` in the header; Carbon `Modal` + `TextArea` with loading, load-error, saving, invalid, save-error, conflict and hint states.
- **Capture helper:** prints one `datadome=<value>` line on stdout (messages on stderr), no file; prerequisite checks; installed-browser discovery; window confirmation and raise; event-based end classification.

## Capture defect analysis

Not reproduced in three diagnostic runs on the reporting host under Node v20.19.2 with `/usr/bin/google-chrome` 149 (details and evidence in the capture technical file, “Reported defect: investigation and root cause”). Node 20 is not the cause. Established defects: D1 the helper reports page close, browser crash and lost CDP connection with one “Browser closed…” message and discards exit code/signal; D2 no window confirmation/raise or guidance on GNOME Wayland, where focus-stealing prevention can leave the new window behind; D3 static Playwright import (raw stack trace when dependencies are missing), no runtime/display checks, default browser is Playwright's Chromium which is not installed on this host. Fix: TS-LBC-CAP-008/009/010. Diagnostic scripts ran in the scratchpad; no cookie value was read or recorded; two LeBoncoin page loads per run (homepage and nonexistent listing 1), three runs; all diagnostic browsers and temporary profiles were removed.

## Refactoring

No prerequisite refactoring. One optional refactor needs a user decision (recommendation: decline or defer):

- **R-1:** move `request`/`ApiError` from `src/api/items.ts` to a shared `src/api/client.ts`. Reason: a second API module (`settings.ts`) uses them. Scope: three files, import changes only. Risk: low; touches item code unrelated to the feature. Without it the feature exports `request` from `items.ts` (one keyword).

Moving session persistence from the collector to service/store is part of replacing the file mechanism, not a separate refactor.

## Decisions for the user

1. **API confirmation (blocking):** the pending settings contract.
2. **R-1** (optional, non-blocking).
3. **Clarification (non-blocking, proposed default):** natural expiry (time passing) does not increment `revision`, so it does not by itself cause a `409` on save; revocation and renewals do. See the settings technical file.
4. **Technical choice for information:** the capture helper will require Node.js ≥ 20.19 (verified on the host's 20.19.2), while `package.json` `engines` stays ≥ 22 for the build toolchain.

## Verification plan (summary)

Go: parser, store migration/upgrade/conditional writes, service interleavings, handler statuses, secret-free logs, `go test ./...`, `go test -race ./...`. Frontend: `npm run build`, new mocked-API Playwright spec `tests/leboncoin-session-settings.spec.ts` (+ `playwright.config.ts` `testMatch`, npm script `test:settings`), existing `npm run test:platform-tabs`. Node: `node --test scripts/capture-leboncoin-session.test.mjs` on Node 20.19 and 22. Release: `scripts/build-release.sh 6`. QA: real app and helper on the GNOME Wayland host, including visible-window observation.

This stage wrote documentation only; no application code was changed and no application tests were run.

## Revision note

2026-10-03, review amendment REV-SET-001: status updated to record the API approval of 2026-10-03, R-1 decision and the expiry clarification. The sections above record the state at handoff time (history).
