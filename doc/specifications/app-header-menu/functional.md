# Functional specification: application header menu

Status: ready\
Owner: functional specification agent\
User decision/reference: user request of 2026-10-03 (quoted in the [change index](../../changes/leboncoin-session-settings/index.md)): “on the top right part of the screen, you should show something similar to a profile icon that will open a menu. on this menu, you should show a settings link. when clicking on it, a modal will appear.”

## Purpose and scope

Give the operator a persistent entry point to application settings from the header of every page. The actor is the operator using the shared, no-login web interface.

Included: a profile-style icon control at the right end of the existing header, the menu it opens, and the menu's **Settings** entry, which opens the [settings modal](../leboncoin-session-settings/functional.md).

Excluded: user accounts, login/logout, user names or avatars, per-user preferences, and any menu entry other than **Settings**. The icon resembles a profile icon only visually; it does not imply that the application has users. The existing header brand link and **Add item** button are unchanged.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-APP-MENU-001 | Any application page is displayed (tracked-items list or item details, including loading and error states). | Show a profile-style icon control in the top-right part of the header. | Given the list page or an item-details page, the icon control is visible at the right end of the header, after the existing **Add item** button, on every page. The existing brand link and **Add item** button keep their behavior. |
| FR-APP-MENU-002 | The operator activates the icon control by pointer or keyboard. | Open a menu attached to the control. | When the control is clicked, or focused and Enter/Space is pressed, a menu opens below/near the control. Activating the control again, pressing Escape, or clicking outside the menu closes it without other effect. |
| FR-APP-MENU-003 | The menu is open. | The menu contains exactly one entry, **Settings**. Selecting it closes the menu and opens the settings modal. | The menu lists only “Settings”. Clicking it, or selecting it with the keyboard, closes the menu and displays the settings modal defined by FR-LBC-SET-001. The current page and its data remain unchanged underneath. |
| FR-APP-MENU-004 | The operator uses keyboard or assistive technology. | Follow the Carbon header/menu accessibility patterns. | The control has an accessible name describing that it opens the application menu (it must not claim a signed-in user), exposes its expanded/collapsed state, and shows a visible focus indicator. Menu entries are reachable with the keyboard. Escape closes the menu and returns focus to the control. After the settings modal closes, focus returns to the control. |
| FR-APP-MENU-005 | The viewport is narrow (about 400 px wide). | Keep the control visible and usable. | At about 400 px width, the control remains visible in the header, does not overlap the brand or **Add item** control, and opens a menu that fits on screen. |
| FR-APP-MENU-006 | The settings modal or another modal (add item, delete) is open. | Do not open a second modal on top of an open modal through the menu. | While a modal is open the header control is not reachable behind the modal; the menu cannot be used to stack the settings modal over another modal. |

## States and corner cases

- The control does not depend on any server data, so it is available while the item list is loading, after a load error, and on a missing-item page.
- Repeated opening/closing of the menu has no side effect and makes no server request; the settings modal's own loading is defined by FR-LBC-SET-002.
- Browser navigation (back/forward) while the menu is open closes it or leaves it harmless; no route or URL is created for the menu or the settings modal.

## Open questions

None affecting this handoff. Menu contents are exactly what the user requested (one Settings link); additional entries are out of scope until requested.

## Traceability

- [Existing functional specifications](../../FUNCTIONAL_SPECIFICATIONS.md), section 7 (pages and interface areas) and section 10 (no accounts or login).
- [LeBoncoin session settings](../leboncoin-session-settings/functional.md), FR-LBC-SET-001.
- Technical handoff target: `doc/specifications/app-header-menu/technical.md`.
