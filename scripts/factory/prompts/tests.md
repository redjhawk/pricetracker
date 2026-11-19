## Test author (TDD red)

Write the automated tests for every case in `<change folder>/testcases.md`, before any implementation exists. Follow the technical specification for where they go.

- Go unit and integration tests in `*_test.go` next to the code (integration: real SQLite in a temp dir, real HTTP handlers through `httptest`). Playwright e2e tests in `tests/*.spec.ts`, mocking the API the way the existing specs do. `playwright.config.ts` lists specs explicitly in `testMatch` and you cannot edit it: prefer adding cases to an existing listed spec; a new spec file fails the gate until the owner adds it by hand.
- Put the case ID in each test name, e.g. `TestItemRefresh_TC001_...` or `test("TC-001 ...")`. Every `TC-` must appear in a test.
- You may add the minimum production stubs needed to compile (types, empty functions returning zero values or "not implemented") so that tests fail on assertions rather than on compilation. Do not implement behaviour.
- Run the tests (`go test ./...`, `npx playwright test <file>`) and check that the new ones fail for the expected reason and existing ones still pass.
- After this stage the test files are locked: the implementation may not modify them. Make them correct and complete now.
- If the change genuinely cannot be tested automatically (documentation or workflow only), write `<change folder>/no-tests.md` with the reason instead.
- If `<change folder>/test-problem.md` exists, a developer found a problem in the earlier tests: fix it as the issue comments say, then delete that file.
