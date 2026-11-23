# Issue #215 Tests fail on master (blocks #204)

The whole test suite must pass on `master`, and today it fails. This blocks the agent factory on #204: its developer reported the failures and the factory reproduced them on `master` at db004c8.

Fix the code or the tests on `master` so that `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build` and `npx playwright test` pass, keeping the behaviour that the functional specifications define.

## Output on master

~~~
  ✓   58 [chromium] › tests/platform-tabs.spec.ts:280:1 › TC-030 Amazon tab shows no stop warning after the upgrade (2.8s)
  ✓   59 [chromium] › tests/responsive-layout.spec.ts:153:5 › 320px › tracked items list on both tabs (2.3s)
  ✓   60 [chromium] › tests/responsive-layout.spec.ts:181:5 › 320px › empty list (1.2s)
  ✓   61 [chromium] › tests/responsive-layout.spec.ts:188:5 › 320px › item details for Amazon and LeBoncoin (2.5s)
  ✓   62 [chromium] › tests/responsive-layout.spec.ts:205:5 › 320px › administrator page with a long username (1.5s)
  ✓   63 [chromium] › tests/responsive-layout.spec.ts:217:5 › 320px › login page and header menu (2.3s)
  ✓   64 [chromium] › tests/responsive-layout.spec.ts:231:5 › 320px › modals (3.9s)
  ✓   65 [chromium] › tests/responsive-layout.spec.ts:153:5 › 360px › tracked items list on both tabs (2.3s)
  ✓   66 [chromium] › tests/responsive-layout.spec.ts:181:5 › 360px › empty list (1.2s)
  ✓   67 [chromium] › tests/responsive-layout.spec.ts:188:5 › 360px › item details for Amazon and LeBoncoin (2.7s)
  ✓   68 [chromium] › tests/responsive-layout.spec.ts:205:5 › 360px › administrator page with a long username (1.7s)
  ✓   69 [chromium] › tests/responsive-layout.spec.ts:217:5 › 360px › login page and header menu (2.3s)
  ✓   70 [chromium] › tests/responsive-layout.spec.ts:231:5 › 360px › modals (3.8s)
  ✓   71 [chromium] › tests/responsive-layout.spec.ts:262:3 › 700px list on both tabs (1.6s)
  ✓   72 [chromium] › tests/responsive-layout.spec.ts:273:3 › QA-F001 700px tables do not scroll inside and keep Status and actions on screen (1.7s)
  ✓   73 [chromium] › tests/responsive-layout.spec.ts:291:3 › 700px admin page (1.4s)
  ✓   74 [chromium] › tests/responsive-layout.spec.ts:262:3 › 800px list on both tabs (1.6s)
  ✓   75 [chromium] › tests/responsive-layout.spec.ts:273:3 › QA-F001 800px tables do not scroll inside and keep Status and actions on screen (1.6s)
  ✓   76 [chromium] › tests/responsive-layout.spec.ts:291:3 › 800px admin page (1.2s)
  ✓   77 [chromium] › tests/responsive-layout.spec.ts:262:3 › 1000px list on both tabs (1.7s)
  ✓   78 [chromium] › tests/responsive-layout.spec.ts:273:3 › QA-F001 1000px tables do not scroll inside and keep Status and actions on screen (1.6s)
  ✓   79 [chromium] › tests/responsive-layout.spec.ts:291:3 › 1000px admin page (1.5s)
  ✓   80 [chromium] › tests/responsive-layout.spec.ts:300:1 › resize keeps the tab, search text and modal input (1.5s)
  ✓   81 [chromium] › tests/responsive-layout.spec.ts:318:1 › desktop keeps the tracked table header row (1.4s)
  ✓   82 [chromium] › tests/responsive-layout.spec.ts:343:3 › 320px tracked-items list spans the full width (1.7s)
  ✓   83 [chromium] › tests/responsive-layout.spec.ts:343:3 › 360px tracked-items list spans the full width (1.6s)
  ✓   84 [chromium] › tests/responsive-layout.spec.ts:583:1 › TC-001 tracked-item cards run edge to edge (5.8s)
  ✓   85 [chromium] › tests/responsive-layout.spec.ts:602:1 › TC-002 card content stays inset and readable (6.3s)
  ✓   86 [chromium] › tests/responsive-layout.spec.ts:641:1 › TC-003 card layout and full width switch together at 1056px (3.1s)
  ✓   87 [chromium] › tests/responsive-layout.spec.ts:677:1 › TC-004 no-match message inside the full-width list (2.8s)
  ✓   88 [chromium] › tests/responsive-layout.spec.ts:706:1 › TC-005 card values side by side (1.8s)
  ✓   89 [chromium] › tests/responsive-layout.spec.ts:740:1 › TC-006 phone card values in reading and focus order (2.5s)
  ✓   90 [chromium] › tests/responsive-layout.spec.ts:813:1 › TC-007 elements above the list use the full width (2.7s)
  ✓   91 [chromium] › tests/responsive-layout.spec.ts:863:3 › TC-008 tracked items empty state uses the full width (1.4s)
  ✓   92 [chromium] › tests/responsive-layout.spec.ts:863:3 › TC-008 tracked items loading state uses the full width (1.4s)
  ✓   93 [chromium] › tests/responsive-layout.spec.ts:863:3 › TC-008 tracked items error state uses the full width (1.5s)
  ✓   94 [chromium] › tests/responsive-layout.spec.ts:879:1 › TC-009 item details use the full width (8.2s)
  ✓   95 [chromium] › tests/responsive-layout.spec.ts:930:1 › TC-009 item details loading state uses the full width (1.4s)
  ✓   96 [chromium] › tests/responsive-layout.spec.ts:945:1 › TC-009 item details error state uses the full width (1.4s)
  ✓   97 [chromium] › tests/responsive-layout.spec.ts:957:1 › TC-010 administrator users list edge to edge (2.3s)
  ✓   98 [chromium] › tests/responsive-layout.spec.ts:993:1 › TC-010 administrator long username wraps inside its card at 320px (1.4s)
  ✓   99 [chromium] › tests/responsive-layout.spec.ts:1016:1 › TC-011 login page uses the full width (2.0s)
  ✓  100 [chromium] › tests/responsive-layout.spec.ts:1047:1 › TC-012 desktop tracked items layout unchanged (2.6s)
  ✓  101 [chromium] › tests/responsive-layout.spec.ts:1074:1 › TC-012 desktop item details layout unchanged (5.2s)
  ✓  102 [chromium] › tests/responsive-layout.spec.ts:1092:1 › TC-012 desktop administrator page layout unchanged (1.6s)
  ✓  103 [chromium] › tests/responsive-layout.spec.ts:1111:1 › TC-012 desktop login page layout unchanged (1.3s)
  ✓  104 [chromium] › tests/responsive-layout.spec.ts:1170:1 › TC-013 header and modals unchanged below desktop (7.1s)
  ✓  105 [chromium] › tests/responsive-layout.spec.ts:1195:1 › TC-013 administrator header and Add user modal unchanged below desktop (2.5s)
  ✓  106 [chromium] › tests/responsive-layout.spec.ts:1208:1 › TC-013 Information modal with a long username at 400px (2.1s)


  1) [chromium] › tests/app-information.spec.ts:125:1 › keyboard opens the entry and focus stays inside the modal 

    Error: expect(received).toBe(expected) // Object.is equality

    Expected: true
    Received: false

      137 |   for (let i = 0; i < 4; i += 1) {
      138 |     await page.keyboard.press("Tab");
    > 139 |     expect(await dialog.evaluate((element) => element.contains(document.activeElement))).toBe(true);
          |                                                                                          ^
      140 |   }
      141 | });
      142 |
        at /tmp/tmp.ZZNdLye1bn/tests/app-information.spec.ts:139:90

    Error Context: test-results/app-information-keyboard-o-c313a-ocus-stays-inside-the-modal-chromium/error-context.md

    attachment #2: trace (application/zip) ─────────────────────────────────────────────────────────
    test-results/app-information-keyboard-o-c313a-ocus-stays-inside-the-modal-chromium/trace.zip
    Usage:

        npx playwright show-trace test-results/app-information-keyboard-o-c313a-ocus-stays-inside-the-modal-chromium/trace.zip

    ────────────────────────────────────────────────────────────────────────────────────────────────

  1 failed
    [chromium] › tests/app-information.spec.ts:125:1 › keyboard opens the entry and focus stays inside the modal 
  105 passed (4.3m)
~~~

## Report of the developer of #204

Category: base-branch

## Failing tests (not in `tests.lock`)

Running the full `npx playwright test` fails, intermittently, on keyboard timing races in existing frontend code:

1. `tests/leboncoin-session-settings.spec.ts:125` "header menu: visible on every page, Information and Settings entries, keyboard and dismissal"
   - `expect(getByRole('menuitem', { name: 'Settings' })).toBeVisible()` at line 150: element not found.
   - Cause: line 147 presses Escape right after Enter opens the menu. Carbon's `OverflowMenu` (`src/components/AppMenu.tsx`) moves focus into the menu only once the menu is placed, so an early Escape reaches the trigger and is ignored. The menu stays open, and the Space at line 149 closes it.
2. `tests/app-information.spec.ts:125` "keyboard opens the entry and focus stays inside the modal" (failed in attempt 2)
   - `expect(dialog.contains(document.activeElement)).toBe(true)` at line 139 is false.
   - Cause: `src/components/InformationModal.tsx` uses Carbon's default sentinel focus wrap, which brings focus back into the modal only after a timeout following Tab. `SettingsModal` avoids this with `FeatureFlags enableFocusWrapWithoutSentinels`.

Both tests pass when the two spec files run alone (3 runs in a row, 93/93). They fail only under the load of the full suite: attempt 2 had 2 failures out of 106, and this attempt had 1 failure out of 106.

## Why this change cannot cause them

- `git diff master -- src tests package.json package-lock.json vite.config.ts index.html playwright.config.ts` is empty. This change edits only Go code, Go test data and docs.
- Playwright starts only the Vite dev server (`npm run dev:web`), and both specs mock every API call through `mockApi(context)`. No Go code runs during these tests.
- Earlier rounds fixed both races inside this PR (a controlled `open` plus closing on Escape in `AppMenu`, and the synchronous focus wrap in `InformationModal`). Review R3-02 ruled those fixes blocking because they are unrelated to #204 and belong in their own PR, as the #180 triage also did. This attempt removes them (the working tree's `src/` now matches master) and restores "No file in `src/` changes" in `doc/specifications/amazon-browser-retrieval/technical.md`. Without a fix on master, the green gate can still fail on these races.

## Suggested base-branch fix

Apply the two fixes from commit `64f47f6` (`src/components/AppMenu.tsx`, `src/components/InformationModal.tsx`) to master in a separate PR. Then this stage can resume with the R3-02 removal already in place.

## Gate status of this attempt

- `gofmt -l .`: only `internal/service/amazon_browser_retrieval_test.go`, a locked test file, which is left unchanged.
- `go vet ./...`, `go test ./...` and `npm run build`: pass.
- `npx playwright test`: 105 passed, 1 failed (test 1 above).

_Opened by the agent factory. When this issue is closed, the supervisor resumes #204._

## Comments by trusted authors

