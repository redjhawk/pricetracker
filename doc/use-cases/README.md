# PriceFollower use cases

Each use case is stored in its own file. Related use cases are grouped by subject.

## Tracked items

- [UC-TI-01 — Review tracked items](tracked-items/UC-TI-01-review-tracked-items.md)
- [UC-TI-02 — Search tracked items](tracked-items/UC-TI-02-search-tracked-items.md)
- [UC-TI-03 — Refresh all prices](tracked-items/UC-TI-03-refresh-all-prices.md)
- [UC-TI-04 — Open a source listing](tracked-items/UC-TI-04-open-source-listing.md)
- [UC-TI-05 — Review tracked items at tablet width](tracked-items/UC-TI-05-review-tracked-items-at-tablet-width.md)
- [UC-TI-06 — Review tracked items full width on a phone](tracked-items/UC-TI-06-review-tracked-items-full-width-on-phone.md)
- [UC-TI-07 — Review tracked items edge to edge below desktop width](tracked-items/UC-TI-07-review-tracked-items-edge-to-edge-below-desktop.md)

## Item details

- [UC-ID-01 — View item details](item-details/UC-ID-01-view-item-details.md)
- [UC-ID-02 — Refresh an item's price](item-details/UC-ID-02-refresh-item-price.md)

## Item management

- [UC-IM-01 — Add a listing](item-management/UC-IM-01-add-listing.md)
- [UC-IM-02 — Delete a tracked item](item-management/UC-IM-02-delete-tracked-item.md)

## Purchase goal choice

- [UC-GOAL-CHOICE-001 — Add a LeBoncoin item with an existing purchase goal](purchase-goal-choice/UC-GOAL-CHOICE-001-add-leboncoin-item-with-existing-goal.md)
- [UC-GOAL-CHOICE-002 — Add a LeBoncoin item with a new purchase goal when no existing goal fits](purchase-goal-choice/UC-GOAL-CHOICE-002-add-leboncoin-item-with-new-goal.md)
- [UC-GOAL-CHOICE-003 — Enter a purchase goal when no existing goal is available](purchase-goal-choice/UC-GOAL-CHOICE-003-enter-first-goal-without-existing-goals.md)
- [UC-GOAL-CHOICE-004 — Change an item's purchase goal on its details page](purchase-goal-choice/UC-GOAL-CHOICE-004-change-goal-on-details-page.md)
- [UC-GOAL-CHOICE-005 — Adapt a goal used by several items for one item only](purchase-goal-choice/UC-GOAL-CHOICE-005-adapt-shared-goal-text-for-one-item.md)
- [UC-GOAL-CHOICE-006 — A goal leaves the list once no item uses it](purchase-goal-choice/UC-GOAL-CHOICE-006-goal-leaves-list-when-no-item-uses-it.md)
- [UC-GOAL-CHOICE-007 — Add an Amazon item while a goal is chosen](purchase-goal-choice/UC-GOAL-CHOICE-007-add-amazon-item-with-goal-chosen.md)
- [UC-GOAL-CHOICE-008 — View the goal of an item shared by another user](purchase-goal-choice/UC-GOAL-CHOICE-008-view-goal-of-shared-item.md)

## Responsive layout

- [UC-RL-01 — Use the full screen width on every page below desktop width](responsive-layout/UC-RL-01-use-full-width-on-every-page-below-desktop.md)
- [UC-RL-02 — Review the users list edge to edge below desktop width](responsive-layout/UC-RL-02-review-users-list-edge-to-edge-below-desktop.md)

## Application header menu

- [UC-APP-MENU-001 — Close the header menu with Escape right after opening it](app-header-menu/UC-APP-MENU-001-close-menu-with-escape-right-after-opening.md)

## Application information

- [UC-APP-INFO-001 — Keep keyboard focus inside the Information modal](app-information/UC-APP-INFO-001-keep-focus-inside-information-modal.md)

## Login attempt limits

- [UC-LOGIN-LIM-001 — Locked out after repeated failed logins from one client](login-attempt-limits/UC-LOGIN-LIM-001-locked-out-after-repeated-failures-from-one-client.md)
- [UC-LOGIN-LIM-002 — Log in while another client is locked out for the same username](login-attempt-limits/UC-LOGIN-LIM-002-log-in-while-another-client-is-locked-out.md)
- [UC-LOGIN-LIM-003 — Log in when too many password checks are waiting](login-attempt-limits/UC-LOGIN-LIM-003-log-in-when-too-many-password-checks-are-waiting.md)

## Session cookie security

- [UC-COOKIE-001 — Log in through the HTTPS reverse proxy](session-cookie-security/UC-COOKIE-001-log-in-through-https-reverse-proxy.md)
- [UC-COOKIE-002 — Log in over plain HTTP in development](session-cookie-security/UC-COOKIE-002-log-in-over-plain-http-in-development.md)
- [UC-COOKIE-003 — Log out behind the HTTPS proxy or over plain HTTP](session-cookie-security/UC-COOKIE-003-log-out-behind-proxy-or-over-plain-http.md)

## HTTP server timeouts

- [UC-HTTP-TMO-001 — A client that sends its request too slowly is dropped](http-server-timeouts/UC-HTTP-TMO-001-slow-client-request-is-dropped.md)
- [UC-HTTP-TMO-002 — A client that reads its response too slowly is cut off](http-server-timeouts/UC-HTTP-TMO-002-slow-reader-response-is-cut-off.md)
- [UC-HTTP-TMO-003 — Send a request with large request headers](http-server-timeouts/UC-HTTP-TMO-003-request-with-large-headers.md)
- [UC-HTTP-TMO-004 — Save the Claude token while Claude is slow to verify it](http-server-timeouts/UC-HTTP-TMO-004-save-claude-token-with-slow-verification.md)
- [UC-HTTP-TMO-005 — Use the application normally with the time limits in place](http-server-timeouts/UC-HTTP-TMO-005-use-application-normally-with-limits.md)

## Masked secrets in Settings

- [UC-SECRET-001 — See which secrets are saved in Settings](settings-secret-masking/UC-SECRET-001-see-which-secrets-are-saved.md)
- [UC-SECRET-002 — Replace the LeBoncoin session without touching the Claude token](settings-secret-masking/UC-SECRET-002-replace-leboncoin-session-only.md)
- [UC-SECRET-003 — Replace the Claude token without touching the LeBoncoin session](settings-secret-masking/UC-SECRET-003-replace-claude-token-only.md)
- [UC-SECRET-004 — Remove a saved secret](settings-secret-masking/UC-SECRET-004-remove-a-saved-secret.md)
- [UC-SECRET-005 — Change the LeBoncoin session after it changed since Settings opened](settings-secret-masking/UC-SECRET-005-change-session-after-it-changed-since-opening.md)
- [UC-SECRET-006 — Follow the operator guide to renew or remove the session](settings-secret-masking/UC-SECRET-006-follow-operator-guide-for-settings-secrets.md)
- [UC-SECRET-007 — Save with a long whitespace-only input](settings-secret-masking/UC-SECRET-007-save-with-long-whitespace-only-input.md)

## Price collection

- [UC-PC-01 — Collect and record prices](price-collection/UC-PC-01-collect-and-record-prices.md)
- [UC-PC-02 — Restart a stopped Amazon item while Refresh all is pending](price-collection/UC-PC-02-restart-stopped-amazon-item-during-refresh-all.md)

## Amazon human-like checks

- [UC-AMZ-HCHK-001 — Scheduled Amazon checks at random times](amazon-human-checks/UC-AMZ-HCHK-001-scheduled-amazon-checks-at-random-times.md)
- [UC-AMZ-HCHK-002 — Add an Amazon item and start its random schedule](amazon-human-checks/UC-AMZ-HCHK-002-add-amazon-item-and-start-random-schedule.md)
- [UC-AMZ-HCHK-003 — Refresh Amazon prices immediately](amazon-human-checks/UC-AMZ-HCHK-003-refresh-amazon-prices-immediately.md)
- [UC-AMZ-HCHK-004 — Restart stopped Amazon items once after the upgrade](amazon-human-checks/UC-AMZ-HCHK-004-restart-stopped-amazon-items-once-after-upgrade.md)
- [UC-AMZ-HCHK-005 — Amazon check answered with a page the browser does not open](amazon-human-checks/UC-AMZ-HCHK-005-amazon-check-answered-with-page-the-browser-does-not-open.md)
- [UC-AMZ-HCHK-006 — Unavailable listing page the browser does not open sends no used-offer request](amazon-human-checks/UC-AMZ-HCHK-006-unavailable-listing-page-the-browser-does-not-open-sends-no-used-offer-request.md)
- [UC-AMZ-HCHK-007 — Amazon answer offered as a file download is not saved on the server's disk](amazon-human-checks/UC-AMZ-HCHK-007-amazon-answer-offered-as-download-is-not-saved.md)
- [UC-AMZ-HCHK-008 — Interrupted Amazon check leaves no downloaded answer on the server's disk](amazon-human-checks/UC-AMZ-HCHK-008-interrupted-amazon-check-leaves-no-downloaded-answer.md)
- [UC-AMZ-HCHK-009 — Failed browser cache clearing is logged without the reason it was started](amazon-human-checks/UC-AMZ-HCHK-009-failed-cache-clearing-logged-without-its-reason.md)

## Amazon browser profile recovery

- [UC-AMZ-BPROF-001 — Amazon check after the browser ended abruptly](amazon-browser-profile/UC-AMZ-BPROF-001-amazon-check-after-browser-ended-abruptly.md)
- [UC-AMZ-BPROF-002 — Amazon checks after the data directory is moved to another machine](amazon-browser-profile/UC-AMZ-BPROF-002-amazon-checks-after-data-directory-moved.md)

## Amazon browser retrieval

- [UC-AMZ-BRET-001 — Amazon check of an available listing returns its price](amazon-browser-retrieval/UC-AMZ-BRET-001-amazon-check-of-available-listing-returns-price.md)
- [UC-AMZ-BRET-002 — Understand a failed Amazon check from the server log](amazon-browser-retrieval/UC-AMZ-BRET-002-understand-failed-amazon-check-from-server-log.md)
- [UC-AMZ-BRET-003 — Install or update the server so that Amazon checks work](amazon-browser-retrieval/UC-AMZ-BRET-003-install-server-for-amazon-checks.md)
- [UC-AMZ-BRET-004 — Restart an Amazon item stopped by the retrieval errors](amazon-browser-retrieval/UC-AMZ-BRET-004-restart-amazon-item-stopped-by-retrieval-errors.md)
- [UC-AMZ-BRET-005 — Use Google Chrome when a Chromium snap is installed](amazon-browser-retrieval/UC-AMZ-BRET-005-use-google-chrome-when-chromium-snap-is-installed.md)

## Amazon thumbnail URLs

- [UC-AMZ-THUMB-001 — Add an Amazon item whose page offers an Amazon image over HTTPS](amazon-thumbnail-urls/UC-AMZ-THUMB-001-add-amazon-item-with-amazon-image.md)
- [UC-AMZ-THUMB-002 — Add an Amazon item whose page offers no accepted Amazon image](amazon-thumbnail-urls/UC-AMZ-THUMB-002-add-amazon-item-without-accepted-image.md)
- [UC-AMZ-THUMB-003 — Re-check an Amazon item that already has a thumbnail](amazon-thumbnail-urls/UC-AMZ-THUMB-003-recheck-amazon-item-with-stored-thumbnail.md)

## Amazon item AI review and purchase goal

- [UC-AMZ-ITEM-AIR-001 — Add an Amazon item with a purchase goal](amazon-item-ai-review/UC-AMZ-ITEM-AIR-001-add-amazon-item-with-purchase-goal.md)
- [UC-AMZ-ITEM-AIR-002 — Edit the purchase goal of an Amazon item](amazon-item-ai-review/UC-AMZ-ITEM-AIR-002-edit-amazon-item-purchase-goal.md)
- [UC-AMZ-ITEM-AIR-003 — Request an AI review of an Amazon item](amazon-item-ai-review/UC-AMZ-ITEM-AIR-003-request-ai-review-of-amazon-item.md)
- [UC-AMZ-ITEM-AIR-004 — An AI review of an Amazon item fails](amazon-item-ai-review/UC-AMZ-ITEM-AIR-004-amazon-ai-review-fails.md)
- [UC-AMZ-ITEM-AIR-005 — Open an Amazon item when no Claude token is saved](amazon-item-ai-review/UC-AMZ-ITEM-AIR-005-amazon-item-without-claude-token.md)
- [UC-AMZ-ITEM-AIR-006 — See the AI review of a stopped Amazon item](amazon-item-ai-review/UC-AMZ-ITEM-AIR-006-ai-review-of-stopped-amazon-item.md)
- [UC-AMZ-ITEM-AIR-007 — See the AI review of an Amazon item without a current price](amazon-item-ai-review/UC-AMZ-ITEM-AIR-007-ai-review-of-amazon-item-without-current-price.md)
- [UC-AMZ-ITEM-AIR-008 — An Amazon check records the customer rating](amazon-item-ai-review/UC-AMZ-ITEM-AIR-008-amazon-check-records-customer-rating.md)
- [UC-AMZ-ITEM-AIR-009 — Get the first AI review of an Amazon item tracked before the upgrade](amazon-item-ai-review/UC-AMZ-ITEM-AIR-009-first-ai-review-of-amazon-item-tracked-before-upgrade.md)
- [UC-AMZ-ITEM-AIR-010 — Read an Amazon review after the price changed](amazon-item-ai-review/UC-AMZ-ITEM-AIR-010-review-stays-after-price-change.md)
- [UC-AMZ-ITEM-AIR-011 — Delete a reviewed Amazon item](amazon-item-ai-review/UC-AMZ-ITEM-AIR-011-delete-reviewed-amazon-item.md)
