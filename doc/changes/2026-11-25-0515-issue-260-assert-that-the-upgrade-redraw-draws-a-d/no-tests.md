# No red phase for issue #260

The change is test code: one added block in `TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce` (`internal/service/amazon_human_checks_issue185_test.go`). The block checks behaviour that has already shipped (FR-AMZ-HCHK-005: `scheduleUnscheduledAmazonItems` calls `s.nextCheckAt` once per item). So it passes on `master`'s production code, and there is nothing for the development stage to implement. The test file is locked after this stage, and FR-TEST-HYG-093 allows no other file outside `doc/`, so this stage makes the change itself (TS-TEST-HYG-122). Run 2 shows the red state, and the development stage writes no code. It only sets the statuses.

## Delivered in `internal/service/amazon_human_checks_issue185_test.go`

Right after the first start's loop that fills `drawn`, and before the `redrawMark` check, the block of the technical specification is added verbatim:

```go
	// FR-AMZ-HCHK-005: each item's time is drawn on its own, so no two items share one (issue #260).
	for i, id := range ids {
		for _, other := range ids[i+1:] {
			if drawn[id].Equal(drawn[other]) {
				t.Fatalf("items %s and %s both have the next check %s after the start, want a time drawn for each item", id, other, drawn[id].In(paris))
			}
		}
	}
```

`git diff master` of the file has one hunk (`@@ -123,6 +123,14 @@`) that adds only these 8 lines. The test name, its comment, the seeds 1851 and 1852, the clocks, the item IDs, `former`, the other checks and messages, `TestAmazonMissedWindow_Issue185_TC001_SkippedRedrawnAndLogged`, `redrawMark`, `setRedrawMark` and the header comment are unchanged. No import changes.

## Runs

`service.go` was never edited. Each fault injection is an edited copy of `internal/service/service.go`, written under `/tmp/claude/inj260` and passed with `go test -overlay`. A scratch test added only through an overlay generated these copies. The overlay files and that generator are listed in `delete.txt`. Every injected time is still drawn by `s.nextCheckAt("amazon", s.now())`, so it is in the evening window and the redraw mark is still cleared. The run is `go test -count=1 -run '^TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce$' -v ./internal/service/`.

1. **No injection (TC-001).** The run passes: `--- PASS: TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce`. With seed 1851, the three drawn times differ. `-count=3` passes, and `go test -count=1 ./internal/service/...` passes.
2. **One time for every item (TC-002).** `shared := s.nextCheckAt("amazon", s.now())` comes before the loop, and the loop calls `SetNextCheck(ctx, id, shared)`. The run fails on the new check, not on the window or the mark check:

   ```text
   amazon_human_checks_issue185_test.go:130: items B000001853 and B000001854 both have the next check 2030-03-05 22:55:39 +0100 CET after the start, want a time drawn for each item
   ```

   The shared time is in the evening of 2030-03-05, not the former 08:00. With the same injection plus `master`'s test file (also given through the overlay), the run passes. That shows the gap that this change closes.
3. **One shared pair at a time (TC-003).** `shared` is drawn before the loop and stored for the two named items, which are selected by ID, so the store's order does not matter. The third item gets a time drawn in the loop. Each run fails:
   1. `B000001853` and `B000001854`: `items B000001853 and B000001854 both have the next check 2030-03-05 22:55:39 +0100 CET after the start, want a time drawn for each item`.
   2. `B000001854` and `B000001855`: `items B000001854 and B000001855 both have the next check 2030-03-05 22:55:39 +0100 CET after the start, want a time drawn for each item`.
   3. `B000001853` and `B000001855`: `items B000001853 and B000001855 both have the next check 2030-03-05 22:55:39 +0100 CET after the start, want a time drawn for each item`. A check that compares only neighbouring items would miss this pair.
4. **Comparison with `master` (TC-004).** `master`'s test file was given through the overlay with no injection, and the run printed `--- PASS: TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce`. With the change, the run prints the same test name and `PASS` (run 1). The diff is the single added hunk described above.
5. **Repository checks and gates (TC-006), run twice with the same result:**
   - `git diff --name-only master -- . ':!doc'` lists only `internal/service/amazon_human_checks_issue185_test.go`.
   - `git diff master` of the issue #185 change folder changes only its `testcases.md`. The technical specification stage added the TC-002 Then line and the corner case there.
   - `gofmt -l internal/service` prints nothing.
   - `go vet ./...` passes.
   - `go test -count=1 ./...` passes, with every package `ok`.

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | `TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce` with the pairwise check; run 1 |
| TC-002 | Run 2 |
| TC-003 | Run 3. Code check: the loop covers every pair of `ids` |
| TC-004 | Run 4. Code check of the one-hunk diff of the test file |
| TC-005 | Document check of `git diff master` of the issue #185 `testcases.md`: two added lines in TC-002 (run 5) |
| TC-006 | Run 5. The development stage runs the gates again |
