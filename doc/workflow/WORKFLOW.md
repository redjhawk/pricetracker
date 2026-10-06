# Specification-led development workflow

This workflow applies to new functionality and behavior changes. The coordinator maintains the handoffs and uses separate expert agents for functional specification, technical specification, implementation, independent review, independent review decisions, QA, pull request review, and PR comment triage. A reviewer or decision agent must not review or adjudicate its own implementation. The decision agent is distinct from the reviewer. Available concurrency limits do not remove this separation: run the roles sequentially when needed.

## Overview: from issue to production

This summary is informational; the numbered sections below prevail.

**Trigger.** The user opens an issue labelled `ai-dev`, or comments `@claude` on an issue or pull request. The `ai-dev` workflow (`.github/workflows/ai-dev.yml`) starts on the barcelona-dev runner, installs dependencies, and creates an `ai-dev/issue-<n>-...` branch. Claude reads `AGENTS.md`, the four project skills, and this workflow.

| # | Stage | What happens | Output |
|---|---|---|---|
| 1 | Functional specifier | Turns the issue into testable requirements, one subject per file, and updates the global summary | `doc/specifications/<subject>/functional.md` (`FR-...`), `doc/FUNCTIONAL_SPECIFICATIONS.md` |
| gate | Functional gate | Anything missing or ambiguous is asked in the issue and the run stops; the user answers with `@claude ...` and the run resumes | Questions on the issue |
| 2 | Technical specifier | Frontend, backend, and API design; refactoring decisions | `doc/specifications/<subject>/technical.md` (`TS-...`) |
| 3 | API contract | Canonical HTTP contract updated before either tier is coded | `API_SPECIFICATION.md` |
| 4 | Developer | Implements only the specifications; runs builds and tests | Code and check results |
| 5 | Independent reviewer | Checks the change against the specifications and contract; flags unspecified behavior | `doc/changes/<change>/review.md` (`REV-...`) |
| 6 | Independent adjudicator | Critical or not; fix, defer, or reject. Fixes return to the developer and are rechecked by the reviewer. Fixed critical findings get a use case and an automated regression test. The decision summary is reported, without waiting for the user | `doc/changes/<change>/decisions.md`, issue comment, final run report |
| 7 | QA tester | Runs the app; Playwright scenarios, corner cases, exploratory sequences. Failures return through 4, 5, and 6, then are retested | `doc/changes/<change>/qa.md` (`QA-...`) |
| 8 | Coordinator | Focused commits (`Refs: #<n>`), push, and ready PRs: about 450 changed lines, 500 maximum; when split, stacked on a feature branch created from `master`; the final feature→`master` PR is created only after every part has merged; refactoring and dependency updates separate | `doc/changes/<change>/commit-step.md`, pull requests |
| 9 | PR reviewer (Go/JS expert) | Reviews every PR line by line; inline GitHub comments on bugs, simplicity, readability, and refactoring mixed into a feature PR. Comment only. Possible bugs: "Request changes" (or a `CHANGES REQUESTED` comment) | GitHub PR reviews, `doc/changes/<change>/pr-review.md` |
| 10 | Triage agent | Decides whether each comment blocks. Non-blocking: a `doc/todo/` file and a GitHub issue (a failure is reported, not fatal). Blocking: back to the developer, then re-review and triage again until none remain. Then merges the PRs in stack order and starts the deploy | `doc/changes/<change>/pr-triage.md`, `doc/todo/`, issues, merged PRs |

All artifacts are linked from `doc/changes/<change>/index.md`.

**Split changes (feature branch).** A change that fits in one PR targets `master` directly. A larger change is delivered like this (details in stage 8). The feature branch is `ai-dev/issue-<n>-feature` in `ai-dev` runs and `feature/<change>` otherwise:

```
master ──► ai-dev/issue-<n>-feature   (exact copy of origin/master, pushed first)
              ▲ PR 1 (part)  ◄── PR 2 (part)  ◄── PR 3 ...   merge into the feature branch: no deploy
              │
              └──── final PR: feature → master   created after all parts merge, "Closes #<n>": the only deploy
```

- Part PRs are at most 500 changed lines each. After one merges, the next is retargeted to the feature branch.
- Stage 8 opens only the part PRs: GitHub cannot open the final PR while the feature branch has no commits beyond `master`. The final PR is created after every part has merged into the feature branch. It lists the parts in merge order. It is exempt from the size check because it only aggregates parts that were already reviewed and measured.
- If Claude committed on a part branch without opening a PR, the `ai-dev` safety-net step targets the feature branch (with `Part of #<n>`) whenever that branch exists, so `master` is never reached early. For the feature branch itself, the fallback is the final PR to `master`.

**After the run.** The `ai-dev` workflow fails if any PR for the issue exceeds 500 changed lines, and opens a PR if Claude committed without opening one. The PR reviewer's comments (stage 9) are triaged in stage 10: blocking ones are fixed in a loop, non-blocking ones become `doc/todo/` files and issues, and the agent merges the PRs when nothing blocks. The user can still comment on a PR at any time; an `@claude ...` comment also ends with the PR reviewer and triage, so a fix run can merge and deploy automatically once nothing blocks. Such a comment on a PR or a diff line makes Claude push fixes to the same branch. No CI runs on pull requests (the repository is public, so fork code never runs on barcelona).

**After the merge.** The stage 10 agent merges the PRs. A push to `master` triggers `ci-deploy`, and for merges made by the `ai-dev` token, which trigger no workflows, the agent starts it explicitly (`.github/workflows/ci-deploy.yml`): `go vet`, `go test`, and `npm run build`, then deployment to the ARMv6 target device; a failing step stops the deployment. Merges into a feature branch do not deploy. For a split change, only the final feature-branch→`master` merge deploys, once. The PR that merges into `master` (the single PR, or the final feature-branch PR) says `Closes #<n>`, and GitHub closes the issue when it merges. Part PRs say `Part of #<n>`.

**Always.** Only functional questions go to the user; agents decide API, design, refactoring, and review outcomes and record the rationale. A critical unresolved finding blocks completion. No check, URL, or pass is reported unless it was actually executed.

## Autonomy and the functional gate

Agents develop autonomously. The user does not approve API contracts, technical designs, refactoring, implementation, review decisions, or commits; the agents decide these, record the rationale, and proceed.

The user owns functionality only. The coordinator and functional specification agent check at the start that the request fully defines the functional behavior. If any functional behavior, product choice, or acceptance criterion is missing, ambiguous, or contradictory, ask the user focused questions to complete the functionality and stop: do not start technical specification or implementation. Never assume or invent functional answers. If a later stage discovers a functional gap, stop at that point, record the blocker, and ask the user in the same way.

For workflow/documentation maintenance, apply the relevant stages to the documentation change; do not invent product requirements or API changes. Existing requirements and contracts may be reused with explicit references.

## Documents and traceability

Keep a separate functional Markdown file and a separate technical Markdown file for each subject. For example, the tracked-items list and item details are separate subjects. Use stable, descriptive subject slugs: `doc/specifications/<subject>/functional.md` and `doc/specifications/<subject>/technical.md`. Keep change reports under `doc/changes/<change>/`, including `review.md`, `decisions.md`, and `qa.md`. Name each change folder `<YYYY-MM-DD-HHMM>-issue-<n>-<slug>` ([requirements](../specifications/workflow-dated-changes/functional.md)), for example `2026-10-04-2102-issue-15-multi-user-login-phase1`. The datetime is in UTC and records when the change started, which is when the coordinator creates the folder; it never changes afterwards, so folders sort chronologically. `issue-<n>` is the work item (GitHub issue) number; omit that segment when there is no work item, as in `2026-10-06-2008-workflow-stage-order`. The slug is short, descriptive and kebab-case. Follow-ups to the same change stay in its folder. Use the starting templates in [`templates/functional.md`](templates/functional.md), [`templates/technical.md`](templates/technical.md), [`templates/review-decisions.md`](templates/review-decisions.md), and [`templates/qa.md`](templates/qa.md), and [`templates/pr-review.md`](templates/pr-review.md). Link existing `doc/FUNCTIONAL_SPECIFICATIONS.md` and `doc/use-cases/` where applicable rather than silently replacing their meaning.

The coordinator records the paths of all artifacts in the feature's index, `doc/changes/<change>/index.md`. The index contains scope, current stage, agent handoffs, user decisions, unresolved questions, and links to the subject specifications, review findings, decision record, and QA report. Record functional decisions only from user messages or existing specifications.

Assign stable requirement IDs such as `FR-ITEM-DETAILS-001`, technical IDs such as `TS-ITEM-DETAILS-001`, review IDs such as `REV-001`, and QA case IDs such as `QA-001`. Link technical design, implementation, review findings, and QA results to their source requirements. Preserve IDs when documents change.

Artifact states are `draft`, `needs-clarification`, `ready`, and `superseded`. A ready artifact has no unresolved question that affects its handoff. A feature progresses through `functional-specification`, `technical-specification`, `api-contract`, `implementation`, `review`, `review-decisions`, `qa`, `commit`, `pull-request`, `pr-review`, `pr-triage`, `merge`, and `complete`. Record a blocked stage and its concrete blocker instead of advancing it.

## 1. Expert functional specification agent

Start with an explicit list of requested functional subjects and behaviors. Read the user request and existing specifications. Write one functional file per subject before technical design begins, and update `doc/FUNCTIONAL_SPECIFICATIONS.md` in the same stage: summarize the new or changed behavior there and link each subject file from its subject specifications list. The subject file prevails for details. Skip this update only for changes that alter no product behavior (for example workflow or documentation maintenance), and record that reason in the change index.

Each file states purpose, actors, scope and exclusions, prerequisites, user-visible behavior, success and failure flows, input rules, empty/loading/error states where relevant, and objectively checkable acceptance criteria. Give every requirement an ID. Describe what the user observes without deciding implementation details.

Identify contradictions, undefined behavior, and missing product decisions. Ask the user focused questions about any functional ambiguity and stop the workflow until they answer. Do not convert guesses into requirements or expand the feature. An ambiguity affecting acceptance must be resolved by the user or a referenced existing decision before the next stage. Explicit requirements already supplied by the user need no repeated confirmation.

Handoff: subject files and their requirement IDs, the `FUNCTIONAL_SPECIFICATIONS.md` update, acceptance criteria, resolved decisions, and any blocker.

## 2. Expert technical specification agent

Once functional requirements are clear, write a technical file for each subject. Cover frontend, backend, and API in every file; state `not affected` with a reason for a tier that needs no change.

Specify the frontend views/components, state and data flow, accessibility and functional UI states; backend responsibilities, validation, persistence and migrations if needed, collection/concurrency implications if relevant; and API requests, responses, errors, and compatibility. Reference requirement IDs and existing project architecture. Include the exact intended implementation scope, affected files or areas, verification approach, and dependencies on other subjects.

Choose straightforward designs with existing conventions. Avoid speculative abstractions, dependencies, unrelated cleanup, and clever code. Document a necessary refactoring separately with its reason, scope, risk, and whether it blocks the feature. The technical agent decides whether to perform it before the feature, defer it, or decline it, and records the decision and rationale. Do not silently include refactoring.

Handoff: technical files, requirement mapping, implementation scope, verification plan, API changes, and recorded refactoring decisions.

## 3. Define the canonical API contract

After technical specifications are ready, update the repository-root `API_SPECIFICATION.md` with the contract. This file is the canonical HTTP contract; technical documents link to it. Define methods and paths, request/response shapes, field types and units, validation, status codes and error bodies, asynchronous/loading semantics, and compatibility where relevant. Contract changes need no user confirmation: record the change and its rationale in the feature index and proceed. For changes spanning frontend and backend, finalize the contract before implementing either tier. Preserve existing contract behavior unless the functional specification requires a change. If implementation exposes a required contract change, return to this stage before implementing the change.

Handoff: canonical contract link and revision, recorded change rationale, and agreement between the contract and technical specifications.

## 4. Expert implementation agent

Implement only the ready specifications and the canonical contract. Read all four project skills listed in `AGENTS.md`; apply the frontend and Go backend guidance relevant to the change.

Use readable names, explicit control flow, focused functions, and the smallest sufficient design. Modify only files necessary for the feature and recorded refactoring decisions. Preserve unrelated work. Do not add functionality to fill specification gaps. Return functional gaps to the coordinator, who asks the user and stops. A newly discovered refactoring returns to the technical agent for a recorded decision before it is performed.

Run checks appropriate to the agreed verification plan. Report actual commands, outcomes, and limitations; never imply checks were executed when they were not. QA and corner-case testing are authorized verification work.

Handoff: requirement-to-change mapping, changed files, actual verification results, and known limitations.

## 5. Independent expert reviewer

Compare the complete change against functional specifications, technical specifications, the canonical API, project conventions, and specified scope. Check correctness, simplicity, readability, accessibility where relevant, and regression risks.

Write each finding with an ID, file/location or observable behavior, relevant requirement or missing requirement, evidence, impact, and suggested resolution. Explicitly report implemented behavior that specifications do not define, unmet requirements, API differences not recorded in the contract stage, and unrelated changes. Distinguish a demonstrated defect from a question or preference. A reviewer does not silently accept an assumption or alter the product specification.

Handoff: a persistent review report, including `no findings` when appropriate, and the exact change revision reviewed.

## 6. Independent expert decision agent

Create a Markdown decision record for every review finding. State whether it is critical and justify that classification using user impact, requirement/contract compliance, data integrity, security, accessibility, and regression likelihood where relevant. Record the evidence, resolution, rationale, owner, and validation or follow-up condition.

Use a disposition of `fix`, `defer`, or `reject`. For `fix`, record what must change; mark it resolved only after evidence of the correction and verification. For `defer`, explain why release is acceptable, remaining risk, and when or under what condition the work will occur. For `reject`, explain in detail why the finding does not require a change and cite supporting requirements or evidence. Neither noncritical status nor a preference alone is a rationale.

A critical finding blocks completion until corrected or resolved with evidence that the blocker no longer applies. For every critical finding that is fixed, the decision agent confirms the fix with evidence (the corrected code, the reviewer recheck, and a passing test). The developer adds a use case under `doc/use-cases/` that describes the scenario which triggered the finding and links it from `doc/use-cases/README.md`, and adds an automated regression test (Go or Playwright) that fails without the fix and passes with it. QA executes the use case in stage 7. The decision record links the use case and the test. The decision agent cannot override user-defined functionality, invent missing requirements, or waive project constraints. Functional ambiguities go to the user and stop the workflow; API changes return to the contract stage; refactoring returns to the technical agent.

Decision report (informational, never a gate): once all decisions are final, the coordinator publishes the decision agent's summary. It is posted as a comment on the originating issue when the run comes from one, and included in the final run report. For every finding the summary gives its ID and title, criticality, disposition, and what was done (the fix applied, or the reason for deferring or rejecting it). For each critical finding it also states that the fix is confirmed and links its use case and regression test. The workflow does not wait for a user reply to this report.

The implementation agent fixes accepted findings, and the reviewer checks those corrections. Preserve the original finding and append the outcome. New findings enter the same decision process. The decision record must explain both changes made and comments left unfixed.

Handoff: the review report and decision record agree, with no unexplained finding and no unresolved critical blocker; fixed critical findings have a linked use case and regression test; the decision summary is published.

## 7. Expert QA tester

Execute interface tests against the running application after review corrections and before stage 8: nothing is committed or pushed and no pull request is opened until QA passes. Cover acceptance criteria, every use case created for a fixed critical finding in stage 6, and relevant corner cases, including invalid input, empty results, missing items, loading/failure/retry behavior, repeated actions, keyboard navigation, and narrow screens when applicable. Include exploratory randomized action sequences; record the seed or complete ordered actions so failures can be reproduced. Use isolated test data for destructive actions where possible.

Keep a Markdown QA case catalog and execution report. For each case record its ID, requirement IDs, corner-case category, the actual application URL tested, environment, required data/setup, exact actions and input, expected and actual behavior, result, execution date, and supporting evidence. For external listing inputs, record the actual URL used and its purpose. Never invent URLs or report an unexecuted case as passed. Distinguish an application route from an external listing input. Record untested cases and concrete blockers explicitly. Avoid credentials and private data in evidence.

Investigate failures through the reviewer and decision agent; apply accepted fixes, have the reviewer and decision agent check them, and retest affected behavior before handing off to stage 8. If browser tooling or a running interface is unavailable, complete other useful verification and report interface QA as blocked, with the exact missing prerequisite. Do not substitute code inspection for executed interface tests.

Handoff: executed QA results, reproducible exploratory sequences, actual URL/corner-case catalog, and unresolved limitations.

## 8. Coordinator commits and pull requests

After review, review decisions, and QA pass for the exact diff, create focused commits containing the requested change and its workflow evidence. Confirm specification consistency, recorded contract decisions, justified outcomes for every review finding, and absence of critical blockers before committing. Any new implementation changes return to the applicable review and decision stages, and to QA.

Inspect the working tree and stage explicit paths or hunks; do not include unrelated user changes. Review the staged diff and run `git diff --cached --check`. Use a descriptive commit message explaining the concrete change. Do not amend existing commits.

Pull requests ([requirements](../specifications/workflow-pull-requests/functional.md)): in autonomous development, push the work branch and open pull requests once review decisions and QA pass: to `master` for a single PR, or to a feature branch when split, with the final feature→`master` PR created only after all parts merge (see Feature branch and stacking below). Open them ready for review, not as drafts, with descriptions written by the agent (purpose, requirement IDs, scope, verification, QA results, limitations, and for stacked PRs their position in the stack and base).

- Size: changed lines are added plus deleted lines (`git diff --numstat`), including specifications and documentation, excluding generated files such as `package-lock.json`. Target 450 changed lines per PR with a ±50 margin when splitting; 500 is the only enforced limit. Split any larger change. Smaller PRs are acceptable when the change, or a separate-kind PR (refactoring, dependencies, formatting, unrelated docs), is smaller; never combine unrelated work to reach the target. Measure each PR with `scripts/pr-size.sh <base>` before opening it or pushing more commits to it, and do not open or grow a PR it rejects, except the final feature-branch→`master` PR, which is exempt; the `ai-dev` CI run also fails when an open PR for the issue exceeds 500.
- Feature branch and stacking ([requirements](../specifications/workflow-feature-branch/functional.md)): when a change is split into several PRs, first create a feature branch from `master` (`ai-dev/issue-<n>-feature` in `ai-dev` runs, or `feature/<change>` otherwise) and push it before any part. Part PRs are stacked on it: PR 1 targets the feature branch, PR 2 targets PR 1's branch, and so on. After a part merges, retarget the next PR to the feature branch (`gh pr edit --base`) if GitHub did not. Stage 8 does not open the final PR: while no part has merged, the feature branch has no commits beyond `master` and GitHub refuses the PR. Once every part has merged into the feature branch, one final PR from the feature branch to `master` is created. It is opened ready, its description lists the part PRs in merge order and says to merge it last, and it carries `Closes #<n>` (part PRs say `Part of #<n>`). Only this final merge reaches `master`, so `ci-deploy` deploys once per feature. The final PR is exempt from the 500-line limit because it only aggregates parts that were already reviewed and measured. A change that fits in a single PR targets `master` directly, without a feature branch.
- Separate PRs: refactoring is always its own PR, placed before the feature when better done first, or after it when found during or after development. Dependency updates and formatting each get their own PR. Feature documentation goes with the feature; unrelated documentation gets its own PR.

If the run's tooling cannot create the required branches or PRs (for example, a CI workflow that permits only `git push origin HEAD` and opens a single PR itself), push what is permitted, record the planned split in `commit-step.md` (PR order, branch and base, changed-line counts, descriptions), and report it as a limitation. Never claim a PR was created unless the tooling confirms it.

Record staged scope, verification commands and results, commit messages, pull requests, and outcomes in `doc/changes/<change>/commit-step.md`, using [the commit template](templates/commit.md), and link it from the change index. Record the preparation and passed gates before committing. Refer to the resulting commit through its message and `git log` evidence; do not insert a commit's own hash into a file included in that commit. If committing or pushing fails, record the error and leave this stage blocked. The user may explicitly waive committing; record that request rather than claim a commit succeeded.

Handoff: successful focused commits, pushed ready pull requests or a recorded planned split with its tooling limitation, recorded scope and verification evidence, or an explicit user no-commit override.

## 9. Expert pull request reviewer

After stage 8 opens the pull requests, a separate expert agent ([role](../../.agents/roles/pr-reviewer.md), [requirements](../specifications/workflow-pr-reviewer/functional.md)) reviews each PR's diff in full detail as a Go backend and JavaScript/TypeScript frontend expert. It is independent of the developer, the stage 5 reviewer, the adjudicator, and the stage 10 triage agent.

It comments on everything that could be improved: possible bugs first, then simpler alternatives, unreadable or clever code, refactoring or unrelated changes that belong in a dedicated PR, missing tests, and small details. Comments are posted as one GitHub review per PR, inline on the diff lines, each labelled (`[bug]`, `[refactor-pr]`, `[simplify]`, `[readability]`, `[test]`, `[nit]`).

The PR reviewer only comments; it does not edit or push code. Its comments go to stage 10, which decides what blocks.

Possible bugs are submitted as **Request changes**. GitHub forbids requesting changes on a PR authored by the same account, so in that case the review is submitted as a comment whose body starts with `CHANGES REQUESTED`.

Handoff: one posted review per PR (URL, reviewed head SHA, and counts per label recorded in `doc/changes/<change>/pr-review.md`, committed to the last PR branch with `Refs: #<n>` and pushed). A commit that only adds or updates `pr-review.md`, `pr-triage.md`, or todo files needs no re-review.

## 10. PR comment triage, fix loop, and merge

A separate triage agent ([role](../../.agents/roles/pr-review-triage.md), [requirements](../specifications/workflow-pr-triage/functional.md)) decides, for every PR reviewer comment, whether it is **blocking** or **non-blocking**, and records the reason in `doc/changes/<change>/pr-triage.md`. It is independent of the developer and the PR reviewer. A comment is blocking when it describes a possible bug, a requirement or contract violation, a security or data risk, or refactoring or unrelated work mixed into a feature PR. Other comments are non-blocking. Its label is a hint and not the decision.

- **Non-blocking:** add a file to `doc/todo/` (`<YYYY-MM-DD>-<slug>.md`, in the format of the existing todo files, plus a date prefix: status, problem, suggested work, and a link to the PR comment) and commit it on the PR branch, or on a separate documentation PR if it would push the PR over 500 changed lines. Commits that only add todo files need no re-review. Then try to create a GitHub issue for it (`gh issue create`) and link the issue and the todo file to each other. If creating the issue fails, record the error in the todo file and `pr-triage.md`, tell the user (issue comment or final run report), and continue; this never stops the workflow.
- **Blocking:** return it to the developer agent, which fixes it on the same PR branch, runs the relevant checks, and pushes. The PR reviewer then reviews that PR again, and this triage runs on the new comments. Repeat until no blocking comment remains on any PR. A blocking comment that needs a functional decision goes to the user and stops the workflow, as in every stage. If the same blocking comment is still open after 5 fix rounds, report it to the user and stop rather than loop forever.
- **Merge:** when no blocking comment remains on any PR of the change, the agent merges them itself. The order is the stack order: each part into the feature branch with `gh pr merge <n> --merge --delete-branch`. A merge commit keeps the stacked history applicable, and deleting the merged part's branch makes GitHub retarget the next part to the feature branch. Before each part merge, the agent checks that the part targets the feature branch, and retargets it if it does not. Before merging the final PR, it checks that every part's commits are in the feature branch (`git merge-base --is-ancestor`). Then it merges the final feature-branch PR, or the single PR, into `master` with `gh pr merge <n> --squash --delete-branch`. If a merge fails (branch protection, conflicts, a required review), the agent records the error, informs the user, and stops without merging the rest. Merges made with the `ai-dev` run's `GITHUB_TOKEN` do not trigger other workflows, so after the merge into `master` the agent starts the deploy with `gh workflow run ci-deploy.yml --ref master`. It reports the merges, the deploy run, the todo files, and the issues created (or the failures) on the originating issue and in the final run report.

Handoff: `pr-triage.md` with every comment's decision; no blocking comment open; todo files and issues (or recorded failures) for non-blocking comments; PRs merged; deploy started.

## Completion

Mark the feature complete only when specifications and the contract are consistent with the final implementation, all review findings have justified recorded outcomes, no critical blocker remains, every fixed critical finding has a passing use case and regression test, the decision summary is published, required QA passes, stage 8 commits succeed (unless the user explicitly requests no commit), the single PR or every part PR exists (for a split change the final feature→`master` PR is created later, once all parts merge) or the planned split is recorded as a tooling limitation, every PR has a posted stage 9 review, stage 10 leaves no blocking comment, non-blocking comments are in `doc/todo/` (with issues or recorded failures), and the PRs are merged with the deploy started. Record deferred noncritical work and its rationale. Summarize delivered behavior, verification evidence, commit and pull request outcome, and material remaining limitations for the user.
