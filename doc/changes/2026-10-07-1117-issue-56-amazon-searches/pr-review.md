# PR review (stage 9) — issue #56

Reviewer: independent PR reviewer agent. Each PR was reviewed on its own diff against its stacked base. `REQUEST_CHANGES` was refused with HTTP 422 because the PRs are our own, so the blocking reviews were posted as `COMMENT` reviews whose body starts with `CHANGES REQUESTED`. The comment IDs could not be read back because the tooling did not allow the listing call, so this record uses the review IDs.

| PR | Reviewed head SHA | Event | Review | bug | other |
| --- | --- | --- | --- | --- | --- |
| #83 | 17b0b8750b264d3030bdb3634135304d15061d2e | COMMENT (no findings) | 5446170963 | 0 | 0 |
| #84 | 7fe01c2b13de4e1e87e390d9e95a8d969564ac37 | COMMENT | 5446171128 | 0 | 1 |
| #85 | d4f92f37208f3596b746d64bf6291ef5c53506c7 | COMMENT | 5446171369 | 0 | 2 |
| #86 | ea33d2b1f1f5c9a5dccead07fa14ac8631fdbd47 | COMMENT | 5446171769 | 0 | 3 |
| #87 | f1a4ab0f2d2e6ed6e78ff132f909a25b2523f15f | COMMENT, CHANGES REQUESTED (REQUEST_CHANGES refused, 422) | 5446174559 | 1 | 3 |
| #88 | 7e27202d7d5a24513c08ef1d8175f5dac7a929e9 | COMMENT | 5446172138 | 0 | 3 |
| #89 | 4d2900f0916a740f3d0771a780e79185942fa377 | COMMENT, CHANGES REQUESTED (422) | 5446175145 | 2 | 3 |
| #90 | bd6c5ae9d0c3113405f2d11fd6357e22a742a036 | COMMENT | 5446172550 | 0 | 3 |
| #91 | ca1b4260815d966cb35d824b74e4ae4d189f9cb1 | COMMENT, CHANGES REQUESTED (422) | 5446175748 | 1 | 3 |
| #92 | 0b5abb0a88ac1f7454ab15b390de751c40079f0a | COMMENT, CHANGES REQUESTED (422) | 5446176333 | 2 | 3 |

Category key: bug, simplification, readability, refactoring. `[test]` and `[nit]` labels are listed under readability unless noted.

## #83
- No findings.

## #84
- `API_SPECIFICATION.md:626`, readability (nit): the `INVALID_URL` message says "starting with https://", but `http` URLs are also accepted.

## #85
- `internal/store/amazon_requests.go:13`, readability: the code and database use `Blocked`/`blocked`, but the API and specs say "stopped".
- `internal/store/searches.go:173`, readability (test): `CaptureSearch` relies on unique canonical URLs, but neither documents nor tests that rule.

## #86
- `internal/amazon/collector.go:131`, readability: the real failure reason is replaced by the generic message, so the gate log loses it for product requests.
- `internal/amazon/collector.go:214`, readability (nit): the feature budget uses `continue` where `break` was meant.
- `internal/amazon/collector.go:227`, simplification: the regexp is compiled on every call.

## #87
- `internal/service/amazon_gate.go:49`, **bug**: a request cancelled by shutdown counts as a consecutive Amazon failure, so deploys can stop all Amazon requests.
- `internal/service/amazon_gate.go:32`, readability (nit): if loading the state fails, a persisted stop is silently lost.
- `internal/service/service.go:59`, readability (nit): `runningSearch` is not used until #89.
- `internal/service/service.go:316`, readability (nit): FR-AMZ-HUMAN-009 says tracked checks must not be skipped silently, but this check is skipped silently.

## #88
- `internal/model/model.go:156`, readability: `Review any` leads to type assertions; a typed accessor is suggested.
- `internal/service/amazon_review.go:53`, readability (nit): a store failure is reported as "product could not be read from Amazon".
- `internal/service/review_test.go:48`, readability (nit): the Amazon fake records a LeBoncoin input type.

## #89
- `internal/service/search_worker.go:73`, **bug**: the loop runs with no pause when `runPass` makes no progress, for example on a database error.
- `internal/service/search_worker.go:166`, **bug**: a failed capture is treated as a success; there are no items and no `lastError` until the next window.
- `internal/service/searches.go:98`, readability: the type assertion is unchecked and can panic.
- `internal/service/search_worker.go:213`, readability (nit): only "close" is logged, but FR-AMZ-HUMAN-004 expects open, read and close.
- `internal/service/searches.go:131`, readability (nit): the message "Tracked item was not found." is misleading here.

## #90
- `internal/httpapi/searches.go:79`, refactoring (nit): the strict-JSON decode is duplicated; a shared helper belongs in a dedicated PR.
- `internal/service/searches_test.go:56`, readability (test): the error from `SetSearchProgress` is ignored.
- `internal/service/search_worker_test.go:110`, readability (test): no test covers a failed first results page or a failed capture.

## #91
- `src/components/AmazonSearchesPage.tsx:95`, **bug**: the delete modal stays enabled during the request; a double submit shows a spurious "not found" error.
- `src/components/AmazonSearchesPage.tsx:130`, readability (nit): `addError` is not cleared while the user edits the URL.
- `src/components/AmazonSearchesPage.tsx:46`, simplification: `errorMessage` is duplicated in three files.
- `internal/httpapi/searches_test.go:104`, readability (test): the zero-time `stoppedAt` fixture is a value the API never returns.

## #92
- `src/components/AmazonSearchItems.tsx:172`, **bug**: there is no link to the Amazon listing, which FR-AMZ-SEARCH-006 requires.
- `src/App.tsx:310`, **bug**: "Back to Amazon searches" goes to `/`, which shows the Amazon tab after a reload or deep link; an item opened from a search does not return to that search.
- `src/components/AmazonSearchItems.tsx:18`, readability: formatters are imported from another page component.
- `src/components/AmazonSearchItems.tsx:29`, simplification: the rating labels and euro formatter are duplicated.
- `src/components/TrackedItemsPage.tsx:65`, readability (nit): the effect is never cancelled and refetches on every poll.
