# Independent review: leboncoin-session-settings

Reviewed specification revisions: working-tree versions of the [change index](index.md) (user decisions, API approval and R-1 “Do it first” of 2026-10-03), [app-header-menu](../../specifications/app-header-menu/functional.md) / [technical](../../specifications/app-header-menu/technical.md), [leboncoin-session-settings](../../specifications/leboncoin-session-settings/functional.md) / [technical](../../specifications/leboncoin-session-settings/technical.md), [leboncoin-session-collection rev. 2](../../specifications/leboncoin-session-collection/functional.md) / [technical](../../specifications/leboncoin-session-collection/technical.md), [leboncoin-session-capture rev. 2](../../specifications/leboncoin-session-capture/functional.md) / [technical](../../specifications/leboncoin-session-capture/technical.md), [leboncoin-session-file-removal](../../specifications/leboncoin-session-file-removal/functional.md) / [technical](../../specifications/leboncoin-session-file-removal/technical.md), and [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) section “LeBoncoin session settings (approved 2026-10-03)” (hashes below).\
Reviewed code revision: working tree on base `509460fc85422c5a7b0a7cb35e47444551c95b80` (`git diff` plus untracked files, excluding `.tmp-session-qa/`), snapshot identified by the SHA-256 list at the end.\
Reviewer: independent reviewer agent (separate invocation; did not implement or specify any part of this change), 2026-10-03.\
Adjudicator: to be assigned (a different agent).

Inputs read: `AGENTS.md`, [reviewer role](../../../.agents/roles/reviewer.md), [workflow](../../workflow/WORKFLOW.md) section 5, the four project skills (`carbon-frontend`, `frontend-architecture`, `go-sqlite-backend`, `go-backend-architecture`), the specifications and API section above, and the implementation records [frontend](implementation-frontend.md), [backend](implementation-backend.md), [capture/docs](implementation-capture.md).

## Summary

No critical defect was found. The store's revision-conditioned writes are correct and serialized (single SQLite connection, one transaction each, conditional `UPDATE … WHERE revision = ?`); the migration is additive and idempotent; status/error mapping follows the approved contract except one edge case of the body limit (REV-SET-002); cookie scope, redirect isolation, verified-only renewal, revocation, never-send-expired/revoked and the unchanged sessionless path match TS-LBC-COL-007/009; no session value reaches logs, item responses or messages in the code paths inspected; the frontend implements every modal state of TS-LBC-SET-006; the capture helper prints exactly one stdout line only on success after cleanup; removal of the file mechanism is complete in current code and documentation.

Eight findings: one documentation inconsistency in the canonical contract and specification statuses (REV-SET-001, provisional medium), five low-severity behavior/contract observations or questions (REV-SET-002 to 005, 008), and two process/tracking items (REV-SET-006, 007). All verification commands pass.

## Findings

Provisional severities are the reviewer's; the adjudicator decides criticality and disposition.

### REV-SET-001: Approved API section and specification statuses still say “not approved / not authorized”

- Type: demonstrated documentation defect (canonical contract inconsistency).
- Location: `API_SPECIFICATION.md` line 258 (section status), lines 4–5 (two consecutive blank lines after the status paragraph); `doc/changes/leboncoin-session-settings/index.md` line 31; status lines (line 3) and API sections of `doc/specifications/leboncoin-session-settings/technical.md` (also line 161), `leboncoin-session-collection/technical.md` (also line 121), `app-header-menu/technical.md` (line 49), `leboncoin-session-capture/technical.md` (line 57), `leboncoin-session-file-removal/technical.md`. The settings technical spec also still says `request`/`ApiError` are reused from `src/api/items.ts` and that R-1 is “not included”, although the user chose “Do it first” and `src/api/client.ts` now exists.
- Requirement: workflow step 3 (canonical `API_SPECIFICATION.md` records the approved contract); change index “User decisions at the API gate” (contract approved 2026-10-03, R-1 approved).
- Evidence: the section heading reads “(approved 2026-10-03)” and the file's top status line lists the approval, but line 258 reads “**Status: proposed 2026-10-03, NOT approved.** … Neither tier may be implemented until the user confirms this section.” `index.md` line 31: “No API contract approval has been given yet for this change.” Technical statuses: “API proposal pending user confirmation; implementation not authorized”.
- Impact: the canonical contract contradicts itself; a later reader cannot tell whether the implemented endpoints are approved, and the statuses claim the committed implementation was unauthorized. No runtime effect.
- Suggested resolution: replace line 258 with an approved status (proposal revision 1 approved 2026-10-03 without change), remove the duplicate blank line, update the stale “pending/not authorized” statuses and R-1 wording to reference the recorded decisions (history may be kept as “was pending until …”).
- Provisional severity: medium (non-critical; must be fixed before commit because it is the canonical contract).

### REV-SET-002: A request whose first JSON value fits but whose total body exceeds 32 768 bytes returns 400 instead of 413

- Type: demonstrated contract mismatch (edge case); same pattern as the pre-existing `POST /api/v1/items` handler.
- Location: `internal/httpapi/server.go` lines 176–194 (`handleLeboncoinSession`, PUT: `MaxBytesError` is only checked on the first `Decode`; the trailing-value `Decode` maps every non-EOF error to `INVALID_JSON`).
- Requirement: API section, error table: “413 `REQUEST_TOO_LARGE` — Body over 32 768 bytes”; TS-LBC-SET-004 “Over-size → 413”.
- Evidence (scratchpad copy of the snapshot, temporary test `zz_review_test.go`, not in the repository): `PUT` with body `{"value":"","revision":0}` followed by 40 000 spaces → `400 {"error":{"code":"INVALID_JSON","message":"Request body must contain one JSON value."}}`; same with 40 000 `x` characters → same 400. Expected 413.
- Impact: low; only a malformed/oversized client body is affected, nothing is saved, and the real frontend never sends such a body. Identical behavior already exists for `POST /items` (pre-existing, not introduced here).
- Suggested resolution: in the new handler, also map a `*http.MaxBytesError` from the second `Decode` to 413 (two lines). Changing `POST /items` would be an unrelated fix and needs its own decision.
- Provisional severity: low.

### REV-SET-003: A bare `Cookie:` / `Set-Cookie:` prefix is reported as invalid characters

- Type: question / message quality (behavior follows the literal rule order of TS-LBC-SET-002).
- Location: `internal/leboncoin/session.go` lines 46–67 (`ParseSessionInput`).
- Requirement: FR-LBC-SET-006 (“a message stating the problem”); TS-LBC-SET-002 rules 3, 4 and 6 do not define a message for an empty value after prefix removal.
- Evidence: scratchpad test: `PUT {"value":"Cookie:","revision":0}` → `400 INVALID_SESSION` “The datadome value contains spaces, quotes or other characters that are not allowed in a cookie value.” The input contains no such characters; it is empty after the prefix. Nothing is saved (correct).
- Impact: the operator gets a misleading message in an unlikely case (pasting only a header name).
- Suggested resolution: either accept as specified, or return “No datadome cookie was found in the pasted text.” when the text is empty after prefix removal (specification clarification, no API change).
- Provisional severity: low (question for the adjudicator).

### REV-SET-004: Any unrecognized `Set-Cookie` attribute silently disables renewal and revocation detection

- Type: question with demonstrated behavior (retained revision 1 rule not restated in revision 2).
- Location: `internal/leboncoin/session.go` line 141 (`len(raw.Unparsed) > 0` skips the cookie).
- Requirement: FR-LBC-COL-014 / user decision 3 (store renewed cookies); FR-LBC-COL-015 (honor explicit deletion). TS-LBC-COL-009 lists the retained rules (“Max-Age precedence over Expires, overflow rejection, value validation …, ambiguity”) and the domain/path scope rule, but not rejection of unknown attributes; revision 1 said “Reject overflow/invalid metadata safely”.
- Evidence: scratchpad test on a verified 200 listing: `datadome=synthetic-new; Max-Age=31536000; Domain=.leboncoin.fr; Path=/; Secure; SameSite=Lax` → `Renewed:true`; the same header plus `; Priority=High` → `Renewed:false`, no diagnostic. A deletion with an extra attribute is likewise ignored.
- Impact: if LeBoncoin's real `datadome` `Set-Cookie` carries any attribute Go does not parse (e.g. `Priority`), automatic renewal (an explicit user requirement) and revocation detection never happen, silently. The real header attribute set was not recorded in this change's evidence, so the practical effect is unknown. Fail-safe from a security standpoint (nothing wrong is stored).
- Suggested resolution: adjudicator to decide whether unknown attributes should be ignored (domain/path/expiry rules already constrain scope) or the fail-safe kept and documented in TS-LBC-COL-009; QA could record the sanitized attribute names of a real LeBoncoin `datadome` `Set-Cookie` (names only, no value).
- Provisional severity: low (question; becomes higher if real headers carry such an attribute).

### REV-SET-005: Canonical-URL failure records a “failed” session attempt although nothing was sent

- Type: minor contract mismatch in an unreachable-in-practice path (developer-flagged, backend limitation 3).
- Location: `internal/leboncoin/collector.go` line 115 returns `attempt.outcome()` (`Attempt: "failed"`); `internal/service/service.go` `collectLeboncoin` then calls `FinishLeboncoinSessionAttempt`, setting `last_attempt_*`.
- Requirement: API field `lastAttempt`: “Latest LeBoncoin check that **sent** this session”; FR-LBC-COL-017.
- Evidence: by inspection; stored LeBoncoin items are validated with the same `ParseURL` at add time, so the branch requires a database row with an invalid URL.
- Impact: none in normal operation; would show a misleading failure hint for a corrupt item.
- Suggested resolution: return a zero `SessionOutcome` (and let the service skip the finish write when `Attempt == ""`), or accept as is with the documented limitation.
- Provisional severity: low.

### REV-SET-006: Deprecation record still lacks the removal commit message(s)

- Type: tracking item (incomplete requirement until the commit stage).
- Location: `doc/deprecated/leboncoin-session-file.md` line 59 (“Removal commits …: to be recorded by message at commit time.”).
- Requirement: FR-LBC-RM-005 (“… and the commit(s) of this removal by message”).
- Evidence: placeholder text in the record.
- Impact: FR-LBC-RM-005 is not met until the coordinator writes the actual message(s). Expected by the developer's handoff.
- Suggested resolution: the coordinator fills in the removal commit message(s) in the commit stage (a message chosen before committing can be cited exactly) and records it in `commit-step.md`.
- Provisional severity: low (blocks completion only if forgotten).

### REV-SET-007: Leftover untracked artifacts outside the change must stay out of the commits

- Type: observation, not attributable to the implementation diff.
- Location: `.tmp-session-qa/main.go` (references removed `NewCollectorWithSession` and `LeboncoinSessionFile`; outside `./...` because of the leading dot); empty untracked directory `session/` (mode 0700, created 2026-10-03 15:33) at the repository root, not shown by `git status` because it is empty, plausibly from the coordinator's reproduction of the old “missing output directory” behavior.
- Requirement: AGENTS.md (explicit staging; preserve unrelated work); FR-LBC-RM-004 (no live references to the removed mechanism).
- Evidence: `ls -la session` (empty); backend record limitation note on `.tmp-session-qa/`.
- Impact: none if not staged; `.tmp-session-qa/main.go` would not compile if ever added to the build.
- Suggested resolution: do not stage either; the user/coordinator may delete them after QA (they are not this reviewer's to remove).
- Provisional severity: informational.

### REV-SET-008: Capture helper cannot report the runtime requirement on Node.js older than 15

- Type: minor specification mismatch (demonstrable by syntax inspection; no old Node binary was executed).
- Location: `scripts/capture-leboncoin-session.mjs` line 2 comment and uses of `??=` (lines in `openBrowser`/`waitForSession`, ES2021, Node ≥ 15) plus top-level `await` (last line, Node ≥ 14.8).
- Requirement: TS-LBC-CAP-008 item 2 (“The check uses only syntax that older Node versions parse”); FR-LBC-CAP-018.
- Evidence: Node.js 14 fails to parse `??=` with a `SyntaxError` before `main` runs, so the “Node.js 20.19 or newer is required” message is never printed; Node 16 and 18 parse the file and reach `checkRuntime`.
- Impact: only operators on Node.js ≤ 14 (end of life since 2023) see a raw syntax error instead of the explanation.
- Suggested resolution: accept (document “Node 16+ parses the helper”), or replace the four `??=` uses with explicit assignments.
- Provisional severity: low.

## Evaluation of deviations and notes flagged by the developers

Frontend ([implementation-frontend.md](implementation-frontend.md)):

| Flagged item | Evaluation |
| --- | --- |
| R-1 performed first as a verbatim move of `request`/`ApiError` to `src/api/client.ts` | Accepted. Approved by the user (“Do it first”); diff shows verbatim move, `items.ts` only imports `request`; build and the 8 existing Playwright tests pass. Specification text not updated (REV-SET-001). |
| `innerRef` (deprecated by Carbon) instead of `ref` on `OverflowMenu` | Accepted as a preference. Focus-return tests prove it targets the trigger; avoids a type cast against the installed 1.117.0 wrapper type. Revisit on a Carbon upgrade. No finding. |
| Date formatter duplicated from `ItemDetail.tsx` | Accepted; extraction would be unapproved refactoring. Same options as the specified format. |
| Mocked API only; server parsing not tested in the frontend | Accepted; server is authoritative by specification; backend tests cover the parser. Real integration is QA scope. |
| Screen-reader announcement not tested; Carbon MCP unavailable | Accepted as limitations; Carbon `InlineNotification`/`TextArea invalidText`/`Modal` patterns are used as specified. QA should do a keyboard pass. |
| `src/index.css` unchanged | Verified: `modal-notification` already exists (`src/index.css` line 363); `app-menu` class has no rule, harmless. |

Backend ([implementation-backend.md](implementation-backend.md)):

| Flagged item | Evaluation |
| --- | --- |
| Store-level `LeboncoinSessionOutcome` struct converted from `leboncoin.SessionOutcome` | Accepted; explicitly allowed by TS-LBC-COL-008; keeps the store free of the adapter. |
| Challenge marker only read on 2xx HTML bodies; non-403 non-2xx challenge → `failed` | Accepted; matches TS-LBC-COL-009 (“final status 403, or its body contains `captcha-delivery.com`”); only bodies already read are inspected. |
| `Set-Cookie` without explicit `Path=/` ignored | Accepted; TS-LBC-COL-009 requires path `/` and ignoring other paths; RFC default path for listing URLs is never `/`. |
| Canonical-URL failure reports `failed` without a request | See REV-SET-005 (low). |
| Shutdown cancels the finish write and logs the fixed diagnostic | Accepted; row stays unchanged, consistent with FR-LBC-COL-016. |
| Tests replace `http.DefaultTransport` / `log` output without `t.Parallel` | Accepted; race suite passes; no parallel tests in those packages. |
| `.tmp-session-qa/` left untouched | See REV-SET-007. |
| `npm run build`, Playwright, `build-release.sh 6` not run by the backend agent | Build and Playwright run by this review (below); `scripts/build-release.sh 6` not run here (QA/commit stage). Backend agent's plain ARMv6 cross-compile evidence noted, not re-run. |
| Unparsed-attribute rejection listed in the record's mapping | Not flagged as a deviation by the developer but not restated by the revision 2 specification; see REV-SET-004. |

Capture/docs ([implementation-capture.md](implementation-capture.md)):

| Flagged item | Evaluation |
| --- | --- |
| 1. One `doc/README.md` line added outside the coordinator's file list | Accepted; required by TS-LBC-RM-004 / FR-LBC-RM-005; only that line changed. |
| 2. Page `close` followed by non-zero exit/signal → unexpected-exit message | Accepted; the TS-LBC-CAP-010 table is silent on this combination, and a crash is the more accurate cause (a deliberate window close exits 0). |
| 3. Extra `no-cookie` timeout reason; 5xx → `navigation-error`, other unverified → `not-listing` | Accepted; consistent with the “last observation” rule and FR-LBC-CAP-017; the guide's timeout row does not list the `no-cookie` reason text (cosmetic). |
| 4. CDP never connects within 20 s → “The browser started but could not be controlled within 20 seconds.” | Accepted; startup-failure category of FR-LBC-CAP-017; documented in the guide. |
| 5. Runtime check before argument parsing; `--help` prints the first line on stderr and usage on stdout | Accepted; TS-LBC-CAP-008 item 1 requires the first message before argument validation. |
| 6. Windows refusal moved into `checkDisplay` | Accepted; preserves revision 1 behavior and the specification's “Windows remains unsupported”. |
| 7. Guide omits the literal obsolete flag | Accepted; keeps the FR-LBC-RM-004 search clean, the message is still recognizable. |
| Limitations: window visibility not human-observed; manual close/quit not run on real Chrome; runtime check only injected; challenge path not exercised | Accepted as QA-stage items; FR-LBC-CAP-014 visible-window confirmation remains open for QA. See REV-SET-008 for the parse limit of the runtime check. |

## Other checks performed (no finding)

- Store: `SetMaxOpenConns(1)` serializes the read-check-write transactions, so two concurrent finishes or a save racing a finish cannot both pass the revision check; `UPDATE … WHERE revision = ?` adds a second guard. Clear of an already-empty session returns the row unchanged without a revision change; stale clear returns 409 (FR-LBC-SET-017 “including an empty entry”). Expiry-only renewal keeps the revision (user clarification “Yes, as proposed”). Revocation keeps the value and sets `revoked_at`, `updated_at`, `revision + 1`. CHECK constraints match the specification. Migration is `CREATE TABLE IF NOT EXISTS` + `INSERT OR IGNORE`, inside the existing `createSchema`; no existing table touched.
- HTTP: exact path routed before item routes; GET/PUT/405 with `Allow: GET, PUT` (HEAD also 405); `MaxBytesReader` 32 768; missing/null `value` or `revision`, type mismatch (`"revision":1.0` → 400 `INVALID_REQUEST`), negative revision → `INVALID_REQUEST`; unknown fields are ignored (consistent with `POST /items`); `Cache-Control: no-store` inherited; storage errors go through `serverError`, whose logged SQLite errors do not contain bound values.
- Collector: sessionless path is the previous code with `attempt == nil` (same headers, logs, redirect policy); session path uses a copied client with the per-attempt jar, validates every redirect before the jar adds the cookie (Go adds jar cookies after `CheckRedirect`), strips forwarded `Cookie` headers, bounds four requests, never logs URLs, cookies or raw errors; expired candidate never sent; renewal only when `verified`; deletion on any status unless a later verified replacement.
- Service: per-attempt read before the 31-second context; `none` → sessionless with no write; `expired`/`revoked` → `request_error` without request; read error → `request_error` without request; finish before `RecordCollection`; fixed log texts; Amazon path unchanged.
- Frontend: GET on every open with an `active` guard; load error hides the field and disables Save; Save sends text as typed with the loaded revision; 400 `INVALID_SESSION` → field `invalidText`, cleared on edit; 409 → non-dismissible warning, Save disabled until reopen; other errors → error notification and retry; ref guard prevents double PUT; close ignored while saving; state reset on close; no browser storage; hint order revoked → expired → rejected → failed; header control after **Add item** in `HeaderGlobalBar`.
- Capture helper: stdout written only by `print(line)` after `capture()` has closed the browser and removed the profile, and by `--help`; every failure path logs one fixed message on stderr; cancellation 130/143; cleanup idempotent.
- Removal: `git grep` for `LEBONCOIN_SESSION_FILE|LeboncoinSessionFile|NewCollectorWithSession|session\.json|state\.json|sidecar` outside `doc/deprecated`, `doc/changes`, `doc/specifications` returns nothing; deleted `config/config_test.go` only tested the removed variable; `doc/changes/leboncoin-session/` and `doc/changes/leboncoin-403-investigation/` unchanged; deprecation record anchors (`#requirements-revision-1/2`, `#revision-1-technical-design-history`) exist.
- No unrelated or unapproved change found in the diff apart from the items above.

## Commands run and results

Go 1.25.0 linux/amd64; `GOCACHE=/tmp/claude-1000/-home-redjhawk-src-pricefollower/db657c8a-daa0-468c-9f0c-59ac91b7b822/scratchpad/gocache`.

| Command | Result |
| --- | --- |
| `go test -race ./...` | `ok` httpapi, leboncoin, service, store (cached) |
| `go test -race -count=1 ./...` | `ok` httpapi 1.149 s, leboncoin 1.050 s, service 1.490 s, store 1.213 s |
| `go vet ./...` | no output, exit 0 |
| `gofmt -l cmd config internal` | no output |
| `/tmp/pricefollower-node22/node_modules/node/bin/node --test scripts/capture-leboncoin-session.test.mjs` (Node v22.23.3) | 22 tests, 22 pass, 0 fail |
| `git diff --check` | no output, exit 0; untracked files also checked for trailing whitespace with `grep`: none |
| `npm run build` (Node v22.23.3, in a scratchpad copy of the snapshot with `node_modules` symlinked, to avoid writing `dist/` in the repository) | passed (`tsc -b && vite build`) |
| `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/google-chrome npx playwright test` (same scratchpad copy) | 28 passed (20 settings, 8 platform-tabs), mocked API |
| Scratchpad-only reproduction tests `internal/httpapi/zz_review_test.go`, `internal/leboncoin/zz_review_test.go` (synthetic values, injected transport, no network) | evidence for REV-SET-002, 003, 004; HEAD → 405 with `Allow`; `revision: 1.0` → 400 `INVALID_REQUEST`; unknown field ignored |
| `git grep` removal search (see above) | no match |

Not run: `scripts/build-release.sh 6`, any live LeBoncoin request, real-browser capture.

## Reviewed snapshot (SHA-256)

Base commit `509460fc85422c5a7b0a7cb35e47444551c95b80`. Deleted: `config/config_test.go`. This file (`review.md`) is excluded.

```text
08524d6a4fa8e9aa6f527ea2b0f38e04a41f0c80d84d62d5a19102ab7fc41fe5  API_SPECIFICATION.md
952ff28acb6d0cd495894f9822a37f0f6d9e68da49fe2ab20448df05c2af0e8f  config/config.go
9c6ed312de7459c192749fda74b014adf1e8834db401754bc9c53f49d7e78880  doc/changes/leboncoin-session-settings/api-step.md
06e4c062c04466b8890c2f51db66508e19949c7484abcf28072afceeb0236ee7  doc/changes/leboncoin-session-settings/functional-step.md
08c13608625a9b00d986b1c9870a34fb155dbd938fc068c5fdae9e62f37e3176  doc/changes/leboncoin-session-settings/implementation-backend.md
60be0058ae9cfee1155b26533165388543384401768179b186380de8ca33534f  doc/changes/leboncoin-session-settings/implementation-capture.md
0f7935acce420210cc442507b5dd7d112d8901494fc1ced1a480db012cf00429  doc/changes/leboncoin-session-settings/implementation-frontend.md
f097d258c2b9b6f6cdf032e506712c26dd69793621ad93829419ed7bce09983d  doc/changes/leboncoin-session-settings/index.md
28a580806f2e8952397b05ae53c246294b8b8aefda591cfac6973b3f99d543cb  doc/changes/leboncoin-session-settings/technical-step.md
3949f19e0909f7a228ccbd57c55fe22869615a667341cdc03b8add525dfdcc41  doc/deprecated/leboncoin-session-file.md
f3196fd935f903e67532ba6efd3c3e7d298219aa21d33693d38715df51ed3f76  doc/deprecated/README.md
ed62255fa695224baafa1f5d9a8c601eb27ffe62ed15a317108246123814f146  doc/leboncoin-session.md
2c4e3b1d669a9d5dbf243a140719d66f742e702ac3d80ddd5ef64d0cac7652f5  doc/README.md
6e281f763802755f8d859abe8d9b7b382f653a64688bc77d40088616af7fc173  doc/specifications/app-header-menu/functional.md
7d0a4c3fa8e89c96026f9d9a0370ef0375d1417c7c9be2e1dd5d88a0557fb33c  doc/specifications/app-header-menu/technical.md
c4e34d5561c8a844f2f1151cf5d50eef3cda10942603533f3b3db4674b4e060e  doc/specifications/leboncoin-session-capture/functional.md
6a3d0b1b9e32debc8eb6fb382afb0eb25edc09ce9cd554ab593128416f367970  doc/specifications/leboncoin-session-capture/technical.md
3d8872904d5228a72d1b1128e04988ec6d21d79673dbb48f52b6857b79a32c61  doc/specifications/leboncoin-session-collection/functional.md
a8eda95601c83d0569d626654230685f18762f45d0efd1b54c34f4a338f20aa4  doc/specifications/leboncoin-session-collection/technical.md
91a2bc5ff3653765b2cc0899cbdbab58bde5f9f19eaebfad853f94359105807e  doc/specifications/leboncoin-session-file-removal/functional.md
08ad7969112465b488c5b6f165ba6ce53a0b83d24df7d21d73eb449e5cedbb10  doc/specifications/leboncoin-session-file-removal/technical.md
12a34832e2d87aacafb0a451aefb9fb747ba68ade8b1366eab567e5282ea9fa7  doc/specifications/leboncoin-session-settings/functional.md
2d3962aff9693c22d8bc83f997de44a0294486119483b3fd7dff33dd57f23f85  doc/specifications/leboncoin-session-settings/technical.md
aa80aaabece87b4a35af9e1addad12c8be82401d20802f55d894cb6513d5d93e  internal/httpapi/server.go
c9aef9d688f1d3ff3be407bac4d571376f0b9d127c8c60433f6c7fbf8f16b031  internal/httpapi/settings_test.go
7eebc19191b53755956296b24593ffe8ae434f4bcce171b6aaae7d363288e7df  internal/leboncoin/collector.go
a22996a9d20e53f8aaf1ab653b5a257a78f1f54b119e01fe2bc4b3f37dea5d46  internal/leboncoin/session.go
68fb9be6b59edd2cbb744f3cb805f8a43c4e37cca2f092d24f22cb64ac1098a1  internal/leboncoin/session_test.go
727bd66ed35af5a2cf94be38e0b968d18e79d8d17b3fb5e96ffc7bc0658997b0  internal/model/model.go
8bedda6f6b5564813ee24997264d13bef65b8e35f4102d27ba7594c1b00dcc11  internal/service/service.go
c1e3251edf0e6be804ce6f7251e226a84facc30931e624d504c913751d6cce6b  internal/service/session_test.go
87aafdb55b75c516f6f87056a361f95eaced8636e3cf921efa3882b2653bfe76  internal/store/session_test.go
9c747de927cc737146d79014a210ed0b492e6f85c629698506097ad5e30597ca  internal/store/store.go
5bc46ff46c69451f98b79f061483fcb62cacefb4da6fe5492b55f86431ce25d3  package.json
b6ee62244ee1fea0c399c8c03245909631b7d91bd2254c5f492ec263922ce322  playwright.config.ts
b1308b456ac44ccf4e71f1253cc0f462cdfc2afe873f12296c6cafa109badc67  README.md
c8ce6e1774924d25fc70e56ce3772d8c37e93a8e2e4935262c6013c40471a578  scripts/capture-leboncoin-session.mjs
4d9e70d4fb855d5722ae614022ef42b974565d31bda615e3de8949c53da4f7bc  scripts/capture-leboncoin-session.test.mjs
cbcc9a0c28c300166c77f926a6acb55dc04375ace8a472deb736476b45a16fc7  src/api/client.ts
995c30efc6dbd03e717f26797f85adc7930d27c1ac6eec97cb4b7f380ee8c95e  src/api/items.ts
e6e0408d13a065a895b736d6e8d239172d314892956c0bf88b28447d0502737f  src/api/settings.ts
61e316575873d86a6e7599304930094f5c92ed73c15a6e542e1d2c04cf983f14  src/App.tsx
1e8c7e1515fad998e8f13efa588341b8baee83d50f9157ed5e8e42f0cb3d03c3  src/components/AppMenu.tsx
1060235a0350a5bbcf32962163f0c2d0fa4f53053cd0a08501a391af2570140d  src/components/SettingsModal.tsx
6dc4eef99451fdf8a4059fc772b2eea074a3d5b66e88f7eece989729cf7f1e44  tests/leboncoin-session-settings.spec.ts
```

## Handoff

All findings (REV-SET-001 to 008), including the questions REV-SET-003 and REV-SET-004, go to the independent adjudicator. No finding is proposed as critical. Any fix must be re-reviewed against an updated snapshot.

## Recheck (2026-10-03)

Scope: the fixes listed by the coordinator. I compared the current working tree with the snapshot above, using the hash list and file diffs against my scratchpad copy of that snapshot.

Changed files and why they changed. No other file differs from the snapshot.

- `internal/httpapi/server.go`: REV-SET-002.
- `internal/leboncoin/session.go`: REV-SET-003 and REV-SET-004.
- `internal/leboncoin/collector.go` and `internal/service/service.go`: REV-SET-005.
- `scripts/capture-leboncoin-session.mjs`: REV-SET-008.
- Matching test files.
- `API_SPECIFICATION.md` and `index.md`: REV-SET-001.
- The settings, collection, capture, header-menu and removal technical specifications, plus `technical-step.md`: status and amendment updates.
- `implementation-backend.md` and `implementation-capture.md`: the developers' "Review fixes" sections.
- New file `decisions.md`.

No frontend file changed, so I did not rerun the build or Playwright. The deleted `config/config_test.go` is unchanged. `.tmp-session-qa/` and `session/` are gone.

| Finding | Recheck outcome |
| --- | --- |
| REV-SET-001 | Fixed. The API section status now reads "approved by the user on 2026-10-03" and links to its evidence. The duplicate blank line is removed. `index.md` line 31 is corrected. A grep for "not authorized", "pending user confirmation" and "NOT approved" in the technical specs, the API spec and the index finds nothing. |
| REV-SET-002 | Fixed for the reported case: valid JSON followed by more than 32 768 bytes of trailing whitespace now returns 413 `REQUEST_TOO_LARGE`. Residual (not a new defect): if the trailing content is invalid JSON, for example `x`, the decoder reports a syntax error before it reaches the size limit, so the response is still 400 `INVALID_JSON`. The body is invalid JSON either way and nothing is saved. Severity: informational. |
| REV-SET-003 | Fixed. A bare `Cookie:` now returns "No datadome cookie was found in the pasted text." |
| REV-SET-004 | Fixed. Unknown attribute names are now ignored: `…; Priority=High` produces `Renewed:true`. A known attribute that cannot be parsed still causes the cookie to be skipped. The domain and path scope checks are unchanged: a cookie without `Path` is still ignored, and `Max-Age=0` still counts as a revocation. |
| REV-SET-005 | Fixed. A failed URL check returns an empty outcome, and the service skips the finish write when `Attempt == ""`. |
| REV-SET-006 | Deferred to the commit stage, as decided. It remains open until `commit-step.md` records the commit message(s). |
| REV-SET-007 | Resolved. Both leftovers were deleted. |
| REV-SET-008 | Fixed. There are no `??=` operators left in the helper; the replacements behave the same, including the idempotent `close`. |

New findings: none.

Commands run:

| Command | Result |
| --- | --- |
| `go test -race -count=1 ./...` | ok for httpapi, leboncoin, service and store |
| `go vet ./...` | clean |
| `gofmt -l` | clean |
| Node 22 capture tests | 23 of 23 pass |
| `git diff --check` | clean |
| Scratchpad repro tests, rerun on the fixed files | outcomes as in the table above |

Final snapshot (SHA-256, excluding `review.md`; deleted file: `config/config_test.go`):

```text
e0b7c98000939d3380bd6af8d683cf97fc9cd3f9ecb7e31e4e9dd37a386ee434  API_SPECIFICATION.md
952ff28acb6d0cd495894f9822a37f0f6d9e68da49fe2ab20448df05c2af0e8f  config/config.go
9c6ed312de7459c192749fda74b014adf1e8834db401754bc9c53f49d7e78880  doc/changes/leboncoin-session-settings/api-step.md
3a329972b251925ae83ece894a338ce6726dd46dfda1388da99149ce4b1c0429  doc/changes/leboncoin-session-settings/decisions.md
06e4c062c04466b8890c2f51db66508e19949c7484abcf28072afceeb0236ee7  doc/changes/leboncoin-session-settings/functional-step.md
6526c88553dfc38e2e432cdd72a056ff68ee69025187129a12e59ed7ab5e915a  doc/changes/leboncoin-session-settings/implementation-backend.md
2e6953ea9544f3a5797bcf6e55d008efd982d565ecefc6ca0eff007aea6fadd2  doc/changes/leboncoin-session-settings/implementation-capture.md
0f7935acce420210cc442507b5dd7d112d8901494fc1ced1a480db012cf00429  doc/changes/leboncoin-session-settings/implementation-frontend.md
b2abe15a929e87053401eaf81215bc44f43d3ded8642e153dfa52d4dc70873bf  doc/changes/leboncoin-session-settings/index.md
2992694f597400f95e74b8bf9050f78aa14d60a9f4de7987792a5f3a24422b3a  doc/changes/leboncoin-session-settings/technical-step.md
3949f19e0909f7a228ccbd57c55fe22869615a667341cdc03b8add525dfdcc41  doc/deprecated/leboncoin-session-file.md
f3196fd935f903e67532ba6efd3c3e7d298219aa21d33693d38715df51ed3f76  doc/deprecated/README.md
ed62255fa695224baafa1f5d9a8c601eb27ffe62ed15a317108246123814f146  doc/leboncoin-session.md
2c4e3b1d669a9d5dbf243a140719d66f742e702ac3d80ddd5ef64d0cac7652f5  doc/README.md
6e281f763802755f8d859abe8d9b7b382f653a64688bc77d40088616af7fc173  doc/specifications/app-header-menu/functional.md
b0e6e40823f76939823a51c1f7f2c0712112cf21f6fec3d52fc328aa2b86c37b  doc/specifications/app-header-menu/technical.md
c4e34d5561c8a844f2f1151cf5d50eef3cda10942603533f3b3db4674b4e060e  doc/specifications/leboncoin-session-capture/functional.md
4e6e06983298a8cd4eb93b88cfa929df3522c058598d6677f0ea88c0f0a3451a  doc/specifications/leboncoin-session-capture/technical.md
3d8872904d5228a72d1b1128e04988ec6d21d79673dbb48f52b6857b79a32c61  doc/specifications/leboncoin-session-collection/functional.md
4c3b7c9f820a09a8d1e0e78aa2c1cf34039309509241182227812c45d7578d0f  doc/specifications/leboncoin-session-collection/technical.md
91a2bc5ff3653765b2cc0899cbdbab58bde5f9f19eaebfad853f94359105807e  doc/specifications/leboncoin-session-file-removal/functional.md
8291c8921173c1239b23dfd6731d1994ba8eae1fdfa2927dd3dd4dadc03ea2c0  doc/specifications/leboncoin-session-file-removal/technical.md
12a34832e2d87aacafb0a451aefb9fb747ba68ade8b1366eab567e5282ea9fa7  doc/specifications/leboncoin-session-settings/functional.md
d9699f81f56704f38d78081cb7a4d03765581cc5934179449bbdc0b33bb61f0f  doc/specifications/leboncoin-session-settings/technical.md
23477295397c932c49c686ebe84726fd14f4c23b75d2545250e0d69b3b576b4b  internal/httpapi/server.go
a4773f4c9ff32e692be59a9767b108a2397e25f695d8c0d70abbb4f69bcf0a34  internal/httpapi/settings_test.go
cacf33fe0e813c0223df4261ab3717d6f9b295ae9ad10f6d5276232e4ea45741  internal/leboncoin/collector.go
7be3551b422e919a13749c6f16649e828e27c926968322c9cc5d15eb8317c480  internal/leboncoin/session.go
94b8cf8c960408657803f53fb3a1418a4f0201df58eb8c74bcded57a592aff2c  internal/leboncoin/session_test.go
727bd66ed35af5a2cf94be38e0b968d18e79d8d17b3fb5e96ffc7bc0658997b0  internal/model/model.go
51af4b8a98897bfc9a5f96841d76d4f29f4d97cbf2884d6d0baac213c1742243  internal/service/service.go
c1cda85e9e1ddfd3ee36b6266e002365ef2b6d041b3b24ca3d2def422e5a3842  internal/service/session_test.go
87aafdb55b75c516f6f87056a361f95eaced8636e3cf921efa3882b2653bfe76  internal/store/session_test.go
9c747de927cc737146d79014a210ed0b492e6f85c629698506097ad5e30597ca  internal/store/store.go
5bc46ff46c69451f98b79f061483fcb62cacefb4da6fe5492b55f86431ce25d3  package.json
b6ee62244ee1fea0c399c8c03245909631b7d91bd2254c5f492ec263922ce322  playwright.config.ts
b1308b456ac44ccf4e71f1253cc0f462cdfc2afe873f12296c6cafa109badc67  README.md
a48c81d4f3d8d8061091a4185b9e0501c4850dc3315c936c133bf28f872ab810  scripts/capture-leboncoin-session.mjs
35cd9ec86fd671c011ceea8d90c0a175ec2b987db8d5bab49bc21e286d0edd0e  scripts/capture-leboncoin-session.test.mjs
cbcc9a0c28c300166c77f926a6acb55dc04375ace8a472deb736476b45a16fc7  src/api/client.ts
995c30efc6dbd03e717f26797f85adc7930d27c1ac6eec97cb4b7f380ee8c95e  src/api/items.ts
e6e0408d13a065a895b736d6e8d239172d314892956c0bf88b28447d0502737f  src/api/settings.ts
61e316575873d86a6e7599304930094f5c92ed73c15a6e542e1d2c04cf983f14  src/App.tsx
1e8c7e1515fad998e8f13efa588341b8baee83d50f9157ed5e8e42f0cb3d03c3  src/components/AppMenu.tsx
1060235a0350a5bbcf32962163f0c2d0fa4f53053cd0a08501a391af2570140d  src/components/SettingsModal.tsx
6dc4eef99451fdf8a4059fc772b2eea074a3d5b66e88f7eece989729cf7f1e44  tests/leboncoin-session-settings.spec.ts
```

## Recheck 2 (2026-10-03): QA-SET-F01 / QA-SET-F02 fix

Change scope, checked against the Recheck snapshot hashes:

- Code: only `src/components/SettingsModal.tsx` and `tests/leboncoin-session-settings.spec.ts` changed.
- Records: `implementation-frontend.md` and `decisions.md` changed, and a new `qa.md` was added. These are the expected records.
- No other file differs from the Recheck snapshot.

Evaluation:

- **`FeatureFlags enableFocusWrapWithoutSentinels`.**
  - What it is: a documented Carbon feature flag. Both `@carbon/react` `FeatureFlags` and `@carbon/feature-flags` in the installed version declare it.
  - Why it is acceptable: it is scoped to this modal only. It replaces Carbon's timer-based sentinel wrap with Carbon's own synchronous keyboard trap, and it adds no custom focus code.
  - Risk: low. The flag is Carbon's future default. Other modals keep the default behavior, so whether they contain focus is still the open QA retest item recorded in `decisions.md`.
- **`selectorPrimaryFocus` pointing at the footer Cancel button.**
  - Why it is correct: Cancel is enabled in every state reached on open (loading, loaded, load error). The default target, Save, is disabled while loading, which caused QA-SET-F01. Avoiding the close icon button is justified because its tooltip consumes the first Escape, which conflicts with FR-LBC-SET-019.
  - Caveat: the selector depends on Carbon's `cds--` class names. This is acceptable with the pinned Carbon version but would need revisiting on a Carbon upgrade (informational).
  - Spec fit: focus starting on Cancel is consistent with FR-LBC-SET-019 ("Opening moves focus into the modal").
- **Edited existing test ("session value is not kept in browser storage").**
  - The only change is an added wait for the dialog to be hidden after Escape.
  - Effect: the test now asserts what it already intended (the modal closed before storage and the DOM are inspected). This strengthens it and weakens nothing.
- **New tests.**
  - They cover focus on open in the loading, loaded and error states, and Tab/Shift+Tab containment in each state.
  - They check that focus never reaches `header` and that no modal is stacked.
  - The F02 loop presses Enter only on non-Save/Cancel/Close buttons. In practice the dialog contains no such button, so the loop mainly proves containment. Combined with the "exactly one visible dialog" check, this is acceptable.

Commands (Node v22.23.3, `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/google-chrome`). I ran these in a scratchpad copy of the snapshot with the two changed files, to avoid writing `dist/` and test output into the repository:

| Command | Result |
| --- | --- |
| `npm run build` | passed |
| `npx playwright test`, first run | 29 passed, 1 failed: `platform-tabs.spec.ts` "both tabs remain available for empty platform and collection". That run was abnormally slow (4.5 min instead of about 1 min), which points to host load. |
| `npx playwright test tests/platform-tabs.spec.ts --repeat-each 3` | 24 passed |
| `npx playwright test`, rerun | 30 passed |

The platform-tabs failure did not reproduce in 4 later executions (3 repeats plus the full rerun), and this fix does not touch that spec or its code. I record it as a non-reproduced flake under load, not as a finding. QA may watch for it.

New findings: none. QA-SET-F01 and QA-SET-F02 are fixed by code inspection and the passing regression tests. The final QA retest remains the QA agent's stage.

Final snapshot (SHA-256, excluding `review.md`; deleted file: `config/config_test.go`):

```text
e0b7c98000939d3380bd6af8d683cf97fc9cd3f9ecb7e31e4e9dd37a386ee434  API_SPECIFICATION.md
952ff28acb6d0cd495894f9822a37f0f6d9e68da49fe2ab20448df05c2af0e8f  config/config.go
9c6ed312de7459c192749fda74b014adf1e8834db401754bc9c53f49d7e78880  doc/changes/leboncoin-session-settings/api-step.md
0cdd2a91cb35fd543c3036af0aca07fee0cc2568cf4a3a2ae6c26ac46b732cc6  doc/changes/leboncoin-session-settings/decisions.md
06e4c062c04466b8890c2f51db66508e19949c7484abcf28072afceeb0236ee7  doc/changes/leboncoin-session-settings/functional-step.md
6526c88553dfc38e2e432cdd72a056ff68ee69025187129a12e59ed7ab5e915a  doc/changes/leboncoin-session-settings/implementation-backend.md
2e6953ea9544f3a5797bcf6e55d008efd982d565ecefc6ca0eff007aea6fadd2  doc/changes/leboncoin-session-settings/implementation-capture.md
2c8475c2b9cc3b010d65b9834d71d041a338eb75a66f31c6ce3d3ad1a644dc39  doc/changes/leboncoin-session-settings/implementation-frontend.md
b2abe15a929e87053401eaf81215bc44f43d3ded8642e153dfa52d4dc70873bf  doc/changes/leboncoin-session-settings/index.md
571059910af919ea4d50ed1e0c343d60414e608317eea3d6334912cb24ac7e62  doc/changes/leboncoin-session-settings/qa.md
2992694f597400f95e74b8bf9050f78aa14d60a9f4de7987792a5f3a24422b3a  doc/changes/leboncoin-session-settings/technical-step.md
3949f19e0909f7a228ccbd57c55fe22869615a667341cdc03b8add525dfdcc41  doc/deprecated/leboncoin-session-file.md
f3196fd935f903e67532ba6efd3c3e7d298219aa21d33693d38715df51ed3f76  doc/deprecated/README.md
ed62255fa695224baafa1f5d9a8c601eb27ffe62ed15a317108246123814f146  doc/leboncoin-session.md
2c4e3b1d669a9d5dbf243a140719d66f742e702ac3d80ddd5ef64d0cac7652f5  doc/README.md
6e281f763802755f8d859abe8d9b7b382f653a64688bc77d40088616af7fc173  doc/specifications/app-header-menu/functional.md
b0e6e40823f76939823a51c1f7f2c0712112cf21f6fec3d52fc328aa2b86c37b  doc/specifications/app-header-menu/technical.md
c4e34d5561c8a844f2f1151cf5d50eef3cda10942603533f3b3db4674b4e060e  doc/specifications/leboncoin-session-capture/functional.md
4e6e06983298a8cd4eb93b88cfa929df3522c058598d6677f0ea88c0f0a3451a  doc/specifications/leboncoin-session-capture/technical.md
3d8872904d5228a72d1b1128e04988ec6d21d79673dbb48f52b6857b79a32c61  doc/specifications/leboncoin-session-collection/functional.md
4c3b7c9f820a09a8d1e0e78aa2c1cf34039309509241182227812c45d7578d0f  doc/specifications/leboncoin-session-collection/technical.md
91a2bc5ff3653765b2cc0899cbdbab58bde5f9f19eaebfad853f94359105807e  doc/specifications/leboncoin-session-file-removal/functional.md
8291c8921173c1239b23dfd6731d1994ba8eae1fdfa2927dd3dd4dadc03ea2c0  doc/specifications/leboncoin-session-file-removal/technical.md
12a34832e2d87aacafb0a451aefb9fb747ba68ade8b1366eab567e5282ea9fa7  doc/specifications/leboncoin-session-settings/functional.md
d9699f81f56704f38d78081cb7a4d03765581cc5934179449bbdc0b33bb61f0f  doc/specifications/leboncoin-session-settings/technical.md
23477295397c932c49c686ebe84726fd14f4c23b75d2545250e0d69b3b576b4b  internal/httpapi/server.go
a4773f4c9ff32e692be59a9767b108a2397e25f695d8c0d70abbb4f69bcf0a34  internal/httpapi/settings_test.go
cacf33fe0e813c0223df4261ab3717d6f9b295ae9ad10f6d5276232e4ea45741  internal/leboncoin/collector.go
7be3551b422e919a13749c6f16649e828e27c926968322c9cc5d15eb8317c480  internal/leboncoin/session.go
94b8cf8c960408657803f53fb3a1418a4f0201df58eb8c74bcded57a592aff2c  internal/leboncoin/session_test.go
727bd66ed35af5a2cf94be38e0b968d18e79d8d17b3fb5e96ffc7bc0658997b0  internal/model/model.go
51af4b8a98897bfc9a5f96841d76d4f29f4d97cbf2884d6d0baac213c1742243  internal/service/service.go
c1cda85e9e1ddfd3ee36b6266e002365ef2b6d041b3b24ca3d2def422e5a3842  internal/service/session_test.go
87aafdb55b75c516f6f87056a361f95eaced8636e3cf921efa3882b2653bfe76  internal/store/session_test.go
9c747de927cc737146d79014a210ed0b492e6f85c629698506097ad5e30597ca  internal/store/store.go
5bc46ff46c69451f98b79f061483fcb62cacefb4da6fe5492b55f86431ce25d3  package.json
b6ee62244ee1fea0c399c8c03245909631b7d91bd2254c5f492ec263922ce322  playwright.config.ts
b1308b456ac44ccf4e71f1253cc0f462cdfc2afe873f12296c6cafa109badc67  README.md
a48c81d4f3d8d8061091a4185b9e0501c4850dc3315c936c133bf28f872ab810  scripts/capture-leboncoin-session.mjs
35cd9ec86fd671c011ceea8d90c0a175ec2b987db8d5bab49bc21e286d0edd0e  scripts/capture-leboncoin-session.test.mjs
cbcc9a0c28c300166c77f926a6acb55dc04375ace8a472deb736476b45a16fc7  src/api/client.ts
995c30efc6dbd03e717f26797f85adc7930d27c1ac6eec97cb4b7f380ee8c95e  src/api/items.ts
e6e0408d13a065a895b736d6e8d239172d314892956c0bf88b28447d0502737f  src/api/settings.ts
61e316575873d86a6e7599304930094f5c92ed73c15a6e542e1d2c04cf983f14  src/App.tsx
1e8c7e1515fad998e8f13efa588341b8baee83d50f9157ed5e8e42f0cb3d03c3  src/components/AppMenu.tsx
c53193851244bf120dee8fef2466edd3e9511b8bcc4608986755b332cc4d393f  src/components/SettingsModal.tsx
3d5cec9c84f80471b3ff6e26457ac623d078abef4ff72beec6e46fc5ed96c94a  tests/leboncoin-session-settings.spec.ts
```
