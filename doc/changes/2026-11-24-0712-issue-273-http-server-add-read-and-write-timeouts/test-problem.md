# Locked test does not compile after the merge of master

`go vet ./...` and `go test ./...` fail for package `internal/httpapi`:

```
vet: internal/httpapi/http_server_limits_test.go:457:6: saveClaudeToken redeclared in this block
```

- `internal/httpapi/http_server_limits_test.go:457` (locked test of this change) declares
  `func saveClaudeToken(t *testing.T, fake *limitsNet, token string, verify limitsVerify) (*limitsServer, int, map[string]any, time.Duration)`.
- `internal/httpapi/amazon_item_ai_review_test.go:95`, merged from master with issue #305 (commit 9fa869b), declares
  `func saveClaudeToken(t *testing.T, database *store.Store, token *string)`.

Both files are in package `httpapi`, so the whole package's tests fail to build. The fix is to rename the helper in the locked test of this change (for example to `limitsSaveClaudeToken`, updating its call sites in `http_server_limits_test.go`), then relock the test. The developer stage cannot edit test files.

The blocking review comment R2-01 needs no further change: `API_SPECIFICATION.md` (15 s row), FR-HTTP-TMO-001 (`functional.md:26`) and `technical.md:57` already list `POST /api/v1/admin/users` / creating a user. FR-43 of the review is now FR-44 after the merge of master, and it lists no endpoints.
