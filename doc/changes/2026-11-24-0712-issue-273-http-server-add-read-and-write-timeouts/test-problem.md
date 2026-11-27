# Locked test still does not compile after the merge of master (second report)

The previous return to the tests stage (commit bfa2a54) did not fix this: the TDD red commit 9b76a06 relocked `internal/httpapi/http_server_limits_test.go` unchanged (same SHA-256 `51fd21eb…acce5`, last changed in 9df90ed). It only reverted the implementation.

`go vet ./...` and `go test ./...` fail for package `internal/httpapi`:

```
vet: internal/httpapi/http_server_limits_test.go:457:6: saveClaudeToken redeclared in this block
```

- `internal/httpapi/http_server_limits_test.go:457` (locked test of this change) declares
  `func saveClaudeToken(t *testing.T, fake *limitsNet, token string, verify limitsVerify) (*limitsServer, int, map[string]any, time.Duration)`.
- `internal/httpapi/amazon_item_ai_review_test.go:95`, merged from master with issue #305 (commit 9fa869b), declares
  `func saveClaudeToken(t *testing.T, database *store.Store, token *string)`.

Both are test files in package `httpapi`, so no production code can make the package's tests build, and the developer stage cannot edit test files.

Required fix in the tests stage: in `internal/httpapi/http_server_limits_test.go`, rename the helper (for example to `limitsSaveClaudeToken`) and update its call sites in that file, then relock it.

The blocking review comment R2-01 needs no further change: `API_SPECIFICATION.md` (15 s row), FR-HTTP-TMO-001 (`functional.md:26`) and `technical.md:57` already list `POST /api/v1/admin/users` / creating a user. FR-43 of the review is now FR-44 after the merge of master, and it lists no endpoints.
