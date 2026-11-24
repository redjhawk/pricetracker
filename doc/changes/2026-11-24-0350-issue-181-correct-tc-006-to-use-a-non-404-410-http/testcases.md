# Test cases: correct TC-006 to use a non-404/410 HTTP status (issue #181)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-030 to FR-TEST-HYG-032). No use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **Issue #168 document:** `doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n/testcases.md`. Its TC-004 and TC-006 are entries of that document, not cases of this catalogue; the IDs below are this change's own.
- **Existing tests:** `TestBrowserCollector_TC004_ResultsMatchTheParser` (sub-test "product page answering 404") and `TestBrowserCollector_TC006_NonSuccessAndRobotCheckAreFailures` (sub-tests "HTTP 503", "HTTP 403", "robot check") in `internal/amazon/browser_test.go`; the 410 product page is exercised by `TestBrowserErrorPage_TC001_EmptyUnavailablePageSendsNoUsedOfferRequest` and `TestBrowserEmptyErrorStatus_Issue185_TC003_ClassifiedByStatus`.
- The change is test documentation maintenance: it only edits two entries of the issue #168 document. No production code, test code, API, database or user interface changes, so no new Go or Playwright test code is written; the cases are repository checks and runs of the existing suites. Narrow screens, keyboard and screen states do not apply.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-TEST-HYG-030 | TC-001, TC-003 |
| FR-TEST-HYG-031 | TC-002, TC-003 |
| FR-TEST-HYG-032 | TC-003, TC-004 |

## Cases

### TC-001 — TC-006 lists 503, 403 and the robot-check page as failures, and no 404 or 410

- Covers: FR-TEST-HYG-030
- Level: repository check (document content and `git diff`)
- Given the change applied on top of `master`
- When the TC-006 entry of the issue #168 document (from its `### TC-006` heading to the next `###` heading) is read and compared with its `master` version
- Then its Given line names, as separate sub-tests, HTTP 503, HTTP 403 for the product page, and HTTP 200 with an Amazon robot-check (captcha) page; the entry contains neither "404" nor "410"; and the only line that differs from `master` is the Given line
- Corner cases:
  - the ID, title, Covers line (FR-AMZ-HCHK-003, FR-AMZ-STOP-001), level, When, Then (failure, not success, not price not found, not unavailable, with request URL, HTTP status and response excerpt) and corner case (no cookie values or session secrets, FR-AMZ-HUMAN-012) are byte-for-byte unchanged;
  - the three failure cases match the three sub-tests of `TestBrowserCollector_TC006_NonSuccessAndRobotCheckAreFailures` (HTTP 503, HTTP 403, robot check), and no fourth case is added;
  - 403 is named for the product page, as 404 was before, not for another resource.

### TC-002 — TC-004 gains the 404/410 "listing unavailable" corner case

- Covers: FR-TEST-HYG-031
- Level: repository check (document content and `git diff`)
- Given the change applied on top of `master`
- When the TC-004 entry of the issue #168 document is read and compared with its `master` version
- Then its corner cases include a product page answered with HTTP 404 or HTTP 410 whose result is "listing unavailable", stated as an Amazon answer and not a failure; and the only line that differs from `master` is the Corner cases line
- Corner cases:
  - both statuses are named (404 and 410), not only 404;
  - the existing corner cases (page with a price but no second-hand offer; page with a second-hand offer and no new price; title and thumbnail kept as today) are still present and unreworded;
  - the ID, title, Covers line (FR-AMZ-HCHK-002; UC-AMZ-HCHK-001, UC-AMZ-HCHK-002, UC-AMZ-HCHK-003), level, Given, When and Then are unchanged;
  - the new corner case agrees with the behaviour the tests prove: the "product page answering 404" sub-test of `TestBrowserCollector_TC004_ResultsMatchTheParser` expects `unavailable` with "The listing is no longer available.", and the 410 tests listed under Conventions expect `unavailable`.

### TC-003 — Only TC-004 and TC-006 of the issue #168 folder change

- Covers: FR-TEST-HYG-030, FR-TEST-HYG-031, FR-TEST-HYG-032
- Level: repository check (`git diff`)
- Given the change applied on top of `master`
- When `git diff --name-status master -- doc/changes/2026-11-20-1634-issue-168-error-amazon-checks-were-stopped-on-20-n` and `git diff master -- <issue #168 document>` are listed
- Then the only file changed in that folder is `testcases.md` (status `M`), and every changed hunk lies inside the TC-004 or TC-006 entry
- Corner cases:
  - no other case of the document (TC-001 to TC-003, TC-005, TC-007 onwards) and no heading, coverage table or introduction changes;
  - no file of the folder is added, deleted or renamed;
  - TC-005 (unavailable listing stored as `unavailable`, not a failure) stays as is, although it also talks about unavailable listings.

### TC-004 — Nothing outside `doc/` or under `doc/todo/` changes, and the gates pass

- Covers: FR-TEST-HYG-032
- Level: repository check and suite run (deterministic gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`, and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-status master -- . ':!doc'` and `git diff --name-status master -- doc/todo` are listed, `go test -count=1 -run 'TestBrowserCollector_TC004|TestBrowserCollector_TC006' ./internal/amazon/` runs, and the gates run
- Then both diffs are empty; both named tests pass; `go vet ./...` and `go test ./...` pass, and so do the other gates; the Playwright suite runs the same test list as on `master`
- Corner cases:
  - `doc/todo/issue-168-r1-04.md`, the review comment this issue comes from, is unchanged (not deleted, not marked done);
  - no production Go code, `_test.go` file, frontend source, `.spec.ts` file, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` changes, so the operator observes no change in the application;
  - the tests give the same result without cached results (`-count=1`) and when the gates run twice.
