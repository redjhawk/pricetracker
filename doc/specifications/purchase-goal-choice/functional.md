# Functional specification: choose an existing purchase goal

Status: draft, waiting for answers to the [open questions](../../changes/2026-11-24-0833-issue-304-ai-purchase-goal-let-the-user-to-choose/questions.md)
Owner: functional specification agent
User decision/reference: GitHub issue #304 (2026-11-24). Extends [LeBoncoin purchase goal](../leboncoin-purchase-goal/functional.md).

## Request

> When adding a purchase goal, the user will be able to select an existing purchase goal. User can add a new AI purchase goal if no option corresponds to the current goal.

## Purpose and scope

Let the operator reuse a purchase goal they already wrote instead of typing it again, while still being able to enter a new goal. Actor: the operator (a user in protected mode).

Included: offering existing purchase goals when a purchase goal is entered; entering a new goal when none fits.

Excluded: Amazon purchase goals; any change to how a goal is used in AI reviews (FR-PURCHASE-GOAL-005) or to the review structure.

Not yet specified, pending the open questions: what counts as an existing goal, copy or link on selection, where the choice is offered, the input form, goals of shared items, merging duplicates, managing the list, option order and display, and behaviour for Amazon URLs.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-GOAL-CHOICE-001 | Operator enters the purchase goal in the add-item form and at least one existing goal is available. | The operator can select an existing goal instead of typing it. The selected goal is saved as the added LeBoncoin item's purchase goal, with the same effect as typing the same text (FR-PURCHASE-GOAL-001, -005). | Given an existing goal "install a light Linux distro", when the operator selects it and adds a LeBoncoin item, then the new item's details page shows that goal. |
| FR-GOAL-CHOICE-002 | Operator enters the purchase goal in the add-item form and no existing goal fits. | The operator can type a new goal as before this feature; it is saved as for FR-PURCHASE-GOAL-001. | Given existing goals that do not fit, when the operator types "photo editing" and adds a LeBoncoin item, then its details page shows "photo editing". |
| FR-GOAL-CHOICE-003 | No existing goal is available. | The purchase goal entry behaves as before this feature (free text, optional). | Given no item with a goal, the add-item form offers an empty optional purchase goal entry and no options. |
| FR-GOAL-CHOICE-004 | Goal entry with options. | The choice is labelled, keyboard operable, and the selected option is announced to assistive technologies. | Using only the keyboard, the operator can open the options, select one, and add the item. |

## Open questions

See [questions.md](../../changes/2026-11-24-0833-issue-304-ai-purchase-goal-let-the-user-to-choose/questions.md) (Q-1–Q-9). Requirements depending on them will be added when they are answered.

## Traceability

- [LeBoncoin purchase goal](../leboncoin-purchase-goal/functional.md) FR-PURCHASE-GOAL-001–009.
- [Multiple users and login](../multi-user-login/functional.md) FR-SHARE-004, FR-SHARE-005.
