# No new tests for issue #256

No test file is added or changed. The change edits no code: it changes the GitHub state of issue #238 (FR-TEST-HYG-065, done by the owner, as no stage has a GitHub token) and status lines under `doc/specifications/test-suite-hygiene/` (FR-TEST-HYG-066). Neither a GitHub issue state nor a status line can be checked by a red-then-green Go or Playwright test, and FR-TEST-HYG-067 forbids `_test.go` and `.spec.ts` changes. The cases are GitHub state checks, repository checks and runs of the existing suites (technical specification, "Test case mapping (issue #256)").

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | GitHub check by the owner: `gh issue view 238 --json state,stateReason` gives `"state":"CLOSED"` and `"stateReason":"COMPLETED"`, the same on a second read. It is the last command of the close step (TS-TEST-HYG-085), and its output is quoted in the trusted comment on issue #256 that the development stage requires before editing (TS-TEST-HYG-086). |
| TC-002 | GitHub check by the owner: `gh issue view 238 --json comments` and `gh api repos/redjhawk/pricetracker/issues/238/timeline` compared with the branch the close step took. Open: one `closed` event (completed) and one new comment equal to the closing comment. Already closed as completed: no new event or comment. Closed with another reason: `reopened`, then `closed`, and one comment. Running the step again takes the `CLOSED/COMPLETED` branch, so it adds nothing. |
| TC-003 | Repository check: `sed -n 3p doc/specifications/test-suite-hygiene/functional.md \| grep -o '[^;]*FR-TEST-HYG-019[^;]*'` prints exactly one entry, `FR-TEST-HYG-019 implemented (issue #256)`, and the line holds no `closed by the merge`. `git diff master -- doc/specifications/test-suite-hygiene/functional.md` changes only line 3, with the three edits of TS-TEST-HYG-087. No edit is made without the evidence of TC-001. |
| TC-004 | Repository check: every path in `git diff --name-status master` starts with `doc/` and none with `doc/todo/`; `scripts/`, `API_SPECIFICATION.md`, the issue #238 and #241 change folders and `doc/todo/issue-241-r1-01.md` are unchanged. Gates: `go vet ./...`, `go test ./...` (also `-count=1`, twice), `node --test scripts/*.test.mjs`, `npm run build`, `npx playwright test` (TS-TEST-HYG-088). |
| TC-005 | GitHub check by the owner: `gh issue list --state all --json number,state,stateReason` before the close step and after the merge differs only for #238 and #256; #233 and #241 are unchanged. Repository check: the pull request body holds only `Closes #256` (`scripts/factory/factory.sh`, unchanged). |

## Run in this stage

On the unchanged base code, `go vet ./...` reports nothing and `go test ./...` passes for every package (`internal/amazon`, `internal/amazon/amazontest`, `internal/claude`, `internal/httpapi`, `internal/leboncoin`, `internal/model`, `internal/service`, `internal/store`, `internal/version`: all `ok`).
