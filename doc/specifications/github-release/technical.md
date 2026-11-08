# Technical specification: GitHub release

Status: ready
Functional specification: [functional.md](functional.md) (2026-11-08)

## Requirement mapping

| Functional ID | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- |
| FR-GHREL-01 | Call `scripts/build-release.sh amd64`; it writes `release/pricefollower`. | `scripts/create-github-release.sh` (new) | `release/pricefollower` exists; `file` reports x86-64 ELF. |
| FR-GHREL-02 | `date -u +%Y.%m.%d`; `git fetch --tags origin` (or `git ls-remote --tags origin`) plus `git tag -l`; loop N=1.. until `v$DATE-$N` is in neither set. | same | Create local or remote-only tag `v<today>-1` in a test clone; script picks `-2`. |
| FR-GHREL-03 | `git tag -a "$TAG" -m "$TAG" HEAD` then `git push origin "$TAG"`. | same | `git ls-remote --tags origin "$TAG"` matches `git rev-parse HEAD`. |
| FR-GHREL-04 | `gh release create "$TAG" --verify-tag --title "$TAG" --generate-notes <asset>`. | same | `gh release view "$TAG"`. |
| FR-GHREL-05 | Copy binary to a temporary dir as `pricefollower-linux-amd64` and pass only that file. | same | `gh release view "$TAG" --json assets` lists one asset with that name. |
| FR-GHREL-06 | Print tag and the URL printed by `gh release create`. | same | Output inspection. |
| FR-GHREL-07 | `set -euo pipefail`; prechecks in order: tools (`go npm gh git`), `git status --porcelain` empty, `gh auth status`, origin reachable; then build, tag, push, release. Errors to stderr, exit non-zero; on post-tag failure name the tag. | same | Run with a dirty tree, a missing tool on PATH, and unauthenticated gh: each exits non-zero before tagging. |

## Frontend

No change: the script is an operations tool and does not touch the web interface.

## Backend

No change: no Go code, persistence, or collection changes. The existing build script is reused unchanged.

## API

No contract change; `API_SPECIFICATION.md` is unaffected.

## Scope and refactoring

- New file `scripts/create-github-release.sh` (executable bash), plus a short usage mention in `README.md` if the developer judges it needed. Note `README.md` and `scripts/build-release.sh` currently carry uncommitted user changes; preserve them and do not stage them with this change.
- Run from the repository root resolved from the script location, like `build-release.sh`.
- The clean-tree check runs before the build; `release/` is git-ignored (`.gitignore`), so the build does not dirty the tree.
- No refactoring.
- Risk: the tag is pushed before the release; if `gh release create` fails the tag remains on origin. Accepted; the error names the tag for manual removal or retry.

## Verification and unresolved questions

- `bash -n` and `shellcheck` (if available) on the script.
- QA without publishing: precondition failures (dirty tree, missing tool, gh unauthenticated) and tag-number computation against local and remote tags. A real publish is done only if the coordinator authorizes it.
- Unresolved questions: none.
