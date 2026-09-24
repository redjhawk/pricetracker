---
name: node-sqlite-backend
description: Build or update this application's Node.js backend and SQLite persistence, including API endpoints, scheduled jobs, data access, and Amazon price collection.
---

# Node.js and SQLite backend developer

Implement backend changes in Node.js using the repository's existing framework, module system, and SQLite library. SQLite is the durable source of truth for tracked items and price observations. Inspect the codebase and existing dependencies before proposing replacements or adding packages.

## Application context

V1 is a single shared service with no login or per-user data. The server owns item storage and collection; the React frontend reads server-provided data. The initial source is European Amazon listings, with approximately 100 tracked items. Each active item is checked twice per day. Prices are item prices in euros. Keep observations while an item is tracked; deleting an item also deletes its history. The frontend needs the latest successful price, its timestamp, up to the last three detections, and collection status. Retries continue after stale data or failures. Other marketplaces and user features are future scope.

## Persistence and data integrity

- Keep database access behind a focused data-access/service layer. Do not let route handlers assemble ad-hoc SQL throughout the application.
- Use parameterized statements for all values. Never interpolate user input into SQL identifiers or query text.
- Define schema changes as ordered, repeatable migrations or the project's established migration mechanism. Preserve existing data during migrations unless a destructive change is explicitly required.
- Use foreign keys and appropriate indexes. Ensure item deletion removes dependent observations, using `ON DELETE CASCADE` or an explicit transaction consistent with the existing schema.
- Store prices without floating-point arithmetic. Prefer integer euro cents if the supported Amazon price precision permits it; otherwise use a documented exact decimal representation. Store currency explicitly or document the v1 EUR invariant.
- Store observation and scheduling timestamps in UTC in a consistent, sortable representation. Convert to local time only for display.
- Use transactions when a user-visible operation must update multiple related records atomically.
- Make connection lifecycle, busy timeout, and SQLite journal/concurrency settings explicit and compatible with how the service is deployed. Avoid holding write transactions during network requests.
- Define uniqueness/deduplication for canonical listing URLs so repeat submissions do not create accidental duplicate tracked items. Preserve a usable source URL for display and navigation.

## API and application behavior

- For any feature that requires changes to both backend and frontend, use API-first development. Draft the contract and present it to the user for confirmation before implementing either tier. Wait for confirmation; then implement backend and frontend against the approved contract. Backend-only work using an existing confirmed API can proceed without a new contract approval.
- A proposed API contract should specify endpoint and method, request/response schemas, validation and error behavior, and status semantics needed by the client. Keep it scoped to the feature and reconcile it with existing contracts before proposing changes.
- Inspect the frontend/API contract before changing response shapes. Keep API responses explicit and stable, with clear validation and error responses.
- Validate and normalize incoming Amazon URLs server-side; do not trust frontend validation. Restrict accepted hosts to the configured European Amazon marketplace set.
- Separate HTTP handlers, business logic, persistence, scheduling, and marketplace-specific collection code so each can evolve independently.
- Return the latest successful price separately from the most recent collection attempt. A failed attempt must never erase or replace a valid observation.
- Provide the frontend enough status and timestamps to distinguish pending, active, stale, unavailable, and retrieval-error states without exposing internal stack traces.
- On deletion, remove the item and associated history from SQLite and make the action idempotent or return a clear not-found response according to the existing API convention.

## Scheduled price collection

- Implement two daily checks per active item using the application's chosen timezone and schedule. Make the schedule and timezone configurable or centralized rather than scattered through code.
- Ensure a restart does not silently duplicate jobs or leave items unscheduled. Keep the schedule durable or reconstructable from SQLite.
- Isolate marketplace fetching/parsing behind an Amazon-specific adapter. Record each attempt's outcome and timestamp; write a price observation only when a valid euro item price is successfully extracted.
- Never hold a database transaction open while making network requests. Fetch first, then persist attempt outcome and observation atomically as appropriate.
- Continue retrying stale items and transient failures; use bounded, configurable retry/backoff for immediate retries if the existing scheduler supports it. Do not stop long-term scheduled checks merely because a listing failed repeatedly.
- Set network timeouts and handle redirects, response parsing errors, missing prices, unavailable listings, and transient failures explicitly. Keep logs useful but avoid dumping full HTML, credentials, or unnecessary personal data.
- Requests are expected to identify as a Google Chrome browser per the product requirement. Keep request headers/configuration in the collector adapter. Do not treat this as a guarantee that a source is accessible; surface blocked or changed-source failures as collection outcomes.
- On price observations, retain enough provenance to diagnose results (item ID, detected price, currency, observed-at time, and relevant outcome/source metadata). Avoid collecting unnecessary page contents.

## Security and operations

- Bind network access and allowed hosts deliberately. Prevent arbitrary URL fetching and redirects to private/local network addresses when handling submitted URLs.
- Keep database files, journals, and backups out of public/static directories. Do not commit live database files or secrets.
- Use structured logging for item/job IDs and outcome categories; redact sensitive headers and avoid logging entire upstream responses.
- Handle process shutdown cleanly so scheduled work and SQLite connections close predictably.
- Follow the repository's existing configuration and deployment practices. Do not introduce a new framework, ORM, scheduler, or migration tool without a concrete need.

## Verification

When asked to verify a change, check the affected API/data flows and migration behavior using the repository's available commands. Do not add or run tests unless the user asks. Report which checks were actually performed and any unverified deployment assumptions.
