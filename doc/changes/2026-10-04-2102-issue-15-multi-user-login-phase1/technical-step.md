# Technical step: multi-user login phase 1

Status: ready. Output: [technical.md](../../specifications/multi-user-login/technical.md) (one subject).

## Decisions

- Owner key `owner_id INTEGER NOT NULL DEFAULT 0` on `items`, `claude_token` and `leboncoin_session`; `0` is the open-mode owner. Reason: avoids NULL uniqueness problems and keeps per-owner queries simple.
- `items` is rebuilt once to replace `UNIQUE(canonical_url)` with `UNIQUE(owner_id, canonical_url)` (FR-SHARE-005). The single-row settings tables are rebuilt keyed by `owner_id`. Data is preserved as owner `0`.
- Passwords: stdlib `crypto/pbkdf2` (Go 1.25), no new dependency; bcrypt via x/crypto declined to avoid a dependency PR.
- Sessions: random token in an HttpOnly SameSite=Lax cookie, SHA-256 stored, fixed 30-day expiry. Lockout kept in memory.
- Mode is derived per request from the existence of a regular user. Admin sessions reach only auth and admin endpoints.
- Device command `pricefollower admin-password`; installer prints it; deploy wrapper gets an `--admin-password` option.

## Refactoring

R-1: thread an owner id through store and service signatures. Required for scoping; done inside the ownership PR, recorded in technical.md.

## Unresolved questions

None.
