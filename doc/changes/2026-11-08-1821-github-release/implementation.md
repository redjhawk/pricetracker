# Implementation note: GitHub release script

- New `scripts/create-github-release.sh` (executable): prechecks in order (tools `go npm gh git`, clean `git status --porcelain`, `gh auth status`, `git ls-remote --tags --refs origin`), then `build-release.sh amd64`, tag number from local tags plus the already fetched remote tag list, annotated tag on HEAD, push, `gh release create --verify-tag --title TAG --generate-notes` with the binary copied to a temp dir as `pricefollower-linux-amd64`. Post-tag failures name the tag. Covers FR-GHREL-01 to 07.
- `DEPLOYMENT.md`: short "Publish a GitHub release (amd64)" section. `README.md` not changed (user changes preserved).
- No refactoring; `scripts/build-release.sh` untouched.

## Checks executed

- `bash -n`: pass. `shellcheck`: not installed.
- Scratch clone with a local bare origin and stub `npm`/`go`/`gh` (nothing pushed to the real origin, no real release): missing `gh` on PATH -> exit 1, no tag; untracked file -> exit 1; `gh auth` failing -> exit 1; unreachable origin -> exit 1; remote-only `-1` -> picks `-2`; local-only `-3` (with `-1`,`-2` taken) -> picks `-4`.
- Same numbering logic, read-only against the real origin (which has `v2026.11.08-1`): yields `v2026.11.08-2`.

## Review fixes (decisions.md)

- RV-01: after the auth check, `gh repo view "$(git remote get-url origin)" --json nameWithOwner` resolves the repository (clear error if it fails); `gh release create` gets `--repo "$REPO"`.
- RV-02: remote tags reduced to ref names (`cut -f2`) and matched exactly with `grep -qxF`.
- RV-03: `export GIT_TERMINAL_PROMPT=0` at the top.

Re-checks: `bash -n` pass. Fresh scratch clone, stub `gh` handling `repo view`: missing `gh`, dirty tree, unauthenticated gh, failing `repo view`, unreachable origin -> each exit 1 with no tag; remote-only `-1` plus decoy `v2026x11x08-1` -> `-2` with `--repo owner/repo`; local-only `-3` -> `-4`; a decoy-only list (`v2026x11x08-1`, `foo/refs/tags/v2026.11.08-1`) -> `-1`. Read-only against the real origin: repo resolves to `redjhawk/pricetracker`, next tag `v2026.11.08-2`.

## QA fixes (decisions.md, "QA findings")

- QA-O1: the build call is wrapped: `fail "building the amd64 release failed; no tag or release was created."`.
- QA-14: the release-failure message no longer says "retry". It says not to rerun the script and prints the exact manual command (`cp <repo>/release/pricefollower /tmp/pricefollower-linux-amd64 && gh release create TAG --repo REPO --verify-tag --title TAG --generate-notes /tmp/pricefollower-linux-amd64`; the copy keeps the asset name, because the script's temporary folder is deleted on exit), plus the tag-removal command as before. The sentence in `DEPLOYMENT.md` was updated to match.

Re-checks: `bash -n` pass. Scratch clone with stubs: failing `go` -> build output, then the new `Error:` line, exit 1, tag count unchanged; failing `gh release create` -> tag `v2026.11.08-5` pushed to the scratch origin, then the new message with `--repo owner/repo`, exit 1.
