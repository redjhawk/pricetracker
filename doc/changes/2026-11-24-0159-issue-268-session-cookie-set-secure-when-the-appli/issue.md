# Issue #268 Session cookie: set Secure when the application is served over HTTPS by a reverse proxy

**Problem:** `setSessionCookie` (`internal/httpapi/auth.go`) sets `Secure` only when the Go server itself received TLS (`r.TLS != nil`). In production TLS ends at the reverse proxy (Cloudflare), so the session cookie is never marked `Secure`.

**Suggested work:** Also set `Secure` when the request carries `X-Forwarded-Proto: https`. Plain HTTP access in development must keep working.


## Comments by trusted authors

