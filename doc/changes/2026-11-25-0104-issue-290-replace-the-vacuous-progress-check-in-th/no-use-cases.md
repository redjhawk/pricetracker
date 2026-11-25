# No use cases for issue #290

This change is test-suite maintenance with no user-visible behaviour, so it has no use cases.

It changes only the e2e test "long whitespace TC-005 a 1025-space token input sends no replacement, shows no message and closes" in `tests/claude-token-settings.spec.ts` (see `doc/specifications/test-suite-hygiene/functional.md`). The requirements cover only test code and test results:

- **FR-TEST-HYG-057**: the test no longer looks for the "Saving settings…" text after the Settings modal has closed, and no longer loops over the sent save requests.
- **FR-TEST-HYG-058**: the test fails if any save request is sent between the Save click and the reopening of the modal, and still fails if a "too long" text is shown after the modal has closed.
- **FR-TEST-HYG-059**: on a normal run the test passes, with the same outcome as before (modal closes, no request, no "too long" text, empty token input with the `••••wxyz` hint and its rejected warning on reopening).
- **FR-TEST-HYG-060**: outside `doc/`, the diff changes only this test; the e2e suite and the unit-test gate pass, and the operator sees no change in the application.

The Settings modal's behaviour for a whitespace-only input is excluded from the change and stays as specified by FR-SECRET-014 in `doc/specifications/settings-secret-masking/functional.md`, already covered by UC-SECRET-007. No production code, API, database, user interface or runtime behaviour changes, and the operator has no new goal or flow. The test cases and the deterministic gates check these requirements directly.
