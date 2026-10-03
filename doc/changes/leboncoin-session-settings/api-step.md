# API contract proposal

Status: **approved** by the user on 2026-10-03 (proposal revision 1, unchanged). Date: 2026-10-03.\
Canonical location: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section “LeBoncoin session settings (approved 2026-10-03)”, proposal revision 1. All previously approved sections are unchanged.

## Proposed changes

1. `GET /api/v1/settings/leboncoin-session` → `200 {"session": {value, revision, updatedAt, status, expiresAt, revokedAt, lastAttempt}}`; `500 INTERNAL_ERROR`.
2. `PUT /api/v1/settings/leboncoin-session` with `{"value": "<as typed>", "revision": <int>}` → `200` with the new object. Empty/whitespace value clears. Errors: `400 INVALID_JSON`, `400 INVALID_REQUEST`, `400 INVALID_SESSION` (specific safe message), `409 SESSION_CHANGED` (stale revision, nothing changed), `413 REQUEST_TOO_LARGE`, `405 METHOD_NOT_ALLOWED` (`Allow: GET, PUT`), `500 INTERNAL_ERROR`.
3. `status`: `none` | `active` | `expired` | `revoked`; `lastAttempt.outcome`: `accepted` | `rejected` | `failed`.
4. Item endpoints: no field/status change; new `lastAttempt.message` texts for LeBoncoin session failures (free text).

## Agreement with technical specifications

Matches [settings TS-LBC-SET-001–007](../../specifications/leboncoin-session-settings/technical.md) (shapes, parsing limits and messages, conflict token, hint rules) and [collection TS-LBC-COL-007–010](../../specifications/leboncoin-session-collection/technical.md) (when `revision`, `revokedAt`, `expiresAt` and `lastAttempt` change; item messages). The header menu, capture helper and file removal subjects use no endpoint.

## Approval record

No approval yet. The coordinator must present the proposal to the user and record the approving message and the approved proposal revision in [index.md](index.md) before any implementation of either tier. If the user requests changes, revise the pending section (new proposal revision) and the affected technical files.

## Approval

On 2026-10-03 the user answered the coordinator's question "Do you approve the proposed API contract (GET/PUT /api/v1/settings/leboncoin-session as described)?" with **"Approve"**. Approved revision: proposal revision 1, as written. The user also answered **"Yes, as proposed"** to the expiry clarification (passing `expiresAt` does not change `revision`, so it alone causes no 409).
