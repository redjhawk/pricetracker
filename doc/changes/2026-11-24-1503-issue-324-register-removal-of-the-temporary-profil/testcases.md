# Test cases: register removal of the temporary profile in the TC-003 test of issue #197 (issue #324)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-053 to FR-TEST-HYG-056). There are no use cases: see [no-use-cases.md](no-use-cases.md). How the collector creates and removes its temporary profile stays as specified in [amazon-browser-profile](../../specifications/amazon-browser-profile/functional.md); this change does not alter it.

Conventions:

- **The subtest:** "TC-003 temporary profile" of `TestBrowserProfile_Issue197_TC003_PreparedProfileReturnsItsPath` in `internal/amazon/browser_profile_values_test.go`. It calls `profileOf("")` (the collector's `profile()` with no profile directory), checks the error, the cleanup, the path and that the path is a directory, then calls the collector's cleanup and checks that the path is gone.
- **Added removal:** the removal of the temporary profile that the subtest registers right after `profileOf("")` (for example with `t.Cleanup`), so that it runs once the subtest has ended. How it is registered is a technical choice; the cases check only its effect.
- **Leftover check:** list `pricefollower-browser-*` directories under the system temporary folder (`os.TempDir()`, or `TMPDIR` set to a fresh empty directory for the run) before and after the run; "no leftover" means no new such directory after the run.
- **Run the subtest:** `go test ./internal/amazon -run '^TestBrowserProfile_Issue197_TC003_PreparedProfileReturnsItsPath$' -count=1 -v`. It needs no real browser.
- **Fault injection:** a temporary edit of the subtest or of `BrowserCollector.profile` in `internal/amazon/browser.go`, made only for the run of a case, reverted afterwards and never committed.
- The change only touches test code. No production code, API, database or user interface changes, and there is no user flow, so there are no integration or e2e cases of the application. The cases are a code check, runs of the subtest (some with fault injection) and the gates. The IDs below are this change's own, not those in the subtests' names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-053 | TC-001, TC-002, TC-003 | — |
| FR-TEST-HYG-054 | TC-001, TC-003 | — |
| FR-TEST-HYG-055 | TC-001, TC-004 | — |
| FR-TEST-HYG-056 | TC-005, TC-006 | — |

## Cases

### TC-001: The subtest registers a guarded removal of exactly its temporary profile, and keeps its checks

- Covers: FR-TEST-HYG-053, FR-TEST-HYG-054, FR-TEST-HYG-055
- Level: code check of `internal/amazon/browser_profile_values_test.go`
- Given the change applied on top of `master`
- When the subtest is read
- Then:
  - the added removal is registered immediately after the `profileOf("")` call, before any check that can stop the subtest (`t.Fatal` on the error, on a nil cleanup, or `t.Fatalf` when the path is not a directory);
  - it runs only once the subtest has ended (after all its checks), not inline;
  - it removes only the `path` returned by `profileOf("")`, and does nothing when `path` is empty;
  - the subtest still calls `cleanup()` itself, and still checks right after that call that `path` no longer exists, failing with "the cleanup did not remove the temporary profile" otherwise;
  - the other checks and messages ("cleanup is nil", "temporary profile %q, want a new directory under %s", "the temporary profile was not created: %v") are unchanged.
- Corner cases:
  - the removal does not target `filepath.Dir(path)`, `os.TempDir()`, a glob such as `pricefollower-browser-*`, or any other path;
  - the removal is registered on the subtest's `t`, not the parent test's, so it runs when this subtest ends;
  - the other subtests of the test are unchanged.

### TC-002: A fatal failure in the subtest leaves no temporary profile behind

- Covers: FR-TEST-HYG-053
- Level: unit (Go, `internal/amazon`; fault injection)
- Given the change applied, a leftover check prepared, and one of these temporary edits at a time:
  1. a `t.Fatal("injected")` added right after the check that the temporary profile was created, before `cleanup()` is called;
  2. `BrowserCollector.profile` made to return the created temporary directory with a nil cleanup (the subtest stops on "cleanup is nil");
  3. `BrowserCollector.profile` made to return the created temporary directory together with an error (the subtest stops on the error);
  4. `BrowserCollector.profile` made to create a regular file named `pricefollower-browser-*` under the system temporary folder instead of a directory, and return its path (the subtest stops on "the temporary profile was not created")
- When the subtest is run
- Then the subtest fails with the expected message, and the leftover check finds no new `pricefollower-browser-*` entry under the system temporary folder
- And, with edit 1 applied to the `master` version of the subtest, the leftover check finds one new `pricefollower-browser-*` directory, which shows the change removes the leak
- Corner cases:
  - the temporary profile is not empty when the subtest stops (edit 1 with a file written into it first) and is still removed;
  - the system temporary folder itself and other entries in it are still there after the run;
  - no fault injection remains in the diff (checked by TC-006).

### TC-003: The added removal cannot hide a cleanup that removes nothing

- Covers: FR-TEST-HYG-053, FR-TEST-HYG-054
- Level: unit (Go, `internal/amazon`; fault injection)
- Given the change applied, a leftover check prepared, and the cleanup returned by `BrowserCollector.profile` for a temporary profile temporarily made to do nothing
- When the subtest is run
- Then the subtest fails with "the cleanup did not remove the temporary profile", and the leftover check finds no new `pricefollower-browser-*` directory after the run
- Corner cases:
  - the failure is reported by the check that follows `cleanup()`, so the added removal ran only after it (otherwise the subtest would pass);
  - the failure is reported once, and no other check of the subtest fails;
  - no fault injection remains in the diff (checked by TC-006).

### TC-004: An empty path removes nothing

- Covers: FR-TEST-HYG-055
- Level: unit (Go, `internal/amazon`; fault injection)
- Given the change applied, a marker file created in the current working directory of the test (`internal/amazon`) and in the system temporary folder, and `BrowserCollector.profile` temporarily made to return an empty path with a non-nil cleanup and no error
- When the subtest is run
- Then the subtest fails on its existing checks (the path is not under the system temporary folder, then "the temporary profile was not created"), and after the run both marker files, the current working directory and the system temporary folder still exist
- Corner cases:
  - the same holds when `profile` returns an empty path with an error (the subtest stops on the error and removes nothing);
  - no fault injection or marker file remains after the case (checked by TC-006).

### TC-005: A normal run passes and leaves no temporary profile

- Covers: FR-TEST-HYG-056
- Level: unit (Go, `internal/amazon`, without fault injection)
- Given the change applied and a leftover check prepared
- When the subtest is run, then again with `-count=3`, then together with the other `TestBrowserProfile_*` tests (`-run 'TestBrowserProfile'`) and with `-race`
- Then every run passes, all subtests of `TestBrowserProfile_Issue197_TC003_PreparedProfileReturnsItsPath` pass, and the leftover check finds no new `pricefollower-browser-*` directory
- Corner cases:
  - the added removal of a path the collector's cleanup already removed raises no failure or log on a normal run;
  - repeated runs give the same result.

### TC-006: Only the test file changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-056
- Level: repository check and suite run (deterministic gates: `gofmt -l internal/amazon`, `go vet ./...`, `go test ./...` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` and `git diff master -- internal/amazon/browser_profile_values_test.go` are listed and the gates run
- Then:
  - the only file listed outside `doc/` is `internal/amazon/browser_profile_values_test.go`;
  - its changed hunk is limited to the subtest "TC-003 temporary profile" (the added removal), with no other subtest, test or helper changed;
  - `gofmt -l internal/amazon` prints nothing, and `go vet ./...` and `go test ./...` pass.
- Corner cases:
  - `internal/amazon/browser.go` and the other production Go code, frontend source, `go.mod`, `go.sum`, `package.json`, `tests/` and `API_SPECIFICATION.md` are unchanged, so the operator sees no change in the application;
  - the issue #197 change folder and `doc/todo/issue-197-r1-03.md` are unchanged;
  - no fault injection from TC-002, TC-003 or TC-004 remains;
  - the gates give the same result when run twice.
