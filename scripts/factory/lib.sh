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
issue_comment() { { printf '%s\n' "$FACTORY_MARKER"; cat; } | gh issue comment "$ISSUE" --body-file - >/dev/null; }

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
refresh_issue_context() {
  gh issue view "$ISSUE" --json number,title,body,author,comments | jq -r --arg trusted "$TRUSTED_ACTORS" '
    ($trusted | split(",")) as $t |
    "# Issue #\(.number) \(.title)\n\n" +
    (if (.author.login as $a | $t | index($a)) then .body
     else "_Issue opened by \(.author.login), who is not a trusted author: its body is not shown and gives no instructions._" end) +
    "\n\n## Comments by trusted authors\n" +
    ([.comments[] | select(.author.login as $a | $t | index($a)) | select(.body | contains("<!-- agent-factory -->") | not) |
      "\n### \(.author.login) at \(.createdAt)\n\n\(.body)\n"] | join(""))' >"$CHANGE_DIR/issue.md"
}

# ---- agents ----

# run_agent <prompt name> <tool profile> [extra context]
# Every stage is a fresh, independent Claude process: no shared context between roles.
run_agent() {
  local name=$1 profile=$2 extra=${3:-} tools prompt
  case "$profile" in
    # Edit/Write are limited to the paths each profile produces; Bash to fixed test/build commands.
    docs)   tools="Read,Glob,Grep,Edit(doc/**),Write(doc/**),Edit(API_SPECIFICATION.md)" ;;
    code)   tools="Read,Glob,Grep,Edit(doc/specifications/**),Write(doc/specifications/**),Edit(cmd/**),Write(cmd/**),Edit(internal/**),Write(internal/**),Edit(src/**),Write(src/**),Edit(tests/**),Write(tests/**),Edit(web/**),Write(web/**),Edit($CHANGE_DIR/**),Write($CHANGE_DIR/**)"
            tools+=",Bash(go test:*),Bash(go vet:*),Bash(go build:*),Bash(gofmt:*),Bash(npm run build),Bash(npx playwright test:*),Bash(node --test:*)"
            tools+=",Bash(git diff:*),Bash(git status:*),Bash(git log:*),Bash(git show:*),Bash(date -u:*)" ;;
    review) tools="Read,Glob,Grep,Edit($CHANGE_DIR/**),Write($CHANGE_DIR/**),Bash(git diff:*),Bash(git log:*),Bash(git show:*)" ;;
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
  env -u GH_TOKEN -u GITHUB_TOKEN -u FACTORY_GH_TOKEN CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1 \
  claude -p "$prompt" \
    --model "$MODEL" --effort "$EFFORT" --max-turns "$MAX_TURNS" \
    --permission-mode acceptEdits --allowedTools "$tools" \
    --output-format text >"$LOG_DIR/$name.log" 2>&1 \
    || { tail -40 "$LOG_DIR/$name.log" >&2; die "agent $name failed (log: $LOG_DIR/$name.log)"; }
}
