# Blocking review comments (round 1)

## R1-01 `internal/service/amazon_browser_retrieval_test.go:221`

Unrelated formatting change in an existing #204 test: the one-line B0CAUSEG01 func literal is wrapped over three lines. gofmt does not require it (the gate only runs `gofmt -l`), no test case of #226 touches TC-007, and the technical specification's 'Files to change' table does not list this file (it says the existing #204 tests stay unchanged). Revert this hunk so that the PR only holds the #226 documentation sentence and its tests; any reformatting belongs in its own commit or PR.

Triage: Commit 623f2cf wraps the one-line B0CAUSEG01 func literal in internal/service/amazon_browser_retrieval_test.go:221 over three lines; the original form passes gofmt, and gates.sh has no line-length rule, so the hunk is not needed. It is a formatting change to a #204 test that no #226 test case touches, which counts as unrelated work mixed into this PR. Revert the hunk.

