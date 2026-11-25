# Test cases: purchase goal and AI review for Amazon items (issue #305)

Sources: functional specification [amazon-item-ai-review](../../specifications/amazon-item-ai-review/functional.md) (FR-AMZ-ITEM-AIR-001 to -034), the amended rows FR-PURCHASE-GOAL-001, -008 ([LeBoncoin purchase goal](../../specifications/leboncoin-purchase-goal/functional.md)), FR-LBC-AIR-004 ([LeBoncoin AI review](../../specifications/leboncoin-ai-review/functional.md)), FR-AMZ-SRCH-DEL-005 ([Amazon searches removal](../../specifications/amazon-searches-removal/functional.md)), FR-CLT-SET-013, -014 ([Claude token settings](../../specifications/claude-token-settings/functional.md)), FR-AMZ-HCHK-002 ([Amazon human-like checks](../../specifications/amazon-human-checks/functional.md)), and use cases [UC-AMZ-ITEM-AIR-001 to -011](../../use-cases/amazon-item-ai-review/).

Conventions:

- **Amazon item A:** ASIN `B012345678` on `amazon.fr`, title "Synthetic headphones", price history 129,99 € then 119,99 €, Amazon-sold second-hand offer 95,00 € "Used - Very good" with one earlier offer 99,00 €, customer rating 4.5 with 1234 ratings, goal "gift for a 10-year-old" unless stated.
- **Current price / stopped:** "no current price" is the specification's definition (no price recorded, or latest answer "price not found" or "listing unavailable"); a retrieval error after a price keeps the current price. "Stopped" is `AmazonStoppedAt` set (FR-AMZ-STOP-003).
- **API shapes:** the request, response and error shapes (review state of an Amazon item, the reason the review is unavailable, the refusal codes for stopped / no current price, the goal save endpoint) are those defined in `API_SPECIFICATION.md` by the technical specification. Messages are the exact texts of the specification: "Configure a Claude token in Settings.", "Amazon checks are stopped: restart the item with Refresh price to request an AI review.", "No current price: an AI review needs a current Amazon price."
- **Go unit tests** use the Amazon page fixtures of `internal/amazon/testdata` (new fixtures add a customer rating) and the review prompt/response code without network.
- **Go integration tests** (`internal/httpapi`, `internal/service`) use a real SQLite database with the normal migrations (`newSettingsServer`, `insertItem`/`insertListing`), the fake Claude transport (`withClaudeStatus`) or `fakeClaude`, which records and counts requests, and the fake Amazon browser of `internal/amazon` (or `amazontest`), which counts every Amazon navigation. "No Amazon request" means that counter is unchanged. Server logs are captured with `log.SetOutput`.
- **E2E tests** (Playwright) go in a new `tests/amazon-item-ai-review.spec.ts`; they mock the API with `page.route`/`context.route` as the existing specs do and record request bodies.
- **Existing tests contradicted by this change** are rewritten by the cases noted: `TestItemDetailsAIReviewField` and `TestRequestAIReviewStatuses` (Amazon `aiReview: null`, `AI_REVIEW_UNSUPPORTED`) → TC-005, TC-011; the Amazon part of `internal/service/review_test.go` (`TestRequestAIReviewErrorsAndUnretrievedListing`) → TC-007; the Amazon "Ignored" goal in `internal/service/purchase_goal_test.go` → TC-005; e2e "Amazon items have no AI review section" and "Amazon details have no purchase goal field" → TC-019; the help text in `tests/leboncoin-session-settings.spec.ts` → TC-027.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-HCHK-002 | TC-001, TC-013 |
| FR-AMZ-ITEM-AIR-001 | TC-005, TC-019 |
| FR-AMZ-ITEM-AIR-002 | TC-006, TC-019, TC-020 |
| FR-AMZ-ITEM-AIR-003 | TC-006, TC-020 |
| FR-AMZ-ITEM-AIR-004 | TC-005, TC-006 |
| FR-AMZ-ITEM-AIR-005 | TC-002, TC-007 |
| FR-AMZ-ITEM-AIR-006 | TC-020 |
| FR-AMZ-ITEM-AIR-007 | TC-015, TC-020 |
| FR-AMZ-ITEM-AIR-010 | TC-005, TC-007, TC-019, TC-021 |
| FR-AMZ-ITEM-AIR-011 | TC-007, TC-014, TC-021 |
| FR-AMZ-ITEM-AIR-012 | TC-007, TC-021 |
| FR-AMZ-ITEM-AIR-013 | TC-008, TC-021 |
| FR-AMZ-ITEM-AIR-014 | TC-003, TC-009, TC-018, TC-023 |
| FR-AMZ-ITEM-AIR-015 | TC-010, TC-024 |
| FR-AMZ-ITEM-AIR-016 | TC-016 |
| FR-AMZ-ITEM-AIR-017 | TC-021, TC-023, TC-028 |
| FR-AMZ-ITEM-AIR-018 | TC-002 |
| FR-AMZ-ITEM-AIR-019 | TC-017, TC-029 |
| FR-AMZ-ITEM-AIR-020 | TC-002, TC-003, TC-007, TC-021 |
| FR-AMZ-ITEM-AIR-021 | TC-002, TC-007, TC-013 |
| FR-AMZ-ITEM-AIR-022 | TC-005, TC-006, TC-010, TC-011, TC-012, TC-014, TC-015 |
| FR-AMZ-ITEM-AIR-023 | TC-007, TC-014, TC-022 |
| FR-AMZ-ITEM-AIR-024 | TC-015 |
| FR-AMZ-ITEM-AIR-025 | TC-009 |
| FR-AMZ-ITEM-AIR-026 | TC-007 |
| FR-AMZ-ITEM-AIR-030 | TC-001, TC-013 |
| FR-AMZ-ITEM-AIR-031 | TC-002, TC-013, TC-015 |
| FR-AMZ-ITEM-AIR-032 | TC-004, TC-011, TC-025 |
| FR-AMZ-ITEM-AIR-033 | TC-004, TC-012, TC-026 |
| FR-AMZ-ITEM-AIR-034 | TC-004, TC-011, TC-012 |
| FR-AMZ-SRCH-DEL-005 | TC-015, TC-019 |
| FR-CLT-SET-013 | TC-027 |
| FR-CLT-SET-014 | TC-027 |
| FR-LBC-AIR-004 | TC-019, TC-029 |
| FR-PURCHASE-GOAL-001 | TC-005, TC-019 |
| FR-PURCHASE-GOAL-008 | TC-007, TC-019, TC-021 |

| Use case | Test cases |
| --- | --- |
| UC-AMZ-ITEM-AIR-001 | TC-005, TC-019 |
| UC-AMZ-ITEM-AIR-002 | TC-006, TC-020 |
| UC-AMZ-ITEM-AIR-003 | TC-002, TC-007, TC-008, TC-017, TC-018, TC-021, TC-028 |
| UC-AMZ-ITEM-AIR-004 | TC-003, TC-009, TC-023 |
| UC-AMZ-ITEM-AIR-005 | TC-010, TC-024, TC-027 |
| UC-AMZ-ITEM-AIR-006 | TC-004, TC-011, TC-025 |
| UC-AMZ-ITEM-AIR-007 | TC-004, TC-012, TC-026 |
| UC-AMZ-ITEM-AIR-008 | TC-001, TC-013 |
| UC-AMZ-ITEM-AIR-009 | TC-015 |
| UC-AMZ-ITEM-AIR-010 | TC-014, TC-022 |
| UC-AMZ-ITEM-AIR-011 | TC-016 |

## Unit (Go)

### TC-001 — The product page parser reads the customer rating without changing the price or second-hand result

- Covers: FR-AMZ-ITEM-AIR-030, FR-AMZ-HCHK-002; UC-AMZ-ITEM-AIR-008
- Level: unit (`internal/amazon`, page fixtures)
- Given product page fixtures: an amazon.fr page showing "4,5 sur 5 étoiles" and "1 234 évaluations"; an amazon.co.uk page showing "4.5 out of 5 stars" and "1,234 ratings" (plus one fixture per other supported marketplace language the technical specification covers); a page with a price and no rating block; a page whose rating block is malformed (no number); the existing fixtures (`price-found.html`, `price-not-found.html`, `unavailable.html`, `used-amazon.html`, …)
- When each page is parsed as a product page answer
- Then the first two give rating 4.5 and 1234 ratings; the page without ratings and the malformed one give rating and count absent, and still give their price; for every existing fixture the price, "price not found", "listing unavailable" and second-hand offer/condition results are identical to today's, and no parse of a rating turns a usable answer into a failure
- Corner cases: rating present without a count (count absent, rating kept) and count without a rating; whole-number rating "5 out of 5"; rating text elsewhere on the page (for example in a recommended product) not taken as the product's rating

### TC-002 — The Amazon review request contains the recorded data and asks for the Amazon review parts in English

- Covers: FR-AMZ-ITEM-AIR-005, FR-AMZ-ITEM-AIR-018, FR-AMZ-ITEM-AIR-020, FR-AMZ-ITEM-AIR-021, FR-AMZ-ITEM-AIR-031; UC-AMZ-ITEM-AIR-003
- Level: unit (review request builder)
- Given item A
- When the Amazon review request is built
- Then it contains the title, `amazon.fr`, the current price 119,99 €, the price history (129,99 €, 119,99 € with dates), the second-hand offer 95,00 € "Used - Very good" and its history (99,00 €), rating 4.5 and 1234 ratings, and the goal "gift for a 10-year-old" as the operator's purchase goal; it asks for an English review with a price rating (Good deal / Fair / Overpriced) and explanation, a fair price range, and an overall recommendation (Buy / Wait for a lower price / Avoid) and explanation, and does not ask for condition, suggested offer, scam/risk signs, missing accessories, seller questions or photo match
- Corner cases: no second-hand offer → the request states there is none; rating never recorded (item from before this feature) → states the rating is unknown; empty goal → no goal is sent; goal with several paragraphs and accents sent unchanged; a Claude token never appears in the built request text

### TC-003 — The Amazon review response is accepted only when it has the Amazon review parts

- Covers: FR-AMZ-ITEM-AIR-020, FR-AMZ-ITEM-AIR-014; UC-AMZ-ITEM-AIR-004
- Level: unit (review response parser)
- Given Claude responses: a complete valid one; one with a price rating outside Good deal / Fair / Overpriced; one with a recommendation outside Buy / Wait for a lower price / Avoid; one missing the fair price range; one that is not valid structured output; one that also contains LeBoncoin-only parts (condition, suggested offer)
- When each response is parsed as an Amazon review
- Then the valid one gives the price rating, its explanation, the fair price range, the recommendation and its explanation; the invalid ones give an AI error (review failure); the LeBoncoin-only parts are not kept in the Amazon review

### TC-004 — Review availability of an Amazon item: available, stopped, or no current price

- Covers: FR-AMZ-ITEM-AIR-032, FR-AMZ-ITEM-AIR-033, FR-AMZ-ITEM-AIR-034; UC-AMZ-ITEM-AIR-006, UC-AMZ-ITEM-AIR-007
- Level: unit (availability decision)
- Given Amazon item states: priced and running; no price recorded yet; latest answer "price not found" after an earlier price; latest answer "listing unavailable"; latest check a retrieval error after a price; stopped with a price; stopped and without a current price
- When the review availability is computed
- Then they are respectively: available; no current price; no current price; no current price; available (the last successful price is current); stopped; stopped (only the stopped reason when both apply)
- Corner cases: the result does not depend on whether a Claude token is saved

## Integration (Go, real SQLite and HTTP handlers)

### TC-005 — Adding an Amazon item saves its goal and starts no review

- Covers: FR-AMZ-ITEM-AIR-001, FR-AMZ-ITEM-AIR-004, FR-AMZ-ITEM-AIR-010, FR-AMZ-ITEM-AIR-022, FR-PURCHASE-GOAL-001; UC-AMZ-ITEM-AIR-001
- Level: integration
- Given a saved Claude token and the fake Amazon browser serving a priced product page
- When the client adds the Amazon URL with goal "gift for a 10-year-old", then gets the item details
- Then the details contain that goal and an AI review state with no latest review and an empty history (not `null`), the first check recorded the price, and fake Claude received no request
- Corner cases: goal "   \n " → stored as empty; a 10 000-character multi-paragraph goal → returned in full (leading/trailing whitespace may be trimmed); no goal field → empty goal; the items list response contains no AI review; an add that fails (invalid URL) stores no item and no goal

### TC-006 — Saving the goal of an Amazon item replaces it and starts no review

- Covers: FR-AMZ-ITEM-AIR-002, FR-AMZ-ITEM-AIR-003, FR-AMZ-ITEM-AIR-004, FR-AMZ-ITEM-AIR-022; UC-AMZ-ITEM-AIR-002
- Level: integration
- Given item A with goal A, one stored review, and a saved token
- When the client saves goal B, then saves B again, then saves an empty goal, then saves a whitespace-only goal
- Then after each save the details return respectively B, B (unchanged, no write), empty, empty; the review and its history are unchanged and fake Claude received no request; a following **Refresh AI review** sends no goal
- Corner cases: saving goal B then requesting a review sends B; saving on an unknown item → 404; in protected mode, saving on another user's Amazon item → not found and their goal unchanged

### TC-007 — Refresh AI review produces a stored Amazon review from recorded data, without any Amazon request

- Covers: FR-AMZ-ITEM-AIR-005, FR-AMZ-ITEM-AIR-010, FR-AMZ-ITEM-AIR-011, FR-AMZ-ITEM-AIR-012, FR-AMZ-ITEM-AIR-020, FR-AMZ-ITEM-AIR-021, FR-AMZ-ITEM-AIR-023, FR-AMZ-ITEM-AIR-026, FR-PURCHASE-GOAL-008; UC-AMZ-ITEM-AIR-003
- Level: integration
- Given item A, a saved token, fake Claude answering a valid Amazon review, and the Amazon navigation counter, price history, next check time and consecutive Amazon failure count noted
- When the client requests an AI review, polls the details until it is no longer running, then requests a second review the same way
- Then the request is accepted (202); while running the details say running; afterwards the latest review has its timestamp, reviewed price 119,99 €, price rating with explanation, fair price range and recommendation with explanation; after the second review the history has both, newest first, each with its timestamp and reviewed price and no goal; fake Claude received the data of TC-002; the Amazon navigation counter, the price history, the next check time and the failure count are unchanged
- Corner cases: the latest check is a retrieval error after 119,99 € → the review is made at 119,99 € with the offer and rating of the latest successful checks; an unknown item → 404; another user's item in protected mode → not found

### TC-008 — Repeated review requests during a run start one review

- Covers: FR-AMZ-ITEM-AIR-013; UC-AMZ-ITEM-AIR-003
- Level: integration
- Given item A, a saved token and fake Claude blocked until released
- When the client sends five review requests (sequential and concurrent) before releasing fake Claude
- Then fake Claude received exactly one request, later requests report the review as already running, and after release exactly one new review is stored

### TC-009 — A failed Amazon review keeps the previous review and is not an Amazon failure

- Covers: FR-AMZ-ITEM-AIR-014, FR-AMZ-ITEM-AIR-025; UC-AMZ-ITEM-AIR-004
- Level: integration
- Given item A with one successful review, a second Amazon item B, 4 consecutive failed Amazon checks recorded, and fake Claude answering in turn 401 (token rejected), 429 (usage limit), 500, a network error, and an invalid review
- When the client requests a review after each answer
- Then each attempt ends failed with its date/time and an understandable error, the latest successful review and history are unchanged, the item can be requested again; after 401 and 429 Settings show the rejected-token warning; the consecutive Amazon failure count is still 4 and neither A nor B is stopped; neither item details, errors nor captured logs contain the token
- Corner cases: no previous review → failed state with no latest review; a retry after the failures, with fake Claude answering a valid review, stores a review and the next successful Amazon check still resets the failure count as today

### TC-010 — Without a Claude token, Amazon items are tracked and not reviewed

- Covers: FR-AMZ-ITEM-AIR-015, FR-AMZ-ITEM-AIR-022; UC-AMZ-ITEM-AIR-005
- Level: integration
- Given no saved token and item A, priced and running, with an earlier review
- When the client gets the details, requests a review, a price check runs, then a token is saved
- Then the details say no token is configured and still show the earlier review; the request is refused with `CLAUDE_TOKEN_MISSING` "Configure a Claude token in Settings." and no review attempt is recorded; the check records its price; after the token is saved fake Claude received no request and the shown review is unchanged

### TC-011 — A stopped Amazon item refuses reviews until it is restarted

- Covers: FR-AMZ-ITEM-AIR-032, FR-AMZ-ITEM-AIR-034, FR-AMZ-ITEM-AIR-022; UC-AMZ-ITEM-AIR-006
- Level: integration
- Given item A stopped by consecutive Amazon failures, with one review
- When the client gets the details and requests a review (with a saved token, then without), then restarts the item with **Refresh price** answered by a priced page
- Then while stopped the details give the stopped reason and the review and history; each request is refused with the stopped refusal and message "Amazon checks are stopped: restart the item with Refresh price to request an AI review.", fake Claude received nothing, and no review or failed attempt is stored; after the restart the details say a review is available and no review is running or stored by the restart
- Corner cases: stopped and with no current price → only the stopped reason; **Refresh price** that fails → still stopped and refused; restart whose check finds no price → no-current-price reason (TC-012)

### TC-012 — An Amazon item without a current price refuses reviews until a price is recorded

- Covers: FR-AMZ-ITEM-AIR-033, FR-AMZ-ITEM-AIR-034, FR-AMZ-ITEM-AIR-022; UC-AMZ-ITEM-AIR-007
- Level: integration
- Given a saved token and three running Amazon items with one review each: one never priced, one whose latest answer is "price not found", one whose latest answer is "listing unavailable"
- When the client gets their details and requests a review, then a check records a price for each
- Then before the check each gives the no-current-price reason, each request is refused with the no-current-price refusal and message "No current price: an AI review needs a current Amazon price.", fake Claude received nothing and no review or failed attempt is stored; after the check each is available, no review was started and the shown review is unchanged
- Corner cases: an item whose latest check is a retrieval error after a price is available (TC-007); the same refusal without a saved token

### TC-013 — Amazon checks record the customer rating; reviews use the latest recorded one

- Covers: FR-AMZ-ITEM-AIR-030, FR-AMZ-ITEM-AIR-031, FR-AMZ-ITEM-AIR-021, FR-AMZ-HCHK-002; UC-AMZ-ITEM-AIR-008
- Level: integration (service with the fake Amazon browser and real SQLite)
- Given an Amazon item added while the fake browser serves a page with 4.5 stars and 1234 ratings
- When the first check runs, then a **Refresh price** check gets a retrieval error, then a review is requested, then a **Refresh all prices** check gets a priced page without ratings, then another review is requested
- Then after the first check the item has 4.5 / 1234 and its price and second-hand result as today; the failed check keeps 4.5 / 1234; the first review request contains 4.5 and 1234; the page without ratings records the price and rating absent; the second review request states the rating is unknown
- Corner cases: a scheduled check records the rating the same way; a "price not found" or "listing unavailable" answer does not invent a rating; the rating is not returned in the items list or as an item field outside the review

### TC-014 — Price changes and other events start no Amazon review; each review keeps its price

- Covers: FR-AMZ-ITEM-AIR-022, FR-AMZ-ITEM-AIR-023, FR-AMZ-ITEM-AIR-011; UC-AMZ-ITEM-AIR-010
- Level: integration
- Given item A with a review made at 100,00 € and a saved token
- When a check records 80,00 € and a different second-hand offer, then the goal is saved, the token is replaced, and the item is stopped and restarted by **Refresh price**
- Then fake Claude received no request, no review is running, and the details show the current price 80,00 € with the same review based on 100,00 €; a following **Refresh AI review** stores a review based on 80,00 € first in the history and keeps the one based on 100,00 €

### TC-015 — Amazon items from before the upgrade get no goal, no review and an unknown rating until refreshed

- Covers: FR-AMZ-ITEM-AIR-007, FR-AMZ-ITEM-AIR-024, FR-AMZ-ITEM-AIR-031, FR-AMZ-ITEM-AIR-022, FR-AMZ-SRCH-DEL-005; UC-AMZ-ITEM-AIR-009
- Level: integration (migration of a database at the previous schema)
- Given a database at the schema before this change with a priced Amazon item, a LeBoncoin item with a goal and reviews, and a saved token
- When the application opens the database (migrations) and starts
- Then the Amazon item has an empty goal, no review and an unknown rating; fake Claude received no request at start-up; the LeBoncoin item's goal and reviews are unchanged; a **Refresh AI review** on the Amazon item stores its first review with the rating sent as absent
- Corner cases: opening the migrated database a second time changes nothing; no Amazon review from before issue #111 reappears

### TC-016 — Deleting an Amazon item deletes its goal, rating and reviews

- Covers: FR-AMZ-ITEM-AIR-016; UC-AMZ-ITEM-AIR-011
- Level: integration
- Given item A with a goal, a recorded rating, two reviews and one failed attempt, and another Amazon item with a review
- When the client deletes item A
- Then the details of A answer 404 and the database holds no review, attempt, goal or rating row for A; the other item's review is unchanged
- Corner cases: deleting while a review of A is running stores no review for A when the run ends

### TC-017 — LeBoncoin reviews and goals are unchanged

- Covers: FR-AMZ-ITEM-AIR-019; UC-AMZ-ITEM-AIR-003
- Level: integration
- Given the existing LeBoncoin review and purchase-goal Go tests, and a LeBoncoin item next to item A with a saved token
- When the existing tests run, and a LeBoncoin item is added and its price changes
- Then the existing LeBoncoin tests pass unchanged (only the Amazon assertions listed in Conventions are rewritten); the LeBoncoin item is reviewed automatically on add and on price change, with its LeBoncoin review parts; an Amazon review request does not change the LeBoncoin item's reviews

### TC-018 — A token replaced during a running Amazon review does not mix outcomes

- Covers: FR-AMZ-ITEM-AIR-014; UC-AMZ-ITEM-AIR-003
- Level: integration
- Given item A and token T, with fake Claude blocked on the review started with T
- When token T is replaced by U (and, separately, removed) before fake Claude is released
- Then the running review finishes or fails with T as the LeBoncoin reviews do today; a rejection of T does not set a warning on U; no new review is started by the token change

## E2E (Playwright, `tests/amazon-item-ai-review.spec.ts`)

### TC-019 — Add an Amazon item with a goal; its details page has the goal field and an AI review section

- Covers: FR-AMZ-ITEM-AIR-001, FR-AMZ-ITEM-AIR-002, FR-AMZ-ITEM-AIR-010, FR-PURCHASE-GOAL-001, FR-PURCHASE-GOAL-008, FR-LBC-AIR-004, FR-AMZ-SRCH-DEL-005; UC-AMZ-ITEM-AIR-001
- Level: e2e
- Given a mocked API with a saved token
- When the user adds an Amazon URL with goal "gift for a 10-year-old" and opens the item details
- Then the add request body contains the goal; the details page shows the labelled **Purchase goal** field containing it and an **AI review** heading with "no review yet" and an enabled **Refresh AI review** button; no review request was sent
- Corner cases: replaces the e2e tests "Amazon items have no AI review section" and "Amazon details have no purchase goal field"

### TC-020 — Edit, clear and fail to save an Amazon goal

- Covers: FR-AMZ-ITEM-AIR-002, FR-AMZ-ITEM-AIR-003, FR-AMZ-ITEM-AIR-006, FR-AMZ-ITEM-AIR-007; UC-AMZ-ITEM-AIR-002
- Level: e2e
- Given an Amazon item with an empty goal (as an item from before this feature) and one review
- When the user types goal B and saves with the keyboard only (Tab to the save action, Enter), then clears the field and saves, then the mocked save answers 500
- Then the field starts empty; the save request carries B, success is announced, no pending review is shown and the review is unchanged (unlike LeBoncoin); clearing sends an empty goal; the failed save shows and announces an error, keeps the typed text, and the review is unchanged

### TC-021 — Refresh AI review shows progress, then the labelled review and its history

- Covers: FR-AMZ-ITEM-AIR-010, FR-AMZ-ITEM-AIR-011, FR-AMZ-ITEM-AIR-012, FR-AMZ-ITEM-AIR-013, FR-AMZ-ITEM-AIR-017, FR-AMZ-ITEM-AIR-020, FR-PURCHASE-GOAL-008; UC-AMZ-ITEM-AIR-003
- Level: e2e
- Given an Amazon item with one earlier review and a saved token; the mocked review endpoint answers 202 and the details answer "running" twice, then a new review
- When the user activates **Refresh AI review** (once by mouse, double-clicked; once by keyboard)
- Then exactly one review request is sent per activation, the button is disabled and the progress is announced (live region) while running; the new review shows its date/time, "Based on price 129,99 €", the labelled **Price** rating in text (for example "Good deal") with its explanation, the fair price range, and the labelled **Overall recommendation** (for example "Wait for a lower price") with its explanation; no condition, suggested offer, risk signs, accessories, seller questions or photo match are shown; the previous reviews are reachable by keyboard, newest first, each with date/time and reviewed price and without the goal

### TC-022 — A review made at another price is labelled with its own price

- Covers: FR-AMZ-ITEM-AIR-023; UC-AMZ-ITEM-AIR-010
- Level: e2e
- Given an Amazon item with current price 80,00 € and a latest review based on 100,00 €
- When the user opens the details page
- Then the item shows 80,00 € as its price and the review is labelled "Based on price 100,00 €"; no review is pending

### TC-023 — A failed Amazon review shows a dated error and keeps the previous review

- Covers: FR-AMZ-ITEM-AIR-014, FR-AMZ-ITEM-AIR-017; UC-AMZ-ITEM-AIR-004
- Level: e2e
- Given an Amazon item whose last attempt failed after a successful review, and another with a failed attempt and no review
- When the user opens each details page
- Then the first shows an understandable error with its date/time, announced, and the previous review with its own date and reviewed price, with **Refresh AI review** enabled; the second shows the error and "no review yet" with the button enabled

### TC-024 — No Claude token: message and disabled button

- Covers: FR-AMZ-ITEM-AIR-015; UC-AMZ-ITEM-AIR-005
- Level: e2e
- Given an Amazon item, priced and running, with an earlier review, and no saved token
- When the user opens its details page
- Then the prices are shown, the section says "Configure a Claude token in Settings", shows the earlier review, and **Refresh AI review** is disabled

### TC-025 — Stopped Amazon item: review visible, message, no button; back after restart

- Covers: FR-AMZ-ITEM-AIR-032; UC-AMZ-ITEM-AIR-006
- Level: e2e
- Given a stopped Amazon item with a review, with a saved token (and, in a second test, without one)
- When the user opens the details page, then chooses **Refresh price** and the mocked API returns the item restarted with a price
- Then before the restart the review is shown, the message "Amazon checks are stopped: restart the item with Refresh price to request an AI review." is shown and no **Refresh AI review** button exists (also no token message in the no-token test); after the restart the message is gone, the button is shown (enabled with a token, disabled without) and no review is pending
- Corner cases: stopped and without a current price → only the stopped message

### TC-026 — Amazon item without a current price: review visible, message, no button; back after a price

- Covers: FR-AMZ-ITEM-AIR-033; UC-AMZ-ITEM-AIR-007
- Level: e2e
- Given a running Amazon item whose latest check is "price not found", with a review and a saved token
- When the user opens the details page, then chooses **Refresh price** and the mocked API returns a recorded price
- Then before the refresh the review is shown, the message "No current price: an AI review needs a current Amazon price." is shown and no **Refresh AI review** button exists; afterwards the message is gone, the button is shown and the shown review is unchanged
- Corner cases: the same message for an item never priced and for "listing unavailable"

### TC-027 — The Claude token help text mentions Amazon and LeBoncoin reviews

- Covers: FR-CLT-SET-014, FR-CLT-SET-013; UC-AMZ-ITEM-AIR-005
- Level: e2e (`tests/leboncoin-session-settings.spec.ts` or `tests/claude-token-settings.spec.ts`)
- Given the Settings modal
- When it is opened
- Then the Claude token help text states that the token is used for Amazon and LeBoncoin AI reviews, and the superseded "used for LeBoncoin AI reviews." wording is gone

### TC-028 — The Amazon AI review section is usable at phone width

- Covers: FR-AMZ-ITEM-AIR-017; UC-AMZ-ITEM-AIR-003
- Level: e2e (viewport 390 × 844, as `tests/responsive-layout.spec.ts`)
- Given an Amazon item with a long goal, a review with long explanations and three history entries
- When the details page is shown at phone width
- Then the page has no horizontal scrolling, the goal field, the **AI review** heading, the ratings (as text), the reviewed price and the history are visible and reachable by keyboard

### TC-029 — LeBoncoin details pages keep their review and goal behaviour

- Covers: FR-AMZ-ITEM-AIR-019, FR-LBC-AIR-004
- Level: e2e (existing `tests/leboncoin-ai-review.spec.ts`, `tests/leboncoin-purchase-goal.spec.ts`)
- Given the existing LeBoncoin e2e tests
- When they run
- Then all pass unchanged except the two Amazon-negative tests replaced by TC-019; saving a LeBoncoin goal still shows a pending review and LeBoncoin reviews still show their LeBoncoin parts
