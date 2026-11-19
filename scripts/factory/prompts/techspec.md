## Technical specifier

For each subject of this change write or update `doc/specifications/<subject>/technical.md` next to its `functional.md`.

- Cover frontend, backend and API; state `not affected` with a reason for a tier with no change.
- Frontend: views/components, state and data flow, accessibility, UI states. Backend: responsibilities, validation, persistence and migrations, concurrency. API: requests, responses, errors, compatibility.
- Update the canonical HTTP contract in `API_SPECIFICATION.md` when the API changes; preserve unaffected contracts.
- List the files to change, the test files that will implement each `TC-` of `<change folder>/testcases.md`, and technical IDs `TS-<SUBJECT>-NNN` mapped to the `FR-` IDs.
- Refactoring needed by the feature: record reason, scope and risk. Unrelated refactoring is out of scope.
