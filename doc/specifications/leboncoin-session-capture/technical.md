# Technical specification: LeBoncoin session capture

Status: ready  
Functional source: [capture requirements](functional.md), FR-LBC-CAP-001–009.  
Companion: [collection design](../leboncoin-session-collection/technical.md).  
Evidence: [browser and collector investigation](../../changes/leboncoin-403-investigation/report.md).

## Requirement mapping

| Technical ID | Functional IDs | Design / intended files | Verification |
| --- | --- | --- | --- |
| TS-LBC-CAP-001 | CAP-001 | CLI URL validation and canonicalization in `scripts/capture-leboncoin-session.mjs` | Table-driven URL cases matching Go acceptance and canonicalization |
| TS-LBC-CAP-002 | CAP-002, CAP-003, CAP-007 | Spawn visible Chromium with private temporary profile; local CDP; bounded lifecycle | Stub process/CDP lifecycle tests; executed visible-browser QA |
| TS-LBC-CAP-003 | CAP-004, CAP-005 | Matching successful document and ad ID before extracting only applicable `datadome` | Challenge, wrong ID, absent cookie, successful fixture cases |
| TS-LBC-CAP-004 | CAP-006 | Private same-directory temporary write, sync, atomic rename | Mode, symlink, unsafe destination, failed-write and renewal tests |
| TS-LBC-CAP-005 | CAP-008, CAP-009 | `doc/leboncoin-session.md` capture, SSH installation, configuration, renewal instructions | Command review and isolated local installation/renewal QA |

`CAP-*` abbreviates `FR-LBC-CAP-*` in this table.

## Frontend

Not affected. No Carbon component, route, browser-upload form, login, or client state is introduced. Existing item refresh/status interfaces are used to check server acceptance. The helper supplies plain keyboard-usable terminal messages and a normal visible browser; the operator handles third-party verification.

## Desktop helper

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

## Shared import format

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

## Backend and API

The helper does not invoke the API or change SQLite. Collection-side format validation and use are in the [companion technical specification](../leboncoin-session-collection/technical.md). [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) stays unchanged; no request, response, status, endpoint or asynchronous behavior changes. Session material never enters a public API response.

## Operator documentation and release

TS-LBC-CAP-005: Document Node/browser prerequisites, an actual generic command with the investigation listing as an example rather than restriction, private directory creation, manual verification, timeout/cancel outcomes, and renewal. Distinguish browser capture from successful Go collection on the Pi.

Reuse `scripts/build-release.sh 6` / `scripts/deploy-armv6.sh user@host` for the application release; do not add a new remote deployment framework or require a remote host to build/test this feature. Document a separate protected SSH/scp session transfer: create a private remote staging directory first, transfer without pasting cookie contents, then install a mode-0600 temporary file owned by `pricefollower` into a mode-0700 `/var/lib/pricefollower/leboncoin` directory owned by that user; atomically rename to `session.json` on the same filesystem. Delete remote staging and temporary copies after installation. Use the operator's normal host-key verification and SSH permissions; never disable checking. Do not use a world-readable intermediate file.

One-time opt-in: an administrator creates a systemd drop-in containing `Environment=LEBONCOIN_SESSION_FILE=/var/lib/pricefollower/leboncoin/session.json`, runs daemon-reload and restarts the service. The existing installer leaves drop-ins in place. Subsequent atomic session replacement takes effect on the next collection without restart. Explain that `session.json.state.json` is collector-managed, must remain private, and should not be manually copied over a newer import. Document existing item refresh controls and safe service diagnostics. A 403/rejection requires repeating desktop verification and transfer; portability and duration are not guaranteed. Removing the opt-in setting and restarting restores sessionless operation.

## Scope, verification and unresolved questions

Permitted source scope: helper and focused test file, plus the operator guide and one minimal README link making that guide discoverable. No new npm script is required. No browser service, personal-profile product feature, dependency update, application UI change, automatic challenge solver or unrelated refactoring. This is new narrow behavior; no prerequisite refactoring is proposed.

Before implementation, record failing behavioral Node tests (an import/compile failure alone is not behavioral RED evidence) for URL validation, matching-page gating, cookie filtering, private/atomic export and lifecycle cleanup using injected narrow filesystem/browser/process collaborators where needed. Record actual RED then GREEN commands/results. Use synthetic secrets and temporary directories outside the repository; assert diagnostics do not disclose them. Exercise success, wrong ID, challenge with cookie, expired/missing/ambiguous cookie, cancellation, timeout, spawn/CDP failure, symlink/unsafe mode and preserved existing output. Execute actual CLI help/invalid input and visible-browser QA. A private task-owned verified investigation profile may seed a controlled QA fixture, but production behavior remains a fresh profile; do not report seeded QA as a freshly completed human challenge. Report a needed human interaction or unavailable display/network as a concrete limitation rather than claiming success.

No unresolved product choice or API approval is required for this handoff.
