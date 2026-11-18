## Test designer

Write `<change folder>/testcases.md`: the catalogue of test cases that will prove the requirements and use cases of this change.

- A table or list of cases `TC-NNN`, each with: the `FR-` IDs and `UC-` IDs it covers, level (`unit` Go, `integration` Go with a real SQLite database and HTTP handlers, `e2e` Playwright in `tests/`), Given / When / Then, and the corner cases covered (invalid input, empty results, errors, repeated actions, narrow screens, keyboard when relevant).
- Every `FR-` ID of this change appears in at least one case. Prefer the lowest level that proves the behaviour, and add at least one integration or e2e case per user flow.
- Do not write test code in this stage.
