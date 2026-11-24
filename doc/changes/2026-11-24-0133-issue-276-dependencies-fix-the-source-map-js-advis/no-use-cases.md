# No use cases for issue #276

This change is dependency maintenance with no user-visible behaviour, so it has no use cases.

It updates `package-lock.json` with `npm audit fix` (without `--force`) so that `npm audit` no longer reports the high-severity advisory GHSA-68fv-2mgg-jv7q for `source-map-js` 1.0.0–1.2.1 (see `doc/specifications/dependency-audit/functional.md`). The requirements cover only the lockfile, the `npm audit` report, the build and the tests:

- **FR-DEP-AUDIT-001**: `npm audit` lists no advisory for `source-map-js`, and the lockfile locks no `source-map-js` version from 1.0.0 to 1.2.1.
- **FR-DEP-AUDIT-002**: `npm audit` reports `found 0 vulnerabilities`.
- **FR-DEP-AUDIT-003**: the lockfile holds the versions `npm audit fix` gives and `npm ci` installs from it without error.
- **FR-DEP-AUDIT-004**: the diff outside `doc/` is limited to `package-lock.json`; `package.json` is unchanged.
- **FR-DEP-AUDIT-005**: `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build` and `npx playwright test` pass.
- **FR-DEP-AUDIT-006**: the operator sees no change in the application.

No application code, API, database, user interface or runtime behaviour changes, and no operator goal or flow is involved. The test cases and the deterministic gates check these requirements directly.
