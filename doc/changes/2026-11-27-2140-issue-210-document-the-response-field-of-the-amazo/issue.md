# Issue #210 Document the response field of the Amazon failure log line

**Problem:** `deploy/README.md` line 52 documents the failure line as `Amazon request failed url=… at=… status=… cause=…`. `amazonGate.do` (`internal/service/amazon_gate.go:64`) logs `… cause=%s response=%s`, where `response` is the start of the page Amazon returned (`result.ResponseExcerpt`). An operator parsing the documented form may read the HTML excerpt as part of the cause.

**Suggested work:** Change the documented line to `Amazon request failed url=… at=… status=… cause=… response=…` and add one sentence saying that `response` is the start of the page Amazon returned (it can be empty). File: `deploy/README.md:52`.

From review comment R2-03 on #206 (`doc/todo/issue-204-r2-03.md`).

## Comments by trusted authors

