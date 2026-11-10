# Review decisions: issue #115 application information

Reviewed specification revisions: `doc/specifications/app-information/functional.md`, `doc/specifications/app-information/technical.md`, `API_SPECIFICATION.md` (session `version`, 2026-11-10) at `abca625`
Reviewed code revision: `2b3eaba848ea5bedd7ea6ab21ed58bebcc14ae1f` (`git diff origin/master...HEAD`); fixes rechecked at `41180419765ec5060db20ed0cc3e9192dd7cb24e`
Reviewer: independent reviewer agent ([review.md](review.md))
Adjudicator: independent review adjudicator agent (separate from developer and reviewer)

## Findings and decisions

### REV-001: Build script yields `dev-dev` when no tag is available

- Evidence: `scripts/build-release.sh:25` is `VERSION="${PRICEFOLLOWER_VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo dev)-dev}"`; the `-dev` suffix is applied after the `|| echo dev` fallback, so an untagged checkout produces `dev-dev`. `API_SPECIFICATION.md:516` requires `dev` when no tag is available, and `internal/version/version.go` `resolve` returns `dev` in that case. The technical spec (line 13 says "else `dev`", line 29 prescribes the faulty line) is internally inconsistent. Verified.
- Impact and scenario: until the maintainer applies the `release.yml` hunk, CI releases build from a depth-1 checkout without tags and do not set `PRICEFOLLOWER_VERSION`, so every published binary would show `dev-dev` in the Information modal and in the session response.
- Criticality: non-critical, because the effect is a wrong-format informational string; no data, security, or other functionality is affected, and operators still see that the build is not a tagged release. It is nevertheless a verified API contract violation that affects real releases.
- Disposition: fix.
- Reason: one-line, low-risk change that restores conformance with the API contract and the Go fallback; leaving it would ship a contract violation in every release built before the workflow hunk lands.
- What must change: compute the tag separately, e.g. `TAG=$(git describe --tags --abbrev=0 2>/dev/null || true)`, then `VERSION="${PRICEFOLLOWER_VERSION:-${TAG:+$TAG-dev}}"; VERSION="${VERSION:-dev}"`; update `technical.md` line 29 to the same logic.
- Validation condition: in a checkout without tags and with `PRICEFOLLOWER_VERSION` unset, the script computes `dev`; with a tag `v20261109200945` it computes `v20261109200945-dev`; with `PRICEFOLLOWER_VERSION=v20261109200945` it computes `v20261109200945`. Reviewer rechecks the script and spec line.
- Specification decision: not applicable (technical correction; API contract unchanged).
- Resolution: fixed in `4118041`; reviewer recheck ([review.md](review.md), "Recheck of review fixes"): empty `TAG` gives `dev`, a tag gives `<tag>-dev`, `PRICEFOLLOWER_VERSION` takes precedence; technical spec aligned. Status: resolved (recheck by code reading).
- Follow-up: none beyond the fix; the `release.yml` hunk remains a recorded maintainer limitation.

### REV-002: Stale username shown when the session ended before opening the modal

- Evidence: `src/components/InformationModal.tsx:21-22` refetches `getSession()` but reads only `session.version`; `username` is a prop from the session loaded at app start (`src/App.tsx`). `GET /api/v1/auth/session` returns `200` with `user: null` for an ended session, so no 401 handling runs. `functional.md:37` requires: "If the session has ended when the modal is opened, the existing multi-user login behavior for an ended session applies; no stale username is shown." Verified; the technical spec's assumption that 401 handling covers this does not hold for this endpoint.
- Impact and scenario: protected mode; the session is ended (logout in another tab or expiry), the operator opens Information and sees the former username. Display only; no data access or privilege, because every other API call still requires a valid session.
- Criticality: non-critical, because the consequence is a stale display of the operator's own username without any data exposure or action enabled; likelihood is low (requires an ended session while the page stays open). It is, however, an explicit unmet functional requirement and must be fixed before completion.
- Disposition: fix.
- Reason: the functional specification states the behavior explicitly; implementing it is small and does not need a user decision.
- What must change: when the refetched session reports `mode` protected with `user === null`, apply the existing ended-session behavior (the same logged-out flow the app uses, i.e. return to login) and never render the stale username. Correct the technical spec statement about 401 handling to describe the actual mechanism. Add a Playwright case.
- Validation condition: Playwright test: logged-in user, session route mocked/ended to return `user: null`, open Information -> the previous username is not displayed and the app shows the login flow; existing app-information tests still pass. Reviewer rechecks.
- Specification decision: not applicable (behavior already specified).
- Resolution: fixed in `4118041`; reviewer recheck: protected mode with `user: null` hides the username and dispatches the existing `authRequiredEvent` (`src/api/client.ts`), which `App.tsx:63` handles by reloading the session into the login page; open mode unaffected; new Playwright case added; technical spec corrected. Status: resolved. The recheck did not run tests; the Playwright suite must pass in QA before the final diff is committed. Note: `sessionEnded` is never reset; harmless because the modal unmounts when the shell switches to login.
- Follow-up: none.

### REV-003: CSS file differs from technical spec

- Evidence: `technical.md:15` names `src/styles.css`, which does not exist; the rule is in `src/index.css:486-493`, the project's real stylesheet. Verified.
- Impact and scenario: none at runtime; only a misleading documentation reference for future maintainers.
- Criticality: non-critical, because it is a documentation inaccuracy with no behavioral effect.
- Disposition: fix (specification correction only; code is correct).
- Reason: trivial edit that keeps the technical spec accurate; changing the code to match the spec would be wrong.
- What must change: replace `src/styles.css` with `src/index.css` in `technical.md` mapping 013.
- Validation condition: `technical.md` references `src/index.css`; no code change.
- Specification decision: not applicable.
- Resolution: fixed in `4118041`; technical spec now names `src/index.css`. Status: resolved.
- Follow-up: none.

## Decision summary (published, informational)

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| REV-001 Build script yields `dev-dev` without tags | non-critical | fix | Fixed in `4118041`: untagged builds report `dev`; technical spec aligned; reviewer rechecked |
| REV-002 Stale username after session ended | non-critical | fix | Fixed in `4118041`: modal hides the username and triggers the existing login flow when the session has ended; Playwright case added; spec corrected; reviewer rechecked |
| REV-003 CSS file name in technical spec | non-critical | fix | Fixed in `4118041`: technical spec references `src/index.css` |

Published as an issue comment: by the coordinator on issue #115. Included in the final run report: yes.

## Release readiness

No critical findings. All three non-critical findings are fixed in `4118041` and the reviewer rechecked them by reading the code, with no new findings. Remaining condition: QA must run the tests, including the new REV-002 Playwright case, and they must pass on the final diff before commit. Known limitation remains: the `.github/workflows/release.yml` hunk needs a maintainer (token scope), so CI releases do not carry the release tag (FR-APP-INFO-007) until applied; after the REV-001 fix they show `dev`.
