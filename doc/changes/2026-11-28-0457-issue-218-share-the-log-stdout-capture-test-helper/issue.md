# Issue #218 Share the log/stdout capture test helper through amazontest

**Problem:** Three copies of the same helper capture `log` and `os.Stdout` through a pipe: `captureOutput` (`internal/amazon/browser_test.go:157`), the helper in `internal/service/amazon_browser_profile_test.go:184` and `captureCheckOutput` (`internal/httpapi/amazon_browser_navigation_test.go:128`).

**Suggested work:** In a separate refactoring PR, add an exported `amazontest.CaptureOutput(t *testing.T) func() string` in `internal/amazon/amazontest`. Replace the three copies with it and add no behaviour change.

From review comment R1-02 on #216 (`doc/todo/issue-186-r1-02.md`).

## Comments by trusted authors

