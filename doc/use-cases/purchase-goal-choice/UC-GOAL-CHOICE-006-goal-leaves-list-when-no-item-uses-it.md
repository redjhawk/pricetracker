# UC-GOAL-CHOICE-006 — A goal leaves the list once no item uses it

**Primary actor:** Operator (in protected mode, the logged-in user)
**Supporting systems:** PriceFollower application, add-item form, LeBoncoin item details page
**Trigger:** The operator deletes the last item that has a goal, or changes or clears that item's goal.
**Origin:** Issue #304 comments: the list cannot be managed directly; a goal is removed with the last item that used it.
**Covers:** FR-GOAL-CHOICE-008

## Preconditions

- The operator owns LeBoncoin item A with goal "Gaming" (no other item of theirs has "Gaming") and item B with goal "Office".

## Main flow

1. The operator deletes A (UC-IM-02).
2. The operator opens the add-item form. The drop-down offers "No goal", "Office", "New goal"; "Gaming" is no longer offered.

## Alternatives and errors

- **Goal changed or cleared instead of deleting:** the operator changes A's goal to "Office" or clears it (UC-GOAL-CHOICE-004); from the next time the goal choice is shown, "Gaming" is no longer offered.
- **Goal still used by another item:** if item C also has "Gaming", deleting A leaves "Gaming" in the list.
- **Last goal removed:** if no goal remains, the add-item form and details pages show only the goal text field (UC-GOAL-CHOICE-003).
- **Managing the list directly:** the drop-down offers no rename or remove action.
- **Deletion fails:** the item and its goal stay; the list is unchanged.

## Postconditions

- The list offers exactly the distinct goals of the operator's current items (LeBoncoin and Amazon).
