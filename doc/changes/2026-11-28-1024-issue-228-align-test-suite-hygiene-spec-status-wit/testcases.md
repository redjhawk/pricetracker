# Test cases: align the test-suite-hygiene status with the index and convention (issue #228)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-136 to FR-TEST-HYG-139). No use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **Index row:** the one line of `doc/FUNCTIONAL_SPECIFICATIONS.md` that starts `| Test suite hygiene |` (section 15). Its Status cell is the third cell.
- **Functional status line:** line 3 of `doc/specifications/test-suite-hygiene/functional.md`. **Technical status line:** line 3 of `doc/specifications/test-suite-hygiene/technical.md`.
- **Entry:** one `; `-separated item of a status line or of the Status cell: a requirement range, a status and `(issue #<n>…)`.
- **Status word:** a whole word (`grep -w`). The notes of the issue #238 and #359 entries contain `already`, which FR-TEST-HYG-136 keeps unchanged, so a substring search for `ready` would report a false failure.
- The change is documentation maintenance. No production code, test code, API, database or user interface changes, so no new Go or Playwright test code is written. The cases are repository checks and runs of the existing suites. Narrow screens, keyboard and screen states do not apply.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-TEST-HYG-136 | TC-001, TC-002 |
| FR-TEST-HYG-137 | TC-003 |
| FR-TEST-HYG-138 | TC-004 |
| FR-TEST-HYG-139 | TC-005, TC-006 |

## Cases

### TC-001: The index row gives each delivered range `implemented`, and nothing else changes in it

- Covers: FR-TEST-HYG-136
- Level: repository check (`grep` and `git diff` against `master`)
- Given the change applied on top of `master`
- When `grep -n '^| Test suite hygiene |' doc/FUNCTIONAL_SPECIFICATIONS.md` is run and its Status cell is compared, entry by entry, with the same cell on `master`
- Then:
  - exactly one line matches;
  - the Status cell starts `FR-TEST-HYG-001–003 implemented (issue #208, `;
  - it holds `FR-TEST-HYG-018, 020 and 021 implemented, FR-TEST-HYG-019 closed by the merge of the issue #238 change (issue #241, `;
  - for each of issues #232, #244, #251, #185, #238, #181, #182, #295, #195, #312, #315, #324, #290, #322, #254, #253, #340, #239, #344, #349, #260, #199, #359 and #358, the entry is `<range on master> implemented (issue #<n>, <note on master>)`;
  - the entries of issues #222, #219, #196, #387, #218 and #384 match `master` character for character, including `FR-TEST-HYG-117 updated` and `FR-TEST-HYG-112 updated`;
  - only the status words and the issue #228 entry differ from `master`.
- Corner cases:
  - the entries keep their order. None is merged, split, dropped or added, apart from the issue #228 entry, which stays last;
  - the ranges keep their en dash (`–`) and leading zeros (for example `FR-TEST-HYG-004–008`, not `FR-TEST-HYG-4-8`);
  - the issue #241 entry on `master` (`FR-TEST-HYG-018–021 specified`) becomes the split wording above, not `FR-TEST-HYG-018–021 implemented`, because FR-TEST-HYG-019 was closed, not implemented;
  - the text in parentheses after each issue number is unchanged, including notes that contain `already`, `specification` or `no behaviour change`;
  - the other cells of the row (subject name, link) are unchanged.

### TC-002: Every index entry agrees with the functional status line

- Covers: FR-TEST-HYG-136
- Level: repository check (comparison of the Status cell with the functional status line)
- Given the change applied, after the development stage
- When, for each issue number, the range and status before ` (issue #<n>` in the Status cell are compared with those of the functional status line
- Then they are the same for every issue, from #208 to #228
- Corner cases:
  - issue #241 has two entries on the functional status line (`FR-TEST-HYG-018, 020 and 021 implemented` and `FR-TEST-HYG-019 closed by the merge of the issue #238 change`) and one combined entry in the index. The combined entry holds both texts in that order, separated by `, `;
  - the issue #238 entry gives `FR-TEST-HYG-027–029 implemented` in both places. The functional line's note `whose merge also settles FR-TEST-HYG-019` is not required in the index status, and the index keeps its own note;
  - the `updated` items of issues #387 and #384 appear in both places;
  - no issue appears in one place and not in the other, and no issue appears twice in either place.

### TC-003: The technical status line marks TS-TEST-HYG-001–009 implemented

- Covers: FR-TEST-HYG-137
- Level: repository check (`sed -n 3p` and `git diff` against `master`)
- Given the change applied on top of `master`
- When `sed -n 3p doc/specifications/test-suite-hygiene/technical.md` is run and compared with line 3 on `master`
- Then:
  - the line starts `Status: TS-TEST-HYG-001–003 implemented (issue #208); TS-TEST-HYG-004–009 implemented (issue #232); TS-TEST-HYG-010–015 implemented (issue #244);`;
  - compared with `master`, the only changes are `ready` → `implemented` in the issue #208 and #232 entries and, if the technical specification stage adds one, a new issue #228 entry at the end.
- Corner cases:
  - the line is still line 3. Line 1 (title), line 2 (blank) and line 4 (`Requirements: … Sources: …`) are unchanged;
  - the other entries keep their ranges, issue numbers and notes. Examples: `TS-TEST-HYG-022 is the manual close of issue #238 after the merge` (issue #241), `the document edit is made by the technical specification stage` (issues #181, #260, #199) and `TS-TEST-HYG-146 updated` (issue #384);
  - the issue #208 and #232 entries keep their ranges (`001–003`, `004–009`, which differ from the functional ranges `004–008`) and their issue numbers;
  - the line does not become `Status: ready (issue #208)` or any other single-status form.

### TC-004: The three places use only the subject's status words

- Covers: FR-TEST-HYG-138
- Level: repository check (`grep -cw` on the three lines)
- Given the change applied, after the development stage (the issue #228 entries of the functional status line and the index row say `implemented`)
- When `grep -cwE 'specified|ready'` is run on the index row, the functional status line and the technical status line, each on its own
- Then each count is 0, and every status in the three lines is `implemented`, `updated` (FR-TEST-HYG-112, FR-TEST-HYG-117, TS-TEST-HYG-146) or `closed by the merge of the issue #238 change` (FR-TEST-HYG-019)
- Corner cases:
  - `already` (issue #238 and #359 notes) and `specification` (issue #358 and #359 notes) do not count, as only whole words are matched;
  - the note of the issue #228 index entry contains no whole word `specified` or `ready`. Quoting them in backticks (as in `` `specified` or `ready` ``) still counts as using them;
  - the issue #228 entries are `FR-TEST-HYG-136–139 implemented (issue #228…` in the functional status line and the index row. A technical issue #228 entry, if there is one, says `implemented`;
  - no other status word (for example `done`, `delivered`, `todo`) appears in any of the three lines;
  - before the development stage, the only `specified` in the functional status line and the index row is in the issue #228 entry. No `ready` appears outside the issue #208 and #232 entries of the technical status line, and none appears after TC-003 is met.

### TC-005: The change touches only the subject's documents and its change folder

- Covers: FR-TEST-HYG-139
- Level: repository check (`git diff` against `origin/master`)
- Given the change applied on top of `origin/master`
- When `git diff --name-only origin/master` and `git diff origin/master -- doc/FUNCTIONAL_SPECIFICATIONS.md` are listed
- Then:
  - the names are only `doc/FUNCTIONAL_SPECIFICATIONS.md`, `doc/specifications/test-suite-hygiene/functional.md`, `doc/specifications/test-suite-hygiene/technical.md` and files under `doc/changes/2026-11-28-1024-issue-228-align-test-suite-hygiene-spec-status-wit/`;
  - the diff of `doc/FUNCTIONAL_SPECIFICATIONS.md` is one removed and one added line, both the **Test suite hygiene** row.
- Corner cases:
  - other subjects' rows and their status lines (for example the specs that say `Status: ready (issue #204)`) are unchanged, as the issue is about this subject only;
  - `doc/workflow/WORKFLOW.md`, the stage prompts under `scripts/factory/` and `doc/todo/issue-208-r1-01.md` are unchanged;
  - the file table of the technical specification is unchanged, as there is no `Status: implemented` step left to drop;
  - the issue #204 and #208 change folders are unchanged;
  - no file is deleted (the change folder has no `delete.txt`, or an empty one);
  - no production Go code, frontend source, `go.mod`, `go.sum`, `package.json`, `API_SPECIFICATION.md`, `_test.go` or `.spec.ts` file is added, changed or deleted.

### TC-006: The existing suites pass and the application does not change

- Covers: FR-TEST-HYG-139
- Level: suite run (deterministic gates: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`). `go test` includes the integration tests with a real SQLite database and HTTP handlers, and Playwright runs the e2e flows.
- Given the change applied on top of `master`
- When the gates run
- Then `go vet ./...` and `go test ./...` succeed, and so do the other gates. The Playwright suite runs the same test list as on `master` and passes, so the operator observes no change in the application.
- Corner cases:
  - the gates give the same result with `go test -count=1 ./...` (no cached results) and when run twice;
  - no test reads the edited documents, so a doc-only diff cannot change a test result. The run proves that nothing else changed.
