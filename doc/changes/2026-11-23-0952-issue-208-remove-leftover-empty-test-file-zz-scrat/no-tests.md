# No new tests for issue #208

This change is repository maintenance with no behaviour change: it only deletes the empty leftover test file `internal/service/zz_scratch_docs_test.go` (FR-TEST-HYG-001).

No new test code is written, because FR-TEST-HYG-003 forbids any other test file change: adding or modifying a test would itself break the requirement. The test cases in [testcases.md](testcases.md) are proved without new tests:

- **TC-001** and **TC-003**: repository checks on the change's diff (the file is deleted, and it is the only entry outside `doc/`).
- **TC-002** and **TC-004**: the existing suites run by the gates (`go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`) still pass.

The deletion lands in this stage (listed in `delete.txt`), because the green gate fails on any `_test.go` file changed after the TDD red commit.
