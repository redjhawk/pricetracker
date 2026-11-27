# UC-GOAL-CHOICE-005 — Adapt a goal used by several items for one item only

**Primary actor:** Operator (in protected mode, the logged-in user)
**Supporting systems:** PriceFollower application, LeBoncoin item details page, AI review
**Trigger:** The operator wants to refine the goal of one item without changing the other items that have the same goal.
**Origin:** Issue #304 comments: selecting a goal copies it; modifying the text of an existing goal does not affect other items' goals.
**Covers:** FR-GOAL-CHOICE-005, FR-GOAL-CHOICE-007, FR-GOAL-CHOICE-010, FR-GOAL-CHOICE-014, FR-GOAL-CHOICE-015

## Preconditions

- The operator owns LeBoncoin items A and B, both with goal "Gaming".
- A Claude token is saved.

## Main flow

1. The operator opens A's details page. "Gaming" is selected and the goal text field shows "Gaming".
2. The operator changes the field text to "Gaming on Linux". The drop-down still shows "Gaming"; the save action becomes available (FR-GOAL-CHOICE-015, -010).
3. The operator saves.
4. A shows "Gaming on Linux" and a new AI review of A is pending (FR-PURCHASE-GOAL-004).
5. The operator opens B's details page: B still shows "Gaming" and no new review was started for B (FR-GOAL-CHOICE-007).
6. The drop-down offers both "Gaming" and "Gaming on Linux".

## Alternatives and errors

- **Text cleared:** the operator empties the field and saves; A gets no goal and B keeps "Gaming" (FR-GOAL-CHOICE-014).
- **Edit abandoned:** after modifying the text, the operator selects another option; the modified text is discarded and replaced by that option's content (FR-GOAL-CHOICE-014).
- **Text made identical to another existing goal:** the operator changes "Gaming" to "Office", an existing goal; A gets "Office" and the drop-down still offers "Office" once (FR-GOAL-CHOICE-005).
- **Save fails:** an error is shown, the modified text stays, and A's stored goal is unchanged.
- **Same flow in the add-item form:** see UC-GOAL-CHOICE-001 "Text adapted before adding".

## Postconditions

- A has the adapted goal; B and every other item keep their goals; only A is re-reviewed.
