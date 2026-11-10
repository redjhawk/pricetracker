# Review decisions: issue #115 application information

Reviewed specification revisions: `doc/specifications/app-information/functional.md`, `doc/specifications/app-information/technical.md`, `API_SPECIFICATION.md` (session `version`, 2026-11-10) at `abca625`
Reviewed code revision: `2b3eaba848ea5bedd7ea6ab21ed58bebcc14ae1f` (`git diff origin/master...HEAD`)
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
- Resolution: pending (routed to developer).
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
- Resolution: pending (routed to developer, then reviewer/QA).
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
- Resolution: pending (routed to technical specifier/developer).
- Follow-up: none.

## Decision summary (published, informational)

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| REV-001 Build script yields `dev-dev` without tags | non-critical | fix | Build script fallback to be corrected to `dev` and technical spec aligned |
| REV-002 Stale username after session ended | non-critical | fix | Modal to apply ended-session flow when refetched `user` is null; Playwright regression case; spec note corrected |
| REV-003 CSS file name in technical spec | non-critical | fix | Technical spec to reference `src/index.css`; no code change |

Published as an issue comment: by the coordinator on issue #115. Included in the final run report: yes.

## Release readiness

No critical findings. All three non-critical findings are accepted as fixes and pending; they must be implemented, rechecked by the reviewer, and covered by QA before commit. Known limitation remains: the `.github/workflows/release.yml` hunk needs a maintainer (token scope), so CI releases do not carry the release tag (FR-APP-INFO-007) until applied; after the REV-001 fix they show `dev`.
