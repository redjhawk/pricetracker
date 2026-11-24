# Issue #324 Register removal of the temporary profile in the TC-003 test of issue #197

**Problem:** In `internal/amazon/browser_profile_values_test.go:210-228` (subtest "TC-003 temporary profile"), the `pricefollower-browser-*` directory created under `os.TempDir()` is removed only by the explicit `cleanup()` call on line 224. A `t.Fatal` on line 213, 216 or 222 leaves it behind.

**Suggested work:** Right after the `profileOf("")` call, add `t.Cleanup(func() { if path != "" { os.RemoveAll(path) } })`. Keep the explicit `cleanup()` call that the test asserts.

From review comment R1-03 on #321 (`doc/todo/issue-197-r1-03.md`).

## Comments by trusted authors

