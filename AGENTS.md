# Development instructions

Before implementing any development request in this repository, read all four project skills:

- `.agents/skills/carbon-frontend/SKILL.md`
- `.agents/skills/frontend-architecture/SKILL.md`
- `project-skills/go-sqlite-backend/SKILL.md`
- `project-skills/go-backend-architecture/SKILL.md`

Apply the guidance relevant to the requested work. For a cross-tier change requiring both frontend and backend changes, follow the API-first workflow in the skills: propose the API contract and wait for the user's confirmation before implementing either tier. For work limited to one tier or using an already confirmed API, proceed under the applicable skills.

## Team workflow

Follow [the staged workflow](doc/workflow/WORKFLOW.md) for development requests. Use a distinct expert agent for each role, loading its instructions from `.agents/roles/`:

1. [Functional specifier](.agents/roles/functional-specifier.md): write clear functional requirements, one Markdown file per subject.
2. [Technical specifier](.agents/roles/technical-specifier.md): write a separate technical file per subject covering frontend, backend, and API.
3. Define or update `API_SPECIFICATION.md` after specifications are ready; obtain user confirmation for contract changes before implementation. Preserve approved contracts when unaffected.
4. [Developer](.agents/roles/developer.md): implement only specified, approved behavior with simple readable code and a narrow change scope.
5. [Reviewer](.agents/roles/reviewer.md): independently inspect correctness, specification mismatches, and implemented behavior absent from the specifications.
6. [Review adjudicator](.agents/roles/review-adjudicator.md): independently justify criticality and disposition of every finding in a Markdown decision record.
7. [QA tester](.agents/roles/qa-tester.md): execute exploratory interface tests and corner cases; record actual URLs, actions, outcomes, and limitations.

The coordinating agent manages handoffs and user decisions. These Markdown files are role instructions for separate agent invocations, not automatically registered runtime agents. Keep reviewer and adjudicator independent of the implementation. Run stages in dependency order; agents must not concurrently edit the same files. If delegation is unavailable, report that limitation rather than claim independent review.

Do not invent answers to unresolved functional questions. Propose necessary refactoring with its reason, scope, and impact; the user decides whether it precedes the feature or is deferred. Do not perform unapproved refactoring. Preserve existing user changes. Record every review finding's resolution and give detailed reasons for deferred or rejected fixes. Critical unresolved findings block completion; specification gaps requiring a product choice go to the user.

This workflow authorizes relevant verification and QA for requested development work, including necessary tests despite older skill defaults against running tests unless asked. For documentation-only workflow changes, verify documentation and role consistency; application QA is required when application behavior is implemented, not by inventing a feature to test.
