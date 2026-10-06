# Functional step: multi-user login phase 1

Status: ready. Sources: [functional.md](../../specifications/multi-user-login/functional.md) and [delivery-plan.md](../../specifications/multi-user-login/delivery-plan.md), both user-approved and unchanged by this step. Decisions D-19 to D-26 override conflicting text.

## In scope

| Step | Requirement IDs | Acceptance criteria | Phase 1 adjustment |
| --- | --- | --- | --- |
| 1. Device script | FR-ADMIN-001 (partial), D-9 (device part), D-14, D-22 | FR-ADMIN-001 row | Script generates and prints a random password (D-22 overrides the typed password of FR-ADMIN-001/D-1); creates `admin`; resets `admin` only; a reset ends admin sessions (D-14). Not available from the browser. |
| 2. Open mode | FR-MODE-001, D-25 | FR-MODE-001 row | "Log in" link in the header for the administrator. |
| 2. Protected mode | FR-MODE-002, D-26 | FR-MODE-002 row | Starts when the first regular user is created. |
| 3. Login and session | FR-AUTH-001 to FR-AUTH-005, D-8, D-11 | FR-AUTH-001 to 005 rows | None. |
| 4. Administrator page | FR-ADMIN-002, FR-ADMIN-003, FR-ADMIN-007 (cannot be removed), D-10, D-17, D-20, D-24 | FR-ADMIN-002 and 003 rows | List shows username and last login only (D-24). Password typed by the administrator, at least 12 characters, not temporary, not changeable afterwards (D-20, D-21). Duplicate username (case-insensitive) is refused. |
| 5. Item ownership | FR-SHARE-001 (without sharing option), FR-SHARE-005, D-6, D-19 | FR-SHARE-001 and 005 rows | Each user sees and manages only their own items. |
| 6. Per-user settings | FR-SETTINGS-001, D-4, D-23 | FR-SETTINGS-001 row | None. |
| 7. Existing data | FR-SHARE-006, D-5, D-26 | FR-SHARE-006 row | Inherited items are owned; no sharing state in phase 1. |
| Deployment | User instruction 2026-10-04 | Deploy scripts and deploy documentation cover running the device script | Installation support for FR-ADMIN-001; no new application behavior. |

## Excluded from phase 1

- In-app password change and forced change at first login: FR-AUTH-010, FR-AUTH-012, FR-AUTH-011 (except the 12-character minimum), FR-ADMIN-004, password-change part of FR-ADMIN-007, D-12, in-app part of D-9 (D-21).
- Reset of regular users' passwords by anyone, including the device script (D-22).
- User removal and renaming: FR-ADMIN-005, FR-ADMIN-006, D-2, D-16.
- Item sharing: FR-SHARE-002 to 004, sharing option of FR-SHARE-001, D-3, D-7, D-15, D-18 (D-19).
- User list columns other than username and last login (D-13 except those two, D-24).

## Precedence notes (no user question needed)

- FR-ADMIN-001 / D-1 (operator types the password) vs D-22 (script generates a random password): D-22 applies, as the delivery plan states.
- FR-ADMIN-001 acceptance ("without the command run, nobody can log in as administrator") remains valid under D-22.
- FR-ADMIN-002 "the only screen is the user list" includes the add-user action of D-20 on the same page (delivery plan step 4).

## Unresolved questions

None blocking. Exact message wording (for example the FR-AUTH-005 wait message) is left to technical specification within the stated behavior.

## Handoff to technical specifier

Use the requirement IDs above, decisions D-1 to D-26 with D-19 to D-26 precedence, and the deploy update instruction.
