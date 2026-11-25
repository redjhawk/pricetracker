# Red phase skipped for issue #290

This change is the test code itself. The 1025-space token test in `tests/claude-token-settings.spec.ts` tests behaviour that issue #286 already shipped (FR-SECRET-014, TS-SECRET-019). The technical specification (`doc/specifications/test-suite-hygiene/technical.md`, "The 1025-space token test asserts that no save request is sent") says no production code changes. The test passes on the `master` code before and after the edit, so the red gate would always fail. This file skips it. The development stage writes no code.

The fault injections below show that the new checks can fail.

## Test edited

In `long whitespace TC-005 a 1025-space token input sends no replacement, shows no message and closes`, these lines after `await expect(dialog).toBeHidden()`

```ts
await expect(page.getByText(/too long|Saving settings…/i)).toHaveCount(0);
for (const put of api.puts) {
  expect(put.claudeToken?.remove).toBeFalsy();
  expect(put.claudeToken?.value?.trim() ?? "").toBe("");
}
expect(JSON.stringify(api.puts)).not.toContain(" ".repeat(1025));
```

are replaced, in place, by

```ts
await expect(page.getByText(/too long/i)).toHaveCount(0);
expect(api.puts).toEqual([]);
```

The `JSON.stringify` check is dropped: it follows from the empty list. The setup, the fill, the Save click, `toBeHidden()` and the three reopening checks are unchanged. No other line of the file changes. No test is added or removed.

| Test case | Proved by |
| --- | --- |
| TC-001 | `git diff master -- tests/claude-token-settings.spec.ts` is the one hunk above, inside the 1025-space token test (lines 101–102 after the change). In that test, `Saving settings…` no longer appears and no loop over `api.puts` is left; `expect(api.puts).toEqual([])` (line 102) sits after `toBeHidden()` (line 100) and before the second `openSettings(page)` (line 103). The only remaining `Saving settings…` in the file is in the TC-013 progress test (line 72), which is out of scope. |
| TC-002 | Run 1 below: the test alone passes (1 passed), `--repeat-each=3` passes (3 passed), and the whole file passes (13 passed). Again after the reverts (3 passed). |
| TC-003 | Fault injections 2.1–2.3 below. |
| TC-004 | Fault injections 3.1–3.3 below. |
| TC-005 | `git diff --name-only master -- . ':!doc'` lists only `tests/claude-token-settings.spec.ts`, and its diff is the one hunk above; `src/components/SettingsModal.tsx` is identical to `master` after the reverts. Gates: see "Gates" below. |

## Runs and fault injections

Run command: `npx playwright test tests/claude-token-settings.spec.ts -g "1025-space token input"`. Each injection was a temporary edit, then the run, then a revert. None is in the diff.

1. **Normal run.** 1 passed; `--repeat-each=3`: 3 passed; `npx playwright test tests/claude-token-settings.spec.ts`: 13 passed.
2. **Save request sent** (edits of `save()` in `src/components/SettingsModal.tsx`):
   1. `else if (claudeValue.trim()) change.claudeToken = …` → `else if (claudeValue) change.claudeToken = …`.
      - **FAIL** at line 102, `expect(api.puts).toEqual([])`: received `[{ "claudeToken": { "value": "<1025 spaces>" } }]`.
   2. `else if (claudeValue) change.claudeToken = { remove: true };` appended after that line.
      - **FAIL** at line 102: received `[{ "claudeToken": { "remove": true } }]`. The old loop would have caught it, but a check limited to "the long value is in no request" would not.
   3. The early return `if (!change.leboncoinSession && !change.claudeToken) { onClose(); return; }` removed.
      - **FAIL** at line 102: received `[{}]`. The old loop and the `JSON.stringify` check both passed on this body.

   Each failure is reported before the reopening.
3. **"Too long" text after the modal has closed** (temporary line in the test, right after `await expect(dialog).toBeHidden()`):

   ```ts
   await page.evaluate((text) => { const note = document.createElement("p"); note.textContent = text; document.body.append(note); }, "<text>");
   ```

   1. `Token too long`: **FAIL** at `page.getByText(/too long/i)`, `toHaveCount`: `Expected: 0`, `Received: 1`.
   2. `TOO LONG`: **FAIL**, same message.
   3. `Value is Too Long`: **FAIL**, same message.
4. **Revert.** After the last revert, `--repeat-each=3` gives 3 passed, and `git diff --name-only master -- . ':!doc'` lists only `tests/claude-token-settings.spec.ts`.

## Gates

Run once in this stage after the reverts: `npx playwright test`, `node --test scripts/*.test.mjs`, `go vet ./...`, `go test ./...`. Results: see below.
