# Blocking review comments (round 2)

## R2-01 `internal/amazon/cdp.go:112`

Blocking R1-01 is still open: the round-1 commit (a19c068) only deleted blocking.md and changed state.json, and no code changed. `exitError` still adds the whole 2,048-byte stderr tail (stderrTailBytes, line 40) after the `browser could not start: the browser exited (<status>): ` prefix. `logText` (browser.go:93-111) then keeps the first 1,024 bytes and cuts the end. The newest stderr lines, which usually hold the fatal reason Chromium prints last (for example `Running as root without --no-sandbox is not supported` after DBus or GPU warnings), are dropped. That breaks FR-AMZ-BRET-005 ('with the browser's own reason'). Fix: keep the newest output. In `exitError`, drop the start of the tail so that prefix + status + tail fits in the 1,024-byte log budget, cutting on a rune boundary, for example with a leading `…`. Another option is to have `logText` keep the end of the text for browser exit reasons. The locked TC-006 subtest 'browser writing several lines' (browser_retrieval_test.go:604-615) only checks for `stub line`, so it passes either way. It cannot prove this case. Ask for a non-blocking follow-up test that requires the last stderr line (`stub line 40`) in the cause.

Triage: Confirmed and unchanged since round 1. exitError (internal/amazon/cdp.go:111-112) appends up to 2,048 bytes of stderr tail (stderrTailBytes, cdp.go:40). requestFailure (browser.go:174) passes the reason through logText (browser.go:93-111), which keeps the first 1,024 bytes and cuts the end. When Chromium writes more than about 1 KB before its fatal line, that newest line is lost, which breaks FR-AMZ-BRET-005 ('with the browser's own reason') and TS-AMZ-BRET-005. The TC-006 multi-line subtest only checks for 'stub line', so it does not prove the case.

## R2-02 `src/components/InformationModal.tsx.new:1`

Blocking R1-02 is still open: the stray editing artifact `src/components/InformationModal.tsx.new` is still in the diff. It is a 35-line fragment that starts at `return (`, with no imports or function header. Nothing imports it, it is not valid TSX on its own, and it duplicates the body now in `InformationModal.tsx`. It is not part of the specification. Delete the file.

Triage: Confirmed: src/components/InformationModal.tsx.new still exists. It is a 35-line fragment starting at `return (` with no imports or function header, nothing imports it, and it is not part of the issue #204 specification. It is unrelated stray work mixed into this PR and must be deleted.

