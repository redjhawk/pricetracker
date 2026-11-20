# Test problem — issue #165, developer stage (attempt 3)

## Summary

The issue #165 implementation (`src/index.css`, working tree) passes every case in the locked `tests/responsive-layout.spec.ts`. The last green gate run reported 98 passed and 7 failed, and none of the 7 failures is in that file.

The 7 failures are in test files that this change did not modify. They contradict the current application on `master`, independently of #165. The green gate rejects any test file that differs from the red commit (doc/workflow/WORKFLOW.md), and the developer must not invent functionality. So the developer stage cannot make the full Playwright suite pass. The tests need a separate fix.

## Failing tests and cause (static analysis)

| Test | Cause |
| --- | --- |
| `tests/platform-tabs.spec.ts:141` "both tabs remain available…" | Line 151 expects `getByRole("tab")` to have count 3. Issue #111 removed the "Amazon searches" tab, so only the Amazon and LeBoncoin tabs remain (`TrackedItemsPage.tsx:118-121`). The #111 QA did not run Playwright (`libnspr4.so` missing, see its `qa.md`), so this stale assertion was not caught. The count should be 2. |
| `tests/leboncoin-session-settings.spec.ts:125` "header menu…" | Lines 142-143 expect exactly one menu item, "Settings", in open mode. Since the Information entry (FR-APP-INFO), open mode shows "Information" and "Settings" (`App.tsx:315-320`, `AppMenu.tsx`). `app-information.spec.ts` "open mode shows only the version" requires that entry, so the two specs contradict each other. |
| `tests/leboncoin-session-settings.spec.ts:415` "QA-SET-F01…" | Lines 431-432 press Enter on the menu trigger and then Enter on the first item, expecting the Settings dialog. The first item is now "Information", so "Could not load settings" never appears. |
| `tests/app-information.spec.ts:102` "Close button, close icon and Escape…" | `page.locator(".cds--modal-close")` matches the close icon of every Carbon modal in the DOM (Add item, Delete, Settings and Information are always rendered), so `.click()` fails with a strict-mode violation. It should be scoped to the Information dialog. |
| `tests/app-information.spec.ts:133` "at 400 px a long username wraps…" | `page.locator(".cds--modal-container").boundingBox()` matches every rendered modal container (strict-mode violation). It should be scoped to the Information dialog. |
| `tests/app-information.spec.ts:71` "REV-002…" and `:89` "failure shows an error and reopening retries" | These depend on the number and order of `/api/v1/auth/session` calls and on the auth-required flow (`InformationModal.tsx`, `api/client.ts`). No CSS is involved, and #165 changes neither area. The exact assertion was cut from the gate log, and the shell was unavailable in this run, so this could not be confirmed. |

## Requested action

Fix these specs in a separate change, or exclude them from this change's green gate:

- set the tab count to 2;
- update the expected menu items and the keyboard path to Settings for the Information entry;
- scope the modal locators to the Information dialog;
- recheck the two session-call tests.

None of them concerns issue #165.

## Technical specification note

`doc/specifications/responsive-layout/technical.md` must also describe the CSS that the implementation adds beyond TS-RESPONSIVE-015 to 022 (comment "TS-RESPONSIVE-023" in `src/index.css`):

- the Carbon spacing scale defined on `.page-shell` and `.login-page` below 1056 px, because `@carbon/styles` defines `--cds-spacing-*` only inside `.cds--guidebanner`;
- the `.page-shell .cds--modal` spacing reset;
- hiding the 1 px close-button placeholder after each tab (`.cds--tabs__nav-item--close--hidden`);
- `td:has(> .item-cell) { flex: 1 1 14rem }`, the actions cell `margin-inline-start: auto` and `.item-cell { min-width: 0 }`;
- the two-column `.detail-summary` from 672 to 1055 px;
- `.page-shell` `padding-block-start: spacing-05`.

This run could not write that file, because the write permission was denied.

## Environment limits of this run

The Bash tool could not start: "Sandbox dependencies not available: socat not installed". So `gofmt`, `go vet`, `go test`, `npm run build` and `npx playwright test` were not run in this attempt. `src/index.css` is the previous attempt's implementation, unchanged.
