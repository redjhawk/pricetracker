# Test cases: `response` field of the Amazon failure log line in the installation documentation (issue #210)

Sources: [functional specification](../../specifications/amazon-browser-retrieval/functional.md) (FR-AMZ-BRET-009), use case [UC-AMZ-BRET-006](../../use-cases/amazon-browser-retrieval/UC-AMZ-BRET-006-read-response-field-of-failed-amazon-check.md).

Conventions:

- **Log list** is the "What the server log shows:" list in the `### Browser for Amazon checks` subsection of the `## Server setup` section of `deploy/README.md`, as read by the documentation tests of issues #204 and #226 (`internal/service/amazon_browser_retrieval_docs_test.go`).
- **Documented failure line** is the inline code token of the log list item "For each failed Amazon check:" that starts with `Amazon request failed`. **Documented no-price line** is the inline code token of the item "For a page without a detectable price:" that starts with `Amazon check found no price`.
- **Field names** of a line are its `name=` tokens, in reading order. **Documented pattern** of a line is the regular expression built from the documented line by quoting its text and replacing each `…` with a lazy capture, except the last one, which captures to the end of the line.
- **Response sentence** is the sentence required by FR-AMZ-BRET-009: one sentence (text between two sentence ends, outside fenced code) in the log list item of the failed check, after the documented failure line, that contains `response`, says it is the start of the page Amazon returned ("start" and "page") and that it can be empty ("empty").
- New Go test names use the prefix `TestAmazonFailureLogDocs_TCnnn_` so they do not clash with the `TCnnn` names of issues #168, #180, #204 and #226.
- Documentation only: no interface change, no e2e case. The existing documentation tests (#204 TC-011, #226 TC-001–TC-002) and the whole repository test suite (`go test ./...`, Playwright tests in `tests/`) must keep passing; that is part of FR-AMZ-BRET-009's acceptance criteria.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-BRET-009 | TC-001, TC-002, TC-003, TC-004, TC-005 |

| Use case | Test cases |
| --- | --- |
| UC-AMZ-BRET-006 | TC-001, TC-002, TC-003, TC-004, TC-005 |

## Documentation

### TC-001 — The documented failure line ends with the `response` field

- Covers: FR-AMZ-BRET-009; UC-AMZ-BRET-006 (main flow steps 1–2)
- Level: unit (documentation test)
- Given the browser section of `deploy/README.md`
- When the test reads the log list
- Then the documented failure line is exactly `Amazon request failed url=… at=… status=… cause=… response=…`: its field names are `url`, `at`, `status`, `cause`, `response`, in that order, and it ends with `cause=… response=…`
- Corner cases (the checks run on the real file and on in-test fixtures, as #204 TC-011 does with `specifiedServerSetup`):
  - fixture with the former line `Amazon request failed url=… at=… status=… cause=…`: fails;
  - fixture with `response=…` before `cause=…`: fails;
  - fixture with the failure line given twice, once with and once without `response`: fails;
  - fixture with the line only in another section (for example `## Administrator account`) or only inside a fenced code block: fails;
  - fixture of the log list as the technical specification describes it: passes.

### TC-002 — One sentence says what `response` holds and that it can be empty

- Covers: FR-AMZ-BRET-009; UC-AMZ-BRET-006 (main flow step 3, alternative "Amazon returned no page")
- Level: unit (documentation test)
- Given the browser section of `deploy/README.md`
- When the test reads the log list item of the failed check
- Then it contains the response sentence
- Corner cases:
  - fixture without the sentence: fails;
  - fixture whose sentence does not say it can be empty: fails;
  - fixture whose sentence does not say it is the start of the page Amazon returned: fails;
  - fixture with the words split over two sentences (`response` in one, "empty" in another): fails;
  - fixture with the sentence only in the no-price item or in another section: fails;
  - fixture of the log list as the technical specification describes it, with the sentence: passes.

### TC-003 — The rest of the log list is unchanged

- Covers: FR-AMZ-BRET-009; UC-AMZ-BRET-006 (postconditions, alternative "page without a detectable price")
- Level: unit (documentation test)
- Given the browser section of `deploy/README.md`
- When the test reads the log list
- Then the start item still reads `Amazon browser not found; Amazon checks will fail until Chromium is installed or …`; the failed-check item still has the sentence describing the causes (browser could not start, page could not be opened or did not finish loading, non-success HTTP status, robot check); the documented no-price line is still exactly `Amazon check found no price url=… at=… status=… cause=…` (no `response` field) followed by the sentence saying it is not counted as a failed request; the list still has exactly these three items
- Corner cases:
  - fixture where `response=…` was also added to the no-price line: fails;
  - fixture with the causes sentence removed or the no-price item removed: fails;
  - #204 TC-011 and #226 TC-001–TC-002 keep passing on the changed file.

## Application behaviour

### TC-004 — The documented field order matches the line the server logs

- Covers: FR-AMZ-BRET-009; UC-AMZ-BRET-006 (main flow steps 4–5, alternative "Amazon returned no page", postconditions)
- Level: unit (service with a fake Amazon collector, as in `internal/service/amazon_gate_test.go`; log captured)
- Given an Amazon item and a fake collector whose result is `request_error` with request URL, HTTP status 503, failure cause `Amazon returned HTTP 503` and response excerpt `<html><body>Service Unavailable fixture</body></html>`
- When Refresh price is requested for the item
- Then the logged line starting with `Amazon request failed` has the same field names, in the same order, as the documented failure line read from `deploy/README.md`; the documented pattern matches the logged line, capturing the cause as exactly the collector's cause and the response as exactly the excerpt
- Corner cases:
  - empty excerpt (Amazon returned no page): the logged line ends with `response=` and nothing after it; the documented pattern captures the cause exactly and an empty response;
  - excerpt containing the texts `cause=` and `status=`: the documented pattern still captures the cause exactly and the whole excerpt as response;
  - a `price_not_found` result: the logged line has the field names of the documented no-price line (`url`, `at`, `status`, `cause`) and no `response`;
  - a successful result: no failure line is logged.

## User flow

### TC-005 — The operator reads the cause and the returned page of real failed checks with the documented form

- Covers: FR-AMZ-BRET-009; UC-AMZ-BRET-006 (main flow, alternatives)
- Level: integration (SQLite + browser test, set up like #204 TC-007 in `internal/service/amazon_browser_retrieval_test.go`)
- Given the service with a real SQLite store and the test browser, the fixture server, Amazon items, and the log captured
- When Refresh price is requested for (a) an item whose page is the robot-check fixture and (b) an item checked with a missing browser executable
- Then for each check the documented pattern built from the failure line read from `deploy/README.md` matches the logged failure line; (a) the captured cause contains "robot check" and no text of the fixture page, and the captured response is non-empty and is the start of the fixture page's HTML; (b) the captured cause contains "browser could not start" and the missing path, and the captured response is empty
- Corner cases:
  - Refresh price repeated for (a): a second line, also matched with the same captures;
  - the failure count and item states are those of #204 TC-007 (behaviour unchanged);
  - no cookie value reaches the log.
