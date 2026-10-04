# LeBoncoin purchase goal

Stage: development
State: functional and technical specifications ready; API contract defined; awaiting implementation.

User request (GitHub issue #10, translated from Spanish):

> Now you can ask an AI its opinion about an item I put on leboncoin to buy. Goal of this new implementation: add a text field that will be passed to the AI. In this text field the purchase objective of the leboncoin item is indicated. E.g. for a computer, I could say it will be used to install a lightweight distro. In that case, the objective is also added to the query sent to the AI.

## Subjects

1. LeBoncoin purchase goal — [functional](../../specifications/leboncoin-purchase-goal/functional.md) FR-PURCHASE-GOAL-001–009; [technical](../../specifications/leboncoin-purchase-goal/technical.md) TS-PURCHASE-GOAL-001–009; ready.

## Stage results

1. Functional specification — ready (open questions Q-1–Q-8 resolved by user answers on issue #10).
2. Technical specification — ready: `purchase_goal` column (`TEXT NOT NULL DEFAULT ''`) via the existing `ensureItemColumn` migration; optional `purchaseGoal` on add; goal read at review time and added to the Claude prompt as a delimited, JSON-encoded block; Carbon `TextArea` in the add modal and a new `PurchaseGoal` component on LeBoncoin details pages. No refactoring (decode-helper extraction declined as unnecessary).
3. API contract — defined in `API_SPECIFICATION.md` "LeBoncoin purchase goal (2026-10-04)": additive `purchaseGoal` item field, optional `purchaseGoal` on `POST /api/v1/items`, new `PUT /api/v1/items/{id}/purchase-goal` returning `{ purchaseGoal, changed, reviewStarted }`. Rationale: a dedicated sub-resource matches the existing `/refresh` and `/ai-review` routing, keeps the change small, and lets the server start the review on change (D-4). A rerun flag ensures the newest goal is reviewed when a review is already running. Add/update bodies are capped at 1 MiB as a transport safeguard, not a goal length limit (D-6). Existing contracts unchanged.
4–8. Not started.

## User decisions

From the user's comment on issue #10:

- D-1 (Q-1): The goal is entered both in the add-item form and on the item details page.
- D-2 (Q-2): The goal is saved per item and reused for all reviews, automatic and manual.
- D-3 (Q-3): The goal is optional; empty means the review works as today.
- D-4 (Q-4/Q-5): The goal can be modified later (clearing counts as a modification); a change automatically launches a new AI review.
- D-5 (Q-6, derived from the issue text "the objective is also added to the query"): the goal is only included in the AI query; no new dedicated review section.
- D-6 (Q-8): No length limit.
- D-7 (Q-7): No goal history; review history entries do not show the goal.

## Unresolved questions

None.
