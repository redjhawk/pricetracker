# Test suite hygiene: functional specification

Status: FR-TEST-HYG-001–003 implemented (issue #208); FR-TEST-HYG-004–008 specified (issue #232)
Source: issue #208 (from review comment R1-03 on PR #206, `doc/todo/issue-204-r1-03.md`); issue #232 ("Make AssertNoDownload fail on unreadable files", from review comment R1-02 on PR #231, `doc/todo/issue-217-r1-02.md`).
Scope: repository maintenance. Application behaviour is not affected.

## Purpose

- Issue #208: remove the empty leftover test file `internal/service/zz_scratch_docs_test.go`, left by the issue #204 test stage. It holds only a package clause and a comment saying it is safe to delete, tests nothing, and adds noise to the `service` package.
- Issue #232: make the test check "no file left by a download" (`AssertNoDownload` in `internal/amazon/amazontest`) fail closed. Today a file it cannot read is taken as "not a download", even if it holds the downloaded body, so the tests that prove FR-AMZ-HCHK-014 and FR-AMZ-HCHK-015 ([Amazon human-like checks](../amazon-human-checks/functional.md)) can pass when they should not.

## Actors

- Developer or agent working on the repository, who runs the test suite and reads its results.

## Terms

- **No-download check:** the test helper `AssertNoDownload`, which looks through given folders (the Amazon browser profile, the home folder of the account running the browser) after the browser has closed and fails the test when a file left by a download remains.
- **Content rule:** the part of the no-download check that reads a file to see whether it holds the downloaded body (`DownloadMarker`). It applies to a regular file whose name does not already identify it as a download.

## Scope and exclusions

- In scope (issue #208): deleting `internal/service/zz_scratch_docs_test.go`.
- In scope (issue #232): what the no-download check does when a file cannot be read.
- Excluded: production code, API, database and user interface changes.
- Excluded: the other rules of the no-download check (which names and contents count as a download, missing folders, symbolic links), which stay as they are.
- Excluded: the issue #204 change records under `doc/changes/2026-11-22-2139-issue-204-retrieval-error/`. The factory checks a `tests.lock` only for the issue it is running, and issue #204 is merged, so its lock is no longer checked and stays unchanged as the historical record of that issue.

## Requirements

### Leftover scratch file (issue #208)

- **FR-TEST-HYG-001** The file `internal/service/zz_scratch_docs_test.go` no longer exists in the repository.
- **FR-TEST-HYG-002** After the removal, `go vet ./...` and `go test ./...` pass, including `go test ./internal/service/...`.
- **FR-TEST-HYG-003** No other test file is deleted or changed by issue #208, and the operator observes no change in the application.

### No-download check fails on unreadable files (issue #232)

- **FR-TEST-HYG-004** When the content rule meets a file under a checked folder that cannot be read (for example permission denied or an I/O error), the no-download check fails the test. The file is never treated as free of a downloaded body.
- **FR-TEST-HYG-005** The failure message of FR-TEST-HYG-004 names the path of the file that could not be read and the read error.
- **FR-TEST-HYG-006** A file that is listed but no longer exists when the check reads it (it disappeared in between) is ignored: it does not fail the test and is not reported as a download.
- **FR-TEST-HYG-007** Readable files are judged as before: a file whose name identifies it as a download (attachment name, uniquified attachment name, `download` prefix, partial-download name) is reported as a download whatever its content, a readable file holding `DownloadMarker` is reported as a download, and any other readable file passes. Missing folders are still ignored and symbolic links are still not followed.
- **FR-TEST-HYG-008** The tests that use the no-download check still pass on a normal run, `go vet ./...` and `go test ./...` pass, and the operator observes no change in the application.

## States

Not applicable: there is no user interface, loading, empty or error state. The only observable outputs are test results and test failure messages.

## Acceptance criteria

Issue #208:

- `ls internal/service/zz_scratch_docs_test.go` reports that the file does not exist.
- `go vet ./...` and `go test ./...` succeed.
- The change's diff outside `doc/` is limited to the deletion of `internal/service/zz_scratch_docs_test.go`.

Issue #232:

- With a checked folder holding a regular file that does not match any download name and cannot be read, the no-download check fails the test with a message holding that file's path and the read error (FR-TEST-HYG-004, FR-TEST-HYG-005).
- With a checked folder holding a file that is removed after the folder is listed and before it is read, the no-download check does not fail the test (FR-TEST-HYG-006).
- With readable files only, the check gives the same result as before for each rule of FR-TEST-HYG-007: a file named after the attachment fails, a file holding `DownloadMarker` fails, a missing folder and an unrelated readable file pass.
- `go vet ./...` and `go test ./...` succeed, and the change's diff outside `doc/` is limited to test code (FR-TEST-HYG-008).

## Open questions

None. Issue #232 states the expected behaviour: fail on a read error, keep ignoring files that disappear between listing and reading.
