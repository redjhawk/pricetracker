# Blocking review comments (round 1)

## R1-02 `src/components/AppMenu.tsx:383`

The keyboard focus race fixes in `AppMenu.tsx` (usedKeyboard/restoreFocus, onOpen/onClose, item onKeyDown) and `InformationModal.tsx` (the `FeatureFlags enableFocusWrapWithoutSentinels` wrapper, line 454) have nothing to do with issue #180, which is backend only and has no UI change. The technical spec explains them as fixes for flaky Playwright gates. AGENTS.md says unrelated changes go in their own commit or PR. Mixing them in makes this PR harder to review and revert, and the frontend behaviour change has no test case in this change. Suggest moving them to a separate PR or issue that tracks the flaky tests `tests/app-information.spec.ts` and `tests/leboncoin-session-settings.spec.ts`.

Triage: Confirmed. `src/components/AppMenu.tsx` (usedKeyboard/restoreFocus, onOpen/onClose, item onKeyDown) and `src/components/InformationModal.tsx` (the `FeatureFlags enableFocusWrapWithoutSentinels` wrapper) change frontend focus behaviour. Issue #180 is backend only and has no UI change. The technical spec itself says these are fixes for flaky Playwright tests that are unrelated to the feature. Unrelated work mixed into the PR blocks the merge: move it to its own issue or PR.

