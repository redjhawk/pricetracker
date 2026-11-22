# Test problem: stale desktop baseline in `tests/responsive-layout.spec.ts`

## Failing locked tests

With the implementation in place, 102 Playwright tests pass and 4 fail. The 4 failures are all desktop-baseline checks (≥ 1056 px):

| Test | Error |
| --- | --- |
| `TC-003 card layout and full width switch together at 1056px` (line 641) | `1056px table left offset 80` (expected 48 ± 1) |
| `TC-012 desktop tracked items layout unchanged` (line 1047) | page offset 64 (expected 32 ± 1) |
| `TC-012 desktop item details layout unchanged` (line 1074) | `#main-content left offset 64` (expected 32 ± 1) |
| `TC-012 desktop administrator page layout unchanged` (line 1092) | `1056px #main-content left offset 64` (expected 32 ± 1) |

## Cause

The constants at lines 366–370 were measured on `master` before commit 248ae54 (`fix(ui): declare the Carbon spacing tokens on :root`, 2026-11-21, already released in v20261121220021 and later tags). The test comment says so:

> the --cds-spacing-* tokens are not defined globally, so the .page-shell inset resolves to 0

248ae54 defines those tokens on `:root`. On desktop, `.page-shell { width: min(100% - 2 * var(--cds-spacing-07), 96rem) }` now resolves, which adds a 32 px inset. The current release's desktop layout is:

- page offset 32 (Carbon `.cds--content` padding) + 32 (`.page-shell` inset) = **64 px**
- tracked table offset 64 + 16 (`.cds--tab-content` padding) = **80 px**

None of this change's CSS applies at ≥ 1056 px. Every new or moved rule is inside `@media (max-width: 1055.98px)` or `(max-width: 671.98px)`. So the base branch produces the same 64/80 px measurements, and these tests fail on `master` too.

Making them pass would mean changing the released desktop layout back to 32/48 px. FR-RESPONSIVE-011 and FR-RESPONSIVE-017 require desktop to stay unchanged from the current release, so the implementation must not do that.

## Fix needed in the tests stage

Re-measure the desktop baseline on the current `master` and update the constants and their comment:

- `DESKTOP_PAGE_OFFSET = 64`
- `DESKTOP_TABLE_OFFSET = 80`
- The comment: `.cds--content` padding (2rem) plus the `.page-shell` inset (`spacing-07`, 2rem), and the tab panel adds 1rem.

Also confirm `LOGIN_FORM_WIDTH`, the header constants and the modal baselines (TC-012 login and TC-013 pass today, so they look current).

## Implementation state

`src/index.css` implements TS-RESPONSIVE-015 to 020 and 023. It is left in place: once the constants are updated, it should pass the whole suite. The TS-RESPONSIVE-023 "spacing scale" and `.page-shell .cds--modal` reset rules are no longer needed, because 248ae54 defines the tokens on `:root`. The technical specification should drop them when development resumes.
