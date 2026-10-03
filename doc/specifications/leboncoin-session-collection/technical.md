# Technical specification: LeBoncoin session collection

Status: ready (revision 2, 2026-10-03; API approved by the user 2026-10-03 (proposal revision 1 unchanged; see the change index and api-step.md); implementation authorized)\
Functional source: [collection requirements](functional.md) revision 2, FR-LBC-COL-011–020 (FR-LBC-COL-007/008 unchanged; 001–006, 009, 010 superseded).\
Revision 1 status: approved design implemented by change [leboncoin-session](../../changes/leboncoin-session/index.md); its file/sidecar parts are superseded below and kept as history. See the [deprecation record](../../deprecated/leboncoin-session-file.md).\
Companions: [settings](../leboncoin-session-settings/technical.md) (storage table, API), [capture rev. 2](../leboncoin-session-capture/technical.md), [file removal](../leboncoin-session-file-removal/technical.md).

## Revision 2 summary

The session source becomes the single-row `leboncoin_session` table defined in the [settings design](../leboncoin-session-settings/technical.md) (TS-LBC-SET-001). The service reads it at the start of every LeBoncoin attempt and passes it to the collector; the collector stays free of persistence and returns the attempt's session outcome; the service persists renewal/revocation and the hint record with one write conditioned on the `revision` read at the start. This replaces the file import, fingerprinted sidecar, private-file checks and the capacity-one gate. Revision 1 IDs are preserved: TS-LBC-COL-001, 003 and 004 are superseded; TS-LBC-COL-002 (exact-origin cookie handling, redirect validation, verified-only renewal, deletion on any status) is retained and amended by TS-LBC-COL-009; TS-LBC-COL-005 is retained as TS-LBC-COL-011.

## Revision 2 requirement mapping

`COL-*` abbreviates `FR-LBC-COL-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-LBC-COL-006 | COL-011, COL-012, RM-001 | Remove file configuration and the collector-owned session store; `leboncoin.NewCollector(userAgent)` is the only constructor; service holds the LeBoncoin collector in a dedicated field | `config/config.go`, `internal/service/service.go`, `internal/leboncoin/collector.go`, `internal/leboncoin/session.go` | Build; sessionless tests unchanged; Amazon receives no cookie |
| TS-LBC-COL-007 | COL-011, COL-013, COL-015, COL-020 | Per-attempt store read in the service and decision table (none / read error / expired-revoked / active) | `internal/service/service.go`, `internal/store/store.go` | Service tests with a real temporary SQLite store and injected transport |
| TS-LBC-COL-008 | COL-013, COL-014, COL-015, COL-016, COL-017 | `Store.FinishLeboncoinSessionAttempt` — one transaction conditioned on start `revision`; renewal, revocation, outcome record; sanitized failure handling | `internal/store/store.go`, `internal/service/service.go` | Deterministic interleaving tests (save during paused attempt), write-failure test |
| TS-LBC-COL-009 | COL-007, COL-008, COL-012, COL-014, COL-015, COL-017 | `Collector.CollectWithSession` with the retained per-attempt jar; fixed cookie identity `.leboncoin.fr` `/`; outcome classification accepted / rejected / failed | `internal/leboncoin/collector.go`, `internal/leboncoin/session.go` | Table tests with fixture transport: 200 matching, wrong ID, 403, challenge marker, 404/410, redirects, Set-Cookie renew/delete/ambiguous |
| TS-LBC-COL-010 | COL-016, COL-018, COL-019 | Fixed safe messages and logs; value never logged or returned outside the settings endpoint | collector, service | Captured-log and API-body assertions with synthetic values |
| TS-LBC-COL-011 | COL-007, COL-008, COL-019, RM-007 | Retained TS-LBC-COL-005: scheduling, observations, item API, pure-Go ARMv6 build unchanged | — | `go test ./...`, `go test -race ./...`, `scripts/build-release.sh 6` |

## Frontend (revision 2)

Not affected by this subject. The item list and detail views keep rendering `lastAttempt.message`; only the text of LeBoncoin session-assisted failure messages changes (TS-LBC-COL-010). The settings modal that displays the session state belongs to the [settings subject](../leboncoin-session-settings/technical.md).

## Backend (revision 2)

### TS-LBC-COL-006: wiring and removal

- `config.Config.LeboncoinSessionFile` and its `LEBONCOIN_SESSION_FILE` read are deleted; the variable is ignored if present (FR-LBC-RM-001).
- `leboncoin.NewCollectorWithSession` and `sessionStore` (file load/save, `privateSessionDirectory`, `readPrivate`, fingerprint, sidecar, gate) are deleted. `leboncoin.Collector` keeps only `client` and `userAgent`.
- `service.New` creates one `leboncoin.NewCollector(cfg.UserAgent)` and stores it in a new field `leboncoin leboncoinCollector`, where

  ```go
  type leboncoinCollector interface {
      CollectWithSession(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome)
  }
  ```

  The `collectors` map keeps only `"amazon"`. `collectReserved` dispatches `item.Platform == "leboncoin"` to `s.collectLeboncoin(ctx, item)`; everything else is unchanged (in-flight tracking, 31-second request context, listing re-check, `RecordCollection`). The interface exists so service tests can inject a fake; no other abstraction is added.

### TS-LBC-COL-007: per-attempt session read

`collectLeboncoin(ctx, item) model.CollectionResult`, called before the 31-second request context is created so the store read does not consume the network budget:

| Stored state (read with `Store.LeboncoinSession` at attempt start) | Behavior | Item result |
| --- | --- | --- |
| Read error | No request, no cookie. Log fixed text `LeBoncoin session could not be read; check skipped` (COL-020). | `request_error`, message “The LeBoncoin session could not be read. The check will be retried at the next scheduled time.” |
| `value` NULL (`status: none`) | Existing sessionless path: `CollectWithSession(ctx, item, nil)` behaves exactly like today's sessionless `Collect` (COL-011). No session write. | Existing outcomes and messages |
| `status` `revoked` or `expired` | No request (no stale value, and no sessionless fallback — FR-LBC-COL-015). No session write (no session-assisted attempt happened). Log fixed text `LeBoncoin session expired or revoked; check skipped`. | `request_error`, message “The saved LeBoncoin session has expired or was revoked. Save a new session in Settings.” |
| `status` `active` | Remember `startRevision`; `CollectWithSession(ctx, item, &Session{Value, ExpiresAt})`; then `FinishLeboncoinSessionAttempt(ctx, startRevision, outcome, now)` (TS-LBC-COL-008) before `RecordCollection`. | Collector result |

Because the read happens per attempt, a save or clear applies to every attempt that starts afterwards without restart (COL-013). Immediate (add), manual (refresh one/all) and scheduled attempts all pass through `collectReserved`, so they share this path.

No gate/serialization is added. Two concurrent attempts are safe because the completion write is conditional (below): the first one that changes the session increments `revision`, and the other's renewal and outcome are discarded as belonging to an older value. Sessionless concurrency is unchanged.

### TS-LBC-COL-008: completion write

`Store.FinishLeboncoinSessionAttempt(ctx, startRevision int64, outcome leboncoin.SessionOutcome, now time.Time) (applied bool, err error)` — the store receives plain fields (the developer may define a small store-level struct to avoid the store importing `leboncoin`). One transaction, no network I/O:

0. If the outcome is zero (`Attempt == ""`: no request was sent, e.g. the stored URL failed validation), the service does not call this method; no session outcome is recorded when no request was sent.
1. Read the row. If `revision != startRevision` → return `applied = false` without writing (an operator save/clear, another attempt's renewal or a revocation happened since the start; COL-013: an earlier attempt never overwrites or resurrects later state; COL-017: its outcome belongs to an older value).
2. If `outcome.Revoked`: `revoked_at = now`, `revision = revision + 1`, `updated_at = now`; value kept for display (FR-LBC-SET-015).
3. Else if `outcome.Renewed`: if the renewed value differs from the stored value → `value = renewed`, `expires_at = renewed expiry (NULL if none)`, `revoked_at = NULL`, `revision = revision + 1`, `updated_at = now`; if only the expiry differs → update `expires_at` only (no revision change, so an open modal is not invalidated by an invisible change).
4. Always `last_attempt_at = now`, `last_attempt_outcome = outcome.Attempt`.
5. `UPDATE … WHERE id = 1 AND revision = startRevision`; commit.

Failure (COL-016): any error is logged as the fixed text `LeBoncoin session update could not be saved; the stored session is unchanged` and ignored; the collection result is still recorded normally. Nothing is retained in memory: the stored row is the single source of truth, so the next attempt uses the previous complete value. A transaction never leaves a partial value (SQLite atomicity; CHECK constraints reject empty or oversize values).

`applied = false` is logged as `LeBoncoin session changed during a check; that check's session update was discarded` (no value).

### TS-LBC-COL-009: collector session handling (amends TS-LBC-COL-002)

New exported types in `internal/leboncoin/session.go`:

```go
type Session struct {
    Value     string
    ExpiresAt *time.Time // nil: unknown/session cookie
}
type SessionOutcome struct {
    Attempt   string     // "accepted", "rejected" or "failed"
    Renewed   bool
    Value     string     // renewed value, when Renewed
    ExpiresAt *time.Time // renewed expiry, when Renewed
    Revoked   bool
}
```

`Collector.CollectWithSession(ctx, item, session *Session)`: with `session == nil`, run the existing sessionless code unchanged and return a zero outcome. Otherwise reuse the existing session request path (per-attempt `http.CookieJar` on a local copy of the client, `allowedSessionURL` validation of the initial URL and every redirect, `Cookie` header removal on redirect, maximum four redirects, canonical listing URL, `attempt.verified` set after the listing ID matches) with these amendments:

- **Cookie identity.** The stored value is sent as cookie `datadome` with domain `.leboncoin.fr` and path `/` — only to `https://leboncoin.fr` and `https://www.leboncoin.fr` (port 443, no userinfo), never to Amazon, a non-LeBoncoin redirect target or any other host (COL-012). This matches the scope captured by the desktop helper (QA RT-LIVE-001: domain `.leboncoin.fr`, path `/`). No other cookie is added.
- **Attribute parsing.** A `datadome` `Set-Cookie` is ignored if it carries malformed metadata: an `Expires`, `Max-Age`, `Domain`, `Path`, `SameSite`, `Secure`, `HttpOnly` or `Partitioned` attribute the parser could not interpret (it appears in Go's `Cookie.Unparsed`; name = text before `=`, trimmed, case-insensitive). Attributes with any other name (e.g. `Priority=High`) are ignored and do not prevent renewal or deletion.
- **No request sent.** If the canonical/initial URL validation fails before any request, `CollectWithSession` returns the item `request_error` and a zero `SessionOutcome{}`; the service skips `FinishLeboncoinSessionAttempt`.
- **Renewal.** A `Set-Cookie: datadome=…` from an allowed HTTPS response is a candidate renewal only if its `Domain` attribute normalizes to `.leboncoin.fr` and its path is `/`; host-only or other-path `datadome` cookies are ignored (fail safe). Retained rules: Max-Age precedence over Expires, overflow rejection, value validation with the existing `validCookie` byte rules, more than one distinct applicable replacement in one response is ambiguous and ignored, and a candidate becomes `Renewed` only if the final response is the verified requested listing (`attempt.verified`), never on a challenge, 403, wrong listing, malformed or unavailable response (COL-014).
- **Revocation.** A matching deletion (`Max-Age<=0` or past `Expires`) from an allowed response sets `Revoked` regardless of status (retained rule), unless a verified listing later in the same chain supplied a valid replacement (then `Renewed`).
- **Expiry.** The service does not call the collector for an expired session; the jar also never sends a candidate whose known expiry has passed (retained check).
- **Outcome classification (COL-017).** `accepted` if `attempt.verified` (LeBoncoin returned the requested listing, including inactive or priceless listings); `rejected` if the final response status is 403, or its body contains `captcha-delivery.com` (DataDome challenge marker) without a matching listing; otherwise `failed` (network/timeout, redirect error, other non-2xx including 404/410, unexpected content type, missing `__NEXT_DATA__`, wrong ID). 404/410 therefore record `failed`, because the requested listing was not returned; the item result stays `unavailable` as today.

### TS-LBC-COL-010: messages and secret handling

`lastAttempt.message` for session-assisted attempts (free text in the approved contract):

| Situation | Item result | Message |
| --- | --- | --- |
| Rejected (403 / challenge) | `request_error` | “LeBoncoin rejected the saved session. Capture a new session and save it in Settings.” |
| Other session-assisted failure | `request_error` | “LeBoncoin could not be reached for a price check.” (same text as sessionless) |
| Expired/revoked pre-check, read error | `request_error` | see TS-LBC-COL-007 |
| Unavailable, price not found, success | unchanged | unchanged |

`sessionCollectionError` and its “private session files” text are removed. Logs for session-assisted requests keep omitting the URL and never include cookie values, `Set-Cookie`/`Cookie` headers, raw library errors from the session path or store errors that could echo input (COL-018). The value leaves the server only through `GET/PUT /api/v1/settings/leboncoin-session`.

FR-LBC-COL-019 needs no code: nothing claims acceptance on save; the hint and existing refresh controls show results.

### TS-LBC-COL-011: preserved behavior

Retains TS-LBC-COL-005: schedule, refresh endpoints, observation storage, euro cents, stale-price retention, Amazon collection and the pure-Go ARMv6 release are unchanged. The migration is additive (settings design). No new Go module dependency.

## API (revision 2)

No change to item endpoints or shapes; see the pending settings proposal in [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) for the session state that this subject writes (`status`, `expiresAt`, `revokedAt`, `lastAttempt`). This document does not constitute approval.

## Scope and refactoring (revision 2)

Permitted files: `config/config.go`, `config/config_test.go` (deleted; it only tests the removed variable), `internal/leboncoin/collector.go`, `internal/leboncoin/session.go` (rewritten: parser, types, jar, classification), `internal/leboncoin/session_test.go` (rewritten), `internal/service/service.go`, `internal/service/session_test.go` (rewritten for store-backed sessions), `internal/store/store.go`, `internal/model/model.go`, new store/httpapi test files. The collector-to-service move of session persistence is part of replacing the file mechanism, not a separate refactor: the go-backend-architecture skill requires the collector to stay free of persistence, and the file store being removed was the only persistence in the collector. No other refactoring is proposed.

## Verification (revision 2)

Write failing behavioral tests first (workflow), with synthetic values and an injected `http.RoundTripper` (no live LeBoncoin in automated tests):

1. No session: request has no `Cookie`; outcomes unchanged; Amazon never receives a cookie with or without a saved session.
2. Active session: initial request carries exactly `datadome=<value>` to `www.leboncoin.fr`; redirect to `leboncoin.fr` keeps it; redirect to another host/port/scheme/userinfo is blocked before the transport sees the value.
3. Verified 200 with `Set-Cookie` renewal (Max-Age, Expires, none) → stored value/expiry updated, `revision` +1, outcome `accepted`; a fresh `Service` on the same DB sends the renewed value (restart).
4. 403 or challenge page with a new `Set-Cookie` value → value unchanged, outcome `rejected`, item `request_error` with the rejection message, last price retained.
5. Matching deletion on 403 → `revoked_at` set, `revision` +1; the next attempt makes no request and records the expired/revoked message; after a new save the session is used again.
6. Expired `expires_at` → no request, `request_error`; status `expired`.
7. Interleaving: fake transport blocks attempt A; operator saves new value (revision changes); release A with a renewal → stored value is the operator's, no hint from A; same for clear (no resurrection). Two concurrent attempts both renewing → exactly one renewal applied.
8. Store read failure (closed DB or injected failing store) → `request_error`, no transport call; write failure on finish → observation still recorded, row unchanged.
9. Logs captured with `log.SetOutput` and every API body contain no synthetic value.
10. `go test ./...`, `go test -race ./...`, `npm run build`, `scripts/build-release.sh 6`.

QA (later stage): isolated server; save a session from the UI, refresh a LeBoncoin item through a fixture or, at most once, live; confirm hint after a rejection, revoked/expired states, restart persistence, clear → sessionless.

No unresolved product question blocks this subject.

---

# Revision 1 technical design (history)

The following revision 1 design (status at the time: ready, implemented) is kept unchanged for history except for the superseded markers in its mapping table. Its functional source was FR-LBC-COL-001–010.

## Revision 1 requirement mapping (history)

| Technical ID | Functional IDs | Design / intended files | Verification |
| --- | --- | --- | --- |
| TS-LBC-COL-001 — **superseded** by TS-LBC-COL-006, TS-LBC-COL-007 | COL-001, COL-002, COL-003 | Optional config and service wiring; lazy protected import | Configuration, missing/invalid file and platform-isolation cases |
| TS-LBC-COL-002 — retained, amended by TS-LBC-COL-009 | COL-002, COL-007, COL-008 | Exact-origin single-cookie request state; existing page/price validation | Injected HTTP transport and redirect cases; actual collector fixture |
| TS-LBC-COL-003 — **superseded** by TS-LBC-COL-007, TS-LBC-COL-008 | COL-004, COL-005 | Immutable import; fingerprint-bound sidecar; serialize session attempts | Cookie update/delete, restart and in-flight renewal tests with deterministic barriers |
| TS-LBC-COL-004 — **superseded** by TS-LBC-COL-008, TS-LBC-COL-010 | COL-003, COL-006, COL-009 | Safe reads, atomic private writes, sanitized errors; retain parsed prices | Permissions/malformed data/write failure/secret-redaction cases |
| TS-LBC-COL-005 — retained (see TS-LBC-COL-011) | COL-001, COL-007–010 | Preserve service scheduling, observations, API and pure-Go ARM build | Service/application QA, Go race suite and ARMv6 production build |

`COL-*` abbreviates `FR-LBC-COL-*` in this table.

## Frontend and API (revision 1, history)

Not affected. The existing Carbon views render the existing attempt result/message and retained observations. No new route, form, field, upload or session status is added. [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) remains the approved contract: add stays asynchronous, existing refresh endpoints stay `202`, collection failures use `request_error`, and successful prices remain integer euro cents. Safe existing `lastAttempt.message` strings may explain unavailable session/renewal; do not expose paths, parser details, upstream HTML or secrets. This single-backend-tier change requires no new API approval.

## Backend integration and file scope (revision 1, history)

TS-LBC-COL-001: Add `Config.LeboncoinSessionFile` from `LEBONCOIN_SESSION_FILE` (empty/unset disables the feature; trim surrounding whitespace). Preserve every other default. Pass it from `internal/service.New` into the LeBoncoin collector. Existing constructor callers must remain clear: an explicit optional session constructor or a direct second parameter with all callers updated is acceptable; avoid variadic options/frameworks. Do not require file existence at process startup. A malformed/missing/unsafe configured file causes a sanitized collection `request_error`, not a configuration failure that stops the service/Amazon.

Permitted application scope: `config/config.go`, `internal/service/service.go` (configuration wiring only except a necessary fixed diagnostic integration), `internal/leboncoin/collector.go`, and one focused `internal/leboncoin/session.go` with corresponding focused tests. Add service/config tests only to prove actual integration requirements. Keep SQLite schema, scheduler, Amazon collector, HTTP handlers and frontend unchanged. No new module/package dependency or generic credential/cookie framework. Use Go standard library. Session filesystem persistence belongs to the narrow LeBoncoin session helper, outside request handlers and SQLite transactions; collector page parsing remains in the adapter. No unrelated or prerequisite refactoring is proposed.

TS-LBC-COL-002: With no file configured, retain the existing headers, 15-second HTTP timeout, outcomes and sessionless behavior. Session attempts share the existing service's 31-second enclosing context. Always validate a session-assisted initial URL and every redirect before any cookie is attached: HTTPS, no userinfo, empty/443 port, exact allowed apex or www LeBoncoin host, with existing canonicalization of a supported initial listing. Preserve the existing maximum of four requests in a redirect chain. Reject unsafe redirects before network dispatch (including subdomains, suffix lookalikes, other ports and userinfo). Never allow automatic forwarding of an existing Cookie header to bypass these rules. Use only one validated `datadome` cookie and its original host/domain/path scope; Amazon never receives session state.

A small per-attempt implementation of `http.CookieJar`, or equivalently explicit scoped request/response handling, may retain one cookie and a deletion marker. It is not a general-purpose cookie framework. Inspect full parsed `Set-Cookie` metadata; `cookiejar.Jar.Cookies` alone loses expiry attributes and is insufficient for persistence. A configured-session attempt may use a local copy of the client's settings with this per-attempt jar, avoiding mutation of a shared client. Filter both cookie writes and reads by the exact HTTPS destination and `datadome` name. Validate values and reject invalid scope. Ignore all unrelated cookies. Do not log raw response cookie headers or cookie validation inputs.

## Session loading and durable state (revision 1, history)

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

### Concurrency and replacement ordering (revision 1, history)

Use a context-cancellable, capacity-one gate for the entire configured-session attempt (load, HTTP chain, validation, state update and save). Waiting observes `ctx.Done()`; canceled/shutdown work cannot block indefinitely. Only LeBoncoin opt-in attempts use this gate; sessionless and Amazon paths retain their behavior. No database transaction is held. Serialization is intentionally local and small; it prevents two responses from rolling the same session backwards and stays within the existing bounded requests.

Each attempt takes its import fingerprint when admitted to the gate and never writes `P`. Its sidecar commit uses that captured fingerprint. If an operator atomically replaces `P` while that request is in flight, the old response may only write state tagged with the old fingerprint; the next admitted attempt reads the new import and ignores that stale sidecar. Because attempts are serialized, an older response cannot subsequently overwrite sidecar state already committed by a newer attempt. This also works across restart. Do not replace this design with a check-then-rename overwrite of `P`: an operator rename between the check and write would lose renewal. Multiple backend processes sharing one configured session directory are outside the existing single-process architecture; document one running owner.

## Response update and deletion rules (revision 1, history)

TS-LBC-COL-002/003: Retain the original effective cookie as the durable baseline. During a redirect chain, stage only applicable, syntactically valid `datadome` updates/deletions in per-attempt memory; subsequent allowed hops may use that provisional state. Persist the candidate only after the final response is allowed HTTPS, has a 2xx status and supported HTML content type, and its parsed `__NEXT_DATA__` contains the requested nonzero listing ID. This is evidence of a legitimate listing response, even if the listing is inactive or its price is absent. Perform that accepted-state step before existing inactive/price-result branches. A final challenge, 403, redirect error, wrong ID, malformed page, 404/410, read error or other unverified response discards provisional new/replacement values. Explicit deletion is the exception below. Existing 404/410 unavailable behavior and other price/result branches stay intact. In particular, an unverified cookie set on a 403 must not replace a previously usable session. No automatic retry is added.

For accepted `Set-Cookie` processing, require an allowed source origin, exact name and domain/path applicability to that source. Missing Domain means the response host (host-only); explicit valid Domain becomes domain scope. Missing Path uses the RFC default path of the response request; match cookies using RFC path boundaries. Preserve Secure metadata but permit sends only over allowed HTTPS. A positive Max-Age takes precedence over Expires and becomes an absolute UTC deadline at response receipt; negative Max-Age or an explicit already-expired Expires deletes the matching cookie; a missing expiry/Max-Age yields null expiry. Go's parsed `MaxAge < 0` represents explicit deletion, including `Max-Age=0`. Reject overflow/invalid metadata safely, never extend expiry through integer wraparound. Cookie replacement/deletion must match the effective cookie's domain/path identity; an unrelated-path deletion cannot remove it. An unambiguous applicable replacement may update that identity; ambiguous applicable cookies fail safe rather than broadening scope. Honor an explicit syntactically valid deletion matching the original effective cookie identity from an allowed HTTPS response regardless of response status, including 403/404/410; recording revocation requires no successful page. Track that deletion separately from provisional replacement values so a later failed chain cannot resurrect the original cookie. A matching successful final listing may validate a subsequent applicable replacement in the same chain; without that success, preserve the deletion rather than any unverified new value. Retain a null deletion tombstone in the sidecar so restart cannot restore the older import. Commit this deletion on error paths as well as accepted-page paths; a response received before a later redirect/read failure still supplies valid revocation evidence. Natural expiration likewise prevents sending the saved value.

Write changed verified replacement state or valid explicit deletion state only; retain other parsed result handling. Use an exclusive mode-0600 temporary file in the same private directory, complete JSON write, sync/close then atomic rename; reject unsafe existing sidecar targets and clean up temporary files. Never truncate either installed file. If persistence fails, emit a fixed sanitized log diagnostic, retain current in-memory effective state, and return the normally parsed success/unavailable/price-not-found result. This failure must not erase an observation or manufacture one. On session rejection, a fixed user-safe attempt message may direct the operator to renew; logs and API must not include raw headers, cookie values, file contents or decoder snippets. Inspect error handling added around redirects and JSON carefully, as raw library errors can include attacker-controlled strings.

## Verification plan and dependency order (revision 1, history)

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

## Revision note (revision 2 amendments)

2026-10-03, review amendments: REV-SET-001: status updated to record the API approval of 2026-10-03. REV-SET-004: TS-LBC-COL-009 ignores unknown `Set-Cookie` attributes and still rejects malformed known metadata. REV-SET-005: TS-LBC-COL-008/009 return a zero outcome and skip the finish write when no request was sent.
