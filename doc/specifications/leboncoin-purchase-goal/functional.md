# Functional specification: LeBoncoin purchase goal

Status: needs-clarification
Owner: functional specification agent
User decision/reference: GitHub issue #10 (2026-10-04); extends [LeBoncoin AI review](../leboncoin-ai-review/functional.md).

## Request

> Now you can ask an AI its opinion about an item I put on leboncoin to buy. Goal: add a text field that will be passed to the AI. In this text field the purchase objective of the leboncoin item is indicated. E.g. for a computer, I could say it will be used to install a lightweight distro. In that case, the objective is also added to the query sent to the AI.

## Purpose and scope

Let the operator state, as free text, what a LeBoncoin item is intended to be used for, so that the AI review (FR-LBC-AIR-002) takes this purchase goal into account. Actor: the operator.

Included (requested): a free-text purchase goal field for LeBoncoin items; the goal, when present, is included in the information sent to the AI for a review.

Excluded (unless the user decides otherwise): Amazon items; the items list.

## Requirements (provisional, pending answers)

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-LBC-GOAL-001 | LeBoncoin item. | A text field lets the operator enter the item's purchase goal. Location, timing and editability depend on Q-1, Q-2, Q-4. | The operator can enter a purchase goal for a LeBoncoin item. |
| FR-LBC-GOAL-002 | A review is requested for an item with a purchase goal. | The goal is added to the information sent to the AI, in addition to FR-LBC-AIR-002. How the review reflects it depends on Q-6. | A review requested after entering a goal is produced with the goal included in the request. |
| FR-LBC-GOAL-003 | Amazon item. | No purchase goal field. | Amazon details pages show no purchase goal field. |

## Open questions

- Q-1 Where is the purchase goal entered: (a) in the add-item form when adding a LeBoncoin item, (b) on the item details page, or (c) both?
- Q-2 Is the goal saved per item and reused for every later review (automatic on price change and manual refresh), or is it entered only for a single review request?
- Q-3 Is the goal optional? If empty, does the review behave exactly as today?
- Q-4 Can the goal be edited or cleared after it is first set?
- Q-5 When the goal is saved or changed, should a new AI review start automatically, or only at the next refresh/price change?
- Q-6 Should the review contain an explicit, labelled section on suitability for the stated goal (e.g. a verdict such as Suitable / Partially suitable / Not suitable with explanation), or should the goal only influence the existing parts (price, condition, recommendation)?
- Q-7 Should each review in the history show the goal it was based on?
- Q-8 Is there a maximum length for the goal text? If yes, how many characters?

## Traceability

- [LeBoncoin AI review](../leboncoin-ai-review/functional.md) FR-LBC-AIR-001–014.
- Change index: [leboncoin-purchase-goal](../../changes/leboncoin-purchase-goal/index.md).
