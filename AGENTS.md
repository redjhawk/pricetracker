# Development instructions

Before implementing any development request in this repository, read all four project skills:

- `.agents/skills/carbon-frontend/SKILL.md`
- `.agents/skills/frontend-architecture/SKILL.md`
- `project-skills/go-sqlite-backend/SKILL.md`
- `project-skills/go-backend-architecture/SKILL.md`

Apply the guidance relevant to the requested work. For a cross-tier change requiring both frontend and backend changes, follow the API-first workflow in the skills: define the API contract before implementing either tier. Agents work autonomously: the user does not confirm API contracts, refactoring, or development, and this overrides any skill text requiring user confirmation. The only user gate is functionality: if the requested functionality is incomplete or ambiguous, ask the user to complete it and stop. Never assume functional answers.

## Team workflow

Follow [the staged workflow](doc/workflow/WORKFLOW.md) for development requests. Use a distinct expert agent for each role, loading its instructions from `.agents/roles/`:

1. [Functional specifier](.agents/roles/functional-specifier.md): write clear functional requirements, one Markdown file per subject, and update `doc/FUNCTIONAL_SPECIFICATIONS.md` to summarize and link them.
2. [Technical specifier](.agents/roles/technical-specifier.md): write a separate technical file per subject covering frontend, backend, and API.
3. Define or update `API_SPECIFICATION.md` after specifications are ready; record contract changes and their rationale without user confirmation. Preserve existing contracts when unaffected.
4. [Developer](.agents/roles/developer.md): implement only specified behavior with simple readable code and a narrow change scope.
5. [Reviewer](.agents/roles/reviewer.md): independently inspect correctness, specification mismatches, and implemented behavior absent from the specifications.
6. [Review adjudicator](.agents/roles/review-adjudicator.md): independently justify criticality and disposition of every finding in a Markdown decision record. Confirm every critical fix, which needs a `doc/use-cases/` use case and an automated regression test. Publish an informational decision summary, without waiting for a reply, as an issue comment and in the final run report.
7. [QA tester](.agents/roles/qa-tester.md): before any commit or pull request, execute exploratory interface tests and corner cases; record actual URLs, actions, outcomes, and limitations.
8. Coordinator: after review decisions and QA pass for the final diff, create focused commits, push, and open ready (non-draft) pull requests with agent-written descriptions: to `master` for a single PR, or to a feature branch when split (below). Target 450±50 changed lines per PR when splitting; 500 is the hard ceiling (generated files excluded). Stack split PRs on a feature branch created from `master`: the part PRs merge into it, and a single final PR merges it into `master`, so only that last merge deploys. Keep refactoring, dependency updates, formatting, and unrelated documentation in their own PRs. Record scope, checks, messages, PRs, and outcomes in `doc/changes/<change>/commit-step.md` (change folders are named `<YYYY-MM-DD-HHMM>-issue-<n>-<slug>`, UTC start time, see the workflow); if tooling cannot create the required branches or PRs, record the planned split there and report it as a limitation. Use explicit staging to preserve unrelated work; do not amend existing commits. A successful commit is required for completion unless the user explicitly requests no commit.
9. [PR reviewer](.agents/roles/pr-reviewer.md): after the PRs are opened, an expert Go/JS agent reviews every PR line by line and posts inline GitHub review comments on possible bugs, simpler alternatives, unreadable code, and refactoring that belongs in a dedicated PR. It only comments; the user decides fixes. Possible bugs get "Request changes" (or a `CHANGES REQUESTED` comment when GitHub forbids it) and are reported to the user on the issue or in the final report.

The coordinating agent manages handoffs and functional questions to the user. These Markdown files are role instructions for separate agent invocations, not automatically registered runtime agents. Keep reviewer and adjudicator independent of the implementation. Run stages in dependency order; QA (7) runs before stage 8 commits and opens the PRs, and accepted QA fixes go through review and decisions before committing; agents must not concurrently edit the same files. If delegation is unavailable, report that limitation rather than claim independent review.

Do not invent answers to unresolved functional questions: ask the user and stop. Agents decide necessary refactoring and record its reason, scope, and impact; do not perform unrecorded refactoring. Preserve existing user changes. Record every review finding's resolution and give detailed reasons for deferred or rejected fixes. Critical unresolved findings block completion; functional gaps go to the user and stop the workflow.

This workflow authorizes relevant verification and QA for requested development work, including necessary tests despite older skill defaults against running tests unless asked. For documentation-only workflow changes, verify documentation and role consistency; application QA is required when application behavior is implemented, not by inventing a feature to test.
