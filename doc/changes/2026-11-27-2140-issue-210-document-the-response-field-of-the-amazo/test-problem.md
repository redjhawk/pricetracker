# Locked test problem: `responseSentenceError` is too weak (blocking comment R1-01)

The blocking comment R1-01 is about the locked test file `internal/service/amazon_failure_log_docs_test.go`, not about the implementation. The developer stage cannot change locked test files, so it cannot fix this comment.

## Problem

`responseSentenceError` (`internal/service/amazon_failure_log_docs_test.go:161-179`) accepts any sentence after the failure line that contains the separate words `response`, `start`, `page` and `empty`. In the failed-check item, `start` and `page` also come from the cause vocabulary ("could not start", "the page could not be opened"). So a sentence such as:

> `response` can be empty, for example when the browser could not start or the page did not load.

passes, although it does not say that `response` is the start of the page Amazon returned. TC-002's corner case "sentence does not say it is the start of the page Amazon returned: fails" is therefore not proven. The existing "no start of page" fixture fails only because it drops both words.

## Requested test change

1. In `responseSentenceError`, replace `strings.Contains(lower, "start") && strings.Contains(lower, "page")` with `strings.Contains(lower, "start of the page")`, keeping the `response` and `empty` checks.
2. Add a rejecting TC-002 fixture whose sentence after the failure line contains "could not start" and "page" but not "start of the page", for example the sentence above, and assert that `responseSentenceError` returns an error for it.

## Implementation status

The documented sentence in `deploy/README.md:56` already satisfies the stricter check:

> `response` is the start of the page Amazon returned; it can be empty, for example when the browser could not start.

No implementation change is needed once the test is tightened.
