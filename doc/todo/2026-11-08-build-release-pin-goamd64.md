# Release build: pin the amd64 microarchitecture level

Status: todo (non-blocking PR #107 review comment, triaged on 2026-11-08; not scheduled)

## Problem

`scripts/build-release.sh` builds the `amd64` target with `GOARCH=amd64` only, so a `GOAMD64` value exported in the caller's environment (e.g. `v3`) is inherited and the published binary may not run on older x86-64 CPUs.

## Suggested work

Set `GO_ENV=(GOARCH=amd64 GOAMD64=v1)` for the amd64 target.

## Source

- PR comment: https://github.com/redjhawk/pricetracker/pull/107#discussion_r4507319857
- Triage: `doc/changes/2026-11-08-1821-github-release/pr-triage.md`
- Issue: https://github.com/redjhawk/pricetracker/issues/108
