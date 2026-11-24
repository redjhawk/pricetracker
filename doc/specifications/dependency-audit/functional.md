# Dependency audit: functional specification

Status: FR-DEP-AUDIT-001–006 specified (issue #276)
Source: issue #276 ("Dependencies: fix the source-map-js advisory reported by npm audit").
Scope: repository maintenance. Application behaviour is not affected.

## Purpose

- Issue #276: `npm audit` reports one high-severity advisory, GHSA-68fv-2mgg-jv7q, for `source-map-js` 1.0.0–1.2.1, a package the frontend build tools pull in indirectly (today `package-lock.json` locks `source-map-js` 1.2.1). Update the locked dependencies with `npm audit fix` so that `npm audit` reports no advisory, with no other change.

## Actors

- Developer or agent working on the repository, who installs the npm dependencies, runs `npm audit`, builds the frontend and runs the tests.

## Terms

- **Lockfile:** `package-lock.json` at the repository root, which fixes the exact version of every npm package installed by `npm ci`.
- **Manifest:** `package.json` at the repository root, which lists the direct npm dependencies and their allowed version ranges.
- **Advisory:** a vulnerability report that `npm audit` lists for an installed npm package.
- **Test gates:** `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build` and `npx playwright test`.

## Scope and exclusions

- In scope: the lockfile, updated as `npm audit fix` (without `--force`) updates it.
- Excluded: the manifest, application code (Go, frontend), tests, API, database and user interface; any other dependency upgrade not made by `npm audit fix`; Go module dependencies.

## Requirements

- **FR-DEP-AUDIT-001** After the change, `npm audit` run on the repository lists no advisory for `source-map-js`, and the lockfile locks no `source-map-js` version from 1.0.0 to 1.2.1.
- **FR-DEP-AUDIT-002** After the change, `npm audit` run on the repository reports no advisory of any severity (`found 0 vulnerabilities`).
- **FR-DEP-AUDIT-003** The lockfile holds the versions that `npm audit fix` without `--force` gives, and stays in step with the manifest: `npm ci` installs from it without error.
- **FR-DEP-AUDIT-004** The change's diff outside `doc/` is limited to `package-lock.json`. The manifest is unchanged.
- **FR-DEP-AUDIT-005** On the repository as changed, every test gate passes.
- **FR-DEP-AUDIT-006** The operator observes no change in the application.

## States

Not applicable: there is no user interface, loading, empty or error state. The only observable outputs are the `npm audit` report, the build and the test results.

## Acceptance criteria

- `npm ci` succeeds, then `npm audit` prints `found 0 vulnerabilities` and lists neither `source-map-js` nor GHSA-68fv-2mgg-jv7q (FR-DEP-AUDIT-001, FR-DEP-AUDIT-002, FR-DEP-AUDIT-003).
- `npm ls source-map-js --all` lists only versions outside 1.0.0–1.2.1 (FR-DEP-AUDIT-001).
- Running `npm audit fix` again on the changed repository leaves the lockfile unchanged (FR-DEP-AUDIT-003).
- `git diff --name-only origin/master` lists, outside `doc/`, only `package-lock.json` (FR-DEP-AUDIT-004).
- `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build` and `npx playwright test` succeed (FR-DEP-AUDIT-005).

## Open questions

None. The issue states the work (`npm audit fix`, lockfile only) and the expected results (no advisory, build and tests pass).

## Notes for later stages

- The lockfile is at the repository root, outside the folders that the factory's `code` profile may edit (`cmd/`, `internal/`, `src/`, `tests/`, `web/`). The change cannot be made unless the stage that runs `npm audit fix` is allowed to write `package-lock.json`.
