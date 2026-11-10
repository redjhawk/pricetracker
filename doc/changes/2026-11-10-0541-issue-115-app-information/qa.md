# QA execution: application information modal (issue #115)

Status: blocked (interface cases); non-interface verification executed
Tested code/specification revisions: branch `ai-dev/issue-115-20261110-0541` at `71c7f4a` (includes REV-001 to REV-003 fixes); specs `doc/specifications/app-information/`
Environment: local `npm run dev` (Vite http://localhost:5173 proxying `/api` to Go API http://127.0.0.1:3001, `PRICEFOLLOWER_ENV=development`, open mode, disposable `.data/` with seeded sample items). Latest git tag: `v20261109200945`. Browser: Playwright 1.63.0 Chromium headless shell 1243, could not start.
Executed at: 2026-11-10 ~05:55 UTC
Tester: QA tester agent (independent of implementation)

## Exploratory session

No browser interface session was possible, so no exploratory UI actions and no seed. Actions run, in order: Playwright suite, full `go test ./...`, `go test -run TestResolve ./internal/version/`, `go test ./internal/httpapi/`, `npm run dev`, curl session through Vite and directly on the API, build with embedded tag, string check of the binaries, stop processes.

## Executed cases

| Case | Requirement / corner case | Actual interface URL | Source listing URL | Preconditions and exact actions | Expected | Observed | Result / evidence / finding |
| --- | --- | --- | --- | --- | --- | --- | --- |
| QA-001 | All UI acceptance criteria (menu entry for user/admin, modal content, open mode, REV-002 ended session, loading, error/retry, close/Escape/focus, keyboard, 400 px wrap) | none (browser did not start) | n/a | `npx playwright test tests/app-information.spec.ts` (9 tests) | Tests run in a browser | All 9 failed at `browserType.launch`: `chrome-headless-shell: error while loading shared libraries: libnspr4.so: cannot open shared object file` | BLOCKED. Missing prerequisite: system library `libnspr4.so` (package `libnspr4`), which cannot be installed here. No other Playwright browser could be listed or used (no access to `~/.cache/ms-playwright` outside the workspace). |
| QA-002 | Dev version = latest tag + `-dev` (decision 4), via frontend proxy | http://localhost:5173/api/v1/auth/session (API request, not an interface test) | n/a | `npm run dev`; `curl -s http://localhost:5173/api/v1/auth/session` | `version` = `v20261109200945-dev` | `{"mode":"open","user":null,"version":"v20261109200945-dev"}` | PASS (API only) |
| QA-003 | Same, direct on API; open mode returns no user (decision 2) | http://127.0.0.1:3001/api/v1/auth/session | n/a | `curl -s http://127.0.0.1:3001/api/v1/auth/session` | `user` null, version present | identical body as QA-002 | PASS (API only) |
| QA-004 | Release build embeds tag (decision 4) | n/a | n/a | `go build -trimpath -ldflags="-s -w -X pricefollower.local/internal/version.Version=v20261109200945" ...`, plus a plain `go build`; `grep -c -a v20261109200945` on both binaries | Tagged binary contains the tag, untagged one does not | tagged: 1 match; untagged: 0 | PASS (static). Running the tagged binary on port 3011 was refused by the sandbox (env-prefixed command needed approval), so its runtime session response was NOT checked. |
| QA-005 | Untagged / no-git fallback to `dev`, embedded value wins | n/a | n/a | `go test -count=1 -v -run TestResolve ./internal/version/` | pass | PASS | PASS (unit) |
| QA-006 | Session handler regression | n/a | n/a | `go test -count=1 ./internal/httpapi/`; `go test ./...` | pass | all packages ok | PASS |

No use case for a fixed critical finding exists in `decisions.md` requiring execution beyond the REV-002 Playwright test, which is BLOCKED in QA-001.

## Coverage and limitations

- Every interface case (menu, modal, user/admin/open modes, ended session, loading, failure and retry, close/Escape/focus, keyboard, narrow viewport) is BLOCKED: no runnable browser (`libnspr4.so` missing). API and unit checks do not count as interface tests.
- Account mode (regular user, administrator) was not exercised on the API either; the dev server ran in open mode only.
- The runtime version reported by the release binary was not observed (sandbox approval); only the embedded string was confirmed.
- Interface QA must be re-run in an environment with Playwright browser dependencies (for example `npx playwright install-deps`, or CI) before this counts as a QA pass.

## Handoff and cleanup

No new failures found. Dev server (Vite and Go API) stopped; port 3001 verified closed. Temporary binaries in `test-results/` deleted. The dev run created or reused the git-ignored `.data/` sample database.
