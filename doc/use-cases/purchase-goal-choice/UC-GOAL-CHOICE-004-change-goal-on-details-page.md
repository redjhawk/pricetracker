# UC-GOAL-CHOICE-004 — Change an item's purchase goal on its details page

**Primary actor:** Operator (in protected mode, the logged-in user)
**Supporting systems:** PriceFollower application, LeBoncoin item details page, AI review
**Trigger:** The operator decides that one of their LeBoncoin items serves another purpose.
**Origin:** Issue #304: the same goal choice is offered on the details page.
**Covers:** FR-GOAL-CHOICE-004, FR-GOAL-CHOICE-007, FR-GOAL-CHOICE-010, FR-GOAL-CHOICE-012, FR-GOAL-CHOICE-013, FR-GOAL-CHOICE-014, FR-GOAL-CHOICE-016

## Preconditions

- The operator owns LeBoncoin item A with goal "Gaming" and item B with goal "Office".
- A Claude token is saved.

## Main flow

1. The operator opens A's details page.
2. The purchase goal shows the drop-down with "Gaming" selected, the goal text field showing "Gaming", and the save action unavailable (FR-GOAL-CHOICE-010).
3. The operator selects "Office". The field shows "Office" and the save action becomes available.
4. The operator saves. The save result is announced (FR-PURCHASE-GOAL-009).
5. A new AI review of A is pending, using "Office" (FR-PURCHASE-GOAL-004). No review is started for B (FR-GOAL-CHOICE-007).
6. The operator reloads the page: "Office" is selected and shown in the field.

## Alternatives and errors

- **Goal removed:** at step 3 the operator selects "No goal"; the field is hidden. After saving, A has no goal (FR-GOAL-CHOICE-010, -014).
- **New goal:** at step 3 the operator selects "New goal", types "photo editing" and saves; A gets "photo editing", which is then offered as an existing goal (FR-GOAL-CHOICE-008).
- **Back to the current goal:** the operator selects "Office" then "Gaming" again; the goal to save equals the current goal and the save action is unavailable (FR-GOAL-CHOICE-010).
- **Keyboard only:** the operator selects an option, edits the field and saves using only the keyboard (FR-GOAL-CHOICE-004).
- **No Claude token:** the goal is saved; no review is started, as in the existing AI review rules (FR-PURCHASE-GOAL-004).
- **Save fails:** an error is shown; the selection and the field text stay; A's stored goal and reviews are unchanged.
- **Goals loading or not loadable:** while loading, the goal entry shows a loading indicator (FR-GOAL-CHOICE-012); if they cannot be loaded, the page shows A's goal in the goal text field with the save action, as before this feature (FR-GOAL-CHOICE-013).
- **Long goal:** if A's goal is "install a light Linux distro for my parents", the drop-down shows "install a light Linux…" and the field shows the full text (FR-GOAL-CHOICE-016).

## Postconditions

- A has the new goal (or none) and, with a token, a new review based on it; B and every other item keep their goals and reviews.
