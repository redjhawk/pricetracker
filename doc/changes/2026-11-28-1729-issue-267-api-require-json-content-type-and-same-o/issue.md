# Issue #267 API: require JSON content type and same-origin requests on state-changing routes

**Problem:** The `POST`, `PUT` and `DELETE` API routes decode the body as JSON whatever its `Content-Type` and do not look at the `Origin` header (`internal/httpapi/server.go`, `settings.go`, `purchase_goal.go`, `auth.go`). In open mode there is no session cookie either, so nothing ties a state-changing request to the application's own pages.

**Suggested work:**
- Refuse state-changing requests with a body whose `Content-Type` is not `application/json` (`415`).
- Refuse state-changing requests whose `Origin` header is present and is not the application's own origin (`403`). The application runs behind a reverse proxy (Cloudflare), so compare with the `Host` the request was sent to.
- Document both rules in `API_SPECIFICATION.md`. The frontend already sends `Content-Type: application/json` (`src/api/client.ts`).


## Comments by trusted authors

