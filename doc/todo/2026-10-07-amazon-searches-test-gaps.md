# Amazon searches test gaps

Status: todo (non-blocking PR review comments, triaged on 2026-10-07; not scheduled)

## Problem

- #85 `internal/store/searches.go:173`: `CaptureSearch` relies on unique canonical URLs, not documented or tested.
- #88 `internal/service/review_test.go:48`: Amazon fake records a LeBoncoin input type.
- #90 `internal/service/searches_test.go:56`: error from `SetSearchProgress` ignored.
- #90 `internal/service/search_worker_test.go:110`: no test for a failed first results page (the failed-capture case is covered by the blocking #89 fix).
- #91 `internal/httpapi/searches_test.go:104`: zero-time `stoppedAt` fixture the API never returns.

## Suggested work

Add the missing tests, check the ignored error, and use realistic fixtures.

## Source

- PR reviews 5446171369 (#85), 5446172138 (#88), 5446172550 (#90), 5446175748 (#91)
