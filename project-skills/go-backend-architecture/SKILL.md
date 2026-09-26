---
name: go-backend-architecture
description: Choose and preserve the simple layered architecture of the PriceFollower Go backend.
---

# Go backend architecture

Use one Go process and one SQLite database. Keep the design small and keep the existing API contract stable.

## Structure

```text
cmd/pricefollower/       # process startup, configuration wiring, shutdown
config/                  # environment configuration
internal/httpapi/        # HTTP routing, validation, JSON and static files
internal/service/        # item use cases, collection workflow, schedule
internal/store/          # SQLite schema, SQL queries, row mapping
internal/amazon/         # Amazon URL validation, fetch, and parsing
internal/model/          # API/domain data structures
web/                     # embedded frontend assets
```

Keep request handlers free of SQL and scraping. Keep the collector free of persistence and HTTP response concerns. Let scheduled jobs call the same service methods as user actions where applicable.

## Data flow

```text
HTTP handler -> service -> store -> SQLite
                    |
                    +-> Amazon collector -> service -> store

Scheduler -> service -> Amazon collector -> service -> store
```

- Use context cancellation for requests, collection work, and shutdown.
- Never hold a transaction during an upstream request.
- Keep persistent observations and next-check scheduling in SQLite so restart resumes tracking.
- Use the embedded React build for production and Vite proxying to the Go API for development.
- Build a single Linux ARM executable with the frontend embedded; store mutable data outside that binary.

## Scope

V1 is a shared, no-login tracker for European Amazon listings. It returns latest price, up to three detections, and collection state. Do not add authentication, alerts, trends, other platforms, or infrastructure layers without a user requirement.

For any frontend/backend behavior change, draft and receive approval for the API contract before implementing either tier. Backend-only changes that preserve the approved API may proceed directly.
