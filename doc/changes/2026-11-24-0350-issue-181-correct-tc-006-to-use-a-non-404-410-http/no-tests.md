# No tests: issue #181

This change only corrects two lines of a test case document, `doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n/testcases.md` (below `<168 doc>`): the TC-006 Given line and the TC-004 Corner cases line. It changes no production code, test code, API, database or user interface, so no automated test can fail before it and pass after it. The behaviour the corrected document describes is already tested on `master`, and those tests stay as they are (FR-TEST-HYG-032). This follows the technical specification ([test-suite-hygiene/technical.md](../../specifications/test-suite-hygiene/technical.md), "TC-006 failure statuses match the specification (issue #181)" and "Test case mapping (issue #181)").

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | Repository check: the `### TC-006` entry of `<168 doc>` names "HTTP 503, HTTP 403 for the product page, and HTTP 200 with an Amazon robot-check (captcha) page" and holds no `404` or `410`; `git diff master -- <168 doc>` changes only its Given line. The three cases match the sub-tests `HTTP 503`, `HTTP 403` and `robot check` of `TestBrowserCollector_TC006_NonSuccessAndRobotCheckAreFailures` (`internal/amazon/browser_test.go`). |
| TC-002 | Repository check: the TC-004 Corner cases line of `<168 doc>` keeps its three corner cases unchanged and ends with `product page answered with HTTP 404 or HTTP 410 gives "listing unavailable" (an Amazon answer, not a failure)`; `git diff master -- <168 doc>` changes only that line. The tests agree: the `product page answering 404` sub-test of `TestBrowserCollector_TC004_ResultsMatchTheParser` expects `unavailable` with "The listing is no longer available.", and for 410 `TestBrowserErrorPage_TC001_EmptyUnavailablePageSendsNoUsedOfferRequest` (`internal/amazon/browser_error_page_test.go`) and `TestBrowserEmptyErrorStatus_Issue185_TC003_ClassifiedByStatus` (`internal/amazon/browser_empty_status_test.go`). |
| TC-003 | Repository check: `git diff --name-status master -- doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n` lists only `M <168 doc>`, and `git diff master -- <168 doc>` has exactly two changed lines, inside the TC-004 and TC-006 entries. |
| TC-004 | Repository check: `git diff --name-status master -- . ':!doc'` and `git diff --name-status master -- doc/todo` are empty. Suite runs: `go test -count=1 -run 'TestBrowserCollector_TC004\|TestBrowserCollector_TC006' ./internal/amazon/` passes, and so do the gates (`go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` with the same test list as `master`). |

## Run in this stage (2026-11-24)

- `git diff master -- doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n`: one file, two changed lines. Line 76 (TC-004 Corner cases) gains the 404/410 "listing unavailable" corner case, and line 91 (TC-006 Given) has `HTTP 404` replaced by `HTTP 403`. Nothing else changes.
- `git diff --name-status master -- . ':!doc'` and `git diff --name-status master -- doc/todo`: both empty. `git diff --name-status master -- doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n`: only `M` for its `testcases.md`.
- `go test -count=1 -run 'TestBrowserCollector_TC004|TestBrowserCollector_TC006' -v ./internal/amazon/`: `PASS`. The sub-test `product_page_answering_404` passes, and so do `HTTP_503`, `HTTP_403` and `robot_check`.
- The full gates (`go vet ./...`, `go test ./...`, `node --test`, `npm run build`, `npx playwright test`) are left to the pipeline and the development stage.
