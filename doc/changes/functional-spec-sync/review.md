# Review: functional-spec-sync

Independent reviewer agent, uncommitted diff of `doc/FUNCTIONAL_SPECIFICATIONS.md`.

- REV-1 (major): §5.6 omitted the overall recommendation (Buy / Negotiate / Avoid) and rating scales (FR-LBC-AIR-003).
- REV-2 (minor): §5.6 omitted that the refresh button is disabled without a token and that saving a token later reviews nothing automatically (FR-LBC-AIR-005, -008).

Links and statements vs subject specs checked: no other findings.

## Round 2: multi-user phase 1 (after PR #41 merged to master)

- REV-3 (low): §5.5 and §7 said the header menu has Settings on every page; the administrator's menu has only Log out (src/App.tsx).
