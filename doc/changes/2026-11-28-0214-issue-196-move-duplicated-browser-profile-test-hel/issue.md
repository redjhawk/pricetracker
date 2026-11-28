# Issue #196 Move duplicated browser-profile test helpers into internal/amazon/amazontest

**Problem:** Three test packages carry their own copies of the same fixtures:
- stale-lock writers: `writeStaleLock` (`internal/amazon/browser_profile_test.go`), `leaveStaleLock` (`internal/service/amazon_browser_profile_test.go`, around line 75), `leaveOtherHostLock` (`internal/httpapi/amazon_browser_profile_test.go`), all with the same three symlink targets;
- kill helpers: `killBrowserOnProfile` / `killBrowser` and the duplicated `/proc` scan `profileProcesses`;
- fixture servers: `profileServer` / `browserProfileServer`, with the same cookie, aod and static handlers.

**Suggested work:** Add `StaleLock(t, profile, host)`, `KillBrowserOnProfile(t, profile)` and `BrowserProcesses(profile)`, plus a shared profile fixture handler if practical, to `internal/amazon/amazontest` (which already provides `Browser`, `Fixture` and `NewServer`). Then replace the local copies in the three test files.

From review comment R1-03 on #194 (`doc/todo/issue-180-r1-03.md`).

## Comments by trusted authors

