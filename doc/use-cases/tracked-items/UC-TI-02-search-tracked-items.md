# UC-TI-02 — Search tracked items

**Primary actor:** Operator
**Supporting system:** Tracked-items page
**Trigger:** The operator enters text in the list search field.

## Preconditions

- The tracked-items page has loaded.

## Main flow

1. The operator enters a search term.
2. The page filters the already-loaded items by title, listing ID, ASIN, platform, and marketplace, without sending a separate search request.
3. The page shows matching rows as the search term changes.

## Alternatives and errors

- If there are no matches, the page says that no items match the search.
- If the operator clears the search field, the full loaded collection appears again.

## Postconditions

- The server collection is unchanged; filtering exists only in the current page view.
