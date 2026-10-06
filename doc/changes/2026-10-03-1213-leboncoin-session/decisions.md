# Review decisions: LeBoncoin verified sessions

Status: review decisions complete for the final reviewed snapshot (2026-10-03, second recheck). QA-LBC-F01 (critical, fix) is **resolved: fixed and independently verified**. No open critical blocker. Two QA coverage gaps are recorded as noncritical deferrals (QA-DEF-001, QA-DEF-002) that the coordinator must report to the user.

Reviewed specifications: [capture functional requirements](../../specifications/leboncoin-session-capture/functional.md), [capture technical design](../../specifications/leboncoin-session-capture/technical.md), [collection functional requirements](../../specifications/leboncoin-session-collection/functional.md), and [collection technical design](../../specifications/leboncoin-session-collection/technical.md), ready working-tree versions dated 2026-10-03. The [approved HTTP API](../../../API_SPECIFICATION.md) is unchanged.

Reviewed code revision: uncommitted LeBoncoin session feature on `bd5b209`; exact initial and final reviewed snapshots (SHA-256) and recheck evidence are in [review.md](review.md). Final snapshot: `scripts/capture-leboncoin-session.mjs` `1651127e…523b`, `scripts/capture-leboncoin-session.test.mjs` `257d2176…664`; all other reviewed files unchanged from the initial freeze.

Reviewer: `/root/leboncoin_session_reviewer` (initial); recheck by a separate independent reviewer agent. Adjudicator: `/root/leboncoin_session_adjudicator`, distinct from both implementation agents and the reviewer. This role changes only this decision record.

## Findings and decisions

### REV-LBC-001: Capture rejects an uppercase listing path accepted by the application

- Type: demonstrated requirement mismatch.
- Evidence: `scripts/capture-leboncoin-session.mjs`, `parseListingURL`, initially matches decoded paths with a case-sensitive `/^\/ad\//` expression, whereas `internal/leboncoin/collector.go` uses `(?i)^/ad/`. Thus `https://www.leboncoin.fr/AD/Voitures/3245888872` is accepted and canonicalized by Go but rejected by capture. FR-LBC-CAP-001 and TS-LBC-CAP-001 require the helper to match existing supported listing rules.
- Impact: an operator pasting this valid spelling cannot begin manual verification. The failure occurs before browsing or writing; it does not disclose session material, overwrite an export, corrupt prices, or affect existing collection. Lowercase `/ad/` is a workaround. Ordinary copied listing URLs make the affected case relatively uncommon, but the mismatch is concrete.
- Criticality: **noncritical**, because the consequence is a bounded input-compatibility failure with a straightforward workaround, rather than loss of data, compromised session protection, or failure of all supported captures. This classification does not waive the specified compatibility requirement.
- Disposition: **fix**.
- Reason: make the helper path expression case-insensitive and add a regression assertion for the uppercase spelling and its lowercase canonical result. This is a narrow correction within the approved parser scope, inexpensive to verify, and preserves the Go/API behavior. Deferral would leave an explicit acceptance requirement unmet without a compensating benefit.
- Specification decision: none required; the existing requirement determines the result. No API change or refactoring is authorized or needed.
- Owner/action: capture developer corrects the parser and tests; independent reviewer rechecks the correction and targeted Node tests.
- Resolution: **pending correction and independent recheck**. A fix decision alone is not evidence of resolution.
- Remaining risk/follow-up: retain paired URL examples to catch future parser drift. No deferred work is approved by this decision.

#### REV-LBC-001 outcome (appended 2026-10-03, re-adjudication)

- Correction: `scripts/capture-leboncoin-session.mjs` `parseListingURL` now matches decoded paths with `/^\/ad\/([a-zA-Z0-9_-]+)\/([0-9]+),?$/i`, equivalent to Go's `(?i)^/ad/([a-z0-9_-]+)/([0-9]+),?$` (`internal/leboncoin/collector.go:21`); the category is still lowercased in the canonical URL. The raw authority/path pre-check, control/backslash rejection and decode-before-match safeguards against WHATWG normalization broadening are unchanged. `scripts/capture-leboncoin-session.test.mjs` now asserts that `https://www.leboncoin.fr/AD/Voitures/3245888872` canonicalizes to `https://www.leboncoin.fr/ad/voitures/3245888872`. Developer statement: [implementation-capture.md](implementation-capture.md), "Accepted review correction".
- Independent recheck (reviewer, [review.md](review.md) "Recheck (2026-10-03)"): the exported Node helper and the existing Go `ParseURL` produced identical results on 12 paired inputs (6 accepted to the same canonical URL, 6 rejected, including `/ADX/`, userinfo, `http`, dot-segment and non-ASCII category cases); 13/13 Node tests pass; `go test -race ./...`, `go vet ./...` and `git diff --check` pass; only the two capture script files changed since the initial freeze.
- Adjudicator verification (own execution): file hashes of both changed files match the reviewer's final snapshot (`1651127e…523b`, `257d2176…664`); inspected the parser and the new assertion; ran the exported `parseListingURL` on `/AD/Voitures/3245888872` (canonical lowercase result), `/ADX/voitures/1` (rejected) and `/AD/../ad/voitures/3245888872` (rejected); ran `/tmp/pricefollower-node22/node_modules/node/bin/node --test scripts/capture-leboncoin-session.test.mjs`: 13 pass, 0 fail.
- Resolution: **resolved — fixed and independently verified**. Classification remains noncritical. The fix stays within the approved parser scope; no API, specification or refactoring change was introduced.
- Remaining risk/follow-up: Node and Go parsers remain separately maintained expressions, so future drift is possible; the paired uppercase assertion and the reviewer's parity table mitigate this. No deferred work.

### REV-LBC-OBS-001: Host-only `leboncoin.fr` cookie domain is accepted by the import format but never applies to the canonical request (observation, not a finding)

- Type: question raised by the recheck reviewer; explicitly not reported as a finding.
- Evidence: `internal/leboncoin/session.go` `validCookie` accepts the domain strings `leboncoin.fr`, `.leboncoin.fr`, `www.leboncoin.fr`, `.www.leboncoin.fr`, exactly the set allowed by the capture technical design (TS for the shared format, `doc/specifications/leboncoin-session-capture/technical.md`: "a leading dot identifies domain scope, otherwise host-only scope. Preserve the browser's scope"). A host-only `leboncoin.fr` cookie therefore applies only to the apex host, never to the canonical `www.leboncoin.fr` listing request, so the collector would send no cookie and fail with the generic fixed session diagnostic. The capture helper's `sessionExport` computes `domainApplies` against the canonical listing host and refuses such a cookie (verified by inspection of `scripts/capture-leboncoin-session.mjs`, `sessionExport`), so the supported workflow cannot produce this import.
- Impact: reachable only by a hand-crafted or third-party import; the result is a safe, diagnosable collection failure, with no cookie sent out of scope, no secret disclosure and no data corruption.
- Criticality: **noncritical**; not a defect.
- Disposition: **reject** (no change).
- Reason: the behavior conforms to the approved format and the scope-preservation rule; rejecting the domain at import or rewriting its scope would either contradict the specified allowed-domain set or broaden a host-only cookie's scope, which the specification forbids. Narrowing the allowed set would be a product/specification change, not a correctness fix, and has no demonstrated user need.
- Specification decision: none required. If the user later wants the import to reject inapplicable host-only apex cookies with a specific diagnostic, that is a specification change routed through the specifiers.
- Resolution: rejected as non-finding; no code change.
- Follow-up: none.

### QA-LBC-F01: Capture helper cannot start with installed Google Chrome 149 (QA failure, appended 2026-10-03)

- Source: [qa.md](qa.md), failure QA-LBC-F01, case QA-CAP-018 (also blocks QA-LIVE-001). Entered into this decision process as a new finding under WORKFLOW section 6.
- Type: demonstrated defect (requirement mismatch at runtime), not unspecified behavior or a suggestion.
- Evidence:
  - QA: with an unmodified `--browser /usr/bin/google-chrome` (Chrome 149.0.7827.53) the helper failed in 4/4 runs after about 3 s with "Browser debugging port collision; retry capture." and exit 1, before the listing was visited; 1/1 with `/opt/google/chrome/chrome`, the path shown in `doc/leboncoin-session.md`. Cleanup and preservation of the prior export were correct.
  - Code: `openBrowser` in `scripts/capture-leboncoin-session.mjs` passes `about:blank#pricefollower-<uuid>` as the startup URL and, on the first successful `/json/list` response, fails immediately unless a `page` target has exactly that URL (no retry).
  - Coordinator probe (2026-10-03, this host, Chrome 149.0.7827.53, fresh temporary profile, same flags as `openBrowser`): with start URL `about:blank#pricefollower-test`, `/json/list` reports the only page as `chrome://newtab/` (the marker is ignored), reproducing the failure; with start URL `data:text/plain,pricefollower-<uuid>` the page target URL is exactly that `data:` URL. The QA tester independently observed the same behavior (`about:blank#…` replaced by `chrome://newtab/`; plain `about:blank` and `data:` URLs preserved).
  - Existing unit test "startup uses own visible isolated target…" stubs `fetch` to echo whatever marker was spawned, so it cannot detect a marker the real browser does not preserve; this explains why 13/13 Node tests pass while the real browser fails.
  - Adjudicator note: the code is consistent with TS-LBC-CAP-002's safety intent (prove the endpoint belongs to the spawned child; treat collision as startup failure), but its chosen proof mechanism does not work with the documented browser. The specification does not prescribe the marker form, so the correction is an implementation fix, not a specification change.
- Impact and requirement obligations:
  - Blocks **FR-LBC-CAP-002** (the operator never sees the requested listing in the dedicated browser) and therefore **FR-LBC-CAP-004/005/006 success paths** (no verified listing, no export) for the browser the operator documentation names. TS-LBC-CAP-002's "executed visible-browser QA" validation cannot pass.
  - Likelihood: deterministic for Chrome 149 on this host (5/5 runs, two executable paths); the Playwright bundled Chromium is untested, so it is unknown whether any supported configuration currently works. Lack of evidence is not proof that another browser is safe.
  - Affected users: every operator following the documented capture procedure with installed Google Chrome; without a capture there is no session file, so the collection feature's practical purpose (FR-LBC-COL with a verified session) is also unreachable in the documented workflow.
  - The diagnostic "port collision; retry capture" is misleading: retrying cannot succeed.
  - No secret disclosure, data loss, overwrite of a prior export or leftover process was observed; failure is safe (fail-closed), which is why the issue is a functional blocker rather than a security defect.
- Criticality: **critical**. The primary user flow of the capture subject fails every time with the documented browser, preventing acceptance of FR-LBC-CAP-002 and FR-LBC-CAP-004 and blocking the live QA case QA-LIVE-001. A safe failure mode does not reduce this: the feature delivers no value in its documented configuration, and no supported workaround exists (the QA wrapper used for QA-CAP-020 is a test double, not an operator procedure).
- Disposition: **fix**.
- Required change (narrowest acceptable fix):
  - In `openBrowser`, replace the startup marker `about:blank#pricefollower-${randomUUID()}` with a unique `data:text/plain,pricefollower-${randomUUID()}` URL, used unchanged both for the spawn argument, the `/json/list` identity check and the `context.pages()` lookup. Keep the identity proof exactly as strong as before: a fresh random UUID per invocation, an exact URL match on a `page` target, immediate fail-closed on a non-matching response, no attaching to an unrelated browser, no new flags, no headless/automation options, and no change to cleanup, timeouts or diagnostics elsewhere.
  - Adjust the unit test so it would have caught this defect: assert that the spawned marker matches `^data:text/plain,pricefollower-[0-9a-f-]{36}$` (and is distinct across two invocations), and keep a collision case in which `/json/list` returns a non-matching page (for example `chrome://newtab/`) and startup is rejected with the profile removed.
  - Not accepted as part of this fix without a separate decision: retrying on a non-matching target list, relaxing the exact-match check (for example prefix matching or accepting `chrome://newtab/`), creating the marker through DevTools `/json/new` before identity is proven, or changing the collision diagnostic text beyond what the marker change requires. These would weaken or alter the specified identity/collision behavior and require reviewer and adjudicator analysis first.
  - Optional, only if trivially in scope: none. Documentation does not mention the marker, so no documentation change is required.
- Specification decision: none required; TS-LBC-CAP-002 already requires the fresh endpoint to be proven as the child's and collisions to fail safely, without prescribing the marker form. No API change (the [HTTP API](../../../API_SPECIFICATION.md) is unaffected) and no refactoring is authorized or needed.
- Owner/action: capture developer implements the marker change and test adjustment; an independent reviewer rechecks the diff (confirming the identity proof is unchanged in strength and nothing else changed); the QA tester retests.
- Validation required before this finding is marked resolved:
  1. `node --test scripts/capture-leboncoin-session.test.mjs` passes, including the new marker-format and collision assertions.
  2. QA retest of **QA-CAP-018** with unmodified `/usr/bin/google-chrome` (and `/opt/google/chrome/chrome`): the helper reaches the waiting stage, then times out with "Capture timed out…", nonzero exit, cleanup, and prior export preserved, without the QA wrapper.
  3. Live **QA-LIVE-001** (coordinator/user, visible Chrome, human solves the challenge): a private 0600 export containing only the applicable `datadome` cookie, with no cookie value in output or evidence.
  4. Reviewer recheck of the final diff and hashes recorded in [review.md](review.md); any additional change reopens review.
- Resolution: **open — fix decided; pending correction, independent recheck and QA retest**. A fix decision alone is not evidence of resolution.
- Remaining risk/follow-up: Chrome may change its handling of `data:` startup URLs in future versions, and the Playwright bundled Chromium remains untested; QA-LIVE-001 and future capture renewals will expose such a regression with a fail-closed outcome. If a later browser rejects `data:` startup URLs, a new finding enters this process. The misleading wording of the collision diagnostic is noted but not separately required to change, because after the fix it again only fires on a genuine identity mismatch.

#### QA-LBC-F01 outcome (appended 2026-10-03, re-adjudication)

- Correction (developer, [implementation-capture.md](implementation-capture.md) "QA-LBC-F01 correction"): the one-line change in `openBrowser` replaces the marker with `` `data:text/plain,pricefollower-${randomUUID()}` ``. The test records the marker from all five startup invocations. It asserts that each marker matches `^data:text/plain,pricefollower-[0-9a-f-]{36}$` and that all five are distinct. Its collision stub now returns a `chrome://newtab/` page, and the collision case still requires rejection, profile removal and child termination. Developer RED: 12 passed, 1 failed, with the format assertion failing on the old `about:blank#…` marker. GREEN: 13 passed, 0 failed.
- Independent reviewer recheck ([review.md](review.md) "Recheck 2 (QA-LBC-F01)"): the reviewer reconstructed the previous files by reversing the claimed edits and reproduced the earlier hashes exactly. This proves that the script diff is the single marker line and that the test diff is only the marker/collision assertions. One `marker` constant is still used for the spawn argument, the exact `page` match in `/json/list` and the `context.pages()` lookup. The identity proof keeps the same strength: a fresh UUID, an exact match and an immediate fail-closed error with no retry. Nothing was relaxed: no prefix matching, no `/json/new` and no new flags. The other 10 reviewed files are byte-identical and `API_SPECIFICATION.md` is unchanged. Node tests: 13/13. `git diff --check` passes. No new findings. The reviewer noted one observation: the collision test asserts a generic rejection rather than the exact message. I accept this without change because the decision did not require a message-level assertion and the code path reaches that `fail(...)` directly.
- Adjudicator verification (own execution, 2026-10-03):
  - `sha256sum` of all 12 reviewed files matches the reviewer's final snapshot exactly. The script is `8ec4e0fd…cfc01` and the test is `0b5e6c47…16b3`.
  - Line 165 contains the `data:text/plain` marker. The test contains the format, uniqueness and `chrome://newtab/` assertions.
  - `/tmp/pricefollower-node22/node_modules/node/bin/node --test scripts/capture-leboncoin-session.test.mjs`: 13 pass, 0 fail.
  - No `pricefollower-capture-*` profiles remain under `/tmp`.
- Validation conditions set by this decision, checked one by one:
  1. *Node tests pass, including the new marker-format and collision assertions*: **met**. The developer recorded RED/GREEN, the reviewer ran 13/13, and the adjudicator ran 13/13.
  2. *Retest QA-CAP-018 with unmodified Chrome on both paths, without the wrapper*: **met**. [qa.md](qa.md) "Retest and live run (2026-10-03)" covers it:
     - RT-CAP-018a/b ran `/usr/bin/google-chrome` and RT-CAP-018c/d ran `/opt/google/chrome/chrome`, with no wrapper or test double.
     - Each run reached the waiting stage and then failed with "Capture timed out…", exit 1.
     - The prior export was byte-identical (sha256 checked) or no output was written. No profiles or processes were left.
     - RT-CAP-019 additionally shows Ctrl+C in the waiting stage with unmodified Chrome (exit 130, clean).
     - The developer's smoke runs agree.
  3. *Live QA-LIVE-001 with a private 0600 export containing only the applicable `datadome` cookie, and no value in output or evidence*: **met, with the gap recorded in QA-DEF-002**.
     - RT-LIVE-001 ran live with `/usr/bin/google-chrome` on the listing `https://www.leboncoin.fr/ad/voitures/3245888872`. It exited 0. The export is mode 0600, with only the version/`capturedAt`/cookie structure. The cookie is `datadome`, domain `.leboncoin.fr`, secure. The value was absent from stdout and stderr and was not recorded.
     - Further evidence that the export is usable: RT-LIVE-002a/c collected live successfully with the captured session (HTTP 200, price 26.900,00 €). Rotation was persisted to a 0600 sidecar, and no value appeared in logs or in 7 API responses. RT-LIVE-002b without a session returned 403 / `request_error`, which shows the session caused the success.
     - No challenge appeared in that run.
  4. *Reviewer recheck of the final diff and hashes recorded in review.md; any additional change reopens review*: **met**. The hashes I recomputed equal the recorded final snapshot, so no change occurred after the recheck.
- Resolution: **resolved: fixed and independently verified**. The classification stays **critical** as an analysis of the original defect; it no longer blocks. No specification, API or refactoring change was introduced.
- Remaining risk/follow-up:
  - The result is point-in-time, for Chrome 149 on this desktop. A future browser that alters `data:` startup URLs would fail closed at startup and would enter this process as a new finding.
  - Playwright's bundled Chromium remains untested. It is not the documented browser, so this is not a release condition.
  - The collision diagnostic again fires only on a genuine identity mismatch.

### QA-DEF-001: Raspberry Pi portability case QA-LIVE-003 not executed (QA coverage gap, appended 2026-10-03)

- Source: [qa.md](qa.md), QA-LIVE-003, which is blocked because no Raspberry Pi target is available. This is not a defect. It is unexecuted verification of FR-LBC-CAP-008/009 and FR-LBC-COL-010 at the server-side outcome level.
- Evidence and analysis:
  - FR-LBC-CAP-008 is a documentation obligation: provide secure transfer and installation instructions. The review covered `doc/leboncoin-session.md` (hash unchanged since the reviewed snapshot), and it was not questioned again.
  - FR-LBC-CAP-009 and FR-LBC-COL-010 explicitly state that cross-device/network acceptance is **not guaranteed** and that the operator renews the session when it is rejected. A Pi run therefore could not prove or disprove a promised outcome. It could only observe whether LeBoncoin accepts this session from that network at that time.
  - The Pi-side code path is the same Go collector and session store that were exercised live on the desktop (RT-LIVE-002a/c) and under the fixture QA. Failures there are safe and diagnosable (`request_error` with the fixed session diagnostic, price kept, never `unavailable`), as shown by QA-COL-002 and the exploratory runs.
  - As supplementary build evidence (not an interface test), the adjudicator cross-compiled the backend for the deployment target: `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build ./cmd/pricefollower` succeeded and produced a 32-bit ARM EABI5 static executable. This shows the change introduces no build-portability regression for ARMv6. It does not show runtime acceptance on a Pi.
- Criticality: **noncritical**. The unverified behavior is explicitly non-guaranteed by the specification, a failure is safe and reversible by renewal, and no data integrity or secret-handling risk is specific to the Pi.
- Disposition: **defer**.
- Reason: no device is available, so the case cannot be executed now. Waiting would block a desktop-verified feature without lowering a specified risk. The specification already tells the operator that Pi acceptance must be checked through normal collection after installation. This deferral does not waive any requirement: the documentation obligation is met, and the acceptance outcome was never guaranteed.
- Owner/condition: the coordinator/user executes QA-LIVE-003 when a Raspberry Pi deployment is next available, at the latest on the first real deployment of this feature to the Pi. Steps: follow `doc/leboncoin-session.md` transfer/installation, then trigger a refresh through the existing UI and record the result in a QA addendum. A rejection is an expected, documented outcome rather than a defect. A crash, an exposed secret, loss of a price or a status other than `request_error` would be a new finding.
- User visibility: the coordinator must report this deferral and its rationale to the user. If the user requires Pi verification before commit or release, that overrides this deferral.
- Status: deferred with condition; not blocking.

### QA-DEF-002: Capture after a human solves an interactive challenge not exercised live (QA coverage gap, appended 2026-10-03)

- Source: [qa.md](qa.md) coverage notes. In RT-LIVE-001 LeBoncoin served the listing without a slider or challenge, so live capture through an operator-solved challenge (FR-LBC-CAP-003 with a solved challenge, then FR-LBC-CAP-004) remains unexercised. This is not a defect. It is a verification gap caused by external behavior that QA cannot trigger on demand.
- Evidence and analysis:
  - The relevant logic is the `waitForSession` polling gate, which waits until the requested listing has loaded with a matching ID and the applicable cookie. It does not depend on how the page arrived there.
  - The unit tests "actual polling gate waits through challenge and only exports a matching loaded document" and "requires matching successful active document, never a challenge or rounded ID" cover waiting through a challenge document and exporting only after a matching load. Both are in the 13/13 passing suite.
  - Live runs exercised the remaining stages with unmodified Chrome:
    - the waiting stage until timeout, on a listing that cannot verify (RT-CAP-018a–d);
    - cancellation while waiting (RT-CAP-019);
    - the verified-export success path (RT-LIVE-001).
  - The specified safety property is fail-closed. A challenge page cannot produce an export, and a failed or abandoned challenge times out or cancels with cleanup and preserves the prior export. All of these were shown.
  - The residual uncertainty is whether, after a real human solve, LeBoncoin's post-challenge navigation reaches the listing within the timeout so that the gate recognizes it. If it does not, the outcome is a safe timeout followed by a retry, not an unsafe export.
- Criticality: **noncritical**. The unexercised scenario can at worst produce a safe, clearly reported unsuccessful capture. Every guarded property (no automated solving, no export without a verified listing, privacy, cleanup, preservation) is covered by tests or by executed live cases.
- Disposition: **defer**.
- Reason: the challenge cannot be provoked deterministically or ethically. Trying to provoke DataDome with repeated or suspicious traffic would conflict with the feature's minimal-traffic, no-evasion design. Blocking release on a condition QA cannot create would not reduce risk.
- Owner/condition: the coordinator/user records the outcome in a QA addendum the first time a capture shows a slider or challenge during normal operator use. If the operator solves it and the helper still times out or fails, that enters this process as a new finding.
- User visibility: the coordinator must report this deferral to the user, together with QA-DEF-001.
- Status: deferred with condition; not blocking.

## Release readiness

Updated 2026-10-03 after the QA-LBC-F01 correction, second recheck and retest.

Review-decisions gate: **passes** for the final reviewed snapshot recorded in [review.md](review.md) "Recheck 2 (QA-LBC-F01)", with the script at `8ec4e0fd…cfc01` and the test at `0b5e6c47…16b3`; the other 10 files are unchanged. The adjudicator verified that every hash matches. The review report, the QA report and this record agree.
- REV-LBC-001: noncritical, fix; resolved and verified.
- REV-LBC-OBS-001: rejected non-finding.
- QA-LBC-F01: critical, fix; **resolved and independently verified**. All four validation conditions are met.
- QA-DEF-001 (Pi portability, QA-LIVE-003) and QA-DEF-002 (live human-solved challenge): noncritical; deferred with stated conditions and owners.
- The reviewer's message-assertion note: an observation, accepted without change.

Critical blockers: **none**. Pending user decisions: none are required by this record. The coordinator must report the two deferrals to the user, who may require either to be executed before commit or release. No API, specification or refactoring change was introduced.

QA: offline/fixture scope passed, apart from QA-LBC-F01, which is now resolved. The retest (RT-CAP-018a–d, RT-CAP-019) passed with unmodified Chrome on both documented paths, live capture (RT-LIVE-001) passed, and live collection (RT-LIVE-002a/c) passed. QA-LIVE-003 is blocked and deferred (QA-DEF-001). Live results are point-in-time for this desktop and network.

Commit-stage condition: these decisions apply only to the hashes above. Any change to the reviewed files before commit reopens review and these decisions.
