# Functional specification: LeBoncoin old price

Status: ready
Owner: functional specification agent
User decision/reference: GitHub issue #42, "Add older price for leboncoin when available".

> Apparently, when requesting a leboncoin page, there is a property called 'old_price' that contains the older price. It could be really interesting to include this detected older price when a new article is added (and only when the article is added, no read of this property after adding the product). If there is a date, it can also be added. If there is not a previous date, just say "old price". This price can be added as a historical price instead of creating a new column. Please, update the functionality to the functionalities file.

## Purpose and scope

When a LeBoncoin item is added, the listing may advertise an older price (the listing's `old_price` property). The user wants that older price kept in the item's price history so the earlier price is visible from the start of tracking.

Actor: the user adding a LeBoncoin item (the operator in open mode, or the logged-in user in protected mode).

Included: reading the old price once, from the collection attempt triggered by adding the item; recording it as a historical price observation of that item; showing it with its date when the listing provides one, otherwise labelled "Old price".

Excluded: Amazon items; reading the old price on scheduled, manual, or "refresh all" collections; items added before this feature (no back-fill); a new list column or a new dedicated field; any change to how the current price is detected.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-LBC-OLD-PRICE-001 | A LeBoncoin item is accepted and its immediate post-add collection attempt (FR-14) succeeds on a listing that has an old price. | The old price is recorded once as a historical price observation of the item, in euros, in addition to the current price observation. | Given a new LeBoncoin item whose listing has an old price, when the add-time collection succeeds, then the item's price history contains exactly one extra entry with that old price. |
| FR-LBC-OLD-PRICE-002 | Any collection other than the add-time collection (scheduled, item refresh, refresh all, retries). | The old price is not read; no old-price entry is added or changed. | Given an existing item, when the listing's old price appears or changes and the item is refreshed, then no new old-price entry appears and an existing one is unchanged. |
| FR-LBC-OLD-PRICE-003 | The listing provides a date for the old price. | The old-price entry carries that date and is shown with it like any other history entry. | Given a listing with an old price and its date, then the details history shows the old price at that date. |
| FR-LBC-OLD-PRICE-004 | The listing provides no usable date for the old price. | The entry is shown with the label "Old price" in place of the date/time. | Given a listing with an old price and no date, then the history entry shows the price with the text "Old price" instead of a date. |
| FR-LBC-OLD-PRICE-005 | An old-price entry exists. | It is part of the item's price history, not a new column: it appears wherever price history is shown (details history per FR-06/FR-20, list price periods per FR-05). An undated entry is ordered as older than all collected observations; a dated one is ordered by its date. | Given an item with an old-price entry, then the details page lists it among the price observations and the list has no new column. |
| FR-LBC-OLD-PRICE-006 | The listing has no old price, or it has no usable numeric euro value. | Nothing is recorded for the old price; adding and collecting behave as today. | Given a listing without an old price, then the history contains only collected observations. |
| FR-LBC-OLD-PRICE-007 | The add-time collection fails (retrieval error, price not found, unavailable). | No old-price entry is recorded, and later collections do not record one (FR-LBC-OLD-PRICE-002). | Given an add whose first collection fails, when a later collection succeeds, then no old-price entry exists. |
| FR-LBC-OLD-PRICE-008 | The item is deleted. | The old-price entry is deleted with the rest of the item's history (FR-09). | Given a deleted item, none of its history, including the old price, remains. |
| FR-LBC-OLD-PRICE-009 | An Amazon item is added. | Not affected. | Adding an Amazon item never creates an old-price entry. |

## States and corner cases

- Old price equal to or lower than the current price: the user did not restrict the value, so it is recorded when present and valid; display follows existing history rules (FR-05 merges consecutive equal prices in list periods; the details page shows every observation).
- The old-price entry is not a collection attempt: it does not change the latest price or last-check time, which remain those of the current collected price.
- AI reviews: adding the item already triggers a review; the old-price entry triggers no additional review.
- Accessibility: the "Old price" label is plain text in the date position, readable by screen readers.

## Open questions

None. Direct interpretations of the request: "when the article is added" means only the immediate collection attempt triggered by the add (FR-LBC-OLD-PRICE-002/007); "just say old price" means the label "Old price" in place of the date (the interface is in English).

## Traceability

- [FUNCTIONAL_SPECIFICATIONS.md](../../FUNCTIONAL_SPECIFICATIONS.md) sections 6, 8 (FR-05, FR-06, FR-09, FR-14, FR-20, FR-31), 9, 15.
- Use cases: [UC-PC-01](../../use-cases/price-collection/UC-PC-01-collect-and-record-prices.md), [UC-ID-01](../../use-cases/item-details/UC-ID-01-view-item-details.md).
- Technical specification: [technical.md](technical.md).
- Change index: [index.md](../../changes/2026-10-05-2041-issue-42-leboncoin-old-price/index.md).
