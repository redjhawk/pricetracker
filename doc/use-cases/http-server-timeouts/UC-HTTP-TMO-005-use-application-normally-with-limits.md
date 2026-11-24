# UC-HTTP-TMO-005 — Use the application normally with the time limits in place

**Primary actor:** Person using the application (regular user or administrator)
**Supporting systems:** Browser, PriceFollower application, marketplaces and Claude (background work)
**Trigger:** The actor uses the application at normal network speed.
**Origin:** Issue #273: the new read and write limits must not break any normal request; no handler may need longer than the limits.
**Covers:** FR-HTTP-TMO-006, FR-HTTP-TMO-007

## Preconditions

- The application is running (development or production mode) and reachable at normal speed.
- The actor is logged in.

## Main flow

1. The actor opens the application; every page and asset loads with its usual response (FR-HTTP-TMO-006).
2. The actor adds an item; the application answers within 5 s while the first check runs in the background (FR-HTTP-TMO-006).
3. The actor opens the item and selects Refresh price; the application answers within 5 s while the check runs in the background (FR-HTTP-TMO-006).
4. The actor selects Refresh all on the tracked items; the application answers at once while the checks run in the background (FR-HTTP-TMO-006).
5. The actor starts an AI review; the application answers within 5 s while the review runs in the background (FR-HTTP-TMO-006).
6. The actor leaves the browser open without using it; an idle kept-alive connection is closed after about 60 s, as before, and the next action opens a new connection without any visible effect (FR-HTTP-TMO-007).

## Alternatives and errors

- **Check or review takes longer than the limits:** a marketplace check or AI review that takes over 60 s does not run inside a request, so no response is cut off; its result appears as today.
- **Saving the Claude token:** see UC-HTTP-TMO-004.
- **Very slow client:** see UC-HTTP-TMO-001 and UC-HTTP-TMO-002.

## Postconditions

- Every normal request completed with its usual response.
- No screen, message, setting or loading state changed.
