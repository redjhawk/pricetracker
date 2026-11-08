# GitHub release script: more robust recovery hints

Status: todo (non-blocking review recheck findings, adjudicated on 2026-11-08; not scheduled)

## Problem

- RC-02 `scripts/create-github-release.sh` (release-failure message): `$PROJECT_DIR` is printed unquoted in the copy-paste manual command, so pasting it fails if the clone path has spaces or shell metacharacters.
- RC-03 same message: if `gh release create` creates the release but then fails to upload the asset, the printed `gh release create` command fails with "release already exists"; the right recovery is `gh release upload "$TAG" <binary>` or deleting the release.

## Suggested work

Quote the path with `printf '%q'` (or double quotes) in the message, and mention `gh release upload` as the recovery for an existing release without its asset.

## Source

- Review recheck in `doc/changes/2026-11-08-1821-github-release/review.md`; decisions in `doc/changes/2026-11-08-1821-github-release/decisions.md`.
