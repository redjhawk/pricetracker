# Specification-led development workflow

This workflow applies to new functionality and behavior changes. The coordinator maintains the handoffs and uses separate expert agents for functional specification, technical specification, implementation, independent review, independent review decisions, and QA. A reviewer or decision agent must not review or adjudicate its own implementation. The decision agent is distinct from the reviewer. Available concurrency limits do not remove this separation: run the roles sequentially when needed.

## Autonomy and the functional gate

Agents develop autonomously. The user does not approve API contracts, technical designs, refactoring, implementation, review decisions, or commits; the agents decide these, record the rationale, and proceed.

The user owns functionality only. The coordinator and functional specification agent check at the start that the request fully defines the functional behavior. If any functional behavior, product choice, or acceptance criterion is missing, ambiguous, or contradictory, ask the user focused questions to complete the functionality and stop: do not start technical specification or implementation. Never assume or invent functional answers. If a later stage discovers a functional gap, stop at that point, record the blocker, and ask the user in the same way.

For workflow/documentation maintenance, apply the relevant stages to the documentation change; do not invent product requirements or API changes. Existing requirements and contracts may be reused with explicit references.

## Documents and traceability

Keep a separate functional Markdown file and a separate technical Markdown file for each subject. For example, the tracked-items list and item details are separate subjects. Use stable, descriptive subject slugs: `doc/specifications/<subject>/functional.md` and `doc/specifications/<subject>/technical.md`. Keep change reports under `doc/changes/<change>/`, including `review.md`, `decisions.md`, and `qa.md`. Use the starting templates in [`templates/functional.md`](templates/functional.md), [`templates/technical.md`](templates/technical.md), [`templates/review-decisions.md`](templates/review-decisions.md), and [`templates/qa.md`](templates/qa.md). Link existing `doc/FUNCTIONAL_SPECIFICATIONS.md` and `doc/use-cases/` where applicable rather than silently replacing their meaning.

The coordinator records the paths of all artifacts in the feature's index, `doc/changes/<change>/index.md`. The index contains scope, current stage, agent handoffs, user decisions, unresolved questions, and links to the subject specifications, review findings, decision record, and QA report. Record functional decisions only from user messages or existing specifications.

Assign stable requirement IDs such as `FR-ITEM-DETAILS-001`, technical IDs such as `TS-ITEM-DETAILS-001`, review IDs such as `REV-001`, and QA case IDs such as `QA-001`. Link technical design, implementation, review findings, and QA results to their source requirements. Preserve IDs when documents change.

Artifact states are `draft`, `needs-clarification`, `ready`, and `superseded`. A ready artifact has no unresolved question that affects its handoff. A feature progresses through `functional-specification`, `technical-specification`, `api-contract`, `implementation`, `review`, `review-decisions`, `commit`, `pull-request`, `qa`, and `complete`. Record a blocked stage and its concrete blocker instead of advancing it.

## 1. Expert functional specification agent

Start with an explicit list of requested functional subjects and behaviors. Read the user request and existing specifications. Write one functional file per subject before technical design begins.

Each file states purpose, actors, scope and exclusions, prerequisites, user-visible behavior, success and failure flows, input rules, empty/loading/error states where relevant, and objectively checkable acceptance criteria. Give every requirement an ID. Describe what the user observes without deciding implementation details.

Identify contradictions, undefined behavior, and missing product decisions. Ask the user focused questions about any functional ambiguity and stop the workflow until they answer. Do not convert guesses into requirements or expand the feature. An ambiguity affecting acceptance must be resolved by the user or a referenced existing decision before the next stage. Explicit requirements already supplied by the user need no repeated confirmation.

Handoff: subject files and their requirement IDs, acceptance criteria, resolved decisions, and any blocker.

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

A critical finding blocks completion until corrected or resolved with evidence that the blocker no longer applies. The decision agent cannot override user-defined functionality, invent missing requirements, or waive project constraints. Functional ambiguities go to the user and stop the workflow; API changes return to the contract stage; refactoring returns to the technical agent.

The implementation agent fixes accepted findings, and the reviewer checks those corrections. Preserve the original finding and append the outcome. New findings enter the same decision process. The decision record must explain both changes made and comments left unfixed.

Handoff: the review report and decision record agree, with no unexplained finding and no unresolved critical blocker.

## 7. Expert QA tester

Execute interface tests against the running application after review corrections. QA runs after stage 8 has pushed and opened the pull requests; push accepted QA fixes to the same PR branches. Cover acceptance criteria and relevant corner cases, including invalid input, empty results, missing items, loading/failure/retry behavior, repeated actions, keyboard navigation, and narrow screens when applicable. Include exploratory randomized action sequences; record the seed or complete ordered actions so failures can be reproduced. Use isolated test data for destructive actions where possible.

Keep a Markdown QA case catalog and execution report. For each case record its ID, requirement IDs, corner-case category, the actual application URL tested, environment, required data/setup, exact actions and input, expected and actual behavior, result, execution date, and supporting evidence. For external listing inputs, record the actual URL used and its purpose. Never invent URLs or report an unexecuted case as passed. Distinguish an application route from an external listing input. Record untested cases and concrete blockers explicitly. Avoid credentials and private data in evidence.

Investigate failures through the reviewer and decision agent; apply accepted fixes and retest affected behavior. If browser tooling or a running interface is unavailable, complete other useful verification and report interface QA as blocked, with the exact missing prerequisite. Do not substitute code inspection for executed interface tests.

Handoff: executed QA results, reproducible exploratory sequences, actual URL/corner-case catalog, and unresolved limitations.

## 8. Coordinator commits and pull requests

After review and review decisions pass for the exact diff, create focused commits containing the requested change and its workflow evidence. Confirm specification consistency, recorded contract decisions, justified outcomes for every review finding, and absence of critical blockers before committing. Any new implementation changes return to the applicable review and decision stages, and to QA if it already ran.

Inspect the working tree and stage explicit paths or hunks; do not include unrelated user changes. Review the staged diff and run `git diff --cached --check`. Use a descriptive commit message explaining the concrete change. Do not amend existing commits.

Pull requests ([requirements](../specifications/workflow-pull-requests/functional.md)): in autonomous development, push the work branch and open pull requests to `master` once review decisions pass; do not wait for QA. Open them ready for review, not as drafts, with descriptions written by the agent (purpose, requirement IDs, scope, verification, limitations, and for stacked PRs their position in the stack and base). QA then runs; push its accepted fixes to the same branches.

- Size: changed lines are added plus deleted lines (`git diff --numstat`), including specifications and documentation, excluding generated files such as `package-lock.json`. Target 450 changed lines per PR with a ±50 margin when splitting; 500 is the only enforced limit. Split any larger change. Smaller PRs are acceptable when the change, or a separate-kind PR (refactoring, dependencies, formatting, unrelated docs), is smaller; never combine unrelated work to reach the target.
- Stacking: split PRs are stacked; the first targets `master`, PR 2 targets PR 1's branch, and so on.
- Separate PRs: refactoring is always its own PR, placed before the feature when better done first, or after it when found during or after development. Dependency updates and formatting each get their own PR. Feature documentation goes with the feature; unrelated documentation gets its own PR.

If the run's tooling cannot create the required branches or PRs (for example, a CI workflow that permits only `git push origin HEAD` and opens a single PR itself), push what is permitted, record the planned split in `commit-step.md` (PR order, branch and base, changed-line counts, descriptions), and report it as a limitation. Never claim a PR was created unless the tooling confirms it.

Record staged scope, verification commands and results, commit messages, pull requests, and outcomes in `doc/changes/<change>/commit-step.md`, using [the commit template](templates/commit.md), and link it from the change index. Record the preparation and passed gates before committing. Refer to the resulting commit through its message and `git log` evidence; do not insert a commit's own hash into a file included in that commit. If committing or pushing fails, record the error and leave this stage blocked. The user may explicitly waive committing; record that request rather than claim a commit succeeded.

Handoff: successful focused commits, pushed ready pull requests or a recorded planned split with its tooling limitation, recorded scope and verification evidence, or an explicit user no-commit override.

## Completion

Mark the feature complete only when specifications and the contract are consistent with the final implementation, all review findings have justified recorded outcomes, no critical blocker remains, required QA passes, stage 8 commits succeed (unless the user explicitly requests no commit), and pull requests exist or their planned split is recorded as a tooling limitation. Record deferred noncritical work and its rationale. Summarize delivered behavior, verification evidence, commit and pull request outcome, and material remaining limitations for the user.
