# API step: multi-user login phase 1

Status: ready. Updated [API_SPECIFICATION.md](../../../API_SPECIFICATION.md): status line, the authentication convention, and the new section "Users and login".

## Contract changes and rationale

- New `GET /auth/session`, `POST /auth/login`, `POST /auth/logout`: the SPA needs the mode and the user to choose between the open app, the login page, the user app and the admin page (FR-MODE-001/002, FR-AUTH-001..003).
- New `GET` and `POST /admin/users`: administrator page list and add user (FR-ADMIN-002/003, D-20, D-24).
- New error codes `AUTH_REQUIRED`, `INVALID_CREDENTIALS`, `FORBIDDEN`, `LOGIN_LOCKED`, `INVALID_USERNAME`, `PASSWORD_TOO_SHORT`, `PASSWORD_TOO_LONG`, `USERNAME_TAKEN`.
- Existing endpoints keep their shapes; in protected mode they need a session and are scoped to the user. Another user's item returns 404 (same as unknown) so item ids are not revealed (FR-SHARE-001, D-19). Duplicate detection is per user (FR-SHARE-005).
- Open mode keeps the previous contract unchanged (D-25).
