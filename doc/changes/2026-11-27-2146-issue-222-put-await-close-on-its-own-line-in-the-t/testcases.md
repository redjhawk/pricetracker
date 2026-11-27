# Test cases: `await close()` on its own line in TC-007 (issue #222)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-108 to FR-TEST-HYG-110). There are no use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **TC-007 test:** `TC-007 (#215) closing after fast tabbing returns focus to the menu control` in `tests/app-information.spec.ts`.
- **Tab-and-close line:** the line on `master` holding both `for (let presses = 0; presses <= count; presses += 1) await page.keyboard.press("Tab");` and `await close();`, separated by spaces.
- **Loop line:** exactly 6 spaces followed by `for (let presses = 0; presses <= count; presses += 1) await page.keyboard.press("Tab");`. **Close line:** exactly 6 spaces followed by `await close();`.
- The change only reformats one line of an e2e test. No production code, API, database or user interface changes and there is no new user flow, so the only e2e case is the run of the existing spec file. The IDs below are this change's own, not those in the test names.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-108 | TC-001, TC-002 | — |
| FR-TEST-HYG-109 | TC-002, TC-003 | — |
| FR-TEST-HYG-110 | TC-002, TC-003, TC-004 | — |

## Cases

### TC-001: The loop and `await close()` are on two lines with the original indentation

- Covers: FR-TEST-HYG-108
- Level: code check of `tests/app-information.spec.ts` (`grep`)
- Given the change applied
- When `grep -n 'press("Tab");  *await close();' tests/app-information.spec.ts`, `grep -n ' $' tests/app-information.spec.ts` and `grep -n 'await close();' tests/app-information.spec.ts` are run, and the TC-007 test is read
- Then the first grep prints nothing (the tab-and-close line no longer exists)
- And, in the TC-007 test, the close line comes directly after the loop line, and neither line is printed by the trailing-space grep
- And the loop line holds no statement other than the Tab loop, and the close line holds nothing but `await close();`
- Corner cases:
  - indentation with a tab character, or with 4 or 8 spaces instead of 6, fails the check;
  - `await close();` wrapped in braces with the loop (`{ … }`) or moved before the loop fails the check;
  - a blank line between the loop line and the close line fails the check (the close line must come directly after);
  - `await close();` appears only once in the TC-007 test.

### TC-002: The diff of the spec file replaces exactly the one line with the two lines

- Covers: FR-TEST-HYG-108, FR-TEST-HYG-109, FR-TEST-HYG-110
- Level: code check (`git diff --unified=0 origin/master -- tests/app-information.spec.ts`)
- Given the change applied on top of `master`
- When the diff is listed
- Then it has one hunk, which removes exactly one line, the tab-and-close line, and adds exactly two lines, the loop line then the close line
- And the TC-007 test keeps its name, its `test.setTimeout(90_000);`, its three rounds, its three closers (Escape, footer Close, icon close) and its checks after the close, in the same order
- Corner cases:
  - no other test of the file, no import and no helper is changed;
  - no whitespace-only change elsewhere in the file (`git diff --stat` reports 2 insertions and 1 deletion);
  - the loop still runs `count + 1` times (`presses <= count`), so the number of Tab presses is unchanged.

### TC-003: The Information modal e2e tests pass as before

- Covers: FR-TEST-HYG-109, FR-TEST-HYG-110
- Level: e2e (Playwright, `npx playwright test tests/app-information.spec.ts`)
- Given the change applied and the Playwright environment of the repository
- When the spec file runs, then the TC-007 test alone runs twice (`npx playwright test tests/app-information.spec.ts -g "TC-007" --repeat-each=2`)
- Then all tests of the file pass, including the TC-007 test: in each of the 3 rounds and for each closer, after Tab is pressed one more time than the modal has tabbable controls and the modal is closed once, the modal is hidden, the Application menu button has the focus, no menu item is shown, the URL is `/`, the empty list heading is visible and no focus left the modal
- Corner cases:
  - repeated runs pass each time (the change adds no flakiness);
  - each closer is called once per iteration, after the loop, not once per Tab press;
  - the test stays within its 90-second timeout.

### TC-004: Only the spec file changes outside `doc/`, and the frontend gates pass

- Covers: FR-TEST-HYG-110
- Level: repository check and gates (frontend type check, lint and build of the pipeline)
- Given the change applied on top of `master`
- When `git diff --name-status origin/master -- . ':!doc'` is listed and the frontend gates run (type check and build with `npm run build`, and the repository's lint)
- Then the only entry is `M tests/app-information.spec.ts`, and the type check, lint and build succeed
- Corner cases:
  - `src/`, `web/`, Go code, `package.json`, lock files, `playwright.config.ts` and `API_SPECIFICATION.md` are unchanged, so the operator sees no change in the application;
  - the gates give the same result when run twice.
