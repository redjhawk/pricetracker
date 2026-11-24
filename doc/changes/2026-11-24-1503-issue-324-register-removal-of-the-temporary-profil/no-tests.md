# No red phase for issue #324

The change is the test code itself: the subtest "TC-003 temporary profile" of `TestBrowserProfile_Issue197_TC003_PreparedProfileReturnsItsPath` in `internal/amazon/browser_profile_values_test.go`. That file is locked after this stage, and FR-TEST-HYG-056 allows no other file outside `doc/`, so the development stage cannot write any of it. This stage delivers the whole change, as the technical specification says (TS-TEST-HYG-072). A normal run passes before and after the change, so no test can fail first on a missing implementation. The red gate is skipped. The development stage writes no code: it only sets the statuses.

## Delivered in `internal/amazon/browser_profile_values_test.go`

Right after `path, cleanup, err := profileOf("")`, before the first check, the subtest registers on its own `t`:

```go
t.Cleanup(func() { // the temporary profile is removed however the subtest ends, after its checks (issue #324)
	if filepath.Dir(path) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(path), "pricefollower-browser-") {
		return
	}
	if err := os.RemoveAll(path); err != nil {
		t.Errorf("remove the temporary profile: %v", err)
	}
})
```

The guard is the subtest's own location check: `path` is removed only when its parent is the system temporary folder and its base name starts with `pricefollower-browser-`. Any other path is left alone, including `""` (`filepath.Dir("")` is `.`), the temporary folder itself and a persistent profile directory (FR-TEST-HYG-055; review comment R1-01 and the round 1 `test-problem.md`). The round 0 guard skipped only `""` and the temporary folder, so it would have removed any other path that a faulty `profile()` returned.

The explicit `cleanup()`, the `os.Stat` check after it with `the cleanup did not remove the temporary profile: %v`, the other checks and messages, the other subtests, the helpers and the imports are unchanged.

## Runs

Repeated in round 1 with the corrected guard. Run command: `go test ./internal/amazon -run '^TestBrowserProfile_Issue197_TC003_PreparedProfileReturnsItsPath$' -count=1 -v`. The injections were temporary `Edit`s of the subtest and of `BrowserCollector.profile` in `internal/amazon/browser.go`. Each run selected its injection through the content of a temporary file, `internal/amazon/inject-324.txt`, because this runner refuses environment-variable prefixes on commands. All the edits were reverted with `Edit` afterwards. The sandbox refused to delete the selector file, so `delete.txt` lists it for the pipeline to remove. It is untracked, and no code reads it any more.

Leftover check: a temporary harness in the parent test, just before the subtest, made `dir := t.TempDir()` and called `t.Setenv("TMPDIR", dir)`, so `os.TempDir()`, `profile()` and the subtest's location check all use `dir`. It wrote `marker-324` and a non-empty directory `other-324/keep` into `dir`. In the test's working directory (`internal/amazon`) it wrote a marker file and a non-empty directory `persist-324/keep`. After `t.Run` returned, so after the subtest and its cleanups, it logged the tree of `dir`, whether the working-directory marker and `persist-324/keep` still existed, and whether the working-directory listing matched the one taken before. It then removed its own files, and the parent's `t.TempDir()` cleanup removed `dir`.

In every run below, the working-directory marker and `persist-324/keep` were still there and the working-directory listing was unchanged. "Only the fixtures" means that `dir` held exactly `marker-324`, `other-324` and `other-324/keep`.

1. Normal run: all five subtests PASS, only the fixtures. Without the harness, `-count=3` passes three times, and `go test ./... -count=1` passes (all packages `ok`, including every `TestBrowserProfile_*` test). The added removal logs nothing on a normal run. `-race` is not available here, because cgo is off on this runner (as in round 0 and for issue #315). The added cleanup runs in the subtest's own goroutine after the subtest and adds no concurrency.
2. Right after the `os.Stat`/`IsDir` check, add `os.WriteFile(filepath.Join(path, "keep"), …)` and `t.Fatal("injected")`: FAILS with `injected`, only the fixtures, so the non-empty profile was removed. With the `t.Cleanup` body also disabled (the `master` behaviour), it FAILS the same way and `dir` also holds `pricefollower-browser-1324516888/keep`. That is the leak the change removes.
3. `profile()` returns the created directory with a `nil` cleanup: FAILS with `cleanup is nil`, only the fixtures.
4. `profile()` returns the created directory with `errors.New("injected")`: FAILS with `injected`, only the fixtures.
5. `profile()` removes the directory, creates a regular file with `os.CreateTemp("", "pricefollower-browser-")` and returns its name: FAILS with `the temporary profile was not created: <nil>`, only the fixtures. The file has a matching name under `dir`, so it is removed.
6. `profile()` returns the created directory with `func() {}` as cleanup: FAILS once, with `the cleanup did not remove the temporary profile: <nil>` as its only message, only the fixtures. So the added removal runs after the check that follows `cleanup()`.
7. `profile()` returns `"", func() {}, nil`: FAILS with `temporary profile "", want a new directory under <dir>`, then `the temporary profile was not created: stat : no such file or directory`. With `"", nil, errors.New("injected")` instead: FAILS with `injected`. In both runs, only the fixtures, and nothing else is removed.
8. New in round 1 (R1-01): `profile()` returns one of three paths, each with `func() {}` as cleanup: (a) `os.TempDir()` itself, (b) the absolute path of the persistent directory `internal/amazon/persist-324`, (c) `<dir>/other-324`, a directory under the temporary folder without the prefix. Each FAILS on the location check (`temporary profile %q, want a new directory under <dir>`) and then on `the cleanup did not remove the temporary profile: <nil>`. In each run, `dir` still holds only the fixtures and `persist-324/keep` still exists, so the added removal touches none of these paths.
9. After the reverts: `git diff master -- . ':!doc'` holds only the `t.Cleanup` block in the subtest. `browser.go` is unchanged, and no harness or injection remains in code. `gofmt -l internal/amazon` prints nothing, `go vet ./...` passes, and `go test ./... -count=1` passes.

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | Code check of the delivered code (above): `t.Cleanup` on the subtest's `t` right after `profileOf("")`, before the first `t.Fatal`. It removes `path` only when `path` passes the subtest's location check (parent is the system temporary folder, base name starts with `pricefollower-browser-`), and removes nothing otherwise. `cleanup()`, the check after it, and every other check, message and subtest are unchanged. Run 8 shows that no other path is targeted |
| TC-002 | Runs 2 (fatal stop with a non-empty profile: nothing left; without the removal: one `pricefollower-browser-*` left), 3 (nil cleanup), 4 (error) and 5 (regular file): each fails with its expected message and leaves only the fixtures |
| TC-003 | Run 6: one failure, `the cleanup did not remove the temporary profile: <nil>`, and nothing left afterwards |
| TC-004 | Run 7: the subtest fails on its existing checks; the temporary folder, its fixtures, the working-directory marker and the `internal/amazon` listing are unchanged. Run 8 adds the temporary folder itself and other paths |
| TC-005 | Run 1 with `-v`, `-count=3` and `go test ./...` (`-race` is not available on this runner) |
| TC-006 | Repository checks and gates, run 9. The development stage runs them again |
