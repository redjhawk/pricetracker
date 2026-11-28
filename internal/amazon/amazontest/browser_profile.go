package amazontest

import (
	"net/http"
	"testing"
)

// Browser-profile helpers of issue #196. Stubs of the TDD red stage: not implemented yet.

// StaleLock leaves in profile the lock marks of a browser of host that did not end normally.
func StaleLock(t testing.TB, profile, host string) {
	t.Helper()
}

// BrowserProcesses lists the running processes whose command line contains profile.
func BrowserProcesses(profile string) []string {
	return nil
}

// KillBrowserOnProfile starts the browser on profile in its own process group, waits until it holds
// the profile, kills the group and checks that the lock is left behind.
func KillBrowserOnProfile(t testing.TB, profile string) {
	t.Helper()
}

// ProfileHandler serves the profile fixture pages of the issue #180 tests.
func ProfileHandler(t testing.TB) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not implemented", http.StatusNotImplemented)
	})
}
