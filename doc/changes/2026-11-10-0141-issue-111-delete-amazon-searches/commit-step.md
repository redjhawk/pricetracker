# Commit step (issue #111)

Stage: pull-request

## Gates before pushing

- Review: [review.md](review.md), REV-001–005. Decisions: [decisions.md](decisions.md). Critical REV-001 was fixed with use case UC-PC-02 and the regression test `TestRefreshItemRestartsStoppedItemWhileRefreshAllQueued`, which fails before the fix and passes after it. REV-002–004 are fixed.
- The decision summary was posted on issue #111.
- QA: [qa.md](qa.md). The API and service cases pass, with no defects. Browser QA is **blocked** because Chromium needs `libnspr4.so`, which this runner does not have. The UI states were checked in the source only.
- Checks on the tip (`ai-dev/issue-111-review-fixes`): `go vet ./...`, `go test ./...` and `npm run build` pass.
- REV-005 limitation: the developer ran the three checks on each part when committing it. The coordinator could not re-run them on each branch alone, because `git worktree` and checking out another branch need approval in this session.

## Stack (each PR targets the previous branch; PR 1 targets `ai-dev/issue-111-feature`)

| # | Branch | Commit | Changed lines |
|---|---|---|---|
| 1 | ai-dev/issue-111-docs | docs: specify Amazon searches removal and per-item stop | 331 |
| 2 | ai-dev/issue-111-frontend-search-items | refactor(frontend): remove Amazon search items page and Amazon AI review | 279 |
| 3 | ai-dev/issue-111-frontend-tab | refactor(frontend): remove the Amazon searches tab | 414 |
| 4 | ai-dev/issue-111-backend-http | refactor(httpapi): remove the Amazon searches routes | 291 |
| 5 | ai-dev/issue-111-service-searches | refactor(service): remove the Amazon search use cases | 400 |
| 6 | ai-dev/issue-111-search-worker | refactor(service): remove the Amazon search worker and request windows | 483 |
| 7 | ai-dev/issue-111-amazon-review | refactor: remove Amazon AI reviews | 275 |
| 8 | ai-dev/issue-111-amazon-search-model | refactor: remove the Amazon search parser and search item handling | 416 |
| 9 | ai-dev/issue-111-store-searches | refactor(store): remove the Amazon search store | 486 |
| 10 | ai-dev/issue-111-search-migration | feat(store): delete saved Amazon searches on upgrade | 118 |
| 11 | ai-dev/issue-111-item-stop-backend | feat: stop Amazon checks per item after five failures | 449 |
| 12 | ai-dev/issue-111-item-stop-frontend | feat(frontend): show the per-item Amazon stop | 106 |
| 13 | ai-dev/issue-111-review-fixes | fix: restart stopped Amazon items during Refresh all; docs: QA and commit records | 312 + this record |

The "refactor" parts remove the feature; they are not unrelated refactoring. The final PR from `ai-dev/issue-111-feature` to `master` is opened after every part has merged.

## Deviation

Each part was committed locally before review so that it could be measured and built on its own. Nothing was pushed before QA finished. Review fixes were added as a new commit, without amending.
