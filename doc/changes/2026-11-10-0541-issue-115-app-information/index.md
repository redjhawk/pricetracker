# Change: application information modal (issue #115)

- Issue: https://github.com/redjhawk/pricetracker/issues/115
- Started (UTC): 2026-11-10 05:41
- Current stage: technical-specification (functional specification ready)

## Scope

Add an **Information** entry to the top-right application menu that opens a modal showing the current user's username and the running release version.

## User decisions (issue #115 comment of 2026-11-10T04:50Z, redjhawk)

1. Do not show the email (the original email request is withdrawn).
2. Open mode (no logged-in user): the modal shows only the version.
3. The administrator also gets the entry and sees the version.
4. Version = the release tag `vYYYYMMDDHHMMSS` embedded in the binary at build time; in development show the latest tag followed by `-dev`.
5. Interface texts in English ("Information").

## Artifacts

- Functional: `doc/specifications/app-information/functional.md`
- Technical: `doc/specifications/app-information/technical.md`
- API: `API_SPECIFICATION.md`
- Review: `review.md`; decisions: `decisions.md`; QA: `qa.md`; commits: `commit-step.md`
