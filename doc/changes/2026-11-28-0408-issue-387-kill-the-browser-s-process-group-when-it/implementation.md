# Issue #387: implementation record

Change: `kill()` added before `t.Fatal("the browser exited before holding the profile")` in the `case <-exited:` branch of `KillBrowserOnProfile` (`internal/amazon/amazontest/browser_profile.go`). Nothing else changes outside `doc/`.

`K` is `go test -count=1 -run '^TestKillBrowserOnProfile_' -v ./internal/amazon/amazontest/`, run with Chromium.

## Run 1 (TC-001, TC-002, TC-004)

- `K`: pass. `Issue196_TC005_LeavesTheLock` (2.39 s), `Issue196_TC006_ExitBeforeHolding` (0.06 s), `Issue196_TC007_NeverHolding` (30.02 s), `Issue196_TC008_LeftoverAndMissingLock` (6.05 s), `Issue387_TC001_ExitBeforeHoldingKillsTheGroup` (0.05 s).
- `K` with `-count=2`: pass (`ok ... 77.318s`).

## Run 2: fault injection (TC-003), not committed

- The `kill()` line was removed and `go test -count=1 -run '^TestKillBrowserOnProfile_Issue(387|196_TC006)' -v ./internal/amazon/amazontest/` run: both subtests of the issue #387 test failed after the 5 s wait with `processes left after the early exit (child <pid>): ["<pid>: sh -c sleep 60; : <profile> "]`; the three TC-006 rows passed.
- `pgrep -af 'sleep 60'` afterwards listed only the shell running the command itself, no process started by the test.
- The line was restored; `git diff` showed only the one added line; the same run passed again.

## Run 3 (TC-005)

- `go test -count=1 ./...` passes, including `internal/amazon/...`, `internal/service/...` and `internal/httpapi/...`.
- The only test file changed against `origin/master` is `internal/amazon/amazontest/browser_profile_test.go`, whose diff adds a single function, `TestKillBrowserOnProfile_Issue387_TC001_ExitBeforeHoldingKillsTheGroup`; so the test names are those of `master` plus the new test.

## Run 4 (TC-006, TC-007)

- `git diff --name-only origin/master -- . ':!doc'`: `internal/amazon/amazontest/browser_profile.go`, `internal/amazon/amazontest/browser_profile_test.go`.
- `gofmt -l .`: empty. `go vet ./...`: pass. `go test -count=1 ./...`: pass, including `TestServerDoesNotDependOnTestHelper`.
- `npm run build`: pass. `npx playwright test`: 208 passed, 1 skipped.
