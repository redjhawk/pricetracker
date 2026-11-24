# No use cases for issue #295

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It corrects one assertion of the e2e test TC-010 in `tests/platform-tabs.spec.ts` (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test code and test results:

- **FR-TEST-HYG-037**: the always-passing assertion that every recorded image request is for the accepted image URL is removed.
- **FR-TEST-HYG-038**: TC-010 fails unless the accepted Amazon image was requested at least once, and it still fails on a request for any other image (the unexpected-request check is kept).
- **FR-TEST-HYG-039**: outside `doc/`, the diff contains only `tests/platform-tabs.spec.ts`; the e2e suite and the unit-test gate pass, and the operator sees no change in the application.

The application behaviour TC-010 checks does not change, and existing use cases already describe it:

- UC-AMZ-THUMB-001 (an Amazon item with an accepted Amazon image is shown with that image in the tracked-items list; FR-AMZ-THUMB-001, FR-AMZ-THUMB-002)
- UC-AMZ-THUMB-002 (an Amazon item without an accepted image is shown with the placeholder; FR-AMZ-THUMB-003)

No production code, API, database, user interface or runtime behaviour changes, and the operator has no new goal or flow. The test cases and the deterministic gates check these requirements directly.
