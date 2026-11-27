# No use cases for issue #201

This change is stylesheet maintenance with no user-visible behaviour, so it has no use cases.

Below 1056 px, `src/index.css` sets `display: block` on card cells in a standalone `.responsive-table td` rule, while every other card-cell declaration is in the `.responsive-table tbody tr td` rule, which matches the same cells. The change moves `display: block` into that rule and removes the standalone rule (see `doc/specifications/responsive-layout/functional.md`). The requirements cover only the stylesheet text and the absence of change:

- **FR-RESPONSIVE-034**: inside the `max-width: 1055.98px` query, the `.responsive-table tbody tr td` rule contains `display: block` and no rule has the selector `.responsive-table td` alone; the `.responsive-table td[data-label]::before` rule and the `overflow-wrap: anywhere` rule are unchanged.
- **FR-RESPONSIVE-035**: on the tracked items page (Amazon and LeBoncoin tabs) and the administrator page, every card or table cell keeps its rendered position and size at 320, 360, 775, 1055 and 1280 px; the existing `tests/responsive-layout.spec.ts` e2e tests pass and FR-RESPONSIVE-018 to 024 still hold.
- **FR-RESPONSIVE-036**: FR-RESPONSIVE-001 to 033, the issue #140, #165, #200, #202 and #203 change records and the existing `doc/todo/` files are unchanged, the diff outside `doc/` touches only the two card-cell rules of `src/index.css`, the checks pass, and the user observes no change in the application.

No API, database, user interface or runtime behaviour changes, and no new user goal or flow is involved. The card layout that FR-RESPONSIVE-035 preserves is already described by UC-TI-07 (`doc/use-cases/tracked-items/UC-TI-07-review-tracked-items-edge-to-edge-below-desktop.md`) and UC-RL-02 (`doc/use-cases/responsive-layout/UC-RL-02-review-users-list-edge-to-edge-below-desktop.md`), which stay valid as they are. The test cases and the deterministic gates check these requirements directly.
