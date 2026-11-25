# No use cases for issue #200

This change is documentation maintenance with no user-visible behaviour, so it has no use cases.

It corrects the wording of TS-RESPONSIVE-017 in `doc/specifications/responsive-layout/technical.md`, which said that no card cell grows. That contradicts TS-RESPONSIVE-023 and the shipped layout in `src/index.css`, where the item cell grows into the space left on its line and the actions cell is pushed to the line end (see `doc/specifications/responsive-layout/functional.md`). The requirements cover only documentation text:

- **FR-RESPONSIVE-028**: the TS-RESPONSIVE-017 entry states that value cells do not grow, that only the item cell grows into the space left on its line, and that the actions cell is pushed to the line end, and it refers to TS-RESPONSIVE-023.
- **FR-RESPONSIVE-029**: `technical.md` contains neither "Cells do not grow" nor "the free space left on a line is always smaller than the next cell's width".
- **FR-RESPONSIVE-030**: FR-RESPONSIVE-001 to 027, TS-RESPONSIVE-023, the issue #140, #165 and #203 change records and `doc/todo/` are unchanged, the diff outside `doc/` is empty, `go vet ./...` and `go test ./...` pass, and the user observes no change in the application.

No production code, API, database, user interface or runtime behaviour changes, and no user goal or flow is involved. The existing responsive-layout use cases (UC-RL-01, UC-RL-02) describe the shipped layout and stay valid as they are. The test cases and the deterministic gates check these requirements directly.
