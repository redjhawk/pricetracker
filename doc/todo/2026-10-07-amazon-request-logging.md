# Clearer Amazon request logs

Status: todo (non-blocking PR review comments, triaged on 2026-10-07; not scheduled)

## Problem

- #86 `internal/amazon/collector.go:131`: the real failure reason of product requests is replaced by a generic message, so the gate log loses it.
- #87 `internal/service/amazon_gate.go:32`: if loading the gate state fails, a persisted stop is lost with only a generic log.
- #87 `internal/service/service.go:316`: a tracked check skipped because requests are stopped is logged tersely.
- #89 `internal/service/search_worker.go:213`: only "close" is logged; FR-AMZ-HUMAN-004 expects open, read and close in the logs.

## Suggested work

Keep the underlying error in the collector message; log open and read steps in `openItem`; make the gate load failure explicit (fail closed or warn clearly).

## Source

- PR reviews 5446171769 (#86), 5446174559 (#87), 5446175145 (#89); see `doc/changes/2026-10-07-1117-issue-56-amazon-searches/pr-triage.md`
