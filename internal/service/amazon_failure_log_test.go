package service

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/model"
)

// The Amazon failure line of issue #210 as the server logs it, read with the form documented in
// deploy/README.md.

// readmeLine returns the documented log line of deploy/README.md that starts with prefix.
func readmeLine(t *testing.T, prefix string) string {
	t.Helper()
	line, _, _, err := documentedLine(readRepositoryFile(t, "deploy/README.md"), prefix)
	if err != nil {
		t.Fatal(err)
	}
	return line
}

// loggedLines returns the logged lines holding prefix, from prefix on (the logger prefix dropped).
func loggedLines(logged, prefix string) []string {
	var lines []string
	for _, line := range strings.Split(logged, "\n") {
		if index := strings.Index(line, prefix); index >= 0 {
			lines = append(lines, line[index:])
		}
	}
	return lines
}

// documentedCaptures matches line with the pattern of documented and returns the captured cause and
// response (captures 4 and 5).
func documentedCaptures(t *testing.T, documented, line string) (cause, response string) {
	t.Helper()
	match := documentedPattern(documented).FindStringSubmatch(line)
	if match == nil {
		t.Fatalf("the documented form %q does not match the logged line %s", documented, shorten(line))
	}
	if len(match) != 6 {
		t.Fatalf("the documented form %q has %d values, want 5 (url, at, status, cause, response)", documented, len(match)-1)
	}
	return match[4], match[5]
}

// TC-004
func TestAmazonFailureLogDocs_TC004_DocumentedFieldOrderMatchesTheLoggedLine(t *testing.T) {
	const id = "B0LOGFA210"
	const cause = "Amazon returned HTTP 503"
	documented := readmeLine(t, failedLinePrefix)
	// check runs Refresh price of an Amazon item whose collector returns result and returns the log.
	check := func(t *testing.T, result model.CollectionResult) string {
		t.Helper()
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		insertPricedAmazonItem(t, database, id, scheduleDay(20, 0))
		service := startHuman(t, database, dir, newClock(scheduleDay(6, 0)),
			fakeAmazon(func(context.Context, model.Listing) model.CollectionResult { return result }), 210)
		logs := captureRetrievalLog(t)
		refreshPrice(t, service, id)
		return logs.String()
	}
	failed := func(excerpt string) model.CollectionResult {
		return model.CollectionResult{Result: "request_error", Message: cause, FailureCause: cause, RequestURL: "https://www.amazon.fr/dp/" + id,
			HTTPStatus: 503, ResponseExcerpt: excerpt}
	}
	// failureLine returns the only failure line of logged.
	failureLine := func(t *testing.T, logged string) string {
		t.Helper()
		lines := loggedLines(logged, failedLinePrefix)
		if len(lines) != 1 {
			t.Fatalf("%d lines starting with %q, want 1:\n%s", len(lines), failedLinePrefix, shorten(logged))
		}
		return lines[0]
	}

	t.Run("excerpt", func(t *testing.T) {
		const excerpt = "<html><body>Service Unavailable fixture</body></html>"
		line := failureLine(t, check(t, failed(excerpt)))
		if got, want := fieldNames(line), fieldNames(documented); !slices.Equal(got, want) {
			t.Errorf("the logged line has the fields %v, the documented line %v: %s", got, want, shorten(line))
		}
		gotCause, gotResponse := documentedCaptures(t, documented, line)
		if gotCause != cause || gotResponse != excerpt {
			t.Fatalf("documented form read cause %q and response %q, want %q and %q", gotCause, gotResponse, cause, excerpt)
		}
	})

	t.Run("empty excerpt", func(t *testing.T) {
		line := failureLine(t, check(t, failed("")))
		if !strings.HasSuffix(line, " response=") {
			t.Errorf("the logged line does not end with an empty response: %s", shorten(line))
		}
		if got, want := fieldNames(line), fieldNames(documented); !slices.Equal(got, want) {
			t.Errorf("the logged line has the fields %v, the documented line %v: %s", got, want, shorten(line))
		}
		gotCause, gotResponse := documentedCaptures(t, documented, line)
		if gotCause != cause || gotResponse != "" {
			t.Fatalf("documented form read cause %q and response %q, want %q and \"\"", gotCause, gotResponse, cause)
		}
	})

	t.Run("excerpt holding field texts", func(t *testing.T) {
		const excerpt = "<p>cause=fake status=999</p>"
		line := failureLine(t, check(t, failed(excerpt)))
		gotCause, gotResponse := documentedCaptures(t, documented, line)
		if gotCause != cause || gotResponse != excerpt {
			t.Fatalf("documented form read cause %q and response %q, want %q and %q", gotCause, gotResponse, cause, excerpt)
		}
	})

	t.Run("price not found", func(t *testing.T) {
		documentedNoPrice := readmeLine(t, noPriceLinePrefix)
		logged := check(t, model.CollectionResult{Result: "price_not_found", Message: "no price", FailureCause: "Amazon page showed no detectable euro price",
			RequestURL: "https://www.amazon.fr/dp/" + id, HTTPStatus: 200})
		if lines := loggedLines(logged, failedLinePrefix); len(lines) != 0 {
			t.Fatalf("a failure line was logged for a page without a price: %s", shorten(lines[0]))
		}
		lines := loggedLines(logged, noPriceLinePrefix)
		if len(lines) != 1 {
			t.Fatalf("%d lines starting with %q, want 1:\n%s", len(lines), noPriceLinePrefix, shorten(logged))
		}
		got, want := fieldNames(lines[0]), fieldNames(documentedNoPrice)
		if !slices.Equal(got, want) || slices.Contains(got, "response") {
			t.Fatalf("the logged no-price line has the fields %v, the documented line %v: %s", got, want, shorten(lines[0]))
		}
	})

	t.Run("success", func(t *testing.T) {
		if lines := loggedLines(check(t, succeeded()), failedLinePrefix); len(lines) != 0 {
			t.Fatalf("a failure line was logged for a successful check: %s", shorten(lines[0]))
		}
	})
}

// TC-005; set up like #204 TC-007, whose failure count and item states stay unchanged.
func TestAmazonFailureLogDocs_TC005_OperatorReadsRealFailuresWithTheDocumentedForm(t *testing.T) {
	documented := readmeLine(t, failedLinePrefix)
	browser := amazontest.Browser(t)
	_, server := newAmazonFixture(t, page(amazontest.Fixture(t, "robot-check-en.html")))
	missing := filepath.Join(t.TempDir(), "missing-chromium")
	const robot, absent = "B0RESPR210", "B0RESPA210"
	collectors := collectorByItem{
		robot:  retrievalBrowser(t, browser, server.URL, 30*time.Second),
		absent: retrievalBrowser(t, missing, server.URL, 30*time.Second),
	}
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	for id := range collectors {
		insertPricedAmazonItem(t, database, id, scheduleDay(20, 0))
	}
	now := newClock(scheduleDay(6, 0))
	service := startHuman(t, database, dir, now, collectors, 210)
	logs := captureRetrievalLog(t)

	// read runs Refresh price of id at 06:minute and returns the cause and response of its failure line.
	read := func(id string, minute int) (string, string) {
		t.Helper()
		at := scheduleDay(6, minute)
		now.Set(at)
		refreshPrice(t, service, id)
		line := checkLogLine(t, logs.String(), id, at)
		index := strings.Index(line, failedLinePrefix)
		if index < 0 {
			t.Fatalf("the check of %s logged no failure line: %s", id, shorten(line))
		}
		return documentedCaptures(t, documented, line[index:])
	}
	assertRobotCheck := func(name, cause, response string) {
		t.Helper()
		if !strings.Contains(strings.ToLower(cause), "robot check") {
			t.Errorf("%s: cause %q does not say robot check", name, cause)
		}
		for _, pageText := range []string{"<", "Enter the characters", "fixture-amzn-204"} {
			if strings.Contains(cause, pageText) {
				t.Errorf("%s: cause %q holds the page text %q", name, cause, pageText)
			}
		}
		if response == "" || len(response) > 2048 || !strings.HasPrefix(response, "<html") || !strings.Contains(response, `<title dir="ltr">Amazon.fr</title>`) {
			t.Errorf("%s: response is not the start of the fixture page (%d bytes): %.300s", name, len(response), response)
		}
	}

	cause, response := read(robot, 10)
	assertRobotCheck("(a) robot check", cause, response)

	cause, response = read(absent, 11)
	if !strings.Contains(cause, "browser could not start") || !strings.Contains(cause, missing) {
		t.Errorf("(b) missing browser: cause %q does not say the browser could not start with %s", cause, missing)
	}
	if response != "" {
		t.Errorf("(b) missing browser: response %.300q, want empty", response)
	}

	cause, response = read(robot, 12)
	assertRobotCheck("(a) robot check repeated", cause, response)

	if got := amazonFailures(t, database); got != 3 {
		t.Errorf("consecutive failure count %d, want 3", got)
	}
	for id := range collectors {
		item := storedItem(t, database, id)
		if item.Status != "retrieval_error" || item.LatestPrice == nil || item.LatestPrice.AmountCents != 1999 || item.AmazonStoppedAt != nil {
			t.Errorf("%s: status %s, latest price %+v, stopped %v", id, item.Status, item.LatestPrice, item.AmazonStoppedAt)
		}
	}
	if strings.Contains(logs.String(), retrievalSecret) {
		t.Fatalf("a cookie value reached the log:\n%s", shorten(logs.String()))
	}
}
