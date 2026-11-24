# No new automated tests for issue #293

The change only moves two source comment lines in `internal/httpapi/auth.go` (the `clientAddress` doc comment, now at lines 186-187 above `secureRequest`, back to directly above `func clientAddress`). It changes no behaviour, so no test can fail before it and pass after it.

FR-COOKIE-009 and TS-COOKIE-008 also forbid any change outside `doc/` other than `auth.go`: no `_test.go`, `.spec.ts` or other file may be added or changed. The technical specification (section "Issue #293", "Tests") tells this stage to write this file.

Where each test case is proved (development stage and gates):

| Test case | Proved by |
| --- | --- |
| TC-001 | `go doc -u ./internal/httpapi clientAddress` shows the comment "clientAddress returns the client address of login limits … else the connection host." (none on `master`); `grep -c 'clientAddress returns the client address' internal/httpapi/auth.go` is `1`; the line above the two lines is blank and the line after them is `func clientAddress(r *http.Request) string {`. |
| TC-002 | `go doc -u ./internal/httpapi secureRequest` shows only the two `secureRequest` lines (no "clientAddress returns"); the line above them is blank; exactly one blank line follows the closing `}` of `secureRequest`. |
| TC-003 | `git diff --name-status master -- . ':!doc'` lists only `M internal/httpapi/auth.go`; `git diff --numstat master -- internal/httpapi/auth.go` is `2 2`, with the same two lines removed and added; `gofmt -l internal/httpapi` prints nothing. |
| TC-004 | Existing, unchanged tests: `TestSessionCookie_TC001`–`TC011` (`internal/httpapi/session_cookie_security_test.go`) and `TestLoginLimits_*` (`internal/httpapi/login_attempt_limits_test.go`, `internal/service/login_attempt_limits_test.go`), run with `go test -count=1 -run 'TestSessionCookie_\|TestLoginLimits_' ./internal/httpapi/ ./internal/service/`; then the gates `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`. |
