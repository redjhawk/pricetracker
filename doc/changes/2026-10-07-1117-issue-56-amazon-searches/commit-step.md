# Commit step (issue #56)

The work-in-progress branch `ai-dev/issue-56-20261007-1650` (about 4,000 changed lines, too big for one PR) was split into 10 stacked PRs on `ai-dev/issue-56-feature`. The final tree of part 10 is identical to that WIP tree. The split differs from the plan in `index.md` because of compile-time dependencies: the worker needs the search use cases and the AI review, and several service tests share the worker test helpers.

| Part | Branch | Content | Lines |
| --- | --- | --- | --- |
| 1 | `ai-dev/issue-56-p1-functional-spec` | Functional specifications, change index | 286 |
| 2 | `ai-dev/issue-56-p2-tech-spec` | Technical specifications, API contract | 376 |
| 3 | `ai-dev/issue-56-p3-store` | Model, `tracked` flag (R-1), search and request-state tables | 406 |
| 4 | `ai-dev/issue-56-p4-amazon-search` | Store tests, single-request product collection | 354 |
| 5 | `ai-dev/issue-56-p5-search-gate` | Search page parser, request gate (R-2), time windows | 426 |
| 6 | `ai-dev/issue-56-p6-ai-review` | Amazon AI review (R-3, R-4), gate tests | 488 |
| 7 | `ai-dev/issue-56-p7-searches-service` | Search use cases, search worker, startup wiring | 429 |
| 8 | `ai-dev/issue-56-p8-http-api` | HTTP API, service and worker tests | 467 |
| 9 | `ai-dev/issue-56-p9-searches-list` | HTTP tests, frontend API module, types, searches list page | 436 |
| 10 | `ai-dev/issue-56-p10-searches-ui` | Tab, search items page, AI review section, token text, Playwright | 433 |
| 11 | `ai-dev/issue-56-p11-workflow-records` | This record, PR review and triage records, todos (moved out of part 10 to keep it under 500 lines) | docs |

PRs: #83–#92 (parts 1–10) and #97 (part 11). The blocking findings of PR review round 1 were fixed with new commits on parts 5–10, which were then rebased and force-pushed; round 2 found nothing blocking. The sizes after the fixes are 430, 499, 437, 496, 442 and 433 lines for parts 5–10.

Known limitation: some tests arrive one or two parts after the code they cover (parts 3, 5, 7), because they share helpers defined with later code. Every part builds and passes `go vet ./...`, `go test ./...` and `npm run build` on its own.

Checks on part 10: `go vet ./...`, `go test -race ./...`, `npm run build` and `npx playwright test` (49 passed, including `tests/amazon-searches.spec.ts`).

Superseded PRs: #81 and #82 (one huge PR each) are superseded by this stack; this run was not permitted to close them, so the user was asked to.
