# Development instructions

Before implementing any development request in this repository, read the project skills relevant to the work:

- `.agents/skills/carbon-frontend/SKILL.md`
- `.agents/skills/frontend-architecture/SKILL.md`
- `project-skills/go-sqlite-backend/SKILL.md`
- `project-skills/go-backend-architecture/SKILL.md`

For a cross-tier change, define the API contract in `API_SPECIFICATION.md` before implementing either tier. Agents decide API contracts, refactoring and technical design without asking. The only user gate is functionality: if the requested functionality is incomplete or ambiguous, ask the user and stop. Never assume functional answers.

## Team workflow

Feature work and behaviour changes go through the agent factory described in [doc/workflow/WORKFLOW.md](doc/workflow/WORKFLOW.md): specification, questions, use cases, test cases, technical specification, tests first (TDD) with integration and e2e tests, implementation, pull request, independent review, triage (blocking comments return to development, the rest become issues), merge, and the tagged release. Run it with `scripts/factory/factory.sh <issue>` or by labelling the issue `ai-dev`.

When you run as one of the factory's stage agents, the stage prompt is your task: do only that stage and do not commit, push or open pull requests.

For small changes made interactively outside the factory (CI, documentation, tooling), keep refactoring, dependency updates and formatting in their own commits or PRs, stage files explicitly to preserve unrelated work, and never amend existing commits.
