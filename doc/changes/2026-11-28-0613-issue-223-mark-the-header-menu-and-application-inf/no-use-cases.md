# No use cases for issue #223

This change is documentation maintenance with no user-visible behaviour, so it has no use cases.

It marks the header menu and application information subjects as implemented. Both are already shipped on master (`src/components/AppMenu.tsx`, `src/components/InformationModal.tsx`) and covered by e2e tests, but the documentation still called parts of them "specified" or "ready" (see `doc/specifications/app-header-menu/functional.md` and `doc/specifications/app-information/functional.md`). The requirements cover only documentation text:

- **FR-APP-MENU-008**: the Status cell of the **Header menu** row in section 15 of `doc/FUNCTIONAL_SPECIFICATIONS.md` is exactly `implemented`, without "FR-APP-MENU-007 specified (issue #215)".
- **FR-APP-INFO-015**: the Status cell of the **Application information** row is exactly `implemented`, without "specified (issue #115)" or "FR-APP-INFO-014 specified (issue #215)".
- **FR-APP-INFO-016**: the Status line of the application-information functional specification says implemented instead of `ready`.
- **FR-APP-INFO-017**: only files under `doc/` change. FR-APP-INFO-001 to 014, FR-APP-MENU-001 to 007, the other section 15 rows, earlier change records and `doc/todo/` stay as they are, the test suite passes, and the operator sees no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no user goal or flow is involved. The existing use cases UC-APP-MENU-001 and UC-APP-INFO-001 describe the shipped behaviour and need no changes. The test cases and the deterministic gates check these requirements directly.
