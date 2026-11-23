package amazon

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The browser lookup of issue #226: the documentation change keeps the order in which FindBrowser
// looks for the browser.

// browserStubs writes an executable stub for each name in a new directory and returns the directory.
func browserStubs(t *testing.T, names ...string) string {
	t.Helper()
	directory := t.TempDir()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil { // #nosec G306 -- test executable
			t.Fatal(err)
		}
	}
	return directory
}

// TC-003
func TestAmazonBrowserSnapDocs_TC003_BrowserLookupOrderIsUnchanged(t *testing.T) {
	if want := []string{"chromium", "chromium-browser", "google-chrome-stable", "google-chrome"}; !slices.Equal(BrowserNames, want) {
		t.Fatalf("BrowserNames is %v, want %v", BrowserNames, want)
	}

	cases := []struct {
		name    string
		onPath  []string
		setting func(directory string) string // value of PRICEFOLLOWER_BROWSER_PATH
		want    func(directory string) string
	}{
		{"chromium and google-chrome-stable", []string{"chromium", "google-chrome-stable"},
			func(string) string { return "" }, func(directory string) string { return filepath.Join(directory, "chromium") }},
		{"chromium-browser and google-chrome", []string{"chromium-browser", "google-chrome"},
			func(string) string { return "" }, func(directory string) string { return filepath.Join(directory, "chromium-browser") }},
		{"only google-chrome-stable", []string{"google-chrome-stable"},
			func(string) string { return "" }, func(directory string) string { return filepath.Join(directory, "google-chrome-stable") }},
		{"setting to google-chrome-stable with chromium on PATH", []string{"chromium", "google-chrome-stable"},
			func(directory string) string { return filepath.Join(directory, "google-chrome-stable") },
			func(directory string) string { return filepath.Join(directory, "google-chrome-stable") }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			directory := browserStubs(t, test.onPath...)
			t.Setenv("PATH", directory)
			t.Setenv(BrowserPathVariable, test.setting(directory))
			got, err := FindBrowser()
			if err != nil {
				t.Fatal(err)
			}
			if want := test.want(directory); got != want {
				t.Fatalf("FindBrowser() = %q, want %q", got, want)
			}
		})
	}

	t.Run("setting unset", func(t *testing.T) {
		directory := browserStubs(t, "chromium", "google-chrome-stable")
		t.Setenv("PATH", directory)
		t.Setenv(BrowserPathVariable, "")
		if err := os.Unsetenv(BrowserPathVariable); err != nil {
			t.Fatal(err)
		}
		got, err := FindBrowser()
		if err != nil || got != filepath.Join(directory, "chromium") {
			t.Fatalf("FindBrowser() = %q, %v, want the chromium stub", got, err)
		}
	})

	t.Run("no browser", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		t.Setenv(BrowserPathVariable, "")
		if got, err := FindBrowser(); err == nil {
			t.Fatalf("FindBrowser() = %q without a browser, want an error", got)
		}
	})
}
