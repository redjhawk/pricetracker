# Test cases: make AssertNoDownload fail on unreadable files (issue #232)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-004 to FR-TEST-HYG-008; FR-TEST-HYG-003 reworded to name issue #208). No use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- The code under test is the test helper `AssertNoDownload` (the no-download check) in `internal/amazon/amazontest`. Its unit tests live in the same package and build their folders under `t.TempDir()`.
- A failure of the check is observed, not suffered: the tests pass a recording `testing.TB` (for example a struct embedding `testing.TB` whose `Fatalf`/`Errorf` record the message and stop the call, run in its own goroutine) and assert whether it failed and with which message.
- "Fails" means the check fails the test (`Fatalf`); "passes" means it returns without recording a failure.
- An unreadable file is made with `chmod 000` and restored to `0o600` in `t.Cleanup`, so `t.TempDir` can be removed. When the file is still readable after `chmod` (tests run as root), the case is skipped with a message rather than passing silently.
- No user interface, API or database is involved: narrow screens, keyboard and screen states do not apply.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-TEST-HYG-003 | TC-006 |
| FR-TEST-HYG-004 | TC-001, TC-002 |
| FR-TEST-HYG-005 | TC-001 |
| FR-TEST-HYG-006 | TC-003 |
| FR-TEST-HYG-007 | TC-002, TC-004 |
| FR-TEST-HYG-008 | TC-005, TC-006 |

## Cases

### TC-001 — An unreadable file fails the check with its path and the read error

- Covers: FR-TEST-HYG-004, FR-TEST-HYG-005
- Level: unit (Go, `internal/amazon/amazontest`)
- Given a checked folder holding a regular file `Preferences` (a name no download rule matches) that holds `DownloadMarker` and cannot be read (mode `000`)
- When `AssertNoDownload(tb, []string{"page.html"}, folder)` runs
- Then the check fails; the failure message holds the full path of `Preferences` and the read error text (`permission denied`); the file is not missing from the result as if it were clean
- Corner cases:
  - the unreadable file holds no marker (any content): the check still fails, since an unreadable file is never treated as free of a downloaded body;
  - the unreadable file is nested in a subfolder (`Default/Cache/data_1`): the check fails and names the nested path;
  - the first root is clean and the unreadable file is in the second root: the check fails (later roots are not skipped);
  - readable clean files sit next to the unreadable file: the check still fails;
  - the message is not the "files left by a download" message: the read failure is reported as such, not as a download;
  - repeated call on the same folder: fails again with the same message (no state kept between calls).

### TC-002 — An unreadable file whose name already marks a download is reported as a download

- Covers: FR-TEST-HYG-007, FR-TEST-HYG-004
- Level: unit (Go, `internal/amazon/amazontest`)
- Given a checked folder holding an unreadable (mode `000`) regular file named `page.html`, attachment name `page.html`
- When `AssertNoDownload(tb, []string{"page.html"}, folder)` runs
- Then the check fails with the "files left by a download" message naming the path and the reason "named after the attachment", whatever the content
- Corner cases:
  - unreadable `x.crdownload`: reported as "partial download";
  - unreadable `download.html`: reported as "default download name".

### TC-003 — A file that disappears before it is read is ignored

- Covers: FR-TEST-HYG-006
- Level: unit (Go, `internal/amazon/amazontest`)
- Given a path inside a checked folder whose file was listed and then removed, so it no longer exists when read (the race is made deterministic by applying the content rule directly to the removed file's path with a name no download rule matches)
- When the content rule judges it
- Then it reports neither a download nor a read error, so the no-download check would neither fail nor report it
- Corner cases:
  - the removed file held `DownloadMarker`: still ignored (nothing left on disk);
  - the parent folder was removed too: still ignored;
  - a removed file whose name matches a download rule is out of scope (the name rule does not read the file; covered by TC-004).

### TC-004 — Readable files, missing folders and symbolic links are judged as before

- Covers: FR-TEST-HYG-007
- Level: unit (Go, `internal/amazon/amazontest`, table of cases)
- Given a checked folder holding one readable regular file per row, attachment names `page.html` and `B0PRICE001`
- When `AssertNoDownload` runs on it
- Then:

| Row | Content of the folder | Expected |
| --- | --- | --- |
| a | `page.html` (no marker) | fails, "named after the attachment" |
| b | `page (1).html` | fails, "named after the attachment" |
| c | `B0PRICE001.html` (extension added) | fails, "named after the attachment" |
| d | `download` and `downloaded.bin` | fails, "default download name" |
| e | `x.crdownload`, `Unconfirmed 123.crdownload`, `.com.google.Chrome.abc`, `.org.chromium.Chromium.abc` (one per run) | fails, "partial download" |
| f | `Cookies` holding `DownloadMarker` | fails, "holds the downloaded body" |
| g | `Preferences` with ordinary content, an empty file, an empty subfolder | passes |
| h | the root does not exist | passes |
| i | a symbolic link `Default/link` to a file outside the root that holds `DownloadMarker` | passes (link not followed) |
| j | a symbolic link to a folder outside the root holding `page.html` | passes (link not followed) |

- Corner cases:
  - no roots at all: passes;
  - several downloads in one folder: one failure listing every path with its reason;
  - `page (1).htm` or `page.html.bak` (not a Chromium-derived name, no marker): passes.

### TC-005 — The check passes on a normal browser profile, and the tests that use it still pass

- Covers: FR-TEST-HYG-008
- Level: unit (Go, `internal/amazon/amazontest`) and integration (existing browser tests in `internal/amazon` with real Chromium, persistent profile and isolated home)
- Given a folder laid out like a closed Chromium profile: nested readable files (`Default/Preferences`, `Default/Cache/Cache_Data/index`), an empty file, and Chromium's `SingletonLock`-style symbolic link to a non-existent target
- When `AssertNoDownload(tb, []string{"page.html"}, profile, home)` runs, with `home` an isolated home without a Downloads folder
- Then the check passes; and the existing tests that call `AssertNoDownload` (the issue #217 and #233 browser tests, `browser_interrupted_test.go`) still pass on a normal run
- Corner cases:
  - the dangling symbolic link is neither followed nor read, so it gives no read error;
  - `go test -count=1 ./internal/amazon/...` passes (no cached result);
  - when Chromium is not installed the browser tests skip as before; the unit part still runs.

### TC-006 — The change is limited to test code and leaves issue #208 and the application untouched

- Covers: FR-TEST-HYG-008, FR-TEST-HYG-003
- Level: repository check and suite run (deterministic gates: `go vet ./...`, `go test ./...`, `npm run build`, `npx playwright test`, and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-status master -- . ':!doc'` is listed and the gates run
- Then the diff outside `doc/` holds only files of `internal/amazon/amazontest` (the helper and its `_test.go`); `go vet ./...` and `go test ./...` pass; the build and the Playwright suite pass with the same test list as on `master`
- Corner cases:
  - `internal/service/zz_scratch_docs_test.go` is still absent and no file under `internal/service/` is added or changed (FR-TEST-HYG-003 still holds);
  - no production Go code, frontend source, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` changes, so the operator observes no change in the application;
  - no existing `_test.go` or `.spec.ts` file is deleted.
