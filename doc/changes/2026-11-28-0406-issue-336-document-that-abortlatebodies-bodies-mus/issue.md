# Issue #336 Document that abortLateBodies bodies must be read on the handler goroutine

**Problem:** `lateBody.Read` in `internal/httpapi/http_server.go:54` panics with `http.ErrAbortHandler`, which net/http recovers only on the goroutine serving the request. A future handler that reads `r.Body` from another goroutine (an `io.Copy` into a pipe, a proxied request body written by `http.Transport`) would crash the process on a slow client instead of closing one connection.

**Suggested work:** Add to the `abortLateBodies` comment (`internal/httpapi/http_server.go:38-39`) that the request body must be read on the handler's goroutine.

From review comment R2-03 on #306 (`doc/todo/issue-273-r2-03.md`).

## Comments by trusted authors

