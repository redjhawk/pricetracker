# Technical specification: LeBoncoin session capture

Status: ready (revision 2, 2026-10-03; depends on the settings API (API approved by the user 2026-10-03 (proposal revision 1 unchanged; see the change index and api-step.md); implementation authorized) only for the operator workflow — the helper itself calls no API)\
Functional source: [capture requirements](functional.md) revision 2, FR-LBC-CAP-010–018 (FR-LBC-CAP-002–004 unchanged; 001, 005–009 superseded).\
Revision 1 status: implemented by change [leboncoin-session](../../changes/2026-10-03-1213-leboncoin-session/index.md); its file-export and Raspberry Pi transfer parts are superseded below and kept as history ([deprecation record](../../deprecated/leboncoin-session-file.md)).\
Companions: [settings](../leboncoin-session-settings/technical.md) (accepts the printed line), [collection rev. 2](../leboncoin-session-collection/technical.md), [file removal](../leboncoin-session-file-removal/technical.md).

## Revision 2 summary

The helper keeps its isolated visible browser, URL validation and listing/cookie verification (TS-LBC-CAP-001–003), stops writing files (TS-LBC-CAP-004 superseded) and prints one `datadome=<value>` line on success. It gains immediate progress messages, prerequisite checks, installed-browser discovery, an explicit window-visibility step, and event-based classification of how the browser ended, which fixes the reported “doesn't open a browser / Browser closed…” behavior (root-cause analysis below).

## Revision 2 requirement mapping

`CAP-*` abbreviates `FR-LBC-CAP-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-LBC-CAP-006 | CAP-010 | CLI: `--url` required; `--browser`, `--timeout-seconds` optional; `--output` rejected with an explanation | `scripts/capture-leboncoin-session.mjs` | `node:test` argument cases; CLI run |
| TS-LBC-CAP-007 | CAP-011, CAP-012, RM-003 | Print instruction (stderr) and exactly one `datadome=<value>` line (stdout) after cleanup; no file code | same | Unit test of output function; CLI success with stub browser; FR-LBC-SET-005 parser accepts the line (Go test fixture) |
| TS-LBC-CAP-008 | CAP-015, CAP-017, CAP-018 | Message before any slow step; Node ≥ 20.19 check; dynamic Playwright import with dependency message; browser discovery; display check | same | Tests with injected `versions`, `env`, `access`, importer |
| TS-LBC-CAP-009 | CAP-014 | After CDP connect: confirm the marker page's window (`Browser.getWindowForTarget`), set `windowState: normal`, `bringToFront`, print where to look; no forced Ozone flag | same | Stub-CDP tests; executed visible QA on GNOME Wayland |
| TS-LBC-CAP-010 | CAP-016, CAP-017 | Event-driven end reasons (page `close`, browser `disconnected`, child `exit` code/signal) with ordering rule; distinct messages; last-observation reason on timeout; 15 s no-cookie grace | same | Fake page/browser/child event-order tests; real-Chrome close-window QA |
| TS-LBC-CAP-011 | CAP-013, RM-004 | Rewrite `doc/leboncoin-session.md` for print-and-paste; README link text | `doc/leboncoin-session.md`, `README.md` | Doc review; grep for removed terms |
| TS-LBC-CAP-012 | all | Rewrite `scripts/capture-leboncoin-session.test.mjs`: remove export/destination tests, add the cases above | `scripts/capture-leboncoin-session.test.mjs` | `node --test scripts/capture-leboncoin-session.test.mjs` on Node 20.19 and 22 |

## Reported defect: investigation and root cause

Reports: the user says the helper “doesn't open a browser” and shows no message. The coordinator's run (GNOME Wayland, `DISPLAY=:1`, `WAYLAND_DISPLAY=wayland-0`, system Node v20.19.2, Chrome 149 via `/usr/bin/google-chrome`, listing ID 1) started Chrome with the temporary profile, then printed “Browser closed before a verified listing was captured.” and exited 1 well before the timeout.

Diagnostic runs by the technical specifier on the same host, 2026-10-03 (scratchpad scripts; no cookie value read or recorded; two LeBoncoin page loads per run, three runs):

1. Spawn exactly as the helper does (`/usr/bin/google-chrome`, temp profile, CDP on 127.0.0.1, `data:` marker), Node v20.19.2, navigate to `data:`/`https://example.com/`: the child stayed alive, `browser.isConnected()` stayed true and `page.isClosed()` stayed false for 10 s.
2. Same, navigating to `https://www.leboncoin.fr/` and `https://www.leboncoin.fr/ad/voitures/1` (both answered 403, i.e. the DataDome challenge, which then reloaded twice): all three flags stayed healthy for 20 s.
3. The unmodified helper (`--url …/ad/voitures/1 --browser /usr/bin/google-chrome --timeout-seconds 20`) on Node v20.19.2: printed both progress lines, waited, and ended with “Capture timed out…” after 22 s — no early exit.
4. Chrome 149 chose the native Wayland backend by itself (`chrome://gpu`: “Ozone platform: wayland”); `Browser.getWindowForTarget` reported a `normal` 945×1060 window. `page.bringToFront()` and `Browser.setWindowBounds` succeeded.
5. Closing the only tab produced, in order: page `close`, browser `disconnected`, child `exit` with code 0 and no signal.
6. `/usr/bin/google-chrome` is a Bash wrapper that `exec`s `/opt/google/chrome/chrome`, so the spawned PID is the browser; no hand-off to the operator's running personal Chrome occurs with a distinct `--user-data-dir`. The Playwright bundled Chromium is **not** installed on this host (`~/.cache/ms-playwright` absent).

Conclusion: the early exit was not reproduced, and Node 20 vs 22 is not the cause (the helper works on 20.19.2; Playwright 1.63 requires Node ≥ 20). The established defects, by inspection and the runs above, are:

- **D1 — misclassified end of browser.** `waitForSession` polls `page.isClosed() || exited || !browser.isConnected()` and reports all of them as “Browser closed before a verified listing was captured.” It discards the child exit code/signal and the order of events, so an operator closing the window (event 5 order, exit 0), a Chrome crash (non-zero code or signal) and a lost CDP connection are indistinguishable. The coordinator's message therefore only proves that one of these happened. Because that run opened a real window on the user's desktop, an operator closing it is as plausible as a crash; the helper cannot tell (violates FR-LBC-CAP-016/017).
- **D2 — no visibility step or guidance.** On GNOME Wayland a newly launched client is mapped without focus when another application is focused (focus-stealing prevention; GNOME may show a “Google Chrome is ready” notification instead of raising the window). The helper never raises the window, never confirms that a window exists, never names the browser, and only says “Opening an isolated visible browser…”. A window opening behind the terminal/editor matches “doesn't open a browser” (FR-LBC-CAP-014).
- **D3 — messages and prerequisites.** Playwright is imported statically, so missing repository dependencies end with a raw Node `ERR_MODULE_NOT_FOUND` stack trace and no helper message; there is no runtime or display check; the default browser is Playwright's bundled Chromium, which is absent on this host, so a run without `--browser` cannot open any browser; and the old `--output` validation failed before launch with a generic message (FR-LBC-CAP-015/017/018).

The fixes below address D1–D3. If the early exit recurs after the fix, the new message will name the actual cause (window closed, browser exit code/signal, or lost connection).

## Frontend (revision 2)

Not affected: the helper is a terminal program. The operator pastes the printed line into the settings modal ([settings design](../leboncoin-session-settings/technical.md)).

## Backend (revision 2)

Not affected: the helper calls no API and touches no database. The Go parser (TS-LBC-SET-002) must accept the printed line unchanged; a Go test uses a synthetic `datadome=<value>` line in exactly the printed format.

## API (revision 2)

Not affected: no endpoint is called. The operator workflow depends on the pending settings endpoints for pasting the value.

## Desktop helper (revision 2)

### TS-LBC-CAP-006: command line

```text
node scripts/capture-leboncoin-session.mjs --url <listing-url> [--browser <executable-path>] [--timeout-seconds <1-3600>]
```

`--help` prints usage and exits 0 without launching. Unknown, duplicate or value-less options print usage to stderr and exit 1. `--output` (and `--output=…`) exits 1 with: “--output is no longer supported. The session is now printed here; paste it into Settings › LeBonCoin session in PriceFollower.” URL validation is TS-LBC-CAP-001 unchanged. Timeout default 600 s, integer 1–3600 (unchanged). All of this happens before any browser launch.

### TS-LBC-CAP-007: printed result, no file

- Delete `validateDestination`, `writeSession`, the repository/destination checks and every `--output` code path. No file is created anywhere except the temporary browser profile, which is removed on exit (retained cleanup rules of TS-LBC-CAP-002).
- `verifiedListing` and the single applicable `datadome` cookie selection (TS-LBC-CAP-003) are kept; `sessionExport` is replaced by `sessionLine(cookies, listing, now)` returning `datadome=<value>` after the same validity checks (name, allowed domain/path scope, value bytes 0x21–0x7E without `"` `,` `;` `\`, ≤ 4096 bytes, not expired).
- Order on success: close the browser and remove the profile; then write to **stderr** “Session captured. Paste the next line into Settings › LeBonCoin session in PriceFollower. Keep it private: do not share it in chats, tickets or logs.”; then write exactly one line `datadome=<value>\n` to **stdout**; exit 0. Stdout contains nothing else, so `… > file` or piping captures only the value; nothing is printed to stdout on any failure path.

### TS-LBC-CAP-008: progress and prerequisites

All progress and failure messages go to stderr, each when its stage begins:

1. Immediately after start, before argument validation and before importing Playwright: “PriceFollower LeBoncoin session capture — checking prerequisites…”.
2. Runtime: `process.versions.node` must be ≥ 20.19.0 (oldest version verified on the reporting host; Playwright 1.63 requires ≥ 20). Otherwise: “Node.js 20.19 or newer is required (found vX). Install a newer Node.js, then retry.” The check uses only syntax that older Node versions parse. `package.json` `engines` (≥ 22, for the build toolchain) is unchanged; the guide states the helper's minimum.
3. Dependencies: `await import('@playwright/test')` (dynamic, replacing the static import). Failure: “Repository dependencies are missing. Run npm ci in the repository, then retry.”
4. Graphical display (Linux only): `DISPLAY` or `WAYLAND_DISPLAY` must be non-empty, else “No graphical display was found (DISPLAY and WAYLAND_DISPLAY are not set). Run the helper from a terminal in your desktop session.” macOS is not checked; Windows remains unsupported (process-group termination is POSIX).
5. Browser discovery: `--browser` path if given (missing/not executable → “The browser at <path> was not found or is not executable.”); otherwise the first executable among `google-chrome`, `google-chrome-stable`, `chromium`, `chromium-browser` on `PATH` (Linux) or `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome` (macOS), then Playwright's `chromium.executablePath()` if that file exists. None → “No Chrome or Chromium browser was found. Install Google Chrome or Chromium, or pass --browser <path>.” Then: “Opening <path> with a temporary profile…”.
6. After TS-LBC-CAP-009: “A new browser window is open. If you do not see it, look for a ‘ready’ notification or switch windows (Alt+Tab / Activities).” and “Waiting for you to complete any LeBoncoin verification in that window. Deadline: <local HH:MM:SS>. Press Ctrl+C to cancel.”
7. When a new top-level LeBoncoin document finishes loading: once per document, “Page loaded; checking the listing…”.
8. Success: TS-LBC-CAP-007.

### TS-LBC-CAP-009: visible window

Spawning, private 0700 profile, fixed flags, `data:text/plain,pricefollower-<uuid>` marker, CDP on 127.0.0.1 with a probed free port, 20-second startup bound and suppressed child output are retained (TS-LBC-CAP-002). Added after `connectOverCDP` within the same 20-second bound:

- Find the marker page; open a CDP session on it and call `Browser.getWindowForTarget`. Failure or no `windowId` → fail with “The browser started but did not open a window within 20 seconds.”
- `Browser.setWindowBounds({ windowId, bounds: { windowState: "normal" } })` (un-minimize) and `page.bringToFront()`; errors here are ignored (best effort; focus-stealing prevention may still keep it behind).
- Do not add `--ozone-platform`/`--ozone-platform-hint`: Chrome 149 already selects Wayland on a Wayland session and X11 otherwise (investigation item 4); forcing a backend would break the other case.

### TS-LBC-CAP-010: end-of-wait classification

Replace the `isClosed()/isConnected()/exited` poll with listeners registered as soon as each object exists: `child.on('exit', (code, signal))`, `child.on('error')`, `browser.on('disconnected')`, `page.on('close')`. The first browser-side event starts a 1-second settling window (to collect the rest of the sequence), then the wait rejects with one reason:

| Observed | Message (stderr) | Exit |
| --- | --- | --- |
| Page `close` seen (before or within the window of a disconnect/exit), exit code 0 or browser still running | “The browser window was closed before verification finished. Run the helper again and leave the window open until it reports success.” | 1 |
| Child exit with non-zero code or a signal, no prior page `close` | “The browser exited unexpectedly (exit code N / signal S) before verification finished. Retry; if it repeats, start the browser manually to check it works.” | 1 |
| Child exit code 0 without page `close` (e.g. operator quit Chrome) | “The browser was quit before verification finished.” | 1 |
| `disconnected` while the child is still running | “Lost the connection to the browser before verification finished. Retry the capture.” (the child is then terminated by cleanup) | 1 |
| Child exit/error during startup (before connect) | “The browser exited during startup (exit code N / signal S). Check that it starts from this desktop session, or pass another --browser.” | 1 |

Waiting otherwise continues until verification succeeds, the deadline passes or the operator cancels (FR-LBC-CAP-016). The loop keeps the last observation of the polled page: `challenge` (document status 403 or a `captcha-delivery.com` frame), `not-listing` (wrong ID, inactive or no `__NEXT_DATA__`), `navigation-error`, `no-cookie` (verified listing, no single valid applicable `datadome`). Timeout message: “Capture timed out after N seconds: <reason>.” with reasons “LeBoncoin was still showing a verification page”, “the page was not the requested active listing (check the URL or try another active listing)”, “the listing page could not be loaded”, or “no page loaded yet”. A verified listing without a usable cookie for 15 consecutive seconds ends early: “The listing loaded but LeBoncoin did not set a usable datadome cookie. Retry, or try another active listing.” Cancellation (SIGINT 130, SIGTERM 143) stays “Capture cancelled.”. Unknown errors print the fixed “Capture failed because of an unexpected error (<error.name>).” (never raw messages, which may contain URLs). Every failure path prints exactly one final message and never prints a cookie value (FR-LBC-CAP-017).

Cleanup (retained): single idempotent close — CDP close, SIGTERM then SIGKILL of the process group with bounded waits, temporary profile removal — on every path, including after the window-closed cases.

### TS-LBC-CAP-011: operator guide

Rewrite `doc/leboncoin-session.md` (keep the file name so existing links resolve) as “LeBoncoin session from Settings”: prerequisites (desktop session, Chrome/Chromium, Node.js ≥ 20.19, `npm ci`); the command; what the operator sees (progress lines, a new window possibly behind other windows, complete verification without signing in); copying the single `datadome=…` line; pasting it into **Settings › LeBonCoin session** (profile icon at the top right); checking with the existing refresh controls and the settings hint; renewing when the hint reports a rejection or revocation; clearing by saving an empty field; no guaranteed acceptance from the server's network or lifetime; the printed value is a secret (terminal scrollback, clipboard); each failure message and its fix; a pointer to `doc/deprecated/leboncoin-session-file.md` for the old file workflow. No `scp`, systemd drop-in, environment variable, private directory or sidecar instructions. `README.md` line 79 link text becomes “LeBoncoin verification: capture a session and paste it in Settings”.

## Scope, verification and unresolved questions (revision 2)

Permitted files: `scripts/capture-leboncoin-session.mjs`, `scripts/capture-leboncoin-session.test.mjs`, `doc/leboncoin-session.md`, `README.md` (one line). No dependency, `package.json` or npm script change. No refactoring beyond deleting the file-export code.

Verification:

- `node --test scripts/capture-leboncoin-session.test.mjs` with Node 20.19.2 and Node 22: argument parsing (`--output` message), runtime check (injected version), missing dependency (injected importer failure), display check (injected env), browser discovery order (injected access/PATH), window confirmation failure, event-order classification for each table row (fake child/browser/page emitters), timeout reasons, 15 s no-cookie rule (fake clock), success writes exactly one stdout line and the instruction to stderr, no stdout on failure, cleanup exactly once on every path, no file created (temp dir listing).
- QA (later stage), on the GNOME Wayland host: run without `--browser` (discovers `/usr/bin/google-chrome`), confirm a window visibly appears (operator observation or screenshot), first message within 1 s, close the window → window-closed message, kill the browser process (`kill -SEGV` on the browser PID) → unexpected-exit message, `--output` rejection, `DISPLAY= WAYLAND_DISPLAY=` → display message, short timeout on a challenge/nonexistent listing → timeout reason, at most one live capture to verify the printed line is accepted by Settings. Visible-window confirmation needs a human observer or a desktop screenshot; record it as a limitation if neither is available.

No unresolved product question. Technical choices made here: Node ≥ 20.19 minimum for the helper, stdout for the value and stderr for messages, 1 s settling window, 15 s no-cookie grace, 20 s startup bound (unchanged).

---

# Revision 1 technical design (history)

The following revision 1 design (implemented) is kept unchanged for history except for the superseded markers in its mapping table and headings. Its functional source was FR-LBC-CAP-001–009.

## Revision 1 requirement mapping (history)

| Technical ID | Functional IDs | Design / intended files | Verification |
| --- | --- | --- | --- |
| TS-LBC-CAP-001 — retained (CLI contract amended by TS-LBC-CAP-006) | CAP-001 | CLI URL validation and canonicalization in `scripts/capture-leboncoin-session.mjs` | Table-driven URL cases matching Go acceptance and canonicalization |
| TS-LBC-CAP-002 — retained, amended by TS-LBC-CAP-008, 009, 010 | CAP-002, CAP-003, CAP-007 | Spawn visible Chromium with private temporary profile; local CDP; bounded lifecycle | Stub process/CDP lifecycle tests; executed visible-browser QA |
| TS-LBC-CAP-003 — retained (export becomes printing, TS-LBC-CAP-007) | CAP-004, CAP-005 | Matching successful document and ad ID before extracting only applicable `datadome` | Challenge, wrong ID, absent cookie, successful fixture cases |
| TS-LBC-CAP-004 — **superseded** by TS-LBC-CAP-007 | CAP-006 | Private same-directory temporary write, sync, atomic rename | Mode, symlink, unsafe destination, failed-write and renewal tests |
| TS-LBC-CAP-005 — **superseded** by TS-LBC-CAP-011 | CAP-008, CAP-009 | `doc/leboncoin-session.md` capture, SSH installation, configuration, renewal instructions | Command review and isolated local installation/renewal QA |

`CAP-*` abbreviates `FR-LBC-CAP-*` in this table.

## Frontend (revision 1, history)

Not affected. No Carbon component, route, browser-upload form, login, or client state is introduced. Existing item refresh/status interfaces are used to check server acceptance. The helper supplies plain keyboard-usable terminal messages and a normal visible browser; the operator handles third-party verification.

## Desktop helper (revision 1, history)

Use Node >=22 and the repository's installed Playwright package (currently supplied by `@playwright/test`); use Node standard libraries for filesystem, child process, networking and tests. No new dependency, package-lock change, platform service, or production Node runtime is needed. Implement the CLI in `scripts/capture-leboncoin-session.mjs`, with exported narrow functions and an entry-point guard for `node:test` verification in `scripts/capture-leboncoin-session.test.mjs`.

CLI contract:

```text
node scripts/capture-leboncoin-session.mjs --url <listing-url> --output <private-file> [--browser <executable-path>] [--timeout-seconds <seconds>]
```

Require URL and output; reject unknown/duplicate options and missing values. `--help` exits successfully without starting a browser. Timeout defaults to 600 seconds; accept an integer from 1 through 3600. Paths may contain spaces and must be passed as argv, never shell-interpolated. Browser defaults to `chromium.executablePath()` from installed Playwright; `--browser` permits a normal installed Chrome/Chromium executable. Report a missing executable with setup guidance (`npx playwright install chromium` or `--browser`). Desktop support requires a graphical POSIX environment where mode-0700 directories and mode-0600 files can be enforced; fail explicitly if secure filesystem protection cannot be provided. Do not pretend Windows mode bits provide equivalent protection.

TS-LBC-CAP-001: Match `internal/leboncoin.ParseURL` rules: HTTPS; no username/password; no port other than absent/443; case-insensitive apex or `www.leboncoin.fr` with an optional terminal host dot; decoded path matching `/ad/{ASCII alphanumeric, underscore or hyphen category}/{digits}` with an optional final comma. Strip query/fragment and comma; lowercase category and canonicalize to `https://www.leboncoin.fr/ad/{category}/{id}`. Do not silently broaden acceptance through WHATWG URL normalization (for example, backslashes, path dot-segments or empty userinfo); cover differences with paired Go/Node examples. Invalid input fails before launching a browser or replacing output. Verify the numeric ad identity without unsafe JavaScript integer rounding; the current listing data uses an integer ID, and an ID that cannot be compared reliably cannot validate export.

TS-LBC-CAP-002: Create a fresh mode-0700 temporary directory and profile. Do not expose a profile-selection or personal-cookie import option. Spawn a visible browser directly with `--user-data-dir`, `--no-first-run`, `--no-default-browser-check`, `--remote-debugging-address=127.0.0.1` and an available, explicit nonzero debugging port. Connect using `chromium.connectOverCDP`; do not use `launchPersistentContext`, a zero debugging port, `--enable-automation`, headless mode, stealth/fingerprint overrides or automated challenge interaction. Keep child stdout/stderr suppressed; raw browser diagnostic streams can contain sensitive URLs. Bound connection startup to at most 20 seconds within the total timeout. Handle spawn errors and premature exit. If the chosen port cannot be bound by the child, fail safely; do not attach to an unrelated existing browser. Use the spawned process lifecycle and the fresh endpoint for this connection; handle port collision as startup failure.

Navigate the fresh browser to the homepage and then the canonical listing (ordinary document loading only, bounded navigation calls). Explain that the operator should complete any verification in the window. Poll at a modest interval, such as one second, until verification succeeds, the browser closes, cancellation occurs or the total deadline expires. Never click/solve/submit CAPTCHA controls. A navigation timeout while a challenge is still usable may continue waiting within the deadline; a failed or unrelated navigation cannot count as success.

TS-LBC-CAP-003: Require the current top-level page URL to be the requested supported canonical listing, the latest relevant top-level document response to be successful (2xx), and parseable `script#__NEXT_DATA__` data with `props.pageProps.ad.list_id` matching the requested ID and `status: active`. A cookie, title, cached snapshot, or challenge page alone is insufficient. Price presence is not required for capture; actual price validation remains Go's responsibility. Read cookies applicable to the canonical URL; select exactly one valid `datadome` cookie with allowed scope and valid value according to the shared format below. Fail safely on ambiguous applicable cookies. Export no HTML, other cookies, screenshots, profile data, browser request headers, or account state.

TS-LBC-CAP-004: Resolve the output to a private existing directory owned by the invoking user (0700); operator documentation creates it with `install -d -m 0700`. Reject symlink parents/final targets, nonregular existing targets, targets owned by another user, existing group/other-readable files and destinations inside the repository. Validate the destination before browsing, then again before replacement. A private directory removes other-user replacement races; do not follow a final symlink or truncate an existing target. Use an exclusive mode-0600 temporary file in that same directory, write the complete JSON, sync and close it, then atomically rename over an allowed destination. Remove temporary output on failure. A failed write before rename, failed capture, cancellation or timeout preserves the old export. A deliberate successful renewal replaces it atomically. Do not print cookie values, raw exception objects, page content, request headers, or CDP URLs. Print concise fixed diagnostics and the success/output location only.

TS-LBC-CAP-002/004 cleanup: `finally` covers browser startup, navigation, export, success and handled errors. SIGINT and SIGTERM enter the same cleanup path and return nonzero (130/143 are appropriate); timeout/failure returns nonzero and success returns zero. Close the CDP/browser context, terminate the spawned process, await exit with a bounded grace period and escalate termination if necessary, then remove the owned temporary profile and any unfinished export. Prevent overlapping cleanup and prevent export after cancellation. Abrupt SIGKILL/power loss cannot be handled; document that leftover private temporary files may require removal, without treating that as a reusable profile feature.

## Shared import format (revision 1, history — **superseded**: no file is written)

Version 1 is a bounded UTF-8 JSON object (maximum 16 KiB), with exactly these fields:

```json
{
  "version": 1,
  "capturedAt": "2026-10-03T12:00:00.000Z",
  "cookie": {
    "name": "datadome",
    "value": "<secret; never include a real value in documentation>",
    "domain": ".leboncoin.fr",
    "path": "/",
    "secure": true,
    "expiresAt": "2027-10-03T12:00:00.000Z"
  }
}
```

`capturedAt` is UTC RFC3339 with fractional seconds permitted and refreshed on every capture; `expiresAt` is either a UTC RFC3339 timestamp or JSON null for a session cookie. Do not invent a lifetime for a session cookie. Cookie values must be nonempty, at most 4096 ASCII bytes, valid unquoted HTTP cookie values without whitespace, control bytes, quote, comma, semicolon or backslash. Allowed domain strings are `leboncoin.fr`, `.leboncoin.fr`, `www.leboncoin.fr`, `.www.leboncoin.fr`; a leading dot identifies domain scope, otherwise host-only scope. Preserve the browser's scope; a domain cookie still goes only to the two exact permitted hosts. Path must start with `/`, contain no control bytes and be applicable using normal cookie path matching. `secure` is a boolean; transport is always HTTPS regardless of its value. Reject expired browser cookies. Ignore HTTP-only/SameSite attributes in this nonbrowser transport format because they do not change the allowed request scope; do not export other attributes. Reject unknown versions, missing/wrong-type required fields, unknown fields, trailing JSON and invalid timestamps. The sidecar is a separate collector-owned format, not a capture output.

## Backend and API (revision 1, history)

The helper does not invoke the API or change SQLite. Collection-side format validation and use are in the [companion technical specification](../leboncoin-session-collection/technical.md). [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) stays unchanged; no request, response, status, endpoint or asynchronous behavior changes. Session material never enters a public API response.

## Operator documentation and release (revision 1, history)

TS-LBC-CAP-005: Document Node/browser prerequisites, an actual generic command with the investigation listing as an example rather than restriction, private directory creation, manual verification, timeout/cancel outcomes, and renewal. Distinguish browser capture from successful Go collection on the Pi.

Reuse `scripts/build-release.sh 6` / `scripts/deploy-armv6.sh user@host` for the application release; do not add a new remote deployment framework or require a remote host to build/test this feature. Document a separate protected SSH/scp session transfer: create a private remote staging directory first, transfer without pasting cookie contents, then install a mode-0600 temporary file owned by `pricefollower` into a mode-0700 `/var/lib/pricefollower/leboncoin` directory owned by that user; atomically rename to `session.json` on the same filesystem. Delete remote staging and temporary copies after installation. Use the operator's normal host-key verification and SSH permissions; never disable checking. Do not use a world-readable intermediate file.

One-time opt-in: an administrator creates a systemd drop-in containing `Environment=LEBONCOIN_SESSION_FILE=/var/lib/pricefollower/leboncoin/session.json`, runs daemon-reload and restarts the service. The existing installer leaves drop-ins in place. Subsequent atomic session replacement takes effect on the next collection without restart. Explain that `session.json.state.json` is collector-managed, must remain private, and should not be manually copied over a newer import. Document existing item refresh controls and safe service diagnostics. A 403/rejection requires repeating desktop verification and transfer; portability and duration are not guaranteed. Removing the opt-in setting and restarting restores sessionless operation.

## Scope, verification and unresolved questions (revision 1, history)

Permitted source scope: helper and focused test file, plus the operator guide and one minimal README link making that guide discoverable. No new npm script is required. No browser service, personal-profile product feature, dependency update, application UI change, automatic challenge solver or unrelated refactoring. This is new narrow behavior; no prerequisite refactoring is proposed.

Before implementation, record failing behavioral Node tests (an import/compile failure alone is not behavioral RED evidence) for URL validation, matching-page gating, cookie filtering, private/atomic export and lifecycle cleanup using injected narrow filesystem/browser/process collaborators where needed. Record actual RED then GREEN commands/results. Use synthetic secrets and temporary directories outside the repository; assert diagnostics do not disclose them. Exercise success, wrong ID, challenge with cookie, expired/missing/ambiguous cookie, cancellation, timeout, spawn/CDP failure, symlink/unsafe mode and preserved existing output. Execute actual CLI help/invalid input and visible-browser QA. A private task-owned verified investigation profile may seed a controlled QA fixture, but production behavior remains a fresh profile; do not report seeded QA as a freshly completed human challenge. Report a needed human interaction or unavailable display/network as a concrete limitation rather than claiming success.

No unresolved product choice or API approval is required for this handoff.

## Revision note (revision 2 amendments)

2026-10-03, review amendment REV-SET-001: status updated to record the API approval of 2026-10-03.
