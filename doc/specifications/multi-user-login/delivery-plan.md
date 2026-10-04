# Delivery plan: multiple local users and login

Status: ready (phase 1)
Owner: functional specification agent
User decision/reference: user request in conversation (2026-10-04). Extends [the functional specification](functional.md); requirement and decision IDs refer to it.

The user asked (2026-10-04) to deliver this feature in phases. Only phase 1 is approved for development; later phases are re-confirmed with the user before they start. Each phase goes through the full workflow (technical specification, API contract, implementation, review, decisions, PRs of at most 500 changed lines, QA).

Phase 1 changes some decisions of the functional specification; where they differ, phase 1 rules apply until later phases are confirmed:

- D-19: no item sharing between users. Each user sees and manages only their own items. FR-SHARE-002, FR-SHARE-003, FR-SHARE-004 and D-3, D-7, D-15, D-18 are postponed.
- D-20: the administrator creates a user on the administrator page by typing a username and the user's password. The administrator cannot change it afterwards.
- D-21: in phase 1, nobody changes a password in the application, neither users nor the administrator; there is no forced change at first login. FR-AUTH-010, FR-AUTH-011 (except the 12-character minimum), FR-AUTH-012, FR-ADMIN-004, D-12 and the in-app part of D-9 are postponed. The user is meant to choose their own password in a later phase.
- D-22: a device script generates a random password and prints it. It creates the `admin` account and resets the administrator's password only. It cannot reset other users' passwords (user request, 2026-10-04).
- D-23: Claude token and LeBoncoin session are per user (D-4); the first user created inherits the existing ones (D-5).
- D-24: the administrator's user list shows username and last login only.
- D-25: open mode (user request, 2026-10-04). While no regular user exists, the application works as it does today: no login is required, and all pages and APIs are open. The `admin` account alone does not change this. In open mode, a "Log in" link in the header lets the administrator log in to create users.
- D-26: protected mode starts when the first regular user is created. From then on, login is required (FR-AUTH-001). Items, the Claude token and the LeBoncoin session used in open mode go to that first user (D-5).

## Phase 1: login and user creation

Steps, in order:

1. Device script (D-22): creates the `admin` account with a generated password, and resets the password of `admin` only to a new generated one, printed on the device. A reset ends the administrator's open sessions (D-14).
2. Open mode (D-25, D-26): without regular users, no login is required and the application behaves as today.
3. Login and session, in protected mode: FR-AUTH-001, FR-AUTH-002, FR-AUTH-003, FR-AUTH-004 (30 days), FR-AUTH-005 (5 failures, 1 minute).
4. Administrator page: the administrator's only screen; user list with username and last login (FR-ADMIN-002, D-24); add user with username (D-17) and password typed by the administrator, at least 12 characters (D-20, D-10).
5. Item ownership: each item belongs to the user who added it; users see and manage only their own items; the same URL for two users gives two independent items (FR-SHARE-001 without the sharing option, FR-SHARE-005, D-19).
6. Per-user settings: Claude token and LeBoncoin session per user, used for that user's items (FR-SETTINGS-001, D-23).
7. Existing data: the first user created inherits existing items and settings (FR-SHARE-006, D-5, D-26).

Not in phase 1: any password change in the application, any reset of a regular user's password (a forgotten user password cannot be recovered until phase 2), user removal and renaming (FR-ADMIN-005, FR-ADMIN-006), item sharing, and the user list columns other than username and last login.

## Later phases (to be confirmed)

- Phase 2: users choose and change their own password; user removal and renaming.
- Phase 3: item sharing between users.
