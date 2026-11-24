package amazon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/model"
)

// Browser profile recovery of issue #180: lock marks left in the persistent profile by a browser
// that did not end normally, on this machine or another one, do not prevent the next check.

const profileCookie = "session-id=fixture-session-180"

var profileLockNames = []string{"SingletonLock", "SingletonSocket", "SingletonCookie"}

// profileFixtureHandler serves the fixture pages of fixtureHandler; the product page of B0PRICE001
// also sets the cookie fixture of issue #180.
func profileFixtureHandler(t *testing.T) http.Handler {
	t.Helper()
	page := amazontest.Fixture(t, "price-found.html")
	mux := http.NewServeMux()
	mux.Handle("/", fixtureHandler(t))
	mux.HandleFunc("/dp/B0PRICE001", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session-id", Value: "fixture-session-180", Path: "/", MaxAge: 3600})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, page)
	})
	return mux
}

// pageCookieSent reports whether a request for path carries the cookie fixture.
func pageCookieSent(requests []amazontest.Request, path string) bool {
	for _, request := range requests {
		if request.Path == path && strings.Contains(request.Header.Get("Cookie"), profileCookie) {
			return true
		}
	}
	return false
}

// writeStaleLock leaves in profile the lock marks of a browser of host that did not end normally.
func writeStaleLock(t *testing.T, profile, host string) {
	t.Helper()
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range profileLockNames {
		if err := os.RemoveAll(filepath.Join(profile, name)); err != nil {
			t.Fatal(err)
		}
	}
	links := map[string]string{"SingletonLock": host + "-12345", "SingletonSocket": "/tmp/org.chromium.Chromium.gone/SingletonSocket", "SingletonCookie": "12345"}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(profile, name)); err != nil {
			t.Fatal(err)
		}
	}
}

// killBrowserOnProfile starts the browser on profile in its own process group, waits until it holds
// the profile, kills the group and checks that the lock is left behind.
func killBrowserOnProfile(t *testing.T, executable, profile string) {
	t.Helper()
	lock := filepath.Join(profile, "SingletonLock")
	if err := os.Remove(lock); err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "--headless=new", "--user-data-dir="+profile, "--no-sandbox", "--no-first-run", "about:blank") // #nosec G204 -- test browser
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
	for deadline := time.Now().Add(5 * time.Second); len(browserProcesses(profile)) > 0 && time.Now().Before(deadline); {
		time.Sleep(20 * time.Millisecond)
	}
	if left := browserProcesses(profile); len(left) > 0 {
		t.Fatalf("browser processes left after the kill: %v", left)
	}
	if _, err := os.Lstat(lock); err != nil {
		t.Fatalf("precondition: the killed browser left no SingletonLock: %v", err)
	}
}

// assertPriceFound checks a successful check of B0PRICE001.
func assertPriceFound(t *testing.T, what string, got model.CollectionResult) {
	t.Helper()
	if got.Result != "success" || got.AmountCents != 12999 {
		t.Fatalf("%s: got %+v, want success at 129.99", what, got)
	}
}

func assertCheckFailed(t *testing.T, got model.CollectionResult) {
	t.Helper()
	if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" {
		t.Fatalf("not a failure: %+v", got)
	}
}

// profileListing is what the stub executable saw in its --user-data-dir at launch.
type profileListing struct {
	directory string
	entries   map[string]string // relative name -> "<kind> <symlink target>"
}

// stubBrowser writes a stub executable that records its profile directory and exits 3, and returns
// it with the function reading the recorded listing.
func stubBrowser(t *testing.T) (string, func() (profileListing, bool)) {
	t.Helper()
	dir := t.TempDir()
	executable, listing, directory := filepath.Join(dir, "stub-browser"), filepath.Join(dir, "listing"), filepath.Join(dir, "directory")
	script := "#!/bin/sh\nfor arg in \"$@\"; do\n  case \"$arg\" in --user-data-dir=*) dir=\"${arg#--user-data-dir=}\";; esac\ndone\n" +
		"printf '%s' \"$dir\" > '" + directory + "'\n" +
		"find \"$dir\" -mindepth 1 -printf '%P\\t%y\\t%l\\n' > '" + listing + ".tmp'\n" +
		"mv '" + listing + ".tmp' '" + listing + "'\nexit 3\n"
	if err := os.WriteFile(executable, []byte(script), 0o755); err != nil { // #nosec G306 -- test executable
		t.Fatal(err)
	}
	read := func() (profileListing, bool) {
		data, err := os.ReadFile(listing)
		if err != nil {
			return profileListing{}, false
		}
		dirData, err := os.ReadFile(directory)
		if err != nil {
			return profileListing{}, false
		}
		result := profileListing{directory: string(dirData), entries: map[string]string{}}
		for _, line := range strings.Split(string(data), "\n") {
			if fields := strings.SplitN(line, "\t", 3); len(fields) == 3 {
				result.entries[fields[0]] = fields[1] + " " + fields[2]
			}
		}
		return result, true
	}
	return executable, read
}

func stubCheck(t *testing.T, executable, profile string) model.CollectionResult {
	t.Helper()
	collector := NewBrowserCollector(BrowserOptions{ExecPath: executable, ProfileDir: profile, NoSandbox: true, NavigationTimeout: 5 * time.Second,
		Pause: func() time.Duration { return 0 }, Origin: "http://127.0.0.1:9"})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return collector.Collect(ctx, fixtureItem("B0PRICE001"))
}

func writeProfileFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertNoLockSeen(t *testing.T, seen profileListing) {
	t.Helper()
	for _, name := range profileLockNames {
		if entry, found := seen.entries[name]; found {
			t.Errorf("the browser found %s (%s) in the profile at launch", name, entry)
		}
	}
}

// TC-001
func TestBrowserProfile_TC001_OnlyLockMarksAreDiscardedBeforeStart(t *testing.T) {
	t.Run("stale lock among profile data", func(t *testing.T) {
		executable, seen := stubBrowser(t)
		profile := filepath.Join(t.TempDir(), "profile")
		outside := t.TempDir()
		socketTarget := filepath.Join(outside, "socket-target")
		writeProfileFile(t, socketTarget, "socket target content")
		files := map[string]string{"Default/Cookies": "cookie database fixture", "Local State": `{"profile":{"fixture":180}}`, "SingletonLock.bak": "look-alike name"}
		for name, content := range files {
			writeProfileFile(t, filepath.Join(profile, name), content)
		}
		for name, target := range map[string]string{"SingletonLock": "otherhost-12345", "SingletonSocket": socketTarget, "SingletonCookie": "4242"} {
			if err := os.Symlink(target, filepath.Join(profile, name)); err != nil {
				t.Fatal(err)
			}
		}

		got := stubCheck(t, executable, profile)
		assertCheckFailed(t, got)
		listing, launched := seen()
		if !launched {
			t.Fatal("the browser was not launched")
		}
		if listing.directory != profile {
			t.Fatalf("the browser got the profile %q, want %q", listing.directory, profile)
		}
		assertNoLockSeen(t, listing)
		for _, name := range []string{"Default", "Default/Cookies", "Local State", "SingletonLock.bak"} {
			if _, found := listing.entries[name]; !found {
				t.Errorf("%s missing from the profile at launch: %v", name, listing.entries)
			}
		}
		for name, content := range files {
			if data, err := os.ReadFile(filepath.Join(profile, name)); err != nil || string(data) != content {
				t.Errorf("%s changed: %q %v", name, data, err)
			}
		}
		if data, err := os.ReadFile(socketTarget); err != nil || string(data) != "socket target content" {
			t.Errorf("the target of SingletonSocket outside the profile was touched: %q %v", data, err)
		}
	})

	t.Run("dangling symlinks", func(t *testing.T) {
		executable, seen := stubBrowser(t)
		profile := filepath.Join(t.TempDir(), "profile")
		writeStaleLock(t, profile, "otherhost")
		assertCheckFailed(t, stubCheck(t, executable, profile))
		listing, launched := seen()
		if !launched {
			t.Fatal("the browser was not launched")
		}
		assertNoLockSeen(t, listing)
	})

	t.Run("only SingletonLock", func(t *testing.T) {
		executable, seen := stubBrowser(t)
		profile := filepath.Join(t.TempDir(), "profile")
		if err := os.MkdirAll(profile, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("otherhost-12345", filepath.Join(profile, "SingletonLock")); err != nil {
			t.Fatal(err)
		}
		assertCheckFailed(t, stubCheck(t, executable, profile))
		listing, launched := seen()
		if !launched {
			t.Fatal("the browser was not launched: removing a lone SingletonLock must not fail")
		}
		assertNoLockSeen(t, listing)
	})

	t.Run("marks are regular files", func(t *testing.T) {
		executable, seen := stubBrowser(t)
		profile := filepath.Join(t.TempDir(), "profile")
		for _, name := range profileLockNames {
			writeProfileFile(t, filepath.Join(profile, name), "regular file")
		}
		assertCheckFailed(t, stubCheck(t, executable, profile))
		listing, launched := seen()
		if !launched {
			t.Fatal("the browser was not launched")
		}
		assertNoLockSeen(t, listing)
	})

	t.Run("no mark and no profile directory yet", func(t *testing.T) {
		executable, seen := stubBrowser(t)
		profile := filepath.Join(t.TempDir(), "profile")
		assertCheckFailed(t, stubCheck(t, executable, profile))
		listing, launched := seen()
		if !launched {
			t.Fatal("the browser was not launched")
		}
		if info, err := os.Stat(profile); err != nil || !info.IsDir() {
			t.Fatalf("the profile directory was not created: %v", err)
		}
		if listing.directory != profile || len(listing.entries) != 0 {
			t.Fatalf("the browser got %q with %v, want the new empty profile", listing.directory, listing.entries)
		}
	})

	t.Run("lock naming this host and a live process", func(t *testing.T) {
		executable, seen := stubBrowser(t)
		profile := filepath.Join(t.TempDir(), "profile")
		host, err := os.Hostname()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(profile, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(host+"-"+strconv.Itoa(os.Getpid()), filepath.Join(profile, "SingletonLock")); err != nil {
			t.Fatal(err)
		}
		assertCheckFailed(t, stubCheck(t, executable, profile))
		listing, launched := seen()
		if !launched {
			t.Fatal("the browser was not launched")
		}
		assertNoLockSeen(t, listing)
	})

	t.Run("temporary profile", func(t *testing.T) {
		executable, seen := stubBrowser(t)
		assertCheckFailed(t, stubCheck(t, executable, ""))
		listing, launched := seen()
		if !launched {
			t.Fatal("the browser was not launched")
		}
		if !strings.HasPrefix(filepath.Base(listing.directory), "pricefollower-browser-") || len(listing.entries) != 0 {
			t.Fatalf("the browser got %q with %v, want a new empty temporary directory", listing.directory, listing.entries)
		}
		if _, err := os.Stat(listing.directory); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("the temporary profile %s was not removed after the check: %v", listing.directory, err)
		}
	})
}

// lockRefusal is the part of Chromium's refusal of a profile locked by host on another computer
// that names that host.
func lockRefusal(host string) string { return "on another computer (" + host + ")" }

// browserRun is what one run of the browser on the locked profile showed.
type browserRun struct {
	pid       int    // process ID and process group of the run, killed and waited for
	requested bool   // the page was requested while the browser ran
	output    string // standard output and standard error, read after Wait returned
}

// runBrowserOnProfile starts executable on profile with url server.URL+path in its own process group,
// waits until it exits or wait has passed, kills the group and waits for it.
func runBrowserOnProfile(t *testing.T, executable, profile string, server *amazontest.Server, path string, wait time.Duration) browserRun {
	t.Helper()
	cmd := exec.Command(executable, "--headless=new", "--user-data-dir="+profile, "--no-sandbox", "--no-first-run", server.URL+path) // #nosec G204 -- test browser
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	before := server.Count(path)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	select {
	case <-exited:
	case <-time.After(wait):
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	<-exited
	return browserRun{pid: cmd.Process.Pid, requested: server.Count(path) != before, output: output.String()}
}

// fixtureCheckFailure returns why run does not prove that the browser refused the profile because
// host locks it, or "" when it does.
func fixtureCheckFailure(run browserRun, host string) string {
	var failed []string
	if run.requested {
		failed = append(failed, "the browser opened the page on a profile locked by another computer")
	}
	if !strings.Contains(run.output, lockRefusal(host)) {
		failed = append(failed, fmt.Sprintf("the browser output does not hold the lock refusal message %q", lockRefusal(host)))
	}
	if len(failed) == 0 {
		return ""
	}
	return fmt.Sprintf("fixture check: %s; browser output (%d bytes):\n%s", strings.Join(failed, "; "), len(run.output), run.output)
}

// assertBrowserRefusesProfile starts the browser directly on profile and checks that it does not
// open the page and says that host on another computer holds the profile.
func assertBrowserRefusesProfile(t *testing.T, executable, profile, host string, server *amazontest.Server, path string) {
	t.Helper()
	if failure := fixtureCheckFailure(runBrowserOnProfile(t, executable, profile, server, path, 15*time.Second), host); failure != "" {
		t.Fatal(failure)
	}
}

const refusalStubVariable = "PRICEFOLLOWER_REFUSAL_STUB"

// refusalStub tells the stub started in place of the browser by the fixture check tests what to do.
type refusalStub struct {
	Request bool   `json:"request"` // GET the last argument (the page URL) and read the body first
	Stderr  string `json:"stderr"`  // written to standard error, after the request
	Hold    bool   `json:"hold"`    // then sleep one minute instead of exiting (killed by the check)
	Exit    int    `json:"exit"`    // exit status otherwise
}

// refusalStubBrowser writes an executable that runs this test binary as stub, and returns it.
func refusalStubBrowser(t *testing.T, stub refusalStub) string {
	t.Helper()
	encoded, err := json.Marshal(stub)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(refusalStubVariable, string(encoded))
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(t.TempDir(), "refusal-stub")
	script := "#!/bin/sh\nexec '" + binary + "' '-test.run=^TestBrowserProfile_Issue195_RefusalStubProcess$' -- \"$@\"\n"
	if err := os.WriteFile(executable, []byte(script), 0o755); err != nil { // #nosec G306 -- test executable
		t.Fatal(err)
	}
	return executable
}

// TestBrowserProfile_Issue195_RefusalStubProcess is the stub of refusalStubBrowser; it does nothing
// in a normal run.
func TestBrowserProfile_Issue195_RefusalStubProcess(t *testing.T) {
	encoded := os.Getenv(refusalStubVariable)
	if encoded == "" {
		return
	}
	fail := func(err error) {
		fmt.Fprintln(os.Stderr, "refusal stub:", err)
		os.Exit(2)
	}
	var stub refusalStub
	if err := json.Unmarshal([]byte(encoded), &stub); err != nil {
		fail(err)
	}
	if stub.Request {
		client := &http.Client{Timeout: 10 * time.Second}
		response, err := client.Get(os.Args[len(os.Args)-1]) // #nosec G107 -- fixture server of the test
		if err != nil {
			fail(err)
		}
		_, err = io.Copy(io.Discard, response.Body)
		response.Body.Close()
		if err != nil {
			fail(err)
		}
	}
	io.WriteString(os.Stderr, stub.Stderr)
	if stub.Hold {
		time.Sleep(time.Minute)
	}
	os.Exit(stub.Exit)
}

// TC-001, TC-002 and TC-003 of issue #195
func TestBrowserProfile_Issue195_FixtureCheckNeedsTheLockRefusal(t *testing.T) {
	const (
		refusal = "[1:1:ERROR:process_singleton_posix.cc(358)] The profile appears to be in use by another Chromium process (12345) on another computer (otherhost). Chromium has locked the profile so that it doesn't get corrupted.\n"
		opened  = "the browser opened the page on a profile locked by another computer"
		path    = "/dp/B0PRICE001"
	)
	missing := fmt.Sprintf("the browser output does not hold the lock refusal message %q", lockRefusal("otherhost"))
	server := amazontest.NewServer(t, profileFixtureHandler(t))
	profile := filepath.Join(t.TempDir(), "profile")
	writeStaleLock(t, profile, "otherhost")

	rows := []struct {
		name   string
		stub   refusalStub
		failed []string // conditions named by the failure, in order; none when the check passes
	}{
		{"TC-001 refusal, then exit", refusalStub{Stderr: refusal, Exit: 21}, nil},
		{"TC-001 refusal, then still running at the end of the wait", refusalStub{Stderr: refusal, Hold: true}, nil},
		{"TC-002 exit at once without output", refusalStub{Exit: 1}, []string{missing}},
		{"TC-002 missing library", refusalStub{Stderr: "error while loading shared libraries: libnss3.so: cannot open shared object file: No such file or directory\n", Exit: 127}, []string{missing}},
		{"TC-002 unknown option", refusalStub{Stderr: "unknown option --headless=new\n", Exit: 1}, []string{missing}},
		{"TC-002 slow start", refusalStub{Hold: true}, []string{missing}},
		{"TC-002 generic lock error only", refusalStub{Stderr: "Failed to create a ProcessSingleton for your profile directory.\n", Exit: 21}, []string{missing}},
		{"TC-002 refusal naming another host", refusalStub{Stderr: strings.Replace(refusal, "(otherhost)", "(elsewhere)", 1), Exit: 21}, []string{missing}},
		{"TC-003 page requested with the refusal", refusalStub{Request: true, Stderr: refusal}, []string{opened}},
		{"TC-003 page requested without output", refusalStub{Request: true}, []string{opened, missing}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			executable := refusalStubBrowser(t, row.stub)
			run := runBrowserOnProfile(t, executable, profile, server, path, 2*time.Second)
			got := fixtureCheckFailure(run, "otherhost")

			if run.output != row.stub.Stderr {
				t.Errorf("captured output %q, want %q", run.output, row.stub.Stderr)
			}
			if run.requested != row.stub.Request {
				t.Errorf("page requested: %v, want %v", run.requested, row.stub.Request)
			}
			if err := syscall.Kill(-run.pid, 0); !errors.Is(err, syscall.ESRCH) {
				t.Errorf("the process group %d of the stub is still there: %v", run.pid, err)
				_ = syscall.Kill(-run.pid, syscall.SIGKILL)
			}

			if row.failed == nil {
				if got != "" {
					t.Fatalf("the check failed on a refusal without page request: %s", got)
				}
				return
			}
			if !strings.HasPrefix(got, "fixture check: ") {
				t.Errorf("the failure does not start with %q: %q", "fixture check: ", got)
			}
			for _, condition := range []string{opened, missing} {
				if named, want := strings.Contains(got, condition), slices.Contains(row.failed, condition); named != want {
					t.Errorf("condition %q named: %v, want %v, in %q", condition, named, want, got)
				}
			}
			if suffix := "browser output (" + strconv.Itoa(len(run.output)) + " bytes):\n" + run.output; !strings.HasSuffix(got, suffix) {
				t.Errorf("the failure does not end with the browser output %q: %q", suffix, got)
			}
			if want := "fixture check: " + strings.Join(row.failed, "; ") + "; browser output (" + strconv.Itoa(len(run.output)) + " bytes):\n" + run.output; got != want {
				t.Errorf("failure %q, want %q", got, want)
			}
		})
	}
}

// TC-002
func TestBrowserProfile_TC002_LockOfAnotherHostDoesNotPreventTheCheck(t *testing.T) {
	server := amazontest.NewServer(t, profileFixtureHandler(t))
	options := testBrowserOptions(t, server.URL)
	item := fixtureItem("B0PRICE001")
	assertPriceFound(t, "first check", browserCheck(t, NewBrowserCollector(options), item))

	writeStaleLock(t, options.ProfileDir, "otherhost")
	assertBrowserRefusesProfile(t, options.ExecPath, options.ProfileDir, "otherhost", server, "/dp/B0PRICE001")
	writeStaleLock(t, options.ProfileDir, "otherhost")

	server.Reset()
	output := captureOutput(t)
	goroutines := runtime.NumGoroutine()
	collector := NewBrowserCollector(options)
	got := browserCheck(t, collector, item)
	checkRequests := server.Requests()
	again := browserCheck(t, collector, item)
	assertNothingLeft(t, options.ProfileDir, goroutines)
	logged := output()
	assertPriceFound(t, "check after the lock of another host", got)
	assertPriceFound(t, "second check", again)
	if !pageCookieSent(checkRequests, "/dp/B0PRICE001") {
		t.Errorf("the page request does not carry the cookie of the earlier check: %+v", checkRequests)
	}
	if strings.Contains(logged, "browser could not start") {
		t.Errorf("a browser start failure was logged:\n%s", logged)
	}
}

// TC-003
func TestBrowserProfile_TC003_BrowserKilledOnThisMachineDoesNotPreventTheCheck(t *testing.T) {
	server := amazontest.NewServer(t, profileFixtureHandler(t))
	options := testBrowserOptions(t, server.URL)
	item := fixtureItem("B0PRICE001")
	collector := NewBrowserCollector(options)
	assertPriceFound(t, "first check", browserCheck(t, collector, item))

	killBrowserOnProfile(t, options.ExecPath, options.ProfileDir)
	server.Reset()
	output := captureOutput(t)
	goroutines := runtime.NumGoroutine()
	got := browserCheck(t, collector, item)
	assertNothingLeft(t, options.ProfileDir, goroutines)
	logged := output()
	assertPriceFound(t, "check after the killed browser", got)
	if !pageCookieSent(server.Requests(), "/dp/B0PRICE001") {
		t.Errorf("the cookie of the earlier check is not sent: %+v", server.Requests())
	}
	if strings.Contains(logged, "browser could not start") {
		t.Errorf("a browser start failure was logged:\n%s", logged)
	}

	t.Run("lock naming this host and a live process", func(t *testing.T) {
		host, err := os.Hostname()
		if err != nil {
			t.Fatal(err)
		}
		writeStaleLock(t, options.ProfileDir, "otherhost")
		lock := filepath.Join(options.ProfileDir, "SingletonLock")
		if err := os.Remove(lock); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(host+"-"+strconv.Itoa(os.Getpid()), lock); err != nil {
			t.Fatal(err)
		}
		server.Reset()
		got := browserCheck(t, collector, item)
		assertPriceFound(t, "check with a lock naming a live process of this host", got)
		if !pageCookieSent(server.Requests(), "/dp/B0PRICE001") {
			t.Errorf("the cookie of the earlier check is not sent: %+v", server.Requests())
		}
	})
}

// TC-004
func TestBrowserProfile_TC004_SuccessiveAbruptEndsRecoverEveryTime(t *testing.T) {
	server := amazontest.NewServer(t, profileFixtureHandler(t))
	options := testBrowserOptions(t, server.URL)
	collector := NewBrowserCollector(options)
	item := fixtureItem("B0PRICE001")
	for cycle := 1; cycle <= 2; cycle++ {
		killBrowserOnProfile(t, options.ExecPath, options.ProfileDir)
		assertPriceFound(t, "check after killed browser "+strconv.Itoa(cycle), browserCheck(t, collector, item))
	}
	writeStaleLock(t, options.ProfileDir, "otherhost")
	assertPriceFound(t, "check after a lock of another host", browserCheck(t, collector, item))
}

// TC-005
func TestBrowserProfile_TC005_ProfileWithoutStaleLockUnchanged(t *testing.T) {
	server := amazontest.NewServer(t, profileFixtureHandler(t))
	options := testBrowserOptions(t, server.URL)
	if _, err := os.Stat(options.ProfileDir); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("precondition: the profile directory exists: %v", err)
	}
	collector := NewBrowserCollector(options)
	item := fixtureItem("B0PRICE001")
	assertPriceFound(t, "first check", browserCheck(t, collector, item))
	server.Reset()
	assertPriceFound(t, "second check", browserCheck(t, collector, item))
	if !pageCookieSent(server.Requests(), "/dp/B0PRICE001") {
		t.Errorf("the second check does not carry the cookie set during the first: %+v", server.Requests())
	}

	if got := browserCheck(t, collector, fixtureItem("B0UNAVAIL1")); got.Result != "unavailable" || got.Message != "The listing is no longer available." {
		t.Errorf("unavailable listing: %+v", got)
	}
	if got := browserCheck(t, collector, fixtureItem("B0NOPRICE1")); got.Result != "price_not_found" {
		t.Errorf("price not found: %+v", got)
	}
}

// TC-006
func TestBrowserProfile_TC006_LockThatCannotBeDiscardedIsAFailure(t *testing.T) {
	server := amazontest.NewServer(t, profileFixtureHandler(t))
	options := testBrowserOptions(t, server.URL)
	wrapper, launches := launchCounter(t, options.ExecPath)
	options.ExecPath = wrapper
	lock := filepath.Join(options.ProfileDir, "SingletonLock")
	writeProfileFile(t, filepath.Join(lock, "keep"), "directory content")
	collector := NewBrowserCollector(options)

	output := captureOutput(t)
	goroutines := runtime.NumGoroutine()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	started := time.Now()
	got := collector.Collect(ctx, fixtureItem("B0PRICE001"))
	elapsed := time.Since(started)
	cancel()
	assertNothingLeft(t, options.ProfileDir, goroutines)
	logged := output()
	assertCheckFailed(t, got)
	if elapsed >= 20*time.Second {
		t.Fatalf("the failed check took %s", elapsed)
	}
	if !strings.Contains(logged, "browser could not start") {
		t.Errorf("the failure is not logged as a browser start failure:\n%s", logged)
	}
	entries, err := os.ReadDir(lock)
	if err != nil || len(entries) != 1 || entries[0].Name() != "keep" {
		t.Fatalf("the SingletonLock directory changed: %v %v", entries, err)
	}
	if data, err := os.ReadFile(filepath.Join(lock, "keep")); err != nil || string(data) != "directory content" {
		t.Fatalf("the SingletonLock directory content changed: %q %v", data, err)
	}
	if server.Count("/dp/B0PRICE001") != 0 || launches() != 0 {
		t.Fatalf("a browser was started although the profile could not be prepared: %d launches, %d page requests", launches(), server.Count("/dp/B0PRICE001"))
	}

	t.Run("missing browser with a stale lock", func(t *testing.T) {
		profile := filepath.Join(t.TempDir(), "profile")
		writeStaleLock(t, profile, "otherhost")
		got := stubCheck(t, filepath.Join(t.TempDir(), "missing-chromium"), profile)
		assertCheckFailed(t, got)
	})

	t.Run("later check after the directory is replaced by a stale lock", func(t *testing.T) {
		writeStaleLock(t, options.ProfileDir, "otherhost")
		assertPriceFound(t, "later check", browserCheck(t, collector, fixtureItem("B0PRICE001")))
	})
}

// launchCounter wraps the browser executable in a script that records each launch.
func launchCounter(t *testing.T, executable string) (string, func() int) {
	t.Helper()
	dir := t.TempDir()
	wrapper, launches := filepath.Join(dir, "counted-browser"), filepath.Join(dir, "launches")
	script := "#!/bin/sh\necho launch >> '" + launches + "'\nexec '" + executable + "' \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil { // #nosec G306 -- test executable
		t.Fatal(err)
	}
	return wrapper, func() int {
		data, err := os.ReadFile(launches)
		if err != nil {
			return 0
		}
		return strings.Count(string(data), "launch")
	}
}

// blockingServer serves the fixture pages; the product page of B0BLOCKED1 (the price-found page)
// waits for release, and each of its requests is reported on arrived.
func blockingServer(t *testing.T) (*amazontest.Server, <-chan struct{}, func()) {
	t.Helper()
	page := amazontest.Fixture(t, "price-found.html")
	arrived := make(chan struct{}, 8)
	release := make(chan struct{})
	var once sync.Once
	mux := http.NewServeMux()
	mux.Handle("/", fixtureHandler(t))
	mux.HandleFunc("/dp/B0BLOCKED1", func(w http.ResponseWriter, r *http.Request) {
		select {
		case arrived <- struct{}{}:
		default:
		}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, page)
	})
	server := amazontest.NewServer(t, mux)
	releaseAll := func() { once.Do(func() { close(release) }) }
	t.Cleanup(releaseAll) // runs before the server is closed
	return server, arrived, releaseAll
}

func waitArrived(t *testing.T, arrived <-chan struct{}) {
	t.Helper()
	select {
	case <-arrived:
	case <-time.After(time.Minute):
		t.Fatal("the blocked check did not reach the server")
	}
}

// TC-008
func TestBrowserProfile_TC008_CheckInProgressKeepsItsProfile(t *testing.T) {
	type outcome struct {
		name   string
		result model.CollectionResult
	}
	run := func(collector *BrowserCollector, ctx context.Context, asin string, results chan<- outcome) {
		go func() {
			results <- outcome{asin, collector.Collect(ctx, fixtureItem(asin))}
		}()
	}
	collect := func(t *testing.T, results <-chan outcome, count int) map[string]model.CollectionResult {
		t.Helper()
		got := map[string]model.CollectionResult{}
		for range count {
			select {
			case result := <-results:
				got[result.name] = result.result
			case <-time.After(3 * time.Minute):
				t.Fatal("a check did not end")
			}
		}
		return got
	}
	setUp := func(t *testing.T) (*BrowserCollector, BrowserOptions, *amazontest.Server, <-chan struct{}, func(), func() int) {
		t.Helper()
		server, arrived, release := blockingServer(t)
		options := testBrowserOptions(t, server.URL)
		wrapper, launches := launchCounter(t, options.ExecPath)
		options.ExecPath = wrapper
		return NewBrowserCollector(options), options, server, arrived, release, launches
	}
	background, cancelAll := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancelAll()

	t.Run("two checks", func(t *testing.T) {
		collector, options, server, arrived, release, launches := setUp(t)
		results := make(chan outcome, 2)
		run(collector, background, "B0BLOCKED1", results)
		waitArrived(t, arrived)
		lock := filepath.Join(options.ProfileDir, "SingletonLock")
		target, err := os.Readlink(lock)
		if err != nil {
			release()
			t.Fatalf("the running browser holds no SingletonLock: %v", err)
		}
		run(collector, background, "B0SCRIPT01", results)
		time.Sleep(2 * time.Second)
		after, err := os.Readlink(lock)
		received, launched := server.Count("/dp/B0SCRIPT01"), launches()
		release()
		if err != nil || after != target {
			t.Errorf("SingletonLock changed while the first check was running: %q -> %q (%v)", target, after, err)
		}
		if received != 0 || launched != 1 {
			t.Errorf("while the first check was running: %d requests for the second item, %d browser launches", received, launched)
		}
		got := collect(t, results, 2)
		if got["B0BLOCKED1"].Result != "success" || got["B0BLOCKED1"].AmountCents != 12999 {
			t.Errorf("first check: %+v", got["B0BLOCKED1"])
		}
		if got["B0SCRIPT01"].Result != "success" || got["B0SCRIPT01"].AmountCents != 4999 {
			t.Errorf("second check: %+v", got["B0SCRIPT01"])
		}
	})

	t.Run("three checks", func(t *testing.T) {
		collector, _, _, arrived, release, _ := setUp(t)
		results := make(chan outcome, 3)
		run(collector, background, "B0BLOCKED1", results)
		waitArrived(t, arrived)
		run(collector, background, "B0SCRIPT01", results)
		run(collector, background, "B0USEDOTH1", results)
		time.Sleep(time.Second)
		release()
		got := collect(t, results, 3)
		want := map[string]int64{"B0BLOCKED1": 12999, "B0SCRIPT01": 4999, "B0USEDOTH1": 7500}
		for asin, amount := range want {
			if got[asin].Result != "success" || got[asin].AmountCents != amount {
				t.Errorf("check of %s: %+v", asin, got[asin])
			}
		}
	})

	t.Run("check cancelled while waiting for the profile", func(t *testing.T) {
		collector, _, server, arrived, release, launches := setUp(t)
		results := make(chan outcome, 2)
		run(collector, background, "B0BLOCKED1", results)
		waitArrived(t, arrived)
		waiting, cancel := context.WithCancel(background)
		run(collector, waiting, "B0SCRIPT01", results)
		time.Sleep(time.Second)
		cancel()
		release()
		got := collect(t, results, 2)
		if got["B0BLOCKED1"].Result != "success" {
			t.Errorf("first check: %+v", got["B0BLOCKED1"])
		}
		assertCheckFailed(t, got["B0SCRIPT01"])
		if server.Count("/dp/B0SCRIPT01") != 0 || launches() != 1 {
			t.Errorf("the cancelled check started a browser: %d page requests, %d browser launches", server.Count("/dp/B0SCRIPT01"), launches())
		}
	})
}
