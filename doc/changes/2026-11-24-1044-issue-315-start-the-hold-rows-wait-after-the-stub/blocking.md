# Blocking review comments (round 1)

## R1-01 `internal/amazon/browser_profile_test.go:546`

FR-TEST-HYG-049 and TC-003 say that when the 2 s after the output signal end, both hold stubs are still running ('the row kills it; it has not exited by itself'). The test never checks this. In the hold wait, `case <-exited:` ends the run early without a trace, and the row then passes as long as the output matches. For 'TC-002 slow start' in particular, a stub that signalled and then exited (for example a crash, or a future edit of the stub) gives exactly the expected result: empty output, no request and the `missing` failure. So the row's meaning, a stub still running when the wait ends, is not proved. no-tests.md cites only the `ESRCH` check for TC-003, but that check passes whether the stub was killed or exited by itself. Suggested fix: record which way the hold wait ended, for example `held := false` set in the `time.After(2 * time.Second)` case. When a signalled stub exited before the 2 s, report `t.Errorf("the stub exited before the end of the wait")`.

Triage: Confirmed: in the hold branch of the t.Run loop (browser_profile_test.go:544-549), `case <-exited:` ends the 2 s hold early without any trace, and the ESRCH check passes whether the stub was killed or exited by itself. So TC-003 ('still running (the row kills it; it has not exited by itself)', FR-TEST-HYG-049), which no-tests.md claims is proved, is not: a 'TC-002 slow start' stub that signals and then exits gives exactly the expected result. The test does not prove its case.

