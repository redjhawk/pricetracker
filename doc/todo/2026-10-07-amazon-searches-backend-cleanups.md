# Amazon searches backend cleanups

Status: todo (non-blocking PR review comments, triaged on 2026-10-07; not scheduled)

## Problem

- #84 `API_SPECIFICATION.md:626`: `INVALID_URL` message says "https://" but http is accepted.
- #85 `internal/store/amazon_requests.go:13`: `Blocked`/`blocked` vs "stopped" in API and specs.
- #86 `internal/amazon/collector.go:214,227`: `continue` where `break` is meant; regexp compiled on every call.
- #87 `internal/service/service.go:59`: `runningSearch` added before its use.
- #88 `internal/model/model.go:156`: `Review any` needs type assertions; #88 `internal/service/amazon_review.go:53`: store failure reported as an Amazon read failure.
- #89 `internal/service/searches.go:98`: unchecked type assertion; `searches.go:131`: "Tracked item was not found." message for search items.
- #90 `internal/httpapi/searches.go:79`: duplicated strict-JSON decode.

## Suggested work

Rename to "stopped", use `break`, hoist the regexp to a package variable, add a typed review accessor or checked assertion, fix the two messages, and extract a shared strict-JSON decode helper in a dedicated refactoring PR.

## Source

- PR reviews 5446171128 (#84), 5446171369 (#85), 5446171769 (#86), 5446174559 (#87), 5446172138 (#88), 5446175145 (#89), 5446172550 (#90)
