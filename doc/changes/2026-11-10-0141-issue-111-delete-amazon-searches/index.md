# Delete Amazon searches (issue #111)

Stage: api-contract ready (next: development)
State: functional, technical and API contract ready; no blocking functional question.

User request (GitHub issue #111, redjhawk):

> Functionality 'Amazon Search' is not working. You can delete it

## Scope

Remove the Amazon searches feature introduced by [issue #56](../2026-10-07-1117-issue-56-amazon-searches/index.md): tab, saved searches, search-only items and their data, search pacing and Amazon AI reviews. Keep tracked Amazon items unchanged, and replace the global Amazon stop with a per-item stop restarted with **Refresh price**. Revert the Claude token help text to LeBoncoin only. Cross-tier (frontend, backend, API, data).

## User functional decisions (redjhawk, relayed by the coordinator)

- D-1: Delete the "Amazon searches" tab and functionality, the saved searches and their search-to-item links.
- D-2: Items found only through a search (never moved to the tracked Amazon list) are deleted with their data (price history, AI reviews). AI reviews only exist for search items, so Amazon AI reviews go away.
- D-3: Tracked Amazon items, including items moved from a search, keep working as today (price history kept), no changes.
- D-4: Keep "stop after 5 consecutive Amazon failures", now for tracked Amazon items. Restart is item by item: after the stop, **Refresh price** on one item checks it; if it succeeds only that item restarts; every other stopped item stays stopped until its own Refresh price succeeds. Each item has its own stopped state. While stopped, the Amazon tab shows a warning telling the user to restart with Refresh price; scheduled 08:00/20:00 checks and "Refresh all prices" skip stopped items.
- D-5: Claude token help text: AI reviews are for LeBoncoin items only. AI reviews for Amazon items will be a new GitHub issue (out of scope).
- D-6: Search-only pacing and time windows (22:00–01:00, 06:00–08:00 Paris) go away with searches.

## Subjects

- [Amazon searches removal](../../specifications/amazon-searches-removal/functional.md) (FR-AMZ-SRCH-DEL-001–007)
- [Amazon request stop](../../specifications/amazon-request-stop/functional.md) (FR-AMZ-STOP-001–010)
- [Claude token settings](../../specifications/claude-token-settings/functional.md) amendment (FR-CLT-SET-013 supersedes FR-CLT-SET-012)
- Superseded: [amazon-searches](../../specifications/amazon-searches/functional.md), [amazon-human-browsing](../../specifications/amazon-human-browsing/functional.md), [amazon-ai-review](../../specifications/amazon-ai-review/functional.md).

## Technical specifications

- [Amazon searches removal technical](../../specifications/amazon-searches-removal/technical.md) (TS-AMZ-SRCH-DEL-001–006)
- [Amazon request stop technical](../../specifications/amazon-request-stop/technical.md) (TS-AMZ-STOP-001–008, with the 12-part implementation plan)
- [Claude token settings technical](../../specifications/claude-token-settings/technical.md) TS-CLT-SET-009 supersedes TS-CLT-SET-008
- Superseded: amazon-searches, amazon-human-browsing, amazon-ai-review technical specifications.

## API contract changes and rationale

[API_SPECIFICATION.md](../../../API_SPECIFICATION.md):

- Removed `/api/v1/amazon-searches…` routes, `tracked`, `ITEM_NOT_TRACKED`, Amazon `aiReview` objects, search-item reuse on add and untrack on delete: the feature is deleted (D-1, D-2, D-5); no client other than this frontend uses them.
- Removed `GET /api/v1/amazon/requests` and `409 AMAZON_REQUESTS_STOPPED`; added item field `amazonStoppedAt`: the stop is now per item (D-4), so the item carries its state, the warning is derived from the items list without an extra request, and Refresh price must be accepted on a stopped item to restart it.
- The "Amazon searches (2026-10-07)" section is kept with a removal banner instead of being deleted, to keep the documentation part of the stacked PRs under the size limit and keep the history readable; the banner states it is not part of the current contract.

## Out of scope

- AI reviews for Amazon items (new GitHub issue to be opened by the coordinator).
- Any change to LeBoncoin collection or to tracked Amazon item display other than the stop.
