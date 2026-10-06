# Functional specification: LeBoncoin purchase goal

Status: ready
Owner: functional specification agent
User decision/reference: GitHub issue #10 (2026-10-04) and the user's answers in the issue comment, recorded as D-1–D-7 in the [change index](../../changes/2026-10-04-0946-issue-10-leboncoin-purchase-goal/index.md). Extends [LeBoncoin AI review](../leboncoin-ai-review/functional.md).

## Request

> Now you can ask an AI its opinion about an item I put on leboncoin to buy. Goal: add a text field that will be passed to the AI. In this text field the purchase objective of the leboncoin item is indicated. E.g. for a computer, I could say it will be used to install a lightweight distro. In that case, the objective is also added to the query sent to the AI.

## Purpose and scope

Let the operator state, as free text, what a LeBoncoin item is intended for, so that every AI review of that item (FR-LBC-AIR-002) takes the purchase goal into account. Actor: the operator.

Included: optional purchase goal entered in the add-item form and on the item details page; stored per item; included in every review request; editing or clearing the goal starts a new review.

Excluded: Amazon items; the items list; goal history; showing the goal in review history entries; a dedicated goal section or verdict in the review (D-5); length limit (D-6).

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-PURCHASE-GOAL-001 | Operator adds an item with the add-item form. | The form offers an optional multi-line, labelled "Purchase goal" text field. When the added item is a LeBoncoin item, a non-empty goal is saved with the item. The goal is not saved for Amazon items (D-1, D-2). | Given a LeBoncoin URL and goal "install a light Linux distro", when the item is added, then its details page shows that goal. |
| FR-PURCHASE-GOAL-002 | Operator opens a LeBoncoin item details page. | The page shows the item's current purchase goal in an editable labelled text field with a save action; empty when no goal is set (D-1). | Given an item with a goal, when its details page opens, then the field contains the goal; given none, the field is empty. |
| FR-PURCHASE-GOAL-003 | Operator saves a changed goal on the details page (including clearing it). | The new goal replaces the previous one; no previous value is kept (D-3, D-7). Saving an unchanged value has no effect. | Given goal A, when the operator saves goal B, then reloading the page shows B; when the operator clears and saves, reloading shows an empty field. |
| FR-PURCHASE-GOAL-004 | Goal changed per FR-PURCHASE-GOAL-003 and a Claude token is saved. | A new AI review is requested automatically and asynchronously, using the new goal, as for FR-LBC-AIR-001 (D-4). Without a token, behavior follows the existing AI review rules for a missing token. | Given a token, when the goal is changed and saved, then the review section shows a pending review, then a new completed review added to history. |
| FR-PURCHASE-GOAL-005 | Any review request for a LeBoncoin item (on add, price change, manual refresh, goal change). | When the item has a non-empty goal, the goal is added to the information sent to the AI (FR-LBC-AIR-002) as the operator's purchase goal, so that the existing review parts take it into account. The review structure (FR-LBC-AIR-003, -014) is unchanged (D-2, D-5). | Given goal "install a light Linux distro" on a laptop, when a review is requested, then the request includes the goal and the review may reference it in its existing explanations. |
| FR-PURCHASE-GOAL-006 | Goal empty or whitespace-only. | Treated as no goal: stored as empty, and reviews behave exactly as before this feature (D-3). | Given an empty goal, a review request contains no purchase goal. |
| FR-PURCHASE-GOAL-007 | Any goal text. | No maximum length is enforced by the application (D-6). Text is shown and sent as entered (leading/trailing whitespace may be trimmed). | A long multi-paragraph goal can be saved and is shown in full. |
| FR-PURCHASE-GOAL-008 | Amazon item details page; review history. | Amazon details pages show no purchase goal field. Review history entries do not show the goal (D-7). | Amazon details page has no goal field; history entries show no goal. |
| FR-PURCHASE-GOAL-009 | Goal field and save action. | Field has a visible label; save result is announced (success or error); keyboard operable. | The field is reachable and savable by keyboard; a failed save shows an error and keeps the entered text. |

## States and corner cases

- Save failure: an error is shown, the entered text stays in the field, the stored goal and reviews are unchanged.
- Goal changed while a review is pending: the newest saved goal is used for the review started by the change; the result of an earlier pending review is handled by the existing AI review rules.
- Items added before this feature have an empty goal; no review is started until the goal is set or another existing trigger occurs.
- Goal saved while the AI is unavailable: the goal is saved; the review error state follows the existing AI review rules.

## Open questions

None. Q-1–Q-8 resolved by D-1–D-7 (Q-6 derived from the issue text: the goal is only added to the query).

## Traceability

- [LeBoncoin AI review](../leboncoin-ai-review/functional.md) FR-LBC-AIR-001–014.
- Change index: [leboncoin-purchase-goal](../../changes/2026-10-04-0946-issue-10-leboncoin-purchase-goal/index.md).
- Technical specification: to be written in `technical.md` in this folder.
