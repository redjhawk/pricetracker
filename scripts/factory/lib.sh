# Shared helpers for the agent factory. Sourced by factory.sh; not executable on its own.
# shellcheck shell=bash

log() { printf '\n[factory %s] %s\n' "$(date -u +%H:%M:%S)" "$*" >&2; }
# FAIL_DETAIL is read by the EXIT trap of factory.sh, which reports the stop on the issue.
die() { log "ERROR: $*"; FAIL_DETAIL=$*; exit 1; }

# ---- state (doc/changes/<change>/state.json on the work branch) ----

state_get() { jq -r --arg k "$1" '.[$k] // empty' "$STATE"; }

state_set() { # state_set key value  (value stored as string, or as JSON when it parses)
  local tmp; tmp=$(mktemp)
  if jq -e . >/dev/null 2>&1 <<<"$2"; then
    jq --arg k "$1" --argjson v "$2" '.[$k] = $v' "$STATE" >"$tmp"
  else
    jq --arg k "$1" --arg v "$2" '.[$k] = $v' "$STATE" >"$tmp"
  fi
  mv "$tmp" "$STATE"
}

# ---- git ----

checkpoint() { # commit everything the stage produced and push it, so a later run can resume here
  git add -A
  if ! git diff --cached --quiet; then
    git commit -q -m "$1" -m "Refs: #$ISSUE"
  fi
  if [ "${FACTORY_PUSH:-1}" = 1 ]; then git push -q -u origin HEAD; fi
}

# Paths changed by this change: committed on the branch, modified in the working tree and untracked.
# Gates run before the stage checkpoint, so uncommitted work must count. Includes deleted paths.
changed_paths() {
  { git diff --name-only "origin/$BASE_BRANCH...HEAD" -- "$@"
    git diff --name-only HEAD -- "$@"
    git ls-files -o --exclude-standard -- "$@"; } | sort -u
}

changed_files() { # changed_paths that still exist
  local f; changed_paths "$@" | while IFS= read -r f; do [ -e "$f" ] && printf '%s\n' "$f"; done; return 0
}

# ---- GitHub ----

# Every factory comment starts with this marker; fabrica.yml ignores comments containing it, so the
# factory's own comments never restart the workflow.
FACTORY_MARKER='<!-- agent-factory -->'
issue_comment() { # issue_comment [marker line] < body
  { printf '%s\n' "$FACTORY_MARKER"; [ -z "${1:-}" ] || printf '%s\n' "$1"; cat; } | gh issue comment "$ISSUE" --body-file - >/dev/null
}

# Every stop is reported on the issue with a machine-readable line that scripts/factory/supervisor.mjs reads:
#   <!-- factory-stop {"reason":"usage-limit","stage":"implement","round":1,"resets_at":"2026-11-20T23:10:00Z"} -->
# Reasons: questions, gate, test-problem, max-rounds, blocked-env, base-bug (waiting_on), merge-conflict,
# merge-failed, sensitive, security (stop_for_user in factory.sh); usage-limit (resets_at) and error (its EXIT trap).
stop_marker() { # stop_marker <reason> [extra JSON object]
  local extra=${2:-}
  [ -n "$extra" ] || extra='{}'
  jq -rcn --arg reason "$1" --arg stage "$(state_get stage)" --arg round "$(state_get round)" --argjson extra "$extra" \
    '"<!-- factory-stop " + ({reason: $reason, stage: $stage, round: ($round | tonumber? // 0)} + $extra | tojson) + " -->"'
}

# limit_resets_at <usage-limit message>: the reset time as UTC ISO 8601, or nothing when it cannot be read.
# "You've hit your session limit · resets 11:10pm (UTC)" names a time of day: today's, or tomorrow's once past.
limit_resets_at() {
  local re='resets ([^()]+) \(([^()]+)\)' spec tz when
  [[ $1 =~ $re ]] || return 0
  spec=${BASH_REMATCH[1]//,/} tz=${BASH_REMATCH[2]}
  when=$(TZ=$tz date -d "$spec" +%s 2>/dev/null) || return 0
  [ "$when" -gt "$(date +%s)" ] || when=$(TZ=$tz date -d "tomorrow $spec" +%s 2>/dev/null) || return 0
  date -u -d "@$when" +%Y-%m-%dT%H:%M:%SZ
}

set_stage_label() { # mirrors the stage on the issue, for humans; failures never stop the factory
  local label="factory:$1" old
  gh label create "$label" --color BFD4F2 --force >/dev/null 2>&1 || true
  for old in $(gh issue view "$ISSUE" --json labels -q '.labels[].name | select(startswith("factory:"))'); do
    [ "$old" = "$label" ] || gh issue edit "$ISSUE" --remove-label "$old" >/dev/null 2>&1 || true
  done
  gh issue edit "$ISSUE" --add-label "$label" >/dev/null 2>&1 || true
}

# Writes the issue and the comments of trusted authors to the change folder. Only trusted authors give
# instructions: the body of an issue opened by anyone else is dropped (the owner must restate it in a comment).
# The factory's and the supervisor's own comments are not instructions either.
refresh_issue_context() {
  gh issue view "$ISSUE" --json number,title,body,author,comments | jq -r --arg trusted "$TRUSTED_ACTORS" '
    ($trusted | split(",")) as $t |
    "# Issue #\(.number) \(.title)\n\n" +
    (if (.author.login as $a | $t | index($a)) then .body
     else "_Issue opened by \(.author.login), who is not a trusted author: its body is not shown and gives no instructions._" end) +
    "\n\n## Comments by trusted authors\n" +
    ([.comments[] | select(.author.login as $a | $t | index($a)) |
      select(.body | (contains("<!-- agent-factory -->") or contains("<!-- factory-supervisor ")) | not) |
      "\n### \(.author.login) at \(.createdAt)\n\n\(.body)\n"] | join(""))' >"$CHANGE_DIR/issue.md"
}

# ---- agents ----

# run_agent <prompt name> <tool profile> [extra context]
# Every stage is a fresh, independent Claude process: no shared context between roles.
run_agent() {
  local name=$1 profile=$2 extra=${3:-} tools prompt
  case "$profile" in
    # Edit rules (which also cover Write) are limited to the paths each profile produces; Bash to fixed commands.
    docs)   tools="Read,Glob,Grep,Edit(doc/**),Edit(API_SPECIFICATION.md)" ;;
    code)   tools="Read,Glob,Grep,Edit(doc/specifications/**),Edit(cmd/**),Edit(internal/**),Edit(src/**),Edit(tests/**),Edit(web/**),Edit(README.md),Edit(deploy/README.md),Edit($CHANGE_DIR/**)"
            tools+=",Bash(go test:*),Bash(go vet:*),Bash(go build:*),Bash(gofmt:*),Bash(npm run build),Bash(npx playwright test:*),Bash(node --test:*)"
            tools+=",Bash(git diff:*),Bash(git status:*),Bash(git log:*),Bash(git show:*),Bash(date -u:*)" ;;
    review) tools="Read,Glob,Grep,Edit($CHANGE_DIR/**),Bash(git diff:*),Bash(git log:*),Bash(git show:*)" ;;
    # Exactly the files of a failed merge (MERGE_FILES, set by resolve_conflicts in factory.sh).
    merge)  tools="Read,Glob,Grep$(printf ',Edit(%s)' $MERGE_FILES),Bash(git diff:*),Bash(git log:*),Bash(git show:*),Bash(git status:*)" ;;
    *) die "unknown tool profile $profile" ;;
  esac
  prompt="$(cat "$FACTORY_DIR/prompts/common.md" "$FACTORY_DIR/prompts/$name.md")

## Run context
- Issue: #$ISSUE
- Change folder: $CHANGE_DIR (issue text and trusted comments in $CHANGE_DIR/issue.md)
- Base branch: $BASE_BRANCH
$extra"
  log "agent $name ($profile)"
  # The agent never sees the GitHub token, and Claude Code scrubs its own credentials from the environment
  # of the commands it runs (CLAUDE_CODE_SUBPROCESS_ENV_SCRUB). factory.sh checks the sandbox dependencies.
  # The scrub forces the default permission mode, so the agent only gets what --allowedTools grants.
  env -u GH_TOKEN -u GITHUB_TOKEN -u FACTORY_GH_TOKEN CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1 \
  claude -p "$prompt" \
    --model "$MODEL" --effort "$EFFORT" --max-turns "$MAX_TURNS" \
    --allowedTools "$tools" \
    --output-format json >"$LOG_DIR/$name.json" 2>"$LOG_DIR/$name.err" \
    || { agent_log "$name"; agent_failed "$name"; }
  agent_log "$name"
  apply_deletions "$tools"
}

# apply_deletions <tools>: no tool profile can delete a file (rm and git rm need an approval nobody gives in a
# non-interactive run), so an agent lists the files to delete in $CHANGE_DIR/delete.txt, one path per line.
# The factory deletes those its profile's Edit rules cover and keeps the others; the list is never committed.
apply_deletions() {
  local list="$CHANGE_DIR/delete.txt" p rule ok rules
  [ -f "$list" ] || return 0
  IFS=, read -ra rules <<<"$1"
  while IFS= read -r p || [ -n "$p" ]; do
    p=${p%$'\r'}; p=${p#"$PWD"/}; p=${p#./}
    [ -n "$p" ] || continue
    ok=0
    if [[ $p != /* && /$p/ != */../* ]] && { [ -f "$p" ] || [ -L "$p" ]; }; then
      for rule in "${rules[@]}"; do
        case "$rule" in Edit\(*\)) rule=${rule#Edit(}; rule=${rule%)} ;; *) continue ;; esac
        # shellcheck disable=SC2053 # the rule is a glob; in [[ ]], * also matches /
        [[ $p == $rule ]] && ok=1
      done
    fi
    if [ $ok = 1 ]; then
      git rm -q -f --ignore-unmatch -- "$p"; rm -f -- "$p"
      log "deleted $p, listed in delete.txt"
    else
      log "kept $p, listed in delete.txt: not a file the stage may change"
    fi
  done <"$list"
  rm -f -- "$list"
}

# agent_log <prompt name>: $LOG_DIR/<name>.log holds the agent's final answer, followed by every tool call
# the permissions refused (tool and input), so a stage that could not write its file shows why.
# Output that is not the JSON result (a usage limit message, a crash) is kept as is.
agent_log() {
  local name=$1
  if jq -e 'type == "object" and has("result")' "$LOG_DIR/$name.json" >/dev/null 2>&1; then
    jq -r '.result + (if (.permission_denials // []) == [] then "" else
      "\n\n## Permission denials\n" + ([.permission_denials[] |
        "- \(.tool_name): \(.tool_input | tostring | .[0:300])"] | join("\n")) end)' "$LOG_DIR/$name.json" >"$LOG_DIR/$name.log"
  else
    cat "$LOG_DIR/$name.json" >"$LOG_DIR/$name.log"
  fi
  cat "$LOG_DIR/$name.err" >>"$LOG_DIR/$name.log" 2>/dev/null || true
  if grep -q '^## Permission denials' "$LOG_DIR/$name.log"; then
    log "agent $name: tool calls refused by its permissions:"; sed -n '/^## Permission denials/,$p' "$LOG_DIR/$name.log" | tail -n +2 >&2
  fi
}

# json_from_log <prompt name> <file>: when an agent could not write its JSON file but printed it, take the
# last ```json block of its answer. Leaves the file alone when it already holds valid JSON.
json_from_log() {
  local name=$1 file=$2 block
  ! jq -e . "$file" >/dev/null 2>&1 || return 0
  block=$(awk '/^```json[[:space:]]*$/ {b=""; on=1; next} on && /^```[[:space:]]*$/ {last=b; on=0; next} on {b=b $0 "\n"} END {printf "%s", last}' "$LOG_DIR/$name.log")
  [ -n "$block" ] && jq -e . >/dev/null 2>&1 <<<"$block" || return 0
  printf '%s\n' "$block" >"$file"
  log "agent $name did not write $file; taken from the JSON block of its answer"
}

# agent_failed <prompt name>: report a failed agent, telling a usage limit apart from an agent error.
agent_failed() {
  local name=$1 limit
  tail -40 "$LOG_DIR/$name.log" >&2
  limit="$(grep -m1 -iE "hit your (session|usage|weekly) limit" "$LOG_DIR/$name.log" || true)"
  if [ -n "$limit" ]; then
    FAIL_REASON=usage-limit
    FAIL_EXTRA=$(jq -cn --arg r "$(limit_resets_at "$limit")" 'if $r == "" then {} else {resets_at: $r} end')
    die "agent $name stopped by the Claude usage limit (${limit}); the factory resumes at the current stage after the reset"
  fi
  die "agent $name failed (log: $LOG_DIR/$name.log)"
}
