# Test cases: return explicit zero values from BrowserCollector.profile when stale lock removal fails (issue #197)

Sources: functional specification [amazon-browser-profile](../../specifications/amazon-browser-profile/functional.md) (FR-AMZ-BPROF-008 to FR-AMZ-BPROF-010). There are no new use cases: see [no-use-cases.md](no-use-cases.md). UC-AMZ-BPROF-001 and UC-AMZ-BPROF-002 describe the Amazon check flows that go through `profile()`. This change does not alter them, and TC-004 re-proves them.

Conventions:

- **`profile()`:** `(*BrowserCollector).profile` in `internal/amazon/browser.go`. It returns `(path string, cleanup func(), err error)`. The unit cases call it directly from a test in package `amazon`, on a collector built with `NewBrowserCollector(BrowserOptions{ProfileDir: …})`. They need no browser.
- **Undiscardable lock:** a persistent profile directory where `SingletonLock` is a non-empty directory (for example `SingletonLock/keep` with the content `directory content`), so `os.Remove` fails with an error other than "does not exist". The existing TC-006 test of issue #180 uses the same fixture.
- **Stale lock:** the `SingletonLock`, `SingletonSocket` and `SingletonCookie` symlinks written by `writeStaleLock` (the lock names `otherhost-12345`).
- **Existing browser profile tests:** `TestBrowserProfile_*` in `internal/amazon/browser_profile_test.go` (issue #180 TC-001 to TC-008, issue #195). They are not cases of this catalogue. The IDs below are this change's own, so new test functions need names that do not clash with the existing `TestBrowserProfile_TC00N_*` names (for example an `Issue197` prefix).
- Only `profile()` changes. No API, database, user interface or runtime behaviour changes, so there is no e2e case. The existing Playwright suite runs unchanged in the gates.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-AMZ-BPROF-008 | TC-001, TC-002, TC-004 | UC-AMZ-BPROF-001, UC-AMZ-BPROF-002 (unchanged) |
| FR-AMZ-BPROF-009 | TC-003, TC-004 | UC-AMZ-BPROF-001, UC-AMZ-BPROF-002 (unchanged) |
| FR-AMZ-BPROF-010 | TC-004, TC-005 | UC-AMZ-BPROF-001, UC-AMZ-BPROF-002 (unchanged) |

## Cases

### TC-001: An undiscardable stale lock gives empty values and the error

- Covers: FR-AMZ-BPROF-008
- Level: unit (Go, `internal/amazon`, temporary directory)
- Given a collector whose `ProfileDir` holds an undiscardable lock
- When `profile()` is called
- Then it returns the path `""`, a `nil` cleanup and a non-nil error that names the lock (`remove stale profile lock SingletonLock`) and wraps the underlying file-system error
- Corner cases:
  - the profile directory itself is kept, and `SingletonLock/keep` still has its content: the failed removal changes nothing;
  - other entries of the profile (for example a `Default/Cookies` file) are kept;
  - a second call on the same profile returns the same three values (repeated action, no state left by the first call);
  - the lock that cannot be discarded is `SingletonSocket` or `SingletonCookie` instead of `SingletonLock` (each a non-empty directory, the others absent): the same empty values, and the error names that entry;
  - with `SingletonLock` absent or a normal symlink and `SingletonSocket` undiscardable, the first entry is removed and the call still returns the empty values and the error. The order and the partial removal are unchanged from before.

### TC-002: Both failure branches of `profile()` return the same form

- Covers: FR-AMZ-BPROF-008
- Level: unit (Go, `internal/amazon`, temporary directory)
- Given two collectors: one whose `ProfileDir` cannot be created (its parent is a regular file, so `os.MkdirAll` fails), and one whose `ProfileDir` holds an undiscardable lock
- When `profile()` is called on each
- Then both calls return `""`, a `nil` cleanup and a non-nil error. They differ only in the error.
- Corner cases:
  - the `MkdirAll` failure creates nothing and leaves the parent file unchanged (existing behaviour, kept).

### TC-003: A persistent profile that can be prepared still returns its path, a no-op cleanup and no error

- Covers: FR-AMZ-BPROF-009
- Level: unit (Go, `internal/amazon`, temporary directory)
- Given a collector whose `ProfileDir` is, in turn:
  1. a profile holding a stale lock and a `Default/Cookies` file;
  2. an existing profile without any lock;
  3. a path that does not exist yet;
- When `profile()` is called
- Then each call returns the `ProfileDir` path itself (not a copy or a temporary directory), a non-nil cleanup and a `nil` error
- And after the call the three lock entries are absent, the profile directory exists, and `Default/Cookies` keeps its content (run 1)
- And calling the cleanup returns without effect: the profile directory and its contents are still there
- Corner cases:
  - calling the cleanup twice is harmless;
  - in run 3 the directory is created with mode `0700`;
  - a `SingletonLock` symlink whose target is an existing file outside the profile: the link is removed and the target file is kept;
  - a second `profile()` call on the same profile gives the same values (repeated check);
  - temporary-profile branch (`ProfileDir` empty): it returns a new directory under the system temporary folder, a non-nil cleanup and a `nil` error, and its cleanup removes that directory. This branch is unchanged.

### TC-004: Amazon checks through the browser behave as before

- Covers: FR-AMZ-BPROF-008, FR-AMZ-BPROF-009, FR-AMZ-BPROF-010 (FR-AMZ-BPROF-001 to FR-AMZ-BPROF-007 unchanged), UC-AMZ-BPROF-001, UC-AMZ-BPROF-002
- Level: integration (Go, `internal/amazon`, the test browser and the Amazon fixture server, as in the existing browser profile tests)
- Given the change applied
- When the existing browser profile tests run: `go test ./internal/amazon -run 'TestBrowserProfile' -count=1`
- Then they all pass unchanged. In particular:
  - a profile whose lock cannot be discarded gives a "browser could not start" check failure that is logged and counted. No browser is launched, the check does not panic on the `nil` cleanup, and the later check after the lock is replaced returns the price (existing TC-006, FR-AMZ-BPROF-006 and FR-AMZ-BPROF-008);
  - a stale lock of this machine or of another host does not prevent the check, and cookies are kept (existing TC-001 to TC-005, FR-AMZ-BPROF-009);
  - a check in progress keeps its profile while another one starts (existing TC-008).
- Corner cases:
  - no existing test is edited to make it pass (only additions for TC-001 to TC-003 are allowed in the test files, FR-AMZ-BPROF-010);
  - no browser process and no extra goroutine are left after the failed check (`assertNothingLeft` in existing TC-006).

### TC-005: The diff is limited to `profile()` and the gates pass

- Covers: FR-AMZ-BPROF-010
- Level: code check and gates (repository root)
- Given the change's branch
- When these run:
  1. `git diff --name-only origin/master` lists the changed files;
  2. `git diff origin/master -- internal/amazon/browser.go` shows the change to `browser.go`;
  3. `gofmt -l internal/amazon`;
  4. `go vet ./...` and `go test ./...`;
- Then:
  1. outside `doc/` it lists only `internal/amazon/browser.go` and, if any, `_test.go` files under `internal/amazon/`;
  2. the only hunk is inside `profile()`. The stale lock branch has the explicit form `if err := removeStaleLocks(…); err != nil { return "", nil, err }` followed by `return c.options.ProfileDir, func() {}, nil`. The `MkdirAll` branch and the temporary-profile branch are unchanged;
  3. it prints nothing;
  4. both succeed.
- Corner cases:
  - no API (`API_SPECIFICATION.md`), migration, frontend or `tests/` file is changed, so the operator sees no change in the application;
  - the existing e2e and frontend gates run unchanged and pass.
