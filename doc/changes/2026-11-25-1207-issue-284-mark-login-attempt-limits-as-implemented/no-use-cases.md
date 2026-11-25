# No use cases for issue #284

This change is documentation maintenance with no user-visible behaviour, so it has no use cases.

It updates the status of the Login attempt limits row in the specification index, because PR #283 delivered FR-LOGIN-LIM-001 to 011 (issue #266) (see `doc/specifications/login-attempt-limits/functional.md`). The requirements cover only documentation text:

- **FR-LOGIN-LIM-012**: the status cell of the Login attempt limits row of `doc/FUNCTIONAL_SPECIFICATIONS.md` begins with "implemented (issue #266)" and does not contain "specified (issue #266)".
- **FR-LOGIN-LIM-013**: only that row of the index changes; the other rows, including the other "specified (issue #NNN)" rows and the Multiple users and login row, are unchanged.
- **FR-LOGIN-LIM-014**: FR-LOGIN-LIM-001 to 011, the subject's `technical.md`, the issue #266 change records and `doc/todo/` are unchanged, the diff outside `doc/` is empty, `go vet ./...` and `go test ./...` pass, and the user observes no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no user goal or flow is involved. The existing login-attempt-limits use cases stay valid as they are. The test cases and the deterministic gates check these requirements directly.
