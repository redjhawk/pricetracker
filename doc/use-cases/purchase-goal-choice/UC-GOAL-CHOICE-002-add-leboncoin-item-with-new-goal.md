# UC-GOAL-CHOICE-002 — Add a LeBoncoin item with a new purchase goal when no existing goal fits

**Primary actor:** Operator (in protected mode, the logged-in user)
**Supporting systems:** PriceFollower application, add-item form
**Trigger:** The operator adds a LeBoncoin listing whose purpose matches none of their existing goals.
**Origin:** Issue #304: the user can add a new AI purchase goal if no option corresponds to the current goal.
**Covers:** FR-GOAL-CHOICE-002, FR-GOAL-CHOICE-005, FR-GOAL-CHOICE-008, FR-GOAL-CHOICE-014

## Preconditions

- The operator's own LeBoncoin items have the goals "Gaming" and "install a light Linux distro".

## Main flow

1. The operator opens the add-item form and enters a LeBoncoin listing URL.
2. The "Purchase goal (optional)" drop-down shows "No goal" selected; no goal text field is shown.
3. The operator selects "New goal". An empty goal text field appears (FR-GOAL-CHOICE-002).
4. The operator types "photo editing" and adds the item.
5. The item is added as in UC-IM-01 with the goal "photo editing"; with a saved Claude token its first AI review uses that goal.
6. The operator opens the add-item form again. The drop-down now offers "No goal", "Gaming", "install a light Linux…", "photo editing", "New goal" (FR-GOAL-CHOICE-008).

## Alternatives and errors

- **New text identical to an existing goal:** the operator types "Gaming"; the item gets "Gaming" and the drop-down still offers "Gaming" once (FR-GOAL-CHOICE-005).
- **New text differing only in letter case or spacing:** the operator types "gaming"; it becomes a separate option next to "Gaming" (FR-GOAL-CHOICE-005).
- **Field left empty or whitespace-only:** the item is added with no goal (FR-GOAL-CHOICE-002, FR-PURCHASE-GOAL-006).
- **Operator switches back:** after typing, the operator selects "No goal"; the field is hidden and the item is added with no goal; the typed text is not saved (FR-GOAL-CHOICE-014).
- **Add fails:** the form shows the error as in UC-IM-01; no goal is saved and the list is unchanged.

## Postconditions

- The new LeBoncoin item has the typed goal, which is offered from the next time the goal choice is shown.
