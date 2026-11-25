# UC-GOAL-CHOICE-001 — Add a LeBoncoin item with an existing purchase goal

**Primary actor:** Operator (in protected mode, the logged-in user)
**Supporting systems:** PriceFollower application, add-item form
**Trigger:** The operator adds a LeBoncoin listing whose purpose matches a goal they already wrote for another item.
**Origin:** Issue #304: let the user choose from existing AI purchase goals.
**Covers:** FR-GOAL-CHOICE-001, FR-GOAL-CHOICE-004, FR-GOAL-CHOICE-005, FR-GOAL-CHOICE-006, FR-GOAL-CHOICE-007, FR-GOAL-CHOICE-012, FR-GOAL-CHOICE-013, FR-GOAL-CHOICE-014, FR-GOAL-CHOICE-015, FR-GOAL-CHOICE-016

## Preconditions

- The operator's own LeBoncoin items have the goals "photo editing", "Gaming", "gaming", "Gaming" and "install a light Linux distro for my parents".
- A user shares with the operator a LeBoncoin item with the goal "Office".

## Main flow

1. The operator opens the add-item form and enters a LeBoncoin listing URL.
2. While the existing goals load, the goal entry shows a loading indicator (FR-GOAL-CHOICE-012).
3. The form shows a labelled "Purchase goal (optional)" drop-down with "No goal" selected and no goal text field (FR-GOAL-CHOICE-001, -002).
4. The operator opens the drop-down. The options are, in order: "No goal", "Gaming" and "gaming" (as two separate options), "install a light Linux…", "photo editing", "New goal". Each identical text appears once; the shared item's "Office" is not offered (FR-GOAL-CHOICE-005, -006, -016).
5. The operator selects "install a light Linux…". The drop-down shows "install a light Linux…" and an editable goal text field below it shows "install a light Linux distro for my parents" (FR-GOAL-CHOICE-015, -016).
6. The operator adds the item.
7. The item is added as in UC-IM-01 with the goal "install a light Linux distro for my parents"; with a saved Claude token its first AI review uses that goal (FR-GOAL-CHOICE-001, FR-PURCHASE-GOAL-005).
8. The new item's details page shows that goal.

## Alternatives and errors

- **Keyboard only:** the operator reaches the drop-down with Tab, opens it, selects an option with the arrow keys and Enter, edits the field and adds the item without a mouse; the selected option is announced (FR-GOAL-CHOICE-004).
- **Text adapted before adding:** at step 5 the operator changes the field to "install a light Linux distro for my sister"; the new item gets that text, and the item that had "…for my parents" keeps it (FR-GOAL-CHOICE-007, -015). Both texts are then offered (FR-GOAL-CHOICE-008).
- **Text cleared before adding:** the field is emptied or left with only spaces; the item gets no goal (FR-GOAL-CHOICE-014, FR-PURCHASE-GOAL-006).
- **Option changed after editing:** the operator types "x" under "New goal", then selects "Gaming"; the item gets "Gaming". The operator modifies "Gaming" then selects "No goal"; the item gets no goal (FR-GOAL-CHOICE-014).
- **"No goal" kept:** the item is added without a goal.
- **Added while goals are loading:** at step 2 the operator adds the item; it is added without a goal (FR-GOAL-CHOICE-012).
- **Goals cannot be loaded:** the form shows the free-text "Purchase goal" field instead of the drop-down; a typed goal is saved as before (FR-GOAL-CHOICE-013, FR-PURCHASE-GOAL-001).
- **Open mode:** without login, the existing goals are those of all items (FR-MODE-001).
- **Add fails (invalid, unsupported or duplicate URL):** the form shows the error as in UC-IM-01; no goal is saved.

## Postconditions

- The new LeBoncoin item has its own copy of the chosen (or adapted) goal text; no other item's goal changed.
- No goal list was created, renamed or edited.
