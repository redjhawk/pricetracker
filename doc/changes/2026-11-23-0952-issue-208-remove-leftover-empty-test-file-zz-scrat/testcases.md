# Test cases: remove the leftover empty test file `zz_scratch_docs_test.go` (issue #208)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-001 to FR-TEST-HYG-003). No use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- This change is repository maintenance with no behaviour change. FR-TEST-HYG-003 and the acceptance criteria limit the diff outside `doc/` to the deletion of `internal/service/zz_scratch_docs_test.go`, so **no new test code is written**: adding a test file would itself break FR-TEST-HYG-003. The cases are proved by repository checks and by the existing suites run by the gates (`go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`).
- Consequence for the tests stage: it writes `no-tests.md` (red phase skipped) and lists `internal/service/zz_scratch_docs_test.go` in `delete.txt`. The deletion must land in the tests stage, because the green gate fails on any test file (`_test.go`) changed after the TDD red commit.
- Baseline: the state of the base branch `master` before the change, where the file exists and holds only `package service` and a comment.
- No user interface, API or database is involved: narrow screens, keyboard and empty/error states do not apply.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-TEST-HYG-001 | TC-001 |
| FR-TEST-HYG-002 | TC-002 |
| FR-TEST-HYG-003 | TC-003, TC-004 |

## Cases

### TC-001 — The leftover file no longer exists

- Covers: FR-TEST-HYG-001
- Level: repository check (no automated test; checked on the change's diff)
- Given the base branch, where `internal/service/zz_scratch_docs_test.go` exists
- When the change is applied
- Then `ls internal/service/zz_scratch_docs_test.go` reports that the file does not exist, and `git diff --name-status master -- internal/service/zz_scratch_docs_test.go` shows `D` (deleted)
- Corner cases:
  - `git ls-files 'internal/service/zz_scratch*'` is empty (the file is not left tracked, nor recreated with another content);
  - no untracked copy remains in the working tree (`git ls-files -o --exclude-standard internal/service/` lists nothing named `zz_scratch*`);
  - no Go source still refers to the file (search for `zz_scratch` outside `doc/` finds nothing).

### TC-002 — Vet and the whole Go suite still pass, including the `service` package

- Covers: FR-TEST-HYG-002
- Level: unit and integration (existing Go suites, real SQLite and HTTP handlers, run by `go test ./...`)
- Given the change applied
- When `go vet ./...`, `go test ./internal/service/...` and `go test ./...` are run
- Then each command succeeds with exit code 0 and the `service` package still compiles and passes
- Corner cases:
  - repeated run with the test cache bypassed (`go test -count=1 ./...`): same result, so the pass does not come from a cached run on the base branch;
  - `go build ./...` succeeds (the package does not depend on the deleted file for its package clause; other files in `internal/service/` declare `package service`);
  - `go test -list . ./internal/service/` prints the same test names as on the base branch (the deleted file defined no test, so none is lost).

### TC-003 — No other test file is deleted or changed

- Covers: FR-TEST-HYG-003
- Level: repository check (no automated test; checked on the change's diff)
- Given the base branch `master`
- When the change's diff outside `doc/` is listed (`git diff --name-status master -- . ':!doc'`)
- Then the only entry is `D internal/service/zz_scratch_docs_test.go`
- Corner cases:
  - no file matching `_test.go`, `.spec.ts` or `.test.mjs` is added, modified or renamed;
  - `playwright.config.ts` and `scripts/*.test.mjs` are unchanged;
  - the issue #204 records are unchanged: `git diff master -- doc/changes/2026-11-22-2139-issue-204-retrieval-error/` is empty, including its `tests.lock` (excluded by the specification);
  - no production code (Go or frontend sources), `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` changes.

### TC-004 — The operator observes no change in the application

- Covers: FR-TEST-HYG-003
- Level: e2e (existing Playwright suite in `tests/`) and build
- Given the change applied
- When `npm run build`, `node --test scripts/*.test.mjs` and `npx playwright test` are run
- Then they all pass, with the same set of Playwright tests as on the base branch (`npx playwright test --list` unchanged)
- Corner cases:
  - since the deleted file is a Go test file, it is not part of the server binary or the frontend bundle: the built application is the same as on the base branch;
  - no API endpoint, database schema or page changes (covered by TC-003's diff check).
