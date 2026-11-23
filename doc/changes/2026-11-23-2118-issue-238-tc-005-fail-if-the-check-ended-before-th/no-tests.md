# No new tests for issue #238

No test file is added or changed. The guard that the issue asks for is already on `master` (added by PR #236) in `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile` (`internal/httpapi/amazon_browser_interrupted_test.go`, lines 40–44), and the change must not edit that file (TS-TEST-HYG-031, TS-TEST-HYG-034). A red phase cannot run on behaviour that already exists, so the cases are proved by the existing test, a reverted mutation run and repository checks.

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | The guard `select { case <-stopped: t.Fatal("the check ended before the server stop: the interruption point was missed"); default: }` sits after the `api.fixture.flushed` wait (60 s message unchanged) and before the first `cancel()`. The mutation run below fails with only the guard's message. |
| TC-002 | `go test -count=3 -run TestAmazonBrowserInterrupted_TC005 ./internal/httpapi/` on the unchanged file: 3 runs, all `PASS`. `git diff --quiet master -- internal/httpapi/amazon_browser_interrupted_test.go` succeeds, so no assertion is removed or weakened. |
| TC-003 | After the merge: `gh issue view 238 --json state,stateReason` shows `CLOSED` / `COMPLETED`; the factory's pull request body holds `Closes #238`. |
| TC-004 | `git diff --name-only master -- . ':!doc'` is empty; the deterministic gates (`go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`) run as on `master`. |

## Missed-interruption mutation (run in this stage, reverted)

Mutation: in TC-005, right after the `flushed` wait and before the guard, insert

```go
cancel()
<-stopped
```

so the check has really ended when the guard runs.

Command: `go test -count=1 -run TestAmazonBrowserInterrupted_TC005 ./internal/httpapi/`

Output:

```
--- FAIL: TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile (0.87s)
    amazon_browser_interrupted_test.go:44: the check ended before the server stop: the interruption point was missed
FAIL
FAIL	pricefollower.local/internal/httpapi	0.876s
```

The only failure is the guard's, with the exact message of FR-TEST-HYG-027; none of "the stopped check did not return within 20 s", "cache clear failure logged" or a download or data-file message appears. The mutation was then removed and `git diff --quiet master -- internal/httpapi/amazon_browser_interrupted_test.go` succeeded.
