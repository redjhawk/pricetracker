# Pull request review: 2026-11-08-1821-github-release

Reviewer: PR reviewer agent (stage 9), independent of the developer, the reviewer and the adjudicator
Reviewed at: 2026-11-08 (UTC)

| PR | Head revision | Review URL | Event (REQUEST_CHANGES / COMMENT) | [bug] | [refactor-pr] | [simplify] | [readability] | [test] | [nit] |
|---|---|---|---|---|---|---|---|---|---|
| #100 | b6e87bfd9fe771d86eac659a0ddc33423970cd8c | https://github.com/redjhawk/pricetracker/pull/100#pullrequestreview-5819102248 | COMMENT, body starts with "CHANGES REQUESTED" (GitHub rejected REQUEST_CHANGES with HTTP 422 because this is our own PR) | 1 | 0 | 1 | 1 | 0 | 3 |

Known context, not a defect of this PR: the script uses the `build-release.sh amd64` target, which the user commits separately.

## Comments

- [bug] `scripts/create-github-release.sh:53`: the script tags and releases HEAD without checking that HEAD is on origin. https://github.com/redjhawk/pricetracker/pull/100#discussion_r4507310350
- [simplify] line 29: `gh repo view` with no argument resolves the repository itself; passing the origin URL is unnecessary and depends on how `gh` parses URLs. https://github.com/redjhawk/pricetracker/pull/100#discussion_r4507310354
- [nit] line 33: the remote tag list is read before a long build, so a tag created during the build is missed. https://github.com/redjhawk/pricetracker/pull/100#discussion_r4507310358
- [nit] line 53: an unwrapped `git tag -a` fails with git's raw message when the git identity is not configured. https://github.com/redjhawk/pricetracker/pull/100#discussion_r4507310362
- [nit] line 59: the recovery message hardcodes `/tmp` and is about 400 characters on one line (overlaps todo RC-02/RC-03). https://github.com/redjhawk/pricetracker/pull/100#discussion_r4507310364
- [readability] line 6: no comment explains why `GIT_TERMINAL_PROMPT=0` is exported. https://github.com/redjhawk/pricetracker/pull/100#discussion_r4507310369

## Possible bugs

- PR #100: a commit that was never pushed can be tagged and released, which pushes it to origin through the tag only. https://github.com/redjhawk/pricetracker/pull/100#discussion_r4507310350. Stage 10 (`pr-triage.md`) decides whether it blocks.

## Re-reviews after fixes (stage 10 blocking fixes or `@claude` fixes)

None yet.

## PR #107

- Review: https://github.com/redjhawk/pricetracker/pull/107#pullrequestreview-5819112468 (head ab0991d)
- Counts: bug 0, nit 1
- Nit: GOAMD64 not pinned for amd64 builds. https://github.com/redjhawk/pricetracker/pull/107#discussion_r4507319857. Stage 10 decided non-blocking.
