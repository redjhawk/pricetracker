# Test suite hygiene: functional specification

Status: specified
Source: issue #208 (from review comment R1-03 on PR #206, `doc/todo/issue-204-r1-03.md`).
Scope: repository maintenance. Application behaviour is not affected.

## Purpose

Remove the empty leftover test file `internal/service/zz_scratch_docs_test.go`, left by the issue #204 test stage. It holds only a package clause and a comment saying it is safe to delete, tests nothing, and adds noise to the `service` package.

## Actors

- Developer or agent working on the repository.

## Scope and exclusions

- In scope: deleting `internal/service/zz_scratch_docs_test.go`.
- Excluded: any other test file, production code, API, database or user interface change.
- Excluded: the issue #204 change records under `doc/changes/2026-11-22-2139-issue-204-retrieval-error/`. The factory checks a `tests.lock` only for the issue it is running, and issue #204 is merged, so its lock is no longer checked and stays unchanged as the historical record of that issue.

## Requirements

- **FR-TEST-HYG-001** The file `internal/service/zz_scratch_docs_test.go` no longer exists in the repository.
- **FR-TEST-HYG-002** After the removal, `go vet ./...` and `go test ./...` pass, including `go test ./internal/service/...`.
- **FR-TEST-HYG-003** No other test file is deleted or changed, and the operator observes no change in the application.

## States

Not applicable: there is no user interface, loading, empty or error state.

## Acceptance criteria

- `ls internal/service/zz_scratch_docs_test.go` reports that the file does not exist.
- `go vet ./...` and `go test ./...` succeed.
- The change's diff outside `doc/` is limited to the deletion of `internal/service/zz_scratch_docs_test.go`.
