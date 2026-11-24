# Functional specification: choose an existing purchase goal

Status: draft, waiting for answers to the [open questions](../../changes/2026-11-24-0833-issue-304-ai-purchase-goal-let-the-user-to-choose/questions.md) (Q-10, Q-11)
Owner: functional specification agent
User decision/reference: GitHub issue #304 (2026-11-24) and the user's answers in the issue comment of 2026-11-24 (Q-1 A, Q-2 copy, Q-3 B, Q-4 B, Q-5 no, Q-6 B, Q-7 no, Q-8 alphabetical, Q-9 always offered). Extends [LeBoncoin purchase goal](../leboncoin-purchase-goal/functional.md).

## Request

> When adding a purchase goal, the user will be able to select an existing purchase goal. User can add a new AI purchase goal if no option corresponds to the current goal.

## Purpose and scope

Let the operator reuse a purchase goal they already wrote instead of typing it again, while still being able to enter a new goal. Actor: the operator (in protected mode, the logged-in user).

Included: a drop-down of existing goals, with a "New goal" option that shows the free-text field, in the add-item form and on the LeBoncoin item details page.

Excluded: a separate saved list of goals; renaming or removing goals from the list (Q-7); goals of items shared by other users (Q-5); Amazon purchase goals; any change to how a goal is used in AI reviews (FR-PURCHASE-GOAL-005) or to the review structure.

## Definitions

- **Existing goals:** the distinct non-empty purchase goal texts currently set on the operator's own LeBoncoin items (Q-1 A). In open mode (FR-MODE-001) these are the goals of all LeBoncoin items. Goals of items shared by other users are not included (Q-5).
- **Goal choice:** the "Purchase goal" drop-down described below.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-GOAL-CHOICE-001 | Operator opens the add-item form and at least one existing goal is available. | The purchase goal is entered with a labelled "Purchase goal (optional)" drop-down offering "No goal", every existing goal, and "New goal" (Q-4 B). "No goal" is selected initially. When an existing goal is selected and the added item is a LeBoncoin item, that text is saved as the item's purchase goal, with the same effect as typing it (FR-PURCHASE-GOAL-001, -005). | Given an existing goal "install a light Linux distro", when the operator selects it and adds a LeBoncoin item, then the new item's details page shows that goal. |
| FR-GOAL-CHOICE-002 | Goal choice shown; operator selects "New goal". | The multi-line free-text purchase goal field is shown. The typed text is saved as for FR-PURCHASE-GOAL-001; empty or whitespace-only text means no goal (FR-PURCHASE-GOAL-006). While "No goal" or an existing goal is selected, the free-text field is not shown. | Given existing goals that do not fit, when the operator selects "New goal", types "photo editing" and adds a LeBoncoin item, then its details page shows "photo editing". |
| FR-GOAL-CHOICE-003 | No existing goal is available (add-item form or details page). | No drop-down is shown; the purchase goal is entered in the free-text field exactly as before this feature. | Given no LeBoncoin item of the operator has a goal, the add-item form shows an empty optional "Purchase goal" text field and no drop-down. |
| FR-GOAL-CHOICE-004 | Goal choice shown. | The drop-down has a visible label, is keyboard operable, and its selected option is announced to assistive technologies. | Using only the keyboard, the operator can open the options, select one, and add the item or save the goal. |
| FR-GOAL-CHOICE-005 | Existing goals listed. | Each goal text appears once; texts merge only when exactly identical. Texts differing in letter case or spacing are separate options (Q-6 B). | Given items with goals "Gaming", "Gaming" and "gaming", the drop-down offers "Gaming" once and "gaming" once. |
| FR-GOAL-CHOICE-006 | Existing goals listed. | Existing goals are listed in alphabetical order, letter case ignored (Q-8). "No goal" is the first option and "New goal" the last. | Given goals "photo editing", "Gaming" and "browsing", the options are, in order: "No goal", "browsing", "Gaming", "photo editing", "New goal". |
| FR-GOAL-CHOICE-007 | Operator selects an existing goal for an item. | The text is copied: the item gets its own goal (Q-2). Later changing or clearing the goal of one item never changes the goal of another item and starts a review only for the changed item (FR-PURCHASE-GOAL-004). | Given items A and B both with goal "Gaming", when the goal of A is changed to "Office", then B still shows "Gaming" and no new review is started for B. |
| FR-GOAL-CHOICE-008 | The goals of the operator's LeBoncoin items change (item added, deleted, or goal changed or cleared). | The list always reflects the current goals: a new goal appears from the next time the goal choice is shown; a goal disappears once no LeBoncoin item of the operator has it any more (Q-7). The list offers no rename or remove action. | Given goal "Gaming" set only on item A, when A is deleted (or its goal changed), then reopening the add-item form no longer offers "Gaming". Given a LeBoncoin item added with new goal "Office", reopening the add-item form offers "Office". |
| FR-GOAL-CHOICE-009 | Add-item form with an Amazon URL. | The goal choice is offered as for any URL (Q-9). The chosen or typed goal is not saved for Amazon items (FR-PURCHASE-GOAL-001 unchanged). | Given an Amazon URL and the existing goal "Gaming" selected, when the item is added, then the Amazon item has no goal and the item is added normally. |
| FR-GOAL-CHOICE-010 | Operator opens the details page of their own LeBoncoin item and at least one existing goal is available. | The purchase goal is shown with the goal choice (Q-3 B): the item's current goal is selected, or "No goal" when it has none. Selecting "No goal", another existing goal, or "New goal" with typed text, then using the save action, changes the item's goal as for FR-PURCHASE-GOAL-003 and FR-PURCHASE-GOAL-004. The save action is unavailable while the selection would leave the goal unchanged. | Given item A with goal "Gaming" and item B with goal "Office", when A's details page opens, then "Gaming" is selected; when the operator selects "Office" and saves, then reloading shows "Office" selected and, with a Claude token, a new review of A is pending. Selecting "No goal" and saving leaves A with no goal. |
| FR-GOAL-CHOICE-011 | A user views a LeBoncoin item shared by someone else (FR-SHARE-004). | The goal is shown read-only, as before this feature; no goal choice and no save action are offered. | Given another user's shared item with a goal, its details page shows the goal and no drop-down or save action. |
| FR-GOAL-CHOICE-012 | The existing goals are being loaded. | The goal entry shows a loading indicator. In the add-item form the item can still be added, without a goal. | While the list loads, the form shows the loading indicator and the add action works. |
| FR-GOAL-CHOICE-013 | The existing goals cannot be loaded. | The free-text field is offered as in FR-GOAL-CHOICE-003, so a goal can still be typed and saved. | Given the list request fails, the add-item form shows the free-text "Purchase goal" field and adding an item with a typed goal works. |
| FR-GOAL-CHOICE-014 | Item added or goal saved with a selection. | The goal saved is exactly the selected option's text, or the typed text for "New goal" (trimmed as in FR-PURCHASE-GOAL-007). Text typed for "New goal" and then abandoned by selecting another option is not saved. | Given the operator types "x" under "New goal", then selects "Gaming" and adds a LeBoncoin item, then the item's goal is "Gaming". |

## States and corner cases

- Save failure on the details page: as for FR-PURCHASE-GOAL-009, an error is shown and the selection (and any typed text) stays; the stored goal is unchanged.
- "New goal" with text identical to an existing goal: saved as typed; the list then still shows that text once (FR-GOAL-CHOICE-005).
- After the first goal is saved on a details page that showed the free-text field (FR-GOAL-CHOICE-003), the page shows the goal choice with that goal selected.

## Open questions

See [questions.md](../../changes/2026-11-24-0833-issue-304-ai-purchase-goal-let-the-user-to-choose/questions.md): Q-10 (how long or multi-line goals are shown in the drop-down and whether the full text of a selected goal is visible) and Q-11 (whether "New goal" on the details page starts empty or pre-filled with the current goal). Requirements depending on them will be added when they are answered.

## Traceability

- [LeBoncoin purchase goal](../leboncoin-purchase-goal/functional.md) FR-PURCHASE-GOAL-001–009.
- [Multiple users and login](../multi-user-login/functional.md) FR-MODE-001, FR-SHARE-004, FR-SHARE-005.
- [Responsive layout](../responsive-layout/functional.md) FR-RESPONSIVE-010.
