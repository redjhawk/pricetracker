# Test cases: Ubuntu Chromium snap pitfall in the installation documentation (issue #226)

Sources: [functional specification](../../specifications/amazon-browser-retrieval/functional.md) (FR-AMZ-BRET-008), use case [UC-AMZ-BRET-005](../../use-cases/amazon-browser-retrieval/UC-AMZ-BRET-005-use-google-chrome-when-chromium-snap-is-installed.md).

Conventions:

- **Browser section** is the `### Browser for Amazon checks` subsection of the `## Server setup` section of `deploy/README.md`, as read by the documentation tests of issue #204 (`internal/service/amazon_browser_retrieval_docs_test.go`).
- **Snap sentence** is the sentence required by FR-AMZ-BRET-008: it mentions a Chromium snap (the word "snap"), Ubuntu, `PRICEFOLLOWER_BROWSER_PATH`, Google Chrome and the example path `/usr/bin/google-chrome-stable`.
- **Snap stub** is an executable script named `chromium` that, like a confined snap Chromium, writes a start error (for example `cannot create user data directory: Permission denied`) to its error output and exits 1.
- New Go test names use the prefix `TestAmazonBrowserSnapDocs_TCnnn_` so they do not clash with the `TCnnn` names of issues #168, #180 and #204.
- Documentation only: no interface change, no e2e case. The existing #204 documentation tests (#204 TC-011) and the whole repository test suite (`go test ./...`, Playwright tests in `tests/`) must keep passing; that is part of FR-AMZ-BRET-008's acceptance criteria.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-BRET-008 | TC-001, TC-002, TC-003, TC-004 |
| FR-AMZ-BRET-006 | TC-002, TC-004 |

| Use case | Test cases |
| --- | --- |
| UC-AMZ-BRET-005 | TC-001, TC-002, TC-003, TC-004 |

## Documentation

### TC-001 — The browser section tells the operator what to do when a Chromium snap is installed

- Covers: FR-AMZ-BRET-008; UC-AMZ-BRET-005 (main flow steps 1–3)
- Level: unit (documentation test)
- Given the browser section of `deploy/README.md`
- When the test reads it
- Then it contains the snap sentence: one sentence (text between two sentence ends, outside fenced code) that contains "snap", "Ubuntu", `PRICEFOLLOWER_BROWSER_PATH`, "Google Chrome" and `/usr/bin/google-chrome-stable`; the sentence is in the same subsection as, and after, the paragraph that explains `PRICEFOLLOWER_BROWSER_PATH` and the names looked up on `PATH`
- Corner cases (the checks run on the real file and on in-test fixtures, as #204 TC-011 does with `specifiedServerSetup`):
  - fixture without the sentence: fails;
  - fixture whose sentence lacks the setting name, or lacks the example path: fails;
  - fixture with the words split over two sentences (snap in one, the setting and path in another): fails;
  - fixture with the sentence only in another section (for example `## Administrator account`) or only inside a fenced code block: fails;
  - fixture of the browser section as the technical specification describes it, with the sentence: passes.

### TC-002 — The documented looked-up names stay those of the application, in the same order

- Covers: FR-AMZ-BRET-008, FR-AMZ-BRET-006; UC-AMZ-BRET-005 (main flow step 2, postconditions)
- Level: unit (documentation test)
- Given the browser section of `deploy/README.md` with the snap sentence
- When the executable names are collected from its prose (backtick tokens matching a browser name, outside fenced code, in reading order, without repeats)
- Then the list is exactly `chromium`, `chromium-browser`, `google-chrome-stable`, `google-chrome`, equal in content and order to `amazon.BrowserNames`; the documented `PRICEFOLLOWER_BROWSER_*` variables are still exactly those the code reads; #204 TC-011 keeps passing on the changed file
- Corner cases:
  - the backticked full path `/usr/bin/google-chrome-stable` is not taken for a looked-up name (no "extra executable name" failure);
  - fixture listing the four names in another order: fails;
  - fixture with an extra bare name in backticks (for example `google-chrome-unstable` or `chromium-snap`): fails;
  - fixture that mentions `chromium` again in the snap sentence after the list: passes (repeats do not change the order).

## Application behaviour unchanged

### TC-003 — The browser lookup order is unchanged

- Covers: FR-AMZ-BRET-008; UC-AMZ-BRET-005 (postconditions, alternative "operator does not set the setting")
- Level: unit
- Given a temporary directory on `PATH` holding executable stubs named `chromium` and `google-chrome-stable`, and `PRICEFOLLOWER_BROWSER_PATH` unset
- When `amazon.FindBrowser` is called
- Then it returns the `chromium` stub (the snap name is still found first); `amazon.BrowserNames` equals `chromium`, `chromium-browser`, `google-chrome-stable`, `google-chrome`
- Corner cases:
  - only `chromium-browser` and `google-chrome` on `PATH`: returns `chromium-browser`;
  - only `google-chrome-stable` on `PATH`: returns it;
  - `PRICEFOLLOWER_BROWSER_PATH` set to the `google-chrome-stable` stub while `chromium` is on `PATH`: returns the set path;
  - `PRICEFOLLOWER_BROWSER_PATH` set to an empty string: same as unset;
  - no browser on `PATH` and the setting unset: returns an error, as before.

## User flow

### TC-004 — Setting the browser path as documented makes checks work despite a failing Chromium snap

- Covers: FR-AMZ-BRET-008, FR-AMZ-BRET-006; UC-AMZ-BRET-005 (main flow steps 5–6, alternatives)
- Level: integration (SQLite + browser test)
- Given the service with the collector it builds from its configuration and environment (only the test origin added, as in #204 TC-012), a directory on `PATH` holding the snap stub as `chromium`, the fixture server serving a product page with a euro price, and the log captured
- When (a) Refresh price is requested for an Amazon item with `PRICEFOLLOWER_BROWSER_PATH` unset, then (b) `PRICEFOLLOWER_BROWSER_PATH` is set to the test browser (standing for Google Chrome) without restarting the service and Refresh price is requested again
- Then (a) the check fails: the item shows a retrieval issue, keeps its last successful price, the failure count is 1 and the log has "browser could not start" with the stub's message; (b) the check uses the set browser, records the fixture price, resets the failure count to 0 and logs no failure
- Corner cases:
  - only the test browser on `PATH` as `google-chrome-stable` and the setting unset (no snap): the check succeeds with no change (UC-AMZ-BRET-005 alternative "only Google Chrome installed");
  - the setting already pointing to the test browser while the snap stub is on `PATH`: succeeds;
  - Refresh price repeated after (b): succeeds again;
  - the setting pointing to a missing file: fails with "browser could not start" and the missing-file reason.
