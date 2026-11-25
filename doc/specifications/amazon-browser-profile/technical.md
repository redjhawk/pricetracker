# Technical specification: Amazon browser profile recovery

Status: ready (issue #180); TS-AMZ-BPROF-005 – 007 ready (issue #197, maintenance: no behaviour change)\
Functional specification: [functional.md](functional.md), FR-AMZ-BPROF-001 – 007 (issue #180), FR-AMZ-BPROF-008 – 010 (issue #197)\
Test cases: [testcases.md](../../changes/2026-11-22-1355-issue-180-clear-stale-chromium-singleton-files-in/testcases.md), TC-001 – TC-010 (issue #180); [testcases.md](../../changes/2026-11-24-1205-issue-197-return-explicit-zero-values-from-browser/testcases.md), TC-001 – TC-005 (issue #197)\
Companions: [Amazon human-like checks](../amazon-human-checks/technical.md) (browser collector), [Amazon request stop](../amazon-request-stop/technical.md) (failure count and stop, unchanged)

## Summary

Chromium marks a profile as in use with three entries in the profile directory: `SingletonLock` (symlink to `<host>-<pid>`), `SingletonSocket` and `SingletonCookie`. A browser that does not end normally leaves them behind. Chromium refuses the profile when the lock names another host, so every Amazon check fails. `BrowserCollector` already runs at most one browser per collector, and the application uses one collector per data directory. Running several application instances on one data directory is out of scope. So while the collector holds its mutex, before it launches the browser, any of these entries in the persistent profile is stale and is removed.

## Requirement mapping

`BPROF-*` abbreviates `FR-AMZ-BPROF-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-AMZ-BPROF-001 | BPROF-001, BPROF-002, BPROF-005 | Before each launch with a persistent profile, remove `SingletonLock`, `SingletonSocket` and `SingletonCookie` from the profile directory with `os.Remove`, ignoring `fs.ErrNotExist`. Remove them whatever their kind or target, and never read or parse the lock. | `internal/amazon/browser.go` | TC-001, TC-002, TC-003, TC-004, TC-009, TC-010 |
| TS-AMZ-BPROF-002 | BPROF-003, BPROF-004 | Remove only these three exact names. `os.Remove` deletes a symlink, never its target. Every other entry (cookies, `Local State`, look-alike names) stays. A temporary profile (`ProfileDir == ""`) is not cleaned: it is new and empty. | `internal/amazon/browser.go` | TC-001, TC-002, TC-003, TC-005, TC-010 |
| TS-AMZ-BPROF-003 | BPROF-006 | If a removal fails for any reason other than `fs.ErrNotExist`, the profile preparation returns `remove stale profile lock <name>: <err>`. `open` wraps it as `browser could not start: …` and launches no browser. The existing `requestFailure` path then logs, counts and stops items. | `internal/amazon/browser.go` | TC-006, TC-007 |
| TS-AMZ-BPROF-004 | BPROF-007 | Do the cleanup only inside `Collect` while `c.mu` is held, so it never runs while the collector's browser holds the profile. After taking the lock, a check whose context has already ended returns a `requestFailure` with `ctx.Err()` before cleaning or launching. | `internal/amazon/browser.go` | TC-008, TC-009 |
| TS-AMZ-BPROF-005 | BPROF-008 | Issue #197. In `profile()`, when `removeStaleLocks(ProfileDir)` fails, return `"", nil, err` with the error unchanged. This is the same form as the `MkdirAll` branch. | `internal/amazon/browser.go` | #197 TC-001, TC-002, TC-004 |
| TS-AMZ-BPROF-006 | BPROF-009 | Issue #197. When `removeStaleLocks` succeeds, return `c.options.ProfileDir, func() {}, nil`. The temporary-profile branch and `removeStaleLocks` do not change. | `internal/amazon/browser.go` | #197 TC-003, TC-004 |
| TS-AMZ-BPROF-007 | BPROF-010 | Issue #197. No other production change. The new tests go in a new test file, and existing tests are not edited. | `internal/amazon/browser.go`, `internal/amazon/browser_profile_values_test.go` (new) | #197 TC-004, TC-005 |

## Explicit zero values from `profile()` (issue #197)

Maintenance only. Today the persistent-profile branch ends with `return c.options.ProfileDir, func() {}, removeStaleLocks(c.options.ProfileDir)`, so a failure returns a path and a cleanup together with the error. Replace that one statement with:

```go
if err := removeStaleLocks(c.options.ProfileDir); err != nil {
	return "", nil, err
}
return c.options.ProfileDir, func() {}, nil
```

- Unchanged: the `MkdirAll` branch, the temporary-profile branch, the doc comment of `profile()`, `staleLockNames` and `removeStaleLocks` (its order, partial removal and error text `remove stale profile lock <name>: %w`).
- Caller safety: the only caller, `open`, returns `browser could not start: …` as soon as `err != nil` and never uses the path or `removeTemp` in that case, so nothing calls the `nil` cleanup. The failure stays a counted `request_error` (FR-AMZ-BPROF-006).
- Concurrency, persistence, validation and logging: unchanged. `profile()` still runs with `c.mu` held, and there is no migration.
- Refactoring: none beyond this statement.

## Frontend

Not affected: there is no new screen, setting or message, and item states and messages stay the same (functional spec, "States and corner cases").

No frontend file changes. The keyboard focus fixes of earlier rounds in `src/components/AppMenu.tsx` and `src/components/InformationModal.tsx` are withdrawn (R1-02): they are unrelated to this issue and belong to their own issue about the intermittent Playwright tests.

## Backend

Package `internal/amazon`, file `browser.go` only:

- New constant list `staleLockNames = []string{"SingletonLock", "SingletonSocket", "SingletonCookie"}` with a comment explaining that Chromium leaves these when it is killed or crashes, and refuses the profile when the lock names another host. (The test file declares its own `profileLockNames`, so the production list uses another name.)
- New function `removeStaleLocks(directory string) error`: for each name, `os.Remove(filepath.Join(directory, name))`. It ignores `errors.Is(err, fs.ErrNotExist)` and returns at the first other error as `fmt.Errorf("remove stale profile lock %s: %w", name, err)`. A non-empty directory under one of these names fails with `ENOTEMPTY`, and its content is left as it is (TC-006).
- `BrowserCollector.profile()`: when `ProfileDir` is set, run `os.MkdirAll(ProfileDir, 0o700)` and then `removeStaleLocks(ProfileDir)`, returning the first error. The temporary-profile branch does not change.
- `BrowserCollector.Collect`: after `c.mu.Lock()`, if `ctx.Err() != nil` return `requestFailure(item, result, ctx.Err().Error())`. A check cancelled while it waited for the profile then starts no browser (TC-008 corner case). The rest of the flow does not change.
- Comment on `BrowserCollector.mu`: it also guards the profile cleanup, so the cleanup never runs while a browser of this collector uses the profile.
- Logging: a successful cleanup logs nothing (the recovery is invisible). A failure logs through the existing `printRequestError`, and that line contains `browser could not start`.
- Concurrency: the service creates one collector (`newBrowserCollector` in `internal/service/service.go`, not changed), so the overlapping checks of the scheduler, **Refresh price** and **Refresh all prices** queue on `c.mu` as they do today. No change to `cdp.go`: `close()` still kills the process group after 5 s, and the next check removes whatever that leaves.
- Persistence and migrations: none. The SQLite schema, failure count, stop and restart rules do not change.
- Validation: none added. The profile path comes from the configuration (`<data dir>/amazon-browser`) as before.

## API

Not affected (issues #180 and #197): there is no new endpoint or field and no change to responses or errors. `API_SPECIFICATION.md` stays as it is. A browser that cannot start still gives `lastAttempt.result: "request_error"` with "Amazon could not be reached for a price check.", and `POST /api/v1/items/{id}/refresh` still answers `202`.

## Files to change

| File | Change |
| --- | --- |
| `internal/amazon/browser.go` | TS-AMZ-BPROF-001 – 004 |
| `internal/amazon/browser_profile_test.go` (new) | TC-001 – TC-006, TC-008 |
| `internal/service/amazon_browser_profile_test.go` (new) | TC-007, TC-009 |
| `internal/httpapi/amazon_browser_profile_test.go` (new) | TC-010 |

Issue #197:

| File | Change |
| --- | --- |
| `internal/amazon/browser.go` | TS-AMZ-BPROF-005 – 006, inside `profile()` only |
| `internal/amazon/browser_profile_values_test.go` (new) | #197 TC-001 – TC-003 |

## Test implementation

New test names use the prefix `TestBrowserProfile_TCnnn_` in `internal/amazon` and `TestAmazonBrowserProfile_TCnnn_` in `internal/service` and `internal/httpapi`. Browser tests get the executable from `amazontest.Browser(t)` and the fixture server from `amazontest.NewServer`. The Amazon host is sent to that server through `Origin`.

| Test case | File | Notes |
| --- | --- | --- |
| TC-001 | `internal/amazon/browser_profile_test.go` | Unit test, no real browser. The stub executable is a `/bin/sh` script written to `t.TempDir()`. It reads `--user-data-dir=` from its arguments, writes the profile's entries (relative name, kind, symlink target, for example with `find "$dir" -printf '%P %y %l\n'`) to a file, and exits 3. Subtests cover each corner case. The temporary-profile case checks that the listed directory is empty and gone after the check. |
| TC-002 | `internal/amazon/browser_profile_test.go` | Cookie handler as in TC-003 of issue #168, with `session-id=fixture-session-180`. Helper `writeStaleLock(t, profile, host)` creates the three symlinks. The sanity sub-check starts the browser directly on the prepared profile and expects it to exit without loading the page. The test also uses the existing `captureOutput` and `assertNothingLeft`. |
| TC-003 | `internal/amazon/browser_profile_test.go` | Helper `killBrowserOnProfile(t, executable, profile)` starts the browser itself (`Setpgid`), waits for `SingletonLock`, sends SIGKILL to the group, waits for the exit, then checks that `SingletonLock` is still there. |
| TC-004 | `internal/amazon/browser_profile_test.go` | One collector, cycles made of `killBrowserOnProfile` followed by a check. The third cycle uses `writeStaleLock` with another host. |
| TC-005 | `internal/amazon/browser_profile_test.go` | Uses the existing `fixtureHandler` together with the cookie handler. |
| TC-006 | `internal/amazon/browser_profile_test.go` | `SingletonLock` is a directory that contains a file. The check runs with a 20 s context. The test checks the directory content and uses `assertNothingLeft`. |
| TC-008 | `internal/amazon/browser_profile_test.go` | The handler for item A blocks on a channel. The test waits until the server has received A, reads the `SingletonLock` target, starts B, checks that B has not reached the server and the lock is unchanged, then releases A. Cancellation corner case: B's context is cancelled while A is blocked; after A is released, B returns `request_error` and the server never received B. |
| TC-007 | `internal/service/amazon_browser_profile_test.go` | Uses the harness of `amazon_human_checks_test.go` (`openHumanStore`, `startHuman`, `insertAt`, `amazonFailures`, `waitIdle`). The collector is `amazon.NewBrowserCollector` on a profile with a `SingletonLock` directory. Logs are captured with `log.SetOutput`. |
| TC-009 | `internal/service/amazon_browser_profile_test.go` | Fake clock (`newClock`, `scheduleDay`), a local fixture mux like TC-020 of issue #168, and a kill helper local to the package, because helpers in `internal/amazon` test files are not exported. The protected-mode corner case uses `insertOwned`. |
| TC-010 | `internal/httpapi/amazon_browser_profile_test.go` | `service.NewWithOptions` with `Options.Amazon = amazon.NewBrowserCollector(amazon.BrowserOptions{ProfileDir: filepath.Join(dataDir, "amazon-browser"), Origin: server.URL, …})`, matching the production path. The test calls the HTTP handlers for `POST /api/v1/items/{id}/refresh` and `GET /api/v1/items`. |

The existing tests must keep passing: `internal/amazon/browser_test.go` (including TC-007 of issue #168, missing executable), `internal/service/amazon_human_checks_test.go`, `internal/httpapi/amazon_human_checks_test.go` and the Playwright tests in `tests/`. There is no e2e case because nothing changes in the UI.

### Test implementation (issue #197)

The new unit tests go in `internal/amazon/browser_profile_values_test.go` (package `amazon`). They call `profile()` directly on `NewBrowserCollector(BrowserOptions{ProfileDir: …})` and launch no browser. Names use the prefix `TestBrowserProfile_Issue197_TCnnn_`, so they do not clash with the issue #180 tests. They reuse `writeStaleLock`, `writeProfileFile` and `profileLockNames` from `browser_profile_test.go`, which is not edited. Each corner case is a subtest with its own `t.TempDir()`. A helper `assertZeroValues(t, path, cleanup, err)` checks `path == ""`, `cleanup == nil` and `err != nil`.

| Test case | File / gate | Notes |
| --- | --- | --- |
| TC-001 | `internal/amazon/browser_profile_values_test.go` | The undiscardable lock is `<name>/keep` with the content `directory content`, for `SingletonLock`, `SingletonSocket` and `SingletonCookie` in turn. Checks: the zero values; the error contains `remove stale profile lock <name>` and `errors.As(err, &pathErr)` (`*fs.PathError`) holds; `keep` and `Default/Cookies` keep their content; a second call gives the same values. Partial-removal case: `SingletonLock` is absent, then a symlink, and `SingletonSocket` is undiscardable. After the call `os.Lstat` on `SingletonLock` gives `fs.ErrNotExist`. |
| TC-002 | `internal/amazon/browser_profile_values_test.go` | `MkdirAll` failure: `ProfileDir` is `<file>/profile`, where `<file>` is a regular file. The second collector holds an undiscardable lock. Both results satisfy `assertZeroValues`. The parent file is still a regular file with its content. |
| TC-003 | `internal/amazon/browser_profile_values_test.go` | Runs 1 – 3 check `path == ProfileDir`, `cleanup != nil`, `err == nil`, the absent lock entries (`os.Lstat`), the cookie content, and that the directory survives two cleanup calls. Run 3 checks `info.Mode().Perm() == 0o700` (a usual umask does not reduce `0o700`). Symlink case: `SingletonLock` points to a file outside the profile, and that file is kept. Temporary branch: `ProfileDir == ""` gives a directory under `os.TempDir()`, and the cleanup removes it. |
| TC-004 | existing `TestBrowserProfile_*` tests | `go test ./internal/amazon -run 'TestBrowserProfile' -count=1` with the files unchanged. It needs the test browser, as before. |
| TC-005 | factory gates and review | `git diff --name-only origin/master`, `gofmt -l internal/amazon`, `go vet ./...` and `go test ./...`. No test file. |

TDD: TC-001 and TC-002 fail on the current code, which returns a non-empty path and a non-nil cleanup on the lock failure. TC-003 already passes and guards the success branch.

## Scope and refactoring

No refactoring. In production code, only `internal/amazon/browser.go` changes. Out of scope: parsing the lock or checking whether its process is alive (not needed, because the collector's mutex already guarantees exclusive use), locking between application instances, and restarting items already stopped (they restart through **Refresh price**, FR-AMZ-STOP-006).

## Verification

`go build ./...`, `go vet ./...`, `go test ./...`, `go test -race ./internal/amazon/... ./internal/service/... ./internal/httpapi/...` with Chromium available (`CI` set, so browser tests cannot skip), and `npm run build`.

No open question.
