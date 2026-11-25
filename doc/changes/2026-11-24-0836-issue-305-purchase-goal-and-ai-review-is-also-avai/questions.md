# Questions for issue #305: purchase goal and AI review for Amazon items

Thank you for the answers to Q-1–Q-5. They are recorded as D-1–D-5 in the [specification](../../specifications/amazon-item-ai-review/functional.md): review content B, manual **Refresh AI review** only with the reviewed price shown, no automatic review of existing items, and the review receives a + d + e. Three points remain.

6. **Where the review information is read.** Q-4 did not choose between f and g. Today an Amazon check records the price and the Amazon-sold second-hand offer, but not the customer rating and number of ratings (Q-4 e). When you click **Refresh AI review**:
   - a) The server reads the Amazon product page again (one extra Amazon request per review) and sends what it reads now: current price, second-hand offer and customer rating. The current price read this way is also recorded in the price history, as a price check would. A failed read fails the review and, per your answer to Q-5 a, is not counted toward the 5 consecutive Amazon failures.
   - b) Same as a), but the price read for the review is not recorded in the price history.
   - c) No Amazon request: the review uses the latest check. Price checks are extended to also record the customer rating and number of ratings.

7. **Refresh AI review on a stopped item.** Your answer to Q-5 b was "ai refresh is not required. inform the user". Which one do you mean?
   - a) While the item is stopped, **Refresh AI review** sends no request: it is disabled or refused, with a message such as "Amazon checks are stopped: restart the item with Refresh price before requesting an AI review".
   - b) While the item is stopped, the review still runs, without any Amazon request, from the latest recorded data, and the section tells the user that the data date from the last successful check (with its date/time).

8. **Item without a current price.** When the latest check found no price ("price not found" or "listing unavailable"), what does **Refresh AI review** do?
   - a) It is refused with a message (for example "No current price: refresh the price first").
   - b) The review still runs from the recorded price history and states that no current price is available.
