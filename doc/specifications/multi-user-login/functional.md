# Functional specification: multiple local users and login

Status: ready
Owner: functional specification agent
User decision/reference: user request in conversation (2026-10-04). Answers recorded as D-1–D-18 and D-25.

## Request

> I need a multiple local user. The user can choose to share the item or not. Other users can choose to list items added by other users. By now, I can have a simple way to login users, no need to use OAuth. There will be a global administrator that only will be able to handle users. For this user, the only screen he will see will be a list of users. He should be able to reset a password and assign a new one. He will also be able to add or remove users. When a user is added, he will be asked to change its password.

## Purpose and scope

Replace today's open single-operator access with local accounts. Each user owns the items they add, decides whether each item is shared, and can choose to also see items shared by others. A single global administrator manages accounts only.

Actors:

- **User**: a regular account that tracks items.
- **Administrator**: a single global account that manages users and does not track items.

Included: login and logout with username and password; mandatory password change on first login; item ownership; per-item sharing; listing items shared by other users; administrator user list, user creation, user removal, and password reset.

Excluded: OAuth or any external identity provider; self-registration; email; multi-factor authentication; roles other than user and administrator; per-user permissions on shared items beyond what is stated below.

## Definitions

- **Owner**: the user who added an item.
- **Shared item**: an item its owner has marked as shared; other users may see it.
- **Temporary password**: a password set by the administrator, at user creation or reset, that must be changed at the next login.

## Requirements

### Login and session

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AUTH-000 | No regular user has been created yet (only the administrator, or nobody, exists). | The application keeps working as it does today: no login is asked, and all items and settings are usable by anyone who opens it (D-25). The administrator can still log in, through a login entry, to create users. FR-AUTH-001 to FR-AUTH-005 apply only once the first regular user exists. | Given no regular user, when opening the app, then the items page is shown without login, as today. Given the first regular user is created, when opening the app without a session, then the login screen is shown. |
| FR-AUTH-001 | At least one regular user exists and anyone opens the application without being logged in. | Only a login screen is shown, asking for username and password. No item, setting, or user data is accessible without login, through the interface or the API. | Given no login, when opening any page or calling any data API, then the login screen is shown or access is refused. |
| FR-AUTH-002 | A person submits the login form. | Correct credentials log the person in. Wrong username or password show one generic error, without revealing which was wrong. | Given a wrong password, when submitting, then "Incorrect username or password" is shown and the person stays logged out. |
| FR-AUTH-003 | A logged-in person chooses Log out. | The session ends and the login screen is shown. | After logout, reloading the page shows the login screen. |
| FR-AUTH-004 | A session stays unused or reaches its maximum duration. | A session lasts 30 days unless the person logs out; then the person must log in again (D-8). | Given a login 31 days ago, when opening the app, then the login screen is shown. |
| FR-AUTH-005 | Repeated failed logins. | After 5 consecutive failed logins for a username, further attempts for it are refused for 1 minute, with a message saying to wait (D-11). | Given 5 wrong passwords, when trying a 6th time within a minute, even with the right password, then login is refused. |

### First login and password change

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AUTH-010 | A user logs in with a temporary password. | Before anything else, the user must set a new password (entered twice). The application cannot be used until it is changed. | Given a newly created user, when logging in, then only the change-password screen is shown. |
| FR-AUTH-011 | A user sets a new password. | The new password must have at least 12 characters and differ from the temporary one (D-10). | Given an 11-character password, or one equal to the temporary one, then an error is shown and the password is unchanged. |
| FR-AUTH-012 | A logged-in user wants to change their password. | Any user can change their own password at any time from the header menu by giving the current password; the new one follows FR-AUTH-011 (D-12). | Given a wrong current password, then the change is refused. |

### Administrator

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-ADMIN-001 | Installation. | Exactly one global administrator exists. Its username is `admin`. It is created on the device with a command (part of installation) where the operator types its password; it cannot be created from the browser (D-1). The same command can reset the administrator's password in an emergency, for example when it is forgotten (D-9). | Given a fresh installation without the command run, then nobody can log in as administrator. |
| FR-ADMIN-002 | The administrator logs in. | The only screen is the user list showing, per user: username, whether a temporary password must still be changed, creation date, last login date, and number of owned items (D-13). The administrator cannot see or manage items or settings. | Given the administrator, when logged in, then no items view, settings, or item API is accessible. |
| FR-ADMIN-003 | The administrator adds a user. | The administrator enters a unique username and a temporary password. Usernames are 3–32 characters of letters, digits, dot, dash, and underscore, and are case-insensitive: `Jorge` and `jorge` are the same user, at creation and at login (D-17). The new user must change it at first login (FR-AUTH-010). | Given an existing username, when adding, then an error is shown and no user is created. |
| FR-ADMIN-004 | The administrator resets a user's password. | The administrator assigns a new temporary password. The user must change it at next login. The user's open sessions end immediately (D-14). | Given a reset user, when they log in with the new password, then the change-password screen is shown. |
| FR-ADMIN-005 | The administrator removes a user. | After confirmation, the user can no longer log in and their sessions end. All their items are deleted, including shared ones, which disappear for other users (D-2). | Given a removed user, when they try to log in, then login fails and their items are no longer listed for anyone. |
| FR-ADMIN-006 | The administrator renames a user. | The new username follows FR-ADMIN-003 rules and must be unique. The user keeps their items, sharing, settings, and password; they log in with the new name from then on (D-16). | Given user `anna` renamed to `ana`, when logging in as `ana` with the same password, then the same items are shown; `anna` no longer works. |
| FR-ADMIN-007 | The administrator account. | The administrator cannot be removed. The administrator can change their own password in the application by giving the current one (D-9). | Given the administrator, when changing their password, then the next login requires the new one. |

### Item ownership and sharing

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-SHARE-001 | A user adds an item. | The user becomes its owner. The add form lets the user choose whether the item is shared; it is not shared by default (D-7). | Given a new item added without changing the option, then other users do not see it. |
| FR-SHARE-002 | An owner changes an item's sharing. | The owner can switch an item between shared and not shared at any time. Unsharing hides it from other users immediately. | Given a shared item, when unshared, then other users no longer see it. |
| FR-SHARE-003 | A user views the items list. | By default the list shows only the user's own items. The user can turn on an option to also list items shared by other users When on, others' shared items appear in a separate "Shared by others" section below the user's own items, each showing its owner (D-18). The choice is not remembered: each visit starts with only own items (D-15). | Given the option off, then only own items are listed; on, then own items plus others' shared items are listed. |
| FR-SHARE-004 | A user sees an item shared by someone else. | The item shows who owns it. The user can view it (price history, purchase goal, AI reviews) but cannot edit, delete, refresh, change its sharing, or request an AI review (D-3). | Given another user's shared item, when its details open, then no edit, delete, refresh, sharing, or AI review action is offered, and the API refuses them. |
| FR-SHARE-005 | Two users add the same product URL. | Each user gets an independent item with its own price history, purchase goal, AI reviews, and sharing. Adding a URL that the same user already tracks behaves as today (D-6). | Given user A tracks URL U, when user B adds U, then B gets a new item and A's item is unchanged. |
| FR-SHARE-006 | Items and settings that exist before this feature. | The first regular user created by the administrator becomes the owner of all existing items, not shared, and receives the existing Claude token and LeBoncoin session (D-5). | Given existing items, when the administrator creates the first user, then that user sees all of them and the second user sees none. |

### Per-user settings

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-SETTINGS-001 | A user opens Settings. | Each user has their own Claude token and LeBoncoin session; nobody else sees them. An item's automatic refreshes and AI reviews use its owner's settings (D-4). | Given user A's token, when user B opens Settings, then B sees only B's own settings. |

## Decisions

- D-1: the administrator is created by a command on the device, with a password typed there.
- D-2: removing a user deletes all their items, shared ones included.
- D-3: another user's shared item is view-only.
- D-4: Claude token and LeBoncoin session are per user; an item uses its owner's settings.
- D-5: the first regular user created inherits existing items and settings.
- D-6: the same URL tracked by two users gives two independent items.
- D-7: new items are not shared by default.
- D-8: a session lasts 30 days.
- D-9: administrator username is `admin`; it changes its password in the app; the device command resets it in an emergency.
- D-10: passwords have at least 12 characters.
- D-11: 5 failed logins for a username block it for 1 minute.
- D-12: users can change their own password at any time.
- D-13: admin user list shows username, must-change-password, created date, last login, item count.
- D-14: reset or removal ends the user's sessions immediately.
- D-15: the "show other users' items" choice is not remembered.
- D-16: the administrator can rename users; items and password are kept.
- D-17: usernames are case-insensitive, 3–32 characters of letters, digits, `.`, `-`, `_`.
- D-18: others' shared items appear in a separate "Shared by others" section.
- D-25: while no regular user exists, the application works as today, without login (user comment on PR #13, 2026-10-04).

## Delivery plan

The feature is delivered in phases; see [the delivery plan](delivery-plan.md). Its phase 1 decisions D-19–D-24 override some decisions above.
