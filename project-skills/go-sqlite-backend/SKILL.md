---
name: go-sqlite-backend
description: Implement the PriceFollower backend in Go with SQLite, including API routes, price collection, scheduling, and deployment.
---

# Go and SQLite backend developer

Implement the backend in Go. Keep SQLite as the durable store and preserve the approved API contract in `API_SPECIFICATION.md` unless the user approves a contract change.

## Boundaries

- Use `net/http`, `database/sql`, and the repository's pure-Go SQLite driver.
- Keep HTTP handling, service rules, persistence, and the Amazon adapter separated in the existing `cmd/` and `internal/` structure. Avoid frameworks or interfaces that do not serve a current need.
- The React frontend only calls the HTTP API. It does not access SQLite or collect marketplace pages.
- Keep API changes behind the API-first confirmation process when a task changes behavior across frontend and backend.

## Data and API

- Keep integer euro cents and UTC timestamps, and return the approved item and error shapes.
- Preserve existing SQLite data when evolving the schema. Use parameterized SQL, foreign keys, cascade deletion, indexes, and transactions for related writes.
- Store successful observations separately from collection attempts. A failed check must not replace the last successful price.
- Validate and canonicalize HTTPS URLs for the supported euro Amazon marketplaces on the server. Reject other schemes, hosts, and listing forms.
- Return safe JSON errors without internal errors, HTML responses, or upstream page content.

## Collection and scheduling

- Collect a new item immediately after saving it, without making the POST wait for Amazon.
- Schedule recurring checks at the configured local times and continue after stale data and errors. Store the next scheduled attempt durably in SQLite.
- Bound network requests, follow only supported Amazon redirects, and classify success, request error, price-not-found, and unavailable outcomes.
- Keep Chrome request headers and price parsing in the Amazon adapter. Do not perform network I/O while holding a database transaction.
- Ensure shutdown cancels network work and closes the database cleanly.

## Development and release

- `npm run dev` runs Vite with the Go API and development seed data.
- Production builds embed the React `dist/` output into the Go executable and target the required ARM architecture. Keep the writable SQLite data directory separate from the executable.
- Do not add or run tests unless asked. Report builds and checks actually performed.
