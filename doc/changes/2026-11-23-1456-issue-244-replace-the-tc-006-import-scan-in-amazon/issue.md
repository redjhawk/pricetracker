# Issue #244 Replace the TC-006 import scan in amazontest_test.go with a dependency check

**Problem:** `TestAssertNoDownload_TC006_ChangeLeavesApplicationUntouched` (`internal/amazon/amazontest/amazontest_test.go:370-401`) walks `../../../cmd` and `../../../internal` and fails when any non-`_test.go` file contains the amazontest import string. A future test-support helper file (for example `internal/httpapi/httpapitest/*.go`) could legitimately import amazontest without the server binary depending on it, and this test would then fail. A text search also does not show what actually reaches the binary. The technical specification (`doc/specifications/test-suite-hygiene/technical.md`, test case mapping) maps TC-006 to a repository diff check plus the gates, not to a unit test.

**Suggested work:** Remove the import scan from the package test. If the guarantee is wanted, check it with `go list -deps ./cmd/...` (assert that `pricefollower.local/internal/amazon/amazontest` is absent) in a repository check or CI script, and record it in the TC-006 mapping.

From review comment R1-01 on #243 (`doc/todo/issue-232-r1-01.md`).

## Comments by trusted authors

