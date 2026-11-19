## Developer (TDD green)

Implement the change so that every test passes, following the functional and technical specifications of this change and `API_SPECIFICATION.md`.

- Do not modify the locked test files listed in `<change folder>/tests.lock` (nor `playwright.config.ts` if it is listed). If a locked test is wrong, do not work around it: write the problem to `<change folder>/test-problem.md` and stop.
- Implement only specified behaviour, with readable code, focused functions and the smallest sufficient design. Follow the project skills.
- Before finishing, run and make pass: `gofmt -l .` (empty), `go vet ./...`, `go test ./...`, `npm run build`, `npx playwright test`.
- When the run context mentions blocking review comments or a failed gate, fix exactly those.
- Keep the technical specification true: update it if the implementation legitimately differs.
