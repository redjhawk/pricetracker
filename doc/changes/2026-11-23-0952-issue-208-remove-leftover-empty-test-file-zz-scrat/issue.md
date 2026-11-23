# Issue #208 Remove leftover empty test file zz_scratch_docs_test.go

**Problem:** `internal/service/zz_scratch_docs_test.go` (line 3) is an empty leftover of the issue #204 test stage. It holds only a comment saying it is safe to delete and adds noise to the `service` package.

**Suggested work:** Delete `internal/service/zz_scratch_docs_test.go` and remove its entry from `doc/changes/2026-11-22-2139-issue-204-retrieval-error/tests.lock` (if the lock is still checked), then run `go test ./internal/service/...`.

From review comment R1-03 on #206 (`doc/todo/issue-204-r1-03.md`).

## Comments by trusted authors

