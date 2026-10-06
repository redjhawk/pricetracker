# ai-dev permissions: technical specification

Status: ready
Requirements: [functional](functional.md).

Audit of the documented commands against the allowlist and job permissions on 2026-10-07:

| Need (source) | Before | Change |
|---|---|---|
| Measure PR size with `scripts/pr-size.sh <base>` (WORKFLOW stage 8) | not allowed | allow `scripts/pr-size.sh:*` and `./scripts/pr-size.sh:*` (the script only runs `git diff` and is read-only) |
| Read review comments and reviews for triage (stage 10) | not allowed | allow `gh api repos/redjhawk/pricetracker/pulls/*/comments` and `.../reviews` for reads (intended as GET; the glob is not strict, see #74) |
| Update a PR's title, description or base (stage 8 and the stack) | `gh pr edit` allowed, but it fails here with a Projects (classic) GraphQL error (verified) | allow `gh api repos/redjhawk/pricetracker/pulls/* --method PATCH --input *`, and say so in the prompt |
| Read the originating issue and its answers (functional gate, resumed runs) | not allowed | allow `gh issue view:*` (read-only) |
| Report the deploy run after `gh workflow run` (stage 10) | not allowed | allow `gh run list --workflow ci-deploy.yml:*` (read-only) |
| Commit messages via `$(cat <<EOF)` (common agent habit) | `cat` not allowed | the prompt says to use `git commit -m` or `-F <file>` |
| Which comments reach Claude (public repository) | every commenter | `include_comments_by_actor: redjhawk,github-actions[bot]`: only the owner's and the workflow's own comments enter Claude's context. This is the user's change, included at their request on 2026-10-07 |
| Push changes under `.github/workflows/` | impossible: `GITHUB_TOKEN` has no `workflows` permission | cannot be granted. Recorded in the header comment and in the prompt: do the rest, record the limitation, tell the user |

Already sufficient: `contents`, `pull-requests`, `issues` and `actions: write`; branch creation and push; the merge forms; the retarget; `gh workflow run`; `gh issue create` and `gh issue comment`; the git commands used by stages 8–10.

The `*` globs in the PATCH entries and in the `.../comments` and `.../reviews` read entries can match extra arguments. This is a known non-blocking hardening item (issue #74). WORKFLOW.md stage 8 no longer tells the run to retarget with `gh pr edit --base`. Frontend, backend and API: not affected. Verification: a YAML parse and `git diff --check`.
