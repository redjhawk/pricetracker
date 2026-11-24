# Issue #273 HTTP server: add read and write timeouts

**Problem:** The HTTP server in `cmd/pricefollower/main.go` sets only `ReadHeaderTimeout` and `IdleTimeout`. A client that sends its body or reads the response very slowly keeps its connection open without limit.

**Suggested work:** Set `ReadTimeout` (15 s), `WriteTimeout` (60 s: saving Settings waits up to 20 s for the Claude token verification) and `MaxHeaderBytes`. Check that no handler needs longer than the write timeout.


## Comments by trusted authors

