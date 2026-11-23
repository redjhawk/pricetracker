# Issue #232 Make AssertNoDownload fail on unreadable files

**Problem:** `downloadFile` in `internal/amazon/amazontest/amazontest.go` (around line 239) ignores the error from `os.ReadFile`. A file that cannot be read (permission denied, I/O error) is therefore treated as "not a download", even if it holds `DownloadMarker`. The evidence for FR-AMZ-HCHK-014 (TC-001 to TC-004) then fails open.

**Suggested work:** Change `downloadFile` to return `(string, error)`. In the `filepath.WalkDir` callback of `AssertNoDownload`, return any read error other than `fs.ErrNotExist`, so `AssertNoDownload` reports it with `t.Fatalf` and does not skip the file. Keep ignoring files that disappear between listing and reading.

From review comment R1-02 on #231 (`doc/todo/issue-217-r1-02.md`).

## Comments by trusted authors

