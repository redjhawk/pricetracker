# Issue #180 Clear stale Chromium Singleton files in the persistent Amazon browser profile

**Problem:** `BrowserCollector` reuses `<data dir>/amazon-browser` (`internal/amazon/browser.go:216-226`, `profile()` at line 240). If Chromium is killed (for example `close()` SIGKILLs the process group after 5 s, a crash, or a stop during a check), `SingletonLock`/`SingletonSocket`/`SingletonCookie` stay in the profile. If the data directory is later used under another hostname (moved server, new container), Chromium refuses the profile ("Failed to create a ProcessSingleton"). Every Amazon check then fails with "browser could not start", and after 5 failures all Amazon items are stopped.

**Suggested work:** `BrowserCollector.mu` already ensures one browser per profile in this process. Before `launchBrowser`, when `ProfileDir` is set, remove `SingletonLock`, `SingletonSocket` and `SingletonCookie` with `os.Remove`, ignoring `fs.ErrNotExist`. Add a browser test that creates a `SingletonLock` symlink to `otherhost-12345` in the profile and checks that the next check still succeeds.

From review comment R1-02 on #179 (`doc/todo/issue-168-r1-02.md`).

## Comments by trusted authors

