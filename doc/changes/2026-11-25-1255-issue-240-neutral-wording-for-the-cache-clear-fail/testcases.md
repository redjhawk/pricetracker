# Test cases: neutral wording for the cache-clear failure log line (issue #240)

Sources: [functional specification](../../specifications/amazon-human-checks/functional.md) (FR-AMZ-HCHK-016, with FR-AMZ-HCHK-015 for when the line appears and its cause), use case [UC-AMZ-HCHK-009](../../use-cases/amazon-human-checks/UC-AMZ-HCHK-009-failed-cache-clearing-logged-without-its-reason.md).

Conventions:

- **Fake browser, interruption, failure result, bound:** as in the test cases of issue #233 ([testcases.md](../2026-11-23-1234-issue-233-clear-the-amazon-browser-http-cache-even/testcases.md)): `useFakeBrowser` / `fakeBrowserScript` in `internal/amazon`, `interruptedContext` (time limit = deadline, server stop = cancellation), `assertInterruptedFailure`, `collectInterrupted` (returns within 20 s of the interruption).
- **Clear failures** scripted in the fake browser, per subtest: **error answer** — `Network.clearBrowserCache` answered with the protocol error message `cache clear refused by fake`; **no answer** — never answered, so the clear's 5 s bound is the cause (`no answer within`).
- **Cache log line:** a line of the captured output (`captureOutput`) containing `Amazon browser cache could not be cleared`. Its **message** is the text after the standard `log` date and time prefix.
- **Neutral line:** the message starts with exactly `Amazon browser cache could not be cleared: ` (prefix, colon, space), followed by a non-empty cause that contains the expected cause (`cache clear refused by fake`, or `no answer within`), and contains neither `context canceled` nor `context deadline exceeded`.
- **Old wording:** no line of the captured output contains `after a page not opened`.
- The UI, the API and the stored data do not change: no Playwright case. Narrow screens and keyboard are not relevant. A failed clearing cannot be provoked with the real Chromium (it confirms `Network.clearBrowserCache`), so the failure branch is proved with the fake browser and the integration level proves the success branch through the application.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-HCHK-016 | TC-001, TC-002, TC-003, TC-004 |

| Use case | Test cases |
| --- | --- |
| UC-AMZ-HCHK-009 | TC-001 (main flow 1–6, postcondition "exactly one log line … followed by its real cause"), TC-002 (alternative "check interrupted while the page is still being requested"), TC-003 and TC-004 (alternative "clearing succeeds", postcondition "nothing else changes"), TC-004 (alternative "protected mode") |

## Collector with a fake browser

### TC-001 — A failed clearing after a page not opened is logged with the neutral line

- Covers: FR-AMZ-HCHK-016 (with FR-AMZ-HCHK-015); UC-AMZ-HCHK-009 (main flow steps 1–6, postconditions)
- Level: unit (`internal/amazon`, fake browser)
- Given a fake browser that answers `Page.navigate` with a frame ID and `errorText: "net::ERR_ABORTED"`, and, per subtest:
  - **live check, error answer:** a 2xx `Document` response event (the download offer), the clear answered with the error; live context;
  - **live check, no answer:** as above, the clear never answered; live context;
  - **interrupted after the offer, error answer:** no response event (the check waits after the offer), the clear answered with the error; the context ends 200 ms after `Page.navigate` was answered (server stop)
- When `Collect` runs
- Then:
  - exactly one cache log line, and it is a neutral line with the cause of the subtest;
  - old wording: no line contains `after a page not opened`;
  - the result is the counted failure (`assertNavigationFailure` for the live check, `assertInterruptedFailure` for the interrupted one), and the browser is closed (`Browser.close` received)
- Corner cases: browser error and unresponsive browser (5 s bound); live and interrupted check; repeated action: a second `Collect` on the same collector with the **live check, error answer** script adds exactly one more neutral line (two in total, one per failed clearing) and the same failure.

### TC-002 — A failed clearing after a check interrupted while the page was still requested is logged with the same neutral line

- Covers: FR-AMZ-HCHK-016 (with FR-AMZ-HCHK-015); UC-AMZ-HCHK-009 (alternative "check interrupted while the page is still being requested")
- Level: unit (`internal/amazon`, fake browser)
- Given a fake browser that does not answer `Page.navigate` until it receives `Browser.close`, and per subtest the clear **error answer** or **no answer**
- When `Collect` runs with a context that ends 200 ms after the fake received `Page.navigate` (time limit; then server stop)
- Then:
  - `Page.navigate` was received before the interruption (otherwise the test fails saying the interruption point was missed); the check is bounded and the result is the failure;
  - exactly one cache log line, a neutral line with the cause of the subtest: the cause is the clearing's (`cache clear refused by fake` or `no answer within`), never the interruption of the check;
  - old wording: no line contains `after a page not opened`, and no line about the cache contains `context canceled` or `context deadline exceeded` (`assertNoInterruptionGivenForCache`)
- Corner cases: both interruption kinds × both clear failures; the line is identical in form to TC-001's, whatever started the clearing.

### TC-003 — A successful clearing logs no cache line (existing tests, unchanged)

- Covers: FR-AMZ-HCHK-016 (acceptance "when the clearing succeeds, the log has no line saying the browser cache could not be cleared"); UC-AMZ-HCHK-009 (alternative "clearing succeeds")
- Level: unit (`internal/amazon`, fake browser)
- Existing tests of issue #233 that prove it and must keep passing unchanged: `TestBrowserInterrupted_TC001_ClearConfirmedBeforeClose` (interrupted after the offer), `TestBrowserInterrupted_TC002_PageStillRequestedIsCleared` (interrupted while the page is still requested), `TestBrowserInterrupted_TC006_UninterruptedCheckIsUnchanged` subtest `download` (live check)
- Given the fake browser answers `Network.clearBrowserCache` successfully
- When `Collect` runs
- Then no cache log line (`assertNoCacheLogLine`, which matches the prefix kept by this change, so it also rejects a line with the old wording)
- Corner cases: both trigger paths; both interruption kinds; repeated checks (TC-001 of #233 runs `Collect` twice).

## Through the application

### TC-004 — A check through the application whose clearing succeeds logs no cache line (existing tests; TC-005 of issue #233 extended by issue #358)

- Covers: FR-AMZ-HCHK-016 (acceptance "when the clearing succeeds …", "the rest of FR-AMZ-HCHK-015 is unchanged"); UC-AMZ-HCHK-009 (alternatives "clearing succeeds", "protected mode"; postcondition "nothing else changes")
- Level: integration (`internal/httpapi`, real SQLite, service and handlers, real Chromium collector with its persistent profile; `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` of issue #233, extended by issue #358 with a check after each owner's Refresh price, and the unchanged real-browser `TestBrowserInterrupted_TC004_RealCheckInterruptedAroundADownloadWritesNoFile` in `internal/amazon`)
- Given two Amazon items of two users (protected mode) and a product page offered as a download
- When a check is stopped with the server while the download is offered, then each owner uses **Refresh price** on the normal page
- Then no line contains `Amazon browser cache could not be cleared`, neither after the server stop nor during either owner's Refresh price; no file is written; each item records 12999 cents, as today
- Corner cases: server stop and time limit (real-browser test); items of several users; the next checks are unchanged by the new wording.
