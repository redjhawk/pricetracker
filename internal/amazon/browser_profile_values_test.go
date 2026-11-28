package amazon

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pricefollower.local/internal/amazon/amazontest"
)

// Explicit zero values of issue #197: profile() returns "", nil and the error when the stale lock
// marks of a persistent profile cannot be removed, as it does when the directory cannot be created.

// assertZeroValues checks the result of a profile() call that failed.
func assertZeroValues(t *testing.T, path string, cleanup func(), err error) {
	t.Helper()
	if path != "" {
		t.Errorf("path %q, want an empty path", path)
	}
	if cleanup != nil {
		t.Errorf("cleanup is not nil")
	}
	if err == nil {
		t.Errorf("no error")
	}
}

// writeUndiscardableLock makes name in profile a non-empty directory, which os.Remove cannot remove.
func writeUndiscardableLock(t *testing.T, profile, name string) {
	t.Helper()
	writeProfileFile(t, filepath.Join(profile, name, "keep"), "directory content")
}

func assertFileContent(t *testing.T, path, content string) {
	t.Helper()
	if data, err := os.ReadFile(path); err != nil || string(data) != content {
		t.Errorf("%s changed: %q %v, want %q", path, data, err, content)
	}
}

func assertLocksAbsent(t *testing.T, profile string) {
	t.Helper()
	for _, name := range profileLockNames {
		if _, err := os.Lstat(filepath.Join(profile, name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s is still in the profile: %v", name, err)
		}
	}
}

func profileOf(directory string) (string, func(), error) {
	return NewBrowserCollector(BrowserOptions{ProfileDir: directory}).profile()
}

// TC-001 of issue #197
func TestBrowserProfile_Issue197_TC001_UndiscardableLockGivesZeroValuesAndTheError(t *testing.T) {
	for _, name := range profileLockNames {
		t.Run("TC-001 "+name+" cannot be discarded", func(t *testing.T) {
			profile := filepath.Join(t.TempDir(), "profile")
			writeUndiscardableLock(t, profile, name)
			writeProfileFile(t, filepath.Join(profile, "Default", "Cookies"), "cookie database fixture")
			collector := NewBrowserCollector(BrowserOptions{ProfileDir: profile})

			for call := 1; call <= 2; call++ {
				path, cleanup, err := collector.profile()
				assertZeroValues(t, path, cleanup, err)
				if err == nil {
					t.Fatalf("call %d: no error", call)
				}
				if want := "remove stale profile lock " + name; !strings.Contains(err.Error(), want) {
					t.Errorf("call %d: error %q does not contain %q", call, err, want)
				}
				var pathErr *fs.PathError
				if !errors.As(err, &pathErr) {
					t.Errorf("call %d: error %q does not wrap the file-system error", call, err)
				}
				if info, statErr := os.Stat(profile); statErr != nil || !info.IsDir() {
					t.Fatalf("call %d: the profile directory is gone: %v", call, statErr)
				}
				assertFileContent(t, filepath.Join(profile, name, "keep"), "directory content")
				assertFileContent(t, filepath.Join(profile, "Default", "Cookies"), "cookie database fixture")
			}
		})
	}

	for _, first := range []string{"absent", "symlink"} {
		t.Run("TC-001 SingletonLock "+first+" and SingletonSocket cannot be discarded", func(t *testing.T) {
			profile := filepath.Join(t.TempDir(), "profile")
			if err := os.MkdirAll(profile, 0o700); err != nil {
				t.Fatal(err)
			}
			if first == "symlink" {
				if err := os.Symlink("otherhost-12345", filepath.Join(profile, "SingletonLock")); err != nil {
					t.Fatal(err)
				}
			}
			writeUndiscardableLock(t, profile, "SingletonSocket")

			path, cleanup, err := profileOf(profile)
			assertZeroValues(t, path, cleanup, err)
			if err != nil && !strings.Contains(err.Error(), "remove stale profile lock SingletonSocket") {
				t.Errorf("error %q does not name SingletonSocket", err)
			}
			if _, err := os.Lstat(filepath.Join(profile, "SingletonLock")); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("SingletonLock is still in the profile: %v", err)
			}
			assertFileContent(t, filepath.Join(profile, "SingletonSocket", "keep"), "directory content")
		})
	}
}

// TC-002 of issue #197
func TestBrowserProfile_Issue197_TC002_BothFailureBranchesReturnTheSameForm(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "file")
	writeProfileFile(t, parent, "regular file")
	uncreatable := filepath.Join(parent, "profile")
	path, cleanup, mkdirErr := profileOf(uncreatable)
	t.Run("TC-002 profile directory cannot be created", func(t *testing.T) {
		assertZeroValues(t, path, cleanup, mkdirErr)
		if info, err := os.Lstat(parent); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("the parent file changed: %v %v", info, err)
		}
		assertFileContent(t, parent, "regular file")
		if _, err := os.Lstat(uncreatable); err == nil {
			t.Errorf("%s was created", uncreatable)
		}
	})

	locked := filepath.Join(t.TempDir(), "profile")
	writeUndiscardableLock(t, locked, "SingletonLock")
	path, cleanup, lockErr := profileOf(locked)
	t.Run("TC-002 stale lock cannot be removed", func(t *testing.T) {
		assertZeroValues(t, path, cleanup, lockErr)
	})

	if mkdirErr != nil && lockErr != nil && mkdirErr.Error() == lockErr.Error() {
		t.Errorf("both branches give the same error %q, want different errors", mkdirErr)
	}
}

// TC-003 of issue #197
func TestBrowserProfile_Issue197_TC003_PreparedProfileReturnsItsPath(t *testing.T) {
	assertPrepared := func(t *testing.T, profile string) {
		t.Helper()
		for call := 1; call <= 2; call++ {
			path, cleanup, err := profileOf(profile)
			if err != nil {
				t.Fatalf("call %d: error %v", call, err)
			}
			if path != profile {
				t.Errorf("call %d: path %q, want %q", call, path, profile)
			}
			if cleanup == nil {
				t.Fatalf("call %d: cleanup is nil", call)
			}
			cleanup()
			cleanup()
			if info, err := os.Stat(profile); err != nil || !info.IsDir() {
				t.Fatalf("call %d: the profile directory is gone after the cleanup: %v", call, err)
			}
			assertLocksAbsent(t, profile)
		}
	}

	t.Run("TC-003 stale lock and cookies", func(t *testing.T) {
		profile := filepath.Join(t.TempDir(), "profile")
		amazontest.StaleLock(t, profile, "otherhost")
		writeProfileFile(t, filepath.Join(profile, "Default", "Cookies"), "cookie database fixture")
		assertPrepared(t, profile)
		assertFileContent(t, filepath.Join(profile, "Default", "Cookies"), "cookie database fixture")
	})

	t.Run("TC-003 existing profile without lock", func(t *testing.T) {
		profile := filepath.Join(t.TempDir(), "profile")
		writeProfileFile(t, filepath.Join(profile, "Local State"), `{"profile":{"fixture":197}}`)
		assertPrepared(t, profile)
		assertFileContent(t, filepath.Join(profile, "Local State"), `{"profile":{"fixture":197}}`)
	})

	t.Run("TC-003 profile that does not exist yet", func(t *testing.T) {
		profile := filepath.Join(t.TempDir(), "profile")
		assertPrepared(t, profile)
		info, err := os.Stat(profile)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Errorf("profile directory mode %v, want 0700", info.Mode().Perm())
		}
	})

	t.Run("TC-003 SingletonLock pointing to a file outside the profile", func(t *testing.T) {
		profile := filepath.Join(t.TempDir(), "profile")
		target := filepath.Join(t.TempDir(), "lock-target")
		writeProfileFile(t, target, "lock target content")
		if err := os.MkdirAll(profile, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(profile, "SingletonLock")); err != nil {
			t.Fatal(err)
		}
		assertPrepared(t, profile)
		assertFileContent(t, target, "lock target content")
	})

	t.Run("TC-003 temporary profile", func(t *testing.T) {
		path, cleanup, err := profileOf("")
		t.Cleanup(func() { // the temporary profile is removed however the subtest ends, after its checks (issue #324)
			if filepath.Dir(path) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(path), "pricefollower-browser-") {
				return
			}
			if err := os.RemoveAll(path); err != nil {
				t.Errorf("remove the temporary profile: %v", err)
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if cleanup == nil {
			t.Fatal("cleanup is nil")
		}
		if filepath.Dir(path) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(path), "pricefollower-browser-") {
			t.Errorf("temporary profile %q, want a new directory under %s", path, os.TempDir())
		}
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Fatalf("the temporary profile was not created: %v", err)
		}
		cleanup()
		if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("the cleanup did not remove the temporary profile: %v", err)
		}
	})
}

// TC-004 of issue #197: the existing TestBrowserProfile_* tests of issue #180 run unchanged, in
// particular its TC-006 (TestBrowserProfile_TC006_LockThatCannotBeDiscardedIsAFailure: a lock that
// cannot be discarded is a logged and counted start failure, with nothing left behind) and its TC-008
// (TestBrowserProfile_TC008_CheckInProgressKeepsItsProfile). This test adds the check that a failed
// profile preparation reaches the check as a start failure, without launching the browser nor
// calling the nil cleanup.
func TestBrowserProfile_Issue197_TC004_UndiscardableLockIsAStartFailure(t *testing.T) {
	executable, seen := stubBrowser(t)
	profile := filepath.Join(t.TempDir(), "profile")
	writeUndiscardableLock(t, profile, "SingletonLock")
	output := captureOutput(t)
	got := stubCheck(t, executable, profile)
	logged := output()
	assertCheckFailed(t, got)
	if _, launched := seen(); launched {
		t.Error("the browser was launched although the profile could not be prepared")
	}
	if !strings.Contains(logged, "browser could not start") {
		t.Errorf("the failure is not logged as a browser start failure:\n%s", logged)
	}
	assertFileContent(t, filepath.Join(profile, "SingletonLock", "keep"), "directory content")

	t.Run("TC-004 later check after the lock is replaced by a stale lock", func(t *testing.T) {
		amazontest.StaleLock(t, profile, "otherhost")
		assertCheckFailed(t, stubCheck(t, executable, profile))
		listing, launched := seen()
		if !launched {
			t.Fatal("the browser was not launched")
		}
		if listing.directory != profile {
			t.Errorf("the browser got the profile %q, want %q", listing.directory, profile)
		}
		assertNoLockSeen(t, listing)
	})
}
