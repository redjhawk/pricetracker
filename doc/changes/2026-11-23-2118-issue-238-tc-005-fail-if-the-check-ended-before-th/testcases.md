# Test cases: TC-005 fails if the check ended before the server stop (issue #238)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-019, FR-TEST-HYG-027 to FR-TEST-HYG-029). No use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **Issue #233 TC-005:** `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` in `internal/httpapi/amazon_browser_interrupted_test.go`, added by PR #236. It is the test this change is about, not a case of this catalogue; the IDs below are this change's own.
- **Guard:** the non-blocking `select` on `stopped`, after the wait on `api.fixture.flushed` and before `cancel()`, that calls `t.Fatal("the check ended before the server stop: the interruption point was missed")`. PR #236 already added it.
- **Missed-interruption mutation:** a temporary, uncommitted change that makes the check end before the server stop, for example making the Amazon collector return as soon as the product page has been answered with the download (after `flushed` is signalled). It is reverted after the run.
- The change is test-suite maintenance: no production code, API, database or user interface changes, so no Playwright test is written and narrow screens, keyboard and screen states do not apply. The user flow is the run of TC-005 itself, so the integration cases run the existing Go test with a real SQLite database, HTTP handlers and a real browser.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-TEST-HYG-019 | TC-003 |
| FR-TEST-HYG-027 | TC-001 |
| FR-TEST-HYG-028 | TC-002 |
| FR-TEST-HYG-029 | TC-003, TC-004 |

## Cases

### TC-001 — A missed interruption point fails TC-005 at once with the required message

- Covers: FR-TEST-HYG-027
- Level: integration (existing Go test with a real SQLite database, HTTP handlers and a real browser), plus a check of the test source
- Given the change applied on top of `master`, and the missed-interruption mutation applied
- When `go test -count=1 -run TestAmazonBrowserInterrupted_TC005 ./internal/httpapi/` runs
- Then TC-005 fails, and its output holds exactly `the check ended before the server stop: the interruption point was missed`
- Corner cases:
  - the message is exactly the one of FR-TEST-HYG-027 (the draft wording of the issue, "interruption point missed", is not accepted);
  - the guard sits after the wait on `api.fixture.flushed` and before the first `cancel()` call, so the server is not stopped before the failure (only the deferred cleanup cancel runs after `t.Fatal`);
  - the failure is the guard's, not a later assertion: the output holds none of "the stopped check did not return within 20 s", "cache clear failure logged" or a download or data-file assertion;
  - the guard has a `default` branch, so it does not wait when the check is still running;
  - the 60 s timeout message for a product page never answered with the download is unchanged;
  - without the mutation (reverted), the same command passes (see TC-002).

### TC-002 — A check still running is stopped and TC-005 passes as before

- Covers: FR-TEST-HYG-028
- Level: integration (existing Go test with a real SQLite database, HTTP handlers and a real browser)
- Given the change applied on top of `master`, without any mutation
- When `go test -count=3 -run TestAmazonBrowserInterrupted_TC005 ./internal/httpapi/` runs
- Then TC-005 passes in every run: after the guard it calls `cancel()`, the stopped check returns within 20 s, no cache clear failure is logged, no download file and no new data file are written, and the next Refresh price of each of the two owners records the normal page (latest price 129.99, each owner sees only their own item, still no download or new data file)
- Corner cases:
  - repeated runs without cached results (`-count=3`) all pass, so the guard does not make the test flaky on a normal run;
  - the assertions after the server stop are the same as on `master` (none is removed or weakened);
  - the 20 s limit for the stopped check is unchanged.

### TC-003 — Issue #238 is closed as completed by the merge

- Covers: FR-TEST-HYG-019, FR-TEST-HYG-029
- Level: repository check (GitHub issue state, `gh issue view 238 --json state,stateReason`, and the pull request body)
- Given the change's pull request merged into `master`
- When the state of issue #238 is read
- Then `state` is `CLOSED` and `stateReason` is `COMPLETED`
- Corner cases:
  - the reason is not `NOT_PLANNED`, since the guard is in place;
  - the pull request links issue #238 with a closing keyword, so the merge itself closes it;
  - an issue already closed as completed stays closed as completed (repeated action);
  - issue #238 is not deleted or transferred, so it stays readable.

### TC-004 — No production code changes and the gates pass

- Covers: FR-TEST-HYG-029
- Level: repository check and suite run (deterministic gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`, and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` is listed and the gates run
- Then the list is empty or holds only `internal/httpapi/amazon_browser_interrupted_test.go`; `go vet ./...` and `go test ./...` pass, and so do the other gates; the Playwright suite runs the same test list as on `master`
- Corner cases:
  - no production Go code, frontend source, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` changes, so the operator observes no change in the application;
  - no other `_test.go` or `.spec.ts` file is added, changed or deleted;
  - the missed-interruption mutation of TC-001 is not part of the diff;
  - the gates give the same result when run twice.
