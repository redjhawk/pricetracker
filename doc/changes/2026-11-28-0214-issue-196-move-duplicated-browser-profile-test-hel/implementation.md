# Implementation notes: issue #196

Added `internal/amazon/amazontest/browser_profile.go` with `StaleLock`, `BrowserProcesses`, `KillBrowserOnProfile` and `ProfileHandler` (TS-TEST-HYG-150–153). It imports only the standard library. The locked caller edits of the tests stage are unchanged.

## TC-010 greps

- `func (writeStaleLock|…|browserProfileServer)\(` in `*.go`: no match.
- `"/proc"` or `org.chromium.Chromium.gone` in `*.go` under `internal` and `cmd`: only `internal/amazon/amazontest/browser_profile.go` and `internal/amazon/amazontest/browser_profile_test.go`.
- `SetCookie|aod-none.html|"/gp/aod/ajax|"/static/` in the service and HTTP profile test files: no match.
- Word grep of the eight former names in `*.go`: no match.

## TC-012 and TC-013 runs

- `go test -count=1 ./...`: every package passes, including `internal/amazon/amazontest` (TC-001 to TC-009; TC-007 takes about 30 s), `internal/amazon` (the `TestBrowserProfile_*`, `TestBrowserCollector_*` and `TestBrowserRetrieval_*` tests), `internal/service` and `internal/httpapi`.
- `go test -count=2 -run 'TestBrowserProfile_|TestAmazonBrowserProfile_' ./internal/amazon/ ./internal/service/ ./internal/httpapi/`: passes.
- `git diff master` on the five caller files adds or removes no `func Test…`, so the test names are those of `master`.

## Fault injections not run

- TC-014 (`PRICEFOLLOWER_BROWSER_PATH=/bin/true go test … ./internal/service/`): this stage's sandbox needs approval for a command that sets an environment variable, so it was not run. Fail-fast on an early exit is proved by `TestKillBrowserOnProfile_Issue196_TC006_ExitBeforeHolding`, which passes, and the service calls that helper.
- TC-013 (temporary `sh -c 'sleep 30; :' <profile>` before `assertNothingLeft`): this needs a temporary edit of a locked test file, which the development stage may not make. It was not run. `assertNothingLeft` keeps its message and now calls `amazontest.BrowserProcesses`, whose entry format is proved by `TestBrowserProcesses_Issue196_TC004_ListsTheProfileProcesses`.

## Round 1 (blocking comment R1-02)

The tests stage widened the window of TC-008 row 2 to `sleep 0.3` in the locked test. The TDD red commit had reset `browser_profile.go` to stubs, so the round 0 implementation is restored unchanged. `go test -count=3 -run TC008 ./internal/amazon/amazontest/` passes each time (row 2 in about 0.5 s).

## Gates

`gofmt -l .` is empty, `go vet ./...` passes, `go test ./...` passes, `npm run build` passes and `npx playwright test` passes (208 passed, 1 skipped). `go doc` needed approval in the sandbox and was not run; every new exported helper has a doc comment.
