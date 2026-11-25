# Red phase skipped for issue #199

This change is the test code itself. The e2e test `TC-005 card values side by side` in `tests/responsive-layout.spec.ts` checks the card layout that issue #165 shipped (FR-RESPONSIVE-024). The technical specification (`doc/specifications/test-suite-hygiene/technical.md`, "TC-005 wrap check counts the item at its base width (issue #199)") says no production code changes. The test passes on the `master` code before and after the edit, so the red gate would always fail. This file skips it. The development stage writes no code.

The fault injections below show that the new check can fail where the old one could not.

## Test edited

`git diff master -- tests/responsive-layout.spec.ts` (14 insertions, 4 deletions) changes only:

- `const ITEM_BASIS_REM = 14;`, declared after `cardLabels` with a comment naming the `td:has(> .item-cell)` basis of FR-RESPONSIVE-024;
- the `Window.layoutProbe.cells` type, now `{ left: number; right: number; cells: CardCell[] }`, and the `cells` function of `layoutProbe`, which now also returns `left: box.left + parseFloat(style.paddingLeft) + parseFloat(style.borderLeftWidth)`, mirroring `right` (and its comment);
- in TC-005, `const rem = await page.evaluate(() => parseFloat(getComputedStyle(document.documentElement).fontSize));` after each `setViewportSize`;
- in TC-005, the wrap check:

```ts
        card.cells.forEach((current, position) => {
          if (position === 0 || followsOnLine(card.cells[position - 1], current)) return;
          let start = position - 1;
          while (start > 0 && followsOnLine(card.cells[start - 1], card.cells[start])) start--;
          const used = card.cells.slice(start, position)
            .reduce((sum, cell, offset) => sum + (start + offset === 0 ? ITEM_BASIS_REM * rem : cell.width), 0);
          const free = card.right - card.left - used;
          expect(free, `${what}: free space ${free.toFixed(1)} before value ${current.label || position} (width ${current.width.toFixed(1)})`).toBeLessThan(current.width + 1);
        });
```

It replaces `const free = card.right - card.cells[position - 1].right;` and the old message. Two comment lines explain the base sizes. The test name, the widths, tabs, data, row count, same-line and label checks with their messages, `sameLine`, `followsOnLine`, `cardLabels`, `mockApi` and every other test are unchanged. No test is added or removed.

| Test case | Proved by |
| --- | --- |
| TC-001 | Run 1: TC-005 passes at 775 and 1055 px, on both tabs and for every card, including both Amazon cards. |
| TC-002 | Run 2: with the 40rem item basis, the new check fails at `775px Amazon card 0` on `Prices`, with a free space of 551.0 px against a width of 175.8 px. With the same injection, the old check passes every 775 px card (see the note in run 2). |
| TC-003 | Code check of the wrap check above. `card.right - card.cells[position - 1].right` is gone. The free space is `card.right - card.left` minus every value of the previous line, found by walking back with `followsOnLine`. The first cell counts as `ITEM_BASIS_REM * rem`, with the constant 14 and not a style read. The other values count at `width`. `toBeLessThan(current.width + 1)` is kept. |
| TC-004 | Run 3: with the 100% item basis, the new check fails at 775 px on `Prices`, the value starting the line after the item, with the same message form. The old check passes the whole test with that injection. |
| TC-005 | Run 4: the list output of `tests/responsive-layout.spec.ts` gives the same 50 test names and results (50 passed) with the old and the new wrap check. Code check: the diff is limited to the items listed above. |
| TC-006 | The technical specification stage already made the edit (commit 91a37ac). `git diff master` of the issue #165 `testcases.md` is one line, the TC-005 Then line, which now contains "14rem". |
| TC-007 | Run 5 and "Gates" below: outside `doc/`, only `tests/responsive-layout.spec.ts` differs from `master`, and `src/index.css` is identical to `master` after the reverts. |

## Runs and fault injections

`R` is `npx playwright test tests/responsive-layout.spec.ts -g "TC-005 card values side by side"`. Each injection was a temporary edit of `.responsive-table tbody tr td:has(> .item-cell) { flex: 1 1 14rem; }` in `src/index.css`. It was followed by the runs and then a revert, and none of it is in the diff. "The old check" means the test file with only its wrap check temporarily set back to the `master` lines (`free = card.right - card.cells[position - 1].right` and the `master` message). The added `left` field, `rem` read and constant are not read by that check.

1. **No injection.** `R`: 1 passed. `R --repeat-each=3`: 3 passed, both before the injections and again after the last revert.
2. **`flex: 1 1 40rem`.**
   - New check: **FAIL** at 775 px, from the wrap check: `Error: 775px Amazon card 0: free space 551.0 before value Prices (width 175.8)`, `Expected: < 176.75`, `Received: 551`. The same-line and label checks of that card ran before it and passed.
   - Old check: every 775 px card passes, on both tabs, so the wrap check misses the gap. The test then fails later at 1055 px, on a different check: `Error: 1055px Amazon card 0: Prices and Status on the same line`. The reason is that with a 40rem item, Prices and Status no longer fit on one line at that width. So the whole `master` test does not pass with this injection, contrary to what the TC-002 wording expects. Still, its wrap check lets through at 775 px exactly the fault that the new check reports.
3. **`flex: 1 1 100%`.**
   - Old check: **PASS** (1 passed). The gap the change closes.
   - New check: **FAIL** at 775 px: `Error: 775px Amazon card 0: free space 551.0 before value Prices (width 175.8)`. Prices is the value that starts the line after the item.
4. **Whole file.** `npx playwright test tests/responsive-layout.spec.ts --reporter=list`: 50 passed with the new check, and 50 passed with the old check. The test names are the same.
5. **Revert.** `src/index.css` is back to `flex: 1 1 14rem;`, and `git status` lists only `tests/responsive-layout.spec.ts` as modified outside `doc/`. `R --repeat-each=3`: 3 passed.

## Gates

Run once in this stage, after the reverts:

- `node --test scripts/*.test.mjs`: 50 passed.
- `npx playwright test`, `go vet ./...` and `go test ./...`: see below.

`npx tsc` was not run in this stage because the command needed approval. Playwright transpiles the spec, and every run above compiled and ran it.
