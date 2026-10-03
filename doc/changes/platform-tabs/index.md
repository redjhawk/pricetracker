# Platform tabs

Stage: commit; all specification, implementation, review, decision and QA gates passed. See [commit preparation and Git completion evidence](commit-step.md).
User request: separate Amazon and LeBoncoin into tabs; omit second-hand offers from LeBoncoin.

## Stage results

1. [Functional specification](../../specifications/platform-tabs/functional.md) and [handoff](functional-step.md).
2. [Technical specification](../../specifications/platform-tabs/technical.md) and [handoff](technical-step.md).
3. [API assessment](api-step.md).
4. [Implementation and test-first evidence](implementation.md).
5. [Independent review](review.md).
6. [Independent decisions](decisions.md).
7. [Browser QA](qa.md).
8. [Commit handoff](commit-step.md).

## Scope

Frontend-only grouping of the existing `platform` field and platform-specific columns. Preserve collection-wide refresh and existing item operations. Use Carbon tabs. A focused failing browser test precedes UI implementation, consistent with the user's earlier TDD request; no broad architecture refactor or unrelated test campaign is included.

## Handoffs and decisions

- Functional specifier: `/root/platform_tabs_functional`; FR-PLATFORM-TABS-001–008 ready.
- Technical specifier: `/root/platform_tabs_technical`; TS-PLATFORM-TABS-001–006 ready.
- Developer: `/root/platform_tabs_developer`; implementation complete, source frozen. Core browser test failed before UI changes; eight focused cases, build, and test/config type check passed afterward.
- Independent reviewer: `/root/platform_tabs_reviewer`; no findings, with reviewed file fingerprints recorded.
- Independent adjudicator: `/root/platform_tabs_adjudicator`; no critical blocker, all reviewed fingerprints match, no findings requiring a fix, deferral, or rejection.
- Independent QA tester: `/root/platform_tabs_qa`; eight durable cases, four desktop/narrow platform combinations and a seeded 24-step exploratory sequence passed. No defects; reviewed source fingerprints remain unchanged.
- Coordinator: `/root`; preparing one focused commit after final documentation and staging checks. Git's successful commit output and the final handoff establish completion; this index is recorded before that commit.
- Existing API preserved; no new contract approval required. No refactoring proposed.
- The requested platform separation and LeBoncoin column omission come directly from the user. Supporting defaults and preserved behavior are identified in the functional specification; they are not represented as separate user approvals.
- No unresolved requirement or API questions block implementation.
