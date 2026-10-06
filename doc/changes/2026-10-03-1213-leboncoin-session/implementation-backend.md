# Go backend implementation evidence

Status: implemented; final native and race suites passed; source frozen for independent review. Date: 2026-10-03. Agent: `/root/leboncoin_session_backend_developer` (implementation role).

Scope follows the ready [collection functional specification](../../specifications/leboncoin-session-collection/functional.md), [technical specification](../../specifications/leboncoin-session-collection/technical.md), [shared capture format](../../specifications/leboncoin-session-capture/technical.md), and [unchanged API record](api-step.md). Read AGENTS.md, developer role, workflow, and all four required skills before implementation. No HTTP contract, SQLite schema, frontend, scheduler behavior, dependency, or unrelated refactoring change.

## Files and requirement coverage

- `config/config.go`, `config/config_test.go`: optional trimmed `LEBONCOIN_SESSION_FILE`; blank preserves sessionless behavior and missing configured files do not prevent startup (TS-LBC-COL-001; FR-LBC-COL-001–003).
- `internal/service/service.go`: pass the configured path to `leboncoin.NewCollectorWithSession`; existing collector map serves immediate, manual, and scheduled attempts unchanged (TS-LBC-COL-001/005).
- `internal/leboncoin/collector.go`: session-only cancellable admission, canonical initial listing validation, strict redirect origins, per-attempt cookie jar, existing matching-listing/price branches, and fixed safe error messages. A successful matching page validates updates before inactive/missing-price branches; deletion finalization also runs on errors (TS-LBC-COL-002/003/004).
- `internal/leboncoin/session.go`: exact single-cookie format validation, bounded owned/private nonsymlink regular-file reads, owned private directory checks, immutable operator import, SHA-256 import fingerprint, atomic mode-0600 sidecar writes, durable null revocation, effective-cookie expiry, in-memory updates after failed persistence, and scoped response-cookie metadata processing (TS-LBC-COL-002–004; FR-LBC-COL-002–006/009).
- `internal/leboncoin/session_test.go`: deterministic injected HTTP transport and private temporary files; invalid input, isolation, updates, deletion, errors, expiry, concurrency, replacement, persistence failure, and redaction coverage. Synthetic cookie values only.
- `internal/service/session_test.go`: isolated SQLite and injected transport exercise add/immediate, item/manual refresh, and scheduled 403 through the real service; verify retained price/history and no cookie on Amazon (FR-LBC-COL-001/007/008/010).

The filesystem helper uses standard-library `os.Root` directory handles and no-follow/nonblocking opens. Descriptor metadata is checked before reads; FIFO input does not block. Atomic writes never overwrite the operator import. This deployment follows the specified single-process owner model.

## Test-first evidence

Before implementing session behavior, added only the minimal `NewCollectorWithSession` constructor stub delegating to the existing collector, plus three behavioral tests. Executed:

```text
GOCACHE=/tmp/pricefollower-session-go-cache go test ./internal/leboncoin -run 'TestSession' -count=1
```

RED: compiled successfully and exited 1 with three actual assertion failures:

- `TestSessionCookieAttached`: no cookie was attached.
- `TestSessionMissingFailsBeforeRequest`: returned success and dispatched one request despite missing configured import.
- `TestSessionVerifiedUpdateSurvivesRestart`: restart did not use the verified update.

After implementation the same cases passed. Expanded focused tests then passed before whole-repository verification.

## Executed checks

| Command | Result |
| --- | --- |
| `GOCACHE=/tmp/pricefollower-session-go-cache go test ./internal/leboncoin -count=1` | PASS, including expanded session cases. |
| `GOCACHE=/tmp/pricefollower-session-go-cache go test ./config ./internal/leboncoin ./internal/service -count=1` | PASS, all three packages. |
| `GOCACHE=/tmp/pricefollower-session-go-cache go test ./...` | PASS, every package. |
| `GOCACHE=/tmp/pricefollower-session-go-cache go test -race ./...` | PASS, every package; no race reports. |
| `git diff --check` | PASS. |

Final native toolchain: `go version go1.25.0 linux/amd64`. There are 20 top-level Go tests: 1 configuration, 18 LeBoncoin session tests (with table-driven subcases), and 1 service integration test. Final whole-repository timings: LeBoncoin 0.031s/service 0.012s; race LeBoncoin 1.096s/service 1.085s (other test package cached).

An intermediate race run failed a newly added service test assertion while that test polled independently assembled item fields during active worker completion. The final fixture waits for the actual service worker before asserting the stored result. A subsequent history assertion was corrected to use `PriceHistory`, because the approved `LastThreeDetections` intentionally collapses repeated equal prices. No application change was made for either test issue; both final suites passed after these corrections.

Coordinator separately reports `scripts/build-release.sh 6` passed using Node 22 and Go 1.25.0; `file` and `go version -m` confirmed ARM EABI5, GOARM=6 and CGO_ENABLED=0. This is coordinator evidence, not a backend-agent execution or remote-device test.

Fixture cases include 403, malformed/wrong-ID pages, unavailable and price-not-found states, donation zero, unsupported initial/redirect origins, four-request redirect limit, default path and host-only scope, unrelated/ambiguous response cookies, overflowing Max-Age, Max-Age precedence over Expires, verified redirect staging, explicit deletion before redirect/body-read failure, restart tombstones, stale in-flight response after atomic renewal, cancelled gate waiter, parallel requests, malformed/oversized/unsafe/symlink/FIFO import, sidecar extension past import expiry, preservation of complete previous sidecar after save failure, and fixed diagnostics without upstream error text.

## Limits and handoff

No live external requests, graphical-browser capture, application-interface QA, Raspberry Pi execution, or portability claim is part of this backend-agent evidence. Coordinator/QA own live acceptance and the existing ARMv6 release script after both developers finish. A saved desktop session may still be rejected remotely. Failed sidecar persistence retains current-process state only; restart can recover only the previous saved state, as the diagnostic states. No commit performed: coordinator owns the commit stage after review, adjudication, and QA.
