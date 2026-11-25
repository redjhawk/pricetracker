# Functional specification: Amazon item AI review and purchase goal

Status: draft — open questions (issue #305)
Owner: functional specification agent
User decision/reference: GitHub issue #305 "Purchase goal and AI review is also available on Amazon items" (2026-11-24) and the answers of redjhawk (2026-11-24) to the first questions, recorded below as D-1–D-5. Remaining questions in the [change folder](../../changes/2026-11-24-0836-issue-305-purchase-goal-and-ai-review-is-also-avai/questions.md).

## Request

> Currently, AI review is only available for Leboncoin items. This option will be available for Amazon items too.

## Purpose and scope

Make the AI (Claude) review and the purchase goal, which exist for LeBoncoin items ([LeBoncoin AI review](../leboncoin-ai-review/functional.md), [LeBoncoin purchase goal](../leboncoin-purchase-goal/functional.md)), available for tracked Amazon items. Actor: the operator (in protected mode, the item's owner). Reviews use the owner's Claude token ([Claude token settings](../claude-token-settings/functional.md)).

Included: purchase goal on the add-item form and on the Amazon item details page; an **AI review** section with history and a **Refresh AI review** button on the Amazon item details page; review content and the information sent to the AI; pending, empty, no-token and error states; deletion with the item.

Excluded: automatic reviews of any kind (D-2); showing reviews in the items list (as for LeBoncoin); notifications; Amazon searches (removed by issue #111); changes to LeBoncoin reviews and goals.

Not yet specified (pending the [open questions](#open-questions)): whether a review reads the Amazon product page again or uses the latest check only, what **Refresh AI review** does while the item is stopped, and the review of an item without a current price.

## User decisions

- D-1 (Q-1 b): an Amazon review contains a price rating with explanation, a fair price range, and an overall recommendation Buy / Wait for a lower price / Avoid with explanation. No condition, scam, seller-question or photo-match parts.
- D-2 (Q-2 f): no automatic review; reviews are produced only by **Refresh AI review**. The price the last review was based on is shown.
- D-3 (Q-3 a): Amazon items added before this feature are not reviewed automatically; **Refresh AI review** produces their first review.
- D-4 (Q-4 a, d, e): the AI receives the title, marketplace, current price and recorded price history; the Amazon-sold second-hand offer (price, condition) and its history; the customer rating and number of ratings. Where this information comes from (Q-4 f or g) was not chosen: see Q-6.
- D-5 (Q-5 a "no"): a failed review does not count toward the 5 consecutive Amazon failures. The answer to Q-5 b ("ai refresh is not required. inform the user") is confirmed in Q-7.

## Requirements

### Purchase goal

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMZ-ITEM-AIR-001 | Operator adds an Amazon item with the add-item form. | The optional **Purchase goal** field of the form (FR-PURCHASE-GOAL-001) is also saved for Amazon items when non-empty. Adding the item starts no review (D-2). | Given an Amazon URL and goal "gift for a 10-year-old", when the item is added, then its details page shows that goal and "no review yet". |
| FR-AMZ-ITEM-AIR-002 | Operator opens an Amazon item details page. | The page shows the item's purchase goal in an editable, labelled multi-line field with a save action, as on LeBoncoin details pages (FR-PURCHASE-GOAL-002); empty when no goal is set. | Given an Amazon item with a goal, its details page field contains the goal; given none, the field is empty. |
| FR-AMZ-ITEM-AIR-003 | Operator saves a changed goal on an Amazon details page, including clearing it. | The new goal replaces the previous one; no previous value is kept; saving an unchanged value has no effect (as FR-PURCHASE-GOAL-003). Saving starts no review; the next **Refresh AI review** uses the new goal (D-2). | Given goal A, after saving goal B a reload shows B and no new review; after clearing and saving, a reload shows an empty field. |
| FR-AMZ-ITEM-AIR-004 | Any Amazon goal text. | Empty or whitespace-only means no goal and is stored as empty; no maximum length is enforced; text is shown as entered, leading/trailing whitespace may be trimmed (as FR-PURCHASE-GOAL-006, -007). | A long multi-paragraph goal is saved and shown in full; a whitespace-only goal shows an empty field after reload. |
| FR-AMZ-ITEM-AIR-005 | Any review request for an Amazon item with a non-empty goal. | The goal is sent to the AI as the operator's purchase goal, so that the review takes it into account; with no goal, no goal is sent. | Given goal "gift for a 10-year-old", the review request contains that goal; given no goal, it contains none. |
| FR-AMZ-ITEM-AIR-006 | Goal field and save action. | Visible label; save success or error is announced; keyboard operable; a failed save shows an error, keeps the entered text and leaves the stored goal and reviews unchanged (as FR-PURCHASE-GOAL-009). | The goal is savable by keyboard; a failed save keeps the text and shows an error. |
| FR-AMZ-ITEM-AIR-007 | Amazon items added before this feature. | They have an empty goal. | Such an item's details page shows an empty goal field. |

### AI review section

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMZ-ITEM-AIR-010 | Amazon item details page. | Shows an **AI review** section with the latest successful review, the date/time it was produced and the Amazon price it was based on, or a "no review yet" state when none exists. | Given a reviewed Amazon item, the section shows the review, its timestamp and reviewed price; given none, it shows that no review exists yet. |
| FR-AMZ-ITEM-AIR-011 | Several successful reviews exist. | All successful reviews are kept; the section lets the operator view previous reviews, newest first, each with its date/time and reviewed price; history entries do not show the goal (as FR-LBC-AIR-011, FR-PURCHASE-GOAL-008). | After two reviews, both are viewable, newest first. |
| FR-AMZ-ITEM-AIR-012 | Amazon details page, Claude token saved, no review running for this item. | A **Refresh AI review** button requests a new review; it is independent of **Refresh price** and is the only way to get a review (D-2). Progress is shown, then the new review and its timestamp. Its behaviour on a stopped item is pending (Q-7). | Clicking shows progress, then a new review at the top of the history. |
| FR-AMZ-ITEM-AIR-013 | A review for the item is running. | No duplicate review is started; the button is disabled and progress is shown. | Repeated clicks during a run produce at most one running review. |
| FR-AMZ-ITEM-AIR-014 | A review attempt fails (token rejected, usage limit, network, AI error). | The section shows an understandable error with its date/time and keeps showing the latest successful review with its own date; the button stays available to retry; no token appears in errors (FR-CLT-SET-008). A rejected token updates the Settings warning (FR-CLT-SET-009). | Given a previous review and a failed refresh, the old review stays visible with an error notice. |
| FR-AMZ-ITEM-AIR-015 | The owner has no Claude token. | No review is attempted; the section says "Configure a Claude token in Settings" and the button is disabled. Saving a token later does not review Amazon items automatically. Prices are still tracked. | Without a token, an Amazon item has prices and no review, the message is shown and the button is disabled. |
| FR-AMZ-ITEM-AIR-016 | An Amazon item is deleted. | Its reviews and goal are deleted with it (FR-09). | No review of the item remains after deletion. |
| FR-AMZ-ITEM-AIR-017 | Keyboard, assistive technology, phone width (FR-RESPONSIVE-*). | The section has a heading; pending and error states are announced; ratings are conveyed in text, not colour alone; history is reachable by keyboard; readable without horizontal scrolling (as FR-LBC-AIR-013). | A screen reader reaches the "AI review" heading and hears state changes. |
| FR-AMZ-ITEM-AIR-018 | Any review. | Written in English (as FR-LBC-AIR-003). | Every Amazon review is in English. |
| FR-AMZ-ITEM-AIR-019 | LeBoncoin items. | LeBoncoin reviews, goals and their triggers are unchanged. | LeBoncoin acceptance criteria of FR-LBC-AIR-* and FR-PURCHASE-GOAL-* still pass. |
| FR-AMZ-ITEM-AIR-020 | A review completes. | Contains (D-1): **Price** rating — Good deal / Fair / Overpriced — with an explanation based on Claude's general market knowledge and the information of FR-AMZ-ITEM-AIR-021; an estimated **fair price range**; **Overall recommendation** — Buy / Wait for a lower price / Avoid — with an explanation. It has no condition rating, suggested offer, scam/risk signs, missing accessories, questions for the seller or description/photo match. | The section shows, labelled, the Price rating and its explanation, the fair price range, and the overall recommendation and its explanation; none of the excluded parts is shown. |
| FR-AMZ-ITEM-AIR-021 | A review is requested. | The AI receives (D-4): the title, the marketplace, the current Amazon price and the item's recorded price history; the Amazon-sold second-hand offer (price and condition) and its recorded history; the customer rating and number of ratings; and the goal (FR-AMZ-ITEM-AIR-005). An absent offer or rating is sent as absent. Where this information is read is pending (Q-6). | A review can reference the price history, the second-hand offer and the customer rating; for a product without a second-hand offer, the request states that there is none. |
| FR-AMZ-ITEM-AIR-022 | Any event other than **Refresh AI review**: item added, price check (scheduled, **Refresh price** or **Refresh all prices**) recording a different price or second-hand offer, goal saved, token saved, item restarted after a stop. | No review is started (D-2). The shown review and its history are unchanged. | After a check records a new price, the section shows the same review as before and no review is pending. |
| FR-AMZ-ITEM-AIR-023 | A review is shown (latest or history entry). | The Amazon price the review was based on is shown with it, labelled (for example "Based on price 129,99 €"), also when the current price differs (D-2). | Given a review made at 100,00 € and a current price of 80,00 €, the section shows the review labelled as based on 100,00 €. |
| FR-AMZ-ITEM-AIR-024 | Amazon items added before this feature. | No automatic review; the section shows that no review exists yet and **Refresh AI review** produces one (D-3). | An existing Amazon item shows "no review yet"; clicking refresh produces a review. |
| FR-AMZ-ITEM-AIR-025 | A review attempt fails for any reason. | The failure is a review failure (FR-AMZ-ITEM-AIR-014), not an Amazon check failure: it does not count toward the 5 consecutive Amazon failures (FR-AMZ-STOP-002) and stops no item (D-5). | 4 failed Amazon checks followed by a failed review leave every Amazon item running. |

## States and corner cases

- No review yet; pending; completed; failed with or without a previous review; no token; several reviews in history; review made at a price different from the current one (FR-AMZ-ITEM-AIR-023).
- No second-hand offer sold by Amazon, or no customer rating: the review is made without them (FR-AMZ-ITEM-AIR-021).
- Token replaced or removed during a running review: that review finishes or fails with the token it started with.
- Save of a goal fails: the stored goal is unchanged.

## Amended requirements

Once this subject is implemented:

- FR-PURCHASE-GOAL-001: the goal is also saved for Amazon items (FR-AMZ-ITEM-AIR-001).
- FR-PURCHASE-GOAL-008: Amazon details pages show the goal field (FR-AMZ-ITEM-AIR-002); history entries still show no goal.
- FR-LBC-AIR-004: Amazon details pages show their own AI review section (FR-AMZ-ITEM-AIR-010).
- FR-AMZ-SRCH-DEL-005: Amazon items are reviewed again as specified here; reviews deleted by issue #111 are not restored.
- FR-CLT-SET-013 is superseded by FR-CLT-SET-014 (help text mentions Amazon and LeBoncoin reviews).
- Depending on Q-6, Amazon checks may also have to record the customer rating and number of ratings, which they do not record today (FR-AMZ-HCHK-002, FR-AMZ-BRET-*).

## Open questions

Q-1–Q-5 are answered (D-1–D-5). Open in [questions.md](../../changes/2026-11-24-0836-issue-305-purchase-goal-and-ai-review-is-also-avai/questions.md): Q-6 where the information sent to the AI is read, Q-7 **Refresh AI review** on a stopped item, Q-8 review of an item without a current price.

## Traceability

- [LeBoncoin AI review](../leboncoin-ai-review/functional.md), [LeBoncoin purchase goal](../leboncoin-purchase-goal/functional.md), [Claude token settings](../claude-token-settings/functional.md), [Amazon request stop](../amazon-request-stop/functional.md), [Amazon human-like checks](../amazon-human-checks/functional.md), [Amazon browser retrieval](../amazon-browser-retrieval/functional.md).
- History: [Amazon AI review](../amazon-ai-review/functional.md) of search items (issue #56, superseded by issue #111).
- Technical handoff target: `doc/specifications/amazon-item-ai-review/technical.md`.
