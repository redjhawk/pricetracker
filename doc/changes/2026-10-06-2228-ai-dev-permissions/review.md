# Review: ai-dev permissions

Reviewer: independent stage 5 reviewer. Scope: uncommitted `git diff` of `.github/workflows/ai-dev.yml` and the [functional](../../specifications/ai-dev-permissions/functional.md) and [technical](../../specifications/ai-dev-permissions/technical.md) specifications.

Method: every command written in `AGENTS.md`, `doc/workflow/WORKFLOW.md`, `.agents/roles/*.md` and the `--append-system-prompt` text was matched literally against the `--allowedTools` list (`Bash(prefix:*)` = prefix match, any other `*` = glob that also matches spaces). Job permissions were checked against the GitHub calls those commands make.

## Coverage confirmed

| Need | Command | Matching entry |
|---|---|---|
| Feature branch from master | `git fetch origin master`; `git checkout -b ai-dev/issue-<n>-feature origin/master` | `git fetch origin:*`; `git checkout -b ai-dev/*` |
| Part branches, splitting | `git checkout -b ai-dev/...`, `git checkout ai-dev/<b> -- <paths>`, cherry-pick, restore, reset, `git rebase --onto` | matching entries present |
| Commits / pushes | `git add`, `git commit`, `git push -u origin HEAD`, `git push origin HEAD` (WORKFLOW), `git push --force-with-lease origin HEAD` | exact entries present |
| Pre-commit checks | `git diff --cached --check`, `git diff --numstat`, `git log` | `git diff:*`, `git log:*` |
| PR size | `scripts/pr-size.sh <base>` (runs only `git diff`) | `scripts/pr-size.sh:*` (new) |
| PR create / update | `gh pr create ...`; `gh api .../pulls/<n> --method PATCH --input <file>` | present (PATCH entry new) |
| PR review post / read | `gh api .../pulls/<n>/reviews --method POST --input <file>`; `gh pr diff <n>`; `.../comments`, `.../reviews` | present (GET entries new) |
| Issues | `gh issue create --title .. --body ..`, `gh issue comment`, `gh issue view` | present (`view` new) |
| Merges, retarget, ancestor | `gh pr merge <n> --merge --delete-branch`, `--squash --delete-branch`, `... --method PATCH -f base=<b>`, `git merge-base --is-ancestor` | present |
| Deploy | `gh workflow run ci-deploy.yml --ref master`; `gh run list --workflow ci-deploy.yml --limit 1` | present (`run list` new); `actions: write` granted |
| QA | `npm run dev`, `npx playwright ...`, `node <script>`, `curl http://localhost...`, `pkill -f vite` | present |
| Change folder time | `date -u +%Y-%m-%d-%H%M` | `date -u:*` |

Job permissions `contents`, `pull-requests`, `issues`, `actions: write` cover all of the above. The `.github/workflows/` push gap (no `workflows` permission for `GITHUB_TOKEN`) is correctly recorded as a limitation in the header and the prompt (FR-AIDEV-PERM-002).

## Findings

### REV-001 — `include_comments_by_actor` is outside the specifications (spec mismatch, medium)

The diff adds `include_comments_by_actor: redjhawk,github-actions[bot]`. Neither specification mentions it; the technical audit table lists only allowlist and prompt changes. It is a security-relevant behavior change (it filters which comments reach Claude's context) and may be desirable, but it is implemented behavior absent from the specifications. Either add it to the technical spec with its rationale, or move it to its own change.

### REV-002 — WORKFLOW.md still prescribes `gh pr edit --base` (documentation inconsistency, low)

`doc/workflow/WORKFLOW.md` stage 8 ("Feature branch and stacking") says to retarget with `gh pr edit --base`, while the technical spec, the prompt and WORKFLOW stage 10 say `gh pr edit` fails here and the REST PATCH must be used. The command is allowed but does not work, so an agent following stage 8 literally will fail once. The scope of this change is the workflow file only, so this is out of scope; record it as a follow-up (or extend the scope to fix that one sentence).

### REV-003 — Commit messages built with `$(cat <<EOF ...)` may prompt/deny (low)

`git commit:*` matches the prefix, but Claude Code also checks command substitutions; `cat` is not allowlisted, so the common heredoc commit form can be denied in the non-interactive run. `git commit -m "..."` with multi-line text or `git commit -F <file written with Write>` works. Not a missing right strictly, but the prompt could say "use `git commit -F <file>` or `-m`" to avoid a failed attempt. Non-blocking.

### REV-004 — New `gh api` globs are broader than their documented use (over-broad, low; known)

- `gh api repos/redjhawk/pricetracker/pulls/* --method PATCH --input *` allows any PR field (`state: closed`, `base: master`) on any PR. Already acknowledged in the technical spec as issue #74.
- `.../pulls/*/comments` and `.../pulls/*/reviews` (GET-only intent): the middle `*` also matches spaces, so extra flags can be inserted before the literal suffix (e.g. `pulls/1 -X POST -f body=x .../comments` shape). `gh api` accepts a single endpoint argument, which limits practical abuse, but the entries are not strictly "no extra arguments" as the spec claims. Non-blocking; fold into #74.

No over-broad entry beyond these. `scripts/pr-size.sh:*`, `gh issue view:*`, `gh run list --workflow ci-deploy.yml:*` are read-only.

## Missing rights

None found for stages 1–10: every documented command has a matching entry or a recorded limitation. (Not documented and therefore not required: `gh run view/watch`, pushing a non-current branch by name — agents switch with `git switch ai-dev/*` first.)

## Verdict

No critical findings. REV-001 needs a decision (spec it or drop it); REV-002–004 are non-blocking.
