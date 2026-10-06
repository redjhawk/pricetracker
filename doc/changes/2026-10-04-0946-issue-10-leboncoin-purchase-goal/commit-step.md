# Commit step: LeBoncoin purchase goal (#10)

## Gates passed before committing

- The functional and technical specs and the API contract are `ready` and consistent with the implementation.
- Review findings REV-001 and REV-002 are fixed and were re-reviewed, REV-003 was rejected, and REV-004 was deferred. See [decisions.md](decisions.md). No critical blocker is open.
- Checks: `go vet ./...` passed, `go test ./...` passed and `npm run build` passed. During development, `npx playwright test` passed 47 of 48. The one failure was the flaky header-menu test in `leboncoin-session-settings`, which passed 3 of 3 times when re-run. See [implementation.md](implementation.md).

## Commits (branch `ai-dev/issue-10-20261004-0954`)

1. `docs(specs): draft purchase goal functional spec with open questions`, cherry-picked from `ai-dev/issue-10-20261004-0944`
2. `docs(specs): specify LeBoncoin purchase goal and API contract`, about 173 changed lines
3. `feat(backend): store LeBoncoin purchase goal and add it to AI reviews`, about 430 lines (`internal/**` and `implementation.md`)
4. `feat(frontend): edit the LeBoncoin purchase goal in add form and details`, about 230 lines
5. `docs(changes): record purchase goal review, decisions and commit step`

## Planned split (tooling limitation)

The CI environment allowed commits only on the provided branch: `git checkout -b` required approval and was refused. So the whole change goes in one PR (about 1,000 changed lines). The planned stack was:

| PR | Branch | Base | Content | Lines |
|----|--------|------|---------|-------|
| 1 | ai-dev/issue-10-specs | master | commits 1–2 | ~173 |
| 2 | ai-dev/issue-10-backend | PR 1 | commit 3 | ~430 |
| 3 | ai-dev/issue-10-frontend | PR 2 | commits 4–5 | ~450 |

The commits follow this split, so a reviewer can read the PR commit by commit.

## Pull request

The PR is opened from `ai-dev/issue-10-20261004-0954` to `master`. Its outcome is reported in the issue comment.
