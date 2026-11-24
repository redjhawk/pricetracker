# No red phase for issue #195

The change is the test code itself: TC-002's fixture sanity check in `internal/amazon/browser_profile_test.go`. That file is locked after this stage, and FR-TEST-HYG-043 allows no other file outside `doc/`. So the development stage cannot write any of it. This stage delivers the whole change (TS-TEST-HYG-054), and no test can fail first on a missing implementation. The red gate is skipped. The development stage writes no code: it only sets the statuses.

## Delivered in `internal/amazon/browser_profile_test.go`

- `assertBrowserRefusesProfile` is split into `runBrowserOnProfile` (the run, steps and order unchanged, wait as a parameter) and `fixtureCheckFailure` (the judgement), with `lockRefusal(host)` = `on another computer (<host>)`. The check fails unless the page was not requested **and** the output holds `lockRefusal(host)`. Both values are read after the group kill and `<-exited`.
- TC-002 changes only at its call (new `"otherhost"` argument). The 15 s wait stays for the real browser.
- `refusalStub`, `refusalStubBrowser`, `TestBrowserProfile_Issue195_RefusalStubProcess` and `TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal` (ten stub rows, 2 s wait). Subtest names have the case ID as a prefix (`TC-001 …`, `TC-002 …`, `TC-003 …`) and otherwise follow the row names of the technical specification.

## Real refusal line

Recorded with a temporary `t.Log(run.output)` in `assertBrowserRefusesProfile` (reverted with `Edit`), during `go test ./internal/amazon -run 'TestBrowserProfile_TC002' -count=1 -v`:

```
[224:224:1124/092829.966982:ERROR:chrome/browser/process_singleton_posix.cc:365] The profile appears to be in use by another Chromium process (12345) on another computer (otherhost). Chromium has locked the profile so that it doesn't get corrupted. If you are sure no other processes are using this profile, you can unlock the profile and relaunch Chromium.
[224:224:1124/092829.967178:ERROR:chrome/browser/ui/dialogs/process_singleton_dialog_linux.cc:93] Cannot show profile-in-use dialog: no system dialog tool (zenity, kdialog, xmessage) was found. To recover, delete the files SingletonLock, SingletonCookie, SingletonSocket in your profile directory.
```

It contains `on another computer (otherhost)`, so `lockRefusal` and TS-TEST-HYG-049 stay as designed. The stub refusal line also stays as designed, because the real line has the same wording around the host.

## Runs

1. `go test ./internal/amazon -run 'TestBrowserProfile_Issue195' -count=1 -v`: all ten rows pass (about 4.1 s; the two `hold` rows take 2 s each).
2. `go test ./internal/amazon -run 'TestBrowserProfile_TC002' -count=1 -v`: PASS (2.27 s). With `-count=3`: PASS three times (2.28 s, 2.50 s, 2.49 s).
3. Fault injection (TS-TEST-HYG-053): the first `writeStaleLock(t, options.ProfileDir, "otherhost")` of TC-002 was removed and TC-002 was run once. It FAILED at the fixture check after the 15 s wait (15.92 s):

   ```
   browser_profile_test.go:545: fixture check: the browser opened the page on a profile locked by another computer; the browser output does not hold the lock refusal message "on another computer (otherhost)"; browser output (6635 bytes):
   ```

   followed by the browser's output (D-Bus, zygote and GCM errors, no refusal line). The line was then restored with `Edit`, and TC-002 passed again in the full run below.
4. Gates: `gofmt -l internal/amazon` prints nothing, `go vet ./...` passes, and `go test ./...` passes (all packages `ok`). `git diff --stat master -- . ':!doc'` lists only `internal/amazon/browser_profile_test.go`.

## Where each case is proved

| Test case | Proved by |
| --- | --- |
| TC-001 | Code check of `assertBrowserRefusesProfile`, `runBrowserOnProfile` and `fixtureCheckFailure`; rows `TC-001 refusal, then exit` and `TC-001 refusal, then still running at the end of the wait` |
| TC-002 | Rows `TC-002 exit at once without output`, `missing library`, `unknown option`, `slow start`, `generic lock error only` and `refusal naming another host`. Each checks the missing-message condition only, the exact output in the message, that the page was not requested and that the process group is gone (`ESRCH`) |
| TC-003 | Rows `TC-003 page requested with the refusal` (opened-page condition only) and `TC-003 page requested without output` (both conditions) |
| TC-004 | `TestBrowserProfile_TC002_LockOfAnotherHostDoesNotPreventTheCheck`, runs 2 and 4 |
| TC-005 | Fault injection, run 3 |
| TC-006 | Repository checks and gates, run 4. The development stage runs them again |
