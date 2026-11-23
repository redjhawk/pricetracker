package amazontest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// Issue #232: the no-download check fails on files it cannot read, ignores files that disappear
// before they are read, and judges everything else as before.

// recordingTB records the failure of a check instead of failing the test. Fatal and Fatalf stop the
// calling goroutine, as testing.T does.
type recordingTB struct {
	testing.TB
	failed  bool
	message string
}

func (r *recordingTB) Error(args ...any) {
	r.failed = true
	r.message += fmt.Sprint(args...)
}

func (r *recordingTB) Errorf(format string, args ...any) {
	r.failed = true
	r.message += fmt.Sprintf(format, args...)
}

func (r *recordingTB) Fatal(args ...any) {
	r.Error(args...)
	runtime.Goexit()
}

func (r *recordingTB) Fatalf(format string, args ...any) {
	r.Errorf(format, args...)
	runtime.Goexit()
}

// runCheck runs AssertNoDownload with a recordingTB and returns whether it failed and with which message.
func runCheck(t *testing.T, names []string, roots ...string) (bool, string) {
	t.Helper()
	recorder := &recordingTB{TB: t}
	done := make(chan struct{})
	go func() {
		defer close(done)
		AssertNoDownload(recorder, names, roots...)
	}()
	<-done
	return recorder.failed, recorder.message
}

// writeFile creates the file at path, and its folders, holding content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// makeUnreadable sets the mode of the file at path to 000 until the end of the test. It skips the test
// when the file can still be read (running as root).
func makeUnreadable(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	if _, err := os.ReadFile(path); err == nil { // #nosec G304 -- file of the test directory
		t.Skip("file still readable after chmod 000 (running as root?)")
	}
}

// assertReadFailure checks that the check failed because of the unreadable file at path.
func assertReadFailure(t *testing.T, failed bool, message, path string) {
	t.Helper()
	if !failed {
		t.Fatalf("the check passed with the unreadable file %s", path)
	}
	if !strings.Contains(message, path) {
		t.Errorf("the failure message does not name %s: %q", path, message)
	}
	if !strings.Contains(message, "permission denied") {
		t.Errorf("the failure message does not give the read error: %q", message)
	}
	if strings.Contains(message, "files left by a download") {
		t.Errorf("the read failure is reported as a download: %q", message)
	}
}

// assertDownloads checks that the check failed with the download message naming every path with its reason.
func assertDownloads(t *testing.T, failed bool, message string, reasons map[string]string) {
	t.Helper()
	if !failed {
		t.Fatalf("the check passed; want downloads %v", reasons)
	}
	if !strings.Contains(message, "files left by a download") {
		t.Errorf("the failure message is not the download message: %q", message)
	}
	for path, reason := range reasons {
		if !strings.Contains(message, path+" ("+reason+")") {
			t.Errorf("the failure message does not give %s (%s): %q", path, reason, message)
		}
	}
}

func assertPasses(t *testing.T, failed bool, message string) {
	t.Helper()
	if failed {
		t.Fatalf("the check failed: %q", message)
	}
}

// TC-001
func TestAssertNoDownload_TC001_FailsOnUnreadableFile(t *testing.T) {
	names := []string{"page.html"}

	t.Run("marker", func(t *testing.T) {
		folder := t.TempDir()
		path := filepath.Join(folder, "Preferences")
		writeFile(t, path, "<html>"+DownloadMarker+"</html>")
		makeUnreadable(t, path)
		failed, message := runCheck(t, names, folder)
		assertReadFailure(t, failed, message, path)
	})

	t.Run("no marker", func(t *testing.T) {
		folder := t.TempDir()
		path := filepath.Join(folder, "Preferences")
		writeFile(t, path, `{"profile":{"name":"Person 1"}}`)
		makeUnreadable(t, path)
		failed, message := runCheck(t, names, folder)
		assertReadFailure(t, failed, message, path)
	})

	t.Run("nested", func(t *testing.T) {
		folder := t.TempDir()
		path := filepath.Join(folder, "Default", "Cache", "data_1")
		writeFile(t, path, DownloadMarker)
		makeUnreadable(t, path)
		failed, message := runCheck(t, names, folder)
		assertReadFailure(t, failed, message, path)
	})

	t.Run("second root", func(t *testing.T) {
		first := t.TempDir()
		writeFile(t, filepath.Join(first, "Preferences"), "{}")
		second := t.TempDir()
		path := filepath.Join(second, "Preferences")
		writeFile(t, path, DownloadMarker)
		makeUnreadable(t, path)
		failed, message := runCheck(t, names, first, second)
		assertReadFailure(t, failed, message, path)
	})

	t.Run("clean neighbours", func(t *testing.T) {
		folder := t.TempDir()
		writeFile(t, filepath.Join(folder, "Cookies"), "cookies")
		writeFile(t, filepath.Join(folder, "Local State"), "{}")
		writeFile(t, filepath.Join(folder, "Default", "History"), "history")
		path := filepath.Join(folder, "Preferences")
		writeFile(t, path, "{}")
		makeUnreadable(t, path)
		failed, message := runCheck(t, names, folder)
		assertReadFailure(t, failed, message, path)
	})

	t.Run("repeated call", func(t *testing.T) {
		folder := t.TempDir()
		path := filepath.Join(folder, "Preferences")
		writeFile(t, path, DownloadMarker)
		makeUnreadable(t, path)
		failed, message := runCheck(t, names, folder)
		assertReadFailure(t, failed, message, path)
		againFailed, againMessage := runCheck(t, names, folder)
		assertReadFailure(t, againFailed, againMessage, path)
		if againMessage != message {
			t.Errorf("the second call failed with %q; want %q", againMessage, message)
		}
	})
}

// TC-002
func TestAssertNoDownload_TC002_ReportsUnreadableDownloadByName(t *testing.T) {
	for _, row := range []struct{ name, reason string }{
		{"page.html", "named after the attachment"},
		{"x.crdownload", "partial download"},
		{"download.html", "default download name"},
	} {
		t.Run(row.name, func(t *testing.T) {
			folder := t.TempDir()
			path := filepath.Join(folder, row.name)
			writeFile(t, path, "<html>page</html>")
			makeUnreadable(t, path)
			failed, message := runCheck(t, []string{"page.html"}, folder)
			assertDownloads(t, failed, message, map[string]string{path: row.reason})
			if strings.Contains(message, "permission denied") {
				t.Errorf("a file named as a download was read: %q", message)
			}
		})
	}
}

// TC-003
func TestDownloadFile_TC003_IgnoresRemovedFile(t *testing.T) {
	patterns := []*regexp.Regexp{downloadNamePattern("page.html")}

	for _, content := range []string{DownloadMarker, "{}"} {
		t.Run(fmt.Sprintf("content %q", content), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "Preferences")
			writeFile(t, path, content)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			reason, err := downloadFile(path, "Preferences", patterns)
			if reason != "" || err != nil {
				t.Errorf("downloadFile of a removed file = (%q, %v); want (\"\", nil)", reason, err)
			}
		})
	}

	t.Run("removed parent", func(t *testing.T) {
		parent := filepath.Join(t.TempDir(), "Default")
		path := filepath.Join(parent, "Preferences")
		writeFile(t, path, DownloadMarker)
		if err := os.RemoveAll(parent); err != nil {
			t.Fatal(err)
		}
		reason, err := downloadFile(path, "Preferences", patterns)
		if reason != "" || err != nil {
			t.Errorf("downloadFile under a removed folder = (%q, %v); want (\"\", nil)", reason, err)
		}
	})
}

// TC-004
func TestAssertNoDownload_TC004_JudgesReadableFiles(t *testing.T) {
	names := []string{"page.html", "B0PRICE001"}

	// downloads lists files, one folder per row, that the check reports with their reason.
	downloads := []struct {
		row    string
		files  map[string]string // file name -> content
		reason string
	}{
		{"a page.html", map[string]string{"page.html": "<html>page</html>"}, "named after the attachment"},
		{"b page (1).html", map[string]string{"page (1).html": "<html>page</html>"}, "named after the attachment"},
		{"c B0PRICE001.html", map[string]string{"B0PRICE001.html": "<html>page</html>"}, "named after the attachment"},
		{"d download names", map[string]string{"download": "x", "downloaded.bin": "x"}, "default download name"},
		{"e x.crdownload", map[string]string{"x.crdownload": "x"}, "partial download"},
		{"e Unconfirmed", map[string]string{"Unconfirmed 123.crdownload": "x"}, "partial download"},
		{"e Chrome", map[string]string{".com.google.Chrome.abc": "x"}, "partial download"},
		{"e Chromium", map[string]string{".org.chromium.Chromium.abc": "x"}, "partial download"},
		{"f marker", map[string]string{"Cookies": "cookies " + DownloadMarker}, "holds the downloaded body"},
	}
	for _, download := range downloads {
		t.Run(download.row, func(t *testing.T) {
			folder := t.TempDir()
			reasons := map[string]string{}
			for name, content := range download.files {
				path := filepath.Join(folder, name)
				writeFile(t, path, content)
				reasons[path] = download.reason
			}
			failed, message := runCheck(t, names, folder)
			assertDownloads(t, failed, message, reasons)
		})
	}

	t.Run("g clean folder", func(t *testing.T) {
		folder := t.TempDir()
		writeFile(t, filepath.Join(folder, "Preferences"), `{"profile":{"name":"Person 1"}}`)
		writeFile(t, filepath.Join(folder, "Empty"), "")
		if err := os.Mkdir(filepath.Join(folder, "Empty folder"), 0o700); err != nil {
			t.Fatal(err)
		}
		failed, message := runCheck(t, names, folder)
		assertPasses(t, failed, message)
	})

	t.Run("h missing root", func(t *testing.T) {
		failed, message := runCheck(t, names, filepath.Join(t.TempDir(), "missing"))
		assertPasses(t, failed, message)
	})

	t.Run("i link to a file", func(t *testing.T) {
		outside := filepath.Join(t.TempDir(), "Cookies")
		writeFile(t, outside, DownloadMarker)
		folder := t.TempDir()
		if err := os.Mkdir(filepath.Join(folder, "Default"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(folder, "Default", "link")); err != nil {
			t.Fatal(err)
		}
		failed, message := runCheck(t, names, folder)
		assertPasses(t, failed, message)
	})

	t.Run("j link to a folder", func(t *testing.T) {
		outside := t.TempDir()
		writeFile(t, filepath.Join(outside, "page.html"), DownloadMarker)
		folder := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(folder, "link")); err != nil {
			t.Fatal(err)
		}
		failed, message := runCheck(t, names, folder)
		assertPasses(t, failed, message)
	})

	t.Run("no roots", func(t *testing.T) {
		failed, message := runCheck(t, names)
		assertPasses(t, failed, message)
	})

	t.Run("several downloads", func(t *testing.T) {
		folder := t.TempDir()
		reasons := map[string]string{
			filepath.Join(folder, "page.html"):                  "named after the attachment",
			filepath.Join(folder, "download"):                   "default download name",
			filepath.Join(folder, "x.crdownload"):               "partial download",
			filepath.Join(folder, "Default", "Cache", "data_1"): "holds the downloaded body",
		}
		for path := range reasons {
			writeFile(t, path, DownloadMarker)
		}
		failed, message := runCheck(t, names, folder)
		assertDownloads(t, failed, message, reasons)
	})

	for _, name := range []string{"page (1).htm", "page.html.bak"} {
		t.Run("not derived "+name, func(t *testing.T) {
			folder := t.TempDir()
			writeFile(t, filepath.Join(folder, name), "<html>page</html>")
			failed, message := runCheck(t, names, folder)
			assertPasses(t, failed, message)
		})
	}
}

// TC-005
func TestAssertNoDownload_TC005_PassesOnBrowserProfile(t *testing.T) {
	profile := t.TempDir()
	writeFile(t, filepath.Join(profile, "Local State"), `{"browser":{}}`)
	writeFile(t, filepath.Join(profile, "Default", "Preferences"), `{"profile":{"name":"Person 1"}}`)
	writeFile(t, filepath.Join(profile, "Default", "Cache", "Cache_Data", "index"), "index")
	writeFile(t, filepath.Join(profile, "Default", "Current Session"), "")
	if err := os.Symlink("runner-host-12345", filepath.Join(profile, "SingletonLock")); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(filepath.Join(home, ".config"), 0o700); err != nil {
		t.Fatal(err)
	}

	failed, message := runCheck(t, []string{"page.html"}, profile, home)
	assertPasses(t, failed, message)
}

// TC-006
func TestAssertNoDownload_TC006_ChangeLeavesApplicationUntouched(t *testing.T) {
	// Issue #208: the leftover scratch file stays absent.
	scratch := filepath.Join("..", "..", "service", "zz_scratch_docs_test.go")
	if _, err := os.Stat(scratch); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("%s exists or cannot be checked: %v", scratch, err)
	}

	// The application never imports the test helper, so changing it does not change the server.
	const importPath = `"pricefollower.local/internal/amazon/amazontest"`
	for _, tree := range []string{"cmd", "internal"} {
		root := filepath.Join("..", "..", "..", tree)
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path) // #nosec G304 -- Go file of the repository
			if err != nil {
				return err
			}
			if strings.Contains(string(data), importPath) {
				t.Errorf("%s imports the test helper %s", path, importPath)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("list the Go files of %s: %v", root, err)
		}
	}
}
