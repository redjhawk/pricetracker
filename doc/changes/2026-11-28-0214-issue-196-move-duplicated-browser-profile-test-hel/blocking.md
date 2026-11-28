# Blocking review comments (round 1)

## R1-02 `internal/amazon/amazontest/browser_profile_test.go:483`

TC-008 row 2 ('lock removed before the kill') is timing-sensitive. The stub keeps SingletonLock for only `sleep 0.1`, and KillBrowserOnProfile polls Lstat every 20 ms plus scheduling delay. On a loaded runner (go test ./... runs packages in parallel with Chromium tests), the helper can miss that window. It then waits the full 30 s and fails with 'the browser did not create SingletonLock in the profile', so the test fails intermittently and takes 30 s. The removal only has to happen before the helper's 500 ms settle sleep ends. Any delay below 0.5 s still proves the case, so use `sleep 0.3` (or 0.4) to widen the window several times.

Triage: At internal/amazon/amazontest/browser_profile_test.go:483, the stub keeps SingletonLock for only `sleep 0.1`. KillBrowserOnProfile polls Lstat every 20 ms, so a scheduling stall of about 100 ms makes it miss the lock. The test then waits the full 30 s and fails with the wrong message ('did not create SingletonLock'). That makes the `go test ./...` gate intermittently fail. The removal only has to happen before the 500 ms settle sleep ends, so a longer delay (for example `sleep 0.3`) still proves the case with a much wider margin.

