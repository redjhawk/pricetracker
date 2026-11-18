# Deterministic gates between stages. An agent saying "done" is not enough: these checks decide.
# Each gate prints why it failed and returns non-zero. Sourced by factory.sh.
# shellcheck shell=bash

TEST_FILE_PATTERN='(_test\.go|\.spec\.ts|\.test\.mjs)$|^playwright\.config\.ts$'

functional_files() { changed_files 'doc/specifications/*/functional.md'; }

requirement_ids() { # FR IDs defined or changed in this change's functional specs
  local f; for f in $(functional_files); do grep -oE 'FR-[A-Z0-9-]+-[0-9]{3}' "$f"; done | sort -u
}

test_files() { changed_files | grep -E "$TEST_FILE_PATTERN" || true; }

run_unit_tests() { go vet ./... && go test ./... && node --test scripts/*.test.mjs; }
run_all_tests() { run_unit_tests && npm run build && npx playwright test; }

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
  if run_all_tests; then echo "the tests pass before implementing: they do not test the new behaviour"; return 1; fi
  # shellcheck disable=SC2086
  sha256sum $files >"$CHANGE_DIR/tests.lock"
  echo "red confirmed; tests locked in tests.lock"
}

# TDD green: the locked tests are untouched and the whole suite, build and e2e pass.
gate_green() {
  if [ -s "$CHANGE_DIR/tests.lock" ]; then
    sha256sum --quiet -c "$CHANGE_DIR/tests.lock" || { echo "locked tests were modified by the implementation"; return 1; }
  fi
  run_all_tests
}

# Files the factory never merges on its own: CI, the factory itself, dependencies.
sensitive_files() { changed_files | grep -E '^(\.github/|scripts/factory/|CODEOWNERS$|go\.(mod|sum)$|package(-lock)?\.json$)' || true; }

security_scan() {
  govulncheck ./... && gosec -quiet ./...
}
