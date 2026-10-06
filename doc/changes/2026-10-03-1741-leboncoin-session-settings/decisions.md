# Review decisions: leboncoin-session-settings

Reviewed specification revisions: working-tree versions listed in [review.md](review.md) (change [index](index.md) user decisions and API-gate decisions of 2026-10-03; app-header-menu, leboncoin-session-settings, leboncoin-session-collection rev. 2, leboncoin-session-capture rev. 2, leboncoin-session-file-removal functional/technical files; [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) section “LeBoncoin session settings (approved 2026-10-03)”).\
Reviewed code revision: working tree on base `509460fc85422c5a7b0a7cb35e47444551c95b80`, snapshot identified by the SHA-256 list in [review.md](review.md). The adjudicator re-hashed the cited files (`API_SPECIFICATION.md`, `internal/httpapi/server.go`, `internal/leboncoin/session.go`, `internal/leboncoin/collector.go`, `internal/service/service.go`, `scripts/capture-leboncoin-session.mjs`) on 2026-10-03: all match the reviewed snapshot.\
Reviewer: independent reviewer agent, [review.md](review.md) (SHA-256 `196ee378…c495` at adjudication).\
Adjudicator: independent review adjudicator agent (separate invocation; did not specify, implement or review this change), 2026-10-03.

Independent verification performed by the adjudicator (scratchpad only, synthetic values, no network):

- Go 1.25 `net/http` cookie parsing (`readSetCookies`, scratchpad program): `…; Path=/; Secure; SameSite=Lax` → `Unparsed=[]`; the same plus `; Priority=High` → `Unparsed=["Priority=High"]`; `Max-Age=abc` with a valid `Expires` → `Unparsed=["Max-Age=abc"]`, `MaxAge=0`, `Expires` set; `Expires=garbage` → `Unparsed=["Expires=garbage"]`, `Expires` zero. So `Unparsed` mixes two different things: unknown attributes (harmless) and malformed known metadata (must stay rejected). Confirms REV-SET-004.
- Code reading of `handleLeboncoinSession` (`internal/httpapi/server.go`): the trailing `decoder.Decode(&struct{}{})` maps any non-EOF error, including `*http.MaxBytesError`, to 400 `INVALID_JSON`. Confirms REV-SET-002.
- Code reading of `ParseSessionInput`: `Cookie:` → empty after prefix strip → no `=`/`;` → `validCookieValue("")` false → “invalid characters” message. Confirms REV-SET-003.
- `CollectWithSession` returns `attempt.outcome()` on the canonical-URL failure branch, and `collectLeboncoin` calls `FinishLeboncoinSessionAttempt` unconditionally. Confirms REV-SET-005.
- `scripts/capture-leboncoin-session.mjs` has three `??=` uses (lines 169, 178, 294; the review says four) and top-level `await` on the last line. No other post-ES2020 syntax was found by a targeted grep (private fields, static blocks, `||=`/`&&=`). Confirms REV-SET-008.
- `API_SPECIFICATION.md` line 257 (section status line) still reads “Status: proposed 2026-10-03, NOT approved.”; `index.md` still says “No API contract approval has been given yet” and lists the API confirmation and R-1 as pending. Confirms REV-SET-001.
- `doc/deprecated/leboncoin-session-file.md` line 59 is still a placeholder; `session/` (empty, 0700) and `.tmp-session-qa/main.go` exist untracked. Confirms REV-SET-006 and 007.

## Findings and decisions

### REV-SET-001: Approved API section and specification statuses still say “not approved / not authorized”

- Evidence: `API_SPECIFICATION.md` section status line (“proposed 2026-10-03, NOT approved … Neither tier may be implemented until the user confirms this section”) and a doubled blank line near the top; `index.md` (“No API contract approval has been given yet for this change.”, the “Pending user decisions from the technical stage” list, “ready (API pending)” status of the settings technical file, stage line 4 “pending”); status lines of the five technical files (“API proposal pending user confirmation; implementation not authorized”); the settings technical file still says `request`/`ApiError` are reused from `src/api/items.ts` and that R-1 is not included. The user decisions at the API gate (index, 2026-10-03) are: contract “Approve” (revision 1 unchanged), R-1 “Do it first”, expiry “Yes, as proposed”.
- Impact and scenario: the canonical contract contradicts itself (heading “approved”, status “NOT approved”), and the committed specifications would state the implementation was unauthorized. A later contributor cannot tell from the canonical source whether the endpoints are approved, which undermines the workflow's API gate. No runtime or data effect.
- Criticality: non-critical, because no user, data, security or accessibility consequence exists and the true approval is recorded in the index; the defect is in record-keeping. It is nevertheless a mandatory fix before the commit stage: workflow step 3 requires `API_SPECIFICATION.md` to record the approved contract, and committing a canonical contract that declares itself unapproved would violate that obligation.
- Disposition: fix.
- Reason: cheap, wording-only correction; leaving it would permanently record a false state in the canonical contract.
- Required change (owner: coordinator; documentation/status only, no change to contract content):
  1. `API_SPECIFICATION.md`: replace the section status line with an approved status (“Approved by the user 2026-10-03, proposal revision 1 without change”, linking the index); remove the duplicate blank line. The endpoint, field, status and error definitions must remain byte-identical.
  2. `index.md`: replace “No API contract approval has been given yet” and the pending-decisions list with references to the recorded API-gate decisions (history may be kept as “was pending until 2026-10-03”); update the technical status “ready (API pending)” and the stage line.
  3. Status lines/API sections of the five technical files: approved/implemented-authorized wording referencing the index decisions.
  4. Settings technical file: `request`/`ApiError` live in `src/api/client.ts` (R-1 performed first per the user's “Do it first”).
- Specification decision: not applicable (records already-made user decisions; no new approval needed).
- Validation condition: reviewer recheck shows no remaining “NOT approved”, “not authorized”, “pending user confirmation” or “No API contract approval” text for this change outside history phrasing; `git diff` of `API_SPECIFICATION.md` for this fix touches only the status line and blank line.
- Resolution: pending.
- Remaining risk / follow-up: none after recheck.

### REV-SET-002: Body whose first JSON value fits but total exceeds 32 768 bytes returns 400 instead of 413

- Evidence: `internal/httpapi/server.go` `handleLeboncoinSession` PUT: the second `decoder.Decode` maps every non-EOF error to 400 `INVALID_JSON`, including `*http.MaxBytesError`; reviewer reproduction with 40 000 trailing bytes → 400. Approved API error table: “413 `REQUEST_TOO_LARGE` — Body over 32 768 bytes”; TS-LBC-SET-004 “Over-size → 413”.
- Impact and scenario: a client sending an oversized body with trailing data gets the wrong status code. Nothing is saved in either case; the real frontend never sends such a body. The same pattern exists in the pre-existing `POST /api/v1/items` handler.
- Criticality: non-critical, because no data is changed or exposed, only a malformed client request is affected, and the request is still refused. It is however a demonstrated deviation from the approved contract in new code.
- Disposition: fix (new handler only).
- Reason: the contract is explicit, the fix is two lines in code introduced by this change, and leaving a known contract mismatch in a newly approved endpoint is not justified. `POST /items` is pre-existing and outside this change's approved scope; changing it here would be an unapproved change.
- Required change (owner: backend developer): in the PUT branch, if the trailing `Decode` error is a `*http.MaxBytesError`, respond 413 `REQUEST_TOO_LARGE` “Request body is too large.”; keep 400 `INVALID_JSON` for other trailing content. Add a test: valid first value followed by enough whitespace/garbage to exceed 32 768 bytes → 413, session unchanged.
- Specification decision: not applicable (implements the approved contract).
- Validation condition: new test passes; existing settings tests pass; `go test -race ./...`, `go vet`, `gofmt` clean; reviewer recheck.
- Resolution: pending.
- Remaining risk / follow-up: the identical pre-existing behavior of `POST /api/v1/items` remains. It is reported to the user through the coordinator as an optional separate change; it is not a blocker for this change.

### REV-SET-003: A bare `Cookie:` / `Set-Cookie:` prefix is reported as invalid characters

- Evidence: `internal/leboncoin/session.go` `ParseSessionInput`; TS-LBC-SET-002 rules 3, 4, 6 applied literally make an input that is empty after prefix removal fall into rule 6 with the “spaces, quotes or other characters” message. FR-LBC-SET-006 requires “a message stating the problem”; its list of unusable inputs includes “a cookie string without a `datadome` pair”, and its example message is “No datadome cookie was found in the pasted text.”
- Impact and scenario: an operator who pastes only a header name (e.g. copies `Cookie:` without the value) is told the value contains invalid characters, which is false and points them in the wrong direction. Nothing is saved (correct). Likelihood low.
- Criticality: non-critical, because the input is still rejected, no data is affected, and only the message text is inaccurate in an unlikely case.
- Disposition: fix.
- Reason: the implementation follows the technical rule order, but that order produces a message that does not state the problem, which FR-LBC-SET-006 requires. A header prefix with nothing after it is a cookie string without a `datadome` pair, a case FR-LBC-SET-006 already lists, so the existing message “No datadome cookie was found in the pasted text.” is correct. This does not invent a requirement or a new message, and it does not change the API: the response is still 400 `INVALID_SESSION`, and the message is the API section's own example.
- Required change: (a) owner coordinator, who routes the wording to the technical specifier: amend TS-LBC-SET-002 rule 3: “if a prefix was stripped and the remainder is empty → ‘No datadome cookie was found in the pasted text.’”; (b) owner backend developer: implement it in `ParseSessionInput` and add tests for `Cookie:`, `set-cookie:  ` and `Cookie: ;`. (`Cookie: ;` already goes through rule 5 and returns “No datadome cookie…”; the test pins that behavior.)
- Specification decision: technical clarification within FR-LBC-SET-006; no API change and no product choice. The user does not need to approve it, but it is reported for information.
- Validation condition: the new parser tests pass, and so do the existing ones. Reviewer recheck confirms the spec text and code agree.
- Resolution: pending.
- Remaining risk / follow-up: none.

### REV-SET-004: Any unrecognized `Set-Cookie` attribute silently disables renewal and revocation detection

- Evidence: `internal/leboncoin/session.go` `SetCookies` skips a `datadome` cookie when `len(raw.Unparsed) > 0`. The adjudicator confirmed that Go 1.25 puts unknown attributes (`Priority=High`) into `Unparsed`, and also malformed known metadata (`Max-Age=abc`, `Expires=garbage`). Revision 1 (TS-LBC-COL-002) required “Reject overflow/invalid metadata safely”. It did not require rejecting unknown attributes. Revision 2 (TS-LBC-COL-009) does not mention unknown attributes. The backend record states “unparsed attributes rejected”, an interpretation the specification does not mandate. User decision 3 / FR-LBC-COL-014 require storing a renewed cookie. FR-LBC-COL-015 requires honoring an explicit deletion.
- Impact and scenario: suppose LeBoncoin/DataDome's real `datadome` `Set-Cookie` carries any attribute that Go does not model (`Priority` is a common Chrome-era attribute; others may be added at any time). Then every renewal and every revocation is ignored silently, for every item and every attempt. The saved value is never refreshed, and revocation is never shown in Settings. It is discovered only when LeBoncoin starts rejecting the stale value. This defeats an explicit user requirement. The behavior is fail-safe for security: nothing out of scope is stored.
- Criticality: non-critical on current evidence, with an explicit condition. The attribute set of the real header has not been observed in this change, so the defect may not trigger today, and failure only degrades to the old manual-renewal path (FR-LBC-COL-015 guidance and the rejection hint still work). If evidence shows that the real `datadome` `Set-Cookie` carries an attribute that Go reports in `Unparsed`, and the fix below has not landed, the finding becomes critical: FR-LBC-COL-014 and FR-LBC-COL-015 would then fail deterministically for all users. Lack of an observation is not proof of safety, so the fix does not wait for one.
- Disposition: fix.
- Reason: a QA observation of the real header could only show whether the defect triggers today. It cannot make the behavior correct, because DataDome can add an attribute at any time without notice and the failure is silent. The robust behavior is cheap. It follows the retained rule's intent (“invalid metadata”) and keeps every security constraint: an exact name, `.leboncoin.fr` domain, `/` path, allowed HTTPS origin, validated value and validated expiry. So it does not rely on the observation. A QA observation is still useful as supplementary evidence.
- Required change:
  1. Technical specification (owner: coordinator, who routes the wording to the technical specifier). Amend TS-LBC-COL-009: “A `datadome` `Set-Cookie` is ignored if it carries malformed metadata. Malformed metadata is an `Expires`, `Max-Age`, `Domain`, `Path`, `SameSite`, `Secure`, `HttpOnly` or `Partitioned` attribute that the parser could not interpret (it appears in Go's `Cookie.Unparsed`). Attributes with any other name are ignored and do not prevent renewal or deletion.”
  2. Backend developer. In `SetCookies`, replace `len(raw.Unparsed) > 0` with a check that skips the cookie only when an `Unparsed` entry's attribute name is one of those known names. The name is the text before `=`, trimmed and matched case-insensitively. Add tests with the existing verified-listing fixture:
     - `…; Priority=High` → `Renewed: true`;
     - a deletion (`Max-Age=0`) with `Priority=High` → `Revoked: true`;
     - `Max-Age=abc` with a valid future `Expires` → ignored, no renewal;
     - `Expires=garbage` → ignored;
     - the existing scope, ambiguity and overflow tests keep passing.
  3. QA (supplementary, not a closing condition): if the coordinator authorizes a live LeBoncoin request during QA, record the attribute **names only** of a real `datadome` `Set-Cookie`. Do not record values, and do not record cookie values in any artifact. If live requests are not authorized, record that limitation in QA.
- Specification decision: technical clarification of TS-LBC-COL-009 implementing FR-LBC-COL-014/015; no API change and no product choice.
- Validation condition: the new tests pass; `go test -race ./...` passes; the reviewer confirms that the code matches the amended TS-LBC-COL-009 and that the backend record's mapping row is corrected.
- Resolution: pending.
- Remaining risk / follow-up: whether a real renewal arrives on verified pages at all is outside this change's control. The rejection hint covers that path.

### REV-SET-005: Canonical-URL failure records a “failed” session attempt although nothing was sent

- Evidence: `internal/leboncoin/collector.go` `CollectWithSession` returns `attempt.outcome()` (`Attempt: "failed"`) when `ParseURL` rejects the stored URL, before any request. `internal/service/service.go` `collectLeboncoin` then always calls `FinishLeboncoinSessionAttempt`, which sets `last_attempt_*`. Approved API field `lastAttempt`: “Latest LeBoncoin check that **sent** this session”; FR-LBC-COL-017.
- Impact and scenario: this happens only with a stored LeBoncoin item whose URL does not parse (corrupt row, a legacy row, or a future tightening of `ParseURL`). The Settings modal would then show a “failed” hint for the session although LeBoncoin never saw it. That could lead the operator to replace a working session. It cannot happen in normal operation because URLs are validated with the same `ParseURL` at add time.
- Criticality: non-critical, because the branch is unreachable with data created through the API, the consequence is a misleading hint rather than data loss, and no value is exposed.
- Disposition: fix.
- Reason: it is a deviation from the approved field semantics in new code. The fix is small and local, and deferring would leave a known contract mismatch with no compensating benefit. A later `ParseURL` tightening would make the branch reachable for existing rows.
- Required change (owner: backend developer): on that branch, return a zero `SessionOutcome{}` (no request was sent). In `collectLeboncoin`, skip `FinishLeboncoinSessionAttempt` when `outcome.Attempt == ""`. The item result (`request_error` via `collectionError`) is unchanged. Add a service or collector test: a session plus an invalid stored URL → no session row change (`revision`, `last_attempt_*` unchanged). Record the behavior in the backend implementation record. The coordinator should have the technical specifier note it in TS-LBC-COL-008/009: “no session outcome is recorded when no request was sent”.
- Specification decision: not applicable (implements the approved API field definition).
- Validation condition: the new test passes; existing service and collector tests pass; reviewer recheck.
- Resolution: pending.
- Remaining risk / follow-up: none.

### REV-SET-006: Deprecation record still lacks the removal commit message(s)

- Evidence: `doc/deprecated/leboncoin-session-file.md` line 59: “Removal commits (change leboncoin-session-settings): to be recorded by message at commit time.” FR-LBC-RM-005 requires the record to reference the removal commit(s) by message.
- Impact and scenario: until the commit stage, FR-LBC-RM-005 is not met. If forgotten, the deprecation trail lacks its link to the removal.
- Criticality: non-critical, because it is an expected placeholder that can only be filled at the commit stage and has no runtime effect. It is a mandatory completion condition: the change is not complete while FR-LBC-RM-005 is unmet.
- Disposition: fix (at the commit stage).
- Reason: the requirement is explicit, and the exact message can be chosen before committing and cited verbatim.
- Required change (owner: coordinator): choose the removal commit message(s), write them exactly into the deprecation record (replacing the placeholder) in the same commit or a documented follow-up commit, and record this in `commit-step.md`.
- Specification decision: not applicable.
- Validation condition: the placeholder is gone, the cited message(s) match `git log` exactly, and `commit-step.md` records it.
- Resolution: pending (commit stage).
- Remaining risk / follow-up: none after the commit stage.

### REV-SET-007: Leftover untracked artifacts outside the change must stay out of the commits

- Evidence: `.tmp-session-qa/main.go` (untracked, dated 2026-10-03 11:32, from the earlier `leboncoin-session` QA; references removed `NewCollectorWithSession` / `LeboncoinSessionFile`; outside `./...` because of the leading dot). The `session/` directory is empty, mode 0700, dated 15:33, and does not appear in `git status`.
- Impact and scenario: none if they are not staged. If `.tmp-session-qa/main.go` were committed, it would add a non-compiling live reference to the removed mechanism (contrary to FR-LBC-RM-004). An empty directory cannot be committed.
- Criticality: non-critical, because neither artifact is part of the diff, the build or the runtime.
- Disposition: reject (no change to the change's diff), with a staging constraint.
- Reason: these are not defects of the implementation. Deleting them is not a change this workflow authorizes without the user. AGENTS.md requires preserving existing user work and explicit staging, and the files may belong to the user's earlier QA. The correct handling is to leave them untouched and keep them out of the commits.
- Required action (owner: coordinator): use explicit path staging only; confirm in `commit-step.md` that neither `.tmp-session-qa/` nor `session/` was staged. Offer the user the optional deletion of both after QA.
- Specification decision: user decision only on the optional cleanup (non-blocking).
- Validation condition: `git show --stat` of the change's commits contains neither path.
- Resolution: rejected (no code/doc change); staging constraint pending at the commit stage.
- Remaining risk / follow-up: the stale `.tmp-session-qa/main.go` stays in the working tree until the user decides.

### REV-SET-008: Capture helper cannot report the runtime requirement on Node.js older than 15

- Evidence: `scripts/capture-leboncoin-session.mjs` uses `??=` at lines 169, 178 and 294 (ES2021, Node ≥ 15). Node 14 fails to parse the module before `main` runs, so the “Node.js 20.19 or newer is required” message is never printed. TS-LBC-CAP-008 item 2: “The check uses only syntax that older Node versions parse.” The file's own line-2 comment states the same intent. FR-LBC-CAP-018.
- Impact and scenario: an operator on Node ≤ 14 (end of life) gets a raw `SyntaxError` instead of the explanation. Node 16 and 18 reach the check correctly. Likelihood low.
- Criticality: non-critical, because the affected runtimes are end-of-life, the helper refuses to run in either case, and no data or security consequence exists.
- Disposition: fix.
- Reason: it is an explicit technical requirement, and the module's comment claims compliance. The fix is three mechanical rewrites with no behavior change. Top-level `await` (Node ≥ 14.8) is retained; fully supporting older ESM loaders is not reasonable and not required. The parse floor becomes Node 14.8, stated in the comment or guide.
- Required change (owner: capture developer): replace each `x ??= y` with an equivalent explicit form (`if (x === undefined || x === null) x = y;`, or `x = x ?? y` / `x ?? (x = y)` where the expression value is used, as at line 178), preserving semantics. Optionally state “parses on Node.js 14.8+” in the line-2 comment. No other change.
- Specification decision: not applicable.
- Validation condition: `grep -n '??=' scripts/capture-leboncoin-session.mjs` returns nothing. `node --test scripts/capture-leboncoin-session.test.mjs` passes on Node 20.19 and 22. Where available, `node --check` with a Node 14.8+ binary, or an ES2020 module parse (e.g. acorn `ecmaVersion: 2020`, top-level await allowed), succeeds. Reviewer recheck.
- Resolution: pending.
- Remaining risk / follow-up: future edits may reintroduce newer syntax. The line-2 comment is the guard.

## Release readiness

- No finding is critical. REV-SET-004 becomes critical only if real `Set-Cookie` headers carry an unmodeled attribute **and** its fix is not made; the fix is required regardless.
- Required fixes before completion:
  - Backend developer: REV-SET-002, 003, 004, 005.
  - Capture developer: REV-SET-008.
  - Coordinator: REV-SET-001 status/text corrections, plus routing the TS-LBC-SET-002 and TS-LBC-COL-008/009 clarifications for REV-SET-003, 004 and 005 to the technical specifier before or with the backend fix.
  - Commit stage (coordinator): REV-SET-006 and the REV-SET-007 staging constraint.
- Every fix must be re-reviewed against an updated snapshot. The full check set runs again: `go test -race ./...`, `go vet`, `gofmt`, `node --test`, `npm run build`, Playwright, and `scripts/build-release.sh 6` (not yet run in this change).
- QA (stage 7) has not run. It still owes the items carried by the review: visible-window confirmation (FR-LBC-CAP-014), the real-Chrome close path, a keyboard pass of the menu/modal, and optionally the REV-SET-004 attribute-name observation.
- No API change is required by any decision. The approved contract text stays unchanged except the REV-SET-001 status line, which records an existing approval.
- Items for the user (none blocking):
  1. Optional cleanup of untracked `.tmp-session-qa/` and empty `session/` (REV-SET-007).
  2. For information: the same 400-instead-of-413 behavior exists in pre-existing `POST /api/v1/items` (REV-SET-002). Fixing it would be a separate change if wanted.
  3. Whether QA may make a live LeBoncoin request to record `Set-Cookie` attribute names (REV-SET-004). This is supplementary, not required.
  4. For information: technical clarifications TS-LBC-SET-002 (REV-SET-003) and TS-LBC-COL-008/009 (REV-SET-004, 005). They implement existing functional requirements and the approved contract without product or API change.

## Recheck outcome (2026-10-03)

The adjudicator verified the reviewer's [Recheck](review.md#recheck-2026-10-03) section independently. The hashes of `server.go`, `session.go`, `collector.go`, `service.go`, the capture script and `API_SPECIFICATION.md` match the reviewer's final snapshot. `go test -count=1 ./internal/...` passes. The code shows each fix: the new MaxBytesError branch, the empty-after-prefix check, the known-attribute-only `Unparsed` rule, the `Attempt == ""` skip, and no remaining `??=`. `.tmp-session-qa/` and `session/` are absent.

| Finding | Status |
| --- | --- |
| REV-SET-001 | Fixed and verified (reviewer recheck plus grep). |
| REV-SET-002 | Fixed and verified for the contract case. Residual: an oversized body whose trailing content is invalid JSON still returns 400 `INVALID_JSON`. I accept this as non-critical and needing no further change: the body is invalid JSON regardless of size, nothing is saved, and the decoder reports the syntax error before the size limit is reached, so neither code is incorrect for that input. |
| REV-SET-003 | Fixed and verified. |
| REV-SET-004 | Fixed and verified. The optional QA attribute-name observation is still open, as supplementary evidence only. |
| REV-SET-005 | Fixed and verified. |
| REV-SET-006 | Open until the commit stage (coordinator; `commit-step.md` must cite the exact message). |
| REV-SET-007 | Resolved: the user approved deletion and both leftovers are deleted. They are no longer a staging concern. |
| REV-SET-008 | Fixed and verified. |

## Gate state

- No critical findings. No open review blockers, apart from REV-SET-006, which belongs to the commit stage by design.
- The review-decision stage is passed for the reviewed final snapshot.
- QA (stage 7) is pending. Any change after QA requires a new review and adjudication.
- Not yet run: `scripts/build-release.sh 6`. The frontend build and Playwright were not rerun because no frontend file changed since the passing run.

## QA findings (stage 7, [qa.md](qa.md))

Verification by the adjudicator: I read the QA reproduction steps and checked them against `src/components/SettingsModal.tsx` (SHA-256 prefix `1060235a0350`, the same as the QA-tested revision). On open the modal renders only `InlineLoading`, Save is disabled (`primaryButtonDisabled={!settings || …}`), and the `TextArea` is disabled until the settings load. The modal sets no explicit initial-focus target. This fits QA's observation that focus stays on `BODY`, while the Add item modal, whose primary button is enabled, receives focus. I did not rerun the browser; the QA evidence (headless and headed Chrome 149, screenshot `stacked-modals.png`) is concrete and reproducible. The mocked Playwright suite missed both failures because it never asserts the initial focus position or a Tab press from the open state.

### QA-SET-F01: Settings modal does not move focus into the dialog, and Tab leaves it

- Evidence: QA case UI-SET-19a. `document.activeElement` is `BODY` at 100, 600 and 1500 ms. Tab then reaches the header and page controls outside `[role=dialog]`. FR-LBC-SET-019 says: "Opening moves focus into the modal; focus stays inside the modal while open."
- Impact and scenario: keyboard and screen-reader users are not placed in the dialog and can operate the page behind it. This also makes QA-SET-F02 possible. It affects every keyboard-initiated or mouse-initiated open.
- Criticality: **critical**. It is a demonstrated failure of an explicit accessibility acceptance criterion in every use of the new modal, not an edge case. It directly enables QA-SET-F02, which breaks an explicit menu requirement.
- Disposition: fix.
- Required change (owner: frontend developer):
  - Ensure focus enters the dialog when it opens, including during loading and load-error states, using the Carbon Modal pattern. Give the modal a focus target that is focusable in every state, for example via `selectorPrimaryFocus` / `data-modal-primary-focus`. Candidates are the field once it is loaded and the close or Cancel button while loading or on error.
  - Keep focus inside the dialog in all states, including after the loading-to-loaded transition.
  - Do not change any behavior beyond FR-LBC-SET-019: the loading, disabled-while-loading, Escape and focus-return behavior stays as specified.
- Specification decision: not applicable (implements FR-LBC-SET-019; no API change).
- Validation:
  - A new Playwright regression test in `tests/leboncoin-session-settings.spec.ts` with a delayed GET. It asserts that focus is inside `[role=dialog]` right after opening during loading, after loading, and in the load-error state. It also asserts that repeated Tab and Shift+Tab presses never move focus outside the dialog.
  - The full Playwright suite and `npm run build` pass.
  - The reviewer rechecks the diff.
  - QA retests UI-SET-19a (and UI-SET-11a/c focus return) against the built app.
- Resolution: pending. This is a blocker.
- Remaining risk: none once validated. The screen-reader announcement is still untested; this is a QA limitation already recorded.

### QA-SET-F02: An Add item modal can be stacked over the Settings modal

- Evidence: QA case UI-MENU-06. With Settings open, two Tab presses reach the header **Add item** button, and Enter opens "Add tracked item" on top of Settings (`stacked-modals.png`). The Application menu trigger is also reachable. FR-APP-MENU-006 says: "While a modal is open the header control is not reachable behind the modal; the menu cannot be used to stack the settings modal over another modal."
- Impact and scenario: two modals open at once. An add-item save while the Settings modal is open, with its own pending revision and focus-return logic, leaves the interface in an unspecified state. The menu trigger is reachable, so stacking Settings itself may also be possible. This is reachable by any keyboard user.
- Criticality: **critical**. It is a demonstrated violation of an explicit functional requirement with a simple keyboard reproduction, and it can put the UI in unspecified states.
- Disposition: fix. It is probably resolved by the QA-SET-F01 fix, but resolution must be shown independently.
- Required change (owner: frontend developer): after the QA-SET-F01 focus containment fix, verify that no header or page control behind the Settings modal can be reached or activated by keyboard. If containment alone does not achieve that, prevent it with the Carbon modal pattern, not with a custom global mechanism. Do not change the Add item or Delete modals beyond what is needed for this requirement; if they also show the same reachability, report it to the coordinator before changing them.
- Specification decision: not applicable (implements FR-APP-MENU-006).
- Validation:
  - A new Playwright regression test. Open Settings and press Tab and Enter repeatedly. Assert that exactly one `[role=dialog]` is visible and that the header **Add item** and the Application menu trigger never receive focus.
  - Optionally, the reverse case: open Add item, then assert that the menu trigger cannot be reached.
  - The reviewer rechecks the diff.
  - QA retests UI-MENU-06 (and UI-SET-19a).
- Resolution: pending. This is a blocker.
- Remaining risk: whether the existing Add item or Delete modals contain focus correctly was not part of the failing evidence. The Add item modal focuses its own button; QA's retest should note any reachability of the menu trigger from those modals.

## Gate state (after QA)

- **Blocked.** QA-SET-F01 and QA-SET-F02 are critical and open. They are assigned to the frontend developer.
- They are resolved only after all of the following:
  - the frontend fix;
  - the new Playwright regression tests passing with the full suite and `npm run build`;
  - an independent reviewer recheck;
  - a QA retest of UI-SET-19a and UI-MENU-06 that passes against the rebuilt application.
- REV-SET-006 remains open for the commit stage. All other review findings are verified as resolved, as recorded above.
- Not yet run: `scripts/build-release.sh 6`. No user decision is needed for these findings.

## QA retest outcome ([qa.md](qa.md) "Retest")

- **QA-SET-F01: fixed and verified.** `SettingsModal.tsx` (SHA-256 prefix `c53193851244`, matching the retested revision) sets `selectorPrimaryFocus` to Cancel and wraps the modal in Carbon `FeatureFlags enableFocusWrapWithoutSentinels`. In R-UI-SET-19a, focus was on Cancel while loading and after loading, and 0 of 24 fast presses left the dialog, at both 1280 px and 400 px. Reviewer "Recheck 2" covered the diff, and the Playwright suite (30 tests, including the new regressions) passed twice.
- **QA-SET-F02: fixed and verified.** In R-UI-MENU-06, focus never reached the header and no stacked modal appeared, at both widths.

### QA-SET-F03: Add item and Delete modals let focus escape on fast Tab/Shift+Tab

- Evidence: QA cases R-REC-ADD and R-REC-DEL. With no delay between key presses, focus ended outside the dialog after 11–12 of 20 presses (Add item) and 14 of 20 presses (Delete): on Carbon's focus sentinel, `BODY`, or header controls. With 200 ms between presses, 0 of 6 presses escaped. In R-REC-STACK, the header Add item was reached, but Enter did not stack a second modal.
- Origin (verified by the adjudicator): this is **pre-existing**. `src/components/AddItemModal.tsx` and `DeleteModal.tsx` are unchanged from HEAD; they were last modified in `873353d` ("Add LeBoncoin listing tracking and refresh"). The diff of `src/App.tsx` only adds the menu, the settings state and `SettingsModal`; it does not change how those modals are rendered. The behavior comes from Carbon's sentinel-based focus wrap, which this change bypassed only for the Settings modal.
- Scope interaction: this change adds one new header control, the Application menu trigger, which can now be one of the escape targets. FR-APP-MENU-006 says the menu "cannot be used to stack the settings modal over another modal". Doing so from Add item or Delete would require focus to escape onto the trigger and then three activations (open the menu, choose Settings) during the escape window. QA did not observe this or test it directly. The escape has been seen only with machine-speed input, and whether human typing triggers it was not established.
- Criticality: **non-critical** for this change. Reasons:
  - The defect predates this change and lies in components this change does not touch.
  - It reproduces only at machine-speed key rates; at 200 ms intervals no press escaped.
  - No stacking was observed.
  - No data is affected.
  - Uncertainty: human-speed reachability is not proven absent. That uncertainty is recorded as residual risk, not treated as proof of safety.
- Disposition: **defer** (outside this change's scope).
- Reason: the fix would be to apply the same Carbon focus-wrap option, and possibly an explicit primary focus, to `AddItemModal` and `DeleteModal`. Those are unrelated components. Changing them here would be an unapproved change outside the approved specifications (AGENTS.md: narrow scope; refactoring and out-of-scope work go to the user). The Settings modal, which this change owns, meets FR-LBC-SET-019 and FR-APP-MENU-006 as verified above. Releasing this change does not make the pre-existing behavior worse beyond adding the menu trigger as one more reachable control, and that path requires machine-speed input.
- Specification decision: **the user decides** whether to open a separate change for focus containment in the Add item and Delete modals. If opened, it should cover:
  - Carbon focus wrapping without sentinels for both modals;
  - Playwright fast-Tab regression tests for both;
  - a check that the Application menu cannot open Settings over either modal.
- Resolution: deferred, pending the user's choice of a follow-up change.
- Follow-up: owner coordinator, who presents the item to the user; if approved, the work goes to a frontend developer in a separate change.
- Remaining risk: a very fast keyboard user or automation may move focus behind the Add item or Delete modal. In the worst case, this could open the Application menu and Settings over another modal. This was not observed and is considered unlikely at human speed.

## Final gate state

- Review findings:
  - REV-SET-001, 002, 003, 004, 005 and 008 are fixed and verified.
  - REV-SET-007 is resolved.
  - **REV-SET-006 is open until the commit stage.** The coordinator must cite the exact removal commit message in `doc/deprecated/leboncoin-session-file.md` and `commit-step.md`.
- QA findings: QA-SET-F01 and F02 (critical) are fixed and verified by the regression tests, reviewer Recheck 2 and the QA retest. QA-SET-F03 is non-critical and pre-existing; it is deferred to a user decision about a separate change.
- **No critical finding is open.** The review and QA gates are passed for the retested snapshot. Any further code change requires another review and QA pass.
- Before or at the commit stage: run `scripts/build-release.sh 6` and record the result (not yet recorded), and complete REV-SET-006.
- For the user: whether to open the QA-SET-F03 follow-up change. This decision is not blocking.
