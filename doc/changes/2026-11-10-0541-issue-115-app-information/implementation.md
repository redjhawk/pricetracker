# Implementation: application information (issue #115)

Sources: [functional](../../specifications/app-information/functional.md), [technical](../../specifications/app-information/technical.md), [API](../../../API_SPECIFICATION.md) session `version`.

## Requirement mapping

| Requirement | Change |
| --- | --- |
| FR-APP-INFO-001..003 | `AppMenu` optional `onOpenInformation` renders "Information" first; passed by the tracker shell (user/open mode) and the admin shell. |
| FR-APP-INFO-004, 011, 012 | New `InformationModal` (Carbon `Modal size="sm"`, heading "Information", primary button "Close", `onRequestClose` for icon/Escape, `launcherButtonRef` returns focus to the menu trigger). Open state is local to each shell; no route change. The admin shell moved into a small `AdminApp` component so it can hold its own open state and trigger ref. |
| FR-APP-INFO-005, 006 | "Username" shown only when the session username is non-null; "Version" always. |
| FR-APP-INFO-007, 008 | `internal/version`: `Version` (ldflags), `Current()` with `sync.Once`, falls back to `git describe --tags --abbrev=0` + `-dev` (2 s timeout), else `dev`. `scripts/build-release.sh` injects `PRICEFOLLOWER_VERSION` (default `<latest tag>-dev`). `release.yml` computes `TAG` first, passes it to the build, and reuses it for `gh release create`. |
| FR-APP-INFO-009, 010 | Modal calls `getSession()` on every open; `InlineLoading` "Loading version" while pending, error `InlineNotification` "The version could not be loaded." on failure; late results after close are ignored. |
| FR-APP-INFO-013 | `.information-list dd { overflow-wrap: anywhere; }` in `src/index.css` (the project stylesheet; the technical spec named `src/styles.css`, which does not exist). |
| API | `GET /api/v1/auth/session` returns `version` (`internal/httpapi/auth.go`); `SessionState.version` in `src/api/auth.ts`. |

## Files

- `internal/version/version.go`, `internal/version/version_test.go` (new)
- `internal/httpapi/auth.go`, `internal/httpapi/auth_test.go` (exact session JSON now includes `version`)
- `scripts/build-release.sh`, `.github/workflows/release.yml`
- `src/api/auth.ts`, `src/components/AppMenu.tsx`, `src/components/InformationModal.tsx` (new), `src/App.tsx`, `src/index.css`
- `tests/app-information.spec.ts` (new), `playwright.config.ts` (spec added to `testMatch`, otherwise it would not run)

Existing Playwright mocks were not changed: they omit `version`, which only affects the Information modal they do not open.

## Checks executed

- `go vet ./...`: passed.
- `go test ./...`: all packages ok, including `internal/version` and `internal/httpapi`.
- `npm run build`: passed (type check and Vite build).
- `npx tsc --noEmit --strict ... tests/app-information.spec.ts`: passed (spec type-checks).
- `npx playwright test tests/app-information.spec.ts`: not executable here; all 8 tests fail at browser launch (`chrome-headless-shell: error while loading shared libraries: libnspr4.so`), no system Chromium available. The full Playwright suite was not run for the same reason.

## Limitations

- Playwright tests are unverified in this environment; QA must run them where Chromium can start.
- Manual release build with `PRICEFOLLOWER_VERSION` and `npm run dev:api` check not executed.
- The `release.yml` push may be refused for missing `workflow` token scope (coordinator handles it, see technical spec).
