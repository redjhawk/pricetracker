package httpapi

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"pricefollower.local/internal/amazon/amazontest"
)

// Amazon answers offered as a file download are refused by the browser (issue #217): through the HTTP
// API, a real browser collector with its persistent profile in the data directory writes no file of them.

// dataFiles returns the regular files of dir outside the browser profile, as paths relative to dir.
func dataFiles(t *testing.T, dir string) map[string]bool {
	t.Helper()
	files := map[string]bool{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if entry.IsDir() && relative == "amazon-browser" {
			return filepath.SkipDir
		}
		if entry.Type().IsRegular() {
			files[relative] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// assertNoNewDataFile checks that the data directory of api holds no regular file outside the browser
// profile that was not in before, other than the SQLite database files.
func assertNoNewDataFile(t *testing.T, api navigationAPI, before map[string]bool, step string) {
	t.Helper()
	for path := range dataFiles(t, api.dir) {
		if !before[path] && !strings.HasPrefix(filepath.Base(path), "pricefollower.sqlite") {
			t.Fatalf("%s: new file %s in the data directory", step, path)
		}
	}
}

// Refresh price and Refresh all prices answered with a download write nothing in the data directory
// profile, under HOME or elsewhere in the data directory; the normal page then records its price.
// TC-003
func TestAmazonBrowserDownload_TC003_RefreshAnsweredWithADownloadWritesNoFile(t *testing.T) {
	const id = "B0DOWN0031"
	api := openNavigationAPI(t, []string{id})
	home := amazontest.IsolatedHome(t)
	before := dataFiles(t, api.dir)
	api.fixture.answer.Store(answerAttachment)

	refresh(t, api.server, id)
	assertRetrievalIssue(t, waitBrowserCheck(t, api.server, id, humanTime(14, 0)), humanTime(14, 0))
	assertRetrievalIssue(t, listItems(t, api.server)[id], humanTime(14, 0))
	assertNoDownloadWritten(t, api, home)
	assertNoNewDataFile(t, api, before, "after Refresh price")

	api.clock.Set(humanTime(14, 5))
	if recorder := callPath(api.server, http.MethodPost, "/api/v1/items/refresh", ""); recorder.Code != http.StatusAccepted {
		t.Fatalf("Refresh all prices: %d %s", recorder.Code, recorder.Body.String())
	}
	assertRetrievalIssue(t, waitBrowserCheck(t, api.server, id, humanTime(14, 5)), humanTime(14, 5))
	assertRetrievalIssue(t, listItems(t, api.server)[id], humanTime(14, 5))
	assertNoDownloadWritten(t, api, home)
	assertNoNewDataFile(t, api, before, "after Refresh all prices")

	api.fixture.answer.Store(answerPage)
	api.clock.Set(humanTime(14, 10))
	refresh(t, api.server, id)
	item := waitBrowserCheck(t, api.server, id, humanTime(14, 10))
	if item.LastAttempt.Result != "success" {
		t.Fatalf("check of the normal page after the downloads: last attempt %+v", item.LastAttempt)
	}
	assertBrowserPriceListed(t, item, humanTime(14, 10))
	assertNoDownloadWritten(t, api, home)
	assertNoNewDataFile(t, api, before, "after the normal page")
}
