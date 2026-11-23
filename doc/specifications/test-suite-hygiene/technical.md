# Test suite hygiene: technical specification

Status: ready
Requirements: [functional](functional.md). Source: issue #208.

## Design

Delete `internal/service/zz_scratch_docs_test.go`. On `master` it holds only `package service` and a comment, so it declares no test, helper, import or symbol. Every other file in `internal/service/` declares `package service`, so the package keeps its package clause and test set. Nothing else refers to the file outside `doc/`.

The file is a test file (`_test.go`). The green gate fails on any test file changed after the TDD red commit, so the deletion must land in the **tests stage**: it writes `no-tests.md` (no new test is written, see `testcases.md`) and lists the path in its `delete.txt`. The development stage then has no code to change.

The issue #204 records (`doc/changes/2026-11-22-2139-issue-204-retrieval-error/`, including `tests.lock`) stay unchanged: the factory checks only the running issue's lock, and the functional specification excludes them.

## Tiers

- Frontend: not affected. No view, component, state or asset changes; the file is not part of the frontend bundle.
- Backend: not affected at runtime. A `_test.go` file is never compiled into the server binary; no service, repository, validation, persistence, migration or concurrency change.
- API: not affected. No endpoint, request, response or error changes; `API_SPECIFICATION.md` is unchanged.

## Technical IDs

| ID | Requirement | Design |
| --- | --- | --- |
| TS-TEST-HYG-001 | FR-TEST-HYG-001 | The tests stage lists `internal/service/zz_scratch_docs_test.go` in `delete.txt`; the pipeline deletes it. |
| TS-TEST-HYG-002 | FR-TEST-HYG-002 | No replacement code: the deleted file defined nothing, so `go build`, `go vet` and `go test` of `./...` are unaffected. |
| TS-TEST-HYG-003 | FR-TEST-HYG-003 | No other path outside `doc/` changes; the issue #204 change folder is untouched. |

## Files to change

| File | Stage | Change |
| --- | --- | --- |
| `internal/service/zz_scratch_docs_test.go` | tests | delete (via `delete.txt`) |
| `doc/changes/2026-11-23-0952-issue-208-remove-leftover-empty-test-file-zz-scrat/no-tests.md` | tests | explain why no test is added |
| `doc/specifications/test-suite-hygiene/functional.md` | development | optional: set `Status: implemented` |

No production code, test code, `API_SPECIFICATION.md`, `go.mod`, `package.json` or `playwright.config.ts` change.

## Test case mapping

No new test file is written: adding one would break FR-TEST-HYG-003. Each `TC-` is proved by a repository check or by the existing suites the gates run.

| Test case | Proved by |
| --- | --- |
| TC-001 | repository check on the diff: `git diff --name-status master -- internal/service/zz_scratch_docs_test.go` shows `D`; `git ls-files 'internal/service/zz_scratch*'` is empty |
| TC-002 | existing Go suites run by the gates: `go vet ./...`, `go test ./...` (including `./internal/service/...`) |
| TC-003 | repository check on the diff: `git diff --name-status master -- . ':!doc'` lists only the deletion |
| TC-004 | existing suites run by the gates: `npm run build`, `node --test scripts/*.test.mjs`, `npx playwright test` |

## Refactoring

None.

## Manual follow-ups

None.
