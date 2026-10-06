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
- Merges in order (PR, base, method, result):
- Deploy: <ci-deploy run link or how it was started>
- User informed: <issue comment link / final run report>
