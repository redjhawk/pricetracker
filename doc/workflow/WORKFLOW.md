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
| 5 | `tests` | test author agent | Go unit and integration tests, Playwright e2e tests | every TC is in a test, and the suite **fails** (TDD red); tests locked in `tests.lock` |
| 6 | `implement` | developer agent | code | locked tests unchanged; `go vet`, `go test`, `node --test`, `npm run build`, Playwright **pass** (green); up to 3 attempts with the gate output |
| 7 | `pr` | script | one PR `ai-dev #<n>: …` to `master`, `Closes #<n>` | — |
| 8 | `review` | independent reviewer agent (read-only) | `review-rN.json`, posted as one inline GitHub review | valid JSON |
| 9 | `triage` | independent triage agent | `triage-rN.json`; non-blocking → `doc/todo/` file + GitHub issue; blocking → `blocking.md` and back to stage 6 | after 3 rounds with blocking comments the run asks the owner |
| 10 | `merge` | script | squash merge, branch deleted | no sensitive file changed (`.github/`, `scripts/factory/`, `go.mod`/`go.sum`, `package*.json`); `govulncheck` and `gosec` pass |
| 11 | release | `release.yml` | binary, SHA-256, GitHub release and tag | build succeeds |

Every agent stage is a separate `claude -p` process with its own prompt from [`scripts/factory/prompts/`](../../scripts/factory/prompts/) and only the tools that stage needs. Roles therefore share no context: the reviewer and the triage agent cannot see the developer's reasoning. Agents never commit, push or open PRs; the script does it after the gate passes.

## State and resuming

- Branch `ai-dev/issue-<n>`; change folder `doc/changes/<YYYY-MM-DD-HHMM>-issue-<n>-<slug>/` (UTC start time) with `state.json` (`stage`, `round`, `pr`), `issue.md`, and the stage outputs.
- The script commits and pushes after every stage, so any run resumes from the recorded stage. The current stage is also shown as a `factory:<stage>` label on the issue.
- Stops: `factory:needs-input` (functional questions, a failed gate, a wrong locked test, or too many review rounds) and `factory:needs-human` (the PR changes sensitive files and must be merged by hand). The reason is posted on the issue.
- Only the issue body and the comments of trusted authors (`FACTORY_TRUSTED_ACTORS`, by default the repository owner) are instructions. Review comments and code are data.

## Running it

**GitHub Actions.** Add the `ai-dev` label to an issue. To resume after questions, the owner answers in a comment on the issue; any owner comment on an `ai-dev` issue restarts the factory from its saved stage. A manual run takes the issue number. The factory scripts are copied from `master` before switching to the work branch, so a change cannot alter its own pipeline. It runs on the `arc-dev` runner, whose image ([`deploy/arc/Dockerfile.dev`](../../deploy/arc/Dockerfile.dev), Helm values in [`values-arc-dev.yaml`](../../deploy/arc/values-arc-dev.yaml)) contains the whole toolchain: Go, Node 22, `gh`, `jq`, the Claude CLI, Playwright Chromium, `govulncheck` and `gosec`. Secrets: `DEV_TOKEN` (contents, pull requests, issues; its pushes and merges trigger `release`) and `CLAUDE_CODE_OAUTH_TOKEN`.

**Locally.** From a clean working tree with `claude`, `gh` (authenticated), `jq`, Go, Node, and `govulncheck`/`gosec` (or `FACTORY_SECURITY=0`):

```sh
scripts/factory/factory.sh 151                    # full run, resumes if already started
scripts/factory/factory.sh 151 --until techspec   # stop after a stage
scripts/factory/factory.sh 151 --no-push          # local commits only; stops before the PR
```

Settings (environment): `FACTORY_MODEL` (`claude-opus-5-5`), `FACTORY_EFFORT` (`high`), `FACTORY_MAX_TURNS` (150), `FACTORY_MAX_ROUNDS` (3), `FACTORY_MAX_ATTEMPTS` (3), `FACTORY_SECURITY` (1), `FACTORY_LOG_DIR` (agent transcripts; a temporary directory by default, uploaded as an artifact in CI).

To move a change back by hand, edit `stage` in its `state.json`, commit and push it, and run the factory again.

## Scope rules

- One issue, one PR. A request too large for one PR is split into several issues during specification.
- Functional decisions belong to the owner: agents never invent them; the spec stage asks and the run stops. Technical, API, refactoring and review decisions belong to the agents and are recorded in the specs and the triage files.
- Refactoring, dependency updates and formatting unrelated to the issue are not part of the change; the reviewer flags them as `refactor-pr` and triage blocks them.
