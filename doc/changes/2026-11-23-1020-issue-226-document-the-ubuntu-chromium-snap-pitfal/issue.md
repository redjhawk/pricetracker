# Issue #226 Document the Ubuntu Chromium snap pitfall for Amazon checks

**Problem:** On Ubuntu, `chromium` and `chromium-browser` are usually the snap Chromium (`/snap/bin/chromium` or the transitional wrapper). The lookup order in `amazon.BrowserNames` (`internal/amazon/browser.go:53`) puts these names before `google-chrome-stable`. So a server that already has the snap uses it, not the Google Chrome that `deploy/README.md:45` tells the operator to install. Snap confinement can stop it from using a profile under a system data directory, or from running as a system service user, and then every Amazon check fails with `browser could not start`.

**Suggested work:** In the `Browser for Amazon checks` section of `deploy/README.md` (around lines 45-47), add one sentence: when a Chromium snap is installed, set `PRICEFOLLOWER_BROWSER_PATH` to the Chrome executable (for example /usr/bin/google-chrome-stable). Do not put that path in a bare-name backtick token, so that the TC-011 documentation test (code and documentation name the same executables) still passes. Run `go test ./...` afterwards.

From review comment R4-01 on #206 (`doc/todo/issue-204-r4-01.md`).

## Comments by trusted authors

