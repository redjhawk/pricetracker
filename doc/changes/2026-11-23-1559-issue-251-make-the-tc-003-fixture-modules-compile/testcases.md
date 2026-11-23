# Test cases: make the TC-003 fixture modules compile (issue #251)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-014 to FR-TEST-HYG-017). No use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **Fixture modules:** the temporary Go modules that `TestListServerDependencies_TC003_FixtureModules` in `internal/amazon/amazontest/server_dependencies_test.go` writes under `t.TempDir()` (module `pricefollower.local`, a stub `internal/amazon/amazontest` package, and the files of each case).
- **Valid fixtures:** the fixtures that stand for compilable code: issue #244 TC-003 rows a, b, c, d, e, the corners "no helper import at all", "blank import of the helper in a program" and "renamed import of the helper in a server dependency", and the `plainMain` module used by row h. **Invalid fixtures:** rows f (missing package) and g (syntax error), which exist to make the listing fail.
- **Offline environment:** `GOFLAGS=-mod=mod GOWORK=off GOTOOLCHAIN=local GOPROXY=off` (the existing `fixtureEnv`).
- "Same result as before" means the result each case gives on `master` before this change.
- No user interface, API or database is involved: narrow screens, keyboard, screen states and Playwright cases do not apply. The existing Playwright suite only has to stay green (TC-004).

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-TEST-HYG-014 | TC-001 |
| FR-TEST-HYG-015 | TC-001, TC-002 |
| FR-TEST-HYG-016 | TC-003 |
| FR-TEST-HYG-017 | TC-004 |

## Cases

### TC-001 — Every valid fixture module builds offline

- Covers: FR-TEST-HYG-014, FR-TEST-HYG-015
- Level: integration (Go, real `go build ./...` run in each fixture module under `t.TempDir()`, offline environment)
- Given each valid fixture module, written exactly as the TC-003 test writes it (same stub helper, same files)
- When `go build ./...` runs in the fixture module with the offline environment
- Then it succeeds for every valid fixture; on failure the test names the fixture case and shows the Go tool's output
- Corner cases:
  - the fixtures that use a name of the helper (row a, row c `cmd/tool`, the renamed-import corner) build, so every name they use is declared by the stub or no longer used;
  - the renamed-import corner builds while still importing the helper under a name other than `amazontest` and using that name (the case it is named after is kept, FR-TEST-HYG-015);
  - the build check is not vacuous: the invalid fixtures f and g fail `go build ./...`, and a module whose program uses a name the stub does not declare (the situation before this change) fails it too;
  - the build needs no network, module download or toolchain download (`GOPROXY=off`, `GOTOOLCHAIN=local`);
  - the build check runs in every `go test ./...` run with no extra flag, and gives the same result when run twice (`-count=1`).

### TC-002 — Each valid fixture keeps its case and its dependency check result

- Covers: FR-TEST-HYG-015
- Level: integration (Go, real `go list -deps ./cmd/...` run by the dependency check on the fixture modules; the existing TC-003 test of issue #244)
- Given the valid fixture modules after the change
- When the dependency check runs on each one with the offline environment
- Then each gives the same result as before:

| Fixture (case it is named after) | Expected |
| --- | --- |
| a: a program imports the helper directly | fails; message names `pricefollower.local/internal/amazon/amazontest` and says the server depends on it |
| b: a program imports the helper through another package | fails, same message |
| c: only the second program (`cmd/tool`) depends on the helper | fails, same message |
| d: only test files import the helper | passes; the listing holds `pricefollower.local/cmd/app` |
| e: a test-support package no program imports uses the helper | passes; the listing holds `pricefollower.local/cmd/app` |
| corner: no helper import at all | passes; the listing holds `pricefollower.local/cmd/app` |
| corner: blank import of the helper in a program | fails, same message |
| corner: renamed import of the helper in a server dependency (`internal/service`) | fails, same message |

- Corner cases:
  - every case listed in FR-TEST-HYG-015 still has exactly one fixture, under the same subtest name;
  - in row a and row c the helper is still imported by a non-test file of a program under `cmd/`; in row b and the renamed corner it is still imported by `internal/service`, which `cmd/app` imports; in row d it is still imported only by `_test.go` files;
  - the renamed corner still uses a renamed (not blank) import;
  - for the failing cases, `go list` itself reports no error (the failure comes from the dependency, not from the listing).

### TC-003 — The listing-failure cases keep failing with the same messages

- Covers: FR-TEST-HYG-016
- Level: integration (Go, real `go list` on the invalid fixtures and with the `go` command unreachable)
- Given the invalid fixtures f and g, and the `plainMain` fixture with an empty `PATH` (row h)
- When the dependency check runs on each one
- Then each fails as a listing failure, not as a dependency failure, with a message holding:

| Row | Situation | Message holds |
| --- | --- | --- |
| f | a program imports `pricefollower.local/internal/nope`, which does not exist | `exit status` and `pricefollower.local/internal/nope` |
| g | `cmd/app/main.go` has a syntax error | `exit status` and `main.go` |
| h | the `go` command cannot be started | `executable file not found` |

- Corner cases:
  - none of the three messages says the server depends on the helper;
  - a change to the stub helper (for example a new declaration) does not make f or g pass or change their messages;
  - the check never passes without having listed the dependencies.

### TC-004 — Gates pass and the change is limited to the helper's test code

- Covers: FR-TEST-HYG-017
- Level: repository check and suite run (deterministic gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`, and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-status master -- . ':!doc'` is listed and the gates run
- Then the diff outside `doc/` holds only files matching `internal/amazon/amazontest/*_test.go`; `go vet ./...` and `go test ./...` pass, and so do the other gates; the Playwright suite runs the same test list as on `master`
- Corner cases:
  - `go test -count=1 ./internal/amazon/amazontest/...` passes (no cached result), including `TestServerDoesNotDependOnTestHelper` on the real repository and the issue #244 TC-002 unit cases;
  - no production Go code, frontend source, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` changes, and no file is added to the real `internal/amazon/amazontest` package outside `_test.go`, so the operator observes no change in the application;
  - no existing `_test.go` or `.spec.ts` file is deleted, and `internal/service/zz_scratch_docs_test.go` is still absent.
