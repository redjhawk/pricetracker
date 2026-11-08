# Review decisions: GitHub release script

Reviewed specification revisions: [functional](../../specifications/github-release/functional.md), [technical](../../specifications/github-release/technical.md) (working tree, 2026-11-08)
Reviewed code revision: working-tree `scripts/create-github-release.sh` (55 lines, uncommitted) on `feature/2026-11-08-1821-github-release`
Reviewer: independent reviewer agent ([review](review.md))
Adjudicator: independent review adjudicator agent (did not write or review the code)

No finding is critical, so no `doc/use-cases/` use case or regression test is required. The script was inspected directly; reviewer severities were re-derived, not copied.

## Findings and decisions

### RV-01: release may target a different repository than `origin`

- Evidence: l.28 lists tags on `origin`, l.47 pushes the tag to `origin`, but l.51 `gh release create` has no `--repo`, so gh resolves the repository from `GH_REPO`, `gh repo set-default`, or remote heuristics (it prefers `upstream`/`github` remotes over `origin`). FR-GHREL-03 requires the tag on origin and FR-GHREL-04 a release for that tag.
- Impact and scenario: in a clone with `GH_REPO` set or an extra remote (fork workflow), the tag is pushed to origin but the release is attempted elsewhere: `--verify-tag` fails leaving a dangling tag on origin, or, if the other repository has the same tag name, a release is published on the wrong repository with the binary of a different commit. Today's clone has a single `origin` (github.com/redjhawk/pricetracker), so it is not triggered now.
- Criticality: non-critical, because the failure requires a non-default environment absent in the current setup, the common outcome is a clean error with recovery instructions, and the wrong-repo success path needs a coincident tag name on a repository the user can write to. It is a real defect (spec says origin), though.
- Disposition: fix.
- Reason: one-line, low-risk change that makes the script deterministic and consistent with FR-GHREL-03/04; cheaper than documenting an assumption users will not read.
- Specification decision: not applicable (spec already says origin).
- Resolution: pending. Developer: resolve the origin repository once in the prechecks (after l.28), e.g. `REPO="$(gh repo view "$(git remote get-url origin)" --json nameWithOwner -q .nameWithOwner)"` with a clear `fail` if it errors, and pass `--repo "$REPO"` to `gh release create`. Reviewer recheck required.
- Follow-up: none after fix.

### RV-02: remote tag match uses an unescaped regex

- Evidence: l.37 `grep -q "refs/tags/v$DATE-$N\$"`; dots in `$DATE` are wildcards, the match is not anchored at the start of the ref field.
- Impact and scenario: only an unusual tag such as `v2026x11x08-1` or `foo/refs/tags/v2026.11.08-1` would match falsely, causing a skipped number (the safe direction: never a reused tag). FR-GHREL-02 stays satisfied.
- Criticality: non-critical, because no realistic tag produces a wrong result and the failure direction cannot overwrite or reuse a tag.
- Disposition: fix.
- Reason: the exact match is equally short and removes reader doubt about correctness of the numbering, which is the core of FR-GHREL-02; changing it now, in the same new file before first commit, costs nothing extra and adds no scope.
- Specification decision: not applicable.
- Resolution: pending. Developer: use an exact field match, e.g. `awk -v ref="refs/tags/v$DATE-$N" '$2 == ref { found = 1 } END { exit !found }' <<<"$REMOTE_TAGS"`, or `cut -f2 <<<"$REMOTE_TAGS" | grep -qxF -- "refs/tags/v$DATE-$N"`. Reviewer recheck required.
- Follow-up: none after fix.

### RV-03: git may prompt for credentials instead of failing

- Evidence: l.28 `git ls-remote` and l.47 `git push` over HTTPS; without a credential helper git prompts on the terminal. The precheck does not prove push rights.
- Impact and scenario: an interactive or CI run without credentials hangs at a prompt (CI) or asks interactively, rather than failing clearly as FR-GHREL-07 expects. A read-only user passes the precheck, builds, tags locally, and fails at the push with a correct message and recovery hint, so no release is created from a failed state.
- Criticality: non-critical, because no wrong release or data loss is possible; the worst case is a hang or late failure with correct recovery guidance.
- Disposition: fix (only the prompt part).
- Reason: `export GIT_TERMINAL_PROMPT=0` near the top is one line and turns a hang into the clear failure FR-GHREL-07 asks for. Proving push rights before the build is rejected: it would need a GitHub API permission query, adds complexity, and the existing late failure already conforms to the spec and leaves only a local tag the message tells the user how to remove.
- Specification decision: not applicable.
- Resolution: pending. Developer: add `export GIT_TERMINAL_PROMPT=0` after `set -euo pipefail`. Reviewer recheck required.
- Follow-up: none.

### RV-04: HEAD is released even if not on `master` or not pushed to a branch

- Evidence: l.46 tags `HEAD` unconditionally; pushing the tag uploads the commit.
- Impact and scenario: a release can be cut from a feature branch or an unpushed commit. FR-GHREL-03 explicitly says "the current commit (HEAD)" and no requirement restricts the branch; the user's confirmed answers did not ask for one.
- Criticality: non-critical, because it is specified behavior, not a defect.
- Disposition: reject.
- Reason: adding a branch restriction would be implementing unspecified behavior and would invent a functional answer, which AGENTS.md forbids. The behavior conforms; tags pushing their commit is standard git. If the user later wants a `master`-only rule, that is a new functional requirement. Optional DEPLOYMENT.md wording is not required since the section already says it releases the current commit.
- Specification decision: none needed; FR-GHREL-03 covers it.
- Resolution: rejected; no code change.
- Follow-up: none.

### RV-05: missing git identity aborts with git's own message

- Evidence: l.46 `git tag -a` fails without `user.name`/`user.email`; `set -e` aborts before any push or release.
- Impact and scenario: user sees git's standard "Please tell me who you are" message, which is clear and actionable; non-zero exit; no tag, no release. FR-GHREL-07 satisfied.
- Criticality: non-critical, because the outcome already meets the requirement.
- Disposition: reject.
- Reason: git's own message is clearer than a wrapper could make it; wrapping adds code for no behavioral gain. The only cost of not fixing is that this failure happens after the build, a few minutes of wasted time on a misconfigured machine.
- Specification decision: not applicable.
- Resolution: rejected; no code change.
- Follow-up: none.

## QA findings ([QA report](qa.md))

### QA-O1: build failure prints no own `Error:` line

- Evidence: QA-11, the `scripts/build-release.sh amd64` call aborts through `set -e`; the only output is the build tool's stderr. Exit 1, no tag, no release.
- Impact and scenario: a maintainer sees npm or go output but no line saying that the release build step failed and that nothing was tagged. Every other failure path prints an `Error:` line naming the step, so this one is inconsistent.
- Criticality: non-critical, because the exit status is correct and no tag or release is created, so the state stays safe. Only the clarity part of FR-GHREL-07 is weak.
- Disposition: fix.
- Reason: FR-GHREL-07 asks for a clear error message. Every other step meets that with the script's own line, and the fix is the same `if ! ...; then fail ...` pattern already used at l.28 and l.47.
- Resolution: pending. Developer: wrap the build call: `if ! "$PROJECT_DIR/scripts/build-release.sh" amd64; then fail "building the amd64 release failed; no tag or release was created."; fi`. Reviewer recheck; QA repeats QA-11.
- Follow-up: none.

### QA-14 wording note: "retry" hint is misleading

- Evidence: the release-failure message says "retry". Rerunning the script makes a new tag (`-N+1`) and leaves the first one on origin without a release, so the only correct retry is running `gh release create` by hand.
- Impact and scenario: a maintainer who follows the hint by rerunning the script ends up with an orphan tag on origin and a skipped version number.
- Criticality: non-critical, because nothing is corrupted and the cleanup command in the same message still works. It is a wording problem on a rare path.
- Disposition: fix.
- Reason: FR-GHREL-07 asks for a clear message, and a recovery hint that leads to the wrong action is not clear. Changing one string costs nothing.
- Resolution: pending. Developer: replace "retry" with the exact manual command, e.g. `create it with 'gh release create $TAG --repo $REPO --verify-tag --title $TAG --generate-notes <binary>'`, or remove the tag with the existing command. If `DEPLOYMENT.md` mentions retrying, update it to match. Reviewer recheck; QA repeats QA-14.
- Follow-up: none.

### QA-O2: tag `v2026.11.08-1` already on the real origin (informational)

- Evidence: lightweight tag `v2026.11.08-1` at `15c9535`, local and on github.com origin.
- Disposition: informational, no action. The coordinator made this release by hand at the user's request; QA did not create it. The script handles it correctly: QA-01 picked `-2`, which is what FR-GHREL-02 requires. It is lightweight, not annotated like the script's tags, but no requirement depends on the tag type.
- Resolution: recorded only.

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| QA-O1 no own error line on build failure | non-critical | fix | Developer to wrap the build call with `fail` naming the step |
| QA-14 "retry" wording | non-critical | fix | Developer to replace "retry" with the manual `gh release create` command |
| QA-O2 manual release v2026.11.08-1 | informational | none | Intentional manual release at the user's request; numbering skips it correctly |

QA readiness update: QA passed with no blocking failures. The two QA fixes above, together with RV-01..03, need a reviewer recheck, and QA must repeat QA-11 and QA-14 before commit.

## Review recheck findings ([review](review.md))

### RC-02: project path printed unquoted in the manual command

- Evidence: in the release-failure message, `$PROJECT_DIR` appears inside the copy-paste command without quotes.
- Impact and scenario: this only matters after a release-creation failure, and only for a clone whose path contains spaces or shell metacharacters. The pasted `cp` then fails with a visible error and does no damage. The real clone path `/home/redjhawk/src/pricefollower` is safe.
- Criticality: non-critical, because nothing is created or deleted wrongly; at worst a hint fails visibly on an uncommon path.
- Disposition: defer.
- Reason: the coordinator asked to defer non-essential items. The script itself is correct, only the advisory text is affected, the condition is absent in the maintainer's environment, and a user can quote the path by hand. Fixing it now would add another fix, recheck and QA cycle for no current benefit. Remaining risk: a failed paste on an unusual path.
- Resolution: deferred.
- Follow-up: [doc/todo/2026-11-08-github-release-recovery-hints.md](../../todo/2026-11-08-github-release-recovery-hints.md); maintainer, unscheduled.

### RC-03: partial gh failure after the release was created

- Evidence: if `gh release create` creates the release and then fails to upload the asset, the printed manual `gh release create` command fails with "release already exists".
- Impact and scenario: a rare network failure partway through. The release exists without its binary; the maintainer sees gh's own error and can run `gh release upload` or delete the release. No wrong binary is published.
- Criticality: non-critical, because the failure is visible (non-zero exit with an `Error:` line), nothing is corrupted, and the spec does not cover partial gh failures.
- Disposition: defer.
- Reason: handling it properly means detecting whether the release exists after a failure, which adds logic and API calls for a rare case. That fails the "simple, narrow change" rule for this change. Remaining risk: the hint is wrong in that case, but gh's own error makes the state clear.
- Resolution: deferred.
- Follow-up: same todo file as RC-02.

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| RC-02 unquoted path in manual command | non-critical | defer | Advisory text only, safe on the real path; tracked in doc/todo |
| RC-03 partial gh failure after release created | non-critical | defer | Rare, visible failure outside the spec; fixing it needs extra detection logic; tracked in doc/todo |

## Decision summary (published, informational)

| Finding | Criticality | Disposition | What was done |
|---|---|---|---|
| RV-01 release repo may differ from origin | non-critical | fix | Developer to pass `--repo` resolved from the origin URL to `gh release create` |
| RV-02 regex tag match | non-critical | fix | Developer to use an exact field match for remote tags |
| RV-03 credential prompt | non-critical | fix (partial) | Developer to set `GIT_TERMINAL_PROMPT=0`; early push-rights check rejected as unneeded complexity, late failure already conforms |
| RV-04 HEAD not on master | non-critical | reject | Specified behavior (FR-GHREL-03); a branch rule would be an invented functional requirement |
| RV-05 missing git identity | non-critical | reject | Git's own error already meets FR-GHREL-07; no tag or release is created |

Published as an issue comment: no originating issue. Included in the final run report: yes.

## Release readiness

No critical or blocking findings. Three non-critical fixes (RV-01..03) are accepted and pending developer implementation, reviewer recheck, and QA. No functional questions open.
