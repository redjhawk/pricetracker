# No red phase for issue #253

The change is test code: one `switch` branch and two comments of `internal/amazon/amazontest/server_dependencies_test.go`. That file is locked after this stage, and FR-TEST-HYG-073 allows no other file outside `doc/`, so this stage makes the change itself, as the technical specification says (TS-TEST-HYG-095). On `master` rows f and g already pass, and the change only narrows when they pass, so no test can be red before it. Run 3 shows the red state instead: with `go nosuchcmd`, rows f and g pass on `master` and fail with the change. The development stage writes no code. It only sets the statuses.

## Delivered in `internal/amazon/amazontest/server_dependencies_test.go`

- The `default` branch of the `fixtureCases` loop of `TestFixtureModulesBuild_Issue251_TC001` (rows f and g) is now:

  ```go
  if err == nil {
  	t.Error("invalid fixture builds, want a build failure")
  } else if !strings.Contains(err.Error(), tc.fails) {
  	t.Errorf("build failure %q does not hold %q", err, tc.fails)
  }
  ```

- The comment above the test is updated, and an issue #253 block is added to the header comment. The other blocks are unchanged.
- `buildModule`, `writeModule`, `fixtureEnv`, `fixtureCases`, the `case "", "dependency":` branch, `TC-003 h fixture: plainMain`, both controls, every other message, the dependency check and its helpers and tests have no hunk in `git diff master`. No import changes (`strings` was already imported).

## Runs

Each fault injection was a temporary in-place edit of the test file, reverted before the next step. The `master` run with edit 1 (run 3) used a copy of the `master` file (`fi_master_test.go.txt`) with the edit, passed with `go test -overlay fi_overlay.json`. Both scratch files are listed in `delete.txt`. Unless stated otherwise, the command is `go test -run '^TestFixtureModulesBuild_Issue251_TC001$' -v -count=1 ./internal/amazon/amazontest/`.

1. **`go build` output of rows f and g.** The sandbox refused new scratch module folders, so the output was not captured from a separate shell run. It was recorded instead through a temporary `t.Logf("%v", err)` in the branch. That logs the error `buildModule` returns, which holds the whole combined output of `go build ./...` run with `fixtureEnv` in the `writeModule` layout:
   - row f: `go build ./...: exit status 1: go: finding module for package pricefollower.local/internal/nope` / `cmd/app/main.go:3:8: cannot find module providing package pricefollower.local/internal/nope: module lookup disabled by GOPROXY=off`. It holds `pricefollower.local/internal/nope`.
   - row g: `go build ./...: exit status 1: cmd/app/main.go:3:8: string literal not terminated`. It holds `main.go`.

   Without the log line, all 14 subtests pass with the change, and `-count=3` passes. No network is used (`GOPROXY=off`, `GOWORK=off`, `GOTOOLCHAIN=local`).
2. **Fault injections with the change, one at a time:**
   1. `buildModule` table `{{"nosuchcmd", "./..."}, {"vet", "./..."}}`: rows f and g FAIL with `build failure "go nosuchcmd ./...: exit status 2: go nosuchcmd: unknown command\nRun 'go help' for usage." does not hold "pricefollower.local/internal/nope"`, and the same message for `"main.go"`. As expected, every other subtest fails too.
   2. Row f `fails` = `pricefollower.local/internal/other`: only row f FAILS, with `build failure "go build ./...: ... pricefollower.local/internal/nope ..." does not hold "pricefollower.local/internal/other"`.
   3. Row g `fails` = `other.go`: only row g FAILS, with `build failure "go build ./...: exit status 1: cmd/app/main.go:3:8: string literal not terminated" does not hold "other.go"`.
   4. Row f `cmd/app/main.go` = `plainMain`: only row f FAILS, with `invalid fixture builds, want a build failure`.

   Without the edits the test passes (run 1).
3. **Edit 1 on the `master` version (overlay):** rows f and g PASS, and every other subtest fails. So the old check accepted any failure.
4. **Comparison with `master`:** the `-v` runs of `TestFixtureModulesBuild_Issue251_TC001`, and of `TestListServerDependencies_TC003_FixtureModules`, `TestCheckServerDependencies_TC002_JudgesListing` and `TestServerDoesNotDependOnTestHelper`, were made before the edit and after it:
   - fixture build test: the same 14 subtests (rows a to g, three corners, `TC-003 h fixture: plainMain`, two controls), all PASS on both.
   - listing tests: the same 26 tests and subtests, all PASS on both, rows f, g and h included.
5. **Repository checks and gates:**
   - `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/amazontest/server_dependencies_test.go`, and no fault injection or log line is left in it.
   - `gofmt -l internal/amazon` prints nothing.
   - `go vet ./...` passes.
   - `go test -count=1 ./...` passes, with every package `ok`, `internal/amazon/amazontest` included.

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | Subtests `TC-003 f` and `TC-003 g` of `TestFixtureModulesBuild_Issue251_TC001`; run 1 (recorded outputs, `-count=3`) |
| TC-002 | Code check of the `default` branch (above): `err == nil` fails, otherwise `tc.fails` is required, and the message names `tc.fails` and holds `err`. The valid-row branch checks no text |
| TC-003 | Runs 2 and 3 |
| TC-004 | Run 4; code check: `buildModule`, `fixtureCases`, the valid-row branch, `TC-003 h fixture: plainMain` and both controls have no hunk in the diff |
| TC-005 | Run 4; code check: no hunk in the dependency check, its helpers, `writeModule` or `fixtureEnv` |
| TC-006 | Run 5. The development stage runs the gates again |
