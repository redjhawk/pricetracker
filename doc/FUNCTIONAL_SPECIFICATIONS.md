# Functional Specifications — Price Tracking Service

## 1. Purpose

The service tracks the price of an item listed on a European Amazon marketplace or LeBoncoin France. The operator adds an item by providing its listing URL. The server checks its price twice per day, stores observations, and serves the information to a web interface. LeBoncoin items additionally receive an AI (Claude) review, guided by an optional purchase goal, and the operator manages the LeBoncoin session and Claude token from a Settings modal.

This document defines the first version (v1) and summarizes the features added since; each added feature is specified in detail in its own subject file (see [Subject specifications](#15-subject-specifications)), which prevails for details. Other marketplaces and possible enhancements are listed under [Next steps](#12-next-steps). Remaining decisions are listed under [Open decisions](#13-open-decisions).

Related documents: [API contract](../API_SPECIFICATION.md) · [Use case index](use-cases/README.md).

## 2. V1 goals and scope

- Add a supported European Amazon or LeBoncoin listing by URL.
- Check each active item twice per day and store detected prices with timestamps.
- Show all tracked items and their latest known prices in one compact Prices column, with up to three recent price periods that combine consecutive checks at the same price.
- Show the latest second-hand offer sold by Amazon, when available, with its condition and recent detections.
- Allow the operator to manually refresh the prices for all tracked items.
- Allow the operator to manually refresh an individual item from its details page.
- Allow the operator to inspect the complete price history for a tracked item and delete an item.
- Keep collection attempts running when a price is stale or a check fails.
- Show tracked items in separate Amazon and LeBoncoin tabs.
- Let the operator save a LeBoncoin session and a Claude token from a Settings modal opened from the header menu.
- Produce, store, and show AI reviews of LeBoncoin items, taking an optional per-item purchase goal into account.

V1 supports euro-priced listings from Amazon Germany, France, Spain, Italy, the Netherlands, and Belgium, plus LeBoncoin France. The expected maximum is approximately 100 tracked items. While no regular user exists (open mode), there is one server-side collection of items and no login is required. Once the administrator creates a regular user (protected mode), login is required and each user has their own items and settings (see [5.7](#57-users-and-login)). The server owns item storage and price collection. The front end displays data supplied by the server and submits add, delete, and refresh actions.

Prices are the item price only, in euros. Shipping, taxes, and other charges are excluded. V1 assumes one price per listing; price ranges are not represented. Notifications and price trends are out of scope. Multiple local users with login are implemented as phase 1 of [multi-user login](specifications/multi-user-login/functional.md); item sharing and in-app password changes are later phases.

## 3. Core concepts

**Tracked item:** A saved reference to an eligible Amazon or LeBoncoin listing, including its platform, URL, display information, tracking status, and observations.

**Price observation:** A successful detection containing the item price in euros and the detection timestamp. Collection failures are recorded separately and do not overwrite the last successful price.

**Latest price:** The value and timestamp from the latest successful observation. It may be stale if later checks have failed or been delayed.

**LeBoncoin session:** The value of LeBoncoin's `datadome` cookie, obtained by the operator on a desktop browser and saved in Settings. LeBoncoin collection attempts send it when saved.

**Claude token:** A Claude Pro/Max subscription token produced by `claude setup-token`, saved in Settings and used for AI reviews.

**AI review:** A structured English assessment of a LeBoncoin item produced by Claude from the listing information, price history, and purchase goal.

**Purchase goal:** Optional free text stating what the operator intends a LeBoncoin item for.

**Price period:** A run of consecutive successful observations with the same price. The main list shows each recent period once, using the timestamp of that price's latest successful observation. A new amount starts a new period, including when the amount returns to an earlier value after changing.

## 4. Technical architecture constraints

- The application has a separate front end and backend.
- The backend must run on Go and is responsible for the service API, scheduled price collection, and persistence.
- Use SQLite for server-side storage of tracked items, price observations, and collection outcomes as needed.
- The front end must use React 19 and IBM Carbon Design System components and visual conventions.
- The front end reads and displays data from the backend; it does not connect directly to SQLite or collect marketplace prices.
- Keep collection logic in separate platform adapters for Amazon and LeBoncoin.
- Use an API-first workflow for cross-tier features: define the API contract and obtain operator confirmation before implementing either backend or frontend changes. After confirmation, implement both tiers against the approved contract.
- Provide a development mode that starts the backend and serves the frontend together, so a developer can open the application and use its pages and backend API without manually starting each tier.
- Development mode must provide repeatable sample data in a development SQLite database so the interface can be reviewed without relying on live marketplace requests. Keep development seed data isolated from any non-development database.
- Development mode must allow the developer to navigate directly to the required frontend page (including a detail page where applicable) and have that page request and display data from the backend.

The API shape, Go HTTP implementation, database access library, scheduling implementation, and deployment arrangement are not prescribed here and can be selected during technical design. An API contract for a cross-tier change should cover endpoints and methods, request/response data, validation/errors, and any client-visible states.

## 5. Main user journeys

### 5.1 Add an item

1. The operator opens the tracked-items page and chooses **Add item**.
2. The operator pastes a supported Amazon European or LeBoncoin France listing URL and submits it.
3. The server validates the URL and confirms that it is an eligible listing.
4. The server attempts to identify the item and immediately fetch its price; the operator should not have to wait for the next scheduled check for the first attempt.
5. The item appears in the list if accepted. Its initial price may be pending if the immediate retrieval is still in progress or did not succeed.
6. The interface explains invalid, unsupported, duplicate, or unprocessable URLs.

### 5.2 Review tracked items

The main page shows tracked items in two tabs, **Amazon** and **LeBoncoin** (Amazon selected initially). Each tab lists only that platform's items; the chosen tab is kept while the application runs, including after opening item details and returning. Each row includes:

- Item title, or a fallback label if unavailable, with the marketplace and listing ID shown below it (no separate Marketplace column).
- Source platform and listing link.
- Thumbnail when available. Amazon thumbnails come only from Amazon image hosts over HTTPS ([amazon-thumbnail-urls](specifications/amazon-thumbnail-urls/functional.md)).
- One compact Prices column containing up to three recent consecutive price periods, newest first. The latest price appears once as the first period; each amount is paired with the time of its latest successful observation. Repeated checks at the same price update that period's time; a changed price starts a new period.
- An explicit unavailable or awaiting-first-price state when no successful price is available.
- Collection/listing status, such as active, stale, retrieval issue, or unavailable.
- Latest second-hand offer sold by Amazon for Amazon items, including price and condition; third-party offers are ignored. This column appears only in the Amazon tab.
- Actions to open item details, delete an item, or refresh all tracked items.

The page distinguishes no tracked items, no items for the selected platform, and no search matches, and has clear loading and error states. The tracked-item count and the refresh action cover all platforms, and the refresh's all-items scope is made clear. A stale price does not stop scheduled checks or retries. The stale threshold is an open decision.

### 5.3 View item details

The detail view shows item metadata, platform, listing URL, latest known price and timestamp, current status, and every successful price detection for as long as the item has been tracked, newest first. Amazon items also show every successful Amazon-sold second-hand offer detection with its condition. The operator can scroll through the full lists and request an immediate price refresh from this page; the interface displays progress and then reloads the server result. V1 does not require a chart, trend calculation, or arbitrary date-range selection.

### 5.4 Delete an item

The operator can delete an item from tracking. Deletion removes the item, its associated price observations, and its AI reviews from the server database. No history of deleted items is retained. V1 does not require pause or archive actions.

### 5.5 Manage settings

A profile-style icon at the right end of the header, on every page, opens a menu with **Settings** and, when logged in, **Log out** (the administrator's menu has only **Log out**). Settings opens a modal that loads the current settings from the server and contains two entries:

- **LeBonCoin session:** shows whether a session is saved and a masked hint (its last 4 characters), with an empty input to replace it. The operator can paste a raw `datadome` value or a cookie string from which the value is extracted; unusable input is rejected with a message. A warning shows when the latest attempt with the saved session was rejected or failed, or when LeBoncoin expired or revoked it. If the operator replaces or removes the session after the stored session changed since the modal loaded, saving is refused and the operator is asked to reload.
- **Claude token:** shows whether a token is saved and a masked hint (its last 4 characters), with an empty input to replace it. A non-empty token is verified with Claude before it is stored; a refused or unverifiable token saves nothing. Its help text states that it is used for Amazon and LeBoncoin AI reviews. A warning shows when the latest AI review failed because Claude rejected the token.

One Save applies both entries; an error in either saves neither. An entry left empty, or containing only whitespace of any length (never rejected as too long, issue #286), keeps its saved secret exactly as it is (no re-verification, no session change); removing a secret is an explicit remove action applied by Save. Once saved, the session and token are never sent back by the server, not even to the modal (issue #265, [masked secrets](specifications/settings-secret-masking/functional.md)). Settings persist in the database across restarts and apply without restart. Details: [settings](specifications/leboncoin-session-settings/functional.md), [Claude token](specifications/claude-token-settings/functional.md), [header menu](specifications/app-header-menu/functional.md).

To obtain a session, the operator runs a desktop helper script with a LeBoncoin listing URL, completes any LeBoncoin verification personally in a dedicated visible browser, and copies the printed `datadome=<value>` line into Settings ([session capture](specifications/leboncoin-session-capture/functional.md)). No session file is used ([removal record](deprecated/leboncoin-session-file.md)).

### 5.6 Review a LeBoncoin item with AI

When a Claude token is saved, an AI review of a LeBoncoin item is requested automatically and asynchronously when the item is added, when a check records a price different from the previous one, and when its purchase goal changes. The LeBoncoin item details page shows an **AI review** section with the latest review, its date/time and reviewed price, a **Refresh AI review** button, and the previous reviews, newest first. A review contains a price rating (Good deal / Fair / Overpriced) and a condition rating (Excellent / Good / Fair / Poor) with explanations, an overall recommendation (Buy / Negotiate / Avoid) with explanation, an estimated fair price range and suggested offer, scam/risk signs, missing accessories or information and questions for the seller, and whether the description matches the photos. Without a token, no review is attempted, the refresh button is disabled, and the section asks the operator to configure one in Settings; saving a token later does not review existing items automatically. Failures are shown with their date/time while the latest successful review stays visible. Items added before this feature are not reviewed automatically. Details: [LeBoncoin AI review](specifications/leboncoin-ai-review/functional.md).

The add-item form offers an optional **Purchase goal** field, saved for LeBoncoin items only. The LeBoncoin details page shows the goal in an editable field with a save action. A non-empty goal is sent with every review request; an empty goal means no goal. Details: [purchase goal](specifications/leboncoin-purchase-goal/functional.md).

When a purchase goal is entered in the add-item form or on the details page of one of the operator's own items (LeBoncoin or Amazon), a drop-down offers "No goal", the distinct goals currently set on the operator's own items (alphabetical; not goals of items shared by others), and "New goal", which shows an empty free-text field. Options show the first four words of each goal. Selecting an existing goal shows its full text in an editable field; the text is copied to the item, so modifying it never changes other items. A goal leaves the list once no item uses it. Without existing goals, the free-text field is shown as before (issue #304). Details: [purchase goal choice](specifications/purchase-goal-choice/functional.md).

### 5.6a Amazon item AI review and purchase goal (specified, issue #305)

Amazon items get the purchase goal (add-item form and an editable field on the details page) and an **AI review** section on their details page, with history, a **Refresh AI review** button, and the same no-token, pending and error states as LeBoncoin. Reviews are produced only by **Refresh AI review**: never automatically, including for Amazon items added before this feature. Each review shows the price it was based on and contains a price rating (Good deal / Fair / Overpriced) with explanation, a fair price range, and an overall recommendation (Buy / Wait for a lower price / Avoid) with explanation. The AI receives the title, marketplace, current price and price history, the Amazon-sold second-hand offer and its history, the customer rating and number of ratings, and the goal. A review sends no Amazon request: it uses the data recorded by the item's checks, which now also record the customer rating and number of ratings. Review failures do not count toward the Amazon failure stop. While the item is stopped, or when it has no current price (none recorded yet, or latest answer "price not found" or "listing unavailable"), the **Refresh AI review** button is not shown and the section explains why; earlier reviews stay visible. Details: [Amazon item AI review](specifications/amazon-item-ai-review/functional.md).

### 5.6b Amazon failure stop

The Amazon searches feature (issue #56) was removed in issue #111: no searches tab, saved searches and search-only items with their history and Amazon AI reviews are deleted, and tracked Amazon items are unchanged. Details: [Amazon searches removal](specifications/amazon-searches-removal/functional.md).

Failed Amazon requests are logged with the server's response. After 5 consecutive failures every tracked Amazon item is stopped, each with its own stopped state: scheduled checks and **Refresh all prices** skip it and the Amazon tab shows a warning. A successful **Refresh price** on an item restarts only that item. Details: [Amazon request stop](specifications/amazon-request-stop/functional.md).

To look less like a bot (issue #168), Amazon pages are requested through a real web browser, and each tracked Amazon item is checked once at a random time between 06:00 and 09:00 and once between 19:00 and 23:00, drawn anew every day and independently for each item; manual refreshes stay immediate. LeBoncoin is unchanged. Amazon items stopped before this change are restarted once automatically. Details: [Amazon human-like checks](specifications/amazon-human-checks/functional.md).

A success response that the browser does not open as a page (file download, no content) is an Amazon failure counted toward the stop, not "price not found" (issue #186). Details: [FR-AMZ-HCHK-011–012](specifications/amazon-human-checks/functional.md).

The Amazon browser refuses file downloads: an Amazon answer offered as a download is never saved on the server's disk, neither in the browser profile nor in the Downloads folder of the account running the server, and the check stays a counted failure (issue #217). This also holds when the check is interrupted by its time limit or by a server stop (issue #233). When the browser cache that may hold such an answer cannot be cleared, the server log says `Amazon browser cache could not be cleared: <cause>`, without claiming why the clearing was started (issue #240). Details: [FR-AMZ-HCHK-014–016](specifications/amazon-human-checks/functional.md).

When the browser cannot open a product page answered with 404 or 410 (for example an empty body), the listing is unavailable and no used-offer request is sent for that check; its second-hand result is a check error and the last offer is kept (issue #187). Details: [FR-AMZ-HCHK-013](specifications/amazon-human-checks/functional.md).

Lock marks left in the Amazon browser's persistent profile by a browser that ended abruptly, including marks naming another machine after the data directory was moved, no longer make Amazon checks fail; cookies are kept (issue #180). Details: [Amazon browser profile recovery](specifications/amazon-browser-profile/functional.md).

Amazon checks through the browser work again (issue #204): an available listing gives its price and second-hand offer as before the browser change, page content that merely mentions "captcha" or "robot" is not taken for a robot check, each failure cause is logged distinctly, and the installation documentation lists what the server needs for the browser. Stopped items are restarted by **Refresh price** as before. The documentation also tells the operator to set `PRICEFOLLOWER_BROWSER_PATH` to Google Chrome when a Chromium snap is installed, as is usual on Ubuntu (issue #226), and shows the failure log line with its final `response` field, the start of the page Amazon returned, which can be empty (issue #210). Details: [Amazon browser retrieval](specifications/amazon-browser-retrieval/functional.md).

### 5.7 Users and login

Phase 1 of [multi-user login](specifications/multi-user-login/functional.md), as defined in its [delivery plan](specifications/multi-user-login/delivery-plan.md), which prevails where it differs from the full specification.

- **Administrator:** exactly one global account, `admin`. A device command (`pricefollower admin-password`, also available through the deployment script) creates it or resets its password, generating and printing a random password; a reset ends the administrator's sessions. It cannot be created from the browser. The administrator's only screen is the user list (username and last login) with an action to add a user by typing a username and password. The administrator does not see or manage items or settings.
- **Open mode:** while no regular user exists, the application works without login, as before; a **Log in** link in the header lets the administrator log in. The `admin` account alone does not change this.
- **Protected mode:** starts when the first regular user is created. From then on only a login screen is shown to anyone not logged in; no item, setting, or user data is accessible without login. The first user inherits all existing items, the Claude token, and the LeBoncoin session.
- **Login:** username and password; wrong credentials give one generic error. After 5 consecutive failures for a username from one client address (Cloudflare's `CF-Connecting-IP`, else the connection address), that client's attempts for that username are refused for 1 minute without a password check; other clients and usernames are unaffected. When too many password checks are already waiting, a login is answered at once with a busy message (issue #266, [login attempt limits](specifications/login-attempt-limits/functional.md)). A session lasts 30 days from login, or until **Log out** in the header menu. The session cookie is `Secure` when the request arrives over HTTPS, directly or through the reverse proxy (`X-Forwarded-Proto: https`), and stays usable over plain HTTP in development (issue #268, [session cookie security](specifications/session-cookie-security/functional.md)).
- **Usernames and passwords:** usernames are 3–32 letters, digits, dot, dash, or underscore, unique and case-insensitive. Passwords have at least 12 characters. Nobody changes a password in the application in phase 1.
- **Ownership:** each item belongs to the user who added it; users see and manage only their own items. Two users adding the same URL get independent items; the same user adding a URL twice is a duplicate as before. Each user has their own Claude token and LeBoncoin session, used for their items' collections and AI reviews.

Not yet implemented (later phases): item sharing and the "Shared by others" list, password changes and forced change at first login, administrator password reset of users, renaming and removing users.

## 6. Price collection behavior

- Each active item is scheduled for two collection attempts per day.
- As soon as a new item is accepted, trigger an immediate price collection attempt; then continue with its regular twice-daily schedule.
- Preserve the submitted Amazon listing URL and use it for collection requests; normalize the Amazon product identity separately for duplicate detection.
- The schedule defaults to 08:00 and 20:00 in `Europe/Paris`; timezone and check times are configurable for deployment. Amazon scheduled checks instead happen at random times, drawn daily per item, within 06:00–09:00 and 19:00–23:00 ([FR-AMZ-HCHK-004](specifications/amazon-human-checks/functional.md)).
- Each successful check stores the detected item price in euros and its timestamp.
- Amazon checks also look for second-hand offers sold by Amazon. If multiple qualifying offers exist, store the lowest-priced offer and its condition. Ignore third-party offers; do not accept an offer unless Amazon's seller attribution can be verified. LeBoncoin items have no separate second-hand-offer field.
- A LeBoncoin free/donation listing with no numeric price is recorded as a zero-euro price only when the listing explicitly indicates that it is a donation/free item. The interface displays this as “Gratuit”. A missing or unparseable price is not treated as free.
- Record the second-hand check outcome as available, no qualifying offer found, pending, or check error. Preserve the last detected Amazon offer when a later check finds none or cannot verify the result.
- Each failed check records a useful outcome, such as temporary retrieval error, listing unavailable, or price not found.
- A failed check never replaces the latest successful price with a blank, zero, or inferred value.
- Continue scheduled retries when an item is stale or previous collection attempts failed.
- A manual refresh from an item's details page queues an immediate check for that item only. An in-progress check is not duplicated.
- Collection requests should present as access from a Google Chrome browser, as specified by the product owner. A challenge or other non-success response is a retrieval error, not a free price or proof that the listing is unavailable.
- When a LeBoncoin session is saved, LeBoncoin collection attempts send only that `datadome` cookie, and only to LeBoncoin's own hosts; without one, collection is sessionless. A session renewed by LeBoncoin on a verified response replaces the saved value; an expired or revoked session is no longer sent. The outcome of the latest session-assisted attempt is recorded for the Settings warning ([session collection](specifications/leboncoin-session-collection/functional.md)).
- When a LeBoncoin item is added and its immediate collection finds an old price on the listing, that old price is recorded once as a historical price observation, with its date when the listing provides one, otherwise shown as “Old price”. Later collections never read it ([old price](specifications/leboncoin-old-price/functional.md)).
- Store every successful price check, including unchanged prices, and retain every observation while the item remains tracked.
- The server should expose when the next check is expected if useful to the interface.

## 7. Pages and interface areas

### Tracked items page

The primary page contains the add-item action, a search field, and Amazon and LeBoncoin tabs with the tracked-items tables. Search filters the selected tab locally by title, listing ID, ASIN, platform, or marketplace, and the query is kept across tab switches. Each row presents the latest price and recent price periods together in the Prices column. No sign-in is required.

### Item details page

Shows item metadata, recent price detections, collection status, source listing, refresh action, and deletion action. LeBoncoin items also show the purchase goal and the AI review section; Amazon items will too once issue #305 is implemented ([5.6a](#56a-amazon-item-ai-review-and-purchase-goal-specified-issue-305)).

### Header menu and Settings modal

A profile-style icon in the header of every page opens a menu whose **Settings** entry opens the Settings modal (LeBoncoin session and Claude token); a logged-in user also gets **Log out**. The administrator's menu has only **Log out**. In open mode a **Log in** link appears in the header.

### Login page

Username and password form, shown to anyone not logged in in protected mode, and at `/login` in open mode.

### Administrator page

The administrator's only screen: the list of users (username, last login) and an action to add a user.

### Add-item flow

A URL input and an optional multi-line Purchase goal field, with validation feedback, submission progress, and a clear success or error result. Its specific layout is not prescribed.

### Server-backed data

The server is the source of truth for tracked items and price observations. The front end retrieves and displays server data and submits add/delete/refresh actions. In protected mode, items, the LeBoncoin session, and the Claude token belong to a user and are only accessible to that user.

## 8. Functional requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-01 | The operator can submit an eligible European Amazon or French LeBoncoin listing URL to add an item. | Must |
| FR-02 | The server validates URL format and eligibility for v1. | Must |
| FR-03 | The server stores accepted items and associates price observations with them. | Must |
| FR-04 | Active items receive two scheduled price collection attempts per day. | Must |
| FR-05 | The list shows the latest successful item price and up to three recent consecutive price periods together in one Prices column, newest first, with each period's latest observation time. Repeated checks at the same price update that period's time; a different price starts a new period. | Must |
| FR-06 | The detail view shows all successful price observations for the full tracking period. | Must |
| FR-07 | Collection failures are recorded without corrupting the last successful price, and retries continue. | Must |
| FR-08 | The operator can open the source listing. | Should |
| FR-09 | The operator can delete an item and its associated price history. | Must |
| FR-10 | The front end displays server-provided data; storage and collection are server-side. | Must |
| FR-11 | The service supports approximately 100 tracked items in v1. | Should |
| FR-12 | Development mode starts the backend and serves the frontend together, and includes repeatable sample data in an isolated development database. | Must |
| FR-13 | Development mode supports direct navigation to frontend pages and successful data requests from those pages to the backend. | Must |
| FR-14 | Adding an accepted item triggers an immediate price collection attempt, followed by the regular twice-daily checks. | Must |
| FR-15 | Amazon item collections check for second-hand offers and only store offers explicitly sold by Amazon; third-party offers are excluded. This does not apply to LeBoncoin items. | Must |
| FR-16 | The main list shows the latest Amazon-sold second-hand offer price and condition, or a clear pending/not-found/error state. | Must |
| FR-17 | The operator can request a check of all tracked items from the main page; the API accepts asynchronously and the interface shows progress. | Must |
| FR-18 | The operator can request an immediate check of one item from its details page; the API accepts asynchronously and the interface shows progress. | Must |
| FR-19 | An explicitly donated/free LeBoncoin listing with no numeric price is recorded as zero euros and displayed as “Gratuit”; missing prices are not assumed to be free. | Must |
| FR-20 | The item details view shows every successful item-price observation for the full tracking period; Amazon details also show every successful Amazon-sold second-hand offer observation. | Must |
| FR-21 | The tracked-items list is split into Amazon and LeBoncoin tabs; search applies to the selected tab; the count and refresh cover all items ([FR-PLATFORM-TABS-*](specifications/platform-tabs/functional.md)). | Must |
| FR-22 | The Item column shows the marketplace and listing ID below the title; there is no separate Marketplace column ([FR-TI-IDENTITY-*](specifications/tracked-items-identity/functional.md)). | Must |
| FR-23 | A header icon menu on every page opens the Settings modal; Escape closes the menu even when pressed at once after opening (issue #215) ([FR-APP-MENU-*](specifications/app-header-menu/functional.md)). | Must |
| FR-24 | The operator can view, set, replace, and clear the LeBoncoin session in Settings; it is stored in the database ([FR-LBC-SET-*](specifications/leboncoin-session-settings/functional.md)). | Must |
| FR-25 | LeBoncoin collection uses, renews, and stops using the saved session as specified, and records session-assisted outcomes ([FR-LBC-COL-*](specifications/leboncoin-session-collection/functional.md)). | Must |
| FR-26 | A desktop helper captures a verified LeBoncoin session and prints it for pasting into Settings ([FR-LBC-CAP-*](specifications/leboncoin-session-capture/functional.md)); no session file is used ([FR-LBC-RM-*](specifications/leboncoin-session-file-removal/functional.md)). | Must |
| FR-27 | The operator can save, verify, replace, and remove a Claude token in Settings ([FR-CLT-SET-*](specifications/claude-token-settings/functional.md)). | Must |
| FR-28 | LeBoncoin items receive AI reviews on add, price change, goal change, and manual refresh, shown with history on the details page ([FR-LBC-AIR-*](specifications/leboncoin-ai-review/functional.md)). | Must |
| FR-29 | The operator can set an optional purchase goal for a LeBoncoin item when adding it and on its details page; it is sent with every review request ([FR-PURCHASE-GOAL-*](specifications/leboncoin-purchase-goal/functional.md)). | Must |
| FR-30 | Local users and login, phase 1: admin device command, open and protected modes, login/logout with lockout and 30-day sessions, administrator user list and user creation, per-user items and settings, inheritance by the first user ([FR-MODE-*, FR-AUTH-*, FR-ADMIN-*, FR-SHARE-001/005/006, FR-SETTINGS-001](specifications/multi-user-login/delivery-plan.md)). | Must |
| FR-32 | Removed (issue #111): no Amazon searches; search-only items and Amazon AI reviews are deleted ([FR-AMZ-SRCH-DEL-*](specifications/amazon-searches-removal/functional.md)). | Must |
| FR-33 | After 5 consecutive Amazon failures each tracked Amazon item is stopped, skipped by scheduled and all-item refresh, warned in the Amazon tab, and restarted individually by a successful Refresh price ([FR-AMZ-STOP-*](specifications/amazon-request-stop/functional.md)). | Must |
| FR-34 | Superseded by FR-32 (issue #111). | — |
| FR-35 | An **Information** entry in the header menu (regular user, administrator, open mode) opens a modal showing the logged-in username (not in open mode) and the running release version, or the latest tag + `-dev` in development; Tab and Shift+Tab never move focus out of the modal, at any typing speed (issue #215) ([FR-APP-INFO-*](specifications/app-information/functional.md)). | Must |
| FR-36 | Every screen and modal is usable on a phone (360 px reference, 320 px minimum) without horizontal page scrolling and with all desktop information and actions; below 1056 px (phone and tablet) every page drops its lateral margins (at most 16 px between screen edge and content), lists shown as cards (tracked items, administrator users) span the full screen width, and each card places its values side by side as far as they fit (issue #165); desktop layout and modals unchanged ([FR-RESPONSIVE-*](specifications/responsive-layout/functional.md)). | Must |
| FR-37 | Amazon checks are made through a real web browser, and each Amazon item is checked at a random time, drawn daily and per item, within 06:00–09:00 and within 19:00–23:00; manual refreshes stay immediate; items stopped before the change are restarted once; a success response the browser does not open as a page is a failure (issue #186); a 404/410 product page the browser does not open sends no used-offer request (issue #187); a response offered as a file download is never saved on the server's disk, even when the check is interrupted (issues #217, #233) ([FR-AMZ-HCHK-*](specifications/amazon-human-checks/functional.md)). | Must |
| FR-38 | An Amazon check starts its browser normally, keeping the profile's cookies, when the persistent browser profile holds lock marks left by an earlier browser that ended abruptly, on this machine or another one ([FR-AMZ-BPROF-*](specifications/amazon-browser-profile/functional.md)). | Must |
| FR-39 | On a documented installation, Amazon checks of available listings succeed with the same price and offer results as before the browser change; mentions of "captcha" or "robot" in a product page are not a robot check; failure causes are logged distinctly; the installation documentation lists the server's browser needs, including setting `PRICEFOLLOWER_BROWSER_PATH` to Google Chrome when a Chromium snap is installed (issue #226) and the `response` field of the failure log line (issue #210) ([FR-AMZ-BRET-*](specifications/amazon-browser-retrieval/functional.md)). | Must |
| FR-40 | Failed logins are counted per client address (`CF-Connecting-IP` when valid, else the connection) and username: 5 consecutive failures refuse only that pair for 1 minute, without a password check; a login is answered 429 at once when the maximum number of password checks are already waiting; failure memory stays bounded (issue #266) ([FR-LOGIN-LIM-*](specifications/login-attempt-limits/functional.md)). | Must |
| FR-41 | Once saved, the Claude token and LeBoncoin session are never sent back by the server; Settings shows whether each is saved with a masked hint (last 4 characters) and an empty input; an empty or whitespace-only input, of any length, keeps the secret unchanged (issue #286), removal is an explicit action, and the session-changed check applies only when the session is replaced or removed (issue #265) ([FR-SECRET-*](specifications/settings-secret-masking/functional.md)). | Must |
| FR-42 | The session cookie set at login and cleared at logout is `Secure` when the request arrives over TLS or carries `X-Forwarded-Proto: https` (first entry, case-insensitive), and not `Secure` otherwise, so plain HTTP login keeps working in development; other cookie attributes are unchanged (issue #268) ([FR-COOKIE-*](specifications/session-cookie-security/functional.md)). | Must |
| FR-43 | Specified (issue #305): Amazon items have a purchase goal and an AI review section with history and manual refresh only (no automatic review) on their details page; reviews use recorded check data (checks also record the customer rating) and are unavailable while the item is stopped or has no current price; the Claude token help text mentions Amazon and LeBoncoin reviews ([FR-AMZ-ITEM-AIR-*](specifications/amazon-item-ai-review/functional.md), FR-CLT-SET-014). | Must |
| FR-31 | When a LeBoncoin item is added, a listed old price is recorded once in its price history, dated when possible, otherwise labelled “Old price”; no new column ([FR-LBC-OLD-PRICE-*](specifications/leboncoin-old-price/functional.md)). | Must |

## 9. Important states and edge cases

- URL is malformed or does not belong to a supported Amazon or LeBoncoin listing.
- URL is valid but the listing cannot be retrieved or parsed.
- Item is added but the initial price is pending.
- The immediate collection attempt after adding an item fails; the item remains tracked and scheduled retries continue.
- The listing is already being tracked.
- The source listing is removed, sold, expired, or made private.
- A price is missing, unchanged, promotional, or ambiguous.
- Listing content changes, redirects, or points to a different item.
- A scheduled check is delayed or fails.
- The latest successful price is stale, but retries continue.
- Amazon blocks or limits a collection request.
- LeBoncoin blocks a request with a JavaScript challenge or returns a non-success response; record a retrieval error and continue retries.
- A LeBoncoin listing has no numeric price but explicitly indicates a donation/free item.
- A LeBoncoin listing has no price and no explicit donation/free text; report price not found, not zero.
- A new LeBoncoin listing shows an old price with or without a date, or its first collection fails so no old price is recorded.

- No Claude token is saved, or Claude rejects the token or its usage limit is reached during a review.
- The saved LeBoncoin session is rejected, expired, revoked, or changed by renewal while the Settings modal is open.

The interface should preserve and clearly label the latest successful observation when later checks fail.

## 10. Non-goals for v1

- Item sharing between users, self-registration, email, external identity providers, and multi-factor authentication.
- AI reviews of Amazon items (being added by issue #305, see [5.6a](#56a-amazon-item-ai-review-and-purchase-goal-specified-issue-305)).
- Marketplaces other than European Amazon and LeBoncoin France (Temu, Vinted, and Wallapop remain future work).
- Price trends, percentage changes, or charts.
- Notifications or target-price alerts.
- Shipping/tax-inclusive prices, currency conversion, or price ranges.
- Retaining records after an item is deleted.
- Export of data or per-user privacy controls.

## 11. Data retention and capacity

Keep price observations for as long as the item remains tracked, including several months of observations. Deleting an item deletes its history. The initial expected tracked-item capacity is around 100; a larger limit may be introduced in a later version.

## 12. Next steps

Potential later marketplace support:

- Temu
- Vinted
- Wallapop

Each platform will need its own supported-region rules, collection behavior, and handling of listing-specific states. Potential later product enhancements include trends, charts, notifications, additional currencies, and a higher item limit.

## 13. Open decisions

1. **Amazon coverage:** Are the currently supported European Amazon country domains and listing forms sufficient?
2. **Schedule:** The implementation defaults to 08:00 and 20:00 in `Europe/Paris`; should the production deployment use different configurable check times or a different timezone?
3. **Price parsing:** How should promotional, coupon, or multi-option prices be interpreted while only storing the item price?
4. **Staleness:** After how long without a successful check should a price be marked stale?
5. **Item limit:** Should the approximately 100-item limit be enforced, and what should happen when it is reached?
6. **Failures:** How should repeated failures and unavailable listings be surfaced? Should retries use a defined backoff?
7. **Presentation:** Which language, locale formatting, and accessibility target should the interface use?

## 14. Initial acceptance criteria

- The operator can submit an eligible European Amazon or French LeBoncoin URL and receive a clear result.
- An accepted item appears in the list, including when its initial price is pending.
- A successful check creates a timestamped euro price observation.
- A qualifying Amazon-sold second-hand offer is recorded with its timestamp and condition; third-party offers are never shown as Amazon offers.
- The main list distinguishes a current Amazon-sold offer, no qualifying offer, a pending check, and an uncertain/failed check.
- The operator can manually refresh all tracked items and see pending progress until the checks finish.
- The operator can refresh an individual item from its details page and see pending progress until the check finishes.
- The operator can track an eligible LeBoncoin France ad, including the example priced item and the explicit free/donation item.
- A free/donation observation is stored as zero euros and shown as “Gratuit”; an absent or unclear price is not shown as free.
- Adding an accepted item immediately triggers its first collection attempt without waiting for the twice-daily schedule.
- Active items have two collection attempts per day, and retries continue when checks fail or prices become stale.
- The list shows the latest successful price once, together with up to three recent consecutive price periods in one Prices column, or a clear unavailable state.
- The item details view displays all successful item-price detections, and all successful Amazon-sold second-hand detections when applicable.
- Failed checks do not erase or misrepresent the last successful price.
- Deleting an item removes its stored price history.
- In open mode no login is required; in protected mode only the login screen is reachable until login, and each user sees only their own items and settings. The server stores data and collects prices, while the front end displays server-provided information.
- The backend runs on Go and persists data in SQLite.
- The separate front end uses React 19 and IBM Carbon Design System.
- A developer can start one development mode, open the frontend directly at the tracked-items or item-details page, and see sample data returned by the backend.
- Development seed data is repeatable and cannot overwrite the non-development database.
- The list shows Amazon and LeBoncoin tabs, with the marketplace and listing ID under each title.
- The operator can open Settings from the header menu, save, replace, and clear the LeBoncoin session and the Claude token, and see warnings when they were rejected.
- A LeBoncoin item with a saved token receives an AI review automatically, can be re-reviewed manually, and shows its review history; the purchase goal is used in reviews.

## 15. Subject specifications

Detailed, current requirements for features added after the initial v1 text. Where a subject file is more detailed, it applies.

| Subject | Functional specification | Status |
|---|---|---|
| Item refresh | [item-refresh](specifications/item-refresh/functional.md) | implemented |
| Platform tabs | [platform-tabs](specifications/platform-tabs/functional.md) | implemented |
| Tracked item identity | [tracked-items-identity](specifications/tracked-items-identity/functional.md) | implemented |
| Header menu | [app-header-menu](specifications/app-header-menu/functional.md) | implemented; FR-APP-MENU-007 specified (issue #215) |
| Application information | [app-information](specifications/app-information/functional.md) | specified (issue #115); FR-APP-INFO-014 specified (issue #215) |
| LeBoncoin session settings | [leboncoin-session-settings](specifications/leboncoin-session-settings/functional.md) | implemented; amended by FR-SECRET-* (issue #265) |
| LeBoncoin session collection | [leboncoin-session-collection](specifications/leboncoin-session-collection/functional.md) | implemented |
| LeBoncoin session capture | [leboncoin-session-capture](specifications/leboncoin-session-capture/functional.md) | implemented |
| Session file removal | [leboncoin-session-file-removal](specifications/leboncoin-session-file-removal/functional.md) | implemented |
| Claude token settings | [claude-token-settings](specifications/claude-token-settings/functional.md) | implemented; FR-CLT-SET-013 specified (issue #111); amended by FR-SECRET-* (issue #265); FR-CLT-SET-014 specified (issue #305) |
| Masked secrets in Settings | [settings-secret-masking](specifications/settings-secret-masking/functional.md) | specified (issue #265); FR-SECRET-014 specified (issue #286) |
| LeBoncoin AI review | [leboncoin-ai-review](specifications/leboncoin-ai-review/functional.md) | implemented |
| LeBoncoin purchase goal | [leboncoin-purchase-goal](specifications/leboncoin-purchase-goal/functional.md) | implemented |
| Purchase goal choice | [purchase-goal-choice](specifications/purchase-goal-choice/functional.md) | specified (issue #304) |
| Multiple users and login | [multi-user-login](specifications/multi-user-login/functional.md), [delivery plan](specifications/multi-user-login/delivery-plan.md) | phase 1 implemented; later phases planned; FR-AUTH-005 superseded by FR-LOGIN-LIM-* (issue #266) |
| Login attempt limits | [login-attempt-limits](specifications/login-attempt-limits/functional.md) | specified (issue #266) |
| Session cookie security | [session-cookie-security](specifications/session-cookie-security/functional.md) | FR-COOKIE-001–006 implemented (issue #268); FR-COOKIE-007–009 specified (issue #293, maintenance: the `clientAddress` doc comment in `internal/httpapi/auth.go` is moved back directly above its function, leaving `secureRequest` with only its own comment; no behaviour change) |
| LeBoncoin old price | [leboncoin-old-price](specifications/leboncoin-old-price/functional.md) | implemented |
| Amazon searches | [amazon-searches](specifications/amazon-searches/functional.md) | superseded (issue #111) |
| Amazon human browsing | [amazon-human-browsing](specifications/amazon-human-browsing/functional.md) | superseded (issue #111) |
| Amazon AI review | [amazon-ai-review](specifications/amazon-ai-review/functional.md) | superseded (issue #111) |
| Amazon item AI review and purchase goal | [amazon-item-ai-review](specifications/amazon-item-ai-review/functional.md) | specified (issue #305) |
| Amazon searches removal | [amazon-searches-removal](specifications/amazon-searches-removal/functional.md) | specified (issue #111) |
| Amazon request stop | [amazon-request-stop](specifications/amazon-request-stop/functional.md) | specified (issue #111) |
| Amazon human-like checks | [amazon-human-checks](specifications/amazon-human-checks/functional.md) | specified (issue #168); FR-AMZ-HCHK-011–012 specified (issue #186); FR-AMZ-HCHK-013 specified (issue #187); FR-AMZ-HCHK-014 specified (issue #217); FR-AMZ-HCHK-015 specified (issue #233); FR-AMZ-HCHK-016 specified (issue #240, log wording: the failed cache-clear line no longer says `after a page not opened`) |
| Amazon browser profile recovery | [amazon-browser-profile](specifications/amazon-browser-profile/functional.md) | specified (issue #180); FR-AMZ-BPROF-008–010 specified (issue #197, maintenance: when the persistent profile's stale lock cannot be discarded, the Amazon browser's profile setup returns an empty path and no cleanup with the error, like its other failure branch; no behaviour change) |
| Amazon browser retrieval | [amazon-browser-retrieval](specifications/amazon-browser-retrieval/functional.md) | specified (issue #204); FR-AMZ-BRET-008 specified (issue #226); FR-AMZ-BRET-009 specified (issue #210, documentation: the failure log line ends with `response=…`, the start of the page Amazon returned, which can be empty; no behaviour change) |
| Amazon thumbnail URLs | [amazon-thumbnail-urls](specifications/amazon-thumbnail-urls/functional.md) | specified (issue #275: an Amazon check records as thumbnail only an HTTPS URL on `m.media-amazon.com` or `images-eu.ssl-images-amazon.com`, otherwise no thumbnail) |
| GitHub release | [github-release](specifications/github-release/functional.md) | specified (operations) |
| Test suite hygiene | [test-suite-hygiene](specifications/test-suite-hygiene/functional.md) | specified (issue #208, maintenance: removes the empty leftover `internal/service/zz_scratch_docs_test.go`; no behaviour change); FR-TEST-HYG-004–008 specified (issue #232, maintenance: the test check that no file is left by an Amazon download fails on a file it cannot read, so the evidence for FR-AMZ-HCHK-014–015 fails closed; no behaviour change); FR-TEST-HYG-009–013 specified (issue #244, maintenance: the text search for the Amazon test helper's import is replaced by a check, run in the unit-test gate, that the server binary's dependencies do not include the test helper; no behaviour change); FR-TEST-HYG-014–017 specified (issue #251, maintenance: the fixture modules used by the dependency check's unit tests compile, except those meant to make the listing fail; no behaviour change); FR-TEST-HYG-018–021 specified (issue #241, maintenance: the finished TC-005 follow-up record `doc/todo/issue-233-r1-03.md` is deleted and issue #238 is closed as completed by PR #236; no behaviour change); FR-TEST-HYG-022–026 specified (issue #185, maintenance: tests prove that a scheduled Amazon check whose window has passed is not sent but redrawn and logged, that every Amazon item gets its own window time at the first start after the upgrade, and that empty-body 404/410 are unavailable while 503/429 are failures with their HTTP status; `internal/amazon/browser_test.go` is `gofmt`-formatted; no behaviour change); FR-TEST-HYG-027–029 specified (issue #238, maintenance: TC-005 fails with a message when the Amazon check ended before the server stop, a guard already added by PR #236 and kept; merging the change closes issue #238 as completed, which settles FR-TEST-HYG-019; no behaviour change); FR-TEST-HYG-030–032 specified (issue #181, maintenance: TC-006 of the issue #168 test cases uses HTTP 403 instead of 404 as a failure case, and TC-004 gains a 404/410 → listing unavailable corner case, matching FR-AMZ-HCHK-012 and the tests; no behaviour change); FR-TEST-HYG-033–036 specified (issue #182, maintenance: the TC-032 Refresh price subtest makes the scheduled check fall due while the Refresh price is running, and a new test proves that an Amazon item stopped at its morning time and restarted by Refresh price at 07:30 is not checked again at 07:31 and gets its next check in the evening window; no behaviour change); FR-TEST-HYG-037–039 specified (issue #295, maintenance: the always-passing image-request assertion of e2e test TC-010 is replaced by one that fails unless the accepted Amazon image was requested; no behaviour change); FR-TEST-HYG-040–043 specified (issue #195, maintenance: the TC-002 fixture sanity check of the Amazon browser profile passes only when the browser reports that it refuses the profile because it is in use on another computer, not merely when the page is not requested; no behaviour change); FR-TEST-HYG-044–047 specified (issue #312, maintenance: in the issue #195 fixture check test, the rows whose stub exits by itself wait up to 15 s instead of 2 s before the stub is stopped, so a slow machine does not make them fail, while the rows whose stub keeps running keep 2 s; no behaviour change); FR-TEST-HYG-048–052 specified (issue #315, maintenance: in the same test, a row whose stub keeps running starts its 2 s wait only once the stub signals that it has written its output, waits at most 15 s for that signal and fails if it does not come, so a slow machine does not make the row fail; no behaviour change); FR-TEST-HYG-053–056 specified (issue #324, maintenance: the issue #197 TC-003 subtest "temporary profile" removes the `pricefollower-browser-*` directory it was given however it ends, including on an early fatal failure, while still checking that the collector's cleanup removes it; no behaviour change); FR-TEST-HYG-057–060 specified (issue #290, maintenance: the issue #286 e2e test of a 1025-space Claude token input drops its always-passing check for the "Saving settings…" text after the modal has closed and its loop over the sent requests, and instead fails if any save request is sent, keeping its "too long" check; no behaviour change); FR-TEST-HYG-061–064 specified (issue #322, maintenance: the issue #197 TC-005 test that compares the source text of `profile()` in `internal/amazon/browser.go` with fixed strings is removed with its `go/ast`, `go/format`, `go/parser` and `go/token` imports, as TC-001 to TC-003 prove FR-AMZ-BPROF-008–009 through behaviour; no behaviour change); FR-TEST-HYG-065–069 specified (issue #254, maintenance: the issue #251 fixture build check also compiles the fixture modules' test files, so the test files of row d are type-checked, and a control proves that a type error in a fixture test file fails it; no behaviour change); FR-TEST-HYG-070–073 specified (issue #253, maintenance: in the fixture build check, rows f and g pass only when the build failure holds their expected text, `pricefollower.local/internal/nope` or `main.go`, not on any build failure; no behaviour change); FR-TEST-HYG-074–077 specified (issue #340, maintenance: the fixture build check's `go vet` step only type-checks the fixtures with their test files, so a valid fixture fails only on a compile error, not on a vet analyzer finding, and the check's comments say so; no behaviour change); FR-TEST-HYG-078–081 specified (issue #239, maintenance: TC-003 of issue #233 fails unless the cache log line of a never-answered clear gives `no answer within`, and in every subtest when it gives `context canceled` or `context deadline exceeded`; the TC-003 entry of the issue #233 test cases says so; no behaviour change); FR-TEST-HYG-082–085 specified (issue #344, maintenance: the analyzer-findings subtest of the fixture build check also runs `go vet ./...` with all analyzers on its fixture and fails with "the fixture has no analyzer finding, so the subtest proves nothing" when that run succeeds, so the subtest cannot pass on a fixture without findings; no behaviour change); FR-TEST-HYG-086–089 specified (issue #349, maintenance: that guard also fails, naming the file, unless the vet output holds a finding position in both `cmd/app/main.go` and `cmd/app/main_test.go`, so the subtest keeps proving that findings in test files are ignored; no behaviour change); FR-TEST-HYG-090–093 specified (issue #260, maintenance: the issue #185 TC-002 test of the upgrade redraw fails if two of its Amazon items get the same next scheduled check, so one time drawn for every item is caught, and TC-002 of the issue #185 test cases says so; no behaviour change); FR-TEST-HYG-094–098 specified (issue #199, maintenance: the e2e test TC-005 of issue #165 checks that a card value starts a new line only when it does not fit by counting the item at its 14rem base width instead of its grown width, so it fails if Prices is pushed below the item while it would fit beside it, and TC-005 of the issue #165 test cases says so; no behaviour change); FR-TEST-HYG-099–102 specified (issue #359, maintenance: `TestBrowserCacheLog_TC003_SuccessfulClearLogsNothing`, which repeated three issue #233 tests, is removed from `internal/amazon/browser_cache_log_test.go`, so TC-003 of issue #240 stays proven by those unchanged tests, as its test cases and technical specification already state; no behaviour change); FR-TEST-HYG-103–107 specified (issue #358, maintenance: TC-005 of issue #233 also fails when a line saying the Amazon browser cache could not be cleared is logged during either owner's Refresh price, and the issue #240 real-Chromium copy of TC-005, `internal/httpapi/amazon_browser_cache_log_test.go`, is deleted; the TC-004 entry of the issue #240 test cases and technical specification name the extended TC-005; no behaviour change) |
| Responsive layout | [responsive-layout](specifications/responsive-layout/functional.md) | specified (issue #140); FR-RESPONSIVE-018 to 024 specified, 016/017 updated (issue #165); FR-RESPONSIVE-025–027 specified (issue #203, maintenance: the issue #165 references say "016/017 updated" and no longer list 020 as updated, as FR-RESPONSIVE-020 was introduced by issue #165; no behaviour change); FR-RESPONSIVE-028–030 specified (issue #200, maintenance: TS-RESPONSIVE-017 says only value cells keep their size, the item cell grows and the actions cell ends the line, as in TS-RESPONSIVE-023; no behaviour change); FR-RESPONSIVE-031–033 specified (issue #202, maintenance: below 1056 px `max-width: none` is set only on the login submit button, with a comment saying it lifts Carbon's 20rem button cap, not on the login form, which has no cap; no behaviour change); FR-RESPONSIVE-034–036 implemented (issue #201, maintenance: below 1056 px the card cells' `display: block` is declared in the `.responsive-table tbody tr td` rule and the standalone `.responsive-table td` rule is removed; no behaviour change) |
