# Deterministic gates between stages. An agent saying "done" is not enough: these checks decide.
# Each gate prints why it failed and returns non-zero. Sourced by factory.sh.
# shellcheck shell=bash

TEST_FILE_PATTERN='(_test\.go|\.spec\.ts|\.test\.mjs)$|^playwright\.config\.ts$'

functional_files() { changed_files 'doc/specifications/*/functional.md'; }

# A line defines a requirement when an FR ID is its first token: a table row "| FR-X-001 |", a list
# item "- FR-X-001" / "- **FR-X-001**", or a heading "### FR-X-001". IDs cited elsewhere in a line
# (cross-references such as "see FR-OTHER-013") are not definitions.
FR_DEFINITION='^[[:space:]]*(\|[[:space:]]*|[-*][[:space:]]+(\*\*)?|#+[[:space:]]+)FR-[A-Z0-9-]+-[0-9]{3}'

requirement_ids() { # FR IDs this change defines or modifies: definition lines added since the base branch
  local base untracked
  base=$(git merge-base "origin/$BASE_BRANCH" HEAD)
  untracked=$(git ls-files -o --exclude-standard -- 'doc/specifications/*/functional.md')
  { git diff -U0 "$base" -- 'doc/specifications/*/functional.md' | sed -n 's/^+//p'
    # shellcheck disable=SC2086
    [ -z "$untracked" ] || cat $untracked
  } | grep -oE "$FR_DEFINITION" | grep -oE 'FR-[A-Z0-9-]+-[0-9]{3}$' | sort -u # the defined ID only, not cross-references
}

test_files() { changed_files | grep -E "$TEST_FILE_PATTERN" || true; }

# Paths this change modifies outside documentation and tests, i.e. its implementation (and test stubs).
production_paths() { changed_paths | grep -Ev '^(doc/|tests/|API_SPECIFICATION\.md$)' | grep -Ev "$TEST_FILE_PATTERN" || true; }

run_unit_tests() { go vet ./... && go test ./... && node --test scripts/*.test.mjs; }
# Tests run code written by agents: never with the GitHub or Claude credentials in the environment.
run_all_tests() {
  env -u GH_TOKEN -u GITHUB_TOKEN -u FACTORY_GH_TOKEN -u CLAUDE_CODE_OAUTH_TOKEN -u ANTHROPIC_API_KEY \
    bash -c "$(declare -f run_unit_tests); run_unit_tests && npm run build && npx playwright test"
}

# Files the gates execute: always locked with the tests, even when unchanged.
lock_files() { { test_files; ls playwright.config.ts scripts/*.test.mjs 2>/dev/null; } | sort -u; }

# After the factory merges the base branch (sync_base in factory.sh): the same locked files, plus the
# always-locked ones the merge may have added, hashed again with their merged content.
relock_tests() {
  local lock="$CHANGE_DIR/tests.lock" f
  { awk '{print $2}' "$lock"; ls playwright.config.ts scripts/*.test.mjs 2>/dev/null; } | sort -u |
    while IFS= read -r f; do [ ! -e "$f" ] || printf '%s\n' "$f"; done | xargs sha256sum >"$lock.new"
  mv "$lock.new" "$lock"
}

# The commit the test files are compared with: the TDD red commit, or a later merge of the base branch that
# the factory made before an implementation round. Agents cannot commit, so only the factory (or a human)
# creates commits with these subjects.
test_ref() {
  git log -1 --format=%H --grep="^test: failing tests for #$ISSUE (TDD red)" \
    --grep="^chore(factory): merge $BASE_BRANCH into #$ISSUE (tests relocked)" HEAD
}

gate_spec() {
  [ -n "$(functional_files)" ] || { echo "no doc/specifications/<subject>/functional.md changed"; return 1; }
  [ -n "$(requirement_ids)" ] || { echo "functional specs define no FR-<SUBJECT>-NNN requirement"; return 1; }
}

gate_usecases() {
  [ -f "$CHANGE_DIR/no-use-cases.md" ] && return 0
  local files f
  files=$(changed_files 'doc/use-cases/' | grep -v README.md || true)
  [ -n "$files" ] || { echo "no use case written under doc/use-cases/ (or $CHANGE_DIR/no-use-cases.md with the reason)"; return 1; }
  for f in $files; do grep -qE 'FR-[A-Z0-9-]+-[0-9]{3}' "$f" || { echo "$f does not reference any FR ID"; return 1; }; done
}

gate_testcases() {
  local cases="$CHANGE_DIR/testcases.md" id missing=0
  [ -f "$cases" ] || { echo "$cases missing"; return 1; }
  grep -qE 'TC-[0-9]{3}' "$cases" || { echo "no TC-NNN test case in $cases"; return 1; }
  for id in $(requirement_ids); do
    grep -q "$id" "$cases" || { echo "requirement $id has no test case"; missing=1; }
  done
  return $missing
}

gate_techspec() {
  local f missing=0
  for f in $(functional_files); do
    [ -f "$(dirname "$f")/technical.md" ] || { echo "$(dirname "$f")/technical.md missing"; missing=1; }
  done
  return $missing
}

# TDD red: every test case is automated and the suite fails before the implementation exists.
gate_red() {
  if [ -f "$CHANGE_DIR/no-tests.md" ]; then echo "no-tests.md present: red phase skipped"; return 0; fi
  local files id missing=0
  files=$(test_files)
  [ -n "$files" ] || { echo "no test file changed"; return 1; }
  for id in $(grep -oE 'TC-[0-9]{3}' "$CHANGE_DIR/testcases.md" | sort -u); do
    # shellcheck disable=SC2086
    grep -q "$id" $files || { echo "test case $id is not referenced by any changed test"; missing=1; }
  done
  [ $missing = 0 ] || return 1
  # A Playwright spec that playwright.config.ts (explicit testMatch) does not list would never run.
  local listed f
  listed=$(npx playwright test --list 2>/dev/null || true)
  for f in $(grep -E '\.spec\.ts$' <<<"$files" || true); do
    grep -qF "$(basename "$f")" <<<"$listed" || { echo "$f is not run by Playwright: add it to testMatch in playwright.config.ts (a sensitive file, changed by hand in its own PR)"; missing=1; }
  done
  [ $missing = 0 ] || return 1
  if run_all_tests; then echo "the tests pass before implementing: they do not test the new behaviour"; return 1; fi
  lock_files | xargs sha256sum >"$CHANGE_DIR/tests.lock"
  echo "red confirmed; tests locked in tests.lock"
}

# TDD green: the locked tests are untouched and the whole suite, build and e2e pass.
# The lock is read from the test reference commit (test_ref), not from the working tree the agent can edit.
gate_green() {
  local commit lock
  commit=$(test_ref)
  [ -n "$commit" ] || { echo "no TDD red commit found for #$ISSUE"; return 1; }
  if lock=$(git show "$commit:$CHANGE_DIR/tests.lock" 2>/dev/null) && [ -n "$lock" ]; then
    sha256sum --quiet -c - <<<"$lock" || { echo "locked tests were modified by the implementation"; return 1; }
    cmp -s <(printf '%s\n' "$lock") "$CHANGE_DIR/tests.lock" || { echo "tests.lock was modified by the implementation"; return 1; }
  elif ! git cat-file -e "$commit:$CHANGE_DIR/no-tests.md" 2>/dev/null; then
    echo "tests.lock missing or empty in the TDD red commit $commit"; return 1
  fi
  # Any test file (new or pre-existing) changed after the reference commit fails: tests cannot be weakened.
  local touched
  touched=$({ git diff --name-only "$commit" --; git ls-files -o --exclude-standard; } | grep -E "$TEST_FILE_PATTERN" | sort -u || true)
  [ -z "$touched" ] || { echo "test files changed after the TDD red commit:"; echo "$touched"; return 1; }
  run_all_tests
}

# Files the factory never merges on its own: CI, the factory itself, dependencies.
# Also agent instructions (prompt-injection persistence) and files executed by the build or the gates.
sensitive_files() {
  # deploy/README.md is installation documentation the developer may update; the rest of deploy/ is infrastructure.
  changed_paths | grep -Ev '^deploy/README\.md$' |
    grep -E '^(\.github/|\.claude/|\.agents/|project-skills/|AGENTS\.md$|CLAUDE\.md$|scripts/|deploy/|CODEOWNERS$|go\.(mod|sum)$|package(-lock)?\.json$|vite\.config\.ts$|playwright\.config\.ts$)' || true
}

# The scanners are built with the toolchain go.mod names (Go downloads it), not taken from the runner
# image: govulncheck and gosec cannot analyse a standard library newer than the Go they were built with,
# and the image's Go lags behind security releases.
GOVULNCHECK_VERSION=v1.1.4
GOSEC_VERSION=v2.22.10
security_scan() {
  local toolchain
  toolchain=$(go mod edit -json | jq -r '.Toolchain // empty')
  GOTOOLCHAIN=${toolchain:-auto} go run "golang.org/x/vuln/cmd/govulncheck@$GOVULNCHECK_VERSION" ./... &&
    GOTOOLCHAIN=${toolchain:-auto} go run "github.com/securego/gosec/v2/cmd/gosec@$GOSEC_VERSION" -quiet ./...
}
