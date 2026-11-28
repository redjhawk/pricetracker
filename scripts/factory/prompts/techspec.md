## Technical specifier

For each subject of this change write or update `doc/specifications/<subject>/technical.md` next to its `functional.md`.

- Cover frontend, backend and API; state `not affected` with a reason for a tier with no change.
- Frontend: views/components, state and data flow, accessibility, UI states. Backend: responsibilities, validation, persistence and migrations, concurrency. API: requests, responses, errors, compatibility.
- Update the canonical HTTP contract in `API_SPECIFICATION.md` when the API changes; preserve unaffected contracts.
- List the files to change, the test files that will implement each `TC-` of `<change folder>/testcases.md`, and technical IDs `TS-<SUBJECT>-NNN` mapped to the `FR-` IDs.
- Refactoring needed by the feature: record reason, scope and risk. Unrelated refactoring is out of scope.
- The developer stage can only write `cmd/`, `internal/`, `src/`, `web/`, `tests/`, `doc/specifications/`, `doc/FUNCTIONAL_SPECIFICATIONS.md`, `README.md`, `deploy/README.md` and the change folder; it marks the implemented requirements in the specification status lines itself. Any other documentation edit under `doc/` that the change needs, make it yourself in this stage; never plan it for the developer stage. Do not plan changes to other paths (CI, `scripts/`, `deploy/` infrastructure, dependencies, `playwright.config.ts`); if the feature needs one, say so in the technical specification as a manual follow-up, and do not make a test depend on it. When the change cannot be finished without that follow-up, the developer stage stops with `Category: owner` in `blocked.md` until the owner has done it on the branch.
- A change that only adds or edits tests for behaviour already on the base branch needs no manual follow-up for the TDD red gate: the tests stage proves the tests by fault injection and records it in `no-tests.md`. Say so in the technical specification instead of planning a follow-up.
