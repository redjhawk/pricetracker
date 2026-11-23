# Test cases: replace the TC-006 import scan in amazontest_test.go with a dependency check (issue #244)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-009 to FR-TEST-HYG-013). No use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **Test helper package:** `pricefollower.local/internal/amazon/amazontest`. **Server dependencies:** the packages `go list -deps ./cmd/...` reports for a normal (non-test) build. **Dependency check:** the new Go test, run by `go test ./...`, that fails when the test helper package is a server dependency.
- "Fails" means the dependency check reports a test failure; "passes" means it records none.
- Fixture modules are temporary Go modules built under `t.TempDir()` with module path `pricefollower.local`, a `cmd/` program and a minimal stub `internal/amazon/amazontest` package, so the real Go tool lists their dependencies without touching the repository or the network (`GOFLAGS=-mod=mod`, `GOWORK=off`, no external requirements).
- Mutations of the real repository (TC-005) are made in a scratch worktree and never committed.
- No user interface, API or database is involved: narrow screens, keyboard, screen states and Playwright cases do not apply. The existing Playwright suite only has to stay green (TC-006).

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-TEST-HYG-009 | TC-001, TC-003, TC-005 |
| FR-TEST-HYG-010 | TC-002, TC-003, TC-005 |
| FR-TEST-HYG-011 | TC-002, TC-003, TC-005 |
| FR-TEST-HYG-012 | TC-002, TC-003, TC-004 |
| FR-TEST-HYG-013 | TC-004, TC-005, TC-006 |

## Cases

### TC-001 — The helper's unit tests no longer search the source of cmd/ and internal/

- Covers: FR-TEST-HYG-009, FR-TEST-HYG-013
- Level: repository check on the changed test file, plus `go test ./internal/amazon/amazontest/...`
- Given the changed `internal/amazon/amazontest/amazontest_test.go`
- When its source is read and its tests run
- Then no test walks `cmd/` or `internal/` and no test compares file contents with the string `pricefollower.local/internal/amazon/amazontest` (no `filepath.WalkDir` over `../../../cmd` or `../../../internal`, no import-path text search); the package's tests pass
- Corner cases:
  - the issue #208 part of `TestAssertNoDownload_TC006_ChangeLeavesApplicationUntouched` is kept: it still fails when `internal/service/zz_scratch_docs_test.go` exists or cannot be checked;
  - the TC-001 to TC-005 tests of issue #232 are unchanged and still pass;
  - `go test -count=1 ./internal/amazon/amazontest/...` passes (no cached result).

### TC-002 — The check decides from a dependency listing and fails closed on a listing error

- Covers: FR-TEST-HYG-010, FR-TEST-HYG-011, FR-TEST-HYG-012
- Level: unit (Go, the package holding the dependency check; the listing is given as input, the Go tool is not run)
- Given a listing result (the list of import paths and the listing error)
- When the check judges it
- Then:

| Row | Listing | Expected |
| --- | --- | --- |
| a | `pricefollower.local/cmd/pricefollower`, `pricefollower.local/internal/httpapi`, `pricefollower.local/internal/amazon/amazontest`, standard library paths | fails; the message holds `pricefollower.local/internal/amazon/amazontest` and says the server depends on it |
| b | the same without `pricefollower.local/internal/amazon/amazontest` | passes |
| c | `pricefollower.local/internal/amazon` only (the parent package, a prefix of the helper path) | passes |
| d | a path that merely contains the helper path, such as `pricefollower.local/internal/amazon/amazontest/sub` or `pricefollower.local/internal/amazon/amazontestutil` | passes (exact path match only) |
| e | listing error `exit status 1: cmd/pricefollower/main.go:3:2: package x is not in std` and no paths | fails; the message holds the Go tool's error text |
| f | listing error together with a partial list that does not hold the helper path | fails with the Go tool's error; it does not pass on the partial list |

- Corner cases:
  - the helper path appears more than once (several programs): one failure, the message still names the path;
  - surrounding whitespace or a trailing empty line in the listing does not hide the helper path (row a with `"\n"`-separated output);
  - the same listing judged twice gives the same result (no state kept).

### TC-003 — The real Go tool listing: direct and indirect imports fail, test-only and non-server imports pass

- Covers: FR-TEST-HYG-009, FR-TEST-HYG-010, FR-TEST-HYG-011, FR-TEST-HYG-012
- Level: integration (Go, real `go list -deps ./cmd/...` run by the dependency check on fixture modules under `t.TempDir()`)
- Given a fixture module with `cmd/app/main.go` and a stub `internal/amazon/amazontest` package
- When the dependency check runs on the fixture module
- Then:

| Row | Fixture | Expected |
| --- | --- | --- |
| a | `cmd/app/main.go` imports the helper directly | fails; message names `pricefollower.local/internal/amazon/amazontest` and says the server depends on it |
| b | `cmd/app` imports `internal/service`, whose non-test file imports the helper | fails (indirect dependency), same message |
| c | two programs `cmd/app` and `cmd/tool`; only `cmd/tool` depends on the helper | fails (any program under `cmd/`) |
| d | only `cmd/app/main_test.go` and `internal/service/service_test.go` import the helper | passes (test files are not server dependencies) |
| e | `internal/httpapi/httpapitest/helper.go` (non-test) imports the helper; no program imports `httpapitest` | passes |
| f | `cmd/app/main.go` imports a package that does not exist | fails with the Go tool's error text |
| g | `cmd/app/main.go` has a syntax error | fails with the Go tool's error text |
| h | the `go` command cannot be started (empty `PATH` for the listing) | fails with the start error; never passes |

- Corner cases:
  - the fixture holds no helper import at all: passes;
  - the helper is imported under a blank (`_`) or renamed import in a server dependency: fails;
  - when the `go` tool is not available to the test process at all, the case fails rather than skips, since the check must never pass without listing.

### TC-004 — The dependency check passes on the repository as changed and runs in the unit-test gate

- Covers: FR-TEST-HYG-012, FR-TEST-HYG-013
- Level: integration (the dependency check run on the real repository by `go test ./...`)
- Given the repository with the change applied
- When the unit-test gate runs: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`
- Then the dependency check runs as part of `go test ./...` with no extra command or flag, lists the dependencies of `./cmd/...` and passes; all three gate commands succeed
- Corner cases:
  - `go test -count=1 ./...` also runs and passes it (not only a cached result);
  - `go test -v ./...` shows the dependency check test and the issue #208 scratch-file test as run, not skipped;
  - the existing `_test.go` files that import the helper (for example in `internal/amazon` and `internal/httpapi`) do not make the check fail.

### TC-005 — End-to-end mutation of the real repository

- Covers: FR-TEST-HYG-009, FR-TEST-HYG-010, FR-TEST-HYG-011, FR-TEST-HYG-013
- Level: repository check (mutations in a scratch worktree, unit-test gate run after each, never committed)
- Given a scratch worktree of the changed repository
- When, one at a time:
  1. a non-test file `internal/httpapi/httpapitest/helper.go` importing `pricefollower.local/internal/amazon/amazontest` is added (no server package imports `httpapitest`), and `go test ./...` runs;
  2. a non-test file of `cmd/pricefollower` importing the helper (blank import) is added, and `go test ./...` runs;
  3. a non-test file of a package `cmd/pricefollower` already uses (for example `internal/service`) importing the helper is added, and `go test ./...` runs
- Then 1 passes (the old text scan would have failed it); 2 and 3 fail, and the output names `pricefollower.local/internal/amazon/amazontest` and says the server depends on it
- Corner cases:
  - after each mutation is reverted, `go test ./...` passes again;
  - mutations 2 and 3 fail through the dependency check, not through a text search of the source.

### TC-006 — The change is limited to test code and leaves the application untouched

- Covers: FR-TEST-HYG-013
- Level: repository check and suite run (deterministic gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`, and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-status master -- . ':!doc'` is listed and the gates run
- Then the diff outside `doc/` holds only `_test.go` files (and, if the technical specification places the check in a test-only package, that package's files used only by tests); all gates pass; the Playwright suite runs the same test list as on `master`
- Corner cases:
  - no production Go code, frontend source, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` changes, so the operator observes no change in the application;
  - `internal/service/zz_scratch_docs_test.go` is still absent;
  - no existing `_test.go` or `.spec.ts` file is deleted.
