# Functional specification: Amazon browser profile recovery

Status: ready (issue #180); FR-AMZ-BPROF-008–010 specified (issue #197, maintenance: no behaviour change)
Owner: functional specification agent
User decision/reference: GitHub issue #180 ("Clear stale Chromium Singleton files in the persistent Amazon browser profile"), from review comment R1-02 on #179; GitHub issue #197 ("Return explicit zero values from BrowserCollector.profile when stale lock removal fails"), from review comment R1-04 on PR #194 (`doc/todo/issue-180-r1-04.md`).

## Purpose and scope

Amazon checks go through a real web browser ([FR-AMZ-HCHK-001](../amazon-human-checks/functional.md)) that keeps its profile (cookies) in the data directory. When that browser ends abruptly (killed when a check is closed or stopped, crash, server stop during a check), it leaves lock marks in the profile. If the data directory is later used on another machine or container (moved server, new container), the browser refuses the profile: every Amazon check fails with "browser could not start" and, after 5 failures, every Amazon item is stopped ([FR-AMZ-STOP-003](../amazon-request-stop/functional.md)). The user asks that such leftovers no longer prevent Amazon checks.

Issue #197 (maintenance): when discarding the stale lock fails, `profile()` in `internal/amazon/browser.go` returns the persistent profile path and an empty cleanup together with the error, while its other failure branch (the profile directory cannot be created) returns an empty path and no cleanup. The user asks that both failure branches return the same explicit empty values; no behaviour changes.

Included: starting the Amazon browser for an Amazon check when the persistent profile holds lock marks left by an earlier browser, on the same machine or another one.

Excluded: LeBoncoin; Amazon schedule, failure count, stop, warning and restart rules ([Amazon request stop](../amazon-request-stop/functional.md), [Amazon human-like checks](../amazon-human-checks/functional.md)), which stay unchanged; Amazon items already stopped are not restarted automatically by this change (each is restarted with **Refresh price**, FR-AMZ-STOP-006); running several application instances on the same data directory at the same time, which is not supported.

Actor: the user who tracks Amazon items. Amazon is the external system. There is no new screen, setting or message.

Terms:

- **Amazon check:** as in [Amazon human-like checks](../amazon-human-checks/functional.md): first check after add, scheduled check, **Refresh all prices**, **Refresh price**.
- **Persistent profile:** the Amazon browser's profile kept in the data directory between checks and restarts.
- **Stale lock:** the lock marks a browser leaves in the persistent profile when it does not end normally, including marks that name another machine.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMZ-BPROF-001 | An Amazon check starts and the persistent profile holds a stale lock from an earlier browser of this application on the same machine (for example after the browser was killed or crashed). | The browser starts with the persistent profile and the check proceeds normally; the stale lock does not cause a failure. | After a browser was killed during a check, the next Amazon check of a listing returns its price; no "browser could not start" failure is logged or counted. |
| FR-AMZ-BPROF-002 | An Amazon check starts and the persistent profile holds a stale lock naming another machine (data directory moved to another server or container). | The browser starts with the persistent profile and the check proceeds normally. | With a profile whose lock names another host (for example `otherhost-12345`), the next Amazon check returns the listing's price and no failure is counted. |
| FR-AMZ-BPROF-003 | An Amazon check starts with a persistent profile without any stale lock (first use or after a normal end). | Behaviour is unchanged: the check proceeds normally. | A check with a fresh or cleanly closed profile returns the price as before. |
| FR-AMZ-BPROF-004 | Any recovery from a stale lock. | Only the lock marks are discarded; the rest of the persistent profile (cookies and other browsing data) is kept and used by the check. | A cookie stored by Amazon in an earlier check is still sent by the check that follows the recovery. |
| FR-AMZ-BPROF-005 | Stale locks are left again after successive abrupt ends. | Each later Amazon check recovers the same way; no manual action (deleting files, restarting) is ever needed. | Two consecutive checks each preceded by a killed browser both return the price. |
| FR-AMZ-BPROF-006 | The browser still cannot start with the persistent profile for another reason (stale lock cannot be discarded, browser missing, other start error). | It remains a failure, logged, counted and stopping items as in FR-AMZ-HCHK-003 and FR-AMZ-STOP-001–010. | A profile whose lock cannot be discarded gives a "browser could not start" failure that is counted. |
| FR-AMZ-BPROF-007 | Amazon checks run one after another within the application. | Recovery never interrupts or breaks an Amazon check in progress: a check's browser is never deprived of its profile by another check of the same application. | Overlapping requests (for example **Refresh price** during a scheduled check) both complete normally, as today. |

Issue #197 (maintenance, no behaviour change):

- **FR-AMZ-BPROF-008** In `internal/amazon/browser.go`, when `profile()` uses the persistent profile and discarding its stale lock fails, it returns an empty path, no cleanup (`nil`) and that error, the same form as its branch where the profile directory cannot be created.
- **FR-AMZ-BPROF-009** When `profile()` uses the persistent profile and the stale lock is discarded (or there is none), it returns the persistent profile path, a cleanup that does nothing, and no error, as before. The temporary-profile branch is unchanged.
- **FR-AMZ-BPROF-010** The change's diff outside `doc/` is limited to `profile()` in `internal/amazon/browser.go` and, if any, tests of FR-AMZ-BPROF-008–009 under `internal/amazon/`. FR-AMZ-BPROF-001–007 behave as before (a lock that cannot be discarded is still a counted "browser could not start" failure, FR-AMZ-BPROF-006), the unit-test gate (`go vet ./...`, `go test ./...`) passes, and the operator observes no change in the application.

## States and corner cases

- Item states and messages are unchanged (pending, active, retrieval issue, unavailable, stopped).
- When no persistent profile is used (temporary profile), nothing changes.
- Recovery is invisible to the user and needs no action; nothing is shown in the interface.
- Items stopped before the installation stay stopped until restarted with **Refresh price**; that refresh now succeeds when the stop was caused by a stale lock.
- In protected mode the behaviour is the same for the Amazon items of all users.

## Acceptance criteria summary (issue #197)

- With a persistent profile whose stale lock cannot be discarded, `profile()` returns `""`, a `nil` cleanup and a non-nil error (FR-AMZ-BPROF-008).
- With a persistent profile whose lock is discarded or absent, `profile()` returns the profile path, a non-nil cleanup and a `nil` error (FR-AMZ-BPROF-009).
- `git diff --name-only origin/master` outside `doc/` lists only `internal/amazon/browser.go` and, if any, test files under `internal/amazon/`; `gofmt -l internal/amazon` prints nothing; `go vet ./...` and `go test ./...` succeed (FR-AMZ-BPROF-010).

## Open questions

None. Issue #197 states the expected code form and asks for no behaviour change.

## Traceability

- Related: [Amazon human-like checks](../amazon-human-checks/functional.md), [Amazon request stop](../amazon-request-stop/functional.md).
- Technical handoff target: `doc/specifications/amazon-browser-profile/technical.md`.
