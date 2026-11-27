# Issue #222 Put `await close()` on its own line in the TC-007 Information modal test

**Problem:** In `tests/app-information.spec.ts` line 378, the Tab loop and `await close();` are on one line, separated by stray spaces: `for (...) await page.keyboard.press("Tab");      await close();`. This makes `close()` look like part of the loop body.

**Suggested work:** Move `await close();` to its own line after the loop, at the same indentation. Do not change behaviour.

From review comment R1-02 on #221 (`doc/todo/issue-215-r1-02.md`).

## Comments by trusted authors

