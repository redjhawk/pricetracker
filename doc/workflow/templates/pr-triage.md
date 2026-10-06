# PR comment triage: <change>

Triage agent: <agent, independent of developer and PR reviewer>

## Rounds

### Round <k> (<timestamp>, PR #<n> at <head sha>)

| Comment (link) | Label | Decision (blocking / non-blocking) | Reason | Outcome |
|---|---|---|---|---|
| <url> | [bug] | blocking | <why> | fixed in <commit message> / re-reviewed in <review url> |
| <url> | [nit] | non-blocking | <why> | `doc/todo/<file>.md`; issue #<n> or "issue creation failed: <error>" |

## Merge

- No blocking comments remain: <yes, at round k>
- Planned merge order (PR, base, method):
- Merge results and deploy: reported on <originating issue comment / final run report>, not in this file (it is committed before the first merge)
