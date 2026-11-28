# Test cases: share the log/stdout capture test helper through amazontest (issue #218)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-127 to FR-TEST-HYG-131, with FR-TEST-HYG-010 for the dependency check). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **The helper:** `CaptureOutput(t *testing.T) func() string` of `internal/amazon/amazontest`. **Ending read**, **live read**, **logged text**, **printed text** and **local capture copies:** as defined in the terms of the functional specification (issue #218). The live read is called as the technical specification defines it; these cases only rely on its behaviour.
- **Previous destinations:** before calling the helper, a case points the standard logger to a buffer of its own and `os.Stdout` to a file of `t.TempDir()`, and restores both with `t.Cleanup`. This lets the case check what reaches them before, during and after the capture.
- **Writes:** `log.Print(...)` for logged text and `fmt.Print(...)` or `fmt.Println(...)` for printed text. Default `log` flags are kept, so a logged line ends with its text and a newline.
- **Fault injection:** a temporary edit made only for the run of a case, reverted afterwards and never committed.
- Levels: TC-001 to TC-009 are `unit` Go tests of the helper package, named `TestCaptureOutput_Issue218_TC00N_...` in a new test file of `internal/amazon/amazontest`. TC-010 is a repository check. TC-011 runs the existing tests of `internal/amazon` as `unit` tests and those of `internal/service` and `internal/httpapi` as `integration` tests (real SQLite, service, HTTP handlers, real Chromium). TC-012 and TC-013 are repository and gate checks. The change only touches test code: there is no user flow and no user interface change, so there is no e2e case. The IDs below are this change's own, not those in existing test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-127 | TC-001, TC-002, TC-008, TC-009 | — |
| FR-TEST-HYG-128 | TC-002, TC-003, TC-004, TC-005, TC-006 | — |
| FR-TEST-HYG-129 | TC-007, TC-008 | — |
| FR-TEST-HYG-130 | TC-010, TC-011 | — |
| FR-TEST-HYG-131 | TC-011, TC-012, TC-013 | — |

## Cases

### TC-001: the helper is exported with the required signature

- Covers: FR-TEST-HYG-127
- Level: unit (Go, helper package, and the compilation of the callers in other packages)
- Given the helper package
- When `go doc pricefollower.local/internal/amazon/amazontest CaptureOutput` runs, and `go vet ./...` compiles the test files of `internal/amazon`, `internal/service` and `internal/httpapi`
- Then `go doc` shows `func CaptureOutput(t *testing.T) func() string`, and the packages outside the helper package compile their calls to it

### TC-002: the capture takes the logged and printed text and the ending read returns both in order

- Covers: FR-TEST-HYG-127, FR-TEST-HYG-128
- Level: unit (Go, helper package)
- Given the previous destinations
- When the case calls the helper, writes `fmt.Println("printed")` then `log.Print("logged")`, and calls the returned function
- Then the ending read ends with `logged\nprinted\n`, so the logged text comes before the printed text whatever the order of the writes; during the capture, the previous destinations received nothing; after the read, `log.Writer()` is the previous logger buffer and `os.Stdout` is the previous file again
- Corner cases:
  - several logged and printed lines: all logged lines come first, in their order, then all printed lines, in their order;
  - text printed without a final newline (`fmt.Print("partial")`) is returned whole;
  - a capture with no write: the ending read returns the empty string.

### TC-003: later calls return the same text and nothing is captured after the end

- Covers: FR-TEST-HYG-128
- Level: unit (Go, helper package)
- Given the previous destinations and a capture with `log.Print("logged")` and `fmt.Println("printed")`, ended by a first call of the returned function
- When the case writes `log.Print("after-log")` and `fmt.Println("after-print")`, then calls the returned function a second and a third time
- Then the second and third calls return exactly the text of the first call, without `after-log` or `after-print`; `after-log` is in the previous logger buffer and `after-print` is in the previous file
- Corner cases:
  - a live read made after the end does not hold `after-log` or `after-print`;
  - the calls after the end return at once (each in less than 1 s), without blocking.

### TC-004: the ending read waits for all text printed before it

- Covers: FR-TEST-HYG-128
- Level: unit (Go, helper package)
- Given a capture
- When the case prints 1 MiB of text (larger than a pipe buffer) followed by a marker line, and calls the returned function immediately after the last print
- Then the ending read holds the whole printed text, byte for byte, ending with the marker line
- Corner cases:
  - the same with printing done by another goroutine that has returned before the call: the text is whole;
  - repeated with `-count=20`: every run holds the whole text.

### TC-005: the capture ends when the test ends if the function is never called

- Covers: FR-TEST-HYG-128
- Level: unit (Go, helper package, subtest)
- Given the previous destinations, set in the parent test
- When a subtest calls the helper, writes `log.Print("logged")` and `fmt.Println("printed")`, keeps the returned function without calling it, and returns
- Then in the parent, after the subtest, `log.Writer()` is the previous logger buffer and `os.Stdout` is the previous file; neither received `logged` nor `printed`; calling the kept function returns text ending with `logged\nprinted\n`
- Corner cases:
  - text written in the parent after the subtest reaches the previous destinations and is not in the kept function's text.

### TC-006: sequential captures in one test each return only their own text

- Covers: FR-TEST-HYG-128
- Level: unit (Go, helper package, subtest)
- Given the previous destinations, set in the parent test
- When a subtest makes a first capture with `log.Print("one")`, ends it with its ending read, then makes a second capture with `fmt.Println("two")` and ends it, as `internal/amazon/browser_navigation_test.go` and `internal/httpapi/amazon_browser_interrupted_test.go` do
- Then the first ending read holds `one` and not `two`; the second holds `two` and not `one`; after the subtest ends, and the cleanups of both captures have run, `log.Writer()` and `os.Stdout` are the previous destinations, not those of the first capture
- Corner cases:
  - the second capture made while the first is still open, then both ended in reverse order: after the subtest the previous destinations are restored.

### TC-007: the live read returns the text so far without ending the capture

- Covers: FR-TEST-HYG-129
- Level: unit (Go, helper package)
- Given the previous destinations and a capture
- When the case makes a live read before any write, then writes `log.Print("logged")` and makes a live read, then writes `fmt.Println("printed")` and polls live reads for up to 5 s, then writes `log.Print("later")` and makes the ending read
- Then the first live read is empty; the live read after `log.Print` holds `logged`; a live read within 5 s holds `printed`; the capture is still on after the live reads (the previous destinations received nothing, `os.Stdout` is not the previous file); the ending read holds `logged`, `printed` and `later`
- Corner cases:
  - a live read with nothing new to collect returns at once (in less than 1 s), without waiting for more text;
  - repeated live reads with no write in between return the same text;
  - a live read never returns text that a later live read or the ending read lacks: each read's text is contained in the ending read.

### TC-008: concurrent writes and reads cause no data race

- Covers: FR-TEST-HYG-127, FR-TEST-HYG-129
- Level: unit (Go, helper package, run with `go test -race`)
- Given a capture
- When 8 goroutines each write 100 numbered lines, half through `log.Print` and half through `fmt.Println`, while another goroutine makes live reads in a loop; then all goroutines are waited for and the ending read is made
- Then `go test -race -count=1 -run '^TestCaptureOutput_Issue218_' ./internal/amazon/amazontest/` passes with no race report, and the ending read holds each of the 800 lines exactly once
- Corner cases:
  - the logged lines appear in the ending read before all printed lines.

### TC-009: a capture that cannot be set up fails the test with the error

- Covers: FR-TEST-HYG-127
- Level: unit (Go, helper package, re-run of the test binary as a subprocess)
- Given a subprocess of the test binary, selected by `-test.run` and an environment variable, in which the pipe cannot be created (for example, the file descriptor limit lowered with `syscall.Setrlimit` and the free descriptors used up before the call)
- When the subprocess test calls the helper
- Then the subprocess test fails and its output holds the error text of the failed setup (`too many open files`); the parent case checks the subprocess's exit status and output
- Corner cases:
  - the parent case itself does not change its own limits or destinations.

### TC-010: the local capture copies no longer exist

- Covers: FR-TEST-HYG-130
- Level: repository check
- Given the change branch
- When `grep -rnE 'func (captureOutput|captureCheckOutput|captureProfileOutput)\(|type (lockedBuffer|profileOutput) ' --include='*.go' .`, `grep -rln 'os.Stdout = ' --include='*.go' internal cmd` and `grep -rln 'os.Pipe()' --include='*.go' internal cmd` run
- Then the first prints nothing; the second lists only files of `internal/amazon/amazontest/`; the third lists only files of `internal/amazon/amazontest/` and `internal/amazon/cdp.go`
- Corner cases:
  - `grep -rnwE 'captureOutput|captureCheckOutput|captureProfileOutput|lockedBuffer|profileOutput' --include='*.go' .` prints nothing either: no call or use of the former names is left;
  - the other log-capturing helpers (`captureRetrievalLog`, `captureSecretLogs`, `captureLogs`, `captureLog`) and `syncBuffer` of `internal/httpapi` still exist, unchanged.

### TC-011: the tests that used the local copies keep their steps and pass

- Covers: FR-TEST-HYG-130, FR-TEST-HYG-131
- Level: unit (`internal/amazon`, real Chromium) and integration (`internal/service` and `internal/httpapi`: real SQLite, service, HTTP handlers, real Chromium)
- Given the tests of `internal/amazon`, `internal/service` and `internal/httpapi` that called `captureOutput`, `captureCheckOutput` or `captureProfileOutput` (among them the browser, navigation, retrieval, cache log, interrupted, error page, empty status and profile tests, and `TestAmazonBrowserProfile_TC007_FailureIsLoggedCountedAndStopsItems` and `TestAmazonBrowserProfile_TC009_ScheduledCheckAndRefreshAfterKilledBrowsers` of `internal/service`)
- When `go test -count=1 -v ./internal/amazon/ ./internal/service/ ./internal/httpapi/` runs with Chromium, and the list of `=== RUN` lines is compared with the same run on `master`
- Then the run passes and lists the same tests and subtests; in the diff, each former call of `captureOutput` or `captureCheckOutput` is a call of `CaptureOutput` whose result is read as before, and `waitStartFailures` and TC-009 of `internal/service` read the output with the live read while the capture goes on
- Corner cases:
  - `waitStartFailures` still counts exactly the wanted start failures within its 5 s wait, so its count depends on the live read showing printed text without ending the capture;
  - the test steps and assertions are unchanged: in the diff of the caller files, only the capture calls, the reads of their result, the removed copies and the imports the compiler requires change;
  - fault injection: making the live read return the empty string makes `TestAmazonBrowserProfile_TC007_FailureIsLoggedCountedAndStopsItems` fail in `waitStartFailures`; reverted, it passes.

### TC-012: the application is unchanged

- Covers: FR-TEST-HYG-131
- Level: integration (Go, repository check)
- Given the change's diff
- When the dependency check (FR-TEST-HYG-010, `TestServerDoesNotDependOnTestHelper`) runs as part of `go test ./...`
- Then it passes: the server's dependencies still exclude `internal/amazon/amazontest`, so the operator observes no change in the application

### TC-013: the diff stays within the allowed files and the gates pass

- Covers: FR-TEST-HYG-131
- Level: repository and gate check
- Given the change branch
- When `git diff --name-only origin/master`, `gofmt -l internal`, `go vet ./...` and `go test ./...` run
- Then the diff outside `doc/` lists only files of `internal/amazon/amazontest/` and Go test files of `internal/amazon`, `internal/service` and `internal/httpapi` that declared or called a local capture copy; no production Go file, `go.mod` or `go.sum` is listed; `gofmt -l internal` prints nothing; `go vet ./...` and `go test ./...` succeed
