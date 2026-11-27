# No new automated tests for issue #373

The change rewrites one doc comment line above `func (s *Store) PurchaseGoals` in `internal/store/store.go`. A comment cannot make a test go red, and FR-GOAL-CHOICE-019 / TS-GOAL-CHOICE-012 allow no file change outside `doc/` other than `store.go`, so no test file is added or changed.

Where each test case is proved:

| Test case | Proved by |
| --- | --- |
| TC-001 | `go doc ./internal/store Store.PurchaseGoals` shows "PurchaseGoals returns the distinct non-empty purchase goals of the owner's tracked items (LeBoncoin and Amazon), unordered."; `grep -c "owner's tracked items (LeBoncoin and Amazon), unordered." internal/store/store.go` is `1`; `grep -n "owner's LeBoncoin items" internal/store/store.go` prints nothing; the line after the comment is `func (s *Store) PurchaseGoals(ctx context.Context, ownerID int64) ([]string, error) {` and the line before it is blank. On the base, the `grep` prints line 833 (the old comment). |
| TC-002 | `git diff --name-status master -- . ':!doc'` lists only `M internal/store/store.go`; `git diff --numstat master -- internal/store/store.go` is `1 1` (old comment line removed, new one added); `git diff --name-only master -- doc/todo doc/changes/2026-11-24-0833-issue-304-ai-purchase-goal-let-the-user-to-choose doc/changes/2026-11-24-0836-issue-305-purchase-goal-and-ai-review-is-also-avai` prints nothing; `gofmt -l internal/store` prints nothing. |
| TC-003 | Existing, unchanged tests: `go test -count=1 -run 'TestPurchaseGoals_\|TestPurchaseGoalsEndpoint_' ./internal/service/ ./internal/httpapi/` (including `TestPurchaseGoals_TC003_AmazonAdditionJoinsList`), then the gates `go vet ./...`, `go test ./...`, `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` (including `tests/leboncoin-purchase-goal.spec.ts`). |
