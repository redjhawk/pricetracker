# Test cases: restore the clientAddress doc comment in auth.go (issue #293)

Sources: functional specification [session-cookie-security](../../specifications/session-cookie-security/functional.md) (FR-COOKIE-007 to FR-COOKIE-009). No new use cases: see [no-use-cases.md](no-use-cases.md); the unchanged behaviour is that of UC-COOKIE-001 to UC-COOKIE-003 and UC-LOGIN-LIM-001 to UC-LOGIN-LIM-003.

Conventions:

- **Base:** `origin/master` (or `master`), compared with the change applied on top of it.
- **The two `clientAddress` lines:** `// clientAddress returns the client address of login limits: the CF-Connecting-IP header when it is` and `// a bare IP address (the application is only reachable through Cloudflare), else the connection host.`
- **The two `secureRequest` lines:** `// secureRequest reports whether the request reached the application over HTTPS: directly over TLS,` and `// or through the reverse proxy, whose first X-Forwarded-Proto entry is https (trusted like CF-Connecting-IP).`
- **Existing tests:** `TestSessionCookie_TC001` to `TestSessionCookie_TC011` (`internal/httpapi/session_cookie_security_test.go`, FR-COOKIE-001–006) and `TestLoginLimits_*` (`internal/httpapi/login_attempt_limits_test.go`, including `TC009_CFConnectingIPIsTheClientAddress` and `TC010_ConnectionAddressWithoutAValidHeader`, and `internal/service/login_attempt_limits_test.go`).
- The change only moves source comment lines. FR-COOKIE-009 allows no other file change outside `doc/`, so no new Go or Playwright test code is written: the cases are repository checks (`go doc`, `gofmt`, `git diff`) and runs of the existing suites. Narrow screens, keyboard, empty results and screen states do not apply; no API, database or user interface changes.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-COOKIE-007 | TC-001, TC-003 |
| FR-COOKIE-008 | TC-002, TC-003 |
| FR-COOKIE-009 | TC-003, TC-004 |

| Use case (unchanged behaviour) | Test cases |
| --- | --- |
| UC-COOKIE-001, UC-COOKIE-002, UC-COOKIE-003 | TC-004 |
| UC-LOGIN-LIM-001, UC-LOGIN-LIM-002, UC-LOGIN-LIM-003 | TC-004 |

## Cases

### TC-001 — `clientAddress` has its own doc comment directly above it

- Covers: FR-COOKIE-007
- Level: repository check (`go doc`, source content)
- Given the change applied on top of the base
- When `go doc -u ./internal/httpapi clientAddress` runs, and the lines of `internal/httpapi/auth.go` just above `func clientAddress(r *http.Request) string {` are read
- Then `go doc` shows the signature followed by a doc comment starting "clientAddress returns the client address of login limits" and ending "else the connection host."; the two lines just above `func clientAddress` are exactly the two `clientAddress` lines, in that order, with no blank line between them and the function
- Corner cases:
  - the line above the two `clientAddress` lines is blank (they are not glued to the end of `secureRequest`);
  - the two `clientAddress` lines appear exactly once in the file (moved, not copied);
  - on the base, the same `go doc` command shows no doc comment for `clientAddress` (the check fails before the change).

### TC-002 — `secureRequest` keeps only its own doc comment

- Covers: FR-COOKIE-008
- Level: repository check (`go doc`, source content)
- Given the change applied on top of the base
- When `go doc -u ./internal/httpapi secureRequest` runs, and the comment block directly above `func secureRequest(r *http.Request) bool {` and the lines between the closing `}` of `secureRequest` and the `clientAddress` doc comment are read
- Then the doc comment is exactly the two `secureRequest` lines, starting "secureRequest reports" and containing no "clientAddress returns"; the line above them is blank (the end of the previous function, `allowMethod`); exactly one blank line separates the closing `}` of `secureRequest` from the first `clientAddress` line
- Corner cases:
  - "CF-Connecting-IP" in the `secureRequest` comment is its own wording and stays (only the `clientAddress` lines must be absent);
  - no second blank line and no stray comment line between the two functions;
  - on the base, `go doc` shows a `secureRequest` comment starting "clientAddress returns" (the check fails before the change).

### TC-003 — The diff outside `doc/` only moves the two comment lines

- Covers: FR-COOKIE-009, FR-COOKIE-007, FR-COOKIE-008
- Level: repository check (`git diff`, `gofmt`)
- Given the change applied on top of the base
- When `git diff --name-status <base> -- . ':!doc'`, `git diff --numstat <base> -- internal/httpapi/auth.go`, `git diff <base> -- internal/httpapi/auth.go` and `gofmt -l internal/httpapi` run
- Then the only file listed outside `doc/` is `M internal/httpapi/auth.go`; its numstat is `2 2`; the removed lines and the added lines are the same two `clientAddress` lines, byte for byte (no wording, spacing or punctuation change); no code line and no `secureRequest` comment line is added or removed; `gofmt -l` prints nothing
- Corner cases:
  - no file is added, deleted or renamed outside `doc/` (no new `_test.go`, frontend source, `.spec.ts`, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` change);
  - the order of the functions in `auth.go` is unchanged (`allowMethod`, `secureRequest`, `clientAddress`, `decodeCredentials`);
  - `doc/todo/issue-268-r1-01.md`, the review comment this issue comes from, is out of this check (under `doc/`).

### TC-004 — Session cookie and login limit behaviour unchanged, gates pass

- Covers: FR-COOKIE-009; UC-COOKIE-001, UC-COOKIE-002, UC-COOKIE-003, UC-LOGIN-LIM-001, UC-LOGIN-LIM-002, UC-LOGIN-LIM-003 (regression)
- Level: integration (existing HTTP handler and browser-like flow tests with real SQLite) and suite run (deterministic gates)
- Given the change applied on top of the base, with the existing tests unchanged
- When `go test -count=1 -run 'TestSessionCookie_|TestLoginLimits_' ./internal/httpapi/ ./internal/service/` runs, then `go vet ./...` and `go test ./...`, and the other gates (`node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`)
- Then every named test passes, as on the base: the `Secure` attribute of the login and logout cookie follows FR-COOKIE-001–006, and login attempt limits still use the `CF-Connecting-IP` address when it is a bare IP and the connection host otherwise; `go vet ./...` and `go test ./...` pass, and so do the other gates, with the same test list as on the base
- Corner cases:
  - the results are the same without cached results (`-count=1`) and when the gates run twice;
  - the operator observes no change in the application (no API, database, setting or screen change: follows from TC-003).
