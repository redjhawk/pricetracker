# Review: issue #115 application information

Reviewer: independent reviewer agent\
Reviewed revision: `2b3eaba848ea5bedd7ea6ab21ed58bebcc14ae1f` (`git diff origin/master...HEAD`)\
Inputs: `doc/specifications/app-information/functional.md`, `technical.md`, `API_SPECIFICATION.md` (Users and login).

Known limitation (not a finding): the `.github/workflows/release.yml` hunk (compute `TAG` first, pass `PRICEFOLLOWER_VERSION="$TAG"`, reuse in `gh release create`) could not be pushed (token scope). Until a maintainer applies it, release binaries do not carry the release tag (FR-APP-INFO-007 not met in CI releases); see REV-001 for the resulting value.

## Findings

### REV-001 — Build script yields `dev-dev` when no tag is available
- Location: `scripts/build-release.sh:25`
- Requirement: API_SPECIFICATION.md session `version` ("`dev` when no tag is available"); FR-APP-INFO-008; technical spec Backend.
- Evidence: `VERSION="${PRICEFOLLOWER_VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo dev)-dev}"` appends `-dev` to the fallback `dev`. The Go fallback (`internal/version/version.go` `resolve`) returns plain `dev`.
- Expected/actual: no tags (e.g. CI checkout depth 1, which applies to every release until the workflow hunk lands) -> expected `dev`, actual `dev-dev` embedded in the binary and shown in the modal.
- Impact: low; cosmetic contract mismatch, but it is what released binaries will show today. The technical spec prescribes this exact line, so spec and API contract disagree.
- Suggested resolution: e.g. `TAG=$(git describe --tags --abbrev=0 2>/dev/null) && VERSION="${PRICEFOLLOWER_VERSION:-$TAG-dev}" || VERSION="${PRICEFOLLOWER_VERSION:-dev}"`, and align the technical spec line.
- Provisional severity: minor.

### REV-002 — Stale username shown when the session ended before opening the modal
- Location: `src/components/InformationModal.tsx:21-24`, `src/App.tsx` (username passed from initial session)
- Requirement: functional "States and corner cases": "If the session has ended when the modal is opened ... no stale username is shown."
- Evidence: the modal refetches `getSession()` but only reads `session.version`; `username` comes from the session loaded at app start. `GET /api/v1/auth/session` returns `200` with `user: null` for an expired session (not 401), so no existing 401 handling triggers.
- Expected/actual: session expired in another tab, modal opened -> expected no stale username (ended-session behavior); actual shows the previous username with the version.
- Impact: low; information-only display, no data access. The technical spec claims existing 401 handling covers it, which does not hold for this endpoint.
- Suggested resolution: when the refetched session has `user === null` while a username is shown (protected mode), hide the username or trigger the existing logged-out flow (reload to login); add a Playwright case.
- Provisional severity: minor.

### REV-003 — CSS file differs from technical spec
- Location: `src/index.css:486-493`
- Requirement: technical spec mapping 013 lists `src/styles.css`.
- Evidence: `src/styles.css` does not exist; the rules were added to `src/index.css`, the project's actual stylesheet.
- Impact: none functionally; spec documentation inaccuracy.
- Suggested resolution: correct the technical spec file name.
- Provisional severity: trivial.

## Checked without findings
- FR-001..006: menu order Information/Settings/Log out (user), Information/Log out (admin), open mode; username shown only when logged in; no email. Covered by Playwright.
- FR-009..013: loading, error text, retry on reopen, Close/icon/Escape with focus return via `launcherButtonRef`, keyboard, 400 px wrap. Covered by Playwright.
- Backend: additive `version` field, `sync.Once` resolution, 2 s git timeout, unit tests for resolve; auth tests updated. No security concern (release tag is public). No unspecified behavior found.

## Recheck of review fixes

Rechecked revision: `41180419765ec5060db20ed0cc3e9192dd7cb24e` (`git diff HEAD~1 HEAD`). The code was read only; the tests were not run.

- REV-001: resolved. `TAG` is empty when there is no tag, so `${TAG:+$TAG-dev}` expands to nothing and `VERSION` defaults to `dev`. `PRICEFOLLOWER_VERSION` still takes precedence. With a tag the result is still `<tag>-dev`. The technical spec was updated to match.
- REV-002: resolved. When the refetched session has `mode: "protected"` and `user: null`, the modal hides the username and dispatches the existing `authRequiredEvent` (`src/api/client.ts`). `App.tsx:63` listens for it and reloads the session, which renders the login page. Open mode (`user: null` is expected there) is not affected. The new Playwright case covers it. The technical spec was updated.
- REV-003: resolved. The technical spec now names `src/index.css`.

New findings: none. Note: `sessionEnded` is never reset, but this does no harm because the shell unmounts the modal when it switches to the login page.
