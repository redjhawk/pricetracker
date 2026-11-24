# Issue #197 Return explicit zero values from BrowserCollector.profile when stale lock removal fails

**Problem:** In `internal/amazon/browser.go` `profile()` (around line 256), `return c.options.ProfileDir, func() {}, removeStaleLocks(c.options.ProfileDir)` returns a path and a cleanup together with a possible error. The `MkdirAll` branch just above returns `"", nil, err`.

**Suggested work:** Use the same explicit form in both branches:
```go
if err := removeStaleLocks(c.options.ProfileDir); err != nil {
	return "", nil, err
}
return c.options.ProfileDir, func() {}, nil
```

From review comment R1-04 on #194 (`doc/todo/issue-180-r1-04.md`).

## Comments by trusted authors

