# Functional specification: Amazon browser profile recovery

Status: ready (issue #180)
Owner: functional specification agent
User decision/reference: GitHub issue #180 ("Clear stale Chromium Singleton files in the persistent Amazon browser profile"), from review comment R1-02 on #179.

## Purpose and scope

Amazon checks go through a real web browser ([FR-AMZ-HCHK-001](../amazon-human-checks/functional.md)) that keeps its profile (cookies) in the data directory. When that browser ends abruptly (killed when a check is closed or stopped, crash, server stop during a check), it leaves lock marks in the profile. If the data directory is later used on another machine or container (moved server, new container), the browser refuses the profile: every Amazon check fails with "browser could not start" and, after 5 failures, every Amazon item is stopped ([FR-AMZ-STOP-003](../amazon-request-stop/functional.md)). The user asks that such leftovers no longer prevent Amazon checks.

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

## States and corner cases

- Item states and messages are unchanged (pending, active, retrieval issue, unavailable, stopped).
- When no persistent profile is used (temporary profile), nothing changes.
- Recovery is invisible to the user and needs no action; nothing is shown in the interface.
- Items stopped before the installation stay stopped until restarted with **Refresh price**; that refresh now succeeds when the stop was caused by a stale lock.
- In protected mode the behaviour is the same for the Amazon items of all users.

## Open questions

None.

## Traceability

- Related: [Amazon human-like checks](../amazon-human-checks/functional.md), [Amazon request stop](../amazon-request-stop/functional.md).
- Technical handoff target: `doc/specifications/amazon-browser-profile/technical.md`.
