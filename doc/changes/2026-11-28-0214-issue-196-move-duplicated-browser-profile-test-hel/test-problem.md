# Locked test problem: TC-008 row 2 is timing-sensitive (review R1-02)

**Test:** `internal/amazon/amazontest/browser_profile_test.go`, `TestKillBrowserOnProfile_Issue196_TC008_LeftoverAndMissingLock`, subtest `lock removed before the kill` (around line 483). The file is in `tests.lock`.

**Problem:** The stub browser keeps `SingletonLock` for only `sleep 0.1`. `KillBrowserOnProfile` polls `os.Lstat` every 20 ms. On a loaded runner, where `go test ./...` runs packages in parallel alongside Chromium tests, a scheduling stall of about 100 ms makes the helper miss the lock. It then waits the full 30 s and fails with `the browser did not create SingletonLock in the profile` instead of the expected `precondition: ...` message. The `go test ./...` gate therefore fails intermittently.

The implementation can't fix this. Polling faster only narrows the window, and the helper has to keep the 20 ms poll, the 30 s timeout and the 500 ms settle wait that TC-005 to TC-008 specify.

**Fix needed (test stage):**

1. In `testcases.md`, TC-008 "Given" row 2: replace "removes it about 100 ms later (within the 500 ms wait)" with "removes it about 300 ms later (within the 500 ms wait)".
2. In the test, change the stub script of the `lock removed before the kill` subtest from `sleep 0.1` to `sleep 0.3`:

   ```go
   stub, _ := writeStub(t, "ln -s \"stub-$$\" \"$profile/SingletonLock\"\nsleep 0.3\nrm -f \"$profile/SingletonLock\"\nwhile :; do sleep 0.1; done")
   ```

The removal still happens before the helper's 500 ms settle sleep ends, so the subtest still proves the case, and the detection window becomes three times wider.
