# Questions for issue #305: purchase goal and AI review for Amazon items

The purchase goal and the AI review section, refresh button, history, no-token and error states are specified as for LeBoncoin ([draft](../../specifications/amazon-item-ai-review/functional.md)). The points below do not carry over from LeBoncoin and need your answer.

1. **Review content.** A LeBoncoin review has: Price rating (Good deal / Fair / Overpriced), Condition rating from the photos (Excellent / Good / Fair / Poor), Overall recommendation (Buy / Negotiate / Avoid), fair price range and suggested offer, scam/risk signs, missing accessories and questions for the seller, and whether the description matches the photos. Several parts suit second-hand listings more than new Amazon products. What should an Amazon review contain?
   - a) Exactly the same parts as LeBoncoin.
   - b) Price rating with explanation, fair price range, and Overall recommendation with explanation, with Buy / Wait for a lower price / Avoid replacing Negotiate. No condition, scam, seller questions or photo-match parts.
   - c) Price rating with explanation only, as in the earlier Amazon search reviews (issue #56).
   - d) Something else (please list the parts).

2. **Automatic reviews.** With a token saved, LeBoncoin items are reviewed automatically when added, when a check records a different price, and when the goal changes. Amazon prices can change often (two checks a day), and each review uses your Claude usage. In issue #56 you chose not to review again on a price change, only to show that the last review was made at an older price. Which automatic triggers apply to Amazon items? Choose any combination:
   - a) When the item is added.
   - b) When a check records a different price.
   - c) Instead of b: a price change starts no review; the section shows "Price changed: last AI review was based on X €".
   - d) When the goal changes.
   - e) When the Amazon-sold second-hand offer changes.
   - f) None: manual **Refresh AI review** only.

3. **Amazon items added before this feature.** Should they:
   - a) not be reviewed automatically, with **Refresh AI review** or another trigger from Q-2 producing the first review (as for LeBoncoin), or
   - b) each be reviewed once automatically when a token is available?

4. **Information sent to the AI.** Today an Amazon check records the title, price, thumbnail and the Amazon-sold second-hand offer. What should a review receive? Choose any combination:
   - a) Title, marketplace, current price and recorded price history.
   - b) Product description, feature bullets and technical details from the product page.
   - c) Product photos.
   - d) The Amazon-sold second-hand offer (price, condition) and its history.
   - e) Customer rating and number of ratings.

   And where should it come from?
   - f) From the latest price check only: no extra Amazon request for a review (the check records what the review needs).
   - g) From a new Amazon page request made at review time, as LeBoncoin reviews read the listing again.

5. **Amazon failure stop.** Only relevant if a review makes its own Amazon request (Q-4 g).
   - a) Does a failed review request count toward the 5 consecutive Amazon failures that stop all Amazon items? (yes / no)
   - b) While Amazon items are stopped, is **Refresh AI review** refused with a message, or still allowed?
