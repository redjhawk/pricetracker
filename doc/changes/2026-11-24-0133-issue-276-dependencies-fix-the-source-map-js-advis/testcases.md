# Test cases: fix the source-map-js advisory reported by npm audit (issue #276)

Sources: functional specification [dependency-audit](../../specifications/dependency-audit/functional.md) (FR-DEP-AUDIT-001 to FR-DEP-AUDIT-006). No use cases: see [no-use-cases.md](no-use-cases.md).

Conventions:

- **Lockfile:** `package-lock.json`. **Manifest:** `package.json`. **Baseline:** the base branch `master`, where the lockfile has one `source-map-js` entry, `node_modules/source-map-js` at version 1.2.1. The packages that depend on it ask for `^1.2.1` and `>=0.6.2 <2.0.0`.
- This change is dependency maintenance with no behaviour change. FR-DEP-AUDIT-004 limits the diff outside `doc/` to `package-lock.json`, so **no new test code is written**: a new test file would break FR-DEP-AUDIT-004. Repository checks (`npm`, `git`) and the existing suites run by the gates prove the cases. Consequence for the tests stage: it writes `no-tests.md` and skips the red phase.
- `npm audit` and `npm audit fix` read the public npm advisory database, so the result depends on the advisories published on the day of the run. The cases are checked on the day of the change.
- No user interface, API or database is involved, so narrow screens, keyboard, and empty or error screen states do not apply.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-DEP-AUDIT-001 | TC-001 |
| FR-DEP-AUDIT-002 | TC-002 |
| FR-DEP-AUDIT-003 | TC-003, TC-004 |
| FR-DEP-AUDIT-004 | TC-005 |
| FR-DEP-AUDIT-005 | TC-006, TC-007 |
| FR-DEP-AUDIT-006 | TC-005, TC-007 |

## Cases

### TC-001: No vulnerable `source-map-js` is locked or reported

- Covers: FR-DEP-AUDIT-001
- Level: repository check (`npm` on the changed lockfile)
- Given the change applied on top of `master`, and dependencies installed with `npm ci`
- When `npm ls source-map-js --all`, `npm audit` and `npm audit --json` are run, and the lockfile's `source-map-js` entries are read
- Then every `source-map-js` version listed or locked is outside 1.0.0–1.2.1, and the `npm audit` output names neither `source-map-js` nor `GHSA-68fv-2mgg-jv7q`
- Corner cases:
  - every `node_modules/**/source-map-js` entry is checked, not only the top-level one, so a nested copy of 1.2.1 or older would fail the case;
  - `npm audit --json` has no `source-map-js` key under `vulnerabilities`;
  - on the baseline the same check fails (1.2.1 is locked and the advisory is listed), so the case can tell a fixed lockfile from an unfixed one;
  - the new version still satisfies both requirer ranges (`^1.2.1`, `>=0.6.2 <2.0.0`), so `npm ls source-map-js --all` reports no `invalid` or `missing` entry.

### TC-002: `npm audit` reports no advisory of any severity

- Covers: FR-DEP-AUDIT-002
- Level: repository check (`npm audit`)
- Given the change applied, and dependencies installed with `npm ci`
- When `npm audit` and `npm audit --json` are run
- Then `npm audit` prints `found 0 vulnerabilities` and exits with code 0, and `metadata.vulnerabilities` in the JSON report is 0 for `info`, `low`, `moderate`, `high` and `critical`
- Corner cases:
  - development dependencies are included (no `--omit=dev`), since `source-map-js` comes in through the frontend build tools;
  - a lower-severity advisory (`low`, `moderate`) also fails the case, not only `high`;
  - running `npm audit` twice gives the same result.

### TC-003: The lockfile is what `npm audit fix` without `--force` gives

- Covers: FR-DEP-AUDIT-003
- Level: repository check (`npm audit fix`, `git diff`)
- Given the change applied, and dependencies installed with `npm ci`
- When `npm audit fix` (without `--force`) is run again
- Then it changes nothing, so `git diff --exit-code -- package-lock.json package.json` succeeds
- Corner cases:
  - repeated action: the fix run again is a no-op;
  - no version in the lockfile is a major-version jump that only `--force` would make (the diff of `package-lock.json` against `master` shows no change of major version for a package whose requirer range would not allow it);
  - `lockfileVersion` and the root `name`/`version` fields of the lockfile are unchanged from `master`.

### TC-004: `npm ci` installs from the lockfile without error

- Covers: FR-DEP-AUDIT-003
- Level: repository check (`npm ci` from a clean tree)
- Given the change applied, and no `node_modules` directory
- When `npm ci` is run
- Then it exits with code 0, so the lockfile is in step with the manifest
- Corner cases:
  - from a clean state (`node_modules` removed first), so a stale install cannot hide a mismatch;
  - `npm ls --all` afterwards reports no `invalid`, `missing` or `extraneous` package;
  - every changed `resolved` URL points at `https://registry.npmjs.org/` and has an `integrity` value.

### TC-005: Only the lockfile changes outside `doc/`

- Covers: FR-DEP-AUDIT-004, FR-DEP-AUDIT-006
- Level: repository check (`git diff`)
- Given the change on top of `master`
- When `git diff --name-status master -- . ':!doc'` is listed
- Then the only entry is `M package-lock.json`
- Corner cases:
  - `git diff --exit-code master -- package.json` succeeds (the manifest is unchanged);
  - no Go source, frontend source (`web/`, `src/`), test (`_test.go`, `.spec.ts`, `.test.mjs`), `go.mod`, `go.sum`, `playwright.config.ts`, `API_SPECIFICATION.md` or script changes, so no application code, API, database or page changes (FR-DEP-AUDIT-006);
  - no file is added or deleted outside `doc/`, including an `.npmrc` or a second lockfile (`npm-shrinkwrap.json`, `yarn.lock`).

### TC-006: Go vet, Go tests and script tests still pass

- Covers: FR-DEP-AUDIT-005
- Level: unit and integration (existing Go suites, with real SQLite and HTTP handlers) and existing Node script tests
- Given the change applied, and dependencies installed with `npm ci`
- When `go vet ./...`, `go test ./...` and `node --test scripts/*.test.mjs` are run
- Then each command exits with code 0
- Corner cases:
  - `go test -count=1 ./...` gives the same result, so the pass does not come from cached runs of the baseline;
  - the Go module dependencies are unchanged (covered by TC-005), so a Go failure points at the environment, not at this change.

### TC-007: The frontend builds and the e2e suite passes unchanged

- Covers: FR-DEP-AUDIT-005, FR-DEP-AUDIT-006
- Level: build and e2e (existing Playwright suite in `tests/`)
- Given the change applied, and dependencies installed with `npm ci`
- When `npm run build` and `npx playwright test` are run
- Then the build succeeds with no new error, and every Playwright test passes, with `npx playwright test --list` showing the same tests as on `master`
- Corner cases:
  - the build is the step that uses `source-map-js` (through the build tools that emit source maps), so a broken update shows here;
  - the existing e2e tests cover the operator's pages, including those at narrow screen widths, so a passing run shows the operator sees no change (FR-DEP-AUDIT-006);
  - no test is skipped or removed to get a pass (the test list matches `master`).
