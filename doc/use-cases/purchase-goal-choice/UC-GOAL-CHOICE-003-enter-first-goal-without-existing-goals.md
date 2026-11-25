# UC-GOAL-CHOICE-003 — Enter a purchase goal when no existing goal is available

**Primary actor:** Operator (in protected mode, the logged-in user)
**Supporting systems:** PriceFollower application, add-item form, LeBoncoin item details page
**Trigger:** The operator enters a goal while none of their items has a goal (for example, their first goal).
**Origin:** Issue #304.
**Covers:** FR-GOAL-CHOICE-003, FR-GOAL-CHOICE-008

## Preconditions

- No item of the operator has a goal. Items shared by other users may have goals; they do not count.

## Main flow

1. The operator opens the add-item form and enters a LeBoncoin listing URL.
2. The form shows the empty optional multi-line "Purchase goal" text field and no drop-down, as before this feature (FR-GOAL-CHOICE-003).
3. The operator types "Gaming" and adds the item; the item is added with that goal (FR-PURCHASE-GOAL-001).
4. The operator opens the add-item form again. It now shows the "Purchase goal (optional)" drop-down offering "No goal", "Gaming", "New goal" (FR-GOAL-CHOICE-008).

## Alternatives and errors

- **Goal entered on the details page:** the operator opens an existing LeBoncoin item without a goal; the page shows only the goal text field and the save action (FR-GOAL-CHOICE-003, FR-PURCHASE-GOAL-002). The operator types "Gaming" and saves; the page then shows the drop-down with "Gaming" selected and shown in the field (FR-GOAL-CHOICE-010). With a Claude token, a new review is pending (FR-PURCHASE-GOAL-004).
- **No goal typed:** the item is added without a goal and the next form still shows only the text field.
- **Save failure on the details page:** an error is shown, the text stays in the field and no goal is stored (FR-PURCHASE-GOAL-009).

## Postconditions

- The typed goal is stored on the item and offered as an existing goal from then on.
