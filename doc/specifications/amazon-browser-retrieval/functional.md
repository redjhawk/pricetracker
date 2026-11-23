# Functional specification: Amazon browser retrieval

Status: ready (issue #204); FR-AMZ-BRET-008 specified (issue #226)
Owner: functional specification agent
User decision/reference: GitHub issue #204 ("Retrieval error": since Amazon prices are read through a web browser, every product returns a retrieval error); GitHub issue #226 (document the Ubuntu Chromium snap pitfall, from review comment R4-01 on PR #206).

## Purpose and scope

Since Amazon pages are requested through a real web browser ([FR-AMZ-HCHK-001](../amazon-human-checks/functional.md), issue #168), every tracked Amazon item shows **Retrieval error**: no Amazon check returns a price any more. The behaviour expected from browser checks is already specified ([FR-AMZ-HCHK-002](../amazon-human-checks/functional.md)): a usable Amazon answer gives the same results as before the change. The user asks that Amazon checks work again.

Included: the outcome of every Amazon check (first check after add, scheduled check, **Refresh all prices**, **Refresh price**) of a tracked Amazon item, on a server installed as documented; the information the operator gets to install the server and to understand a failed check.

Issue #226 adds one documentation requirement (FR-AMZ-BRET-008). On Ubuntu, the `chromium` and `chromium-browser` names usually lead to the snap Chromium, and the application finds it before Google Chrome. Snap confinement can stop it from starting for the service (profile under a system data directory, system service user), and then every Amazon check fails with "browser could not start". Issue #226 does not change the application's behaviour, including the order in which it looks for a browser.

Excluded: LeBoncoin; the use of a real web browser itself (FR-AMZ-HCHK-001 stays); the random check windows ([FR-AMZ-HCHK-004–009](../amazon-human-checks/functional.md)); the failure count, stop, warning and restart rules ([Amazon request stop](../amazon-request-stop/functional.md)), which stay unchanged; price and second-hand offer detection rules ([Amazon price detection](../amazon-price-detection/functional.md), FR-15); item states, labels and messages in the interface, which stay unchanged.

Actors: the user who tracks Amazon items; the operator who installs and runs the server. Amazon is the external system.

Terms:

- **Amazon check:** as in [Amazon human-like checks](../amazon-human-checks/functional.md).
- **Retrieval error:** the item status shown when the latest check did not succeed and the listing is not unavailable: a failed request ("Amazon could not be reached for a price check.") or a page without a detectable price ("Amazon did not show a detectable euro price.").
- **Available listing:** an Amazon product page of a supported marketplace that a person opening it in a desktop browser sees with its product and its price in euros.
- **Robot check:** an Amazon page that, instead of the product, asks the visitor to prove they are not a robot (for example to type the characters of an image).
- **Documented installation:** a server prepared as described in the repository's installation documentation (FR-AMZ-BRET-006).

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMZ-BRET-001 | Any Amazon check of an available listing, on a documented installation, when Amazon shows the product page. | The check succeeds and records the listing's price, title and image as before the browser change (FR-AMZ-HCHK-002, FR-AMAZON-PRICE-001–004). The item shows its price, not **Retrieval error**. | For an available listing, **Refresh price** records the price that the product page shows; the item status is not Retrieval error and the latest attempt has no error message. |
| FR-AMZ-BRET-002 | Same as FR-AMZ-BRET-001. | The check's second-hand offer result is the same as before the browser change (FR-15): the latest offer sold by Amazon with its condition when the page or its used offers show one, otherwise no offer. | For a listing whose page shows a second-hand offer sold by Amazon, the check records that offer's price and condition. |
| FR-AMZ-BRET-003 | The product page shows the product and its price, and its content (texts, links, page scripts) contains words such as "captcha" or "robot" without the page being a robot check. | The page is read as a product page (FR-AMZ-BRET-001); it is not taken for a robot check or a failure. | A product page whose scripts mention "captcha" gives its price; no "blocked" failure is logged or counted. |
| FR-AMZ-BRET-004 | Amazon answers with a robot check, a non-success response, or no answer. | Unchanged: it is a failure, logged, counted and stopping items as in FR-AMZ-HCHK-003 and FR-AMZ-STOP-001–010. The last successful price is kept (FR-07). | A robot-check page gives Retrieval error and counts one failure; 5 consecutive failures stop every tracked Amazon item. |
| FR-AMZ-BRET-005 | An Amazon check fails. | The server log entry of the failure states its cause in words the operator can act on, distinguishing at least: the browser could not start (with the browser's own reason), the page could not be opened or did not finish loading, Amazon answered with a non-success status (with the status), Amazon showed a robot check, the page showed no detectable price. Logs never contain tokens or session secrets (FR-AMZ-HUMAN-012). | For each listed cause, the logged failure names that cause, and the request URL and time (FR-AMZ-STOP-001); two different causes never produce the same log text. |
| FR-AMZ-BRET-006 | The operator installs or updates the server. | The installation documentation states everything the server needs for Amazon checks to work: the web browser to install, how the application finds it, and the settings needed when the service runs in a container or as root. At start, when the browser cannot be found, the server log says so (as today). | Following only the installation documentation on a server without a browser gives a server whose Amazon checks of available listings succeed (FR-AMZ-BRET-001). |
| FR-AMZ-BRET-007 | Amazon items stopped by the failures of issue #204, after the fix is installed. | Unchanged rules: each stopped item stays stopped until a successful **Refresh price** restarts it (FR-AMZ-STOP-006); items not stopped get their next scheduled check. | After installation, **Refresh price** on a stopped item of an available listing records its price and restarts the item. |
| FR-AMZ-BRET-008 | The operator reads the installation documentation's section on the browser for Amazon checks (issue #226). | Next to the explanation of how the application finds the browser, the section has one sentence: when a Chromium snap is installed (usual on Ubuntu), set `PRICEFOLLOWER_BROWSER_PATH` to the Google Chrome executable, for example `/usr/bin/google-chrome-stable`. The executable names that the documentation lists as looked up on the service's `PATH` stay exactly those the application looks up, in the same order. | The section contains the sentence with the setting name and the example path. The documented list of looked-up names is still `chromium`, `chromium-browser`, `google-chrome-stable`, `google-chrome`. The repository test suite passes. |

## States and corner cases

- Item states and messages are unchanged (pending, active, stale, retrieval issue, unavailable, stopped).
- A listing that is really unavailable (404/410 or Amazon's unavailable text) is still shown as **Listing unavailable**, not Retrieval error.
- A product page whose only price is an instalment amount still gives "price not found" (FR-AMAZON-PRICE-003).
- Cookie or consent banners shown by Amazon over the product page do not prevent reading the price.
- The browser is never shown to the user and needs no user action; no Amazon login is used.
- In protected mode the behaviour is the same for the Amazon items of all users.
- A server where only Google Chrome is installed, or where `PRICEFOLLOWER_BROWSER_PATH` is already set, needs no change for FR-AMZ-BRET-008.

## Open questions

None. Stopped items follow the existing restart rule (FR-AMZ-BRET-007); the issue asks for no other change. Issue #226 gives the sentence's content and the example path (FR-AMZ-BRET-008).

## Traceability

- Related: [Amazon human-like checks](../amazon-human-checks/functional.md), [Amazon browser profile recovery](../amazon-browser-profile/functional.md), [Amazon request stop](../amazon-request-stop/functional.md), [Amazon price detection](../amazon-price-detection/functional.md).
- Technical handoff target: `doc/specifications/amazon-browser-retrieval/technical.md`.
