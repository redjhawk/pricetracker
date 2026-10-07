# Amazon searches with AI review (issue #56)

Stage: technical specification (next)
State: functional specification ready; no open functional question.

User request (GitHub issue #56, redjhawk):

> Black Friday are approaching and it is difficult to review all interesting material. I'd need this application to do it for me.
> Application should be able to receive, in a new tab, a filter coming from amazon (ex: https://www.amazon.fr/joursprime/?_encoding=UTF8&...&discounts-widget=...&promotionsSearchLastSeenAsin=B0HBWZ42P9&promotionsSearchStartIndex=0&promotionsSearchPageSize=60) This returns a list of elements.
> You can do this on a new tab and you can keep track on each research. For each research, you also keep track of the first 30 items (so you keep the link and the title and the price). Each item information is shown as like the others items. Price and history of price.
> And for each item, you perform an AI analysis using claude and the token the user has added to do review on leboncoin. So you also have to change the description of the text to say that it will be used to do amazon review.
> Review of the item must consider price.
> You should simulate human behavior (so take your time to click on an item and review all data, and then, close the link). This order IS REALLY IMPORTANT. There's no hurry.
> Request must be done between 10PM and 01.00AM paris time. And from 6AM to 8AM.
> If more than 5 request start failing, stop requesting Amazon and tell the user. IF there is this problem, log it. Log the error return from the server.

## Scope

New Amazon searches tab with frozen first-30 items, price tracking and history, sharing with tracked Amazon items, move to tracked list, deletion; human-like request pacing within time windows with a global Amazon stop after failures; Claude AI review of search items; Claude token help text update. Cross-tier (frontend, backend, API).

## User functional decisions (redjhawk)

- D-1: "Search items is another level on the interface: an "Amazon searchs" tab. Inside, the list of added searches. Clicking a search shows its first 30 items."
- D-2: Each item keeps link, title and price and is shown like other items: price and price history.
- D-3: "Those 30 items are not dynamic: they are the first 30 items first got. Refresh is done like for the other amazon items."
- D-4: "Always the 30 first elements, no parameter, no other filter."
- D-5: "If an item drops, show it as "unreachable", as done for amazon/leboncoin items."
- D-6: "If the same item appears in multiple searches, the item is shared. Same if it is added on the amazon item-by-item tab."
- D-7: "User can move an element from the search list to the list of tracked amazon elements. No need for the other way around. A moved item stays visible on the search too."
- D-8: "Once added, can only delete it (and so all related items, unless the item has been moved to the amazon tracked list). No rename, no edit."
- D-9: "What to judge: the price regarding characteristics. No goal for now."
- D-10: "Store detected price and keep a history (as other amazon items)."
- D-11: "When to review: as soon as information is got from Amazon. Act as a human: do the search, take time to open an item link, take time to review, go to next item."
- D-12: "Token description: say it will be used for amazon and leboncoin reviews."
- D-13: "Review language and format: English."
- D-14: "Review again: when price changed, just inform that the price has changed and the last AI review was done with the older price (indicate that price)."
- D-15: "Users: searches belong to each user; also usable in open mode (as other items)."
- D-16: "Time window only applies to items added as search items (10PM-01AM and 6AM-8AM Paris time)."
- D-17: "Pauses between 30 seconds and 2 minutes, random." / "5 failures then stop ALL requests to Amazon, even those for individual tracked elements."
- D-18: "If there's a success, continue; failures are kept to be tested again once all the others have been done."
- D-19: "After a stop: show a refresh button by a search item on the list of searches. Inform with an information line below the search item; it disappears if a refresh fixes the problem."
- D-20: "Restart (refresh after stop): starts checking the items again, review included; price history must not be lost."
- D-21: "Tracked amazon tabs: agent's choice" — decision: keep the existing tracked Amazon tab unchanged.

Recorded interpretation: the 5-failure limit counts consecutive failed Amazon requests; a success resets the count (D-18).

## Subjects

1. Amazon searches — [functional](../../specifications/amazon-searches/functional.md) FR-AMZ-SEARCH-001–016; ready.
2. Amazon human browsing — [functional](../../specifications/amazon-human-browsing/functional.md) FR-AMZ-HUMAN-001–012; ready.
3. Amazon AI review — [functional](../../specifications/amazon-ai-review/functional.md) FR-AMZ-AIR-001–010; ready.
4. Claude token settings (amendment) — [functional](../../specifications/claude-token-settings/functional.md) FR-CLT-SET-012; ready.

Global summary: [FUNCTIONAL_SPECIFICATIONS.md](../../FUNCTIONAL_SPECIFICATIONS.md) §5.5, §5.6b, FR-32–FR-34, §15.

## Stage results

1. Functional specification — ready (2026-10-07).
2–10. Not started.
