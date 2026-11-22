# Agent factory workflow

Development requests go through the agent factory, [`scripts/factory/factory.sh`](../../scripts/factory/factory.sh). It runs the same way on a workstation and in GitHub Actions ([`fabrica.yml`](../../.github/workflows/fabrica.yml)). It takes one GitHub issue to one merged pull request. The push to `master` then runs [`release.yml`](../../.github/workflows/release.yml), which builds the binary and publishes the tagged release `v<UTC timestamp>`.

## Stages

| # | Stage | Who | Output | Gate (deterministic) |
|---|---|---|---|---|
| 1 | `spec` | functional specifier agent | `doc/specifications/<subject>/functional.md` (`FR-<SUBJECT>-NNN`), `doc/FUNCTIONAL_SPECIFICATIONS.md` | a functional spec changed and defines FR IDs |
| ⏸ | questions | — | `questions.md`, posted on the issue | the run stops until the owner answers in a comment |
| 2 | `usecases` | use-case agent | `doc/use-cases/<subject>/UC-*.md` | every new use case cites an FR ID |
| 3 | `testcases` | test designer agent | `<change>/testcases.md` (`TC-NNN`, Given/When/Then, level) | every FR ID has at least one TC |
| 4 | `techspec` | technical specifier agent | `technical.md` per subject, `API_SPECIFICATION.md` | each subject has a technical spec |
| 5 | `tests` | test author agent | Go unit and integration tests, Playwright e2e tests | every TC is in a test, and the suite **fails** (TDD red); tests locked in `tests.lock`. Before the agent runs, the script merges the base branch and resets every changed path outside `doc/`, `tests/` and `API_SPECIFICATION.md` to it, so code from an earlier implementation attempt cannot make the red gate pass |
| 6 | `implement` | developer agent | code | locked tests unchanged; `go vet`, `go test`, `node --test`, `npm run build`, Playwright **pass** (green); up to 3 attempts with the gate output. Each round first merges the base branch (see below) |
| 7 | `pr` | script | one PR `ai-dev #<n>: …` to `master`, `Closes #<n>` | — |
| 8 | `review` | independent reviewer agent (read-only) | `review-rN.json`, posted as one inline GitHub review | valid JSON |
| 9 | `triage` | independent triage agent | `triage-rN.json`; non-blocking → `doc/todo/` file + GitHub issue; blocking → `blocking.md` and back to stage 6 | after 3 rounds with blocking comments, 3 more start once without asking; after those the run asks the owner |
| 10 | `merge` | script | squash merge, branch deleted | no sensitive file changed (`.github/`, `scripts/factory/`, `go.mod`/`go.sum`, `package*.json`); `govulncheck` and `gosec` pass |
| 11 | release | `release.yml` | binary, SHA-256, GitHub release and tag | build succeeds |

When the gate of stages 1 to 5 fails, the agent runs once more with the gate's output (`FACTORY_STAGE_ATTEMPTS`), and only then does the run stop.

Every agent stage is a separate `claude -p` process with its own prompt from [`scripts/factory/prompts/`](../../scripts/factory/prompts/) and only the tools that stage needs. Roles therefore share no context: the reviewer and the triage agent cannot see the developer's reasoning. Agents never commit, push or open PRs; the script does it after the gate passes.

## State and resuming

- Branch `ai-dev/issue-<n>`; change folder `doc/changes/<YYYY-MM-DD-HHMM>-issue-<n>-<slug>/` (UTC start time) with `state.json` (`stage`, `round`, `pr`), `issue.md`, and the stage outputs.
- The script commits and pushes after every stage, so any run resumes from the recorded stage. The current stage is also shown as a `factory:<stage>` label on the issue.
- A developer who finds a wrong locked test writes `test-problem.md`; the run deletes `tests.lock` and returns to `tests`, where the test author checks the report against the specifications. The first time per issue (`test_problem_returns` in `state.json`) this happens without asking; the second time the run stops for the owner. The tests stage resets the implementation (see stage 5) and deletes `test-problem.md` once the red gate passes.
- A developer who cannot finish for a reason outside the change writes `blocked.md`, whose first line names the category. `Category: environment` stops the run (`blocked-env`). `Category: base-branch` (tests outside the change fail) makes the factory run the whole suite on the base branch in a separate worktree. If it passes there, the failures come from the change and the developer gets them back as the next attempt. If it fails there too, the factory opens an issue labelled `base-bug` and `ai-dev`, so the factory fixes the base branch first, and stops with `waiting_on` that issue. While one `base-bug` issue is open, other blocked changes wait for it instead of opening another.
- Each implementation round starts by merging the base branch, so a base fix reaches the change. After the TDD red commit, test files may change only through that merge: its commit, `chore(factory): merge <base> into #<n> (tests relocked)`, becomes the test reference (`test_ref` in `gates.sh`), with `tests.lock` computed again for the same files. A merge conflict stops the run for the owner.
- Stops: `factory:needs-input` (functional questions, a failed gate, a wrong locked test the second time, or too many review rounds), `factory:needs-human` (sensitive files, a merge conflict, a failed merge), `factory:blocked` (environment), `factory:waiting` (a base bug), `factory:waiting-limit` (Claude usage limit) and `factory:failed` (any other error). The reason is posted on the issue. Every stop comment carries a machine-readable line, `<!-- factory-stop {"reason": …, "stage": …, "round": …} -->`, with `resets_at` for a usage limit, `waiting_on` for a base bug and `gate` for a failed gate. A run that ends with an error (a usage limit, a failed command, an agent failure) reports it through an `EXIT` trap; a cancelled run reports nothing.
- Only trusted authors (`FACTORY_TRUSTED_ACTORS`, by default the repository owner) give instructions. The issue body is passed to the agents only when a trusted author opened the issue; otherwise `issue.md` says it was dropped and the owner must restate the request in a comment. Comments by other authors, the factory's own comments, review comments and code are data.
- When the review rounds reach `FACTORY_MAX_ROUNDS`, a new series starts. The first extra series per issue (`extra_series` in `state.json`) starts without asking; after that the factory sets `stage` to `implement` before stopping, so the owner's answer resumes implementation with `blocking.md` and their comment.
- `stage` becomes `done` only after `gh pr merge` succeeds; a failed merge is reported on the issue (`factory:needs-human`) and the next run retries it.
- Non-blocking triage decisions become `doc/todo/issue-<n>-<id>.md`; a file that already records its GitHub issue is skipped, so resuming never duplicates issues.

## Security model

- Agents run with `GH_TOKEN`/`GITHUB_TOKEN` removed from their environment and with `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`, so the commands they run do not see the Claude credentials. Tests, build and Playwright run with every credential removed. Only `factory.sh` itself uses `DEV_TOKEN`, to push, comment, open and merge.
- Tools are per profile: `docs` edits only `doc/**` and `API_SPECIFICATION.md`; `code` edits only `cmd/`, `internal/`, `src/`, `tests/`, `web/` and the change folder, and runs only `go test|vet|build`, `gofmt`, `npm run build`, `npx playwright test` and `node --test`.
- Gates see committed, modified and untracked files, because they run before the stage checkpoint.
- `tests.lock` is read from the test reference commit (the TDD red commit or the factory's later merge of the base branch); it covers the changed tests plus `playwright.config.ts` and `scripts/*.test.mjs`. A missing or empty lock, or a changed lock, fails the green gate, and so does any test file (new or pre-existing, matching `_test.go`, `.spec.ts`, `.test.mjs`) that differs from the red commit.
- The red gate fails if a changed Playwright spec is not listed by `npx playwright test --list` (`testMatch` in `playwright.config.ts` is explicit and only a human may change it).
- `round` counts every review round and never resets, so review ids and `review-rN.json`/`triage-rN.json`/todo files stay unique; `series_start` marks where the current series of `FACTORY_MAX_ROUNDS` rounds began.
- Triage fails closed: exactly one decision per review id (`R<round>-<n>`), boolean `blocking`, and `title`/`todo` for non-blocking ones.
- Never merged by the factory: `.github/`, `.claude/`, `.agents/`, `project-skills/`, `AGENTS.md`, `CLAUDE.md`, `scripts/`, `deploy/`, dependencies, `vite.config.ts` and `playwright.config.ts`.
- Every factory comment starts with `<!-- agent-factory -->`, and `fabrica.yml` ignores comments that contain it, so the factory cannot restart itself. A supervisor relaunch is the one comment by the owner's token without it: `fabrica.yml` starts on it, and `refresh_issue_context` keeps it, like the factory's own comments, out of the agents' `issue.md`.
- The supervisor trusts stop markers only in comments by trusted authors, so a comment by anyone else cannot make it relaunch a run.
- [`pruebas.yml`](../../.github/workflows/pruebas.yml) runs go vet, go test, the script tests and the frontend build on every PR into `master`, without secrets, so PRs written by hand also have CI. `actions/checkout` in `fabrica.yml` uses `persist-credentials: false`; pushes authenticate through `gh auth setup-git`, which reads `GH_TOKEN` only from `factory.sh`'s environment. The ARC runner files (`deploy/arc/`) stay in this PR because `fabrica.yml` cannot run without the `arc-dev` image; the split was judged not worth an unrunnable intermediate state.

## Supervisor

[`supervisor.yml`](../../.github/workflows/supervisor.yml) runs [`scripts/factory/supervisor.mjs`](../../scripts/factory/supervisor.mjs) every 15 minutes on `arc-dev`. GitHub's schedule is best effort and never fired in this repository, so each run ends with a job on a GitHub-hosted runner that waits 13 minutes and dispatches the next run, unless a newer run already exists; the schedule stays as a backup. It reads the open `ai-dev` issues, their comments and their `fabrica` runs, and resumes the runs that stopped for a reason that goes away without the owner:

| Stop | Supervisor |
|---|---|
| `usage-limit` | relaunches 5 minutes after `resets_at`; without it, as soon as the probe passes. While one issue waits for a reset, no other issue is relaunched (the limit is per account) |
| `error`, `blocked-env`, `merge-failed`, or a run that failed without reporting a stop (crash) | relaunches 15, 30 and 45 minutes after the stop, at most 3 times at the same stage and round, then gives up: one comment and `factory:needs-human` |
| `base-bug` | relaunches when the `waiting_on` issue is closed; the run merges the base branch before implementing |
| an owner comment that started no run | relaunches 15 minutes after the comment, once |
| an issue labelled `ai-dev` or launched whose run never started | relaunches 15, 30 and 45 minutes later, then gives up |
| anything else (questions, failed gate, sensitive files, …) | waits for the owner |

- Before relaunching anything, a probe agent runs `cat` on a file with a random value through the Bash tool, with the same environment as the factory's agents. When it fails for another reason than the usage limit, the supervisor opens one issue labelled `factory-infra` with the probe output and relaunches nothing; it closes that issue when the probe passes again.
- A relaunch is a comment `<!-- factory-supervisor {…} -->` that names the reason and the attempt, and that starts `fabrica.yml`. Retries are counted from these comments since the last owner comment; an owner comment starts the count again.
- Off switch: the repository variable `FACTORY_SUPERVISOR=off`; after switching it back on, run the workflow once by hand to restart the loop. The label `factory-hold` excludes one issue. `node scripts/factory/supervisor.mjs --dry-run` prints the decisions without acting.

### Queue

The supervisor also starts new issues, so the factory always has work:

- An issue is **in development** while the factory goes on with it without the owner: a run is queued or running, or the supervisor will relaunch it (retry, usage limit, base bug, a run that has not started). An issue stopped for the owner (questions, failed gate, merge conflict, given up…) or labelled `factory-hold` frees its place.
- While fewer than `FACTORY_MAX_ACTIVE` issues (repository variable, 1 by default) are in development and no issue waits for the usage limit, the supervisor launches the next candidate: it posts a comment with the reason and the rest of the queue, then adds `ai-dev` with the owner's token, which starts `fabrica.yml`. Its probe must pass first, as for a relaunch. One at a time keeps the branches free of conflicts with each other and spends the usage limit slowest.
- **Candidates:** open issues opened by a trusted author (the owner, or the factory with the owner's token) that have never been in the factory (no `ai-dev` and no `factory:*` label) and are not labelled `factory-hold`, `factory-infra`, `question`, `wontfix`, `duplicate` or `invalid`. Issues by anyone else are never launched or shown to the ranking agent.
- **Order:** the owner's `priority:high` issues first and `priority:low` last; within each, the order of a Claude agent (`FACTORY_RANK_MODEL`, `claude-sonnet-5-5`, no tools) that ranks the candidates by bugs that block, dependencies, size and age. When it fails, oldest first. A single candidate is launched without asking it.
- `FACTORY_MAX_ACTIVE=0` stops the launches and keeps the relaunches.
- The decisions and the queue are tested in [`scripts/factory-supervisor.test.mjs`](../../scripts/factory-supervisor.test.mjs).

## Running it

**GitHub Actions.** Add the `ai-dev` label to an issue. To resume after questions, the owner answers in a comment on the issue; any owner comment on an `ai-dev` issue restarts the factory from its saved stage. A manual run takes the issue number. The factory scripts are copied from `master` before switching to the work branch, so a change cannot alter its own pipeline. It runs on the `arc-dev` runner, whose image ([`deploy/arc/Dockerfile.dev`](../../deploy/arc/Dockerfile.dev), Helm values in [`values-arc-dev.yaml`](../../deploy/arc/values-arc-dev.yaml)) contains the whole toolchain: Go, Node 22, `gh`, `jq`, the Claude CLI, Playwright Chromium, `govulncheck` and `gosec`. Secrets: `DEV_TOKEN` (contents, pull requests, issues; its pushes and merges trigger `release`) and `CLAUDE_CODE_OAUTH_TOKEN`.

**Locally.** From a clean working tree with `claude`, `gh` (authenticated), `jq`, Go, Node, and `govulncheck`/`gosec` (or `FACTORY_SECURITY=0`):

```sh
scripts/factory/factory.sh 151                    # full run, resumes if already started
scripts/factory/factory.sh 151 --until techspec   # stop after a stage
scripts/factory/factory.sh 151 --no-push          # local commits only; stops before the PR
```

Settings (environment): `FACTORY_MODEL` (`claude-opus-5-5`), `FACTORY_EFFORT` (`high`), `FACTORY_MAX_TURNS` (150), `FACTORY_MAX_ROUNDS` (3), `FACTORY_MAX_ATTEMPTS` (3), `FACTORY_STAGE_ATTEMPTS` (2), `FACTORY_SECURITY` (1), `FACTORY_LOG_DIR` (agent transcripts; a temporary directory by default, uploaded as an artifact in CI).

To move a change back by hand, edit `stage` in its `state.json`, commit and push it, and run the factory again.

## Scope rules

- One issue, one PR. A request too large for one PR is split into several issues during specification.
- Functional decisions belong to the owner: agents never invent them; the spec stage asks and the run stops. Technical, API, refactoring and review decisions belong to the agents and are recorded in the specs and the triage files.
- Refactoring, dependency updates and formatting unrelated to the issue are not part of the change; the reviewer flags them as `refactor-pr` and triage blocks them.
