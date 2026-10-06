# Workflow feature branch: technical specification

Status: ready
Requirements: [functional](functional.md).

- **TS-WORKFLOW-FB-001** Branch naming: `ai-dev/issue-<n>-feature` in ai-dev runs, because only `ai-dev/` branches may be pushed there, and `feature/<change>` otherwise. The branch is created from `master` and pushed before any part.
- **TS-WORKFLOW-FB-002** Parts are stacked: PR 1 targets the feature branch, and each later PR targets the previous part branch. After a part merges, the next PR is retargeted to the feature branch. This keeps each part's diff reviewable and within the size limit.
- **TS-WORKFLOW-FB-003** The final PR (feature→`master`) is ready, lists the parts in merge order, says it must be merged last, and carries `Closes #<n>`. It is exempt from the 500-line limit because it only aggregates parts that were already measured, and the ai-dev size check skips the head `ai-dev/issue-<n>-feature`.
- **TS-WORKFLOW-FB-004** A change that fits in one PR targets `master` directly, without a feature branch, because a single merge already deploys once.
- **TS-WORKFLOW-FB-005** The ai-dev safety-net PR step targets `ai-dev/issue-<n>-feature` with `Part of #<n>` when that branch exists, so a fallback PR never bypasses the feature branch. The feature branch fetches master and branches from `origin/master`, so it is an exact copy.
- FR-WORKFLOW-PR-007 is amended to point here. `ci-deploy.yml` is unchanged because it already deploys only on pushes to `master`/`main`. Frontend, backend and API are not affected.
- Verification: `git diff --check`, a YAML parse of ai-dev.yml, and a consistency grep.
