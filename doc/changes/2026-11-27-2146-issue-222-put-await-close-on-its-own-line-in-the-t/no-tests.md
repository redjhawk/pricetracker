# Red phase skipped for issue #222

This change only reformats test code. In `TC-007 (#215) closing after fast tabbing returns focus to the menu control` (`tests/app-information.spec.ts`), a `for` without braces takes one statement as its body, so `await close();` already ran once, after the loop. Only the layout was misleading. The technical specification (`doc/specifications/test-suite-hygiene/technical.md`, "`await close()` on its own line in TC-007 (issue #222)") changes no production code. TC-007 passes on `master` before and after the edit, so the red gate would always fail. This file skips it. The development stage writes no code.

## Test edited

Line 378 of `tests/app-information.spec.ts`

```ts
      for (let presses = 0; presses <= count; presses += 1) await page.keyboard.press("Tab");      await close();
```

is replaced, in place, by

```ts
      for (let presses = 0; presses <= count; presses += 1) await page.keyboard.press("Tab");
      await close();
```

Each line is 6 spaces followed by the text, with no trailing space. No braces and no blank line are added. No other line of the file changes, and no test is added or removed.

| Test case | Proved by |
| --- | --- |
| TC-001 | `grep -n 'press("Tab");  *await close();'` and `grep -n ' $'` on the file print nothing. `grep -n 'await close();'` prints line 120 (another test) and line 379, directly after the loop line 378. `^ {6}(<loop>\|await close\(\);)$` matches exactly lines 378 and 379. |
| TC-002 | `git diff --unified=0 origin/master -- tests/app-information.spec.ts` is one hunk `@@ -378 +378,2 @@`. It removes the tab-and-close line and adds the loop line, then the close line. `git diff --stat` reports `1 file changed, 2 insertions(+), 1 deletion(-)`. |
| TC-003 | Before the edit, `npx playwright test tests/app-information.spec.ts` gave 20 passed. After it, the same command gave 20 passed, and `-g "TC-007" --repeat-each=2` gave 2 passed (8.3 s each, within the 90 s timeout). `--list` gives the same 20 tests at the same lines before and after (TC-007 stays at line 362). The fault injection below shows that a close inside the loop fails the test. |
| TC-004 | `git diff --name-status origin/master -- . ':!doc'` lists only `M tests/app-information.spec.ts`. `npm run build` (`tsc -b && vite build`) succeeds. The repository has no lint script, and `tsconfig.json` does not include `tests/`. |

## Fault injection (TC-003)

This was a temporary edit, then a revert. It is not in the diff.

1. **Close inside the loop.** Line 378 became `for (let presses = 0; presses <= count; presses += 1) { await page.keyboard.press("Tab"); await close(); }`. Then `npx playwright test tests/app-information.spec.ts -g "TC-007"`:
   - **FAIL** at line 380, `await expect(menu).toBeFocused()`: `expect(locator).toBeFocused() failed`, `Locator: getByRole('button', { name: 'Application menu', exact: true })`, `Expected: focused`, `Received: inactive`, `Timeout: 5000ms`.
2. **Revert.** After the two-line form was restored with `Edit`, `npx playwright test tests/app-information.spec.ts -g "TC-007"` gave 1 passed.
