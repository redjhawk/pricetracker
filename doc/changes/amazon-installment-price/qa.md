# QA execution: amazon-installment-price (issue #45)

Status: executed — **failed** (QA-001, finding QA-F-001)
Tested code/specification revisions: branch `ai-dev/issue-45-20261005-2049` at `ce79e10` (PR #49); specs `doc/specifications/amazon-price-detection/functional.md`, `technical.md`
Environment: `npm run dev` (Vite at http://localhost:5173/, proxying `/api` to the Go API at http://127.0.0.1:3001, `PRICEFOLLOWER_ENV=development`, local SQLite `.data/pricefollower.sqlite` with seeded sample items); Playwright 1.63.0 headless Chromium; anonymous (open) mode; outbound access to amazon.fr **was available**.
Executed at: 2026-10-05 ~20:56–21:05 (runner local time, UTC)
Tester: QA tester agent (independent of implementation)

## Exploratory session

There was no seeded randomness. The actions ran in a fixed order in one browser session: load `/`, open the "Add item" modal, then submit these listing URLs in order: (1) issue URL, (2) empty, (3) `not a url`, (4) `https://www.amazon.fr/`, (5) `https://example.com/dp/B0F99B78JN`, (6) the issue URL again (duplicate). After that, a separate Chromium page (locale fr-FR) opened the live Amazon page to check which price Amazon actually displays.

## Executed cases

| Case | Requirement / corner case | Actual interface URL | Actual source listing URL | Preconditions and exact actions | Expected | Observed | Result / evidence / finding |
| --- | --- | --- | --- | --- | --- | --- | --- |
| QA-001 | FR-AMAZON-PRICE-001, -002 (issue #45 page) | http://localhost:5173/ | https://www.amazon.fr/dp/B0F99B78JN | Seeded DB. Clicked "Add item", filled the URL, clicked the modal's "Add item", waited 25 s. | Item stored with 514,63 €; 128,65 € never stored. | `POST /api/v1/items` returned 201. The row "ASUS Vivobook 15 F1504VA-BQ128W …", amazon.fr · B0F99B78JN, shows **128,65 €** (05 Oct, 20:59). Second-hand column: "Amazon offer check unavailable". | **FAIL**. The instalment amount was stored. Finding QA-F-001. |
| QA-002 | Supporting evidence for QA-001 (live page markup) | n/a (direct Chromium visit) | https://www.amazon.fr/dp/B0F99B78JN | Opened the page and read the price elements. | n/a | `.priceToPay` / `#corePriceDisplay_desktop_feature_div` = 514,63 €. `#apex_desktop` contains "Ou 128,65€ x4 (0,0% de frais inclus)". The instalment price is an `a-price` inside `span#price-block-message.price-block-message > span#price-block-amount-prefix` ("Ou ") `+ span#price-block-amount`. It has no "mois" text and no installment-named container. | Confirms the real price is 514,63 € and shows the markup that the parser's instalment filter misses. |
| QA-003 | Corner case: empty URL | http://localhost:5173/ | (empty) | Cleared the field and clicked "Add item". | Validation error, no item. | No request was sent. "Could not add item: Enter an Amazon or LeBoncoin listing URL." | PASS |
| QA-004 | Corner case: malformed URL | http://localhost:5173/ | `not a url` | Filled the field and submitted. | Validation error. | 400 `INVALID_URL`. "Enter a valid HTTPS Amazon or LeBoncoin listing URL." | PASS |
| QA-005 | Corner case: Amazon URL with no ASIN | http://localhost:5173/ | https://www.amazon.fr/ | Filled the field and submitted. | Rejected. | 422 `UNSUPPORTED_LISTING`. "This listing is outside the supported Amazon euro marketplaces and LeBoncoin listings." | PASS |
| QA-006 | Corner case: unsupported host | http://localhost:5173/ | https://example.com/dp/B0F99B78JN | Filled the field and submitted. | Rejected. | 422 `UNSUPPORTED_LISTING` (same message). | PASS |
| QA-007 | Corner case: duplicate | http://localhost:5173/ | https://www.amazon.fr/dp/B0F99B78JN | Submitted again after QA-001. | Rejected as a duplicate. | 409 `ITEM_ALREADY_TRACKED`. "This listing is already being tracked." | PASS |

Supporting non-interface evidence: `go test -v ./internal/amazon/` passed. All 6 `TestProductPriceIgnoresInstalments` subtests passed (issue_45_instalment_after_price, instalment_before_price, instalment_container, german_instalment, instalment_only, normal_price). The fixtures do not reproduce the live `price-block-message` markup, which is why the tests pass while QA-001 fails.

## Coverage and limitations

- FR-001/-002 were tested against the live page: **failed**.
- FR-003 (instalment-only page) and FR-004 (normal page) were covered only by unit tests. No live page of those kinds was tested.
- Amazon content can vary with locale, session, geo and A/B tests. The observation above was collected at about 20:59–21:05 on 2026-10-05.
- Not tested: refresh of an existing item, keyboard-only journey, and logged-in mode.

## Handoff and cleanup

- **QA-F-001 (critical, FR-AMAZON-PRICE-001/002)**: on the live issue #45 page, 128,65 € is stored instead of 514,63 €. The instalment block `#price-block-message` / `#price-block-amount-prefix` ("Ou ") / `#price-block-amount` is not recognised as an instalment. The fix should also exclude those candidates (or prefer `.priceToPay`), and a fixture with this markup should be added. This goes to the adjudicator, then to development, then a QA-001 retest.
- Cleanup: the dev server processes were stopped and ports 3001 and 5173 were confirmed down. Temporary scripts were deleted. The test item remains in the local, untracked `.data` SQLite file only.
