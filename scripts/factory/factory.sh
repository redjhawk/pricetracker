#!/usr/bin/env bash
# Agent factory: takes a GitHub issue from specification to a merged pull request.
#
#   scripts/factory/factory.sh <issue> [--no-push] [--until <stage>]
#
# Stages (state in doc/changes/<change>/state.json on branch ai-dev/issue-<n>):
#   spec → usecases → testcases → techspec → tests (TDD red) → implement (green)
#   → pr → review → triage ─ blocking ─► implement (max FACTORY_MAX_ROUNDS)
#                         └ none ─────► merge → done   (push to master runs .github/workflows/release.yml)
#
# Each agent stage is a fresh `claude -p` process with its own prompt and tools; deterministic gates
# (gates.sh) decide whether it may advance. Every stage is committed and pushed, so a later run resumes
# where the previous one stopped. The run stops, and asks on the issue, when it needs a functional answer.
# Runs the same locally and in .github/workflows/fabrica.yml.
set -euo pipefail

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

MODEL=${FACTORY_MODEL:-claude-opus-5-5}
EFFORT=${FACTORY_EFFORT:-high}
MAX_TURNS=${FACTORY_MAX_TURNS:-150}
MAX_ROUNDS=${FACTORY_MAX_ROUNDS:-3}     # review → fix rounds before asking the user
MAX_ATTEMPTS=${FACTORY_MAX_ATTEMPTS:-3} # implementation attempts against the green gate per round
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
  jq -n --argjson i "$ISSUE" '{issue: $i, stage: "spec", round: 0, pr: ""}' >"$CHANGE_DIR/state.json"
fi
STATE="$CHANGE_DIR/state.json"
export ISSUE CHANGE_DIR BASE_BRANCH LOG_DIR
log "issue #$ISSUE, branch $BRANCH, change $CHANGE_DIR, stage $(state_get stage), logs $LOG_DIR"

# ---- stage transitions ----

advance() { # advance <next stage> <commit message>
  state_set stage "$1"
  checkpoint "$2"
  set_stage_label "$1"
}

stop_for_user() { # stop_for_user <label> <message for the issue>
  printf '%s\n\n_Agent factory, stage `%s`. Answer in a comment and run the factory again._\n' "$2" "$(state_get stage)" | issue_comment
  checkpoint "chore(factory): stop at $(state_get stage) for user input"
  set_stage_label "$1"
  log "stopped: $1"
  exit 0
}

gate_or_die() { # gate_or_die <gate function>
  local out
  if ! out=$("$1" 2>&1); then
    printf '%s\n' "$out" >&2
    stop_for_user needs-input "The \`$1\` gate failed after the agent finished:
\`\`\`
$(tail -30 <<<"$out")
\`\`\`"
  fi
}

# ---- stages ----

stage_spec() {
  run_agent spec docs
  if [ -s "$CHANGE_DIR/questions.md" ]; then
    stop_for_user needs-input "## Functional questions

$(cat "$CHANGE_DIR/questions.md")"
  fi
  gate_or_die gate_spec
  advance usecases "docs(spec): functional specification for #$ISSUE"
}

stage_usecases() {
  run_agent usecases docs
  gate_or_die gate_usecases
  advance testcases "docs(use-cases): use cases for #$ISSUE"
}

stage_testcases() {
  run_agent testcases docs
  gate_or_die gate_testcases
  advance techspec "docs(test-cases): test cases for #$ISSUE"
}

stage_techspec() {
  run_agent techspec docs
  gate_or_die gate_techspec
  advance tests "docs(tech): technical specification and API contract for #$ISSUE"
}

stage_tests() {
  run_agent tests code
  gate_or_die gate_red
  advance implement "test: failing tests for #$ISSUE (TDD red)"
}

stage_implement() {
  local attempt feedback="" out
  [ ! -s "$CHANGE_DIR/blocking.md" ] || feedback="Fix the blocking review comments in $CHANGE_DIR/blocking.md."
  for attempt in $(seq 1 "$MAX_ATTEMPTS"); do
    run_agent implement code "- Attempt $attempt of $MAX_ATTEMPTS
$feedback"
    if [ -s "$CHANGE_DIR/test-problem.md" ]; then
      # A locked test is wrong: back to the test stage once the user has answered
      rm -f "$CHANGE_DIR/tests.lock"
      state_set stage tests
      stop_for_user needs-input "The developer reports a problem in the locked tests:

$(cat "$CHANGE_DIR/test-problem.md")

The next run rewrites the tests (stage \`tests\`)."
    fi
    if out=$(gate_green 2>&1); then
      rm -f "$CHANGE_DIR/blocking.md"
      advance pr "feat: implement #$ISSUE (round $(state_get round))"
      return
    fi
    printf '%s\n' "$out" | tail -60 >"$LOG_DIR/green-gate.txt"
    feedback="The previous attempt failed the green gate (locked tests unchanged + go vet, go test, npm run build, playwright). Its output:
\`\`\`
$(cat "$LOG_DIR/green-gate.txt")
\`\`\`
"
    if [ -s "$CHANGE_DIR/blocking.md" ]; then
      feedback+="Also fix the blocking review comments in $CHANGE_DIR/blocking.md."
    fi
  done
  stop_for_user needs-input "The implementation did not pass the tests after $MAX_ATTEMPTS attempts. Last gate output:
\`\`\`
$(cat "$LOG_DIR/green-gate.txt")
\`\`\`
If a locked test is wrong, say how it should change."
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
  local pr round review triage
  pr=$(state_get pr); round=$(state_get round)
  review="$CHANGE_DIR/review-r$round.json"; triage="$CHANGE_DIR/triage-r$round.json"
  if [ "$(jq length "$review")" -eq 0 ]; then
    echo '[]' >"$triage"
  else
    run_agent triage review "- Pull request: #$pr
- Review to triage: $review
- Write your decisions to $triage"
    validate_triage "$review" "$triage" || die "invalid triage in $triage (see above)"
  fi
  create_todos "$pr" "$triage"
  jq -r '"Triage of review round '"$round"':\n" + ([.[] | "- **\(.id)** \(if .blocking then "🚫 blocking" else "todo" end): \(.reason)"] | join("\n"))' "$triage" |
    gh pr comment "$pr" --body-file - >/dev/null

  if jq -e 'any(.[]; .blocking)' "$triage" >/dev/null; then
    jq -r --slurpfile r "$review" '"# Blocking review comments (round '"$round"')\n" +
      ([.[] | select(.blocking) | .id as $id | ($r[0][] | select(.id == $id)) as $c |
        "\n## \($id) `\($c.file):\($c.line)`\n\n\($c.body)\n\nTriage: \(.reason)\n"] | join(""))' "$triage" >"$CHANGE_DIR/blocking.md"
    if [ "$round" -ge "$MAX_ROUNDS" ]; then
      # The user's answer resumes a new series of rounds: implementation with blocking.md and their comment.
      state_set round 0
      state_set stage implement
      stop_for_user needs-input "After $round review rounds, PR #$pr still has blocking comments:

$(cat "$CHANGE_DIR/blocking.md")

Answer with how to resolve them; the next run returns to implementation with $MAX_ROUNDS new review rounds."
    fi
    advance implement "chore(factory): blocking comments return to development (round $round)"
  else
    advance merge "chore(factory): triage round $round, nothing blocks"
  fi
}

# Fails closed: exactly one decision per review id, boolean blocking, safe ids, title/todo when not blocking.
validate_triage() { # validate_triage <review> <triage>
  jq -e --slurpfile r "$1" '
    type == "array" and
    ([.[].id] | sort) == ([$r[0][].id] | sort) and
    ([.[].id] | length) == ([.[].id] | unique | length) and
    all(.[]; (.id | type == "string" and test("^R[0-9]+-[0-9]+$")) and (.blocking | type == "boolean") and
             (.reason | type == "string") and
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
    stop_for_user needs-human "PR #$pr is ready, but it changes files the factory never merges on its own:
\`\`\`
$sensitive
\`\`\`
Review and merge it by hand."
  fi
  if [ "${FACTORY_SECURITY:-1}" = 1 ]; then
    security_scan || stop_for_user needs-input "The security scan (govulncheck, gosec) failed on PR #$pr. Log in the run output."
  fi
  [ "${FACTORY_PUSH:-1}" = 1 ] || { log "--no-push: not merging"; exit 0; }
  checkpoint "chore(factory): ready to merge #$pr"
  if [ "$(gh pr view "$pr" --json state -q .state)" != MERGED ]; then
    local out
    if ! out=$(gh pr merge "$pr" --squash --match-head-commit "$(git rev-parse HEAD)" 2>&1); then
      # stage stays `merge`: the next run retries. Typical causes: branch protection, conflict with master.
      stop_for_user needs-human "Merging PR #$pr failed:
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
