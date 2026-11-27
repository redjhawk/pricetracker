# Test cases: fix the stale doc comment on Store.PurchaseGoals (issue #373)

Sources: functional specification [purchase-goal-choice](../../specifications/purchase-goal-choice/functional.md) (FR-GOAL-CHOICE-017 to FR-GOAL-CHOICE-019). No new use cases: see [no-use-cases.md](no-use-cases.md); the unchanged behaviour is that of the existing purchase goal choice flows (FR-GOAL-CHOICE-001 to 016).

Conventions:

- **Base:** `origin/master` (or `master`), compared with the change applied on top of it.
- **The new comment line:** `// PurchaseGoals returns the distinct non-empty purchase goals of the owner's tracked items (LeBoncoin and Amazon), unordered.`
- **The old comment line:** `// PurchaseGoals returns the distinct non-empty purchase goals of the owner's LeBoncoin items, unordered.`
- **Existing tests:** `TestPurchaseGoals_*` (`internal/service/purchase_goal_choice_test.go`, including `TC003_AmazonAdditionJoinsList`, which proves Amazon goals are listed), `TestPurchaseGoalsEndpoint_*` (`internal/httpapi/purchase_goal_choice_test.go`) and `tests/leboncoin-purchase-goal.spec.ts` (Playwright).
- The change only rewrites one source comment line. FR-GOAL-CHOICE-019 allows no other file change outside `doc/`, so no new Go or Playwright test code is written: the cases are repository checks (`go doc`, `grep`, `gofmt`, `git diff`) and runs of the existing suites. Narrow screens, keyboard, empty results and screen states do not apply; no API, database or user interface changes.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-GOAL-CHOICE-017 | TC-001, TC-002 |
| FR-GOAL-CHOICE-018 | TC-002, TC-003 |
| FR-GOAL-CHOICE-019 | TC-002, TC-003 |

| Use case | Test cases |
| --- | --- |
| none (no use cases for this change) | — |

## Cases

### TC-001 — `PurchaseGoals` has the new doc comment directly above it

- Covers: FR-GOAL-CHOICE-017
- Level: repository check (`go doc`, `grep`, source content)
- Given the change applied on top of the base
- When `go doc ./internal/store Store.PurchaseGoals` runs, the lines of `internal/store/store.go` just above `func (s *Store) PurchaseGoals(ctx context.Context, ownerID int64) ([]string, error) {` are read, and `grep -n "owner's LeBoncoin items" internal/store/store.go` runs
- Then `go doc` shows the signature followed by "PurchaseGoals returns the distinct non-empty purchase goals of the owner's tracked items (LeBoncoin and Amazon), unordered."; the line just above the function declaration is exactly the new comment line; the line above it is not a comment line (blank, the end of the previous function); `grep` prints nothing
- Corner cases:
  - the new comment line appears exactly once in the file and the old comment line appears nowhere;
  - no blank line between the comment and the function declaration;
  - on the base, the `grep` prints the old comment line (the check fails before the change).

### TC-002 — The diff outside `doc/` only rewrites the comment line

- Covers: FR-GOAL-CHOICE-018, FR-GOAL-CHOICE-019, FR-GOAL-CHOICE-017
- Level: repository check (`git diff`, `gofmt`)
- Given the change applied on top of the base
- When `git diff --name-status <base> -- . ':!doc'`, `git diff --numstat <base> -- internal/store/store.go`, `git diff <base> -- internal/store/store.go`, `git diff --name-only <base> -- doc/todo doc/changes ':!doc/changes/2026-11-27-2251-issue-373-fix-stale-doc-comment-on-store-purchaseg'` and `gofmt -l internal/store` run
- Then the only file listed outside `doc/` is `M internal/store/store.go`; its numstat is `1 1`; the removed line is the old comment line and the added line is the new comment line; the signature, query and body of `PurchaseGoals` (no platform filter) are untouched; no file under `doc/todo/` (including `doc/todo/issue-304-r3-01.md`) or under the issue #304 and #305 change folders is changed; `gofmt -l` prints nothing
- Corner cases:
  - no file is added, deleted or renamed outside `doc/` (no new `_test.go`, frontend source, `.spec.ts`, `go.mod`, `go.sum`, `package.json` or `API_SPECIFICATION.md` change);
  - the rows FR-GOAL-CHOICE-001 to 016 of `doc/specifications/purchase-goal-choice/functional.md` are unchanged (`git diff <base>` of that file adds lines only in the issue #373 sections, status line, open questions and traceability).

### TC-003 — Purchase goal choice behaviour unchanged, gates pass

- Covers: FR-GOAL-CHOICE-018, FR-GOAL-CHOICE-019 (regression of FR-GOAL-CHOICE-001 to 016)
- Level: integration (existing service and HTTP handler tests with real SQLite), e2e (existing Playwright spec) and suite run (deterministic gates)
- Given the change applied on top of the base, with the existing tests unchanged
- When `go test -count=1 -run 'TestPurchaseGoals_|TestPurchaseGoalsEndpoint_' ./internal/service/ ./internal/httpapi/` runs, then `go vet ./...`, `go test ./...`, and the other gates (`node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test`, including `tests/leboncoin-purchase-goal.spec.ts`)
- Then every named test passes, as on the base: the goal list still holds the distinct non-empty goals of the owner's LeBoncoin and Amazon items (`TC003_AmazonAdditionJoinsList`), in alphabetical order, protected in protected mode; `go vet ./...` and `go test ./...` pass, and so do the other gates, with the same test list as on the base
- Corner cases:
  - the results are the same without cached results (`-count=1`) and when the gates run twice;
  - the operator observes no change in the application (no API, database, setting or screen change: follows from TC-002).
