# Functional specification: choose an existing purchase goal

Status: ready; FR-GOAL-CHOICE-017–019 implemented (issue #373, maintenance)
Owner: functional specification agent
User decision/reference: GitHub issue #304 (2026-11-24) and the user's answers in the issue comments of 2026-11-24 (Q-1 A, Q-2 copy, Q-3 B, Q-4 B, Q-5 no, Q-6 B, Q-7 no, Q-8 alphabetical, Q-9 always offered; Q-10 "just the first 4 words"; Q-11 "select 'new goal' and write a new one, or select an existing one and modify the text; modifying the text of an existing one does not impact other items"). Extends [LeBoncoin purchase goal](../leboncoin-purchase-goal/functional.md).

## Request

> When adding a purchase goal, the user will be able to select an existing purchase goal. User can add a new AI purchase goal if no option corresponds to the current goal.

## Purpose and scope

Let the operator reuse a purchase goal they already wrote instead of typing it again, while still being able to enter a new goal or adapt an existing one. Actor: the operator (in protected mode, the logged-in user).

Included: a drop-down of existing goals, with a "New goal" option, and an editable goal text field, in the add-item form and on the item details page, for LeBoncoin and Amazon items (the same goal choice in both places, Q-3 B; Amazon items since issue #305, owner's decision of 2026-11-25).

Excluded: a separate saved list of goals; renaming or removing goals from the list (Q-7); goals of items shared by other users (Q-5); any change to how a goal is used in AI reviews (FR-PURCHASE-GOAL-005) or to the review structure.

## Definitions

- **Existing goals:** the distinct non-empty purchase goal texts currently set on the operator's own items, LeBoncoin and Amazon (Q-1 A; Amazon since issue #305). In open mode (FR-MODE-001) these are the goals of all items. Goals of items shared by other users are not included (Q-5).
- **Goal choice:** the "Purchase goal" drop-down and the goal text field described below.
- **Goal text field:** the multi-line free-text purchase goal field of FR-PURCHASE-GOAL-001/-003.
- **Word:** a run of characters separated by spaces, tabs or line breaks.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-GOAL-CHOICE-001 | Operator opens the add-item form and at least one existing goal is available. | The purchase goal is entered with a labelled "Purchase goal (optional)" drop-down offering "No goal", every existing goal, and "New goal" (Q-4 B). "No goal" is selected initially. When an existing goal is selected and the added item is a LeBoncoin item, the goal text is saved as the item's purchase goal, with the same effect as typing it (FR-PURCHASE-GOAL-001, -005). | Given an existing goal "install a light Linux distro", when the operator selects it and adds a LeBoncoin item, then the new item's details page shows that goal. |
| FR-GOAL-CHOICE-002 | Goal choice shown; operator selects "New goal". | The goal text field is shown empty (Q-11), and the operator types a new goal. The typed text is saved as for FR-PURCHASE-GOAL-001; empty or whitespace-only text means no goal (FR-PURCHASE-GOAL-006). While "No goal" is selected, the goal text field is not shown. | Given existing goals that do not fit, when the operator selects "New goal", the field is empty; when they type "photo editing" and add a LeBoncoin item, then its details page shows "photo editing". |
| FR-GOAL-CHOICE-003 | No existing goal is available (add-item form or details page). | No drop-down is shown; the purchase goal is entered in the goal text field exactly as before this feature. | Given no item of the operator has a goal, the add-item form shows an empty optional "Purchase goal" text field and no drop-down. |
| FR-GOAL-CHOICE-004 | Goal choice shown. | The drop-down and the goal text field have visible labels, are keyboard operable, and the selected option is announced to assistive technologies. | Using only the keyboard, the operator can open the options, select one, edit the goal text, and add the item or save the goal. |
| FR-GOAL-CHOICE-005 | Existing goals listed. | Each goal text appears once; texts merge only when exactly identical. Texts differing in letter case or spacing are separate options (Q-6 B). | Given items with goals "Gaming", "Gaming" and "gaming", the drop-down offers "Gaming" once and "gaming" once. |
| FR-GOAL-CHOICE-006 | Existing goals listed. | Existing goals are listed in alphabetical order of their full text, letter case ignored (Q-8). "No goal" is the first option and "New goal" the last. | Given goals "photo editing", "Gaming" and "browsing", the options are, in order: "No goal", "browsing", "Gaming", "photo editing", "New goal". |
| FR-GOAL-CHOICE-007 | Operator selects an existing goal for an item. | The text is copied: the item gets its own goal (Q-2). Later changing or clearing the goal of one item, including by modifying the copied text (Q-11), never changes the goal of another item and starts a review only for the changed item (FR-PURCHASE-GOAL-004). | Given items A and B both with goal "Gaming", when the goal of A is changed to "Office", then B still shows "Gaming" and no new review is started for B. |
| FR-GOAL-CHOICE-008 | The goals of the operator's items change (item added, deleted, or goal changed or cleared). | The list always reflects the current goals: a new goal appears from the next time the goal choice is shown; a goal disappears once no item of the operator has it any more (Q-7). The list offers no rename or remove action. | Given goal "Gaming" set only on item A, when A is deleted (or its goal changed), then reopening the add-item form no longer offers "Gaming". Given a LeBoncoin item added with new goal "Office", reopening the add-item form offers "Office". |
| FR-GOAL-CHOICE-009 | Add-item form with an Amazon URL. | The goal choice is offered as for any URL (Q-9). The chosen or typed goal is saved for the Amazon item, as for a LeBoncoin item (issue #305), and joins the existing goals (FR-GOAL-CHOICE-008). | Given an Amazon URL and the existing goal "Gaming" selected, when the item is added, then the Amazon item's details page shows the goal "Gaming". Given an Amazon item added with new goal "Brand new text", reopening the add-item form offers it. |
| FR-GOAL-CHOICE-010 | Operator opens the details page of their own item (LeBoncoin or Amazon) and at least one existing goal is available. | The purchase goal is shown with the goal choice (Q-3 B): the item's current goal is selected and shown in the goal text field (FR-GOAL-CHOICE-015), or "No goal" is selected when it has none. Selecting "No goal", selecting another existing goal, modifying the goal text, or selecting "New goal" and typing text, then using the save action, changes the item's goal as for FR-PURCHASE-GOAL-003 and FR-PURCHASE-GOAL-004. The save action is unavailable while the goal that would be saved (FR-GOAL-CHOICE-014) equals the item's current goal. | Given item A with goal "Gaming" and item B with goal "Office", when A's details page opens, then "Gaming" is selected, the field shows "Gaming" and the save action is unavailable; when the operator selects "Office" and saves, then reloading shows "Office" selected and, with a Claude token, a new review of A is pending. Selecting "No goal" and saving leaves A with no goal. |
| FR-GOAL-CHOICE-011 | A user views an item shared by someone else (FR-SHARE-004). | The goal is shown read-only, as before this feature; no goal choice and no save action are offered. | Given another user's shared item with a goal, its details page shows the goal and no drop-down or save action. |
| FR-GOAL-CHOICE-012 | The existing goals are being loaded. | The goal entry shows a loading indicator. In the add-item form the item can still be added, without a goal. | While the list loads, the form shows the loading indicator and the add action works. |
| FR-GOAL-CHOICE-013 | The existing goals cannot be loaded. | The goal text field is offered as in FR-GOAL-CHOICE-003, so a goal can still be typed and saved. | Given the list request fails, the add-item form shows the free-text "Purchase goal" field and adding an item with a typed goal works. |
| FR-GOAL-CHOICE-014 | Item added or goal saved while the goal choice is shown. | With "No goal" selected, no goal is saved. Otherwise the goal saved is the text in the goal text field at that moment (trimmed as in FR-PURCHASE-GOAL-007; empty means no goal, FR-PURCHASE-GOAL-006). Selecting an option replaces the field's content: an existing goal puts its full text, "New goal" empties it, "No goal" hides it. Text typed or modified and then abandoned by selecting another option is not saved. | Given the operator types "x" under "New goal", then selects "Gaming" and adds a LeBoncoin item, then the item's goal is "Gaming". Given the operator modifies "Gaming" and then selects "No goal", the item gets no goal. |
| FR-GOAL-CHOICE-015 | Operator selects an existing goal (add-item form or details page). | The goal text field is shown, pre-filled with the goal's full text, and stays editable (Q-11). The operator may modify the text; the drop-down keeps showing the selected goal. The modified text is saved for this item only (FR-GOAL-CHOICE-007, -014). | Given items A and B with goal "Gaming", when the operator opens A's details page, changes the field text to "Gaming on Linux" and saves, then A shows "Gaming on Linux", B still shows "Gaming", and both goals are now offered. |
| FR-GOAL-CHOICE-016 | Existing goals shown in the drop-down (options and selected value). | Each existing goal is shown by its first four words, on one line, followed by "…" when the goal has more than four words (Q-10). Goals of four words or fewer are shown in full. Two goals with the same first four words remain separate options (FR-GOAL-CHOICE-005), in the order of FR-GOAL-CHOICE-006. The full text of the selected goal is visible in the goal text field (FR-GOAL-CHOICE-015). | Given the goal "install a light Linux distro for my parents", the option and the selected value read "install a light Linux…" and the field below shows the full text. Given the goal "photo editing", the option reads "photo editing". |

### Store doc comment (issue #373, maintenance)

The doc comment of `Store.PurchaseGoals` in `internal/store/store.go` says it returns the goals of the owner's "LeBoncoin items", but since issue #305 it returns the goals of the owner's tracked LeBoncoin and Amazon items (the query has no platform filter), as the Definitions above state. The comment must match. No behaviour change.

- **FR-GOAL-CHOICE-017** The doc comment directly above `func (s *Store) PurchaseGoals` in `internal/store/store.go` is the single line `// PurchaseGoals returns the distinct non-empty purchase goals of the owner's tracked items (LeBoncoin and Amazon), unordered.` Check: the line just before the function declaration is exactly that text, the line before it is not a comment line, and `grep -n "owner's LeBoncoin items" internal/store/store.go` prints nothing.
- **FR-GOAL-CHOICE-018** `Store.PurchaseGoals` keeps its signature, its query and its body: it still returns the goals of the owner's tracked LeBoncoin and Amazon items, with no platform filter. Check: `git diff origin/master -- internal/store/store.go` changes only the doc comment line of FR-GOAL-CHOICE-017; FR-GOAL-CHOICE-001 to 016 still hold and the existing purchase goal choice tests pass.
- **FR-GOAL-CHOICE-019** Nothing else changes: the text of FR-GOAL-CHOICE-001 to 016, the issue #304 and #305 change records under `doc/changes/` and the files under `doc/todo/` (including `doc/todo/issue-304-r3-01.md`, which the issue does not ask to change) are unchanged. Check: outside `doc/`, `git diff --name-only origin/master` lists only `internal/store/store.go`; `gofmt -l internal/store` prints nothing; `go vet ./...`, `go test ./...` and the frontend checks pass; the operator observes no change in the application.

## States and corner cases

- Save failure on the details page: as for FR-PURCHASE-GOAL-009, an error is shown and the selection and the field text stay; the stored goal is unchanged.
- "New goal", or a modified existing goal, with text identical to another existing goal: saved as typed; the list then still shows that text once (FR-GOAL-CHOICE-005).
- An existing goal selected and its text cleared in the field: the item gets no goal (FR-GOAL-CHOICE-014).
- After the first goal is saved on a details page that showed only the goal text field (FR-GOAL-CHOICE-003), the page shows the goal choice with that goal selected and shown in the field.

## Open questions

None. Issue #373 needed no question: it gives the exact comment text and changes no behaviour.

## Traceability

- [LeBoncoin purchase goal](../leboncoin-purchase-goal/functional.md) FR-PURCHASE-GOAL-001–009.
- [Multiple users and login](../multi-user-login/functional.md) FR-MODE-001, FR-SHARE-004, FR-SHARE-005.
- [Responsive layout](../responsive-layout/functional.md) FR-RESPONSIVE-010.
- Issue #373 (review R3-01 on PR #327, `doc/todo/issue-304-r3-01.md`): the `Store.PurchaseGoals` doc comment said "LeBoncoin items" though Amazon items are included since issue #305 → FR-GOAL-CHOICE-017 to 019. No behaviour change. Change folder: [issue 373](../../changes/2026-11-27-2251-issue-373-fix-stale-doc-comment-on-store-purchaseg/issue.md).
