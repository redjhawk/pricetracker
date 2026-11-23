# Issue #251 Make the TC-003 fixture modules compile

**Problem:** In `internal/amazon/amazontest/server_dependencies_test.go`, `writeModule` (line 143) writes the stub helper as `package amazontest\n`, but the fixtures `helperMain` (line 162) and the TC-003 renamed-import row (line 235) use `amazontest.DownloadMarker` / `helper.DownloadMarker`. These fixture modules would not compile; the tests pass only because `go list -deps` does not type-check, and they would break if the listing gained a flag that compiles (for example `-export` or `-compiled`).

**Suggested work:** Add `const DownloadMarker = ""` to the stub in `writeModule`, or change both fixtures to blank imports as `helperTest` (line 166) already does.

From review comment R2-01 on #247 (`doc/todo/issue-244-r2-01.md`).

## Comments by trusted authors

