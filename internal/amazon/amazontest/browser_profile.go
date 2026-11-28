package amazontest

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// lockMarks gives, in the order they are written, the names and link targets of the lock marks of a
// browser of host that did not end normally.
func lockMarks(host string) []struct{ name, target string } {
	return []struct{ name, target string }{
		{"SingletonLock", host + "-12345"},
		{"SingletonSocket", "/tmp/org.chromium.Chromium.gone/SingletonSocket"},
		{"SingletonCookie", "12345"},
	}
}

// StaleLock leaves in profile, created when missing, the lock marks of a browser of host that did
// not end normally, replacing any existing marks. Any error fails the test.
func StaleLock(t testing.TB, profile, host string) {
	t.Helper()
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	marks := lockMarks(host)
	for _, link := range marks {
		if err := os.RemoveAll(filepath.Join(profile, link.name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, link := range marks {
		if err := os.Symlink(link.target, filepath.Join(profile, link.name)); err != nil {
			t.Fatal(err)
		}
	}
}

// BrowserProcesses lists the running processes whose command line contains profile, each as
// "<pid>: <command line>" with the arguments separated by spaces.
func BrowserProcesses(profile string) []string {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var found []string
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil || !strings.Contains(string(cmdline), profile) {
			continue
		}
		found = append(found, entry.Name()+": "+strings.ReplaceAll(string(cmdline), "\x00", " "))
	}
	return found
}

// KillBrowserOnProfile starts the browser of Browser on profile in its own process group, waits until
// it holds the profile, kills the group and checks that no process is left and the lock is left
// behind, as after a browser killed during a check.
func KillBrowserOnProfile(t testing.TB, profile string) {
	t.Helper()
	lock := filepath.Join(profile, "SingletonLock")
	if err := os.Remove(lock); err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	cmd := exec.Command(Browser(t), "--headless=new", "--user-data-dir="+profile, "--no-sandbox", "--no-first-run", "about:blank") // #nosec G204 -- test browser
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	kill := func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-exited
	}
	for deadline := time.Now().Add(30 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if _, err := os.Lstat(lock); err == nil {
			break
		}
		select {
		case <-exited:
			t.Fatal("the browser exited before holding the profile")
		default:
		}
		if time.Now().After(deadline) {
			kill()
			t.Fatal("the browser did not create SingletonLock in the profile")
		}
	}
	time.Sleep(500 * time.Millisecond) // let the browser finish starting, as during a check
	kill()
	for deadline := time.Now().Add(5 * time.Second); len(BrowserProcesses(profile)) > 0 && time.Now().Before(deadline); {
		time.Sleep(20 * time.Millisecond)
	}
	if left := BrowserProcesses(profile); len(left) > 0 {
		t.Fatalf("browser processes left after the kill: %v", left)
	}
	if _, err := os.Lstat(lock); err != nil {
		t.Fatalf("precondition: the killed browser left no SingletonLock: %v", err)
	}
}

// ProfileHandler serves the profile fixture pages of the issue #180 tests: the price-found page at
// /dp/<ASIN> for every ASIN with the session cookie, the used-offer responses without offers and the
// page resources. Other paths give 404. It keeps no state.
func ProfileHandler(t testing.TB) http.Handler {
	t.Helper()
	page, noOffer := Fixture(t, "price-found.html"), Fixture(t, "aod-none.html")
	mux := http.NewServeMux()
	mux.HandleFunc("/dp/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session-id", Value: "fixture-session-180", Path: "/", MaxAge: 3600})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, page)
	})
	usedOffers := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, noOffer)
	}
	mux.HandleFunc("/gp/aod/ajax", usedOffers)
	mux.HandleFunc("/gp/aod/ajax/", usedOffers)
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	return mux
}
