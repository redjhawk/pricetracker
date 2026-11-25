# Issue #344 Guard the analyzer-findings subtest against going vacuous

**Problem:** In `internal/amazon/amazontest/server_dependencies_test.go` (subtest `analyzer findings in non-test and test files do not fail the check`, about line 403), the test only checks that `buildModule` returns nil. Nothing in the file checks that the fixture really has analyzer findings. If a later edit removes them (for example someone "fixes" the `Printf` or renames `Testfoo`), or a toolchain update stops reporting one, the subtest still passes but proves nothing. The file's other negative paths have in-file controls (the `DownloadMarker` and `amazontest.Nope` controls).

**Suggested work:** In the same subtest, after the `buildModule` check, run `go vet ./...` with all analyzers on the same `dir` (`exec.Command("go", "vet", "./...")`, `cmd.Dir = dir`, `cmd.Env = append(os.Environ(), fixtureEnv...)`). Fail with `t.Error("the fixture has no analyzer finding, so the subtest proves nothing")` if that command succeeds. Optionally, also check that the output holds `wrong type string`, `passes lock by value` and `malformed name`, keeping in mind that exact analyzer messages can change between Go releases.

From review comment R1-01 on #343 (`doc/todo/issue-340-r1-01.md`).

## Comments by trusted authors

