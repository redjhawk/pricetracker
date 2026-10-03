# UC-TI-02 — Search tracked items

**Primary actor:** Operator
**Supporting system:** Tracked-items page
**Trigger:** The operator enters text in the list search field.

## Preconditions

- The tracked-items page has loaded.

## Main flow

1. The operator enters a search term.
2. The page filters the already-loaded items of the selected platform by title, listing ID, ASIN, platform, and marketplace, ignoring surrounding whitespace and letter case, without sending a separate search request.
3. The page shows matching rows as the search term changes.
4. Switching between Amazon and LeBoncoin retains the typed query and applies it to the newly selected platform.

## Alternatives and errors

- If there are no matches, the page says that no items match the search.
- A match in the other platform does not appear until that platform's tab is selected.
- If the operator clears the search field, all loaded items of the selected platform appear again.

## Postconditions

- The server collection is unchanged; filtering exists only in the current page view.
