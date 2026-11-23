# Issue #217 Deny downloads in the Amazon browser and make the technical spec precise about it

**Problem:** `internal/amazon/browser.go` never sets a download behaviour. With `--headless=new`, an Amazon answer with `Content-Disposition: attachment` (now a `request_error`, issue #186) may still be written to disk (profile or `$HOME/Downloads`) on every such check. `doc/specifications/amazon-human-checks/technical.md:205` says "a failed download leaves nothing for the next check", but no test proves it for the disk.

**Suggested work:**
- When the browser starts (in the launch/target set-up steps of `browserTab`/`BrowserCollector`), call `Browser.setDownloadBehavior{behavior: "deny"}`.
- Extend `TestBrowserNavigation_TC001_*` (`internal/amazon/browser_navigation_test.go`) to assert that no file appears in the profile directory or in a temporary `HOME`'s `Downloads` after the attachment check.
- Reword the sentence in technical.md (section "Success response not opened as a page (issue #186)", Backend, last bullet) to describe the actual guarantee, and move the download behaviour out of "Out of scope".

From review comment R1-01 on #216 (`doc/todo/issue-186-r1-01.md`).

## Comments by trusted authors

