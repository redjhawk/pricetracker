# Frontend implementation: LeBoncoin session settings

Status: implemented; ready for independent review. Not committed.\
Role: frontend developer (separate agent), 2026-10-03.\
Inputs: [change index](index.md) (user decisions and API/R-1 approvals), [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) section "LeBoncoin session settings (approved 2026-10-03)", [header menu functional](../../specifications/app-header-menu/functional.md) / [technical](../../specifications/app-header-menu/technical.md), [settings functional](../../specifications/leboncoin-session-settings/functional.md) / [technical](../../specifications/leboncoin-session-settings/technical.md) (frontend parts).

File ownership: frontend only (`src/**`, `tests/leboncoin-session-settings.spec.ts`, `playwright.config.ts`, `package.json` scripts). No Go, script, or operator documentation file was touched. No dependency was added.

Environment for every command below: Node v22.23.3 (`PATH=/tmp/pricefollower-node22/node_modules/node/bin:$PATH`), Playwright 1.63.0 with `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/google-chrome` (Playwright's own Chromium is not installed). All API traffic in tests is mocked by route interception; the backend endpoint was not exercised. Only synthetic session values (e.g. `Synthetic~Value_123`) are used.

## Step 1: refactoring R-1 (approved "Do it first")

Scope: move `request` and `ApiError` (plus the private `ApiErrorBody` type) from `src/api/items.ts` into a new shared `src/api/client.ts`; `items.ts` imports `request` from it. Code moved verbatim; `request` becomes exported. No behavior change. `ApiError` had no importer before the move.

| File | Change |
| --- | --- |
| `src/api/client.ts` (new) | `ApiErrorBody`, `ApiError`, `request` moved verbatim; `request` exported |
| `src/api/items.ts` | removed the moved code; added `import { request } from "./client";` |

Verification (before and after, separately from the feature):

| Command | Before R-1 (baseline) | After R-1 |
| --- | --- | --- |
| `npm run build` | passed (tsc + vite) | passed (tsc + vite; JS bundle size unchanged at 416.04 kB) |
| `npx playwright test` (only `platform-tabs.spec.ts` registered then) | 8 passed | 8 passed |

## Step 2: feature

### Requirement to change mapping

| Requirement | Implementation | Test (`tests/leboncoin-session-settings.spec.ts`) |
| --- | --- | --- |
| FR-APP-MENU-001, 005 / TS-APP-MENU-001 | `AppMenu` rendered in `HeaderGlobalBar` after **Add item**; header is outside page branches so it shows on list, detail, loading and error views | "header menu…", "narrow viewport…" |
| FR-APP-MENU-002, 003 / TS-APP-MENU-002 | Carbon `OverflowMenu` (`renderIcon={UserAvatar}`, `flipped`, `size="lg"`) with one `OverflowMenuItem` "Settings" that sets `settingsOpen` | "header menu…" (click, Enter, Space, Escape, outside click, toggle, single item, opens dialog, URL and page unchanged, no GET on menu use) |
| FR-APP-MENU-004 / TS-APP-MENU-003 | `aria-label`/`iconDescription` "Application menu"; trigger ref passed to the modal's `launcherButtonRef` | focus on trigger after Escape, after Cancel, after Escape on modal and after successful save |
| FR-APP-MENU-006 / TS-APP-MENU-004 | Carbon modal overlay and focus trap; no extra code | "menu cannot be used while another modal is open" |
| FR-LBC-SET-001 | `SettingsModal`: heading "Settings", one `TextArea` "LeBonCoin session", Save/Cancel, helper text about raw value / `datadome=…` / empty save | "settings modal: loading…" |
| FR-LBC-SET-002 | GET on every `open` transition; `InlineLoading` "Loading settings…"; TextArea and Save disabled until loaded | "settings modal: loading…", "load failure… reopening retries" (2 GETs) |
| FR-LBC-SET-003 | error `InlineNotification` "Could not load settings" with server message; TextArea not rendered; Save disabled; Cancel works; reopen retries | "load failure…" |
| FR-LBC-SET-004 | TextArea value = `value ?? ""`, unmasked | "settings modal…", hint cases (incl. `none` → empty) |
| FR-LBC-SET-005, 006 | Text sent exactly as typed with the loaded `revision`; server parses. `400 INVALID_SESSION` message shown as TextArea `invalidText`; input kept; editing clears it | "saving sends the text and revision…", "invalid session…" |
| FR-LBC-SET-007 | `200` closes the modal; next open reloads | "saving sends the text…" |
| FR-LBC-SET-008 | Empty text sent as `""` | "saving an empty field clears the session" |
| FR-LBC-SET-009 | Save labelled "Saving…" and disabled, TextArea disabled, `InlineLoading`, close/Escape/Cancel ignored, ref guard prevents a second PUT | "saving in progress…" (one PUT) |
| FR-LBC-SET-010 | other errors: error notification "Could not save settings", input kept, Save re-enabled | "server failure keeps input and allows retry" |
| FR-LBC-SET-011 | close resets component state (effect cleanup on `open` → false); no PUT | "cancel and Escape discard edits…" |
| FR-LBC-SET-014, 015 | Warning `InlineNotification` (text, not colour only) per technical spec order: revoked, expired, active+rejected, active+failed; none otherwise. Dates with the same `Intl.DateTimeFormat("en-GB", …, UTC)` as `ItemDetail.tsx` | 7 "hint for … session" cases |
| FR-LBC-SET-016 | Reload on every open | covered by reopen tests |
| FR-LBC-SET-017 | `409 SESSION_CHANGED`: non-dismissible warning "Session changed" with the specified text; input kept and editable; Save disabled until reopened | "changed session warns…" |
| FR-LBC-SET-018 | No browser storage; state cleared on close | "session value is not kept in browser storage" |
| FR-LBC-SET-019 | Carbon `Modal` (focus trap, Escape, `launcherButtonRef`), labelled TextArea, `invalidText` on field, `sm` modal with wrapping TextArea | "narrow viewport…" (400×800: no overlap, menu in viewport, Save in viewport, no horizontal scroll with a 600-character value) |

FR-LBC-SET-012 and 013 are backend behavior; the frontend has no part beyond calling the endpoints.

### Changed files

| File | Change |
| --- | --- |
| `src/api/settings.ts` (new) | `LeboncoinSessionSettings` type, `getLeboncoinSession`, `saveLeboncoinSession` (unwrap `{ session }`) using the shared `request` |
| `src/components/AppMenu.tsx` (new) | header `OverflowMenu` with the Settings entry |
| `src/components/SettingsModal.tsx` (new) | settings modal, states, hints |
| `src/App.tsx` | `settingsOpen` state, `menuButtonRef`, render `AppMenu` and `SettingsModal` |
| `tests/leboncoin-session-settings.spec.ts` (new) | 20 Playwright tests with mocked API |
| `playwright.config.ts` | `testMatch` now also includes the new spec |
| `package.json` | script `test:settings` (as listed in the technical verification plan) |

`src/index.css` was not changed: screenshots at 1280 and 400 px show the app's header is light and the Carbon icon is visible and aligned without extra styling.

### RED / GREEN evidence

RED (spec written before any feature code; R-1 already applied):

```bash
PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/google-chrome npm run test:settings
```

Result: 20 failed, 0 passed. Every test failed waiting for the "Application menu" button (header menu and settings modal did not exist). One assertion was then edited before implementing (loading state: TextArea disabled and empty instead of absent) to match TS-LBC-SET-006 exactly.

GREEN:

```bash
npm run build
npx playwright test tests/leboncoin-session-settings.spec.ts --timeout 15000
```

First run: build passed; 19 passed, 1 failed — a test-locator ambiguity (`getByText("Session changed")` also matched the substring in the notification body). Fixed the locator with `exact: true` (no product code change). Final run of the full suite:

```bash
npm run build                                   # passed (tsc -b && vite build)
PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/google-chrome npx playwright test
```

Result: 28 passed (20 new settings tests + 8 existing platform-tabs tests).

Visual check: a throwaway script (scratchpad, not in the repository) captured the header, open menu and modal at 1280×800 and 400×800 against a temporary Vite server with mocked API; menu fits on screen, long values wrap, warning hint readable.

### Deviations and limitations

- **Trigger ref:** the installed `@carbon/react` 1.117.0 `OverflowMenu` export is a feature-flag wrapper whose type declares `RefAttributes<HTMLDivElement>`, although at runtime (flag `enable-v12-overflowmenu` off) `ref` reaches the trigger button. To avoid a type cast, `AppMenu` uses the documented `innerRef` prop ("The ref to the overflow menu's trigger button element"), which Carbon marks deprecated in favour of `ref`. Focus-return tests confirm it targets the trigger. Reviewer may prefer `ref` with a cast; behavior is identical.
- The date formatter is duplicated from `ItemDetail.tsx` (a 9-line constant) rather than extracted, to avoid unapproved refactoring.
- Tests use a mocked API only; integration with the real Go endpoint and real cookie values are left to QA. Server-side parsing/validation (FR-LBC-SET-005/006 rules) is not tested in the frontend by design (no client parser).
- Screen-reader announcement was not tested (no assistive technology available); Carbon roles and labels were relied on.
- Carbon MCP was not available; component APIs were checked against the installed package type declarations.

## QA fixes (QA-SET-F01, QA-SET-F02)

Inputs: [decisions.md](decisions.md) (both critical, disposition fix), [qa.md](qa.md) reproductions. Only `src/components/SettingsModal.tsx` and `tests/leboncoin-session-settings.spec.ts` changed. The Add item and Delete modals were not touched and needed no change for these findings. Whether they contain focus correctly was not tested; that is still the QA retest note in the decisions record.

### Root causes

1. **No initial focus (F01).** Carbon `Modal` focuses `selectorPrimaryFocus` (default `[data-modal-primary-focus]`), or otherwise the primary button, and it does this only when `open` changes. At open, Save and the TextArea are disabled (loading), so nothing received focus and it stayed on `BODY`. A Tab from `BODY` then reached the header, and Enter on the header **Add item** stacked a second modal (F02).
2. **Unreliable wrap at the edges (found while writing the regression test).** Carbon's default focus wrap uses hidden "Focus sentinel" spans, and the wrap runs in a `setTimeout` after blur. With quick Tab/Shift+Tab presses, focus sometimes stayed on a sentinel outside `[role=dialog]`, or moved on to `BODY`, and the wrap never ran. With about 150 ms between key presses, it wrapped correctly.
3. **First attempt rejected.** I first tried the close icon button as the initial focus target. With that, Escape no longer closed the modal ("browser storage" test failed). The icon button's tooltip opens on focus and consumes the first Escape, which conflicts with FR-LBC-SET-019.

### Changes (`SettingsModal.tsx`)

- `selectorPrimaryFocus=".cds--modal-footer .cds--btn--secondary"`: on open, Carbon focuses **Cancel**. Cancel is enabled in every state reachable on open: loading, loaded and load error. Once loaded, focus stays in the dialog because Cancel remains enabled.
- The Settings `Modal` is wrapped in Carbon `<FeatureFlags enableFocusWrapWithoutSentinels>`. This enables Carbon's documented keyboard-based focus trap (synchronous Tab/Shift+Tab wrap with no sentinels) for this modal only. No custom global mechanism was added, and other modals are unaffected.
- Not changed: loading, disabled-while-loading, Escape, saving guard and focus return (`launcherButtonRef`).

### Regression tests (written first)

- `QA-SET-F01: focus enters and stays in the dialog while loading, loaded and on error`:
  - The GET is delayed. Focus must be inside `[role=dialog]` during loading, after loading, and in the load-error state.
  - In each state, 5 Tab and 6 Shift+Tab presses never leave the dialog.
  - Escape returns focus to the trigger.
- `QA-SET-F02: header controls are unreachable behind Settings; no stacked modal`:
  - Six rounds of Tab, then Enter on any focused button other than Save, Cancel or Close.
  - Focus never lands inside `header` (brand, **Add item**, Application menu).
  - Exactly one visible `[role=dialog]` remains at the end.
- The reverse case (Add item open, menu unreachable) was already covered by "menu cannot be used while another modal is open", which uses a pointer click.
- One existing test was made deterministic: "session value is not kept in browser storage" now waits for the dialog to be hidden after Escape before inspecting the page.

### RED / GREEN

Environment: Node v22.23.3, `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/google-chrome`.

- **RED** (`npx playwright test tests/leboncoin-session-settings.spec.ts -g "QA-SET" --timeout 15000`, run against the unfixed `SettingsModal`): 2 failed. Both failed in `expectFocusInDialog` (`Received: false`): focus was outside the dialog after open (F01) and after the first Tab (F02). The first RED run of the F02 test also used a header locator that did not resolve. That locator was replaced with a `closest("header")` check, and RED was re-run with the fix temporarily removed: 2 failed, as above.
- **Intermediate:**
  - With the close button as the focus target: `--repeat-each 3` gave failures in the storage test (Escape did not close) and in the F01 test.
  - With Cancel as the focus target, sentinel wrap: the F01 test failed 3 of 3 times, with focus stuck on a sentinel. There was also one 30 s timeout in "saving an empty field clears the session", which did not recur after the final change.
- **GREEN:**
  - `npx playwright test tests/leboncoin-session-settings.spec.ts --repeat-each 3`: 66 passed.
  - `npm run build`: passed.
  - `npx playwright test` (full suite): 30 passed (22 settings + 8 platform-tabs).

### Limitations

- The tests use a mocked API in headless Chrome. QA must retest UI-SET-19a, UI-SET-11a/c and UI-MENU-06 against the rebuilt application.
- Screen-reader behavior is still untested.
- Focus on open goes to **Cancel**, not the field. Carbon applies initial focus only when the modal opens, while the field is still disabled. Moving focus to the field after loading would need extra focus code, which the decision did not require.
- `enableFocusWrapWithoutSentinels` is a Carbon feature flag in 1.117.0. It is applied only to this modal.
