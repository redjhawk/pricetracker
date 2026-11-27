# No use cases for issue #199

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It makes the wrap check of the e2e test "TC-005 card values side by side" in `tests/responsive-layout.spec.ts` count the item value at its base width of 14rem when it computes the free space of a card line, so that a grown item value no longer makes that space about 0, and updates the TC-005 entry of the issue #165 test case document to match (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test results, test failure messages and test documentation:

- **FR-TEST-HYG-094**: for every value that starts a later line of a card, the wrap check fails unless the free space of the previous line, with the item counted at its base width, is smaller than that value's width (1 px tolerance).
- **FR-TEST-HYG-095**: TC-005 fails when Prices starts a new line below the item while the item base width plus the Prices width fits in the card, with a message naming the width, card, tab, value, its width and the free space.
- **FR-TEST-HYG-096**: TC-005 keeps its name, widths, tabs, data and other checks and messages, and passes on the repository as changed.
- **FR-TEST-HYG-097**: TC-005 in the issue #165 test case document states that the free space counts the item value at its base width of 14rem.
- **FR-TEST-HYG-098**: the diff outside `doc/` is limited to the test file, only the TC-005 entry changes in the issue #165 change folder, the e2e suite and unit-test gate pass, and the operator observes no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no operator goal or flow is involved. The card layout itself (FR-RESPONSIVE-024) is unchanged and already has its use cases. The test cases and the deterministic gates check these requirements directly.
