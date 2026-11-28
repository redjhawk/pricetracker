# Test cases: align backend step 4 of the Amazon human-checks spec with TS-AMZ-HCHK-017 (issue #234)

Sources: functional specification [amazon-human-checks](../../specifications/amazon-human-checks/functional.md) (FR-AMZ-HCHK-017 to FR-AMZ-HCHK-019). No use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **Technical specification:** `doc/specifications/amazon-human-checks/technical.md`.
- **Step 4:** the one line of the technical specification, in the Backend `Collect` steps, that starts ``4. `Page.navigate(item.URL)` `` (line 73 on `master`).
- **New clause:** `with a 2xx response or none (a response may come after the 1 s wait), the HTTP cache is cleared first (issue #217, TS-AMZ-HCHK-017)`.
- **Old clause:** `with a 2xx response the HTTP cache is cleared first`.
- Searches are fixed-string (`grep -F`), so the parentheses, `#` and the en dash (`–`) in `TS-AMZ-HCHK-018–019` are matched literally.
- The change is documentation maintenance. No production code, test code, API, database or user interface changes, so no new Go or Playwright test code is written. The cases are repository checks and runs of the existing suites. Narrow screens, keyboard and screen states do not apply.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-AMZ-HCHK-017 | TC-001, TC-003 |
| FR-AMZ-HCHK-018 | TC-002, TC-003 |
| FR-AMZ-HCHK-019 | TC-004, TC-005, TC-006 |

## Cases

### TC-001: Step 4 says the cache is cleared with a 2xx response or none

- Covers: FR-AMZ-HCHK-017
- Level: repository check (`grep -nF` on the technical specification)
- Given the change applied on top of `master`
- When `grep -nF '<new clause>' doc/specifications/amazon-human-checks/technical.md` is run
- Then exactly one line matches, and it is step 4.
- Corner cases:
  - the clause is in step 4, not in another step, the TS-AMZ-HCHK-017 row or a bullet;
  - it is written once: step 4 holds no second, older or partial copy of it;
  - the exact characters match: `1 s` with a space, `issue #217`, a comma and a space before `TS-AMZ-HCHK-017`, both parentheses closed;
  - step 4 is still one numbered line, between steps 3 and 5, and the step numbering is unchanged.

### TC-002: The old wording is gone and the rest of step 4 keeps its meaning

- Covers: FR-AMZ-HCHK-018
- Level: repository check (`grep -cF` and `git diff --word-diff` against `master`)
- Given the change applied on top of `master`
- When `grep -cF '<old clause>'` is run on the technical specification, `grep -F` is run on step 4 for `also when the check is interrupted` and for `clears it too (issue #233, TS-AMZ-HCHK-018–019)`, and `git diff --word-diff master -- doc/specifications/amazon-human-checks/technical.md` is read for step 4
- Then:
  - the old clause count is 0;
  - step 4 contains both kept phrases, in this order, after the new clause;
  - the word diff of step 4 is only the insertion of `or none (a response may come after the 1 s wait),` after `with a 2xx response`.
- Corner cases:
  - no other text of the technical specification says that the cache is cleared only with a 2xx response (for example `only with a 2xx response`, or a reworded copy in another section);
  - the rest of step 4 is unchanged character for character: the `errorText` cases, the failure messages `Amazon page could not be opened: <errorText>` and `Amazon page did not finish loading`, FR-AMZ-HCHK-011/012 and issue #186 references, and the `Page.loadEventFired` sentence;
  - the en dash in `TS-AMZ-HCHK-018–019` is kept (not `-` or `to`).

### TC-003: Step 4 agrees with the TS-AMZ-HCHK-017 row, its Backend bullet and the code

- Covers: FR-AMZ-HCHK-017, FR-AMZ-HCHK-018
- Level: review check (reading the technical specification and `internal/amazon/browser.go`)
- Given the change applied
- When step 4 is read next to the TS-AMZ-HCHK-017 row, the TS-AMZ-HCHK-017 Backend bullet and its sub-bullets, and the `errorText` branch of `browserTab.load` in `internal/amazon/browser.go`
- Then all of them give the same rule: after an `errorText` navigation the HTTP cache is cleared unless a non-2xx Document response of the navigated frame was recorded within the 1 s wait, that is with a 2xx response or with none.
- Corner cases:
  - no response at all (DNS failure, refused, reset) clears the cache in step 4 as in the bullet `A navigation without a response (DNS, refused) also clears the HTTP cache`;
  - a response that comes after the 1 s wait is covered by `or none`;
  - a non-2xx error page (FR-AMZ-HCHK-012) still does not clear the cache: step 4 keeps it as the loaded page, marked `errorPage`;
  - the interrupted check and the interrupted `Page.navigate` (issue #233) are still described as clearing the cache.

### TC-004: No other documentation text changes

- Covers: FR-AMZ-HCHK-019
- Level: repository check (`git diff` against `master`)
- Given the change applied on top of `master`
- When `git diff --name-only master -- doc/` and the diffs of `doc/specifications/amazon-human-checks/functional.md` and `technical.md` are read
- Then:
  - in `functional.md`, the lines of FR-AMZ-HCHK-001 to FR-AMZ-HCHK-016 are unchanged;
  - in `technical.md`, the only changed lines are step 4 and the lines that record the issue #234 change (for example the status line, the functional specification and test case references, an issue #234 section);
  - the TS-AMZ-HCHK-017 row (line 30 on `master`), its Backend bullet with the code block and sub-bullets, and every other TS row and entry are unchanged;
  - the changed files are only `doc/FUNCTIONAL_SPECIFICATIONS.md`, the two amazon-human-checks specification files and files under `doc/changes/2026-11-28-1330-issue-234-align-backend-step-4-of-the-amazon-human/`.
- Corner cases:
  - the change folders of other issues under `doc/changes/` are unchanged, in particular the issue #217, #233 and #240 folders that the technical specification links to;
  - every existing file under `doc/todo/` is unchanged and none is deleted, including `doc/todo/issue-217-r2-02.md`, the source of the issue;
  - the change folder has no `delete.txt`, or an empty one;
  - the diff of `doc/FUNCTIONAL_SPECIFICATIONS.md` is only the **Amazon human-like checks** row;
  - the issue #234 records give no new requirement for the code: they say the change is maintenance with no behaviour change.

### TC-005: The diff outside `doc/` is empty

- Covers: FR-AMZ-HCHK-019
- Level: repository check (`git diff` against `master`)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` is run
- Then it prints nothing.
- Corner cases:
  - `internal/amazon/browser.go` and every other Go file, `_test.go` file, frontend source, `.spec.ts` file, `API_SPECIFICATION.md`, migrations, `go.mod`, `go.sum` and `package.json` are unchanged;
  - no file outside `doc/` is added or deleted.

### TC-006: The existing suites pass and the application does not change

- Covers: FR-AMZ-HCHK-019
- Level: suite run (deterministic gates: `go vet ./...`, `go test ./...`, the frontend build and `npx playwright test`). `go test` includes the integration tests with a real SQLite database and HTTP handlers, and Playwright runs the e2e flows.
- Given the change applied on top of `master`
- When the gates run
- Then `go vet ./...` and `go test ./...` succeed, and so do the other gates. The Playwright suite runs the same test list as on `master` and passes, so the operator observes no change in the application.
- Corner cases:
  - the existing tests that prove the behaviour step 4 now describes keep passing unchanged: `TestBrowserInterrupted_TC001_ClearConfirmedBeforeClose` (`errorText` with no recorded response, cache cleared), `TestBrowserInterrupted_TC006_UninterruptedCheckIsUnchanged` (subtest `download`: 2xx, cleared; subtest `error page`: 503, not cleared), `TestBrowserCacheLog_TC001_PageNotOpened` and `TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile`;
  - the gates give the same result with `go test -count=1 ./...` (no cached results) and when run twice;
  - no test reads the edited documents, so a doc-only diff cannot change a test result. The run proves that nothing else changed.
