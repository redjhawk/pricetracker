# Questions — issue #165 (use all space for the item list on reduced screens)

The screenshot attached to the issue could not be read by the pipeline, so the yellow area is unknown. Context: today, between 672 px and 1055 px wide (the screenshot is 775 px wide), the tracked-items table is shown as stacked cards. Each card lists Item, Prices, Amazon second-hand offer (Amazon tab), Status and the row actions one below the other, so most of the card's right side stays empty. The list also keeps lateral page margins at these widths. Below 672 px the cards already span the full screen width (FR-RESPONSIVE-014).

1. **Which unused space should the list fill?**
   - A. The empty right-hand part of each item card: place the card's values side by side (for example Item on the left with Prices, offer, Status and actions next to it, or several values per line) so that the card width is used.
   - B. The lateral page margins around the list: make the list span the full window width, edge to edge, as on phones.
   - C. Both A and B.
   - D. Something else (please describe the yellow area).

2. **At which window widths does the change apply?**
   - A. Only the intermediate (tablet) range where cards are shown today, 672 px to 1055 px.
   - B. Every width below desktop (below 1056 px), including phones.
   - C. Every width, including desktop (for example the margins on wide screens beyond 1536 px, where the page stops growing).
   - D. Other range (please state it).

3. **If question 1 includes A (values side by side): should the layout keep the current stacked cards with labels, or return to a table with column headers at these widths?**
   - A. Keep cards with each value labelled ("Prices", "Amazon second-hand offer", "Status"), arranged side by side.
   - B. Show a table with column headers (Item, Prices, Amazon second-hand offer, Status, actions) that fits the width without sideways scrolling.
   - C. No preference; any arrangement that uses the width, keeps every value and action visible without sideways scrolling, and keeps each value identifiable.

4. **Does the change cover only the tracked-items list, or also other screens that leave space unused at reduced widths?**
   - A. Only the tracked-items list (Amazon and LeBoncoin tabs).
   - B. Also the administrator users list.
   - C. Also the item details page.
   - D. Other (please list).
