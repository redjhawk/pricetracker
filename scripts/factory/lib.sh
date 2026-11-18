# Shared helpers for the agent factory. Sourced by factory.sh; not executable on its own.
# shellcheck shell=bash

log() { printf '\n[factory %s] %s\n' "$(date -u +%H:%M:%S)" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

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

changed_files() { git diff --name-only "origin/$BASE_BRANCH...HEAD" -- "$@"; }

# ---- GitHub ----

issue_comment() { gh issue comment "$ISSUE" --body-file - >/dev/null; }

set_stage_label() { # mirrors the stage on the issue, for humans; failures never stop the factory
  local label="factory:$1" old
  gh label create "$label" --color BFD4F2 --force >/dev/null 2>&1 || true
  for old in $(gh issue view "$ISSUE" --json labels -q '.labels[].name | select(startswith("factory:"))'); do
    [ "$old" = "$label" ] || gh issue edit "$ISSUE" --remove-label "$old" >/dev/null 2>&1 || true
  done
  gh issue edit "$ISSUE" --add-label "$label" >/dev/null 2>&1 || true
}

# Writes the issue and the comments of trusted authors to the change folder. Only these authors give instructions.
refresh_issue_context() {
  gh issue view "$ISSUE" --json title,body,comments | jq -r --arg trusted "$TRUSTED_ACTORS" '
    ($trusted | split(",")) as $t |
    "# Issue #\(.number // "") \(.title)\n\n\(.body)\n\n## Comments by trusted authors\n" +
    ([.comments[] | select(.author.login as $a | $t | index($a)) |
      "\n### \(.author.login) at \(.createdAt)\n\n\(.body)\n"] | join(""))' >"$CHANGE_DIR/issue.md"
}

# ---- agents ----

# run_agent <prompt name> <tool profile> [extra context]
# Every stage is a fresh, independent Claude process: no shared context between roles.
run_agent() {
  local name=$1 profile=$2 extra=${3:-} tools prompt
  case "$profile" in
    docs)   tools="Read,Glob,Grep,Edit,Write" ;;
    code)   tools="Read,Glob,Grep,Edit,Write,Bash(go:*),Bash(gofmt:*),Bash(npm:*),Bash(npx:*),Bash(node:*),Bash(git diff:*),Bash(git status:*),Bash(git log:*),Bash(git show:*),Bash(date -u:*)" ;;
    review) tools="Read,Glob,Grep,Edit($CHANGE_DIR/**),Bash(git diff:*),Bash(git log:*),Bash(git show:*)" ;;
    *) die "unknown tool profile $profile" ;;
  esac
  prompt="$(cat "$FACTORY_DIR/prompts/common.md" "$FACTORY_DIR/prompts/$name.md")

## Run context
- Issue: #$ISSUE
- Change folder: $CHANGE_DIR (issue text and trusted comments in $CHANGE_DIR/issue.md)
- Base branch: $BASE_BRANCH
$extra"
  log "agent $name ($profile)"
  claude -p "$prompt" \
    --model "$MODEL" --effort "$EFFORT" --max-turns "$MAX_TURNS" \
    --permission-mode acceptEdits --allowedTools "$tools" \
    --output-format text >"$LOG_DIR/$name.log" 2>&1 \
    || { tail -40 "$LOG_DIR/$name.log" >&2; die "agent $name failed (log: $LOG_DIR/$name.log)"; }
}
