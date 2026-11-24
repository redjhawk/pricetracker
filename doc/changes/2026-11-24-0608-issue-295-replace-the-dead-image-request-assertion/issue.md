# Issue #295 Replace the dead image-request assertion in the TC-010 e2e test

**Problem:** In `tests/platform-tabs.spec.ts:361`, `expect(imageRequests.every((url) => url === imageUrl)).toBe(true)` always passes, because the route handler (lines 327-328) only pushes `imageUrl` into `imageRequests`. Foreign image requests are already caught through `api.unexpected` (line 332).

**Suggested work:** Replace the line with `expect(imageRequests.length).toBeGreaterThan(0)` so the test proves A's accepted image was actually requested, or delete it.

**File:** `tests/platform-tabs.spec.ts:361`

From review comment R1-01 on #294 (`doc/todo/issue-275-r1-01.md`).

## Comments by trusted authors

