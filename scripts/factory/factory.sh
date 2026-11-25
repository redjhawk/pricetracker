#!/usr/bin/env bash
# Agent factory: takes a GitHub issue from specification to a merged pull request.
#
#   scripts/factory/factory.sh <issue> [--no-push] [--until <stage>]
#
# Stages (state in doc/changes/<change>/state.json on branch ai-dev/issue-<n>):
#   spec → usecases → testcases → techspec → tests (TDD red) → implement (green)
#   → pr → review → triage ─ blocking ─► [contract →] implement (max FACTORY_MAX_ROUNDS)
#                         └ none ─────► merge → done   (push to master runs .github/workflows/release.yml)
#
# Each agent stage is a fresh `claude -p` process with its own prompt and tools; deterministic gates
# (gates.sh) decide whether it may advance. Every stage is committed and pushed, so a later run resumes
# where the previous one stopped. The run stops, and asks on the issue, when it needs a functional answer.
# Runs the same locally and in .github/workflows/fabrica.yml.
set -euo pipefail
# No core dumps: a crashed browser or test leaves core.<pid> in its directory, the stage checkpoint
# (git add -A) commits it, and GitHub refuses files over 100 MB, so the push fails.
ulimit -c 0

FACTORY_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$FACTORY_DIR/lib.sh"
# shellcheck source=gates.sh
source "$FACTORY_DIR/gates.sh"

usage() { sed -n '2,4p' "$0" | sed 's/^# \{0,1\}//'; exit 2; }

ISSUE="" UNTIL=""
while [ $# -gt 0 ]; do
  case "$1" in
    --no-push) export FACTORY_PUSH=0 ;;
    --until) UNTIL=$2; shift ;;
    -h|--help) usage ;;
    *) [[ "$1" =~ ^[0-9]+$ ]] || usage; ISSUE=$1 ;;
  esac
  shift
done
[ -n "$ISSUE" ] || usage
for cmd in claude gh git jq go npm; do command -v "$cmd" >/dev/null || die "$cmd is required"; done
# On Linux the Bash tool of the agents runs in Claude Code's sandbox, which needs bubblewrap and socat;
# without them no agent can run a test or a build, so the run stops before changing any stage.
if [ "$(uname -s)" = Linux ]; then
  for cmd in bwrap socat; do command -v "$cmd" >/dev/null || die "$cmd is required by the Claude Code sandbox (runner image: deploy/arc/Dockerfile.dev)"; done
fi

MODEL=${FACTORY_MODEL:-claude-opus-5-5}
EFFORT=${FACTORY_EFFORT:-high}
MAX_TURNS=${FACTORY_MAX_TURNS:-150}
MAX_ROUNDS=${FACTORY_MAX_ROUNDS:-3}     # review → fix rounds before asking the user
MAX_ATTEMPTS=${FACTORY_MAX_ATTEMPTS:-3} # implementation attempts against the green gate per round
STAGE_ATTEMPTS=${FACTORY_STAGE_ATTEMPTS:-2} # attempts of the other agent stages against their gate
BASE_BRANCH=${FACTORY_BASE_BRANCH:-master}
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
TRUSTED_ACTORS=${FACTORY_TRUSTED_ACTORS:-${REPO%%/*}}
LOG_DIR=${FACTORY_LOG_DIR:-$(mktemp -d)}
BRANCH="ai-dev/issue-$ISSUE"

# ---- branch and change folder ----

cd "$(git rev-parse --show-toplevel)"
[ -z "$(git status --porcelain)" ] || die "the working tree is not clean"
git fetch -q origin "$BASE_BRANCH"
if git ls-remote --exit-code --heads origin "$BRANCH" >/dev/null; then
  git fetch -q origin "$BRANCH"
  git switch -q -C "$BRANCH" "origin/$BRANCH"
else
  git switch -q -C "$BRANCH" "origin/$BASE_BRANCH"
fi

CHANGE_DIR=$(find doc/changes -maxdepth 1 -type d -name "*-issue-$ISSUE-*" | sort | head -1)
if [ -z "$CHANGE_DIR" ]; then
  slug=$(gh issue view "$ISSUE" --json title -q .title | iconv -t ascii//TRANSLIT |
    tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-' | cut -c1-40 | sed 's/^-//;s/-$//')
  CHANGE_DIR="doc/changes/$(date -u +%Y-%m-%d-%H%M)-issue-$ISSUE-$slug"
  mkdir -p "$CHANGE_DIR"
  jq -n --argjson i "$ISSUE" '{issue: $i, stage: "spec", round: 0, series_start: 0, pr: ""}' >"$CHANGE_DIR/state.json"
fi
STATE="$CHANGE_DIR/state.json"
export ISSUE CHANGE_DIR BASE_BRANCH LOG_DIR
log "issue #$ISSUE, branch $BRANCH, change $CHANGE_DIR, stage $(state_get stage), logs $LOG_DIR"

# ---- unexpected stops ----

# die, a failed command or a usage limit end the run with a non-zero status. The EXIT trap reports it on the
# issue with a stop marker, so that scripts/factory/supervisor.mjs resumes the run once the cause is gone.
# A --no-push run keeps its state local and reports nothing; neither does a cancelled run (a signal).
FAIL_REASON=error FAIL_EXTRA='{}' FAIL_DETAIL='' FAIL_COMMAND=''
report_failure() {
  local code=$? detail
  set +e
  [ "$code" -ne 0 ] && [ "${FACTORY_PUSH:-1}" = 1 ] || return 0
  detail=${FAIL_DETAIL:-"\`${FAIL_COMMAND:-?}\` failed with exit status $code"}
  if [ "$FAIL_REASON" = usage-limit ]; then
    printf '%s\n\n_Agent factory, stage `%s`. The supervisor relaunches it after the reset; it resumes at this stage._\n%s\n' \
      "The run stopped: $detail" "$(state_get stage)" "$(run_link)" | issue_comment "$(stop_marker usage-limit "$FAIL_EXTRA")"
    set_stage_label waiting-limit
  else
    printf '%s\n\n_Agent factory, stage `%s`. The supervisor retries it._\n%s\n' \
      "The run failed: $detail" "$(state_get stage)" "$(run_link)" | issue_comment "$(stop_marker error)"
    set_stage_label failed
  fi
}
run_link() { [ -z "${GITHUB_RUN_ID:-}" ] || echo "Run: $GITHUB_SERVER_URL/$GITHUB_REPOSITORY/actions/runs/$GITHUB_RUN_ID"; }
set -E # the ERR trap also runs in functions
trap 'FAIL_COMMAND=$BASH_COMMAND' ERR
trap report_failure EXIT
trap 'trap - EXIT; exit 130' INT TERM

# ---- stage transitions ----

advance() { # advance <next stage> <commit message>
  state_set stage "$1"
  checkpoint "$2"
  set_stage_label "$1"
}

# stop_for_user <label> <reason> <message for the issue> [extra JSON for the stop marker]
# The reason (see stop_marker in lib.sh) tells the supervisor whether it may resume the run on its own.
stop_for_user() {
  local next
  case "$2" in
    base-bug) next="The supervisor resumes it when that issue is closed." ;;
    blocked-env|merge-failed) next="The supervisor retries it; you can also answer in a comment." ;;
    *) next="Answer in a comment to resume it." ;;
  esac
  printf '%s\n\n_Agent factory, stage `%s`. %s_\n' "$3" "$(state_get stage)" "$next" | issue_comment "$(stop_marker "$2" "${4:-}")"
  checkpoint "chore(factory): stop at $(state_get stage) ($2)"
  set_stage_label "$1"
  log "stopped: $1 ($2)"
  exit 0
}

# Gate output for humans: the Vite dev server's own log lines (proxy errors while Playwright runs
# without a backend) are noise that hides the result.
gate_log() { grep -v '^\[WebServer\]' || true; }

gate_failed() { # gate_failed <gate function> <its output>; a gate prints its reason as its last line
  printf '%s\n' "$2" >&2
  stop_for_user needs-input gate "The \`$1\` gate failed after the agent finished: **$(gate_log <<<"$2" | tail -1)**

Last lines of its output:
\`\`\`
$(gate_log <<<"$2" | tail -30)
\`\`\`" "$(jq -cn --arg g "$1" '{gate: $g}')"
}

# run_stage <prompt> <tool profile> <gate function> [extra context]
# An agent stage, run again with the gate's output while the gate fails (STAGE_ATTEMPTS): agents sometimes
# miss one item that a gate checks, such as a requirement without a test case.
run_stage() {
  local name=$1 profile=$2 gate=$3 extra=${4:-} attempt out feedback=""
  for attempt in $(seq 1 "$STAGE_ATTEMPTS"); do
    run_agent "$name" "$profile" "$extra$feedback"
    [ "$name" != spec ] || stop_if_questions
    if out=$("$gate" 2>&1); then return 0; fi
    log "attempt $attempt of $name failed $gate: $(gate_log <<<"$out" | tail -1)"
    feedback="
- Attempt $((attempt + 1)) of $STAGE_ATTEMPTS. The previous attempt failed the \`$gate\` gate. Keep its work and fix what the gate reports:
\`\`\`
$(gate_log <<<"$out" | tail -40)
\`\`\`"
  done
  gate_failed "$gate" "$out"
}

stop_if_questions() {
  [ -s "$CHANGE_DIR/questions.md" ] || return 0
  stop_for_user needs-input questions "## Functional questions

$(cat "$CHANGE_DIR/questions.md")"
}

# ---- stages ----

stage_spec() {
  run_stage spec docs gate_spec
  advance usecases "docs(spec): functional specification for #$ISSUE"
}

stage_usecases() {
  run_stage usecases docs gate_usecases
  advance testcases "docs(use-cases): use cases for #$ISSUE"
}

stage_testcases() {
  # The exact list gate_testcases checks: new requirements and existing ones whose definition line changed.
  run_stage testcases docs gate_testcases "- Requirements to cover (each must appear in testcases.md): $(requirement_ids | tr '\n' ' ')"
  advance techspec "docs(test-cases): test cases for #$ISSUE"
}

stage_techspec() {
  run_stage techspec docs gate_techspec
  advance tests "docs(tech): technical specification and API contract for #$ISSUE"
}

# The developer stage cannot edit API_SPECIFICATION.md: the contract belongs to the technical specification.
# A blocking review comment that triage marks as a contract change, or a developer report with
# `Category: contract`, comes here first. The contract author (docs profile) updates the contract and the
# technical specification; implementation then aligns the code and fixes the other blocking comments.
stage_contract() {
  local extra="- Blocking review comments: $CHANGE_DIR/blocking.md (fix those marked \"Contract change needed\")"
  [ -s "$CHANGE_DIR/blocking.md" ] || extra="- Blocking review comments: none"
  [ ! -s "$CHANGE_DIR/contract-request.md" ] || extra+="
- Contract change requested by the developer stage: $CHANGE_DIR/contract-request.md"
  run_stage contract docs gate_contract "$extra"
  rm -f "$CHANGE_DIR/contract-request.md"
  advance implement "docs(api): contract changes for the review of #$ISSUE (round $(state_get round))"
}

stage_tests() {
  prepare_tests
  run_stage tests code gate_red
  rm -f "$CHANGE_DIR/test-problem.md" # answered by this stage
  advance implement "test: failing tests for #$ISSUE (TDD red)"
}

# resolve_conflicts any|no-tests: after a merge of the base branch that stopped on conflicts, a merge agent
# resolves them, with Edit on exactly the conflicted files. Fails, leaving the merge to abort, when the merge
# failed for another reason, when a conflicted test file must stay locked (no-tests: after the TDD red commit,
# where the merge commit becomes the test reference), or when conflict markers remain. The resolved files are
# staged; the gates that follow check the result as usual.
resolve_conflicts() {
  local files
  git rev-parse -q --verify MERGE_HEAD >/dev/null || return 1
  files=$(git diff --name-only --diff-filter=U)
  [ -n "$files" ] || return 1
  if [ "$1" = no-tests ] && grep -qE "$TEST_FILE_PATTERN" <<<"$files"; then
    log "merge conflict in locked test files, left to the owner: $(grep -E "$TEST_FILE_PATTERN" <<<"$files" | tr '\n' ' ')"
    return 1
  fi
  MERGE_FILES=$(tr '\n' ' ' <<<"$files")
  log "merge conflict in: $MERGE_FILES; the merge agent resolves it"
  run_agent merge merge "- Merging origin/$BASE_BRANCH into $BRANCH stopped on conflicts in: $MERGE_FILES"
  # shellcheck disable=SC2086
  if git grep -n -E '^(<<<<<<<|=======$|>>>>>>>)' -- $files >&2; then
    log "conflict markers remain after the merge agent"
    return 1
  fi
  # shellcheck disable=SC2086
  git add -- $files
}

# The red gate needs the code without this change. Bring in the base branch, so the tests are written
# against current code (including test fixes made there), and drop any implementation left on the branch:
# a test problem sends the change back here after the developer has written code, and with that code
# the new tests would pass and the red gate could never succeed. The tests stage writes its stubs anew.
prepare_tests() {
  local out p
  if ! out=$(git merge --no-edit "origin/$BASE_BRANCH" 2>&1) && ! { resolve_conflicts any && git commit -q --no-edit; }; then
    git merge --abort >/dev/null 2>&1 || true
    stop_for_user needs-human merge-conflict "Merging \`$BASE_BRANCH\` into \`$BRANCH\` before writing the tests failed, and the merge agent could not resolve it:
\`\`\`
$(tail -30 <<<"$out")
\`\`\`
Resolve it on the branch and run the factory again."
  fi
  for p in $(production_paths); do
    if git cat-file -e "origin/$BASE_BRANCH:$p" 2>/dev/null; then
      git checkout -q "origin/$BASE_BRANCH" -- "$p"
    else
      git rm -q -f --ignore-unmatch -- "$p"; rm -f -- "$p"
    fi
    log "tests stage: $p reset to $BASE_BRANCH"
  done
}

# The developer may not add or change test files, not even throw-away ones. The agent cannot delete
# files in a non-interactive run, so such a file used to stay, fail the green gate on every attempt and
# be committed by the stop checkpoint. The factory restores every test file to the test reference commit.
discard_test_edits() {
  local commit f
  commit=$(test_ref)
  [ -n "$commit" ] || return 0
  for f in $({ git diff --name-only "$commit" --; git ls-files -o --exclude-standard; } | grep -E "$TEST_FILE_PATTERN" | sort -u); do
    if git cat-file -e "$commit:$f" 2>/dev/null; then
      git checkout -q "$commit" -- "$f"
    else
      git rm -q -f --ignore-unmatch -- "$f"; rm -f -- "$f"
    fi
    log "implement stage: $f restored to the test reference commit"
  done
}

# Each implementation round starts by merging the base branch, so fixes made there reach the change: a
# base bug this change waited for, a corrected test, a newer dependency. After the TDD red commit, test
# files may change only through this merge: its commit becomes the test reference (test_ref in gates.sh)
# and tests.lock is computed again from it.
sync_base() {
  local out
  ! git merge-base --is-ancestor "origin/$BASE_BRANCH" HEAD || return 0
  if [ -n "$(git status --porcelain)" ]; then # test files just restored by discard_test_edits
    git add -A
    git commit -q -m "chore(factory): restore test files before merging $BASE_BRANCH" -m "Refs: #$ISSUE"
  fi
  if ! out=$(git merge --no-ff --no-commit "origin/$BASE_BRANCH" 2>&1) && ! resolve_conflicts no-tests; then
    git merge --abort >/dev/null 2>&1 || true
    stop_for_user needs-human merge-conflict "Merging \`$BASE_BRANCH\` into \`$BRANCH\` before the implementation failed, and the merge agent could not resolve it:
\`\`\`
$(tail -30 <<<"$out")
\`\`\`
Resolve it on the branch and run the factory again."
  fi
  if [ -s "$CHANGE_DIR/tests.lock" ]; then relock_tests; git add "$CHANGE_DIR/tests.lock"; fi
  git commit -q -m "chore(factory): merge $BASE_BRANCH into #$ISSUE (tests relocked)" -m "Refs: #$ISSUE"
  log "implement stage: merged $BASE_BRANCH; test files now refer to $(git rev-parse --short HEAD)"
}

# base_suite: the whole suite on the base branch, in a separate worktree. It checks a developer's report
# that tests outside this change already fail there. Prints the output. Exit status 1 when the suite
# fails, 2 when the worktree cannot be prepared.
base_suite() {
  local dir rc=0
  dir=$(mktemp -d)
  git worktree add -q --detach "$dir" "origin/$BASE_BRANCH" || return 2
  if ! (cd "$dir" && npm ci --prefer-offline --no-audit --no-fund >/dev/null 2>&1); then
    rc=2
  else
    (cd "$dir" && run_all_tests) || rc=1
  fi
  git worktree remove --force "$dir" >/dev/null 2>&1 || rm -rf "$dir"
  return $rc
}

BASE_BUG_LABEL=base-bug # no factory: prefix, which set_stage_label replaces on the bug's own issue

# wait_for_base_fix <developer report> <base suite output>
# A confirmed bug on the base branch blocks this change: open an issue for the factory to fix it on the
# base branch (label ai-dev starts fabrica.yml), then stop until that issue is closed; the supervisor
# resumes this one then. Base bugs are fixed one at a time: while a base-bug issue is open, other blocked
# changes wait for it instead of opening a duplicate, and find out on resume whether their tests pass.
wait_for_base_fix() {
  local bug url
  bug=$(gh issue list --state open --label "$BASE_BUG_LABEL" --json number -q 'map(.number) | min // empty')
  if [ -z "$bug" ]; then
    gh label create "$BASE_BUG_LABEL" --color D93F0B --description "Tests fail on the base branch; blocks agent factory runs" --force >/dev/null 2>&1 || true
    url=$(gh issue create --title "Tests fail on $BASE_BRANCH (blocks #$ISSUE)" --label "$BASE_BUG_LABEL" --body "$(base_bug_body "$1" "$2")") ||
      die "could not open the base-bug issue: $url"
    bug=${url##*/}
    gh issue edit "$bug" --add-label ai-dev >/dev/null || die "could not add the ai-dev label to #$bug"
  fi
  stop_for_user waiting base-bug "The developer reports tests that fail outside this change, and the factory confirmed that the whole suite fails on \`$BASE_BRANCH\` too ($(git rev-parse --short "origin/$BASE_BRANCH")). The fix goes to \`$BASE_BRANCH\` first, in #$bug.

Output on \`$BASE_BRANCH\`:
~~~
$(gate_log <<<"$2" | tail -30)
~~~" "{\"waiting_on\": $bug}"
}

base_bug_body() { # base_bug_body <developer report> <base suite output>
  cat <<EOF
The whole test suite must pass on \`$BASE_BRANCH\`, and today it fails. This blocks the agent factory on #$ISSUE: its developer reported the failures and the factory reproduced them on \`$BASE_BRANCH\` at $(git rev-parse --short "origin/$BASE_BRANCH").

Fix the code or the tests on \`$BASE_BRANCH\` so that \`go vet ./...\`, \`go test ./...\`, \`node --test scripts/*.test.mjs\`, \`npm run build\` and \`npx playwright test\` pass, keeping the behaviour that the functional specifications define.

## Output on $BASE_BRANCH

~~~
$(gate_log <<<"$2" | tail -80)
~~~

## Report of the developer of #$ISSUE

$1

_Opened by the agent factory. When this issue is closed, the supervisor resumes #$ISSUE._
EOF
}

stage_implement() {
  local attempt feedback="" out blocked rc returns
  discard_test_edits
  sync_base
  [ ! -s "$CHANGE_DIR/blocking.md" ] || feedback="Fix the blocking review comments in $CHANGE_DIR/blocking.md."
  for attempt in $(seq 1 "$MAX_ATTEMPTS"); do
    discard_test_edits
    run_agent implement code "- Attempt $attempt of $MAX_ATTEMPTS
$feedback"
    discard_test_edits
    if [ -s "$CHANGE_DIR/blocked.md" ]; then
      # Not a test of this change: the environment or the base branch. Stay at implement.
      blocked=$(cat "$CHANGE_DIR/blocked.md")
      rm -f "$CHANGE_DIR/blocked.md"
      if grep -qiE '^Category:[[:space:]]*base-branch' <<<"$blocked"; then
        rc=0; out=$(base_suite 2>&1) || rc=$?
        case $rc in
          0) log "implement stage: the suite passes on $BASE_BRANCH, so the reported failures come from this change"
             feedback="The previous attempt wrote blocked.md about tests failing outside this change, but the whole suite passes on \`$BASE_BRANCH\`: this change causes those failures. Fix them and do not report them as a base-branch problem again. That report:
$blocked"
             [ ! -s "$CHANGE_DIR/blocking.md" ] || feedback+="
Also fix the blocking review comments in $CHANGE_DIR/blocking.md."
             continue ;;
          1) wait_for_base_fix "$blocked" "$out" ;;
          *) die "could not prepare a worktree of $BASE_BRANCH to check the developer's report: $(tail -5 <<<"$out")" ;;
        esac
      fi
      # The fix needs API_SPECIFICATION.md, which only the contract stage may write: there once per review
      # round, then back here. A second request in the same round is a manual step for the owner.
      if grep -qiE '^Category:[[:space:]]*contract' <<<"$blocked"; then
        if [ "$(state_get contract_request_round)" != "$(state_get round)" ]; then
          state_set contract_request_round "$(state_get round)"
          printf '%s\n' "$blocked" >"$CHANGE_DIR/contract-request.md"
          advance contract "chore(factory): the fix needs an API contract change, to the contract stage"
          return
        fi
        stop_for_user needs-human manual-step "The developer needs an API contract change again in round $(state_get round), after the contract stage. Update \`API_SPECIFICATION.md\` on \`$BRANCH\`, push it, then answer here:

$blocked"
      fi
      # A step no stage may take (a dependency or lockfile update, CI): the owner does it, so neither a retry
      # nor a repair helps. The owner's answer resumes here, with the owner's commit on the branch.
      if grep -qiE '^Category:[[:space:]]*owner' <<<"$blocked"; then
        stop_for_user needs-human manual-step "The change needs a step that no factory stage may take. Do it on \`$BRANCH\`, push it, then answer here:

$blocked"
      fi
      stop_for_user blocked blocked-env "The developer cannot finish for a reason outside this change:

$blocked

Fix it on the runner and run the factory again, or let the supervisor retry: it resumes at \`implement\` with the same tests."
    fi
    if [ -s "$CHANGE_DIR/test-problem.md" ]; then
      # A locked test is wrong: back to the test stage. The test author, an independent agent, checks the
      # report against the specifications. The first time per issue without asking, then once the owner answers.
      rm -f "$CHANGE_DIR/tests.lock"
      returns=$(state_get test_problem_returns)
      if [ "${returns:-0}" -lt 1 ]; then
        state_set test_problem_returns $(( ${returns:-0} + 1 ))
        printf '%s\n\n%s\n\n_%s_\n' "The developer reports a problem in the locked tests:" "$(cat "$CHANGE_DIR/test-problem.md")" \
          "Agent factory: back to the \`tests\` stage without asking, once per issue. The test author checks the report against the specifications." | issue_comment
        advance tests "chore(factory): locked tests reported wrong, back to the tests stage"
        return
      fi
      state_set stage tests
      stop_for_user needs-input test-problem "The developer reports a problem in the locked tests, again:

$(cat "$CHANGE_DIR/test-problem.md")

The next run rewrites the tests (stage \`tests\`)."
    fi
    if out=$(gate_green 2>&1); then
      rm -f "$CHANGE_DIR/blocking.md"
      advance pr "feat: implement #$ISSUE (round $(state_get round))"
      return
    fi
    gate_log <<<"$out" | tail -60 >"$LOG_DIR/green-gate.txt"
    feedback="The previous attempt failed the green gate (locked tests unchanged + go vet, go test, npm run build, playwright). Its output:
\`\`\`
$(cat "$LOG_DIR/green-gate.txt")
\`\`\`
"
    if [ -s "$CHANGE_DIR/blocking.md" ]; then
      feedback+="Also fix the blocking review comments in $CHANGE_DIR/blocking.md."
    fi
  done
  stop_for_user needs-input gate "The implementation did not pass the tests after $MAX_ATTEMPTS attempts. Last gate output:
\`\`\`
$(cat "$LOG_DIR/green-gate.txt")
\`\`\`
If a locked test is wrong, say how it should change." '{"gate": "gate_green"}'
}

stage_pr() {
  [ "${FACTORY_PUSH:-1}" = 1 ] || { checkpoint "chore(factory): ready for pull request"; log "--no-push: stopping before the pull request"; exit 0; }
  checkpoint "chore(factory): open pull request"
  local pr title
  pr=$(gh pr list --head "$BRANCH" --state open --json number -q '.[0].number')
  if [ -z "$pr" ]; then
    title="ai-dev #$ISSUE: $(gh issue view "$ISSUE" --json title -q .title)"
    pr=$(gh pr create --base "$BASE_BRANCH" --head "$BRANCH" --title "$title" --body "$(pr_body)" | grep -oE '[0-9]+$')
  fi
  state_set pr "$pr"
  advance review "chore(factory): pull request #$pr"
}

pr_body() {
  cat <<EOF
Implements #$ISSUE through the agent factory (\`scripts/factory/factory.sh\`).

- Functional specification: $(functional_files | tr '\n' ' ')
- Use cases and test cases: \`$CHANGE_DIR/testcases.md\`
- Tests written first and locked: \`$CHANGE_DIR/tests.lock\`
- Gates passed: functional spec, use cases, FR→TC coverage, TDD red, TDD green (go vet, go test, build, Playwright)

Closes #$ISSUE

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
}

stage_review() {
  local pr round review
  pr=$(state_get pr); round=$(( $(state_get round) + 1 ))
  review="$CHANGE_DIR/review-r$round.json"
  run_agent review review "- Pull request: #$pr, head $(git rev-parse HEAD)
- Diff: git diff origin/$BASE_BRANCH...HEAD
- Write your findings to $review"
  json_from_log review "$review"
  jq -e 'type == "array"' "$review" >/dev/null 2>&1 || die "review did not write a JSON array to $review"
  post_review "$pr" "$review"
  state_set round "$round"
  advance triage "chore(factory): review round $round of #$pr"
}

post_review() { # one inline GitHub review; falls back to a plain comment when a line is outside the diff
  local pr=$1 review=$2 payload
  [ "$(jq length "$review")" -gt 0 ] || { gh pr comment "$pr" --body "Review round $(( $(state_get round) + 1 )): no findings." >/dev/null; return; }
  payload=$(jq --arg sha "$(git rev-parse HEAD)" '{commit_id: $sha, event: "COMMENT",
    body: "Agent factory review: \(length) finding(s).",
    comments: [.[] | {path: .file, line: .line, side: "RIGHT", body: "**\(.id)** [\(.category)] \(.body)"}]}' "$review")
  gh api "repos/$REPO/pulls/$pr/reviews" --method POST --input - <<<"$payload" >/dev/null 2>&1 ||
    jq -r '"Agent factory review:\n" + ([.[] | "- **\(.id)** [\(.category)] `\(.file):\(.line)` \(.body)"] | join("\n"))' "$review" |
      gh pr comment "$pr" --body-file - >/dev/null
}

stage_triage() {
  local pr round review triage series_start
  pr=$(state_get pr); round=$(state_get round)
  review="$CHANGE_DIR/review-r$round.json"; triage="$CHANGE_DIR/triage-r$round.json"
  if [ "$(jq length "$review")" -eq 0 ]; then
    echo '[]' >"$triage"
  else
    run_agent triage review "- Pull request: #$pr
- Review to triage: $review
- Write your decisions to $triage"
    json_from_log triage "$triage"
    validate_triage "$review" "$triage" || die "invalid triage in $triage (see above)"
  fi
  create_todos "$pr" "$triage"
  jq -r '"Triage of review round '"$round"':\n" + ([.[] | "- **\(.id)** \(if .blocking then "🚫 blocking" else "todo" end): \(.reason)"] | join("\n"))' "$triage" |
    gh pr comment "$pr" --body-file - >/dev/null

  if jq -e 'any(.[]; .blocking)' "$triage" >/dev/null; then
    jq -r --slurpfile r "$review" '"# Blocking review comments (round '"$round"')\n" +
      ([.[] | select(.blocking) | .id as $id | ($r[0][] | select(.id == $id)) as $c |
        "\n## \($id) `\($c.file):\($c.line)`\n\n\($c.body)\n\nTriage: \(.reason)\n" +
        (if .contract then "\nContract change needed: `API_SPECIFICATION.md`.\n" else "" end)] | join(""))' "$triage" >"$CHANGE_DIR/blocking.md"
    # `round` counts every review and never resets (review ids R<round>-NN and file names stay unique);
    # `series_start` is the round at which the current series of MAX_ROUNDS rounds began.
    series_start=$(state_get series_start); local series=$(( round - ${series_start:-0} )) extra_series
    if [ "$series" -ge "$MAX_ROUNDS" ]; then
      # A new series of rounds: implementation with blocking.md. The first extra series starts without
      # asking; after that the owner's answer starts the next one.
      state_set series_start "$round"
      extra_series=$(state_get extra_series)
      if [ "${extra_series:-0}" -ge 1 ]; then
        state_set stage implement
        stop_for_user needs-input max-rounds "After $series more review rounds, PR #$pr still has blocking comments:

$(cat "$CHANGE_DIR/blocking.md")

Answer with how to resolve them; the next run returns to implementation with $MAX_ROUNDS new review rounds."
      fi
      state_set extra_series $(( ${extra_series:-0} + 1 ))
      printf '%s\n' "After $series review rounds, PR #$pr still has blocking comments. _Agent factory: $MAX_ROUNDS more rounds start without asking, once per issue._" | issue_comment
    fi
    if jq -e 'any(.[]; .blocking and .contract == true)' "$triage" >/dev/null; then
      advance contract "chore(factory): blocking comments need an API contract change (round $round)"
    else
      advance implement "chore(factory): blocking comments return to development (round $round)"
    fi
  else
    advance merge "chore(factory): triage round $round, nothing blocks"
  fi
}

# Fails closed: exactly one decision per review id, boolean blocking (and contract, when present), safe ids,
# title/todo when not blocking.
validate_triage() { # validate_triage <review> <triage>
  jq -e --slurpfile r "$1" '
    type == "array" and
    ([.[].id] | sort) == ([$r[0][].id] | sort) and
    ([.[].id] | length) == ([.[].id] | unique | length) and
    all(.[]; (.id | type == "string" and test("^R[0-9]+-[0-9]+$")) and (.blocking | type == "boolean") and
             (.reason | type == "string") and (.contract == null or (.contract | type == "boolean")) and
             (.blocking or ((.title | type == "string" and length > 0) and (.todo | type == "string" and length > 0))))' \
    "$2" >/dev/null 2>&1 || { log "triage $2 must hold one valid decision for every id in $1"; return 1; }
}

create_todos() { # non-blocking comments become doc/todo files and GitHub issues; idempotent on resume
  local pr=$1 triage=$2 id title body file url
  for id in $(jq -r '.[] | select(.blocking | not) | .id' "$triage"); do
    [[ "$id" =~ ^R[0-9]+-[0-9]+$ ]] || die "unsafe review id: $id"
    file="doc/todo/issue-$ISSUE-$(tr '[:upper:]' '[:lower:]' <<<"$id").md"
    if [ -f "$file" ] && grep -q '^Issue: ' "$file"; then continue; fi # done by an earlier run
    title=$(jq -r --arg id "$id" '.[] | select(.id == $id) | .title' "$triage")
    body=$(jq -r --arg id "$id" '.[] | select(.id == $id) | .todo' "$triage")
    printf '# %s\n\nStatus: todo\nSource: PR #%s, %s\n\n%s\n' "$title" "$pr" "$id" "$body" >"$file"
    if url=$(gh issue create --title "$title" --body "$body

From review comment $id on #$pr (\`$file\`)." 2>&1); then
      printf '\nIssue: %s\n' "$url" >>"$file"
    else
      printf '\nIssue creation failed: %s\n' "$url" >>"$file"
      log "could not create the issue for $id: $url"
    fi
  done
}

stage_merge() {
  local pr sensitive
  pr=$(state_get pr)
  sensitive=$(sensitive_files)
  if [ -n "$sensitive" ]; then
    stop_for_user needs-human sensitive "PR #$pr is ready, but it changes files the factory never merges on its own:
\`\`\`
$sensitive
\`\`\`
Review and merge it by hand."
  fi
  # The base branch moved since the review and now conflicts with the branch: GitHub refuses the merge, and
  # retrying never helps. Back to implementation, whose sync_base merges the base branch (the merge agent
  # resolves the conflicts), then through the green gate and a new review of the result.
  local conflict=0
  git fetch -q origin "$BASE_BRANCH"
  git merge-tree --write-tree HEAD "origin/$BASE_BRANCH" >/dev/null 2>&1 || conflict=$?
  if [ "$conflict" = 1 ]; then
    log "PR #$pr conflicts with $BASE_BRANCH: back to implementation to merge it"
    advance implement "chore(factory): #$pr conflicts with $BASE_BRANCH, back to implementation"
    return
  fi
  if [ "${FACTORY_SECURITY:-1}" = 1 ]; then
    security_scan || stop_for_user needs-input security "The security scan (govulncheck, gosec) failed on PR #$pr. Log in the run output."
  fi
  [ "${FACTORY_PUSH:-1}" = 1 ] || { log "--no-push: not merging"; exit 0; }
  checkpoint "chore(factory): ready to merge #$pr"
  if [ "$(gh pr view "$pr" --json state -q .state)" != MERGED ]; then
    local out
    if ! out=$(gh pr merge "$pr" --squash --match-head-commit "$(git rev-parse HEAD)" 2>&1); then
      # stage stays `merge`: the next run retries. Typical causes: branch protection, conflict with master.
      stop_for_user needs-human merge-failed "Merging PR #$pr failed:
\`\`\`
$out
\`\`\`
Fix the cause (for example update the branch from \`$BASE_BRANCH\`) and run the factory again, or merge it by hand."
    fi
  fi
  # Merged: the branch is kept only until done is recorded on it, then removed.
  state_set stage done
  git add "$STATE"; git commit -q -m "chore(factory): #$pr merged" -m "Refs: #$ISSUE"
  git push -q origin --delete "$BRANCH" >/dev/null 2>&1 || true
  set_stage_label done
  echo "PR #$pr merged into \`$BASE_BRANCH\`. The \`release\` workflow builds it and publishes the tagged GitHub release." | issue_comment
}

# ---- main loop ----

refresh_issue_context
while :; do
  stage=$(state_get stage)
  [ "$stage" != done ] || { log "issue #$ISSUE is done"; break; }
  set_stage_label "$stage"
  "stage_$stage"
  [ "$stage" != "$UNTIL" ] || { log "stopped after --until $UNTIL"; break; }
done
