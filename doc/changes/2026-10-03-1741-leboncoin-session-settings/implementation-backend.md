# Go backend implementation evidence

Status: implemented; native, race and repeated suites pass; ready for independent review. Date: 2026-10-03. Agent: Go backend developer (developer role, separate agent invocation).

Inputs read before coding: `AGENTS.md`, [developer role](../../../.agents/roles/developer.md), [workflow](../../workflow/WORKFLOW.md) section 4, all four project skills (`carbon-frontend`, `frontend-architecture`, `go-sqlite-backend`, `go-backend-architecture`), the [change index](index.md) (API contract approved 2026-10-03, expiry clarification “Yes, as proposed”), [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) section “LeBoncoin session settings (approved 2026-10-03)”, and the specifications [settings](../../specifications/leboncoin-session-settings/technical.md), [collection revision 2](../../specifications/leboncoin-session-collection/technical.md) and [file removal](../../specifications/leboncoin-session-file-removal/technical.md) (backend parts).

Ownership: Go code and Go tests only (`cmd/**`, `config/**`, `internal/**`). No frontend, script, operator-guide, deprecation, README or DEPLOYMENT file was touched. No new module dependency. No refactoring beyond the specified removal/replacement. Nothing committed or staged.

## Requirement / technical ID to change mapping

| Technical ID | Functional IDs | Change |
| --- | --- | --- |
| TS-LBC-SET-001 | SET-012, SET-016, SET-017 | `internal/store/store.go`: additive idempotent `CREATE TABLE IF NOT EXISTS leboncoin_session` (single row `id = 1`, CHECK constraints exactly as specified) + `INSERT OR IGNORE` of row 1 in `createSchema`. `internal/model/model.go`: `LeboncoinSession`, `LeboncoinSessionAttempt` (API-shaped JSON). |
| TS-LBC-SET-001 / 005 | SET-014, SET-015 | `Store.LeboncoinSession` (status derived at read time: `none` / `revoked` / `expired` when `expires_at <= now` / `active`; `lastAttempt` from the row), `Store.SaveLeboncoinSession` (one transaction; `ErrSessionChanged` on revision mismatch; clear of an empty session returns the row unchanged; otherwise sets value, clears expiry/revocation/attempt, `revision + 1`, `updated_at`). |
| TS-LBC-COL-008 | COL-013, 014, 015, 016, 017 | `Store.FinishLeboncoinSessionAttempt(ctx, startRevision, LeboncoinSessionOutcome, now) (applied, error)`: one transaction; no write if revision changed; revocation sets `revoked_at`, `revision + 1`; renewal with a different value sets value/expiry, clears `revoked_at`, `revision + 1`; same-value renewal updates only `expires_at` (no revision change); always records `last_attempt_at/outcome`; `UPDATE … WHERE id = 1 AND revision = startRevision`. Store-level `LeboncoinSessionOutcome` struct (same fields as `leboncoin.SessionOutcome`, converted in the service) so the store does not import `leboncoin`. |
| TS-LBC-SET-002 | SET-005, SET-006, SET-008 | `internal/leboncoin/session.go`: `ParseSessionInput` + `SessionInputError`, following rules 1–6 in order (8192-byte input limit, trim, optional case-insensitive `Cookie:`/`Set-Cookie:` prefix, raw value when no `=`/`;`, cookie-string extraction of exact-name `datadome` with none/empty/distinct-values errors, value validation 1–4096 bytes of 0x21–0x7E excluding `"` `,` `;` `\`), with the exact specified messages. |
| TS-LBC-SET-003 | SET-002, 004, 007, 008, 010, 013, 017 | `internal/service/service.go`: `Service.LeboncoinSession`, `Service.SaveLeboncoinSession` (parse → `400 INVALID_SESSION`; `ErrSessionChanged` → `409 SESSION_CHANGED` with the specified message; blank → clear; never queues a collection). |
| TS-LBC-SET-004 | SET-002–010, 017, 018 | `internal/httpapi/server.go`: `GET`/`PUT /api/v1/settings/leboncoin-session` before item routes; `{"session": …}` envelope; `MaxBytesReader` 32 768 → `413 REQUEST_TOO_LARGE`; malformed JSON / trailing value → `400 INVALID_JSON`; missing/null `value` or `revision`, `*json.UnmarshalTypeError`, negative revision → `400 INVALID_REQUEST` with the specified message; other methods → `405` with `Allow: GET, PUT`; storage errors via existing `serverError` (500). `Cache-Control: no-store` comes from the existing `/api/` handler. |
| TS-LBC-COL-006 / TS-LBC-RM-001 | COL-011, 012, RM-001, RM-002 | `config/config.go`: `LeboncoinSessionFile` field and `LEBONCOIN_SESSION_FILE` read removed (struct alignment returns to the pre-session layout). `config/config_test.go`: deleted (only tested the removed variable). `internal/leboncoin/collector.go`: `NewCollectorWithSession`, the `session` field, the capacity-one gate and `sessionCollectionError` (“private session files” text) removed. `internal/leboncoin/session.go`: all file/sidecar code removed (`sessionLimit`, `errSession*`, `sessionState`, `sessionStore` load/save/finish, `exactObject`, `utcTimestamp`, `decodeCookie`, `owned`, `privateSessionDirectory`, `readPrivate`, fingerprint). `internal/service/service.go`: `collectors` holds only `"amazon"`; new field `leboncoin leboncoinCollector` (interface with `CollectWithSession`, used for test injection) set to `leboncoin.NewCollector(cfg.UserAgent)`. No code opens, imports or deletes old session files. |
| TS-LBC-COL-007 | COL-011, 013, 015, 020 | `Service.collectLeboncoin`, called from `collectReserved` for `platform == "leboncoin"` before the 31-second request context is created: read error → fixed log, `request_error` “The LeBoncoin session could not be read…”, no request; `none` → `CollectWithSession(…, nil)` (sessionless, no session write); `expired`/`revoked` → fixed log, `request_error` “The saved LeBoncoin session has expired or was revoked…”, no request; `active` → collect with `&Session{Value, ExpiresAt}`, then `FinishLeboncoinSessionAttempt(startRevision)` before `RecordCollection`. Write failure and discarded updates are logged with the specified fixed texts; the collection result is still recorded. Immediate, manual and scheduled attempts all pass through `collectReserved`. |
| TS-LBC-COL-009 | COL-007, 008, 012, 014, 015, 017 | `leboncoin.Session`, `leboncoin.SessionOutcome`, `Collector.CollectWithSession`. Retained per-attempt jar on a local copy of the client, `allowedSessionURL` on the initial URL (canonical listing) and every redirect, `Cookie` header removal on redirect, four-request bound. Amendments: the cookie is sent only as `datadome` to `https://leboncoin.fr` / `https://www.leboncoin.fr` (port 443, no userinfo); a `Set-Cookie: datadome` is considered only if its Domain normalizes to `.leboncoin.fr` and `Path=/` (host-only and other paths ignored); Max-Age before Expires, overflow rejected, malformed known metadata rejected (see Review fixes, REV-SET-004), value validation, distinct-value ambiguity ignored; renewal reported only when the final response is the verified requested listing; matching deletion (Max-Age ≤ 0 or past Expires) on any status reports `Revoked` unless a later verified listing in the same chain supplied a replacement; outcome `accepted` if verified, `rejected` on final 403 or a body containing `captcha-delivery.com` without verification, otherwise `failed` (404/410 → `failed`, item result stays `unavailable`). An expired candidate is never sent (retained check). |
| TS-LBC-COL-010 / TS-LBC-SET-007 | COL-016, 018, 019, SET-018 | Session-assisted failures return “LeBoncoin rejected the saved session. Capture a new session and save it in Settings.” (rejected) or the sessionless text “LeBoncoin could not be reached for a price check.” (other). Session-assisted logs contain listing ID and status only: no URL, cookie, `Set-Cookie`, raw library error or store error. The value leaves the server only through the settings endpoint. |
| TS-LBC-COL-011 / TS-LBC-RM-006 | COL-007, 008, RM-007 | Item endpoints, outcomes, schedule, Amazon collector and pure-Go ARMv6 build unchanged (verified below). |

## Changed files

- Modified: `config/config.go`, `internal/httpapi/server.go`, `internal/leboncoin/collector.go`, `internal/leboncoin/session.go` (rewritten), `internal/leboncoin/session_test.go` (rewritten), `internal/model/model.go`, `internal/service/service.go`, `internal/service/session_test.go` (rewritten), `internal/store/store.go`.
- Added: `internal/store/session_test.go`, `internal/httpapi/settings_test.go`.
- Deleted: `config/config_test.go` (working-tree deletion, not staged).

`git diff --stat -- cmd config internal`: 10 tracked files, 1101 insertions, 948 deletions (plus the two new test files, 360 lines).

## Test-first evidence

Tests were written first. To obtain behavioral (not compile) RED evidence, temporary stub files were added (`internal/{leboncoin,store,model,service}/zz_stub.go` with the new types and zero-value methods; `CollectWithSession` delegated to the old sessionless `Collect`, `ParseSessionInput` returned its input unchanged, store/service methods returned zero values, no HTTP route) plus the `leboncoin` field on `Service`. The stubs were deleted before the real implementation.

RED commands (GOCACHE set to the session scratchpad cache):

```text
go test -timeout 60s ./internal/leboncoin/ ./internal/store/ ./internal/httpapi/
go test -timeout 60s ./internal/service/ -skip 'TestOperatorSaveDuringAttemptWins|TestConcurrentRenewalsApplyOnce'
go test -timeout 10s ./internal/service/ -run 'TestOperatorSaveDuringAttemptWins'
```

RED result: all four packages `FAIL`, 25 failing tests, plus a deadlock timeout:

- leboncoin (9): `TestParseSessionInput` (e.g. `"datadome=abc123"` returned unchanged, `"a=1; b=2"` accepted), `TestSessionCookieAttachedToLeboncoinOrigins` (`cookie = ""`), `TestSessionOriginAndRedirectIsolation` (outcome empty; transport saw redirects to `https://user@…` and `:444`), `TestSessionOutcomeClassificationAndVerifiedRenewal`, `TestSessionNetworkErrorIsFailedAndUndisclosed`, `TestSessionResponseCookieMetadata`, `TestSessionDeletionIsRevocationOnAnyStatus`, `TestSessionDeletionSurvivesReadError`, `TestSessionRedirectStagingAndDeletion`.
- store (5): fresh database, migration/persistence, save/conflict/clear, status derivation, conditional finish.
- httpapi (4): every settings route test (route absent → 404).
- service (7): validation/conflict, renewal and restart, rejection, revocation, expiry, write failure, read failure.
- `TestOperatorSaveDuringAttemptWins` panicked with “test timed out after 10s” (the stub never invoked the injected collector, so the test waited forever). `TestConcurrentRenewalsApplyOnce` waits on the injected collector in the same way and was skipped in the bounded run; a first unbounded `go test ./...` run under the stubs did not finish within 120 s and was stopped.

Tests passing under the stub were regression guards of unchanged behavior: `TestCollectWithoutSessionSendsNoCookie`, `TestSessionExpiredValueNeverSent`, `TestSessionlessAndAmazonCollectionSendNoCookie`.

GREEN: after implementing, each package passed individually (`leboncoin`, `store`, `service`, `httpapi`), then the full suite below.

## Verification (actual commands and results)

All with `GOCACHE=/tmp/claude-1000/-home-redjhawk-src-pricefollower/db657c8a-daa0-468c-9f0c-59ac91b7b822/scratchpad/gocache`, Go 1.25.0 linux/amd64:

| Command | Result |
| --- | --- |
| `gofmt -l cmd config internal` | no output |
| `go vet ./...` | no output, exit 0 |
| `go test ./...` | `ok` httpapi, leboncoin, service, store; other packages have no test files |
| `go test -race ./...` | `ok` httpapi, leboncoin, service, store |
| `go test -count=5 ./internal/...` | `ok` for all four tested packages (flakiness check of the interleaving tests) |
| `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -o <scratchpad>/pf ./cmd/pricefollower` | success; `file`: ELF 32-bit LSB executable, ARM, EABI5, statically linked |
| `git grep -n -E 'LEBONCOIN_SESSION_FILE\|LeboncoinSessionFile\|NewCollectorWithSession\|session\.json\|state\.json\|sessionCollectionError\|private session' -- cmd config internal` | no match |
| Startup smoke (native build, `HOST=127.0.0.1 PORT=38471`, fresh scratchpad data directory) with `LEBONCOIN_SESSION_FILE` unset, set to an existing file and set to a missing path | identical: `GET` → `{"session":{"value":null,"revision":0,"updatedAt":null,"status":"none","expiresAt":null,"revokedAt":null,"lastAttempt":null}}`; `PUT {"value":"datadome=synthetic-smoke","revision":0}` → `200`, `value` `synthetic-smoke`, `revision` 1, `status` `active`; server log contained no `synthetic` string. Data directory removed afterwards. |

Covered test cases (synthetic values, injected `http.RoundTripper` or fake collector, temporary SQLite; no live request): every accepted/rejected parser example of SET-005/006 including the 4096/8192 limits; sessionless and Amazon requests carry no cookie even with a saved session; cookie sent on the initial request and a redirect to `leboncoin.fr`; redirects to other scheme/userinfo/port/subdomain/suffix lookalike/Amazon blocked before the transport sees them; four-request bound; classification for success, inactive, missing price, donation, 403, challenge page, malformed, wrong ID, 404, 410, 500, network error; renewal only on verified listings with Max-Age/Expires/session-cookie, host-only/other-path/wrong-domain/unrelated/invalid/overflow ignored, ambiguity, identical duplicates; deletion on 200/403/404/410, deletion surviving a body read error, redirect staging; store fresh DB, upgrade of a database created without the table (existing item kept), reopen persistence, save/stale/resave/clear/no-op clear, status derivation with injected time (expiry does not bump the revision), conditional finish (outcome-only, expiry-only, value change, stale attempt, schema-rejected write leaves the row intact); service renewal stored and reused by a new `Service` on the same database (restart), rejection keeps value and last price, revocation stops requests until a new save, expired session makes no request, operator save or clear during a paused attempt wins with no hint from the old attempt, two concurrent renewals apply exactly once, completion write failure keeps the observation and the row with the fixed diagnostic, read failure skips the request; HTTP 200/400 (`INVALID_JSON`, `INVALID_REQUEST`, `INVALID_SESSION`)/405 with `Allow`/409/413/500 bodies, `no-store`, and captured logs / item bodies never containing the synthetic values.

## Limitations and notes for review

- Challenge detection reads the body only on 2xx HTML responses (existing flow); a non-403, non-2xx challenge response is classified `failed`, not `rejected`. DataDome challenges observed in the investigation were 403 or 200 pages, both covered.
- A `Set-Cookie: datadome` without an explicit `Path=/` is ignored. The spec requires path `/`; for listing URLs the RFC default path would never be `/`, so this only differs for requests to the site root, which the collector never makes.
- A session-assisted attempt whose listing URL fails canonical validation (never the case for stored items) reports outcome `failed` without a request.
- When the worker context is cancelled at shutdown during a session-assisted attempt, the completion write fails and logs the fixed “could not be saved” diagnostic; the stored row stays unchanged.
- Shared mutable test state: service tests replace `http.DefaultTransport` and `log` output; they do not run in parallel (`t.Parallel` unused), and the race suite passes.
- The untracked `.tmp-session-qa/main.go` (pre-existing QA scratch outside `./...` patterns) still references the removed `NewCollectorWithSession` and `LeboncoinSessionFile`; it is not part of the build or test patterns and was left untouched (not owned).
- `npm run build`, Playwright and `scripts/build-release.sh 6` were not run by this agent (frontend/scripts owned by other developers); the ARMv6 check was a plain Go cross-compile without the embedded frontend build step.
- Not executed on a Raspberry Pi; no live LeBoncoin request.

## Review fixes

Source: [decisions.md](decisions.md) (fix dispositions) and the amended specifications: TS-LBC-SET-002 rule 3, TS-LBC-COL-008 step 0, TS-LBC-COL-009 “Attribute parsing” and “No request sent”. Go files only.

| Finding | Change | Test (written first) |
| --- | --- | --- |
| REV-SET-002 | `internal/httpapi/server.go` `handleLeboncoinSession` PUT: a `*http.MaxBytesError` from the trailing-content `Decode` now returns 413 `REQUEST_TOO_LARGE` “Request body is too large.”; other trailing content stays 400 `INVALID_JSON`. The pre-existing `POST /items` handler is unchanged. | `TestReviewOversizedTrailingDataIs413` (valid value + 40 000 spaces + `x` → 413, revision still 0) |
| REV-SET-003 | `ParseSessionInput`: if a prefix was stripped and nothing remains → “No datadome cookie was found in the pasted text.” | `TestReviewBarePrefixHasNoDatadome` (`Cookie:`, `set-cookie:  `, `Cookie: ;`) |
| REV-SET-004 | `SetCookies`: the `len(raw.Unparsed) > 0` check is replaced by `hasMalformedMetadata`. It skips the cookie only when an `Unparsed` entry's name (text before `=`, trimmed, case-insensitive) is `expires`, `max-age`, `domain`, `path`, `samesite`, `secure`, `httponly` or `partitioned`. Other names are ignored. | `TestReviewUnknownAttributesIgnored` (`Priority=High` renewal → Renewed; deletion with `Priority=High` → Revoked; `Max-Age=abc` + valid future `Expires` → ignored; `Expires=garbage` → ignored). The existing overflow and scope tests still pass. |
| REV-SET-005 | `CollectWithSession` returns a zero `SessionOutcome{}` when canonical URL validation fails before any request. `collectLeboncoin` skips `FinishLeboncoinSessionAttempt` when `outcome.Attempt == ""`. The item result is unchanged (`request_error`). | `TestReviewInvalidURLReturnsZeroOutcome` (collector: no request, zero outcome); `TestReviewNoRequestRecordsNoSessionOutcome` (service: item with stored URL `https://example.com/ad/test/123` → revision and `lastAttempt` unchanged) |

RED (`go test -timeout 60s -run TestReview ./internal/...`, before the fixes): all 5 new tests failed.
- 002: `status 400, want 413` with `INVALID_JSON`.
- 003: `Cookie:` and `set-cookie:  ` gave the invalid-characters message; `Cookie: ;` already gave the correct message.
- 004: renewal with `Priority=High` → `Renewed:false`.
- 005: the collector returned `Attempt:failed`, and the service stored `LastAttempt`.

GREEN and checks after the fixes:
- `gofmt -l cmd config internal`: no output.
- `go vet ./...`: clean.
- `go test -race ./...`: ok for httpapi, leboncoin, service and store.
- `go test -count=3 ./internal/...`: ok.
- ARMv6 cross-compile (`GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0`): succeeds; output is an ELF 32-bit ARM executable.

The “unsupported listing URL” limitation above no longer applies: such an attempt now records no session outcome.
