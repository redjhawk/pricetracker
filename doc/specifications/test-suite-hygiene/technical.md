# Test suite hygiene: technical specification

Status: TS-TEST-HYG-001–003 ready (issue #208); TS-TEST-HYG-004–009 ready (issue #232)
Requirements: [functional](functional.md). Sources: issue #208, issue #232.

## Design

### Leftover scratch file (issue #208)

Delete `internal/service/zz_scratch_docs_test.go`. On `master` it holds only `package service` and a comment, so it declares no test, helper, import or symbol. Every other file in `internal/service/` declares `package service`, so the package keeps its package clause and test set. Nothing else refers to the file outside `doc/`.

The file is a test file (`_test.go`). The green gate fails on any test file changed after the TDD red commit, so the deletion must land in the **tests stage**: it writes `no-tests.md` (no new test is written, see `testcases.md`) and lists the path in its `delete.txt`. The development stage then has no code to change.

The issue #204 records (`doc/changes/2026-11-22-2139-issue-204-retrieval-error/`, including `tests.lock`) stay unchanged: the factory checks only the running issue's lock, and the functional specification excludes them.

### No-download check fails on unreadable files (issue #232)

Only `internal/amazon/amazontest/amazontest.go` changes (a non-`_test.go` file of a test-support package; the server binary never imports it).

- `downloadFile(path, name string, patterns []*regexp.Regexp) (string, error)`. The name rules (attachment name or a Chromium-derived variant, `download` prefix, partial-download prefixes and suffix) stay first and in the same order, and return `(reason, nil)` without reading the file. Only then does the content rule run `os.ReadFile(path)`:
  - `err == nil`: `("holds the downloaded body", nil)` when the data contains `DownloadMarker`, otherwise `("", nil)`;
  - `errors.Is(err, fs.ErrNotExist)`: `("", nil)`. The file disappeared after it was listed, and a removed parent folder gives the same error;
  - any other error: `("", err)`. The error is returned unchanged; it is the `*fs.PathError` of `os.ReadFile` (for example `open <path>: permission denied`), so it already names the full path and the cause.
- In `AssertNoDownload`, the `filepath.WalkDir` callback returns the error of `downloadFile`. The walk stops, and the existing `t.Fatalf("list the files of %s: %v", root, err)` reports it. The message holds the root, the file's path and the read error, and it is not the `files left by a download` message. Downloads found before the read error are not reported. The test fails either way, and the developer fixes the read error first.
- Unchanged: a missing root and a `fs.ErrNotExist` passed to the callback are ignored; entries that are not regular files (folders, symbolic links, including dangling ones) are skipped and never read; `WalkDir` does not follow symbolic links; the `files left by a download` message and its reasons do not change. The helper keeps no state between calls.
- Update the doc comments of `AssertNoDownload` and `downloadFile` to say that an unreadable file fails the check and that a file that has disappeared is ignored.

Refactoring: changing `downloadFile`'s signature is the change the feature needs. The function is unexported, and only `AssertNoDownload` calls it. The risk is low: the four callers of `AssertNoDownload` (`internal/amazon/browser_navigation_test.go` (2), `internal/amazon/browser_interrupted_test.go`, `internal/httpapi/amazon_browser_navigation_test.go`) keep the same API and are not edited.

## Tiers

- Frontend: not affected. Only Go test-support code changes; no view, component, state or asset.
- Backend: not affected at runtime. `internal/amazon/amazontest` is imported only by `_test.go` files, so the server binary does not change. No service, repository, validation, persistence, migration or concurrency change. `AssertNoDownload` runs after the browser has closed, so this change adds no concurrency.
- API: not affected. No endpoint, request, response or error changes; `API_SPECIFICATION.md` is unchanged.

## Technical IDs

| ID | Requirement | Design |
| --- | --- | --- |
| TS-TEST-HYG-001 | FR-TEST-HYG-001 | The tests stage lists `internal/service/zz_scratch_docs_test.go` in `delete.txt`; the pipeline deletes it. |
| TS-TEST-HYG-002 | FR-TEST-HYG-002 | No replacement code: the deleted file defined nothing, so `go build`, `go vet` and `go test` of `./...` are unaffected. |
| TS-TEST-HYG-003 | FR-TEST-HYG-003 | No other path outside `doc/` changes; the issue #204 change folder is untouched. Issue #232 does not touch `internal/service/`. |
| TS-TEST-HYG-004 | FR-TEST-HYG-004 | `downloadFile` returns any `os.ReadFile` error other than `fs.ErrNotExist`. The `WalkDir` callback returns it, and `AssertNoDownload` fails with `t.Fatalf`. |
| TS-TEST-HYG-005 | FR-TEST-HYG-005 | The read error is passed on unchanged (`*fs.PathError`: operation, full path, cause) into `list the files of <root>: <err>`. |
| TS-TEST-HYG-006 | FR-TEST-HYG-006 | `downloadFile` maps `fs.ErrNotExist` from `os.ReadFile` to `("", nil)`. |
| TS-TEST-HYG-007 | FR-TEST-HYG-007 | The name rules run before any read and keep their order and reasons. The marker rule, missing roots, non-regular entries and not following links are unchanged. |
| TS-TEST-HYG-008 | FR-TEST-HYG-008 | `AssertNoDownload`'s signature does not change, and its callers are untouched. The diff outside `doc/` is limited to `internal/amazon/amazontest/`. |
| TS-TEST-HYG-009 | FR-TEST-HYG-004–007 (testability) | New internal unit tests in `package amazontest` observe failures with a recording `testing.TB` (see below). |

## Files to change

| File | Stage | Change |
| --- | --- | --- |
| `internal/service/zz_scratch_docs_test.go` | tests (issue #208) | delete (via `delete.txt`) |
| `doc/changes/2026-11-23-0952-issue-208-remove-leftover-empty-test-file-zz-scrat/no-tests.md` | tests (issue #208) | explain why no test is added |
| `internal/amazon/amazontest/amazontest_test.go` | tests (issue #232) | new: unit tests of TC-001 to TC-005 |
| `internal/amazon/amazontest/amazontest.go` | development (issue #232) | `downloadFile` returns `(string, error)`; `AssertNoDownload` callback returns it; doc comments |
| `doc/specifications/test-suite-hygiene/functional.md` | development | set the status of FR-TEST-HYG-004–008 to implemented |

No production code, frontend, `API_SPECIFICATION.md`, `go.mod`, `go.sum`, `package.json` or `playwright.config.ts` change. No existing test file changes.

## Test design (issue #232)

`internal/amazon/amazontest/amazontest_test.go`, `package amazontest` (internal, so TC-003 can call `downloadFile`). Standard library only; it must be `gofmt`-clean.

- **Recording TB:** `type recordingTB struct { testing.TB; failed bool; message string }` embedding the real `*testing.T` (so `Helper`, `Name`, `TempDir` etc. work). It overrides `Fatalf`/`Fatal` (record, then `runtime.Goexit()`) and `Errorf`/`Error` (record). A helper `runCheck(t, names, roots...) (failed bool, message string)` runs `AssertNoDownload` in a new goroutine and waits on a channel closed by a `defer`.
- **Unreadable file:** `os.WriteFile`, then `os.Chmod(path, 0)`, with `t.Cleanup` restoring `0o600` (and `0o700` for folders) so `t.TempDir` can be removed. If `os.ReadFile` still succeeds (running as root), `t.Skip("file still readable after chmod 000 (running as root?)")`.
- Expected messages are checked with `strings.Contains`. Read failure: the message contains the full path and `permission denied`, and does not contain `files left by a download`. Downloads: the message contains `files left by a download`, the path and the reason text.
- The red state comes from TC-001: the helper passes today, so the assertions fail. TC-003 calls `downloadFile` with its new two-value result, so the package does not compile until development; this is the expected TDD red.

## Test case mapping

| Test case | Proved by |
| --- | --- |
| TC-001 | `internal/amazon/amazontest/amazontest_test.go`: `TestAssertNoDownloadFailsOnUnreadableFile` (subtests: marker, no marker, nested `Default/Cache/data_1`, second root, clean neighbours, repeated call) |
| TC-002 | `internal/amazon/amazontest/amazontest_test.go`: `TestAssertNoDownloadReportsUnreadableDownloadByName` (`page.html`, `x.crdownload`, `download.html`) |
| TC-003 | `internal/amazon/amazontest/amazontest_test.go`: `TestDownloadFileIgnoresRemovedFile` (`downloadFile` on a removed `Preferences` with or without marker, and under a removed parent, returns `("", nil)`) |
| TC-004 | `internal/amazon/amazontest/amazontest_test.go`: `TestAssertNoDownloadJudgesReadableFiles` (table rows a–j and the corner cases: no roots, several downloads in one message, `page (1).htm`, `page.html.bak`) |
| TC-005 | Unit: `internal/amazon/amazontest/amazontest_test.go`: `TestAssertNoDownloadPassesOnBrowserProfile`. Integration: the existing `internal/amazon/browser_navigation_test.go`, `internal/amazon/browser_interrupted_test.go` and `internal/httpapi/amazon_browser_navigation_test.go`, unchanged, run by `go test ./...` |
| TC-006 | Repository check on the diff (`git diff --name-status master -- . ':!doc'` lists only `internal/amazon/amazontest/`) and the gates: `go vet ./...`, `go test ./...`, `npm run build`, `npx playwright test` |

Issue #208 cases (unchanged record):

| Test case | Proved by |
| --- | --- |
| TC-001 | `git diff --name-status master -- internal/service/zz_scratch_docs_test.go` shows `D`; `git ls-files 'internal/service/zz_scratch*'` is empty |
| TC-002 | `go vet ./...`, `go test ./...` (including `./internal/service/...`) |
| TC-003 | `git diff --name-status master -- . ':!doc'` lists only the deletion |
| TC-004 | `npm run build`, `node --test scripts/*.test.mjs`, `npx playwright test` |

## Manual follow-ups

None.
