# PR comment triage: 2026-10-07-1117-issue-56-amazon-searches

Triage agent: independent PR comment triage agent (not the developer or the PR reviewer). Each comment was checked against the code at the stack tip (`ai-dev/issue-56-p10-searches-ui`, which contains every part) and against `doc/specifications/amazon-*/`. Comment links were not available (see `pr-review.md`); the review IDs listed there identify the source review of each row.

## Rounds

### Round 1 (2026-10-07, PRs #83–#92, heads in `pr-review.md`)

| PR | Comment (file:line) | Label | Decision | Reason | Outcome |
|---|---|---|---|---|---|
| #84 | `API_SPECIFICATION.md:626` | nit | non-blocking | Message wording only; behavior accepts http and https as specified. | `doc/todo/2026-10-07-amazon-searches-backend-cleanups.md` |
| #85 | `internal/store/amazon_requests.go:13` | readability | non-blocking | Internal name `blocked` vs API "stopped"; the API maps it correctly. | `doc/todo/2026-10-07-amazon-searches-backend-cleanups.md` |
| #85 | `internal/store/searches.go:173` | test | non-blocking | Missing documentation/test of the unique canonical URL rule; no wrong behavior shown. | `doc/todo/2026-10-07-amazon-searches-test-gaps.md` |
| #86 | `internal/amazon/collector.go:131` | readability | non-blocking | Log detail loss; FR-AMZ-HUMAN-006 still logs URL, status and response excerpt. | `doc/todo/2026-10-07-amazon-request-logging.md` |
| #86 | `internal/amazon/collector.go:214` | nit | non-blocking | `continue` vs `break` gives the same result for a budget check; style. | `doc/todo/2026-10-07-amazon-searches-backend-cleanups.md` |
| #86 | `internal/amazon/collector.go:227` | simplification | non-blocking | Per-call regexp compile is a small inefficiency. | `doc/todo/2026-10-07-amazon-searches-backend-cleanups.md` |
| #87 | `internal/service/amazon_gate.go:49` | bug | **blocking** | Verified: `do` counts any `request_error`, and a request aborted by context cancellation (shutdown/deploy) returns `request_error`. Five such aborts persist `Blocked=true` and stop all Amazon requests, violating FR-AMZ-HUMAN-007 (only failed Amazon requests count). | returned to developer |
| #87 | `internal/service/amazon_gate.go:32` | nit | non-blocking | Load failure is logged; startup DB failure is an edge case already logged. | `doc/todo/2026-10-07-amazon-request-logging.md` |
| #87 | `internal/service/service.go:59` | nit | non-blocking | Field introduced one PR early; harmless in the stack. | `doc/todo/2026-10-07-amazon-searches-backend-cleanups.md` |
| #87 | `internal/service/service.go:316` | nit | non-blocking | Comment partly wrong: FR-AMZ-HUMAN-009 is met in the UI, the Amazon tab of `TrackedItemsPage.tsx` shows the stopped notification. Only the log is terse. | `doc/todo/2026-10-07-amazon-request-logging.md` |
| #88 | `internal/model/model.go:156` | readability | non-blocking | `Review any` design preference; no bug. | `doc/todo/2026-10-07-amazon-searches-backend-cleanups.md` |
| #88 | `internal/service/amazon_review.go:53` | nit | non-blocking | Misleading error message for a rare store failure. | `doc/todo/2026-10-07-amazon-searches-backend-cleanups.md` |
| #88 | `internal/service/review_test.go:48` | nit | non-blocking | Test fake detail; test is still valid. | `doc/todo/2026-10-07-amazon-searches-test-gaps.md` |
| #89 | `internal/service/search_worker.go:73` | bug | **blocking** | Verified: `run` does `continue` after `runPass` without sleeping. When `runPass` returns early with no progress (`SearchItemIDs` error, `canContinue` false from a `store.Search` error), the search stays due and the loop spins with no pause: CPU/log flood, and repeated results page requests possible, against the human pauses of FR-AMZ-HUMAN-004. | returned to developer |
| #89 | `internal/service/search_worker.go:166` | bug | **blocking** | Verified: in `openResults`, a `captureSearch` error is only logged and the function returns `true`; the search has no items and no `lastError`, and the pass ends as done. The user sees an empty search with no error, against FR-AMZ-SEARCH-006 and FR-AMZ-HUMAN-006 (a failure is put aside and retried). | returned to developer |
| #89 | `internal/service/searches.go:98` | readability | non-blocking | The assertion only applies to Amazon search items whose reviews are always `*AmazonAIReviewContent`; a checked form is safer but not a current bug. | `doc/todo/2026-10-07-amazon-searches-backend-cleanups.md` |
| #89 | `internal/service/search_worker.go:213` | nit | non-blocking | Open/read are not logged; FR-AMZ-HUMAN-004 behavior (order and pauses) is implemented, only the log evidence is partial. | `doc/todo/2026-10-07-amazon-request-logging.md` |
| #89 | `internal/service/searches.go:131` | nit | non-blocking | Error message wording. | `doc/todo/2026-10-07-amazon-searches-backend-cleanups.md` |
| #90 | `internal/httpapi/searches.go:79` | refactoring | non-blocking | Duplicated strict-JSON decode is existing pattern; a shared helper belongs in its own PR, as the reviewer says. | `doc/todo/2026-10-07-amazon-searches-backend-cleanups.md` |
| #90 | `internal/service/searches_test.go:56` | test | non-blocking | Ignored error in test setup. | `doc/todo/2026-10-07-amazon-searches-test-gaps.md` |
| #90 | `internal/service/search_worker_test.go:110` | test | non-blocking | Missing tests; the capture-failure regression test is required anyway by the #89 blocking fix. Remaining cases deferred. | `doc/todo/2026-10-07-amazon-searches-test-gaps.md` |
| #91 | `src/components/AmazonSearchesPage.tsx:95` | bug | **blocking** | Verified: `handleDelete` has no in-flight state, so the modal stays enabled; a double submit sends a second DELETE and shows a spurious "Amazon search was not found." error. | returned to developer |
| #91 | `src/components/AmazonSearchesPage.tsx:130` | nit | non-blocking | Stale error text while editing; UX polish. | `doc/todo/2026-10-07-amazon-searches-frontend-cleanups.md` |
| #91 | `src/components/AmazonSearchesPage.tsx:46` | simplification | non-blocking | Duplicated `errorMessage`; refactoring for its own PR. | `doc/todo/2026-10-07-amazon-searches-frontend-cleanups.md` |
| #91 | `internal/httpapi/searches_test.go:104` | test | non-blocking | Unrealistic fixture; test still checks the mapping. | `doc/todo/2026-10-07-amazon-searches-test-gaps.md` |
| #92 | `src/components/AmazonSearchItems.tsx:172` | bug | **blocking** | Verified: search item rows have Details and Track actions but no link to the Amazon listing. FR-AMZ-SEARCH-006 explicitly requires "link to the listing". | returned to developer |
| #92 | `src/App.tsx:310` | bug | non-blocking | Back goes to `/` and `TrackedItemsPage` shows the in-memory selected tab; no specification defines back navigation from a search or from an item opened from a search. Navigation improvement, not a spec violation. | `doc/todo/2026-10-07-amazon-searches-frontend-cleanups.md` |
| #92 | `src/components/AmazonSearchItems.tsx:18` | readability | non-blocking | Import location of shared formatters. | `doc/todo/2026-10-07-amazon-searches-frontend-cleanups.md` |
| #92 | `src/components/AmazonSearchItems.tsx:29` | simplification | non-blocking | Duplicated rating labels and euro formatter; refactoring. | `doc/todo/2026-10-07-amazon-searches-frontend-cleanups.md` |
| #92 | `src/components/TrackedItemsPage.tsx:65` | nit | non-blocking | Extra refetch per poll; no wrong state shown. | `doc/todo/2026-10-07-amazon-searches-frontend-cleanups.md` |

Blocking items: 5 (#87 ×1, #89 ×2, #91 ×1, #92 ×1). The developer fixes them on their PR branches, the PR reviewer reviews again, and round 2 is triaged here. Issue creation for the todo files is left to the coordinator (this run's triage was told not to create issues).

## Merge

- No blocking comments remain: no (round 1 has 5 blocking comments)
- Planned merge order (PR, base, method), once no blocking comment remains:
  1. #83 → `ai-dev/issue-56-feature`, merge commit
  2. #84 → `ai-dev/issue-56-feature`, merge commit
  3. #85 → `ai-dev/issue-56-feature`, merge commit
  4. #86 → `ai-dev/issue-56-feature`, merge commit
  5. #87 → `ai-dev/issue-56-feature`, merge commit
  6. #88 → `ai-dev/issue-56-feature`, merge commit
  7. #89 → `ai-dev/issue-56-feature`, merge commit
  8. #90 → `ai-dev/issue-56-feature`, merge commit
  9. #91 → `ai-dev/issue-56-feature`, merge commit
  10. #92 → `ai-dev/issue-56-feature`, merge commit
  11. Final PR `ai-dev/issue-56-feature` → `master`, squash, `Closes #56`
  Retarget each next part to `ai-dev/issue-56-feature` before merging the previous part.
- Merge results and deploy: reported on the issue #56 comment and the final run report, not in this file (it is committed before the first merge)
