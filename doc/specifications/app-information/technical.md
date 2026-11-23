# Technical specification: application information

Status: ready
Functional specification: [functional.md](functional.md) (FR-APP-INFO-001..013, issue #115; FR-APP-INFO-014, issue #215, see [Issue #215](#issue-215-focus-kept-inside-at-any-speed-fr-app-info-014))

## Requirement mapping

| Functional ID | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- |
| 001–003 | `AppMenu` gets an optional `onOpenInformation` prop rendered as the first `OverflowMenuItem` "Information"; both `App.tsx` usages pass it. | `src/components/AppMenu.tsx`, `src/App.tsx` | Playwright: entry visible for user, admin, open mode |
| 004, 011, 012 | New `InformationModal` (Carbon `Modal`, `passiveModal={false}` with a single "Close" primary button, `modalHeading="Information"`); open state held in `App.tsx`, no route change; focus returns to the menu trigger. | `src/components/InformationModal.tsx`, `src/App.tsx` | Playwright: open, Close/Escape/close icon, focus on `Application menu` |
| 005, 006 | Modal shows "Username" (from current session `user.username`) only when `user` is non-null, and "Version". | `src/components/InformationModal.tsx` | Playwright: user/admin show username; open mode has no "Username" |
| 007, 008 | `internal/version` package: `var Version string` set by `-ldflags -X`; empty → `git describe --tags --abbrev=0` + `-dev`, else `dev`. `build-release.sh` injects `PRICEFOLLOWER_VERSION`. | `internal/version/version.go`, `internal/version/version_test.go`, `scripts/build-release.sh`, `.github/workflows/release.yml` | Go test; build with `PRICEFOLLOWER_VERSION=v20261109200945` and query the session |
| 009, 010 | On each modal open, the modal calls `getSession()`; shows Carbon `InlineLoading` while pending and `InlineNotification` (kind `error`, lowContrast, hideCloseButton) "The version could not be loaded." on failure. | `src/components/InformationModal.tsx`, `src/api/auth.ts` | Playwright: delayed and 500 mocks |
| 013 | Carbon `Modal size="sm"`; values use `overflow-wrap: anywhere`. | `src/components/InformationModal.tsx`, `src/index.css` (one rule) | Playwright at 400 px: no horizontal scroll |

## Frontend

- `src/api/auth.ts`: add `version: string` to `SessionState`.
- `src/components/AppMenu.tsx`: optional `onOpenInformation?: () => void`; when present render `<OverflowMenuItem itemText="Information" onClick={onOpenInformation} />` before Settings. Keyboard behaviour is Carbon's (FR-012).
- `src/components/InformationModal.tsx` (new): props `open`, `username: string | null`, `onClose`. While `open`, fetch `getSession()` once per opening (effect keyed on `open`, ignore results after close); state `loading | {version} | error`. Body: a simple `<dl>`: "Username" (only if `username`), "Version" with value / `InlineLoading description="Loading version"` / error notification. Close via `onRequestClose` (close icon, Escape) and the primary button "Close". Carbon Modal provides focus trap and accessible title.
- `src/App.tsx`: in both shells (admin ~L87, user/open ~L302) add `informationOpen` state, pass `onOpenInformation={() => setInformationOpen(true)}`, render `InformationModal` with the session username (`null` in open mode). On close, `menuButtonRef.current?.focus()`; the admin shell gets its own `triggerRef` the same way. Username is taken from the already-loaded session; the session endpoint answers `200` with `user: null` for an ended session (no 401), so when the modal's refetch returns `mode: "protected"` and `user === null` it hides the username and dispatches the existing `authRequiredEvent` (`src/api/client.ts`); the root reloads the session and shows the login page, the existing logged-out flow (REV-002).
- No new routes or global state; other modals unchanged (menu only opens from the header, so no stacking, FR-APP-MENU-006).

## Backend

- New `internal/version/version.go`: `var Version string` (ldflags target `github.com/.../internal/version.Version`, using the module path from `go.mod`) and `func Current() string`, resolved once with `sync.Once`: if `Version != ""` return it; else run `git describe --tags --abbrev=0` (2 s timeout, working directory = process cwd); on success return `<tag>-dev`, otherwise `dev`. Covers `npm run dev:api` (`go run`) and any build without ldflags.
- `internal/httpapi/auth.go`: session response adds `"version": version.Current()`. No auth required, same as the existing endpoint; no persistence or migration.
- `scripts/build-release.sh`: `TAG="$(git describe --tags --abbrev=0 2>/dev/null || true)"`, `VERSION="${PRICEFOLLOWER_VERSION:-${TAG:+$TAG-dev}}"`, `VERSION="${VERSION:-dev}"` (`dev` when no tag is available, REV-001; when `PRICEFOLLOWER_VERSION` is unset, local release builds are labelled as development, FR-008); add `-X <module>/internal/version.Version=$VERSION` to the existing `-ldflags`; print the version in the final message.
- `.github/workflows/release.yml` (unavoidable for FR-007: the tag is currently computed only after the build): add a first step `echo "TAG=v$(date -u +%Y%m%d%H%M%S)" >> "$GITHUB_ENV"`, run the build as `PRICEFOLLOWER_VERSION="$TAG" ./scripts/build-release.sh amd64`, and use `"$TAG"` in `gh release create`. The push may be refused (token without `workflow` scope); if so, record it as a limitation and leave this hunk for a maintainer: releases then show `<previous tag>-dev` until applied (checkout depth 1 has no tags, so in practice `dev`).

## API

`GET /api/v1/auth/session` gains a required field `version` (string, never null): `200 { "mode": "open", "user": null, "version": "v20261109200945" }`. Additive and backward compatible; no status codes change. Recorded in [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) "Users and login". Rationale: the session endpoint is already fetched by every shell, is open to all modes, and avoids a new endpoint.

## Scope and refactoring

Change boundary: the files listed above plus tests (`internal/httpapi/auth_test.go`, `tests/app-information.spec.ts`) and existing Playwright session mocks only if they break typing (they do not; extra field is optional for mocks). No refactoring.

## Verification and unresolved questions

- `go test ./...` (version fallback; session JSON contains non-empty `version`).
- `npm run build` (type check).
- `tests/app-information.spec.ts`: user, admin, open-mode entries and contents (001–006); version from mock (007/008 display); delayed response → loading (009); 500 → error text and retry on reopen (010); Close button, close icon, Escape, focus back on `Application menu` (011); keyboard open and focus inside modal (012); 400 px viewport no horizontal scroll with a long username (013).
- Manual: `PRICEFOLLOWER_VERSION=v20261109200945 ./scripts/build-release.sh amd64` then run and query the session (007); `npm run dev:api` shows `<tag>-dev` (008).

Unresolved: none.

## Issue #215: focus kept inside at any speed (FR-APP-INFO-014)

| Technical ID | Functional IDs | Technical solution | Files | Verification |
| --- | --- | --- | --- | --- |
| TS-APP-INFO-001 | FR-APP-INFO-014, FR-APP-INFO-012, FR-APP-INFO-011 | The Carbon `Modal` is wrapped in `<FeatureFlags enableFocusWrapWithoutSentinels>`, as in `SettingsModal`: Tab and Shift+Tab are wrapped synchronously in the modal's `keydown` handler | `src/components/InformationModal.tsx` (already on master, PR #214) | Playwright TC-004 – TC-007 |

### Cause

By default the Carbon `Modal` (1.117) traps focus with two sentinel spans (`tabIndex=0`) around the dialog and a `blur` handler that moves focus back inside after a `setTimeout`. A Tab from the last control first lands on the end sentinel, outside the dialog, and comes back only after the timeout. Under load (full e2e suite) the existing test "keyboard opens the entry and focus stays inside the modal" saw focus outside the dialog.

### Frontend

The fix is already on master (PR #214, commit `542bedc`, merged 2026-11-23):

- With the `enable-focus-wrap-without-sentinels` flag, Carbon renders no sentinels. Its `keydown` handler calls `wrapFocusWithoutSentinels`: on Tab from the last tabbable element of the dialog (or Shift+Tab from the first) it calls `preventDefault()` and focuses the opposite end in the same event. Tabs between controls are native moves inside the dialog. Focus is therefore inside the dialog right after every key press, with no timer.
- Initial focus: no element in the body has `data-modal-primary-focus`, so Carbon focuses the primary **Close** button on opening, a tabbable control inside the dialog.
- Tab order: close icon and **Close** button. `InlineLoading` and the error `InlineNotification` (`hideCloseButton`) add no tabbable element, so the loading, error and loaded states share the same order, and the version arriving during tabbing changes nothing (TC-006). If the body scrolls (400 px), Carbon makes `.cds--modal-content` tabbable; the wrap uses the current tabbable list, so it stays correct.
- Closing (Escape, **Close**, close icon) is unchanged: `launcherButtonRef` puts focus back on the "Application menu" trigger.
- Accessibility: same dialog role, title and labels; only the trap mechanism changes.

Expected developer work: none, unless a new test fails. Any fix stays inside `src/components/InformationModal.tsx`: no change to Carbon, no new dependency.

Residual, outside FR-APP-INFO-014's key-only scope and not covered by the test cases: if a pointer click on a non-interactive part of the dialog focuses the dialog container (`tabIndex=-1`, not tabbable), a following Shift+Tab is a native move to the page behind it. This can be filed as an issue if wanted.

### Backend and API

Not affected: no endpoint, request or response changes. `API_SPECIFICATION.md` is unchanged.

### Tests

All in `tests/app-information.spec.ts` (already in `testMatch`; its `mockApi` serves open, user and admin sessions, delayed replies and failures):

| Test case | Notes |
| --- | --- |
| TC-004 | Count N tabbable controls in the dialog, press Tab 2×N + 1 times; after each press a single synchronous `dialog.evaluate(el => el.contains(document.activeElement))` (no `expect.poll`) |
| TC-005 | Shift+Tab from the first control lands on the last control, then N more presses stay inside |
| TC-006 | Session reply delayed (`delayMs`) and failing (500), set after the page has loaded (`state.replies`, `state.sessionCalls = 0`) |
| TC-007 | Escape, **Close** and the close icon after fast tabbing; focus on "Application menu"; 3 repetitions |

Shared helpers inside the spec: CPU throttling through `context.newCDPSession(page)` and `Emulation.setCPUThrottlingRate` (rate 6), and a `focusin` recorder installed with `page.evaluate` once the dialog has focus. The recorder pushes every focus target outside `[role="dialog"]` of the visible modal into a `window` array, which must stay empty. Test names start with the TC ID and `(#215)`. The existing test "keyboard opens the entry and focus stays inside the modal" stays and must pass in the full run.

Pipeline note: as for the header menu ([technical.md](../app-header-menu/technical.md#pipeline-note)), the fix was already on master, so the TDD red gate may report that the tests pass before implementation. That is for the owner to decide; tests must not be made to fail on purpose.

## Revision note

2026-11-23, issue #215: TS-APP-INFO-001 added for FR-APP-INFO-014. The code was already on master (PR #214).
