# UC-GOAL-CHOICE-007 — Add an Amazon item while a goal is chosen

**Primary actor:** Operator (in protected mode, the logged-in user)
**Supporting systems:** PriceFollower application, add-item form
**Trigger:** The operator adds an Amazon listing and selects or types a goal in the add-item form.
**Origin:** Issue #304 comments: the goal entry is always shown, whatever the URL.
**Covers:** FR-GOAL-CHOICE-009

## Preconditions

- The operator's own LeBoncoin items have the goal "Gaming".

## Main flow

1. The operator opens the add-item form and enters an Amazon listing URL.
2. The form offers the goal choice as for any URL.
3. The operator selects "Gaming" and adds the item.
4. The Amazon item is added normally (UC-IM-01, UC-AMZ-HCHK-002) without a goal; its details page shows no purchase goal (FR-PURCHASE-GOAL-008).

## Alternatives and errors

- **New goal typed:** the typed text is not saved and is not offered afterwards.
- **Add fails:** the form shows the error as in UC-IM-01.

## Postconditions

- The Amazon item has no goal; the list of existing goals is unchanged.
