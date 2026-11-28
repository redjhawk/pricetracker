package amazontest

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Issue #196: the browser-profile helpers shared by the issue #180 tests (StaleLock, BrowserProcesses,
// KillBrowserOnProfile and ProfileHandler). TC-001 to TC-009 are proved here; TC-010, TC-011 and
// TC-015 are repository and code checks, and TC-012, TC-013 and TC-014 run the existing tests of
// internal/amazon, internal/service and internal/httpapi, which call these helpers (TC-013: the
// TestBrowserCollector_ and TestBrowserRetrieval_ tests, through assertNothingLeft). Issue #387:
// TC-001 is proved by the Issue387 test, TC-002 by the issue #196 TC-006 test and TC-004 by its
// TC-005, TC-007 and TC-008 tests; TC-003 and TC-005 to TC-007 are runs and checks.

// staleLockLinks are the names and targets of the lock marks StaleLock leaves for host.
func staleLockLinks(host string) [][2]string {
	return [][2]string{
		{"SingletonLock", host + "-12345"},
		{"SingletonSocket", "/tmp/org.chromium.Chromium.gone/SingletonSocket"},
		{"SingletonCookie", "12345"},
	}
}

// record runs the helper with a recordingTB in its own goroutine and returns whether it failed, with
// which message, and how long it took.
func record(t *testing.T, run func(tb testing.TB)) (bool, string, time.Duration) {
	t.Helper()
	recorder := &recordingTB{TB: t}
	done := make(chan struct{})
	start := time.Now()
	go func() {
		defer close(done)
		run(recorder)
	}()
	<-done
	return recorder.failed, recorder.message, time.Since(start)
}

// assertStaleLockOf checks that the three lock marks of profile are the links StaleLock leaves for host.
func assertStaleLockOf(t *testing.T, profile, host string) {
	t.Helper()
	for _, link := range staleLockLinks(host) {
		path := filepath.Join(profile, link[0])
		info, err := os.Lstat(path)
		if err != nil {
			t.Errorf("%s: %v", link[0], err)
			continue
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			t.Errorf("%s is not a symbolic link: mode %s", link[0], info.Mode())
			continue
		}
		if target, err := os.Readlink(path); err != nil || target != link[1] {
			t.Errorf("%s points to %q (%v), want %q", link[0], target, err, link[1])
		}
	}
}

// readFile returns the content of the file at path.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) // #nosec G304 -- file of the test directory
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// writeStub writes an executable /bin/sh browser stub running body, after setting $profile from its
// --user-data-dir= argument and writing its pid to <dir>/stub.pid. The stub's process group is killed
// at the end of the test.
func writeStub(t *testing.T, body string) (stub, dir string) {
	t.Helper()
	dir = t.TempDir()
	stub = filepath.Join(dir, "browser.sh")
	script := "#!/bin/sh\n" +
		"for arg in \"$@\"; do case \"$arg\" in --user-data-dir=*) profile=\"${arg#--user-data-dir=}\";; esac; done\n" +
		"echo $$ > '" + filepath.Join(dir, "stub.pid") + "'\n" +
		body + "\n"
	if err := os.WriteFile(stub, []byte(script), 0o700); err != nil { // #nosec G306 -- executable stub of the test
		t.Fatal(err)
	}
	t.Cleanup(func() {
		killPidFile(filepath.Join(dir, "stub.pid"), stub, true)
	})
	return stub, dir
}

// killPidFile kills the process whose pid is in path, and its process group when group is true. A
// running process whose command line does not hold marker is another one that reused the pid: it is
// left alone.
func killPidFile(path, marker string, group bool) {
	data, err := os.ReadFile(path) // #nosec G304 -- file of the test directory
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 1 {
		return
	}
	cmdline, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline")) // #nosec G304 -- process of the test
	if err == nil && len(cmdline) > 0 && !strings.Contains(string(cmdline), marker) {
		return
	}
	if group {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
}

// startMarked starts sh -c 'sleep 30; :' <path> in its own process group, so that its command line
// holds path, and returns it with an idempotent stop function, also called at the end of the test.
func startMarked(t *testing.T, path string) (*exec.Cmd, func()) {
	t.Helper()
	cmd := exec.Command("sh", "-c", "sleep 30; :", path) // #nosec G204 -- test process
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	stop := func() {
		once.Do(func() {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			_ = cmd.Wait()
		})
	}
	t.Cleanup(stop)
	return cmd, stop
}

// waitProcesses waits up to 5 s until BrowserProcesses(profile) gives want and returns its last
// result. The wait covers a shell's forked child, which holds the shell's command line until it runs
// its own command or dies.
func waitProcesses(profile string, want []string) []string {
	for deadline := time.Now().Add(5 * time.Second); !slices.Equal(BrowserProcesses(profile), want) && time.Now().Before(deadline); {
		time.Sleep(20 * time.Millisecond)
	}
	return BrowserProcesses(profile)
}

// waitNoProcesses waits up to 5 s until BrowserProcesses(profile) gives no entry and returns what is left.
func waitNoProcesses(profile string) []string {
	return waitProcesses(profile, nil)
}

// TC-001
func TestStaleLock_Issue196_TC001_CreatesTheProfile(t *testing.T) {
	const gone = "/tmp/org.chromium.Chromium.gone"
	_, err := os.Lstat(gone)
	goneBefore := err == nil

	profile := filepath.Join(t.TempDir(), "a", "profile")
	StaleLock(t, profile, "somehost")
	entries, err := os.ReadDir(profile)
	if err != nil {
		t.Fatalf("the profile folder was not created: %v", err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if want := []string{"SingletonCookie", "SingletonLock", "SingletonSocket"}; !slices.Equal(names, want) {
		t.Errorf("the profile holds %v, want %v", names, want)
	}
	assertStaleLockOf(t, profile, "somehost")

	other := filepath.Join(t.TempDir(), "profile")
	StaleLock(t, other, "otherhost")
	assertStaleLockOf(t, other, "otherhost")

	if _, err := os.Lstat(gone); !goneBefore && err == nil {
		t.Errorf("StaleLock created %s; the link targets need not exist", gone)
	}
}

// TC-002
func TestStaleLock_Issue196_TC002_ReplacesTheMarks(t *testing.T) {
	const preferences, extra = `{"profile":{"name":"Person 1"}}`, "not a lock mark"
	rows := []struct {
		name    string
		prepare func(t *testing.T, profile string)
		calls   int
	}{
		{"regular files", func(t *testing.T, profile string) {
			for _, link := range staleLockLinks("somehost") {
				if err := os.WriteFile(filepath.Join(profile, link[0]), []byte("regular file"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}, 1},
		{"lock directory and links of another host", func(t *testing.T, profile string) {
			StaleLock(t, profile, "firsthost")
			lock := filepath.Join(profile, "SingletonLock")
			if err := os.Remove(lock); err != nil && !errors.Is(err, fs.ErrNotExist) {
				t.Fatal(err)
			}
			if err := os.MkdirAll(lock, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(lock, "keep"), []byte("directory content"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, 1},
		{"no lock mark", func(t *testing.T, profile string) {}, 1},
		{"repeated call", func(t *testing.T, profile string) {}, 2},
		{"dangling SingletonLock", func(t *testing.T, profile string) {
			if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), filepath.Join(profile, "SingletonLock")); err != nil {
				t.Fatal(err)
			}
		}, 1},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			profile := filepath.Join(t.TempDir(), "profile")
			if err := os.MkdirAll(filepath.Join(profile, "Default"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(profile, "Default", "Preferences"), []byte(preferences), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(profile, "SingletonLockExtra"), []byte(extra), 0o600); err != nil {
				t.Fatal(err)
			}
			row.prepare(t, profile)
			for range row.calls {
				StaleLock(t, profile, "somehost")
			}
			assertStaleLockOf(t, profile, "somehost")
			if got := readFile(t, filepath.Join(profile, "Default", "Preferences")); got != preferences {
				t.Errorf("Default/Preferences changed: %q", got)
			}
			if got := readFile(t, filepath.Join(profile, "SingletonLockExtra")); got != extra {
				t.Errorf("SingletonLockExtra changed: %q", got)
			}
		})
	}
}

// TC-003
func TestStaleLock_Issue196_TC003_ErrorFailsTheTest(t *testing.T) {
	t.Run("folder that cannot be created", func(t *testing.T) {
		base := t.TempDir()
		file := filepath.Join(base, "file")
		if err := os.WriteFile(file, []byte("regular file"), 0o600); err != nil {
			t.Fatal(err)
		}
		failed, message, _ := record(t, func(tb testing.TB) { StaleLock(tb, filepath.Join(file, "profile"), "somehost") })
		if !failed {
			t.Fatal("StaleLock passed on a profile whose parent is a regular file")
		}
		if !strings.Contains(message, "not a directory") || !strings.Contains(message, file) {
			t.Errorf("the failure message does not give the error and its path %s: %q", file, message)
		}
		entries, err := os.ReadDir(base)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "file" || readFile(t, file) != "regular file" {
			t.Errorf("StaleLock left something in %s: %v", base, entries)
		}
	})

	t.Run("read-only profile", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("a read-only folder does not deny root")
		}
		profile := filepath.Join(t.TempDir(), "profile")
		if err := os.MkdirAll(profile, 0o700); err != nil {
			t.Fatal(err)
		}
		lock := filepath.Join(profile, "SingletonLock")
		if err := os.WriteFile(lock, []byte("regular file"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(profile, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(profile, 0o700) })
		failed, message, _ := record(t, func(tb testing.TB) { StaleLock(tb, profile, "somehost") })
		if !failed {
			t.Fatal("StaleLock passed on a read-only profile")
		}
		if !strings.Contains(message, "permission denied") || !strings.Contains(message, lock) {
			t.Errorf("the failure message does not give the error and its path %s: %q", lock, message)
		}
		if strings.Contains(message, "SingletonSocket") || strings.Contains(message, "SingletonCookie") {
			t.Errorf("StaleLock went on after the first error: %q", message)
		}
		if info, err := os.Lstat(lock); err != nil || !info.Mode().IsRegular() {
			t.Errorf("SingletonLock is no longer the regular file: %v %v", info, err)
		}
	})
}

// TC-004
func TestBrowserProcesses_Issue196_TC004_ListsTheProfileProcesses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile-"+strconv.FormatInt(time.Now().UnixNano(), 36))
	other := filepath.Join(t.TempDir(), "profile-"+strconv.FormatInt(time.Now().UnixNano()+1, 36))
	if got := BrowserProcesses(path); len(got) != 0 {
		t.Fatalf("before the process: %q, want no entry", got)
	}

	first, stopFirst := startMarked(t, path)
	second, stopSecond := startMarked(t, other)
	entry := func(cmd *exec.Cmd, path string) string {
		return strconv.Itoa(cmd.Process.Pid) + ": sh -c sleep 30; : " + path + " "
	}
	if want, got := []string{entry(first, path)}, waitProcesses(path, []string{entry(first, path)}); !slices.Equal(got, want) {
		t.Errorf("during the process: %q, want %q", got, want)
	}
	if want, got := []string{entry(second, other)}, waitProcesses(other, []string{entry(second, other)}); !slices.Equal(got, want) {
		t.Errorf("the second process: %q, want %q", got, want)
	}
	for _, got := range BrowserProcesses(path) {
		if strings.Contains(got, "\x00") {
			t.Errorf("the entry holds a NUL byte: %q", got)
		}
	}

	stopFirst()
	stopSecond()
	if got := waitNoProcesses(path); len(got) != 0 {
		t.Errorf("after the process: %q, want no entry", got)
	}
	if got := waitNoProcesses(other); len(got) != 0 {
		t.Errorf("after the second process: %q, want no entry", got)
	}
}

// TC-005
func TestKillBrowserOnProfile_Issue196_TC005_LeavesTheLock(t *testing.T) {
	Browser(t)
	// killed kills the browser on profile and returns the target of the lock it left.
	killed := func(t *testing.T, profile string) string {
		t.Helper()
		failed, message, _ := record(t, func(tb testing.TB) { KillBrowserOnProfile(tb, profile) })
		if failed {
			t.Fatalf("KillBrowserOnProfile failed: %s", message)
		}
		lock := filepath.Join(profile, "SingletonLock")
		if _, err := os.Lstat(lock); err != nil {
			t.Fatalf("no SingletonLock after the kill: %v", err)
		}
		if left := BrowserProcesses(profile); len(left) != 0 {
			t.Fatalf("browser processes left after the kill: %q", left)
		}
		target, err := os.Readlink(lock)
		if err != nil {
			t.Fatal(err)
		}
		return target
	}

	t.Run("new profile", func(t *testing.T) {
		profile := filepath.Join(t.TempDir(), "profile")
		if err := os.MkdirAll(profile, 0o700); err != nil {
			t.Fatal(err)
		}
		killed(t, profile)
	})

	t.Run("stale lock of another host", func(t *testing.T) {
		profile := filepath.Join(t.TempDir(), "profile")
		StaleLock(t, profile, "otherhost")
		if target := killed(t, profile); target == "otherhost-12345" {
			t.Errorf("SingletonLock is still the stale lock of otherhost")
		}
	})

	t.Run("second kill on the same profile", func(t *testing.T) {
		profile := filepath.Join(t.TempDir(), "profile")
		if err := os.MkdirAll(profile, 0o700); err != nil {
			t.Fatal(err)
		}
		first := killed(t, profile)
		if second := killed(t, profile); second == first {
			t.Errorf("SingletonLock is still the lock of the first killed browser: %q", second)
		}
	})
}

// TC-006
func TestKillBrowserOnProfile_Issue196_TC006_ExitBeforeHolding(t *testing.T) {
	const want = "the browser exited before holding the profile"
	exiting, _ := writeStub(t, "exit 3")
	rows := []struct {
		name    string
		browser string
		stale   bool
	}{
		{"/bin/true with a stale lock", "/bin/true", true},
		{"stub exiting with status 3", exiting, true},
		{"profile without SingletonLock", "/bin/true", false},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			t.Setenv("PRICEFOLLOWER_BROWSER_PATH", row.browser)
			profile := filepath.Join(t.TempDir(), "profile")
			if err := os.MkdirAll(profile, 0o700); err != nil {
				t.Fatal(err)
			}
			lock := filepath.Join(profile, "SingletonLock")
			if row.stale {
				if err := os.Symlink("otherhost-12345", lock); err != nil {
					t.Fatal(err)
				}
			}
			failed, message, took := record(t, func(tb testing.TB) { KillBrowserOnProfile(tb, profile) })
			if !failed || message != want {
				t.Errorf("failed %v with %q, want %q", failed, message, want)
			}
			if took >= 5*time.Second {
				t.Errorf("the failure took %s, want less than 5s", took)
			}
			if _, err := os.Lstat(lock); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("SingletonLock is still there: %v", err)
			}
		})
	}
}

// TC-007
func TestKillBrowserOnProfile_Issue196_TC007_NeverHolding(t *testing.T) {
	const want = "the browser did not create SingletonLock in the profile"
	stub, _ := writeStub(t, "sh -c 'sleep 120; :' \"$profile\" &\nwhile :; do sleep 0.1; done")
	t.Setenv("PRICEFOLLOWER_BROWSER_PATH", stub)
	profile := filepath.Join(t.TempDir(), "profile")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	failed, message, took := record(t, func(tb testing.TB) { KillBrowserOnProfile(tb, profile) })
	if !failed || message != want {
		t.Errorf("failed %v with %q, want %q", failed, message, want)
	}
	if took < 30*time.Second || took > 40*time.Second {
		t.Errorf("the failure took %s, want 30s to 40s", took)
	}
	if left := waitNoProcesses(profile); len(left) != 0 {
		t.Errorf("processes left after the failure: %q", left)
	}
}

// TC-008
func TestKillBrowserOnProfile_Issue196_TC008_LeftoverAndMissingLock(t *testing.T) {
	t.Run("process left after the kill", func(t *testing.T) {
		const prefix = "browser processes left after the kill: "
		profile := filepath.Join(t.TempDir(), "profile")
		if err := os.MkdirAll(profile, 0o700); err != nil {
			t.Fatal(err)
		}
		escaped := filepath.Join(t.TempDir(), "escaped.pid")
		t.Cleanup(func() { killPidFile(escaped, profile, false) })
		stub, _ := writeStub(t, fmt.Sprintf("ln -s \"stub-$$\" \"$profile/SingletonLock\"\n"+
			"setsid sh -c 'sleep 60; :' \"$profile\" &\necho $! > '%s'\nwhile :; do sleep 0.1; done", escaped))
		t.Setenv("PRICEFOLLOWER_BROWSER_PATH", stub)
		failed, message, took := record(t, func(tb testing.TB) { KillBrowserOnProfile(tb, profile) })
		if !failed || !strings.HasPrefix(message, prefix) {
			t.Fatalf("failed %v with %q, want a message starting with %q", failed, message, prefix)
		}
		pid := strings.TrimSpace(readFile(t, escaped))
		if !strings.Contains(message, pid+": ") || !strings.Contains(message, profile) {
			t.Errorf("the message does not list the escaped process %s on %s: %q", pid, profile, message)
		}
		if took >= 10*time.Second {
			t.Errorf("the failure took %s, want less than 10s", took)
		}
	})

	t.Run("lock removed before the kill", func(t *testing.T) {
		const prefix = "precondition: the killed browser left no SingletonLock: "
		stub, _ := writeStub(t, "ln -s \"stub-$$\" \"$profile/SingletonLock\"\nsleep 0.3\nrm -f \"$profile/SingletonLock\"\nwhile :; do sleep 0.1; done")
		t.Setenv("PRICEFOLLOWER_BROWSER_PATH", stub)
		profile := filepath.Join(t.TempDir(), "profile")
		if err := os.MkdirAll(profile, 0o700); err != nil {
			t.Fatal(err)
		}
		failed, message, _ := record(t, func(tb testing.TB) { KillBrowserOnProfile(tb, profile) })
		if !failed || !strings.HasPrefix(message, prefix) || !strings.Contains(message, "no such file or directory") {
			t.Errorf("failed %v with %q, want %q followed by the Lstat error", failed, message, prefix)
		}
	})
}

// Issue #387, TC-001
func TestKillBrowserOnProfile_Issue387_TC001_ExitBeforeHoldingKillsTheGroup(t *testing.T) {
	const want = "the browser exited before holding the profile"
	for _, row := range []struct {
		name  string
		stale bool
	}{
		{"stale lock", true},
		{"no SingletonLock", false},
	} {
		t.Run(row.name, func(t *testing.T) {
			profile := filepath.Join(t.TempDir(), "profile")
			if err := os.MkdirAll(profile, 0o700); err != nil {
				t.Fatal(err)
			}
			lock := filepath.Join(profile, "SingletonLock")
			if row.stale {
				if err := os.Symlink("otherhost-12345", lock); err != nil {
					t.Fatal(err)
				}
			}
			child := filepath.Join(t.TempDir(), "child.pid")
			t.Cleanup(func() { killPidFile(child, profile, false) })
			stub, _ := writeStub(t, fmt.Sprintf("sh -c 'sleep 60; :' \"$profile\" &\necho $! > '%s'\nexit 0", child))
			t.Setenv("PRICEFOLLOWER_BROWSER_PATH", stub)
			failed, message, took := record(t, func(tb testing.TB) { KillBrowserOnProfile(tb, profile) })
			if !failed || message != want {
				t.Errorf("failed %v with %q, want %q", failed, message, want)
			}
			if took >= 5*time.Second {
				t.Errorf("the failure took %s, want less than 5s", took)
			}
			pid := strings.TrimSpace(readFile(t, child))
			if n, err := strconv.Atoi(pid); err != nil || n <= 1 {
				t.Fatalf("the stub recorded no background child: %q (%v)", pid, err)
			}
			if left := waitNoProcesses(profile); len(left) != 0 {
				t.Errorf("processes left after the early exit (child %s): %q", pid, left)
			}
			if _, err := os.Lstat(lock); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("SingletonLock is still there: %v", err)
			}
		})
	}
}

// TC-009
func TestProfileHandler_Issue196_TC009_ServesTheProfilePages(t *testing.T) {
	page, noOffer := Fixture(t, "price-found.html"), Fixture(t, "aod-none.html")
	handler := ProfileHandler(t)
	servers := []*Server{NewServer(t, handler), NewServer(t, handler)}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	rows := []struct {
		path   string
		status int
		body   string
		cookie bool
	}{
		{"/dp/B0ANY00001", http.StatusOK, page, true},
		{"/dp/B0OTHER002", http.StatusOK, page, true},
		{"/gp/aod/ajax?asin=B0ANY00001", http.StatusOK, noOffer, false},
		{"/gp/aod/ajax/B0ANY00001", http.StatusOK, noOffer, false},
		{"/static/x.css", http.StatusNoContent, "", false},
		{"/static/js/a.js", http.StatusNoContent, "", false},
		{"/other", http.StatusNotFound, "", false},
		{"/", http.StatusNotFound, "", false},
		{"/gp/aod", http.StatusNotFound, "", false},
	}
	for index, server := range servers {
		for _, row := range rows {
			for attempt := 1; attempt <= 2; attempt++ {
				what := fmt.Sprintf("server %d, GET %s, request %d", index+1, row.path, attempt)
				resp, err := client.Get(server.URL + row.path)
				if err != nil {
					t.Fatalf("%s: %v", what, err)
				}
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					t.Fatalf("%s: %v", what, err)
				}
				if resp.StatusCode != row.status {
					t.Errorf("%s: status %d, want %d", what, resp.StatusCode, row.status)
					continue
				}
				if row.status == http.StatusNotFound {
					continue
				}
				if string(body) != row.body {
					t.Errorf("%s: body of %d bytes is not the expected one of %d bytes", what, len(body), len(row.body))
				}
				if row.status == http.StatusOK {
					if got := resp.Header.Get("Content-Type"); got != "text/html; charset=utf-8" {
						t.Errorf("%s: content type %q", what, got)
					}
				}
				var session *http.Cookie
				for _, cookie := range resp.Cookies() {
					if cookie.Name == "session-id" {
						session = cookie
					}
				}
				switch {
				case row.cookie && session == nil:
					t.Errorf("%s: no session-id cookie", what)
				case row.cookie && (session.Value != "fixture-session-180" || session.Path != "/" || session.MaxAge != 3600):
					t.Errorf("%s: cookie %+v, want session-id=fixture-session-180; Path=/; Max-Age=3600", what, session)
				case !row.cookie && session != nil:
					t.Errorf("%s: unexpected session-id cookie %+v", what, session)
				}
			}
		}
	}
}
