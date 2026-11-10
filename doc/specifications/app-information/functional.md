# Functional specification: application information

Status: ready\
Owner: functional specification agent\
User decision/reference: issue #115 and the user decisions recorded in the [change index](../../changes/2026-11-10-0541-issue-115-app-information/index.md) (no email; open mode shows only the version; administrator gets the entry; version is the build-time release tag, or latest tag + `-dev` in development; English texts).

## Purpose and scope

Let any person using the interface see who is logged in and which release is running. Actors: a regular logged-in user, the logged-in administrator, and the operator in open mode (no login, see [multi-user login](../multi-user-login/functional.md)).

Included: an **Information** entry in the top-right [application menu](../app-header-menu/functional.md) and the modal it opens.

Excluded: email or any other account data, changelogs, release notes, update checks, and any change to the existing **Settings** and **Log out** entries.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-APP-INFO-001 | A regular user is logged in and opens the application menu. | The menu shows an **Information** entry in addition to the existing **Settings** and **Log out** entries. | Given a logged-in regular user, when the menu opens, it lists “Information” with “Settings” and “Log out”; existing entries keep their behavior. |
| FR-APP-INFO-002 | The administrator is logged in and opens the application menu. | The menu shows **Information** in addition to **Log out**. | Given the logged-in administrator, the menu lists “Information” and “Log out”; selecting Information opens the modal. |
| FR-APP-INFO-003 | Open mode (no logged-in user); the operator opens the menu. | The menu shows **Information** in addition to its existing entries. | Given open mode, the menu lists “Information” and selecting it opens the modal. |
| FR-APP-INFO-004 | The operator selects **Information** (pointer or keyboard). | The menu closes and a modal titled “Information” opens over the unchanged current page. | The modal is displayed; the page and its data underneath are unchanged; no URL or route changes. |
| FR-APP-INFO-005 | The modal is open and a user is logged in (regular or administrator). | Show the logged-in username, labelled “Username”, and the running version, labelled “Version”. | The modal shows “Username” with the exact logged-in username and “Version” with the version value. No email is shown. |
| FR-APP-INFO-006 | The modal is open in open mode. | Show only the version, labelled “Version”; no username field, placeholder, or email. | In open mode the modal contains “Version” and its value and nothing about a user. |
| FR-APP-INFO-007 | A release build is running. | The version is the release tag `vYYYYMMDDHHMMSS` embedded in the application at build time. | Given a build of release tag `v20261109200945`, the modal shows `v20261109200945`. |
| FR-APP-INFO-008 | A development or untagged local build is running. | The version is the latest release tag followed by `-dev`. | Given a local build whose latest tag is `v20261109200945`, the modal shows `v20261109200945-dev`. |
| FR-APP-INFO-009 | The modal is open and the version is being retrieved. | Show a loading indicator in place of the version value. | While retrieval is pending, a loading indicator appears in the Version area. |
| FR-APP-INFO-010 | The version cannot be retrieved. | Show a clear error message instead of the version: “The version could not be loaded.”; never a blank or stale value. The modal stays closable. | When retrieval fails, the error text is visible in the Version area; reopening the modal retries. |
| FR-APP-INFO-011 | The modal is open. | Close via a Close button, the modal close icon, or Escape; closing has no other effect. Focus then returns to the header menu control. | After closing by any of these means, the modal is gone, nothing else changed, and focus is on the header menu control. |
| FR-APP-INFO-012 | Keyboard or assistive technology use. | The entry is reachable and selectable with the keyboard; the modal follows Carbon modal accessibility (focus moved into the modal, focus kept inside while open, accessible title, labelled values). | The entry can be reached with arrow keys/Tab and selected with Enter/Space; on opening, focus is inside the modal; Tab does not leave the modal while open. |
| FR-APP-INFO-013 | Viewport about 400 px wide. | The entry and modal remain usable. | At about 400 px, the modal fits on screen without horizontal scrolling; long usernames and the version wrap or stay readable without overlap. |

## States and corner cases

- Loading and error states apply to the version only (FR-APP-INFO-009/010); the username comes from the current session.
- Opening and closing repeatedly has no side effect.
- If the session has ended when the modal is opened, the existing multi-user login behavior for an ended session applies; no stale username is shown.
- The modal cannot be stacked over another open modal (as FR-APP-MENU-006).

## Open questions

None. All functional choices are recorded user decisions.

## Traceability

- [Header menu](../app-header-menu/functional.md) FR-APP-MENU-003, FR-APP-MENU-004, FR-APP-MENU-006.
- [Multiple users and login](../multi-user-login/functional.md) FR-AUTH-003, open and protected modes.
- [GitHub release](../github-release/functional.md) for release tags.
- Technical handoff target: `doc/specifications/app-information/technical.md`.
