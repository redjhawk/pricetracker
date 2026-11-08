# Commit step: GitHub release script

## Scope

Single PR to `master` (about 480 changed lines, under the 500 ceiling), from branch `feature/2026-11-08-1821-github-release`.

Included:

- `scripts/create-github-release.sh` (new) and its `DEPLOYMENT.md` section.
- Specifications `doc/specifications/github-release/`, the `doc/FUNCTIONAL_SPECIFICATIONS.md` row, this change folder, and the deferred items in `doc/todo/2026-11-08-github-release-recovery-hints.md`.

Excluded on the user's instruction: the uncommitted user changes to `README.md` and `scripts/build-release.sh` (the `amd64` target). The new script calls `scripts/build-release.sh amd64`, so it only works on `master` once the user commits that change.

## Checks

- `bash -n scripts/create-github-release.sh`: passed. `shellcheck` is not installed.
- Review, decisions and QA: see [review.md](review.md), [decisions.md](decisions.md), [qa.md](qa.md). QA passed; no critical findings.

## Commits

1. `docs(github-release): specifications, review, decisions and QA records`
2. `feat(scripts): create GitHub releases of the amd64 binary`

## Limitations

- The real publish to GitHub was not run by QA. The user authorized a real release (`v2026.11.08-2`), but it needs a clean tree with the `amd64` build target on `master`, which depends on the user's own commit. It runs after merge.
- Release `v2026.11.08-1` was published manually by the coordinator at the user's request, before the script existed.
