# Dependency audit: technical specification

Status: TS-DEP-AUDIT-001–006 ready (issue #276; TS-DEP-AUDIT-005 is a manual follow-up by the owner)
Requirements: [functional](functional.md). Source: issue #276.

## Design

### Lockfile update (issue #276)

On `master`, `package-lock.json` (`lockfileVersion` 3) has one `source-map-js` entry, `node_modules/source-map-js` at 1.2.1. Two packages need it: `postcss` (`^1.2.1`, through `vite`) and `sass` (`>=0.6.2 <2.0.0`, a `peer` entry). It is not a direct dependency, so `package.json` does not name it.

- The fix is `npm audit fix` without `--force`, run on a clean install of `master` (`npm ci` first). It moves `node_modules/source-map-js` to the patched version that GHSA-68fv-2mgg-jv7q names. That version is above 1.2.1 and below 2.0.0, so it satisfies both requirer ranges, and no requirer or manifest range changes. The version is the one the advisory database gives on the day of the run, not one chosen by hand.
- Whatever else `npm audit fix` changes in the lockfile on that day is accepted as it is (FR-DEP-AUDIT-003): no entry is edited by hand, reverted or added. `lockfileVersion` and the root `name`/`version` stay the same.
- If `npm audit fix` without `--force` cannot clear every advisory, or changes `package.json`, the change stops and goes back to the owner: `--force`, overrides and manifest edits are out of scope (functional specification, exclusions).
- No `.npmrc`, `npm-shrinkwrap.json` or other lockfile is added.

### Tiers

- **Frontend: not affected.** No file under `web/` or `src/` changes. `source-map-js` is used only at build time (by `postcss` and `sass` inside `vite build`) to build source maps. It does not reach the application bundle's behaviour, so the pages, their states and their accessibility do not change (FR-DEP-AUDIT-006).
- **Backend: not affected.** No Go file, `go.mod` or `go.sum` changes. No schema or migration change.
- **API: not affected.** No endpoint, request, response or error changes. `API_SPECIFICATION.md` is not changed.

### Delivery (manual follow-up)

No factory stage can make this change:

- the `code` profile (tests and development stages) may edit only `cmd/`, `internal/`, `src/`, `tests/`, `web/`, `doc/specifications/`, `README.md`, `deploy/README.md` and the change folder, and may run only `npm run build` among the npm commands (no `npm ci`, `npm audit` or `npm audit fix`);
- the `docs` profile (this stage) may edit only `doc/` and `API_SPECIFICATION.md`;
- `package-lock.json` is a sensitive file for the factory (`sensitive_files` in `scripts/factory/gates.sh`), so the factory never merges a pull request that changes it on its own.

So the owner commits the lockfile by hand on branch `ai-dev/issue-276`, in its own commit that changes only `package-lock.json`, as the issue asks:

```sh
git switch ai-dev/issue-276 && git pull
npm ci
npm audit fix            # never --force
git diff --name-only     # must print only package-lock.json
npm audit                # must print "found 0 vulnerabilities"
git add package-lock.json
git commit -m "build(deps): fix the source-map-js advisory with npm audit fix" -m "Refs: #276"
git push
```

The owner then merges the pull request by hand once the gates pass (the factory stops it as `sensitive`). Giving a factory profile `Edit(package-lock.json)` and `Bash(npm audit fix)` would let a later dependency fix run unattended; that is a change to `scripts/factory/` and is out of scope here.

### Stages

- **Tests stage.** No test code is written: FR-DEP-AUDIT-004 allows no change outside `doc/` except the lockfile, and the cases are checked by `npm`/`git` commands and the existing suites. It writes `<change folder>/no-tests.md` with that reason and the mapping below; the red gate is skipped.
- **Development stage.** No code. It checks the lockfile on the branch with `git diff origin/master -- package-lock.json package.json` and `git log`:
  - if the owner's lockfile commit is there (a `package-lock.json` diff, no `package.json` diff, and the `node_modules/source-map-js` version above 1.2.1), it runs the gates, sets FR-DEP-AUDIT-001–006 in `functional.md` and TS-DEP-AUDIT-001–006 here to implemented (the `doc/FUNCTIONAL_SPECIFICATIONS.md` row is outside its profile; it says so in its summary);
  - otherwise it writes `<change folder>/blocked.md` starting with `Category: environment`, saying that the `code` profile cannot write `package-lock.json` or run `npm ci`/`npm audit fix`, and pointing to the commands of "Delivery". It changes nothing else.

## Technical IDs

| ID | Requirement | Decision |
| --- | --- | --- |
| TS-DEP-AUDIT-001 | FR-DEP-AUDIT-001, FR-DEP-AUDIT-002 | `npm audit fix` without `--force` on a clean `npm ci` of `master` moves every `source-map-js` entry outside 1.0.0–1.2.1 and clears every advisory. |
| TS-DEP-AUDIT-002 | FR-DEP-AUDIT-003 | The lockfile is exactly what `npm audit fix` gives: no hand edit, `lockfileVersion` 3 kept, registry `resolved` URLs with `integrity`; `npm ci` installs from it. |
| TS-DEP-AUDIT-003 | FR-DEP-AUDIT-004 | Only `package-lock.json` changes outside `doc/`; `package.json` unchanged; no new npm configuration or lockfile. A fix that needs more stops the change. |
| TS-DEP-AUDIT-004 | FR-DEP-AUDIT-005 | The existing gates prove the change: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`. |
| TS-DEP-AUDIT-005 | FR-DEP-AUDIT-003, FR-DEP-AUDIT-004 | Manual follow-up: the owner commits the lockfile on the branch (commands in "Delivery") and merges the pull request by hand. |
| TS-DEP-AUDIT-006 | FR-DEP-AUDIT-006 | Frontend, backend and API not affected: no application file changes. |

## Files to change

- `package-lock.json`: by the owner (TS-DEP-AUDIT-005), not by a factory stage.
- `doc/specifications/dependency-audit/functional.md`, `doc/specifications/dependency-audit/technical.md`: statuses, by the development stage.
- `<change folder>/no-tests.md`: by the tests stage.
- No file under `cmd/`, `internal/`, `src/`, `web/`, `tests/` or `scripts/`; no `package.json`, `go.mod`, `go.sum`, `playwright.config.ts` or `API_SPECIFICATION.md`.

## Tests

No new test file. Each case is proved by commands on the changed branch and by the existing suites:

| Test case | Proof |
| --- | --- |
| TC-001 | `npm ci`, then `npm ls source-map-js --all`, `npm audit`, `npm audit --json` (no `vulnerabilities.source-map-js`), and the `source-map-js` entries of `package-lock.json`; run by the owner (TS-DEP-AUDIT-005) and the reviewer. |
| TC-002 | `npm audit` prints `found 0 vulnerabilities` and exits 0; `metadata.vulnerabilities` all 0 in `npm audit --json`; run by the owner. |
| TC-003 | `npm audit fix` run again, then `git diff --exit-code -- package-lock.json package.json`; run by the owner. |
| TC-004 | `rm -rf node_modules && npm ci`, then `npm ls --all`; run by the owner. |
| TC-005 | `git diff --name-status origin/master -- . ':!doc'` lists only `M package-lock.json`; checked by the development stage and the reviewer (`git diff` is allowed to both). |
| TC-006 | `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`: the factory's unit gate (`run_unit_tests` in `scripts/factory/gates.sh`) and the development stage. |
| TC-007 | `npm run build`, `npx playwright test`: the factory's full gate and the development stage; the test list is unchanged because no test file changes (green gate). |

## Refactoring

None.

## Risks

- `npm audit fix` may also move other transitive packages. That is accepted (FR-DEP-AUDIT-003); the build and the e2e suite catch a broken update.
- The advisory database changes over time: a new advisory published before the owner's run fails FR-DEP-AUDIT-002 and stops the change for the owner, as the functional specification states.
