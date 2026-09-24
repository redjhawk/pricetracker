---
name: backend-architecture
description: Choose, preserve, or evolve the structure and data flow of this application's Node.js and SQLite backend. Use for backend architecture decisions and cross-feature organization, not routine isolated route changes.
---

# Backend architecture

Use one Node.js backend application with SQLite. Keep a simple layered structure so HTTP handling, price-tracking rules, persistence, and Amazon access have clear boundaries. Do not split v1 into services or processes without a concrete operational need.

## Structure

Organize the backend around these responsibilities, adapting directory names to existing repository conventions:

```text
src/
  app/                  # application creation and startup
  routes/               # HTTP route registration
  controllers/          # translate HTTP requests/responses
  services/             # item use cases and price-tracking rules
  repositories/         # SQLite queries and persistence
  db/                   # connection setup and schema migrations
  collectors/           # Amazon fetch and price extraction adapter
  jobs/                 # twice-daily scheduling and retry orchestration
```

Keep this as a guide to boundaries, not a requirement to create empty folders. Combine closely related files where that is simpler, but do not put SQL, collection network calls, and HTTP response logic in the same module.

## Request and collection flows

HTTP flow:

```text
Route -> Controller -> Service -> Repository -> SQLite
                       |
                       +-> Amazon collector (when a use case needs collection)
```

Scheduled flow:

```text
Scheduler -> Service -> Amazon collector -> Service -> Repository -> SQLite
```

- Routes define paths and methods and delegate to controllers.
- Controllers validate/translate the transport-level request and response. They do not contain SQL or business workflows.
- Services implement use cases such as list items, add an item, delete an item, collect a price, and record an attempt. The scheduler and HTTP controllers call the same service layer where appropriate.
- Repositories encapsulate SQLite statements and row mapping. Use parameterized SQL; do not expose a database connection to controllers or collectors.
- The Amazon collector is an adapter for URL fetching and extraction. It returns a normalized result or typed failure; it does not write to SQLite or format HTTP responses.
- The scheduler starts once with the application, invokes service-level collection, and is configured centrally. Do not open a database transaction around a network request.

## V1 data and behavior

The service has one shared collection, no login, and no per-user ownership. SQLite stores tracked European Amazon listing URLs, item metadata, successful euro item-price observations, and collection-attempt status as needed. The frontend needs latest successful price/time, the last three successful detections, and clear pending/stale/unavailable/error status.

Keep prices exact (for example, integer euro cents where suitable), timestamps in UTC, and use foreign keys so deleting a tracked item deletes its observations. Continue twice-daily checks and retries after stale data or failures. Avoid speculative generic domain layers; keep Amazon-specific behavior behind the collector boundary so later platforms can be added deliberately.

## API-first coordination

When a requested feature requires both backend and frontend changes, define the API contract first and present it to the user for confirmation. Do not implement either tier before confirmation. Include endpoint/method, request and response shapes, validation and errors, and client-visible status semantics. Once confirmed, implement backend and frontend against that contract. Backend-only work using an existing confirmed API does not require a new approval.

Keep response schemas explicit and stable. The API must distinguish the latest successful price from the most recent collection attempt; a failure never overwrites a valid observation. Validate Amazon URLs on the server and only accept configured European Amazon hosts. Do not expose internal errors or stack traces to clients.

## Keep it simple

- Follow the existing Node.js framework, module system, SQLite library, and migration convention. Add a dependency only when it materially simplifies a requirement.
- Use one backend process and one SQLite database for v1. Do not add a message broker, ORM, microservices, or a separate worker process without a demonstrated need.
- Keep schema migrations explicit and data-preserving. Avoid speculative abstractions and repository interfaces that have only one implementation and no useful boundary.
- Keep operational configuration centralized and use clean startup/shutdown handling for the scheduler and database.
