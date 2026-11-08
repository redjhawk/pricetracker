# Review: GitHub release script

Reviewer: independent reviewer agent (did not write the code). Date: 2026-11-08.
Scope: uncommitted `scripts/create-github-release.sh` (new), `DEPLOYMENT.md` section "Publish a GitHub release (amd64)", this change folder. Specs: [functional](../../specifications/github-release/functional.md), [technical](../../specifications/github-release/technical.md).
Known context: `README.md` and `scripts/build-release.sh` carry uncommitted user changes (amd64 target) that the script depends on; not reviewed as part of this change.
Checks: read-through, `bash -n` evidence from implementation note; `shellcheck` not installed. Nothing was pushed or released.

Overall: implementation matches FR-GHREL-01..07 and the technical mapping. No critical defects found. Severities below are provisional for the adjudicator.

## Findings

| ID | Severity | Location | Requirement | Finding | Suggested action |
| --- | --- | --- | --- | --- | --- |
| RV-01 | Medium | `scripts/create-github-release.sh:51` | FR-GHREL-03/04 | Tag is pushed to `origin` (l.47) and its existence checked against `origin` (l.28), but `gh release create` targets gh's resolved repo (`GH_REPO`, `gh repo set-default`, or an `upstream` remote in a fork clone), not necessarily `origin`. In a fork/multi-remote clone the release is created on another repo, where `--verify-tag` fails after the tag was already pushed, or worse succeeds if that repo has the same tag. Current single-remote clone (`origin` = github.com/redjhawk/pricetracker) is unaffected. | Pass `--repo` derived from the origin URL (e.g. `gh repo view "$(git remote get-url origin)" --json nameWithOwner`) or document the single-remote assumption. |
| RV-02 | Low | `scripts/create-github-release.sh:37` | FR-GHREL-02 | `grep -q "refs/tags/v$DATE-$N\$"` treats the dots in the date as regex wildcards and matches any line suffix. Anchoring to `$` prevents `-1` matching `-11`, so no realistic wrong result; only a bizarre tag such as `v2026x11x08-1` would cause a skipped number (harmless direction). | Optional: `grep -qF -- "refs/tags/v$DATE-$N"` combined with field-exact match (e.g. `awk '$2==ref'`) for clarity. |
| RV-03 | Low | `scripts/create-github-release.sh:28,47` | FR-GHREL-07 | `git ls-remote` / `git push` over HTTPS without a credential helper may prompt interactively instead of failing; the precheck at l.28 does not prove push rights (read-only access passes), so a permissions problem only appears at l.47 after build and local tag. Error message there is correct (names tag, how to delete), so behavior stays within spec. | Optional: set `GIT_TERMINAL_PROMPT=0`; accept as-is otherwise. |
| RV-04 | Info (question) | `scripts/create-github-release.sh:46` | unspecified | HEAD is tagged and released even if the commit is not on `master` or not pushed to any branch (pushing the tag uploads the commit). Spec says "current commit (HEAD)" and does not restrict the branch, so this conforms; flagged only so the adjudicator can confirm it is intended. | None required; optionally note in DEPLOYMENT.md. |
| RV-05 | Info | `scripts/create-github-release.sh:46` | FR-GHREL-07 | `git tag -a` needs a configured `user.name`/`user.email`; if missing, `set -e` aborts with git's own message only (no tag, no release, so spec still holds). | None required. |

## Checked and found correct

- Precheck order matches technical spec: tools, clean tree, `gh auth status`, origin reachable, then build (FR-GHREL-01, 07). `release/` and `web/static/dist/` are git-ignored, so the build does not dirty the tree.
- `if ! VAR="$(cmd)"` correctly captures exit status under `set -e` (l.28, l.51); build failure aborts via `set -e` before any tag (l.32).
- Tag numbering: local `rev-parse --verify` plus remote list from `--refs`; anchored suffix prevents `-1`/`-11` confusion (FR-GHREL-02).
- Annotated tag on HEAD, pushed by name; `--verify-tag`, `--title "$TAG"`, `--generate-notes`, single asset `pricefollower-linux-amd64` (FR-GHREL-03..05). Published, not draft/prerelease, as recorded in the functional interpretation.
- EXIT trap removes the temp dir on every exit path after creation, including `fail`; trap is set immediately after `mktemp`.
- Final output prints tag and the URL from `gh release create` stdout (FR-GHREL-06); errors go to stderr with non-zero exit.
- `DEPLOYMENT.md` section accurately describes behavior; no behavior beyond the specs found. `install-pricefollower.sh` is not attached.

## Recheck (2026-11-08)

Re-reviewed only the RV-01..03 fixes in `scripts/create-github-release.sh` against `decisions.md` and the "Review fixes" section of `implementation.md`. Read-through plus the developer's recorded scratch-clone evidence; nothing pushed or released.

| ID | Status | Evidence |
| --- | --- | --- |
| RV-01 | Resolved | l.29 resolves `nameWithOwner` from the origin URL after `gh auth status`; l.56 passes `--repo "$REPO"`. The error at l.30 comes before the build and before any tag. The release repo now matches the repo the tag is pushed to. |
| RV-02 | Resolved | l.33 `cut -f2` keeps only ref names; l.42 `grep -qxF` gives an exact, literal whole-line match. `pipefail` still propagates an `ls-remote` failure through the pipe into the l.33 error. |
| RV-03 | Resolved | l.6 `export GIT_TERMINAL_PROMPT=0` applies to `ls-remote` and `push`; a missing credential now fails instead of prompting, and the existing messages cover it. |

New observations:

- RC-01, Info, l.29: if `origin` does not exist, `git remote get-url origin` fails inside the nested substitution and `gh repo view ""` runs with an empty argument. Depending on gh, this either errors (the clear l.30 message) or falls back to the default repo. Either way the l.33 `ls-remote origin` check fails before the build and before any tag, so FR-GHREL-07 still holds. No action required.
- No regressions found.

### Recheck of QA fixes QA-O1 and QA-14 (2026-11-08)

Read-through of `scripts/create-github-release.sh` l.37-39 and l.58-59 and the `DEPLOYMENT.md` sentence, plus the developer's recorded stub evidence. Nothing was pushed or released.

| ID | Status | Evidence |
| --- | --- | --- |
| QA-O1 | Resolved | l.37-39 wrap the build in `if ! ...; then fail`. Inside an `if` condition `set -e` is suspended, so a non-zero build exit reaches `fail`. The message names the step and confirms that nothing was tagged. This runs before the tag and before `mktemp`. |
| QA-14 | Resolved, with one Low note | l.59 no longer says "retry" and warns against rerunning the script. The printed command uses an absolute source path, so it works from any directory. The copy restores the asset name `pricefollower-linux-amd64`, which is needed because the EXIT trap deletes the temporary folder. `TAG` and `REPO` contain no shell metacharacters (`vYYYY.MM.DD-N`, `owner/repo`). The tag-removal command is unchanged and correct. `DEPLOYMENT.md` now matches the message. |

New observations:

- RC-02, Low, l.59: `$PROJECT_DIR` is printed without quotes inside the copy-paste command. If the clone path contains spaces or shell metacharacters, the pasted `cp` splits the path and fails. The current path `/home/redjhawk/src/pricefollower` is safe. Suggested fix: print the path in double quotes inside the message, `cp \"$PROJECT_DIR/release/pricefollower\" ...`, or use `printf '%q'`. This does not block.
- RC-03, Info, l.58-59: if `gh release create` creates the release and then fails while uploading the asset, the printed manual command fails with "release already exists". The right recovery then is `gh release upload` or deleting the release. This is a rare partial-failure path that the spec does not cover. No action required.
- No regressions found.
