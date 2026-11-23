# Locked test problem: unrelated reformatting in `internal/service/amazon_browser_retrieval_test.go`

Blocking review comment R1-01 asks to revert a hunk in `internal/service/amazon_browser_retrieval_test.go:221`. That hunk is not implementation code. It comes from the TDD red commit 623f2cf ("chore(factory): stop at tests (gate)"), and the file is listed in `tests.lock` (sha256 `1b02496e…`, which matches the TDD red version). The developer stage cannot edit locked test files: the factory restores them to the TDD red commit. So only the tests stage can apply this fix.

The hunk wraps the one-line `B0CAUSEG01` func literal of the existing #204 test TC-007 over three lines:

```go
"B0CAUSEG01": func() (int, string) { return http.StatusNotFound, "<html><body><h1>Page introuvable</h1></body></html>" },
```

became

```go
"B0CAUSEG01": func() (int, string) {
	return http.StatusNotFound, "<html><body><h1>Page introuvable</h1></body></html>"
},
```

The change is not needed: `git show master:internal/service/amazon_browser_retrieval_test.go | gofmt -l` prints nothing, so the master form is already gofmt-clean. No #226 test case touches TC-007, and the technical specification says the existing #204 tests stay unchanged.

Requested fix (tests stage): restore `internal/service/amazon_browser_retrieval_test.go` to its `master` content (`git checkout master -- internal/service/amazon_browser_retrieval_test.go`), so the file has no diff against master. Then drop it from `tests.lock` or re-lock it with the master hash. The #226 tests are in `internal/amazon/browser_snap_test.go`, `internal/service/amazon_browser_snap_test.go` and `internal/service/amazon_browser_snap_docs_test.go`, and they do not depend on this file's change.
