# No red phase for issue #324

The change is the test code itself: the subtest "TC-003 temporary profile" of `TestBrowserProfile_Issue197_TC003_PreparedProfileReturnsItsPath` in `internal/amazon/browser_profile_values_test.go`. That file is locked after this stage, and FR-TEST-HYG-056 allows no other file outside `doc/`, so the development stage cannot write any of it. This stage delivers the whole change, as the technical specification says (TS-TEST-HYG-072). A normal run passes before and after the change, so no test can fail first on a missing implementation. The red gate is skipped. The development stage writes no code: it only sets the statuses.

## Delivered in `internal/amazon/browser_profile_values_test.go`

Right after `path, cleanup, err := profileOf("")`, before the first check, the subtest registers on its own `t`:

```go
t.Cleanup(func() { // the temporary profile is removed however the subtest ends, after its checks (issue #324)
	if path == "" || filepath.Clean(path) == filepath.Clean(os.TempDir()) {
		return
	}
	if err := os.RemoveAll(path); err != nil {
		t.Errorf("remove the temporary profile: %v", err)
	}
})
```

The explicit `cleanup()`, the `os.Stat` check after it with `the cleanup did not remove the temporary profile: %v`, the other checks and messages, the other subtests, the helpers and the imports are unchanged.

## Runs

Run command: `go test ./internal/amazon -run '^TestBrowserProfile_Issue197_TC003_PreparedProfileReturnsItsPath$' -count=1 -v`. Each injection was a temporary `Edit` of the test file or of `BrowserCollector.profile` in `internal/amazon/browser.go`, reverted with `Edit` before the next step.

Leftover check: this runner's sandbox refused to set `TMPDIR` in the shell, and to list or create folders outside the repository. So the fresh temporary folder of the design was set up by a temporary harness in the parent test, placed just before the subtest and removed with the injections. The harness made `dir := t.TempDir()`, called `t.Setenv("TMPDIR", dir)` (so `os.TempDir()`, and hence `profile()` and the subtest's location check, use `dir`), and wrote a marker file into `dir` and one into the test's working directory (`internal/amazon`). After `t.Run` returned, so after the subtest and its cleanups, it logged what `dir` held, whether the working-directory marker was still there, and whether the working directory listing matched the one taken before. It then removed its marker. The parent's `t.TempDir()` cleanup removed `dir` afterwards, so a deliberate leftover needed no deletion.

1. Normal run: all five subtests PASS. `dir` holds only `[marker-324]`. With `-count=3`: PASS three times, `[marker-324]` each time. With `-run 'TestBrowserProfile' -count=1` (harness removed): every `TestBrowserProfile_*` test passes, including the real-browser tests of issue #180 (26.2 s). The added removal logs nothing on a normal run. `-race` could not run here, because it needs cgo and cgo is off on this runner (`go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`), as for issue #315. The added cleanup runs in the subtest's own goroutine after the subtest and adds no concurrency.
2. Right after the `os.Stat`/`IsDir` check, `os.WriteFile(filepath.Join(path, "keep"), …)` and `t.Fatal("injected")` were added. The subtest FAILS with `injected`, and `dir` holds only `[marker-324]`, so the non-empty profile was removed. With the `t.Cleanup` block also disabled (the `master` behaviour), the subtest fails the same way and `dir` holds `[marker-324 pricefollower-browser-3842367496]`: one leaked profile, the leak that the change removes.
3. `profile()` returned the created directory with a `nil` cleanup. The subtest FAILS with `cleanup is nil`, and `dir` holds only `[marker-324]`.
4. `profile()` returned the created directory with `errors.New("injected")`. The subtest FAILS with `injected`, and `dir` holds only `[marker-324]`.
5. `profile()` removed the directory, created a regular file with `os.CreateTemp("", "pricefollower-browser-")`, closed it, and returned its name with its removal as cleanup. The subtest FAILS with `the temporary profile was not created: <nil>`, and `dir` holds only `[marker-324]`.
6. `profile()` returned the created directory with `func() {}` as cleanup. The subtest FAILS once, with `the cleanup did not remove the temporary profile: <nil>` as its only message, and `dir` holds only `[marker-324]`. So the added removal ran after the check that follows `cleanup()`, and cannot hide a cleanup that removes nothing.
7. `profile()` removed the directory and returned `"", func() {}, nil`. The subtest FAILS on its existing checks: `temporary profile "", want a new directory under <dir>`, then `the temporary profile was not created: stat : no such file or directory`. Then `profile()` returned `"", nil, errors.New("injected")`, and the subtest FAILS with `injected`. In both runs `dir` still exists and holds `[marker-324]`, the working-directory marker is still there, and the `internal/amazon` listing is unchanged. So an empty path removes nothing.
8. After the reverts: `git diff --name-only master -- . ':!doc'` lists only `internal/amazon/browser_profile_values_test.go`. `git diff master -- internal/amazon/browser_profile_values_test.go` holds only the `t.Cleanup` block. `browser.go` is unchanged, and neither the harness, an injection nor a marker file remains. `gofmt -l internal/amazon` prints nothing, `go vet ./...` passes, and `go test ./... -count=1` passes twice (all packages `ok`).

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | Code check of the delivered code (above): `t.Cleanup` on the subtest's `t` right after `profileOf("")`, before the first `t.Fatal`; it returns on `""` or the system temporary folder, and otherwise removes only `path`. `cleanup()`, the check after it, and every other check, message and subtest are unchanged |
| TC-002 | Runs 2 (fatal stop with a non-empty profile: nothing left; without the removal: one `pricefollower-browser-*` left), 3 (nil cleanup), 4 (error) and 5 (regular file): each fails with its expected message and leaves only the marker |
| TC-003 | Run 6: one failure, `the cleanup did not remove the temporary profile: <nil>`, and nothing left afterwards |
| TC-004 | Run 7: the subtest fails on its existing checks; the temporary folder, its marker, the working-directory marker and the `internal/amazon` listing are unchanged |
| TC-005 | Run 1 with `-v`, `-count=3` and `-run 'TestBrowserProfile'` (`-race` is not available on this runner), and `go test ./...` in run 8 |
| TC-006 | Repository checks and gates, run 8. The development stage runs them again |
