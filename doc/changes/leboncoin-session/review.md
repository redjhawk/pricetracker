# Independent LeBoncoin session review

Status: recheck 2 (QA-LBC-F01) complete; the correction matches the adjudicated fix exactly, and review found no new findings. The review gate passes for the "Recheck 2" final snapshot below. Date: 2026-10-03. QA retest (QA-CAP-018) and live QA-LIVE-001 are still pending. Reviewer (initial): `/root/leboncoin_session_reviewer`. Separate independent reviewer agents did the rechecks. This report does not adjudicate its own findings or claim live QA completion.

Read the reviewer role, AGENTS.md, staged workflow, all four required project skills, both subjects' functional and technical specifications, the unchanged canonical API, and both implementation handoffs. Reviewed the working-tree implementation against base `bd5b209`, frozen by the coordinator after both developers finished.

## Findings

### REV-LBC-001 — Capture rejects an application-supported uppercase ad path

- Subject/requirements: session capture, FR-LBC-CAP-001 and TS-LBC-CAP-001 (match existing Go URL acceptance/canonicalization).
- Location: `scripts/capture-leboncoin-session.mjs:42`, `parseListingURL`; comparison: `internal/leboncoin/collector.go:21`, `listingPathPattern`.
- Provisional severity: low. Ordinary copied lowercase URLs work, but the helper rejects a supported variation before opening the browser. There is no data loss or secret exposure.
- Evidence: the helper's decoded-path expression is case-sensitive for `/ad/`, while Go's existing expression starts with `(?i)` and accepts `/AD/`. Executed the actual exported helper with `https://www.leboncoin.fr/AD/Voitures/3245888872`; it returned `Use a supported HTTPS LeBoncoin listing URL.` Go's existing parser accepts this path and canonicalizes it to `https://www.leboncoin.fr/ad/voitures/3245888872` by inspection of its matching and canonicalization code.
- Expected: application-supported casing variants reach the same canonical capture URL. Actual: the helper rejects the uppercase path prefix, despite the explicit parity requirement.
- Suggested correction: make the helper path matching case-insensitive and cover uppercase `/AD/` in its URL tests. Preserve the existing safeguards against WHATWG normalization broadening.
- Resolution: open, for the independent adjudicator and implementing agent. No code edited by the reviewer.

## Reviewed behavior and scope

- The opt-in is trimmed configuration, lazy loading, and existing service collector wiring. No API, frontend, schema, scheduler or dependency change was found. Sessionless construction remains available. Invalid configured imports do not prevent process startup or Amazon collection.
- The import remains immutable. Fingerprint-bound sidecars, serialized cancellable attempts, renewal selection, effective expiry and memory retention after failed persistence follow the stated design. State replacement is private and atomic; reads validate open descriptors and reject blocking nonregular input.
- Session request and redirect checks constrain cookies to exact permitted HTTPS origins. Response handling distinguishes staged replacement from explicit revocation. Successful matching listing validation precedes inactive/missing-price branches; deletion is finalized on error paths. The inspected tests cover restart, stale in-flight renewal, scope, ambiguous cookies, deletion on errors, unsafe input, save failure and retained successful observations.
- The helper uses a dedicated visible browser, private profile and unique startup target to identify the spawned browser endpoint. It performs no CAPTCHA interaction. Export requires the requested loaded successful active listing plus one applicable cookie, and writes only the specified metadata. Cancellation, private atomic replacement, browser/profile cleanup, fixed diagnostics and signal tests were inspected.
- The final operator guide includes protected remote staging, an interactive SSH terminal for sudo, private atomic installation, one-time systemd configuration, hot renewal, damaged-sidecar repair, and explicit cross-device limitations. It uses the existing deployment scripts. Shell commands were reviewed, not executed on a remote device.
- No unrelated refactoring or additional unspecified product behavior was identified. The finding above is a concrete specification mismatch; this is not a general cleanup request.

## Evidence and limitations

Reviewer executed the focused uppercase-path reproduction and `git diff --check` (passed), and inspected the actual implementation/tests. The full Go/race suites, ARMv6 build and thirteen helper tests are developer/coordinator evidence documented in the implementation handoffs; this report does not relabel those executions as reviewer executions. Live visible-browser success, actual interface exploration and Raspberry Pi portability remain separate QA/operator checks.

## Exact initial reviewed snapshot

SHA-256 of each source/test/operator file at the review freeze (before any finding correction):

```text
6658fcc3c063b2cee24106e41c8ba1338539ea8b9e2e319a1dd446ad173dd1a7  README.md
a23f5f685abf8b50debc7fd1fb3573a12ab5fb5d6c8db96ef80535e97dc7b381  config/config.go
4f303274f65d90b6d0a01b7329e62dd1477744bc533d1c5d2fb948c59b93d93a  config/config_test.go
0c81edcf16c47395f45eb07805da07cb50bb916557f9a75bf866fba475d458c9  internal/service/service.go
bd9235f80d7b1bc8ef393b4df9e0ff572c4e2242e588c1dcf3f2ba963c6cfec7  internal/service/session_test.go
234b9aaab3c9f31673e04ed41b57aec1001c1d7e9b72236f887a9cfd5ddb06a2  internal/leboncoin/collector.go
01e42bf1549e1928c6951961d4daddfb4f0ce13caafaf7c0fe746704c66bb0f6  internal/leboncoin/session.go
17ad966df5efdedbc3b3299979d78903ab6e12729c7d777eb6514719f503a4f2  internal/leboncoin/session_test.go
005a6893e1b1d0b05e81f9e50a99ac0bb6d246a59270aad65a1d0f5743d2c7f2  scripts/capture-leboncoin-session.mjs
85de4d57eeb92c9eb80d1655fe2add6e19959fdc2c1f8c3bc6270798368d323c  scripts/capture-leboncoin-session.test.mjs
6a3a06f3f541254f26aecf3a1b2177cb2564f6eb043eccc44490237b29afde55  doc/leboncoin-session.md
c122f7e64748963889866b7347ae8609d4b3a0f580aaf9166efd30daa4b748fa  API_SPECIFICATION.md (unchanged)
```

Any correction must be independently re-reviewed and appended here with its final file hashes before the review gate passes.

## Recheck (2026-10-03)

Recheck reviewer: an independent reviewer agent distinct from both implementation agents and from the adjudicator. Base: `bd5b209` plus the uncommitted working tree. The untracked `.tmp-session-qa/` directory is a leftover QA harness and was excluded. No code or other documentation was edited; only this file was appended.

### REV-LBC-001 outcome: resolved (verified)

- Fix inspected: `scripts/capture-leboncoin-session.mjs:41` now uses `/^\/ad\/([a-zA-Z0-9_-]+)\/([0-9]+),?$/i`, equivalent to Go's `(?i)^/ad/([a-z0-9_-]+)/([0-9]+),?$` (`internal/leboncoin/collector.go:21`). The raw authority/path pre-check (line 37), the control/backslash rejection, and the decode-before-match safeguards against WHATWG normalization broadening are unchanged. Category is still lowercased in the canonical URL.
- Regression assertion present: `scripts/capture-leboncoin-session.test.mjs:19` asserts `/AD/Voitures/3245888872` canonicalizes to the shared lowercase `listing` value.
- Parity evidence: executed the exported `parseListingURL` (Node 22) and the existing Go `ParseURL` (via a scratchpad test file injected with `go test -overlay`, never written to the repository) on the same 12 inputs. All 12 results are identical:

| Input | Node helper | Go `ParseURL` |
|---|---|---|
| `https://www.leboncoin.fr/AD/Voitures/3245888872` | `https://www.leboncoin.fr/ad/voitures/3245888872` | same |
| `https://www.leboncoin.fr/ad/voitures/3245888872` | same canonical | same |
| `https://WWW.LEBONCOIN.FR/Ad/VOITURES/3245888872,` | same canonical | same |
| `HTTPS://leboncoin.fr:443/aD/voitures/3245888872?x=1#f` | same canonical | same |
| `https://www.leboncoin.fr/%41D/voitures/3245888872` | same canonical | same |
| `https://www.leboncoin.fr./AD/voitures/3245888872` | same canonical | same |
| `https://www.leboncoin.fr/AD/voitures/` | reject | reject |
| `https://www.leboncoin.fr/ADX/voitures/3245888872` | reject | reject |
| `https://user@www.leboncoin.fr/AD/voitures/3245888872` | reject | reject |
| `http://www.leboncoin.fr/AD/voitures/3245888872` | reject | reject |
| `https://www.leboncoin.fr/AD/../ad/voitures/3245888872` | reject | reject |
| `https://www.leboncoin.fr/ad/voit%C3%BCres/3245888872` | reject | reject |

### Changes since the initial freeze

`sha256sum` against the frozen list: only `scripts/capture-leboncoin-session.mjs` and `scripts/capture-leboncoin-session.test.mjs` changed. All Go sources/tests, `config/*`, `README.md`, `doc/leboncoin-session.md` and `API_SPECIFICATION.md` are byte-identical to the freeze. The frozen script contents were not retained (untracked files), so a byte diff of the two changed files is not possible; the recheck re-inspected the parser/test regions in full and found only the case-insensitive flag and the added assertion relevant to parsing, consistent with the developer's statement in `implementation-capture.md`. No other behavior change was observed in the inspected helper code.

### Final correctness pass

Re-inspected the complete `git diff` (`config/config.go`, `internal/service/service.go`, `internal/leboncoin/collector.go`, `README.md`) and the untracked `internal/leboncoin/session.go` in full against TS-LBC-COL-002/003 and the sidecar rules:

- Sessionless path: with an empty `LEBONCOIN_SESSION_FILE`, `c.session` is nil; the request, client, `CheckRedirect`, logging and outcomes are unchanged (the only edit is `collectionError` becoming a method whose nil-session branch is the original body).
- Session path: gate serialization makes all `sessionStore.current` access single-threaded; `finish` is deferred only after a successful `load`. Initial URL is canonicalized with the existing `ParseURL`; redirects strip the copied `Cookie` header and are rejected unless exact HTTPS apex/www, no userinfo, empty/443 port, max four hops. Cookies are attached per hop only via the per-attempt jar after `CheckRedirect`, so a rejected destination never receives the value.
- Cookie scope/expiry: host/domain/path matching follows RFC boundaries; Max-Age precedes Expires with overflow guard; negative Max-Age and past Expires are deletions; deletions require identity match; ambiguous multiple replacements fail safe; expired candidates are never sent. `verified` is set only after a 2xx HTML page with the requested nonzero ID, before inactive/price branches; revocation is committed on error paths.
- Persistence: import is never written; sidecar is fingerprint-bound, written via exclusive 0600 temp file and atomic rename through an `os.Root` handle; unsafe/malformed sidecars fail closed; save failure is logged with a fixed message and in-memory state retained.
- Diagnostics: session-mode errors return/log fixed strings; raw library errors, headers and cookie values are not logged.

Question (not a finding): the shared format and `validCookie` accept a host-only `domain: "leboncoin.fr"`, which can never apply to the canonical `www.leboncoin.fr` request, so such an import would be sent without a cookie and fail with the generic session diagnostic. The capture helper already refuses to export that combination, so it is unreachable through the supported workflow; no change is requested.

### New findings

None. No demonstrated defects or concrete specification mismatches were found beyond REV-LBC-001, which is resolved.

### Commands executed by the recheck reviewer

| Command | Result |
|---|---|
| `sha256sum <12 frozen files>` | 2 changed (both capture script files), 10 unchanged |
| `node <scratchpad>/cmp.mjs <scratchpad>/urls.txt` (Node 22, imports exported `parseListingURL`) | 6 accepted to the canonical URL, 6 rejected (table above) |
| `GOCACHE=<scratchpad>/gocache go test -overlay <scratchpad>/overlay.json -run TestReviewerParity -v ./internal/leboncoin/` | identical results for all 12 inputs |
| `/tmp/pricefollower-node22/node_modules/node/bin/node --test scripts/capture-leboncoin-session.test.mjs` | 13 tests, 13 pass, 0 fail |
| `GOCACHE=<scratchpad>/gocache go test -race ./...` and again with `-count=1` (go1.25.0) | `config`, `internal/leboncoin`, `internal/service` ok; others have no test files |
| `go vet ./...` | passed |
| `git diff --check` | passed |

Limitations: live visible-browser capture, interface QA and Raspberry Pi portability remain QA/operator checks and were not executed here.

### Final reviewed snapshot (SHA-256)

```text
6658fcc3c063b2cee24106e41c8ba1338539ea8b9e2e319a1dd446ad173dd1a7  README.md
a23f5f685abf8b50debc7fd1fb3573a12ab5fb5d6c8db96ef80535e97dc7b381  config/config.go
4f303274f65d90b6d0a01b7329e62dd1477744bc533d1c5d2fb948c59b93d93a  config/config_test.go
0c81edcf16c47395f45eb07805da07cb50bb916557f9a75bf866fba475d458c9  internal/service/service.go
bd9235f80d7b1bc8ef393b4df9e0ff572c4e2242e588c1dcf3f2ba963c6cfec7  internal/service/session_test.go
234b9aaab3c9f31673e04ed41b57aec1001c1d7e9b72236f887a9cfd5ddb06a2  internal/leboncoin/collector.go
01e42bf1549e1928c6951961d4daddfb4f0ce13caafaf7c0fe746704c66bb0f6  internal/leboncoin/session.go
17ad966df5efdedbc3b3299979d78903ab6e12729c7d777eb6514719f503a4f2  internal/leboncoin/session_test.go
1651127e86389cab2546a61e20b3644db1f54eabfd00843223b0b1dbf821523b  scripts/capture-leboncoin-session.mjs
257d21767e99f0b9b2700106ec7feebdacfd259fe9d503ec8bc28b76ede58664  scripts/capture-leboncoin-session.test.mjs
6a3a06f3f541254f26aecf3a1b2177cb2564f6eb043eccc44490237b29afde55  doc/leboncoin-session.md
c122f7e64748963889866b7347ae8609d4b3a0f580aaf9166efd30daa4b748fa  API_SPECIFICATION.md (unchanged)
```

## Recheck 2 (QA-LBC-F01) (2026-10-03)

Recheck reviewer: an independent reviewer agent. It is distinct from the implementation agents and the adjudicator, and it did not implement the correction. Inputs read: the QA-LBC-F01 decision in [decisions.md](decisions.md), the finding in [qa.md](qa.md), and the "QA-LBC-F01 correction" section of [implementation-capture.md](implementation-capture.md). Base: `bd5b209` plus the uncommitted working tree. The untracked `.tmp-session-qa/` directory (empty QA leftover) was excluded. No live browser was launched, because QA owns that. Only this file was edited.

### Outcome: correction matches the adjudicated fix; no new findings

**Exact diff against the previous final snapshot.** The earlier recheck could not byte-diff the untracked scripts. This time, both previous files were reconstructed in the scratchpad from the current files by reversing the claimed edits. Each reconstruction reproduced the previous recorded hash exactly, which proves that the claimed edits below are the complete change:

- Script: reverting line 165 to `about:blank#pricefollower-${randomUUID()}` gives `1651127e…523b`, the previous final hash. The whole script diff is therefore this single line:
  ```diff
  -    const marker = `about:blank#pricefollower-${randomUUID()}`;
  +    const marker = `data:text/plain,pricefollower-${randomUUID()}`;
  ```
- Test: removing the four added `markers` lines and restoring the collision stub URL `'about:blank'` gives `257d2176…664`, the previous final hash. The whole test diff is therefore:
  - `const markers = [];` and `markers.push(marker);` in the spawn stub (records every spawned marker);
  - the collision stub's `/json/list` page URL changes from `'about:blank'` to `'chrome://newtab/'`;
  - the test now asserts `/^data:text\/plain,pricefollower-[0-9a-f-]{36}$/` for every marker, and that all markers are distinct (`new Set(markers).size === markers.length`; 5 invocations).

**Same variable at all three points.** The single `marker` constant is used unchanged in three places: the spawn argument (line 166), the exact `target.type === 'page' && target.url === marker` check on `/json/list` (line 178), and the `context.pages()` lookup `candidate.url() === marker` (line 188).

**Identity proof unchanged in strength.** The proof still uses:

- a fresh `randomUUID()` per invocation;
- an exact match on a `page` target;
- an immediate fail-closed `CaptureError` on the first non-matching response, with no retry;
- the unchanged collision diagnostic.

Nothing was added: no prefix matching, no `/json/new`, no new flags, and no headless or automation options. The cleanup, 20 s startup bound, timeouts and other diagnostics are byte-identical.

**Test coverage meets the decision.** The decision required checks for marker format, uniqueness, and `chrome://newtab/` collision rejection with cleanup:

- Format and uniqueness are asserted over all five invocations.
- The collision case returns a `chrome://newtab/` page. It asserts rejection (`assert.rejects`), removal of the profile (`ENOENT`), and that the child was killed.
- The developer's RED run (12/1, failing on the format assertion against `about:blank#…`) shows the new assertion detects the defect. That RED run is developer evidence; this reviewer did not re-execute it.

**Other files unchanged.** All other reviewed files are byte-identical to the previous final snapshot, and `API_SPECIFICATION.md` is unchanged.

**Non-blocking observation (not a finding).** The collision case asserts a generic rejection rather than the specific "Browser debugging port collision" message. Code inspection shows the path reaches that `fail(...)` directly. The decision did not require message-level assertion.

**Note on scope of evidence.** It is runtime behavior that Chrome 149 preserves the `data:text/plain,…` startup URL in `/json/list`. The coordinator probe, the QA observation and the developer's smoke runs support it; this reviewer did not execute it. QA-CAP-018 retest and QA-LIVE-001 remain required by the decision.

### Commands executed by the recheck reviewer

| Command | Result |
|---|---|
| `sha256sum <12 reviewed files>` | the 2 capture script files changed; the other 10 are identical to the previous final snapshot |
| Reverse-apply the claimed script edit into the scratchpad, then `sha256sum` and `diff` | `1651127e…523b` (exact previous hash); one-line diff |
| Reverse-apply the claimed test edits into the scratchpad, then `sha256sum` and `diff` | `257d2176…664` (exact previous hash); 4 added lines plus 1 changed stub line |
| `/tmp/pricefollower-node22/node_modules/node/bin/node --test scripts/capture-leboncoin-session.test.mjs` | 13 tests, 13 pass, 0 fail |
| `git diff --check` | passed (exit 0) |

### Final reviewed snapshot (SHA-256)

```text
6658fcc3c063b2cee24106e41c8ba1338539ea8b9e2e319a1dd446ad173dd1a7  README.md
a23f5f685abf8b50debc7fd1fb3573a12ab5fb5d6c8db96ef80535e97dc7b381  config/config.go
4f303274f65d90b6d0a01b7329e62dd1477744bc533d1c5d2fb948c59b93d93a  config/config_test.go
0c81edcf16c47395f45eb07805da07cb50bb916557f9a75bf866fba475d458c9  internal/service/service.go
bd9235f80d7b1bc8ef393b4df9e0ff572c4e2242e588c1dcf3f2ba963c6cfec7  internal/service/session_test.go
234b9aaab3c9f31673e04ed41b57aec1001c1d7e9b72236f887a9cfd5ddb06a2  internal/leboncoin/collector.go
01e42bf1549e1928c6951961d4daddfb4f0ce13caafaf7c0fe746704c66bb0f6  internal/leboncoin/session.go
17ad966df5efdedbc3b3299979d78903ab6e12729c7d777eb6514719f503a4f2  internal/leboncoin/session_test.go
8ec4e0fd493be6c52a698cf2c6bf770eeb2cfacfd7c01ccb808a184c253cfc01  scripts/capture-leboncoin-session.mjs
0b5e6c474e3b0b39808804151aa66b0f3f6e8d5e5866dff552d0df63338716b3  scripts/capture-leboncoin-session.test.mjs
6a3a06f3f541254f26aecf3a1b2177cb2564f6eb043eccc44490237b29afde55  doc/leboncoin-session.md
c122f7e64748963889866b7347ae8609d4b3a0f580aaf9166efd30daa4b748fa  API_SPECIFICATION.md (unchanged)
```

Any further change to these files reopens review.
