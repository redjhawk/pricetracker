# No new tests for issue #201

This change moves one CSS declaration (`display: block`) from the standalone `.responsive-table td` rule into the `.responsive-table tbody tr td` rule in `src/index.css`. It changes no behaviour. FR-RESPONSIVE-036 and TS-RESPONSIVE-035 allow no file change outside `doc/` except those two rules. So no test file (`*_test.go`, `*.spec.ts`, `*.test.mjs`) may be added or changed, and a committed test would itself break TC-002. There is also no failing test to write: the rendered result is the same before and after the change, so a TDD red step cannot exist.

The test cases are checked this way (see `doc/specifications/responsive-layout/technical.md`, section "Card cell display rule", Tests):

- **TC-001**: `grep -n -A10 'responsive-table' src/index.css`. `display: block` appears exactly once in the `.responsive-table tbody tr td` rule, the rule's other declarations are the same as on `master`, and no media query has a rule whose selector list is exactly `.responsive-table td`.
- **TC-002**: `git diff --name-status master -- . ':!doc'` lists only `M src/index.css`. `git diff -U0 master -- src/index.css` shows only the removed standalone rule and the added `display: block`.
- **TC-003**: an uncommitted Playwright measurement, reusing the `mockApi` fixtures of `tests/responsive-layout.spec.ts`. It reads the bounding box and computed `display` of every `.responsive-table td` (Amazon and LeBoncoin tabs, the no-match row included, and the administrator page) at 320, 360, 775, 1055 and 1280 px, once on a `master` build and once on the branch build. The developer records the results here.
- **TC-004**: the existing gates, unchanged: `go vet ./...`, `go test -count=1 ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`. `tests/responsive-layout.spec.ts` already covers FR-RESPONSIVE-009 and 018 to 024.
- **TC-005**: `git diff master` of `doc/specifications/responsive-layout/functional.md`, the issue #140, #165, #200, #202 and #203 change folders, `doc/todo/`, UC-TI-07 and UC-RL-02.

## TC-003 results

To be filled in by the developer.
