# No use cases for issue #202

This change is stylesheet maintenance with no user-visible behaviour, so it has no use cases.

Below 1056 px, a rule of `src/index.css` sets `max-width: none` on both the login form and its submit button, though only the button has a width cap to lift (Carbon caps buttons at 20rem). The change limits the override to the submit button and comments it (see `doc/specifications/responsive-layout/functional.md`). The requirements cover only the stylesheet text and the absence of change:

- **FR-RESPONSIVE-031**: `max-width: none` for the login page is set only on `.login-form .cds--btn[type="submit"]`, in a rule preceded by a comment containing "20rem"; no rule sets `max-width` on `.login-form` alone; below 1056 px the form and the button still get `width: 100%`.
- **FR-RESPONSIVE-032**: the login form and its submit button keep their rendered width at 320, 360, 775, 1055 and 1280 px; FR-RESPONSIVE-009 and FR-RESPONSIVE-022 still hold.
- **FR-RESPONSIVE-033**: FR-RESPONSIVE-001 to 030, the issue #140, #165, #200 and #203 change records and `doc/todo/` are unchanged, the diff outside `doc/` touches only the login form rule of `src/index.css`, the checks pass, and the user observes no change in the application.

No API, database, user interface or runtime behaviour changes, and no new user goal or flow is involved. The login page layout below 1056 px that FR-RESPONSIVE-032 preserves is already described by step 6 of UC-RL-01 (`doc/use-cases/responsive-layout/UC-RL-01-use-full-width-on-every-page-below-desktop.md`), which stays valid as it is. The test cases and the deterministic gates check these requirements directly.
