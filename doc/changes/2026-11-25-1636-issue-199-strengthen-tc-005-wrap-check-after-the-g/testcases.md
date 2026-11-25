# Test cases: strengthen the TC-005 wrap check after the growing item cell (issue #199)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-094 to FR-TEST-HYG-098). There are no use cases: see [no-use-cases.md](no-use-cases.md). The layout the test proves is FR-RESPONSIVE-024, which this change does not alter.

Conventions:

- **The test file:** `tests/responsive-layout.spec.ts`.
- **TC-005:** the e2e test `TC-005 card values side by side` in the test file, with its widths 775 px and 1055 px, its tabs Amazon and LeBoncoin and its data `sampleItems`.
- **The run:** `npx playwright test tests/responsive-layout.spec.ts -g "TC-005 card values side by side"`.
- **The wrap check:** the part of TC-005 that, for every value starting a later line of a card, compares the free space of the previous line with that value's width (1 px tolerance).
- **Free space of the previous line:** the card's content width minus the widths of the values on the line before, the item value counted at its base width of 14rem, every other value at its rendered width.
- **The issue #165 test cases:** `doc/changes/2026-11-19-2053-issue-165-use-all-space-to-show-item-list-when-the/testcases.md`.
- **Fault injection:** a temporary edit of `src/index.css`, made only for the run of a case, reverted afterwards and never committed.
- The change only touches test code and test documentation. No production code, API, database or user interface changes, and there is no user flow. TC-005 itself is an e2e test (Playwright with the mocked API); the cases below run it. The IDs below are this change's own, not those of the issue #165 test cases.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-094 | TC-001, TC-002, TC-003, TC-004 | — |
| FR-TEST-HYG-095 | TC-002 | — |
| FR-TEST-HYG-096 | TC-001, TC-005 | — |
| FR-TEST-HYG-097 | TC-006 | — |
| FR-TEST-HYG-098 | TC-007 | — |

## Cases

### TC-001: On the repository as changed, TC-005 passes

- Covers: FR-TEST-HYG-094, FR-TEST-HYG-096
- Level: e2e (Playwright, `tests/`)
- Given the change applied and no fault injection
- When the run executes
- Then TC-005 passes at 775 px and 1055 px on both tabs, for every card
- Corner cases:
  - repeated runs (`--repeat-each=3`) give the same result;
  - the Amazon card without a second-hand offer and the one with it both pass the wrap check;
  - at 1055 px, where every value may fit on the item's line, the wrap check has no value starting a later line, or only values that do not fit, and does not fail.

### TC-002: Prices pushed below the item while it would fit fails TC-005 with the full message

- Covers: FR-TEST-HYG-094, FR-TEST-HYG-095
- Level: e2e (Playwright; fault injection)
- Given the change applied, and `src/index.css` temporarily edited so that `.responsive-table tbody tr td:has(> .item-cell)` has `flex: 1 1 40rem`, so that at 775 px Prices starts a new line below the item while 14rem plus the Prices width fits in the card's content width
- When the run executes
- Then TC-005 fails at 775 px, and the failure message names the width and card (for example `775px Amazon card 0`), the tab, the value `Prices`, its width and the free space
- And, without the fault injection, the run passes (TC-001)
- And, with the same fault injection on `master` (the test file before the change), the wrap check passes every 775 px card, which shows the gap the change closes
- Corner cases:
  - the failure comes from the wrap check, not from a same-line or label check: Prices, Amazon second-hand offer and Status stay together on their line, and every value is still labelled;
  - the reported free space is clearly larger than the Prices width, not about 0.

### TC-003: The free space is not the card's right edge minus the previous value's right edge

- Covers: FR-TEST-HYG-094
- Level: code check of the diff of the test file
- Given the change applied
- When the wrap check in TC-005 is read
- Then:
  - the free space is computed from the card's content width and the widths of the values of the previous line, not as `card.right - previous.right`;
  - the item value is counted at a fixed 14rem (converted to pixels with the root font size), not at its rendered width and not at a base width read from the page's styles;
  - the other values of the previous line are counted at their rendered widths;
  - the check still fails unless the free space is smaller than the value's width plus 1 px.
- Corner cases:
  - a previous line without the item value (for example a third line) is counted from its values' rendered widths only;
  - a previous line holding several values (item and actions, or several value cells) counts all of them, not only the last one.

### TC-004: An item basis filling the whole line also fails TC-005

- Covers: FR-TEST-HYG-094
- Level: e2e (Playwright; fault injection)
- Given the change applied, and `src/index.css` temporarily edited so that `.responsive-table tbody tr td:has(> .item-cell)` has `flex: 1 1 100%`, so the item fills its own line while 14rem plus the next value's width fits in the card
- When the run executes
- Then TC-005 fails at 775 px on the value that starts the line after the item, with a message naming the width and card, the tab, that value, its width and the free space
- And, with the same fault injection on `master`, the run passes
- Corner cases:
  - the check fails for any too-large item basis, not only for the 40rem of TC-002;
  - values that wrap because they really do not fit next to a 14rem item are not reported: without fault injection the run passes (TC-001).

### TC-005: TC-005 keeps its name, widths, tabs, data and other checks and messages

- Covers: FR-TEST-HYG-096
- Level: code check of the diff of the test file, and the list output of the e2e run on `master` and with the change
- Given the change applied
- When the diff of the test file is read and `npx playwright test tests/responsive-layout.spec.ts --reporter=list` is run on `master` and with the change
- Then:
  - TC-005 keeps the name `TC-005 card values side by side`, the widths 775 px and 1055 px, the tabs Amazon and LeBoncoin and the data `sampleItems`;
  - the same-line checks (Amazon: Prices, Amazon second-hand offer and Status; LeBoncoin: Prices and Status) and the label checks are kept with their messages;
  - the diff only changes the wrap check of TC-005 (and helpers it alone uses);
  - every test of the test file gives the same result in both runs.
- Corner cases:
  - shared helpers (`layoutProbe`, `sameLine`, `followsOnLine`, `cardLabels`, `mockApi`) keep their behaviour for the other tests; if `layoutProbe` gains a field, the other tests' results are unchanged;
  - no other test of the test file changes.

### TC-006: TC-005 of the issue #165 test cases states that the item counts at 14rem

- Covers: FR-TEST-HYG-097
- Level: document check of the issue #165 test cases
- Given the change applied
- When `git diff master -- <the issue #165 test cases>` is read
- Then:
  - the Then clause of TC-005 about a value starting a later line states that the free space of the previous line counts the item value at its base width of 14rem, not at its grown width, and contains "14rem";
  - the Covers line, Level, Given, When, the other Then clauses and the corner cases of TC-005 are unchanged;
  - no other line of the document changes.
- Corner cases:
  - the coverage table of that document is unchanged.

### TC-007: Only the test file changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-098
- Level: repository check and suite run (deterministic gates: e2e suite, `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs` and the change's diff)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` and `git diff --name-only master -- doc/changes/2026-11-19-2053-issue-165-use-all-space-to-show-item-list-when-the/` are listed and the gates run
- Then the only file listed outside `doc/` is `tests/responsive-layout.spec.ts`, the only file listed in the issue #165 change folder is its `testcases.md` (TC-006 checks it changes only TC-005), and the e2e suite, `go vet ./...`, `go test ./...` and `node --test scripts/*.test.mjs` pass
- Corner cases:
  - `src/` (including `src/index.css`), Go code, `go.mod`, `go.sum`, `package.json`, `package-lock.json`, `playwright.config.*`, `API_SPECIFICATION.md` and the responsive layout specifications are unchanged, so the operator observes no change in the application;
  - the follow-up record `doc/todo/issue-165-r1-01.md` and the issue #165 `tests.lock`, if present, are unchanged;
  - no fault injection from TC-002 or TC-004 remains;
  - the gates give the same result when run twice.
