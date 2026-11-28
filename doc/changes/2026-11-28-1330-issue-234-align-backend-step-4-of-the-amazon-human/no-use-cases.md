# No use cases for issue #234

This change is documentation maintenance with no user-visible behaviour, so it has no use cases.

It corrects step 4 of the Backend `Collect` steps in `doc/specifications/amazon-human-checks/technical.md`, which says the HTTP cache is cleared before the failure "with a 2xx response". The server also clears it when the browser recorded no response within the 1 s wait, as the TS-AMZ-HCHK-017 row and its Backend bullet already say (see `doc/specifications/amazon-human-checks/functional.md`). The requirements cover only documentation text:

- **FR-AMZ-HCHK-017**: step 4 contains `with a 2xx response or none (a response may come after the 1 s wait), the HTTP cache is cleared first (issue #217, TS-AMZ-HCHK-017)`.
- **FR-AMZ-HCHK-018**: `technical.md` no longer contains `with a 2xx response the HTTP cache is cleared first`, and step 4 still contains `also when the check is interrupted` and `clears it too (issue #233, TS-AMZ-HCHK-018–019)`.
- **FR-AMZ-HCHK-019**: FR-AMZ-HCHK-001 to 016, the TS-AMZ-HCHK-017 row and its Backend bullets, every other technical entry, the change records of other issues and `doc/todo/` (including `doc/todo/issue-217-r2-02.md`) are unchanged, the diff outside `doc/` is empty, `go vet ./...` and `go test ./...` pass, and the operator observes no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no user goal or flow is involved. The existing use cases UC-AMZ-HCHK-007 to UC-AMZ-HCHK-009 describe the shipped behaviour that the cache clearing serves (FR-AMZ-HCHK-014–016) and stay valid as they are. The test cases and the deterministic gates check these requirements directly.
