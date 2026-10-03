# Technical specification: LeBoncoin session collection

Status: ready  
Functional source: [collection requirements](functional.md), FR-LBC-COL-001–010.  
Dependency: [capture and shared format](../leboncoin-session-capture/technical.md).

## Requirement mapping

| Technical ID | Functional IDs | Design / intended files | Verification |
| --- | --- | --- | --- |
| TS-LBC-COL-001 | COL-001, COL-002, COL-003 | Optional config and service wiring; lazy protected import | Configuration, missing/invalid file and platform-isolation cases |
| TS-LBC-COL-002 | COL-002, COL-007, COL-008 | Exact-origin single-cookie request state; existing page/price validation | Injected HTTP transport and redirect cases; actual collector fixture |
| TS-LBC-COL-003 | COL-004, COL-005 | Immutable import; fingerprint-bound sidecar; serialize session attempts | Cookie update/delete, restart and in-flight renewal tests with deterministic barriers |
| TS-LBC-COL-004 | COL-003, COL-006, COL-009 | Safe reads, atomic private writes, sanitized errors; retain parsed prices | Permissions/malformed data/write failure/secret-redaction cases |
| TS-LBC-COL-005 | COL-001, COL-007–010 | Preserve service scheduling, observations, API and pure-Go ARM build | Service/application QA, Go race suite and ARMv6 production build |

`COL-*` abbreviates `FR-LBC-COL-*` in this table.

## Frontend and API

Not affected. The existing Carbon views render the existing attempt result/message and retained observations. No new route, form, field, upload or session status is added. [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) remains the approved contract: add stays asynchronous, existing refresh endpoints stay `202`, collection failures use `request_error`, and successful prices remain integer euro cents. Safe existing `lastAttempt.message` strings may explain unavailable session/renewal; do not expose paths, parser details, upstream HTML or secrets. This single-backend-tier change requires no new API approval.

## Backend integration and file scope

TS-LBC-COL-001: Add `Config.LeboncoinSessionFile` from `LEBONCOIN_SESSION_FILE` (empty/unset disables the feature; trim surrounding whitespace). Preserve every other default. Pass it from `internal/service.New` into the LeBoncoin collector. Existing constructor callers must remain clear: an explicit optional session constructor or a direct second parameter with all callers updated is acceptable; avoid variadic options/frameworks. Do not require file existence at process startup. A malformed/missing/unsafe configured file causes a sanitized collection `request_error`, not a configuration failure that stops the service/Amazon.

Permitted application scope: `config/config.go`, `internal/service/service.go` (configuration wiring only except a necessary fixed diagnostic integration), `internal/leboncoin/collector.go`, and one focused `internal/leboncoin/session.go` with corresponding focused tests. Add service/config tests only to prove actual integration requirements. Keep SQLite schema, scheduler, Amazon collector, HTTP handlers and frontend unchanged. No new module/package dependency or generic credential/cookie framework. Use Go standard library. Session filesystem persistence belongs to the narrow LeBoncoin session helper, outside request handlers and SQLite transactions; collector page parsing remains in the adapter. No unrelated or prerequisite refactoring is proposed.

TS-LBC-COL-002: With no file configured, retain the existing headers, 15-second HTTP timeout, outcomes and sessionless behavior. Session attempts share the existing service's 31-second enclosing context. Always validate a session-assisted initial URL and every redirect before any cookie is attached: HTTPS, no userinfo, empty/443 port, exact allowed apex or www LeBoncoin host, with existing canonicalization of a supported initial listing. Preserve the existing maximum of four requests in a redirect chain. Reject unsafe redirects before network dispatch (including subdomains, suffix lookalikes, other ports and userinfo). Never allow automatic forwarding of an existing Cookie header to bypass these rules. Use only one validated `datadome` cookie and its original host/domain/path scope; Amazon never receives session state.

A small per-attempt implementation of `http.CookieJar`, or equivalently explicit scoped request/response handling, may retain one cookie and a deletion marker. It is not a general-purpose cookie framework. Inspect full parsed `Set-Cookie` metadata; `cookiejar.Jar.Cookies` alone loses expiry attributes and is insufficient for persistence. A configured-session attempt may use a local copy of the client's settings with this per-attempt jar, avoiding mutation of a shared client. Filter both cookie writes and reads by the exact HTTPS destination and `datadome` name. Validate values and reject invalid scope. Ignore all unrelated cookies. Do not log raw response cookie headers or cookie validation inputs.

## Session loading and durable state

TS-LBC-COL-003/004: The operator import is immutable from the collector's perspective. For configured path `P`, persist collector state at `P + ".state.json"`; both are inside the private directory described by the capture workflow. Read files only as bounded regular files, reject final symlinks and unsafe parent directories, require owner-only file permissions (0600 or stricter), and require a private owner directory (0700 or stricter) owned by the running service user. Check the opened descriptor rather than relying solely on a path `stat` before an unsafe open; use no-follow open or lstat/open/fstat identity validation with reads delayed until verified. Reject nonregular files without blocking on a FIFO. Bound reads to 16 KiB plus one overflow byte. Use sanitized fixed diagnostic categories, not raw decoder or OS errors containing content. Strictly validate the shared import format.

Compute a SHA-256 fingerprint of the complete import bytes. A renewal has fresh `capturedAt` and therefore a new fingerprint even when the cookie value is unchanged. The private sidecar has exactly:

```json
{
  "version": 1,
  "importFingerprint": "<64 lowercase hex characters>",
  "cookie": null
}
```

`cookie` is either the same cookie object as the shared format, or null for a legitimate deletion. No additional cookies, URL, account details or HTML. Unknown fields, invalid types, invalid cookie attributes or excess content are invalid. A missing sidecar is ordinary. An intact sidecar for a different fingerprint is obsolete and ignored; a sidecar for the current fingerprint overrides the import, including a null deletion or expired cookie. A malformed/unreadable/unsafe sidecar fails closed with a renewal/state diagnostic rather than resurrecting a possibly deleted cookie. Removing a damaged sidecar is an explicit operator repair; normal renewal should use a fresh import and, if the sidecar cannot even be parsed/read, privately remove that damaged collector file before the next attempt.

Check expiry on the **effective** selected cookie, after sidecar selection. A legitimate update can extend beyond the original import's expiry; rejecting the old import before selecting its newer state would break durable renewal. A null or expired effective cookie is `request_error` with renewal guidance and sends no upstream request. Session-cookie null expiry is retained without inventing an arbitrary timeout. The operator still renews when the server rejects it.

Keep one current fingerprint/effective cookie in collector memory. Every attempt reads and validates the current import so replacement is recognized; on first load or a changed fingerprint, load the corresponding sidecar/import. Within one fingerprint, keep the most recent in-memory accepted update even if persistence previously failed. This prevents an immediate rollback after a write failure; across a restart only successfully saved state can be recovered, and the diagnostic must not promise otherwise.

### Concurrency and replacement ordering

Use a context-cancellable, capacity-one gate for the entire configured-session attempt (load, HTTP chain, validation, state update and save). Waiting observes `ctx.Done()`; canceled/shutdown work cannot block indefinitely. Only LeBoncoin opt-in attempts use this gate; sessionless and Amazon paths retain their behavior. No database transaction is held. Serialization is intentionally local and small; it prevents two responses from rolling the same session backwards and stays within the existing bounded requests.

Each attempt takes its import fingerprint when admitted to the gate and never writes `P`. Its sidecar commit uses that captured fingerprint. If an operator atomically replaces `P` while that request is in flight, the old response may only write state tagged with the old fingerprint; the next admitted attempt reads the new import and ignores that stale sidecar. Because attempts are serialized, an older response cannot subsequently overwrite sidecar state already committed by a newer attempt. This also works across restart. Do not replace this design with a check-then-rename overwrite of `P`: an operator rename between the check and write would lose renewal. Multiple backend processes sharing one configured session directory are outside the existing single-process architecture; document one running owner.

## Response update and deletion rules

TS-LBC-COL-002/003: Retain the original effective cookie as the durable baseline. During a redirect chain, stage only applicable, syntactically valid `datadome` updates/deletions in per-attempt memory; subsequent allowed hops may use that provisional state. Persist the candidate only after the final response is allowed HTTPS, has a 2xx status and supported HTML content type, and its parsed `__NEXT_DATA__` contains the requested nonzero listing ID. This is evidence of a legitimate listing response, even if the listing is inactive or its price is absent. Perform that accepted-state step before existing inactive/price-result branches. A final challenge, 403, redirect error, wrong ID, malformed page, 404/410, read error or other unverified response discards provisional new/replacement values. Explicit deletion is the exception below. Existing 404/410 unavailable behavior and other price/result branches stay intact. In particular, an unverified cookie set on a 403 must not replace a previously usable session. No automatic retry is added.

For accepted `Set-Cookie` processing, require an allowed source origin, exact name and domain/path applicability to that source. Missing Domain means the response host (host-only); explicit valid Domain becomes domain scope. Missing Path uses the RFC default path of the response request; match cookies using RFC path boundaries. Preserve Secure metadata but permit sends only over allowed HTTPS. A positive Max-Age takes precedence over Expires and becomes an absolute UTC deadline at response receipt; negative Max-Age or an explicit already-expired Expires deletes the matching cookie; a missing expiry/Max-Age yields null expiry. Go's parsed `MaxAge < 0` represents explicit deletion, including `Max-Age=0`. Reject overflow/invalid metadata safely, never extend expiry through integer wraparound. Cookie replacement/deletion must match the effective cookie's domain/path identity; an unrelated-path deletion cannot remove it. An unambiguous applicable replacement may update that identity; ambiguous applicable cookies fail safe rather than broadening scope. Honor an explicit syntactically valid deletion matching the original effective cookie identity from an allowed HTTPS response regardless of response status, including 403/404/410; recording revocation requires no successful page. Track that deletion separately from provisional replacement values so a later failed chain cannot resurrect the original cookie. A matching successful final listing may validate a subsequent applicable replacement in the same chain; without that success, preserve the deletion rather than any unverified new value. Retain a null deletion tombstone in the sidecar so restart cannot restore the older import. Commit this deletion on error paths as well as accepted-page paths; a response received before a later redirect/read failure still supplies valid revocation evidence. Natural expiration likewise prevents sending the saved value.

Write changed verified replacement state or valid explicit deletion state only; retain other parsed result handling. Use an exclusive mode-0600 temporary file in the same private directory, complete JSON write, sync/close then atomic rename; reject unsafe existing sidecar targets and clean up temporary files. Never truncate either installed file. If persistence fails, emit a fixed sanitized log diagnostic, retain current in-memory effective state, and return the normally parsed success/unavailable/price-not-found result. This failure must not erase an observation or manufacture one. On session rejection, a fixed user-safe attempt message may direct the operator to renew; logs and API must not include raw headers, cookie values, file contents or decoder snippets. Inspect error handling added around redirects and JSON carefully, as raw library errors can include attacker-controlled strings.

## Verification plan and dependency order

The capture format and these state rules are ready before implementation. Start with focused failing tests and record real behavioral RED evidence, then implement and record GREEN. A minimal new-API stub may precede behavioral tests; a compile failure alone is not behavioral RED evidence. Use synthetic cookie values and in-memory injected HTTP RoundTrippers/fixtures for deterministic Go tests; do not change global transport in parallel tests. Narrow unexported test seams are permitted where they directly exercise response ordering or I/O failure; do not create production options solely for hypothetical extensions.

Required Go cases:

1. Unconfigured behavior; configured missing/malformed/oversized/unknown-version/unsafe-permission/symlink/nonregular/expired import; process and unrelated collectors remain usable.
2. Correct cookie attached only to valid initial origin/path; all host, userinfo, port, scheme and redirect escape attempts blocked before a transport observes secrets.
3. Matching listing success, donation zero, wrong ID, absent price, unavailable and 403/challenge; no retained price is lost through the service path.
4. Accepted cookie update then second attempt; fresh collector instance uses sidecar update; Max-Age and session-cookie metadata; valid deletion then restart cannot resurrect import; wrong-scope deletion ignored.
5. 403/malformed/wrong-ID response with a new Set-Cookie value cannot replace the original verified state; a 403 exact-scope deletion persists a tombstone and remains deleted after restart; valid staged redirect update commits only after matching final listing.
6. Operator atomic renewal while a deterministic old request is paused; release it, collect again and recreate collector; new import wins at every subsequent admission. Repeated parallel attempts remain race-free. Waiting gate honors cancellation.
7. Sidecar write failure preserves import and previous complete sidecar, retains successful collection result and emits no cookie value. In-memory accepted state remains effective. Test extended sidecar expiry when the import's original expiry has passed.
8. Configuration and service wiring demonstrate initial/manual/scheduled paths all use the same collector; no HTTP/API or schema additions.

Run relevant package tests during development, then `go test ./...` and `go test -race ./...` on the native supported toolchain, helper Node tests, and `scripts/build-release.sh 6` using Node >=22 and Go >= module requirement. Verify the release is an ARMv6 pure-Go binary; cross-compilation is not remote-device execution.

QA follows independent review/adjudication: exercise the actual helper CLI and visible browser; run an isolated application with fixture-backed collection to observe add/refresh/error/retained prices, file replacement and restart through existing routes; record real local application URLs separately from the actual external listing input. An isolated task-owned verified browser profile may supply a sanitized private QA fixture; it is not a new production profile option. If possible exercise the actual Go collector with the captured cookie against the investigation listing and assert the returned ID/result/price, without treating a subprocess exit code alone as success. Live rejection is a normal observed limit requiring human renewal, not permission to fabricate a price or claim remote success. The current Vite proxy targets port 3001; for an isolated backend port, use the existing embedded production frontend or a QA browser route proxy without changing Vite product configuration. No Raspberry Pi hostname is required for implementation/testing; record remote portability as unverified unless actually tested.

No unresolved product decision, API change or refactoring approval blocks this specification.
