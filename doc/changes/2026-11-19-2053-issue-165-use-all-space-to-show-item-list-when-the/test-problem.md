# Test problem: desktop baseline constants in `tests/responsive-layout.spec.ts` are wrong

Stage: developer (TDD green), attempt 2. The locked spec cannot pass unless the desktop layout changes, and FR-RESPONSIVE-011 and FR-RESPONSIVE-017 forbid that.

## The problem

`tests/responsive-layout.spec.ts` lines 353–357:

```ts
// Baseline from master before issue #165, derived from its CSS: ...
const DESKTOP_PAGE_OFFSET = 64;
const DESKTOP_TABLE_OFFSET = 80;
```

The test cases say the baseline must be "Positions measured on `master` before this change" (testcases.md, Shared definitions, **Baseline**). These constants were derived from the CSS source instead, and that derivation is wrong. On master, at 1056 px and 1440 px:

- `#main-content` is **32 px** from each edge, not 64 px.
- `.tracked-table-container` is **48 px** from each edge, not 80 px.

The failed green run measured this directly: `1056px #main-content left offset 32`. The desktop rules of `src/index.css` are the same as on master.

## Why master is at 32 px

The app's CSS uses Carbon spacing tokens such as `var(--cds-spacing-07)` and `var(--cds-spacing-05)`. In `@carbon/styles` 1.116.0, `css/styles.css` defines these custom properties only inside the `.cds--guidebanner` block (line 22672, tokens at lines 22908–22914). They are not defined at `:root`, so every `var(--cds-spacing-*)` in `src/index.css` is invalid at computed-value time. Each such declaration falls back to `unset`.

As a result, `.page-shell { width: min(100% - 2 * var(--cds-spacing-07), 96rem) }` resolves to `width: auto`, and the page-shell inset is 0. The only desktop gaps are:

- Carbon's `.cds--content` padding: 2rem = 32 px (`#main-content`, `.detail-summary`, admin `.responsive-table-container`).
- The tab panel `.cds--tab-content` padding: 1rem, which is defined, so the tracked table sits at 32 + 16 = 48 px.

This matches the issue author's comment: `app-content` padding plus `cds--tab-content` padding "make 48 px at each side". There is no 16 px page-shell inset in that sum.

## Affected assertions

- `TC-012 desktop tracked items layout unchanged` (lines 1044–1048): expects 64 and 80; master gives 32 and 48.
- `TC-012 desktop item details layout unchanged` (lines 1069–1070): expects 64; master gives 32.
- `TC-012 desktop administrator page layout unchanged` (lines 1087–1088): expects 64; master gives 32.
- `TC-003 card layout and full width switch together at 1056px` (lines 652–653): expects 80 at 1056 px; master gives 48.

To pass these, desktop margins would have to grow from 32/48 px to 64/80 px, for example by defining the spacing tokens at `:root`. That would also turn on every other `--cds-spacing-*` margin, gap and padding in `src/index.css` at desktop width. FR-RESPONSIVE-011 says "Layout and behavior are unchanged from the current release", and FR-RESPONSIVE-017 says "every screen keeps its current margins and layout". So that would be working around a wrong test.

## Requested fix (test stage)

Use the baseline measured on master:

```ts
// Baseline measured on master before issue #165: Carbon's .cds--content padding (2rem) is the only
// page inset (the --cds-spacing-* tokens are not defined globally, so the .page-shell inset resolves to 0);
// the tracked-items tab panel adds 1rem.
const DESKTOP_PAGE_OFFSET = 32;
const DESKTOP_TABLE_OFFSET = 48;
```

Every other use of these constants still holds with these values. `TC-003`'s `gaps.left > 16` holds with 48, and the table still lines up with the search row because both sit in the tab panel.

## Note for the next development attempt (not a test change)

Below 1056 px, the implementation must not rely on the undefined tokens. With the current working tree, the rules for the page gutter (`.page-shell`), card cell padding (`.responsive-table tbody tr td`), the negative list margins, the login padding and others resolve to 0. That causes the inset, gutter and "touches an edge" failures in TC-002, TC-004, TC-006 to TC-011 and in the 320/360 px full-width tests. Fix this inside `@media (max-width: 1055.98px)` only, for example by defining the needed `--cds-spacing-*` values on `:root` there, so desktop stays exactly as on master.

## Other failures in the same run (not locked, outside this change)

These specs failed in the previous run too. They look outdated compared with the current UI, not affected by this CSS change:

- `tests/platform-tabs.spec.ts:141` expects 3 tabs; the page renders 2 (Amazon, LeBoncoin).
- `tests/leboncoin-session-settings.spec.ts:125` expects the open-mode menu to have exactly one entry, "Settings"; the menu also has "Information".
- `tests/app-information.spec.ts` (REV-002, failure/retry, Close/Escape focus, 400 px username) and `tests/leboncoin-session-settings.spec.ts:415`.

If they also fail on master, the green gate cannot pass because of them. They need their own triage.

I could not run the suite in this attempt because the shell sandbox would not start (`socat` not installed). The analysis above comes from reading the Carbon CSS, `src/index.css`, the locked spec and the previous run's output.
