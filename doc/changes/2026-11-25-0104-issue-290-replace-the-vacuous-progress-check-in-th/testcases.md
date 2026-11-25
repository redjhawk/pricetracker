# Test cases: replace the vacuous progress check in the long whitespace TC-005 token e2e test (issue #290)

Sources: functional specification [test-suite-hygiene](../../specifications/test-suite-hygiene/functional.md) (FR-TEST-HYG-057 to FR-TEST-HYG-060). There are no use cases: see [no-use-cases.md](no-use-cases.md). The Settings modal's behaviour for a whitespace-only input stays as specified by FR-SECRET-014 ([settings-secret-masking](../../specifications/settings-secret-masking/functional.md), UC-SECRET-007); this change does not alter it.

Conventions:

- **The test:** the Playwright test "long whitespace TC-005 a 1025-space token input sends no replacement, shows no message and closes" in `tests/claude-token-settings.spec.ts` (issue #286). It is an existing test, not a case of this catalogue; the IDs below are this change's own.
- **Recorded save requests:** `api.puts`, the bodies of the `PUT /api/v1/settings` requests recorded by the test's `mockApi` route.
- **Run the test:** `npx playwright test tests/claude-token-settings.spec.ts -g "1025-space token input"`.
- **Fault injection:** a temporary edit of `src/components/SettingsModal.tsx` or of a local copy of the test, made only for the run of a case, reverted afterwards and never committed.
- The change only touches one e2e test. No production code, API, database or user interface changes, and there is no user flow, so no Go test and no new Playwright test is written; the cases are a code check, runs of the test (some with fault injection) and the gates.

## Coverage summary

| Requirement | Test cases | Use cases |
| --- | --- | --- |
| FR-TEST-HYG-057 | TC-001 | — |
| FR-TEST-HYG-058 | TC-001, TC-003, TC-004 | — |
| FR-TEST-HYG-059 | TC-002 | — |
| FR-TEST-HYG-060 | TC-005 | — |

## Cases

### TC-001: The vacuous checks are gone and the replacement assertion sits between the Save click and the reopening

- Covers: FR-TEST-HYG-057, FR-TEST-HYG-058
- Level: code check of `tests/claude-token-settings.spec.ts`
- Given the change applied on top of `master`
- When the test is read
- Then:
  - it contains no `Saving settings…` text (in a string or a regular expression) and no loop over `api.puts` (no `for (const put of api.puts)`, `api.puts.forEach` or similar);
  - it contains `expect(api.puts).toEqual([])`, placed after `await expect(dialog).toBeHidden()` and before the second `openSettings(page)` call;
  - it still checks `await expect(page.getByText(/too long/i)).toHaveCount(0)` after `await expect(dialog).toBeHidden()`, with the case-insensitive flag kept;
  - the setup (saved token with hint `wxyz` and `lastRejectedAt` `2026-10-03T20:00:00Z`, the 1025-space fill, the Save click) and the reopening checks are unchanged.
- Corner cases:
  - the removed checks are not merely commented out (no `Saving settings…` text at all in the test);
  - the empty-requests assertion is not placed after the reopening, which could not be told apart from a request sent by the reopened modal, nor before `toBeHidden()`, which could run before a request was recorded;
  - whether `expect(JSON.stringify(api.puts)).not.toContain(" ".repeat(1025))` is kept is a technical choice (it follows from the empty list); either way the test passes TC-002 to TC-004.

### TC-002: A normal run passes with the same outcome as before

- Covers: FR-TEST-HYG-059
- Level: e2e (Playwright, `tests/claude-token-settings.spec.ts`, without fault injection)
- Given the change applied and the e2e environment of the suite
- When the test is run, then again with `--repeat-each=3`, then with the whole file `tests/claude-token-settings.spec.ts`
- Then every run passes: after Save the modal closes, `api.puts` is empty, no "too long" text is shown, and the reopened modal shows an empty "Claude token" input, the text "A token is saved: ••••wxyz." and the warning "Claude rejected this token for an AI review on 03 Oct 2026, 20:00 UTC"
- Corner cases:
  - the other tests of the file (TC-013, the 9000-space session test, TC-016 and the rest) still pass;
  - repeated runs give the same result (no dependence on timing or order).

### TC-003: The test fails when a save request is sent for the whitespace-only input

- Covers: FR-TEST-HYG-058
- Level: e2e (Playwright; fault injection in `src/components/SettingsModal.tsx`)
- Given the change applied, and one of these temporary edits of `save()` at a time:
  1. the `trim()` check of the Claude token is dropped (`else if (claudeValue) change.claudeToken = { value: claudeValue };`), so the 1025 spaces are sent as a replacement;
  2. a whitespace-only Claude token input is sent as a removal (`change.claudeToken = { remove: true }`);
  3. the early return for an empty change is removed, so an empty body `{}` is sent
- When the test is run
- Then each run fails at the `expect(api.puts).toEqual([])` assertion, with the sent body in the reported value; and with the edit reverted the test passes again
- Corner cases:
  - edit 2 sends no 1025-space value, so it would pass a check limited to "the long value is in no request"; it is caught only by the empty-requests assertion;
  - edit 3 sends neither part, so it would pass the removed loop; it is caught by the empty-requests assertion;
  - the failure is reported before the reopening, so it is not caused by a request of the reopened modal;
  - no fault injection remains in the diff (checked by TC-005).

### TC-004: The test still fails when a "too long" text is shown after the modal has closed

- Covers: FR-TEST-HYG-058
- Level: e2e (Playwright; fault injection in a local copy of the test)
- Given the change applied, and a temporary step added right after `await expect(dialog).toBeHidden()` that adds a visible text to the page, run once with each of: "Token too long", "TOO LONG" and "Value is Too Long"
- When the test is run
- Then each run fails at the `page.getByText(/too long/i)` check; and with the temporary step removed the test passes again
- Corner cases:
  - all three casings are caught (the check stays case-insensitive);
  - the text is outside the closed modal, so the check looks at the whole page, not the dialog;
  - no fault injection remains in the diff (checked by TC-005).

### TC-005: Only the 1025-space token test changes outside `doc/`, and the gates pass

- Covers: FR-TEST-HYG-060
- Level: repository check and suite run (deterministic gates: the change's diff, the e2e suite, `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`)
- Given the change applied on top of `master`
- When `git diff --name-only master -- . ':!doc'` and `git diff master -- tests/claude-token-settings.spec.ts` are listed and the gates run
- Then:
  - the only file listed outside `doc/` is `tests/claude-token-settings.spec.ts`;
  - its changed hunk is limited to the body of the 1025-space token test, with no other test, the `mockApi` helper or `openSettings` changed;
  - the e2e suite, `go vet ./...`, `go test ./...` and `node --test scripts/*.test.mjs` pass.
- Corner cases:
  - `src/` (in particular `src/components/SettingsModal.tsx`), the Go code, `API_SPECIFICATION.md`, `package.json` and the lock file are unchanged, so the operator sees no change in the application;
  - no fault injection from TC-003 or TC-004 remains;
  - the gates give the same result when run twice.
