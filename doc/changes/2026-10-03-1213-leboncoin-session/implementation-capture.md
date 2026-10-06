# Desktop capture implementation

Status: ready for independent review. Date: 2026-10-03. Agent: `/root/leboncoin_session_capture_developer` (developer role).

Read `AGENTS.md`, the developer role, all four required project skills, the staged workflow, both subjects' functional requirements, capture/collection technical designs, and the unchanged [API record](api-step.md). No refactoring, API, frontend, dependency, package-lock, or deployment change was introduced.

## Delivered scope and traceability

- `scripts/capture-leboncoin-session.mjs`: CAP-001–007 / TS-LBC-CAP-001–004. Strict CLI and listing validation before browsing; existing `@playwright/test` Chromium export; fresh private profile and visible directly spawned browser with explicit nonzero loopback CDP port. An unpredictable startup-page fragment identifies the task's own endpoint before connecting; the connected context must contain that same page. No personal-profile access, fingerprint overrides or challenge interaction.
- The helper tracks main-frame document requests/responses and DOM readiness, then requires matching canonical URL, 2xx status, safe integer identity and active ad data. It exports only one applicable valid `datadome` cookie with versioned minimal metadata. Private ownership/permission checks and same-directory exclusive temporary write/sync/rename protect new and renewed exports.
- Bounded startup and total capture timeout, SIGINT/SIGTERM cancellation, idempotent browser cleanup, process-group termination and temporary-profile disposal cover handled exit paths. Diagnostics are fixed safe messages; browser streams are suppressed.
- `scripts/capture-leboncoin-session.test.mjs`: thirteen focused Node tests with synthetic cookies, temporary files outside the repository, browser/page mocks, write-failure injection, and actual CLI signal handling using a local sleeping stand-in executable.
- `doc/leboncoin-session.md` and one README link: CAP-008–009 / TS-LBC-CAP-005. Prerequisites, manual verification, protected SSH/scp staging and atomic service-user installation, one-time systemd drop-in, no-restart renewal, sidecar repair, normal refresh checks and portability limitations. The SSH sequence enters an interactive remote terminal before the sudo heredoc so normal sudo prompts work.

## Actual RED and GREEN evidence

1. Before implementation, six behavioral tests ran against minimal exported-function stubs with `/tmp/pricefollower-node22/node_modules/node/bin/node scripts/capture-leboncoin-session.test.mjs`. All six failed assertions: URL canonicalization, successful matching-document gating, minimal cookie export, aborted write rejection/old-file preservation, atomic renewal, and cleanup on failure. The module imported successfully; these were behavioral failures, not missing-module evidence.
2. Initial `node --test` execution inside the restricted sandbox returned a generic child-runner exit failure. Direct execution produced complete node:test TAP; no test framework or dependency was changed to work around sandbox restrictions.
3. After implementation the initial six tests passed, then the expanded suite passed. Final command, with scoped sandbox escalation to permit Node's child test runner and local loopback startup probe: `/tmp/pricefollower-node22/node_modules/node/bin/node --test scripts/capture-leboncoin-session.test.mjs`. Result: **13 passed, 0 failed**, including real CLI SIGINT (130) and SIGTERM (143), unchanged previous output, and removed temporary profiles.
4. Fault checks cover file write, sync and rename failure; mode/symlink/nonregular/repository rejection; challenge/wrong-ID/inactive/rounded-ID gating; missing/expired/ambiguous/wrong-scope cookies; startup collision/spawn/CDP/cancellation failures; success, timeout and cancellation cleanup. Synthetic cookie values are absent from emitted lifecycle diagnostics.

No live LeBoncoin browser capture, personal-profile access, remote SSH installation, or Raspberry Pi execution was performed by this developer. Live human verification and independent review/QA belong to the coordinator's subsequent stages. No commits were created by this agent.

## Accepted review correction

REV-LBC-001: changed decoded listing-path matching to case-insensitive, preserving the Go parser's acceptance of `/AD/Voitures/3245888872` and canonicalizing it to `/ad/voitures/3245888872`. Added that exact regression assertion. Reran `/tmp/pricefollower-node22/node_modules/node/bin/node --test scripts/capture-leboncoin-session.test.mjs` with scoped sandbox escalation: **13 passed, 0 failed**. Independent recheck remains the reviewer's responsibility. No other behavior or file scope changed for this correction.

## QA-LBC-F01 correction

Date: 2026-10-03. Agent: capture developer (developer role). This implements only the fix decided in [decisions.md](decisions.md) for QA-LBC-F01. Files changed: `scripts/capture-leboncoin-session.mjs`, `scripts/capture-leboncoin-session.test.mjs` and this record. No specification, API, documentation, dependency or refactoring change.

### Change

- `openBrowser` now uses the startup marker `data:text/plain,pricefollower-${randomUUID()}` instead of `about:blank#pricefollower-${randomUUID()}`. Only that one line changed, so the same `marker` value is still used for the spawn argument, the exact `page` URL match in `/json/list`, and the `context.pages()` lookup. Flags, the immediate fail-closed collision check and its message, the 20 s startup bound, cleanup and timeouts are unchanged. There is no retry, prefix matching, `/json/new` or other relaxation.
- Test "startup uses own visible isolated target; collision and failed spawn remove profiles": records the spawned marker for each of the five invocations and asserts that each one matches `^data:text/plain,pricefollower-[0-9a-f-]{36}$` and that all five are distinct. The collision stub now returns a page whose URL is `chrome://newtab/`, which is what Chrome 149 actually reported. The existing assertions still require that startup is rejected, the temporary profile is removed and the child is killed.

### RED and GREEN

All commands use `/tmp/pricefollower-node22/node_modules/node/bin/node` (v22.23.3).

1. RED: I changed the test first, then ran `node --test scripts/capture-leboncoin-session.test.mjs`. Result: **12 passed, 1 failed**. The startup test failed with `The input did not match the regular expression /^data:text\/plain,pricefollower-[0-9a-f-]{36}$/`, and the actual value was `about:blank#pricefollower-<uuid>`.
2. GREEN: I changed the marker in the script and reran the same command. Result: **13 passed, 0 failed** (including the real CLI SIGINT/SIGTERM test).

Resulting SHA-256: script `8ec4e0fd493be6c52a698cf2c6bf770eeb2cfacfd7c01ccb808a184c253cfc01`, test `0b5e6c474e3b0b39808804151aa66b0f3f6e8d5e5866dff552d0df63338716b3`.

### Real-browser smoke runs (unmodified Google Chrome 149.0.7827.53, `DISPLAY=:1`)

The output directory was a new scratchpad directory, `…/scratchpad/f01-smoke`, with mode 0700 and no pre-existing export. Before each run there were no `pricefollower-capture-*` profiles.

1. Ran `node scripts/capture-leboncoin-session.mjs --url https://www.leboncoin.fr/ad/voitures/3245888872 --output <dir>/s.json --browser /usr/bin/google-chrome --timeout-seconds 15`.
   - Expected: the waiting message, then a timeout.
   - Actual: the helper printed "Opening an isolated visible browser…" and then "Waiting for the requested active listing and its verified session…". It did not report a port collision, which confirms the marker fix with the real browser. LeBoncoin then served the listing without a challenge. Nobody interacted with the browser. The helper therefore verified the listing and printed "Session saved privately to …/s.json". Exit 0 after 8.6 s.
   - The export was mode 0600 and 311 bytes. A structure-only check, which did not print the value, showed: keys `version, capturedAt, cookie`; version 1; cookie name `datadome`, domain `.leboncoin.fr`, path `/`, secure `true`, an expiry present, and a non-empty value. The cookie value was not displayed or recorded.
   - The file was then deleted with `shred -u`.
   - Afterwards there were 0 `pricefollower-capture-*` profiles and 0 Chrome processes using such a profile.
2. Because the run above did not reach the timeout, I added one run with a listing that cannot verify: the same command with `--url https://www.leboncoin.fr/ad/voitures/1`.
   - Actual: the two messages above, then "Capture timed out. Complete verification and try again.", exit 1 after 17.1 s (15 s capture timeout plus startup).
   - No output file was written (the directory was empty).
   - Afterwards there were 0 profiles and 0 helper Chrome processes.

### Limitations

- I did not run `/opt/google/chrome/chrome` or Playwright's bundled Chromium.
- I did not test preservation of an existing prior export in the real-browser runs; the Node tests cover it.
- The QA retest of QA-CAP-018 and live QA-LIVE-001 remain with the QA tester and coordinator. Run 1 was not an operator live capture and is not QA-LIVE-001 evidence.
- Independent reviewer recheck is pending. No commits were made.
