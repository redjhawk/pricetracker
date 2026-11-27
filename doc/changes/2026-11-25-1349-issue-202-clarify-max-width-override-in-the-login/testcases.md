# Test cases: clarify the max-width override in the login form rule (issue #202)

Sources: functional specification [responsive-layout](../../specifications/responsive-layout/functional.md) (FR-RESPONSIVE-031 to FR-RESPONSIVE-033). No use cases: see [no-use-cases.md](no-use-cases.md). The login page layout below 1056 px that does not change is described by step 6 of UC-RL-01, which this change does not modify. No new `UC-` IDs are defined.

Conventions:

- **Base:** `origin/master` (or `master`), compared with the change applied on top of it.
- **The media block:** the `@media (max-width: 1055.98px)` block of `src/index.css` that holds the login form rule (the block that also holds `.login-form .cds--inline-notification`).
- **The button selector:** `.login-form .cds--btn[type="submit"]`.
- The change only edits a stylesheet rule and its comment. FR-RESPONSIVE-033 allows no other file change outside `doc/`, so no new Go or Playwright test code is committed: the cases are repository checks (`grep`, `git diff`, reading `src/index.css`), runs of the existing suites, and one uncommitted width measurement (TC-004). No API, database or user interface changes; invalid input, empty results and repeated actions do not apply. Narrow screens and keyboard use are covered by the existing responsive-layout e2e suite (TC-005).

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-RESPONSIVE-031 | TC-001, TC-002 |
| FR-RESPONSIVE-032 | TC-004, TC-005 |
| FR-RESPONSIVE-033 | TC-003, TC-005, TC-006 |

## Cases

### TC-001 — `max-width: none` for the login page is set only on the submit button, with a "20rem" comment

- Covers: FR-RESPONSIVE-031
- Level: repository check (file content of `src/index.css`)
- Given the change applied on top of the base
- When every rule of `src/index.css` whose selector list contains `.login-form` is listed (for example `grep -n -B3 -A4 'login-form' src/index.css`)
- Then:
  - exactly one rule sets `max-width: none` with a selector containing `.login-form`, and its selector list is the button selector alone;
  - that rule is inside the media block and is immediately preceded by a CSS comment (`/* ... */`, only whitespace between the comment end and the selector) containing `20rem`;
  - no rule whose selector list includes `.login-form` alone (with no descendant part) sets `max-width` or `max-inline-size`, inside or outside a media query;
- Corner cases:
  - the comment is attached to the button rule, not to a neighbouring rule (the inline-notification rule above or the `.item-cell` rule below);
  - the comment mentions Carbon's button cap (`20rem`), matched case-sensitively as written in the requirement;
  - the `.login-form .cds--inline-notification` rule (`min-inline-size: 0; max-inline-size: none`) is unchanged: it targets the notification, not `.login-form` itself;
  - on the base, the check fails: the rule with selectors `.login-form,` and the button selector sets `max-width: none` on both, with no comment.

### TC-002 — Below 1056 px the form and the submit button still get `width: 100%`

- Covers: FR-RESPONSIVE-031
- Level: repository check (file content of `src/index.css`)
- Given the change applied on top of the base
- When the media block is read
- Then:
  - a rule in the media block sets `width: 100%` on `.login-form`;
  - a rule in the media block sets `width: 100%` on the button selector (the same rule as in TC-001 or another one);
  - the top-level `.login-form { width: min(100%, 24rem); }` rule outside the media block is unchanged
- Corner cases:
  - splitting into two rules or keeping one rule for `width` and a separate one for `max-width` are both accepted, as long as TC-001 holds;
  - no new media query or breakpoint is introduced for the login form (the 1055.98px boundary stays the only one used by these rules).

### TC-003 — Outside `doc/`, the diff touches only the login form rule

- Covers: FR-RESPONSIVE-033
- Level: repository check (`git diff`)
- Given the change applied on top of the base
- When `git diff --name-status <base> -- . ':!doc'` and `git diff -U0 <base> -- src/index.css` run
- Then:
  - the first command lists only `M src/index.css`;
  - every changed hunk of `src/index.css` lies within the media block, between the end of the `.login-form .cds--inline-notification` rule and the start of the `.item-cell` rule, and only adds or changes the login form selectors, their `width` / `max-width` declarations and the comment
- Corner cases:
  - no `.tsx`, `.ts`, `.spec.ts`, `_test.go`, `.go`, `go.mod`, `go.sum`, `package.json`, `package-lock.json` or `API_SPECIFICATION.md` file changes;
  - no other rule of `src/index.css` changes, including the top-level `.login-form` rule (line 64 on the base) and the inline-notification rule;
  - no formatting-only change elsewhere in `src/index.css` (whitespace, reordering).

### TC-004 — The login form and the submit button keep their rendered width

- Covers: FR-RESPONSIVE-032
- Level: e2e (Playwright measurement, run against the base build and the changed build; not committed, as FR-RESPONSIVE-033 allows no test file change)
- Given the login page in protected mode with no session (as in the existing `TC-011 login page uses the full width` test: `mockApi` with `session: { mode: "protected", user: null }`), built once from the base and once from the change
- When, at 320 px, 360 px, 775 px, 1055 px and 1280 px wide, the bounding box widths of `.login-form` and of the "Log in" submit button, and their computed `width` and `max-width`, are read
- Then:
  - at each width, both bounding box widths are equal on the base and on the change (difference at most 0.5 px);
  - at 775 px, the username field, password field and "Log in" button each span the viewport width minus at most 16 px on each side, and do not touch either edge (FR-RESPONSIVE-022);
  - at 1280 px, the form is at most 24rem (384 px) wide and centred; the button keeps Carbon's cap (computed `max-width` 20rem, 320 px), as on the base;
  - at 320 px, the username field, password field and "Log in" button are fully visible within the viewport and the document is not wider than the viewport (FR-RESPONSIVE-009)
- Corner cases:
  - computed `max-width` of `.login-form` is `none` at every width on both builds (on the change, because the top-level rule sets none; this is why the removed declaration had no effect);
  - computed `max-width` of the button is `none` at 320, 360, 775 and 1055 px on both builds (the cap is still lifted below 1056 px);
  - 1055 px versus 1056 px: at 1055 px the button spans the form width (wider than 320 px), at 1056 px the desktop layout of TC-012 (issue #165) applies;
  - with a login error shown at 775 px, the error notification still spans the form width (as in the existing TC-011 e2e test).

### TC-005 — The existing suites and checks pass

- Covers: FR-RESPONSIVE-033, FR-RESPONSIVE-032
- Level: unit, integration and e2e (existing Go tests and Playwright suite) and frontend checks
- Given the change applied on top of the base, with the existing tests unchanged
- When `go vet ./...`, `go test -count=1 ./...`, `node --test scripts/*.test.mjs`, `npm run build` and `npx playwright test` run
- Then all pass, with the same test list as on the base; in particular in `tests/responsive-layout.spec.ts`:
  - `login page and header menu` at 320 px and 360 px (FR-RESPONSIVE-009: "Log in" button in viewport, no horizontal scroll, header menu entries on screen);
  - `TC-011 login page uses the full width` at 360, 775 and 1055 px (FR-RESPONSIVE-022: fields and button within the 16 px gutter, not touching the edges; full-width login error at 775 px);
  - `TC-012 desktop login page layout unchanged` at 1056 and 1440 px (form 384 px wide, centred);
  - `TC-014 (#266)` and `TC-015 (#266)` login messages
- Corner cases:
  - narrow screens (320 to 1055 px) and the 1056 px desktop switch behave as on the base;
  - keyboard: the login tests that type in the fields and submit still pass (focus and Tab order are unaffected by a width rule);
  - repeated runs give the same result (`-count=1`, no cached Go results).

### TC-006 — Nothing else changes in the specifications, change records and todo files

- Covers: FR-RESPONSIVE-033
- Level: repository check (`git diff`)
- Given the change applied on top of the base
- When these commands run:
  - `git diff <base> -- doc/specifications/responsive-layout/functional.md`;
  - `git diff --name-status <base> -- doc/changes/*issue-140-* doc/changes/*issue-165-* doc/changes/*issue-200-* doc/changes/*issue-203-*`;
  - `git diff --name-status --diff-filter=DMRT <base> -- doc/todo/`
- Then:
  - in `functional.md`, no line of FR-RESPONSIVE-001 to FR-RESPONSIVE-030 is added, removed or modified; the only changes are the issue #202 section (FR-RESPONSIVE-031 to 033) and the issue #202 Traceability bullet;
  - the second and third commands print nothing: the change folders of issues #140 (`2026-11-11-1142-issue-140-responsive-layout`, `2026-11-11-1513-issue-140-full-width-list`), #165, #200 and #203, and every existing file under `doc/todo/` (including `issue-165-r1-04.md`), are unchanged; new follow-up records for issue #202 (`doc/todo/issue-202-r*.md`) are allowed
- Corner cases:
  - no FR-RESPONSIVE or UC-RL ID is renumbered or removed (the sorted list of FR-RESPONSIVE IDs defined in `functional.md` equals the base list plus FR-RESPONSIVE-031 to 033);
  - `doc/use-cases/responsive-layout/UC-RL-01-use-full-width-on-every-page-below-desktop.md` is unchanged;
  - the user observes no change in the application: no API, database, setting or screen change (follows from TC-003, TC-004 and TC-005).
