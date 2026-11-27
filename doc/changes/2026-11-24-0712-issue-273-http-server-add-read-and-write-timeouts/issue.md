# Issue #273 HTTP server: add read and write timeouts

**Problem:** The HTTP server in `cmd/pricefollower/main.go` sets only `ReadHeaderTimeout` and `IdleTimeout`. A client that sends its body or reads the response very slowly keeps its connection open without limit.

**Suggested work:** Set `ReadTimeout` (15 s), `WriteTimeout` (60 s: saving Settings waits up to 20 s for the Claude token verification) and `MaxHeaderBytes`. Check that no handler needs longer than the write timeout.


## Comments by trusted authors

### redjhawk at 2026-11-24T22:06:27Z

Done: R1-01 and R1-02 applied as listed (all 11 edits) on `ai-dev/issue-273` in 739d2c5. Please resume.

### redjhawk at 2026-11-27T20:36:22Z

Done (owner step for R2-01), pushed to `ai-dev/issue-273` as 812af07.

`API_SPECIFICATION.md` line 706 (HTTP server limits table, "Whole request (headers and body) received" row) now lists `POST /api/v1/admin/users` between login and items: `(POST /api/v1/auth/login, POST /api/v1/admin/users, POST /api/v1/items, ...`. No other files changed. I checked that `abortLateBodies` wraps the whole mux (`server.go:32`) and that the admin users handler reads its body through `decodeCredentials` (`auth.go:162`), so the code already matches the spec.

Please resume the implement stage.

