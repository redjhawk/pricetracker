# Issue #253 Check the build failure message of fixture rows f and g

**Problem:** In `internal/amazon/amazontest/server_dependencies_test.go:323-326`, the `default` branch of `TestFixtureModulesBuild_Issue251_TC001` accepts any `go build ./...` failure for the invalid fixtures (rows f and g). A failure unrelated to the fixture would be accepted as well.

**Suggested work:** In that branch, also require `strings.Contains(err.Error(), tc.fails)` (`pricefollower.local/internal/nope` for f, `main.go` for g), as the control subtest already does for `DownloadMarker`. Confirm that `go build` prints these texts, then run `go test -count=1 ./internal/amazon/amazontest/...`.

From review comment R1-01 on #252 (`doc/todo/issue-251-r1-01.md`).

## Comments by trusted authors

