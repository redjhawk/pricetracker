# Blocking review comments (round 1)

## R1-01 `internal/amazon/cdp.go:112`

The browser's own reason can be cut off at exactly the part that matters. `exitError` puts up to 2,048 bytes of stderr tail (stderrTailBytes) after the `browser could not start: the browser exited (<status>): ` prefix. `requestFailure` then passes the text through `logText`, which keeps the first 1,024 bytes and drops the end. The kept part is the oldest part of the tail; the newest output, usually the fatal line, is discarded. Example: Chromium prints DBus/GPU warnings and then `Running as root without --no-sandbox is not supported`. If the warnings fill more than about 970 bytes, the logged cause shows only warnings plus `…` and loses the reason FR-AMZ-BRET-005 asks for. TC-006's multi-line case misses this because it only looks for `stub line`, which appears everywhere in the output. Fix: keep the newest output. For example, drop the start of the tail in `exitError` so that prefix + tail fits in 1,024 bytes (cut on a rune boundary), or set stderrTailBytes below the log budget. Add a test where the last stderr line must appear in the cause after more than 1 KB of earlier output.

Triage: Confirmed: exitError (internal/amazon/cdp.go:112) appends up to 2,048 bytes of stderr tail (stderrTailBytes, cdp.go:40), and logText (internal/amazon/browser.go:93) keeps only the first 1,024 bytes, so the newest stderr lines (normally the fatal reason) are dropped when earlier output exceeds about 1 KB. That breaks FR-AMZ-BRET-005 ('with the browser's own reason'). The TC-006 multi-line subtest (browser_retrieval_test.go:604-615) writes 40 lines of about 85 bytes but only checks for 'stub line', so it passes while losing the last line and does not prove the case.

## R1-02 `src/components/InformationModal.tsx.new:1`

Stray editing artifact committed: a fragment of `InformationModal.tsx`, starting at `return (` without its imports or function header. It is not imported, not valid TSX on its own, and not part of the specification. Delete the file.

Triage: Confirmed: src/components/InformationModal.tsx.new was added in commit 64f47f6 (implement round 0). It is a headless fragment starting at `return (`, imported nowhere, and not part of the specification. It is a stray artifact mixed into this PR and must be deleted.

