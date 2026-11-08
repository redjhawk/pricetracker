# QA execution: GitHub release script

Status: executed (pass, one low-severity observation)
Tested code/specification revisions: base `15c9535`; working-tree `scripts/create-github-release.sh` and `scripts/build-release.sh` (uncommitted amd64 support) copied into the scratch clone and committed there as `87d175a`; [functional](../../specifications/github-release/functional.md), [technical](../../specifications/github-release/technical.md), [decisions](decisions.md).
Environment: Debian 13 amd64, go1.25.0, node v20.19.2, real `npm run build` and `go build`. Scratch dir `/tmp/claude-1000/-home-redjhawk-src-pricefollower/d6b83b15-4b76-4f32-b7de-a98b7348d8b9/scratchpad/qa`: `origin.git` (local bare clone of the repo, used as `origin`), `work` (clone under test, `node_modules` symlinked to the repo's and excluded via `.git/info/exclude`). Only `gh` is stubbed (`bin/gh`: logs arguments, prints `qa-owner/pricefollower` for `repo view`, copies the asset and prints a fake github.com URL for `release create`; `GH_FAIL=auth|release` forces failures).
Executed at: 2026-11-08 18:28-18:45 UTC
Tester: independent QA agent

No interface URL applies: the change is a maintainer CLI script with no web/API behavior. No listing URLs are involved.

## Exploratory session

Manual sequence, no seed, in the order of the table below (each run on the same clone, so tag state accumulates). Commands run from `work` with `QA_DIR=<scratch>/qa PATH=<scratch>/qa/bin:$PATH ./scripts/create-github-release.sh`.

## Executed cases

| Case | Requirement / corner case | Preconditions and exact actions | Expected | Observed | Result |
| --- | --- | --- | --- | --- | --- |
| QA-01 | FR-GHREL-01..06 | Clean clone; origin already has `v2026.11.08-1` (inherited from the real repo, see QA-O2). Full run. | Real build, tag `-2` at HEAD pushed, one asset `pricefollower-linux-amd64`, tag + URL printed | exit 0; `Released v2026.11.08-2: https://github.com/qa-owner/pricefollower/releases/tag/v2026.11.08-2`; `file` on the asset: `ELF 64-bit LSB executable, x86-64, statically linked, stripped`; bare origin `refs/tags/v2026.11.08-2^{}` = HEAD `87d175a`; gh called with `release create v2026.11.08-2 --repo qa-owner/pricefollower --verify-tag --title v2026.11.08-2 --generate-notes /tmp/tmp.zXaBD6GZSb/pricefollower-linux-amd64` (single asset); temp dir removed afterwards; tree still clean, `web/static/dist` removed | Pass |
| QA-02 | FR-GHREL-02 remote-only tag | `v2026.11.08-3` created and pushed from a second clone (absent locally). Run. | Skip `-3`, pick `-4` | exit 0, `v2026.11.08-4` | Pass |
| QA-03 | FR-GHREL-02 local-only tag | `git tag v2026.11.08-5` (not pushed). Run. | Pick `-6` | exit 0, `v2026.11.08-6` | Pass |
| QA-04 | Several releases same day | Run again. | `-7` | exit 0, `v2026.11.08-7` | Pass |
| QA-05 | Dirty tree (modified) | `echo x >> README.md`. Run. | Refuse before build | exit 1, `Error: the working tree has uncommitted or untracked changes...`; no tag, no gh release call | Pass |
| QA-06 | Untracked file | `touch qa.tmp`. Run. | Refuse | same as QA-05 | Pass |
| QA-07 | Staged change | modify + `git add`. Run. | Refuse | same as QA-05 | Pass |
| QA-08 | Missing tool (go, npm, gh, git, each) | PATH reduced to a dir of symlinks omitting one tool; run via `/usr/bin/bash`. | Error naming the tool | exit 1, `Error: <tool> is required to create a GitHub release.` for each of the four; no tag | Pass |
| QA-09 | gh unauthenticated | `GH_FAIL=auth`. Run. | Error before tag | exit 1, `Error: the GitHub CLI is not authenticated; run 'gh auth login' first.`; no build, no tag | Pass |
| QA-10 | Origin unreachable | `git remote set-url origin <scratch>/missing.git`. Run. | Error before tag | exit 1, git `fatal` lines then `Error: cannot reach the origin remote.`; no tag | Pass |
| QA-11 | Build failure | `npm` stub printing `npm stub failure` and exiting 1 first in PATH. Run. | Non-zero, no tag/release | exit 1, only output `npm stub failure` (from npm itself); no tag, no release call | Pass (see QA-O1) |
| QA-12 | Tag push failure | `pre-receive` hook in origin.git rejecting all pushes. Run. | Error naming step and tag | exit 1, `! [remote rejected] v2026.11.08-8`, `Error: pushing tag v2026.11.08-8 to origin failed; the tag exists locally (remove it with 'git tag -d v2026.11.08-8').`; no release call | Pass |
| QA-13 | Rerun after push failure without cleanup | Hook removed, local `-8` left. Run. | Next free N | exit 0, `v2026.11.08-9` | Pass |
| QA-14 | Release creation failure | `GH_FAIL=release`. Run. | Error naming tag; tag on origin | exit 1, `HTTP 422`, `Error: creating the GitHub release for tag v2026.11.08-10 failed; the tag is on origin (retry, or remove it with ...)`; `-10` present on origin | Pass |
| QA-15 | Suggested cleanup command | `git push origin --delete v2026.11.08-10 && git tag -d v2026.11.08-10` | Tag gone both sides | deleted; `git ls-remote` returns nothing | Pass |
| QA-16 | HEAD not on any origin branch | Empty local commit, not pushed. Run. | Tag at HEAD | exit 0, `v2026.11.08-10` re-used after cleanup; origin `^{}` = new HEAD `2da69d4` (commit objects pushed with the tag) | Pass (spec says HEAD) |

## Observations

- QA-O1 (low, non-blocking): on build failure the script adds no `Error:` line of its own; the only message is the build tool's stderr. Exit status is correct and nothing is tagged. FR-GHREL-07 asks for a clear message; real npm/go failures print their own diagnostics, so this is cosmetic. For adjudication.
- QA-O2 (information): the real repository already has a lightweight tag `v2026.11.08-1` at `15c9535`, both locally and on the real GitHub origin (`git ls-remote --tags origin 'v2026*'`, read-only). It was not created by QA. A real run today would therefore produce `v2026.11.08-2`.
- QA-14 message says "retry": rerunning the script creates a new tag (`-N+1`), so "retry" means re-running `gh release create` manually. Wording only.

## Coverage and limitations

- Not executed (not authorized): pushing any tag to the real GitHub origin and creating a real GitHub release. FR-GHREL-04 (generated notes visible), FR-GHREL-05 as seen on GitHub, real `gh auth status`/`gh repo view` against github.com and real `--verify-tag` behavior are verified only through the stub's recorded arguments.
- Date rollover across UTC midnight and concurrent runs from two clones were not exercised.
- The script refuses dirty trees, so the uncommitted working-tree scripts were committed inside the scratch clone only.

## Handoff and cleanup

QA-O1 goes to adjudication as a low-severity observation; no blocking failure. All test data lives in the scratch directory; the real repository and its origin were not modified.

## Retest after fixes (2026-11-08, UTC evening)

I copied the current working-tree `scripts/create-github-release.sh` into the scratch clone and committed it there as `d0e6118`. The setup and gh stub are unchanged from the first round.

| Case | Action | Observed | Result |
| --- | --- | --- | --- |
| QA-11b (QA-O1 retest) | Put the failing `npm` stub first in PATH, then run the script. | exit 1. Output: `npm stub failure`, then `Error: building the amd64 release failed; no tag or release was created.` No new tag. | Pass, QA-O1 fixed |
| QA-14b (wording retest) | Run with `GH_FAIL=release`. | exit 1. Tag `v2026.11.08-11` was pushed to origin. The error says not to rerun the script and prints a manual command: `cp <clone>/release/pricefollower /tmp/pricefollower-linux-amd64 && gh release create v2026.11.08-11 --repo qa-owner/pricefollower --verify-tag --title v2026.11.08-11 --generate-notes /tmp/pricefollower-linux-amd64`. It also prints the tag-removal alternative. | Pass |
| QA-14c | Run the printed manual command exactly as shown, against the gh stub. | exit 0. Printed the release URL. The stub received one asset, `pricefollower-linux-amd64`, and `file` reports it as an x86-64 ELF. I removed `/tmp/pricefollower-linux-amd64` afterwards. | Pass |

Note: the manual command only works while `release/pricefollower` from that run still exists. Running a different build first would overwrite that file. This is a minor caveat and does not block.
