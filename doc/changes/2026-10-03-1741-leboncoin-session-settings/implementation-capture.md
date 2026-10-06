# Capture helper and documentation implementation evidence

Status: implemented; Node 20.19.2 and Node 22.23.3 suites pass; real-Chrome smoke runs executed; ready for independent review. Date: 2026-10-03. Agent: capture/docs developer (developer role, separate agent invocation).

Inputs read before coding: `AGENTS.md`, [developer role](../../../.agents/roles/developer.md), [workflow](../../workflow/WORKFLOW.md) section 4, all four project skills (`carbon-frontend`, `frontend-architecture`, `go-sqlite-backend`, `go-backend-architecture`; no frontend or Go code was in scope), the [change index](index.md) (user decisions 6 and 7, API approval), [capture functional](../../specifications/leboncoin-session-capture/functional.md) and [technical](../../specifications/leboncoin-session-capture/technical.md) specifications revision 2 (FR-LBC-CAP-010–018, TS-LBC-CAP-006–012, root cause D1–D3), and file removal [functional](../../specifications/leboncoin-session-file-removal/functional.md) and [technical](../../specifications/leboncoin-session-file-removal/technical.md) specifications (documentation parts).

Ownership: `scripts/capture-leboncoin-session.mjs`, `scripts/capture-leboncoin-session.test.mjs`, `doc/leboncoin-session.md`, `doc/deprecated/**`, `README.md`, plus the one `doc/README.md` line required by TS-LBC-RM-004 (see “Scope notes”). No Go, frontend, specification, `package.json` or history file was touched. No dependency added. Nothing committed or staged.

## Requirement / technical ID to change mapping

| Technical ID | Functional IDs | Change |
| --- | --- | --- |
| TS-LBC-CAP-006 | CAP-010 | `parseArguments`: `--url` required, `--browser` and `--timeout-seconds` (1–3600, default 600) optional; `--output` and `--output=…` anywhere rejected first with the exact specified message; unknown, duplicate or value-less options → usage on stderr, exit 1; `--help` → usage on stdout, exit 0. URL validation (`parseListingURL`, TS-LBC-CAP-001) unchanged and run before any browser launch. |
| TS-LBC-CAP-007 / TS-LBC-RM-002 | CAP-011, CAP-012, RM-003 | `validateDestination`, `writeSession`, `sessionExport`, the repository/destination checks and the static output path removed. `sessionLine(cookies, listing, now)` keeps the single-applicable-cookie and validity checks and returns `datadome=<value>`. `main` prints, after `capture` has closed the browser and removed the profile, the instruction on stderr and then exactly one `datadome=<value>` line on stdout; nothing reaches stdout on any failure path. |
| TS-LBC-CAP-008 | CAP-015, CAP-017, CAP-018 | First stderr line printed before anything else; `checkRuntime` (≥ 20.19.0, uses only syntax older Node parses); `loadPlaywright` dynamic `import('@playwright/test')` with the `npm ci` message; `checkDisplay` (Linux: `DISPLAY` or `WAYLAND_DISPLAY`); `findBrowser` (`--browser`, else `google-chrome`, `google-chrome-stable`, `chromium`, `chromium-browser` on `PATH` in name order, macOS Chrome app path, then Playwright's `chromium.executablePath()`); progress messages “Opening <path> with a temporary profile…”, window message, “Waiting … Deadline: HH:MM:SS …”, “Page loaded; checking the listing…” once per loaded LeBoncoin top-level document. |
| TS-LBC-CAP-009 | CAP-014 | `showWindow` after CDP connect, within the 20-second startup bound: `Browser.getWindowForTarget` on the marker page's CDP session (failure/no `windowId` → “The browser started but did not open a window within 20 seconds.”), then best-effort `Browser.setWindowBounds { windowState: "normal" }` and `page.bringToFront()`. No Ozone flag added. Spawn flags, private 0700 profile, marker page, probed port, suppressed child output retained. |
| TS-LBC-CAP-010 | CAP-016, CAP-017 | The `isClosed()/isConnected()/exited` poll replaced by listeners (`child` `exit`/`error` at spawn, `browser` `disconnected` and `page` `close` after connect). The first post-connect event starts a 1-second settling window, then `browserEndMessage` classifies the collected events into one of the specified messages and aborts the wait. Startup exit/error → “The browser exited during startup (exit code N / signal S / it could not be started)…”. `waitForSession` records the last observation (`challenge` on 403 or a `captcha-delivery.com` frame, `not-listing`, `navigation-error`, `no-cookie`) for `timeoutMessage`; a verified listing without a usable cookie for 15 consecutive seconds ends early with the specified message. Unknown errors → “Capture failed because of an unexpected error (<name>).” Cancellation unchanged (130/143, “Capture cancelled.”). Idempotent cleanup retained on every path. |
| TS-LBC-CAP-011 / TS-LBC-RM-003 | CAP-013, RM-004 | `doc/leboncoin-session.md` rewritten as “LeBoncoin session from Settings” (prerequisites, command, what the operator sees including a window possibly behind others, copying the line, pasting into Settings via the profile icon, checking with refresh controls and the settings hint, renewing, clearing with an empty save, no acceptance/lifetime guarantee, secret handling, every failure message with its fix, pointer to the deprecation record). No `scp`, systemd, environment variable, private directory or sidecar instruction. `README.md` link text → “LeBoncoin verification: capture a session and paste it in Settings”. |
| TS-LBC-RM-004 | RM-005 | New `doc/deprecated/README.md` (purpose, table row for the file-based session) and `doc/deprecated/leboncoin-session-file.md` (what it was, why removed with the quoted request, replacement links, clean-up commands, history with the four commit messages and `git log` usage, links to the change records, superseded revision 1 rows and revision 1 technical designs; removal commit message left for the coordinator). `doc/README.md` gains the specified link line. |
| TS-LBC-CAP-012 | all | `scripts/capture-leboncoin-session.test.mjs` rewritten: export/destination tests removed; 22 tests cover the cases listed under TS-LBC-CAP-012 verification. |
| TS-LBC-RM-005 | RM-006 | `doc/changes/2026-10-03-1213-leboncoin-session/` and `doc/changes/2026-10-03-0854-leboncoin-403-investigation/` untouched (`git diff --stat` empty). |

## Changed files

- Modified: `scripts/capture-leboncoin-session.mjs` (rewritten), `scripts/capture-leboncoin-session.test.mjs` (rewritten), `doc/leboncoin-session.md` (rewritten), `README.md` (one line), `doc/README.md` (one line).
- Added: `doc/deprecated/README.md`, `doc/deprecated/leboncoin-session-file.md`, this record.
- `DEPLOYMENT.md`: checked, contains no session reference; unchanged.

## Test-first evidence

RED, before any implementation change (old helper, new test file importing the module as a namespace so missing exports fail per test instead of at link time):

```text
$ node --test scripts/capture-leboncoin-session.test.mjs        # Node v20.19.2
# tests 21  # pass 2  # fail 19
```

Behavioral failures included: `--url` alone rejected with the old usage requiring `--output` (test 4); `openBrowser` rejecting with “Browser executable unavailable. Run npx playwright install chromium…” instead of confirming a window (11–13); `waitForSession` calling the removed-by-design `page.isClosed` poll (14–16); `main` printing no first message and exiting 1 for a valid run (18–19); the real CLI accepting/requiring `--output` and the SIGINT/SIGTERM run never starting the browser (20–21). Tests 3, 5–10 failed because the new functions did not exist. The two passing tests are the retained URL and listing-verification rules.

GREEN after implementation: 21/21 on Node 20.19.2.

A second RED/GREEN cycle came from the real-Chrome smoke run (below): killing the browser with SIGSEGV during the polling pause printed “Capture failed because of an unexpected error (AbortError).” because `timers/promises` `delay` rejects with a generic `AbortError` rather than the abort reason. Regression test “an abort during the polling pause reports its own reason…” was added first and failed (`'The operation was aborted'`); the fix is a small `pauseFor` helper used for the poll pause and the startup retry delay (which had the same issue on a startup-phase timeout). Then:

```text
$ node --test scripts/capture-leboncoin-session.test.mjs                                          # v20.19.2
# tests 22  # pass 22  # fail 0
$ /tmp/pricefollower-node22/node_modules/node/bin/node --test scripts/capture-leboncoin-session.test.mjs   # v22.23.3
# tests 22  # pass 22  # fail 0
```

No `pricefollower-capture-*` directory remained in `/tmp` before or after the suites.

## Real smoke runs (GNOME Wayland host, `DISPLAY=:1`, `WAYLAND_DISPLAY=wayland-0`, Node v20.19.2, Google Chrome via `/usr/bin/google-chrome`, no `--browser`)

| Run | Command | Observed |
| --- | --- | --- |
| a | `--url https://www.leboncoin.fr/ad/voitures/1 --timeout-seconds 25` | Browser auto-discovered (`Opening /usr/bin/google-chrome with a temporary profile…`). Timestamps: first message at 0.21 s, opening at 0.80 s, window confirmed at 1.79 s, waiting/deadline at 1.81 s, three “Page loaded” lines, then at 25.97 s “Capture timed out after 25 seconds: LeBoncoin was still showing a verification page.” Exit 1, stdout empty, no leftover profile or Chrome process. |
| a (final code) | same, `--timeout-seconds 20` | Same progress; “Capture timed out after 20 seconds: the page was not the requested active listing (check the URL or try another active listing).” (no challenge this time, LeBoncoin served the missing listing page). Exit 1, stdout empty, no leftovers. |
| crash | same URL, `--timeout-seconds 40`, `kill -SEGV` on the helper's Chrome child after 7 s | Before the fix: “Capture failed because of an unexpected error (AbortError).” After the fix: “The browser exited unexpectedly (signal SIGSEGV) before verification finished. Retry; if it repeats, start the browser manually to check it works.” Exit 1, stdout empty, profile removed, no Chrome process left. (A first attempt mistakenly signalled the coordinating shell because its own command line matched the `pgrep` pattern; the helper kept waiting correctly and was then tested with the right PID.) |
| b | `--url https://www.leboncoin.fr/ad/voitures/3245888872 --timeout-seconds 60`, stdout redirected to a scratchpad file | Exit 0; stderr ended with the paste instruction; stdout had exactly 1 line matching `^datadome=\S+$` (checked with `grep -qxE`, value not printed or recorded); the file was then removed with `shred -u`. No human interaction was needed (LeBoncoin served the listing directly). No leftover profile. |
| CLI | `--help`; `--output /tmp/x`; `DISPLAY= WAYLAND_DISPLAY=`; `--browser /nonexistent`; `--url http://example.com/` | Each printed the first message then, respectively: usage (exit 0); the `--output` message; the display message; “The browser at /nonexistent was not found or is not executable.”; “Use a supported HTTPS LeBoncoin listing URL.” (all exit 1). |

## Repository search (FR-LBC-RM-004)

```bash
git grep -n -E 'LEBONCOIN_SESSION_FILE|session\.json|state\.json|leboncoin-session\.conf|--output' -- ':!doc/deprecated' ':!doc/changes' ':!doc/specifications'
```

Returns only the helper's `--output` rejection constant/check and its tests, as expected by TS-LBC-RM-001 verification. A working-tree search (tracked and untracked, excluding `node_modules`, `doc/`, `.tmp-session-qa/`, build output) for those terms plus `sidecar`, `NewCollectorWithSession` and `LeboncoinSessionFile` found nothing; the current documentation set (`README.md`, `DEPLOYMENT.md`, `doc/README.md`, `doc/leboncoin-session.md`, `doc/FUNCTIONAL_SPECIFICATIONS.md`, `doc/use-cases/`, `API_SPECIFICATION.md`, `doc/architecture/`, `doc/workflow/`) has no match. No leftover reference outside my ownership to report. All relative links (with anchors) in the new and changed documents resolve (scripted check).

## Scope notes and technical interpretations for the reviewer

1. `doc/README.md` is not in the coordinator's file list for this agent but its one link line is required by TS-LBC-RM-004 and no other agent owns documentation; only that line was added.
2. Classification gap: the TS-LBC-CAP-010 table does not cover a page `close` followed by a non-zero exit code or a signal. The implementation reports it as an unexpected exit (a deliberate window close exits with code 0), and treats a child `error` after connect the same way without a code.
3. `no-cookie` timeout reason: the specification lists four timeout reasons; a timeout that falls within the 15-second no-cookie grace uses “the listing loaded but LeBoncoin had not set a usable datadome cookie”. HTTP 5xx documents are classified `navigation-error`, other non-2xx or unverified loaded documents `not-listing`.
4. A CDP connection that never succeeds within 20 seconds (child still running) reports “The browser started but could not be controlled within 20 seconds. Retry the capture.” (startup failure category of FR-LBC-CAP-017; exact text not specified).
5. The runtime check runs before argument parsing so `--help` also prints the first progress line on stderr; usage goes to stdout.
6. Windows: the previous helper refused non-POSIX systems inside the removed destination check; that refusal is kept in `checkDisplay` as “This helper supports Linux and macOS desktops only.”
7. The operator guide documents the obsolete-option message without the literal `--output` flag so the TS-LBC-RM-001 search stays limited to the helper and its test.

## Limitations

- Window visibility on screen was not observed by a human and no desktop screenshot was taken (GNOME Wayland blocks non-portal screenshots). Evidence is the helper's CDP confirmation (`Browser.getWindowForTarget` returned a window, followed by `windowState: normal` and `bringToFront`) on every real run; the visible-window confirmation remains a QA-stage item.
- Closing the window by hand and quitting Chrome were not exercised on real Chrome (no human operator); they are covered by event-order unit tests, and the investigation's recorded close order (page `close`, `disconnected`, exit 0) maps to the window-closed message.
- The runtime check was verified with injected versions only; no Node < 20.19 binary was executed.
- Smoke run b obtained a session without a challenge; the human-challenge path itself was not exercised.
- The printed line's acceptance by the Settings parser (FR-LBC-SET-005) is covered by the backend agent's Go fixture, not re-run here.

## Review fixes

### REV-SET-008 (accepted): logical assignment prevented older Node.js from reaching the runtime message

- RED: added test “source avoids logical assignment so older Node.js can parse it and report the runtime message” (asserts the helper source contains no `??=`, `||=` or `&&=`). Run before the fix: `node --test --test-name-pattern='logical assignment' scripts/capture-leboncoin-session.test.mjs` → `not ok 23`, fail 1 (three `??=` present).
- Fix in `scripts/capture-leboncoin-session.mjs`: replaced the three `??=` uses with explicit equivalents: `if (settleTimer === undefined) settleTimer = setTimeout(…)`; `close` now returns the existing `closing` promise if set, otherwise assigns and returns the new cleanup promise; `if (noCookieSince === undefined) noCookieSince = now()`. Behavior unchanged.
- GREEN: `node --test scripts/capture-leboncoin-session.test.mjs` (v20.19.2) → 23/23 pass; `/tmp/pricefollower-node22/node_modules/node/bin/node --test scripts/capture-leboncoin-session.test.mjs` (v22.23.3) → 23/23 pass.
- Limitation: no Node.js older than 20.19 was executed; the parse guarantee rests on the source assertion.
