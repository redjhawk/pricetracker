# No new tests: issue #276

This change only updates `package-lock.json` (`npm audit fix` for the `source-map-js` advisory GHSA-68fv-2mgg-jv7q). It changes no behaviour, so there is nothing new to cover with Go or Playwright tests.

A new test file cannot be added either. FR-DEP-AUDIT-004 says the diff outside `doc/` may contain only `package-lock.json`, and TC-005 checks this. Any `*_test.go`, `tests/*.spec.ts` or `scripts/*.test.mjs` file would break that requirement.

The test cases are proved by repository checks and by the existing suites, as [testcases.md](testcases.md) says:

- TC-001, TC-002: `npm ls source-map-js --all`, `npm audit`, `npm audit --json` after `npm ci`.
- TC-003: running `npm audit fix` again (without `--force`) leaves `git diff --exit-code -- package-lock.json package.json` clean.
- TC-004: `npm ci` from a clean tree, then `npm ls --all`.
- TC-005: `git diff --name-status master -- . ':!doc'` lists only `M package-lock.json`.
- TC-006: `go vet ./...`, `go test -count=1 ./...`, `node --test scripts/*.test.mjs`.
- TC-007: `npm run build`, `npx playwright test`, and `npx playwright test --list` showing the same tests as on `master`.

No red phase is possible: the existing suites already pass on `master` and must still pass after the change.
