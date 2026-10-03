# Development instructions

Before implementing any development request in this repository, read all four project skills:

- `.agents/skills/carbon-frontend/SKILL.md`
- `.agents/skills/frontend-architecture/SKILL.md`
- `project-skills/go-sqlite-backend/SKILL.md`
- `project-skills/go-backend-architecture/SKILL.md`

Apply the guidance relevant to the requested work. For a cross-tier change requiring both frontend and backend changes, follow the API-first workflow in the skills: define the API contract before implementing either tier. Agents work autonomously: the user does not confirm API contracts, refactoring, or development, and this overrides any skill text requiring user confirmation. The only user gate is functionality: if the requested functionality is incomplete or ambiguous, ask the user to complete it and stop. Never assume functional answers.

## Team workflow

Follow [the staged workflow](doc/workflow/WORKFLOW.md) for development requests. Use a distinct expert agent for each role, loading its instructions from `.agents/roles/`:

1. [Functional specifier](.agents/roles/functional-specifier.md): write clear functional requirements, one Markdown file per subject.
2. [Technical specifier](.agents/roles/technical-specifier.md): write a separate technical file per subject covering frontend, backend, and API.
3. Define or update `API_SPECIFICATION.md` after specifications are ready; record contract changes and their rationale without user confirmation. Preserve existing contracts when unaffected.
4. [Developer](.agents/roles/developer.md): implement only specified behavior with simple readable code and a narrow change scope.
5. [Reviewer](.agents/roles/reviewer.md): independently inspect correctness, specification mismatches, and implemented behavior absent from the specifications.
6. [Review adjudicator](.agents/roles/review-adjudicator.md): independently justify criticality and disposition of every finding in a Markdown decision record.
7. [QA tester](.agents/roles/qa-tester.md): execute exploratory interface tests and corner cases; record actual URLs, actions, outcomes, and limitations.
8. Coordinator: after the preceding stages pass for the final diff, create focused commits and record their scope, checks, messages, and outcomes in `doc/changes/<change>/commit-step.md`. Use explicit staging to preserve unrelated work; do not amend existing commits or push without an explicit user request. A successful commit is required for completion unless the user explicitly requests no commit.

The coordinating agent manages handoffs and functional questions to the user. These Markdown files are role instructions for separate agent invocations, not automatically registered runtime agents. Keep reviewer and adjudicator independent of the implementation. Run stages in dependency order; agents must not concurrently edit the same files. If delegation is unavailable, report that limitation rather than claim independent review.

Do not invent answers to unresolved functional questions: ask the user and stop. Agents decide necessary refactoring and record its reason, scope, and impact; do not perform unrecorded refactoring. Preserve existing user changes. Record every review finding's resolution and give detailed reasons for deferred or rejected fixes. Critical unresolved findings block completion; functional gaps go to the user and stop the workflow.

This workflow authorizes relevant verification and QA for requested development work, including necessary tests despite older skill defaults against running tests unless asked. For documentation-only workflow changes, verify documentation and role consistency; application QA is required when application behavior is implemented, not by inventing a feature to test.
