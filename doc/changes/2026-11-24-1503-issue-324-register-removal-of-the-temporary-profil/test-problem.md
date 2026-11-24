# Problem in the locked test of issue #324 (blocking review comment R1-01)

The whole change for this issue is the `t.Cleanup` block in the subtest "TC-003 temporary profile" of `internal/amazon/browser_profile_values_test.go` (the tests stage delivered it, `no-tests.md`, TS-TEST-HYG-072). Blocking comment R1-01 (`blocking.md`) says this block is wrong. The development stage cannot fix it, because the factory restores test files to the TDD red commit.

## What is wrong

The cleanup returns early only when `path` is `""` or equals `filepath.Clean(os.TempDir())`. It passes every other path to `os.RemoveAll`. If `profile()` in `internal/amazon/browser.go` regressed and returned some other directory, the test would delete it recursively at the end of the subtest, on the developer's or CI machine. For example, the subtest could get the persistent `c.options.ProfileDir` (from the branch at `browser.go:312`), a default profile under `~/.config`, or any other absolute path. The subtest's location check uses `t.Errorf`, so the subtest keeps running and reports that the path is wrong, but the cleanup removes the path anyway. This breaks FR-TEST-HYG-055 ("never removes anything other than the temporary profile it was given") and the TC-001 corner case that the removal targets only the temporary profile.

## Fix for the tests stage

Use the subtest's own location check as the guard, and drop the two special cases. That check already covers them, because `filepath.Dir("")` is `"."` and `filepath.Dir(os.TempDir())` is its parent:

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

`strings` is already imported (the location check uses it). Fault injections 2 to 7 in `no-tests.md` should behave the same way. In run 5, the regular file still has a matching name under the temporary folder, so it is still removed. Run 7 (empty path) still removes nothing. A normal run is unaffected.

Also update TS-TEST-HYG-071 in `doc/specifications/test-suite-hygiene/technical.md` to match. Its current wording is "returns at once when `path` is `""` or equals `filepath.Clean(os.TempDir())`". Suggested wording: it removes `path` only when the parent of `path` is `filepath.Clean(os.TempDir())` and its base name starts with `pricefollower-browser-` (the subtest's location check), and otherwise removes nothing. Update the matching bullet at `technical.md:465` and the code shown in `no-tests.md` the same way.
