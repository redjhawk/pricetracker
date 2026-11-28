package service

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"pricefollower.local/internal/amazon"
	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// Amazon browser profile recovery of issue #180 through the service: a real browser collector on a
// profile holding lock marks left by a killed browser or by another host, with SQLite and a fake clock.

// profileServer serves the profile fixture pages of amazontest.ProfileHandler. Each /dp/<ASIN>
// request is reported on arrived, and the page of blocked waits for the returned release function.
func profileServer(t *testing.T, blocked string) (*amazontest.Server, <-chan string, func()) {
	t.Helper()
	pages := amazontest.ProfileHandler(t)
	arrived := make(chan string, 16)
	release := make(chan struct{})
	var once sync.Once
	server := amazontest.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if asin, ok := strings.CutPrefix(r.URL.Path, "/dp/"); ok {
			select {
			case arrived <- asin:
			default:
			}
			if asin == blocked {
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
			}
		}
		pages.ServeHTTP(w, r)
	}))
	releaseAll := func() { once.Do(func() { close(release) }) }
	t.Cleanup(releaseAll) // runs before the server is closed
	return server, arrived, releaseAll
}

func profileCollector(t *testing.T, profile, origin string) *amazon.BrowserCollector {
	t.Helper()
	return amazon.NewBrowserCollector(amazon.BrowserOptions{ExecPath: amazontest.Browser(t), ProfileDir: profile, NoSandbox: true,
		NavigationTimeout: 30 * time.Second, Pause: func() time.Duration { return 0 }, Origin: origin})
}

// leaveLockDirectory makes SingletonLock of profile a non-empty directory, which cannot be removed.
func leaveLockDirectory(t *testing.T, profile string) string {
	t.Helper()
	lock := filepath.Join(profile, "SingletonLock")
	if err := os.MkdirAll(lock, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lock, "keep"), []byte("directory content"), 0o600); err != nil {
		t.Fatal(err)
	}
	return lock
}

type profileOutput struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (o *profileOutput) Write(data []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.buffer.Write(data)
}

func (o *profileOutput) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.buffer.String()
}

// captureProfileOutput collects the log and the standard output (where the Amazon collector prints
// its failures) until the end of the test.
func captureProfileOutput(t *testing.T) *profileOutput {
	t.Helper()
	output := &profileOutput{}
	previousLog := log.Writer()
	log.SetOutput(output)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previousStdout := os.Stdout
	os.Stdout = writer
	copied := make(chan struct{})
	go func() { io.Copy(output, reader); close(copied) }()
	t.Cleanup(func() {
		os.Stdout = previousStdout
		writer.Close()
		<-copied
		reader.Close()
		log.SetOutput(previousLog)
	})
	return output
}

// waitStartFailures waits until want browser start failures are printed and checks there are no more.
// It counts the collector's line only: other log lines (the request gate) may repeat the cause.
func waitStartFailures(t *testing.T, output *profileOutput, want int) {
	t.Helper()
	count := func() int {
		return countLines(output.String(), func(line string) bool {
			return strings.HasPrefix(line, "Amazon collection failed for ") && strings.Contains(line, "browser could not start")
		})
	}
	for deadline := time.Now().Add(5 * time.Second); count() < want && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	if got := count(); got != want {
		t.Fatalf("%d browser start failures logged, want %d:\n%s", got, want, output.String())
	}
}

// waitChecksDone waits until no check of the service is running (browser checks take longer than waitIdle allows).
func waitChecksDone(t *testing.T, service *Service) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Minute); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		service.mu.Lock()
		idle := len(service.inFlight) == 0
		service.mu.Unlock()
		if idle {
			return
		}
	}
	t.Fatal("the checks did not finish")
}

func storedItem(t *testing.T, database *store.Store, id string) model.Item {
	t.Helper()
	item, err := database.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

// assertFixturePriceRecorded checks that the fixture price was recorded for id at the check time at.
func assertFixturePriceRecorded(t *testing.T, database *store.Store, id string, at time.Time) {
	t.Helper()
	item := storedItem(t, database, id)
	if item.LastAttempt == nil || item.LastAttempt.Result != "success" || !item.LastAttempt.Timestamp.Equal(at) {
		t.Fatalf("item %s: last attempt %+v, want a success at %s", id, item.LastAttempt, at.In(paris))
	}
	if item.LatestPrice == nil || item.LatestPrice.AmountCents != 12999 || item.LatestPrice.Timestamp == nil || !item.LatestPrice.Timestamp.Equal(at) {
		t.Fatalf("item %s: latest price %+v, want 129.99 at %s", id, item.LatestPrice, at.In(paris))
	}
	if item.Status != "active" || item.AmazonStoppedAt != nil {
		t.Fatalf("item %s: status %s, stopped %v", id, item.Status, item.AmazonStoppedAt)
	}
}

func recordEarlierPrice(t *testing.T, database *store.Store, id string) {
	t.Helper()
	if err := database.RecordCollection(context.Background(), id, model.CollectionResult{Result: "success", AmountCents: 1999, SecondHandStatus: "not_found"},
		scheduleDay(8, 0).AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
}

// TC-007
func TestAmazonBrowserProfile_TC007_FailureIsLoggedCountedAndStopsItems(t *testing.T) {
	server, _, _ := profileServer(t, "")

	t.Run("five failures stop the items", func(t *testing.T) {
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		profile := filepath.Join(t.TempDir(), "profile")
		leaveLockDirectory(t, profile)
		const other = "B0PROF0072"
		insertAt(t, database, other, "amazon", scheduleDay(20, 0))
		now := newClock(scheduleDay(7, 30))
		service := startHuman(t, database, dir, now, profileCollector(t, profile, server.URL), 7)
		output := captureProfileOutput(t)
		ctx := context.Background()

		added, err := service.Add(ctx, "https://www.amazon.fr/dp/B0PROF0071", "")
		if err != nil {
			t.Fatal(err)
		}
		waitFor(t, "the first check after add", func() bool { return storedItem(t, database, added.ID).LastAttempt != nil })
		waitChecksDone(t, service)
		steps := []struct {
			name  string
			id    string
			check func()
		}{
			{"first check after add", added.ID, func() {}},
			{"Refresh price", added.ID, func() {
				if _, err := service.RefreshItem(ctx, added.ID); err != nil {
					t.Fatal(err)
				}
				waitChecksDone(t, service)
			}},
			{"Refresh price of the other item", other, func() {
				if _, err := service.RefreshItem(ctx, other); err != nil {
					t.Fatal(err)
				}
				waitChecksDone(t, service)
			}},
			{"scheduled check", added.ID, func() {
				if err := database.SetNextCheck(ctx, added.ID, now.Now().Add(-time.Minute)); err != nil {
					t.Fatal(err)
				}
				service.collectDue(ctx)
			}},
			{"scheduled check of the other item", other, func() {
				if err := database.SetNextCheck(ctx, other, now.Now().Add(-time.Minute)); err != nil {
					t.Fatal(err)
				}
				service.collectDue(ctx)
			}},
		}
		for index, step := range steps {
			failure := index + 1
			step.check()
			waitStartFailures(t, output, failure)
			item := storedItem(t, database, step.id)
			if item.LastAttempt == nil || item.LastAttempt.Result != "request_error" || item.LastAttempt.Message == nil || *item.LastAttempt.Message != amazonCheckFailedText {
				t.Fatalf("failure %d (%s): last attempt %+v", failure, step.name, item.LastAttempt)
			}
			if failure < amazonFailureLimit {
				if item.Status != "retrieval_error" {
					t.Fatalf("failure %d (%s): status %s, want a retrieval issue", failure, step.name, item.Status)
				}
				if got := amazonFailures(t, database); got != failure {
					t.Fatalf("failure %d (%s): consecutive failure count %d", failure, step.name, got)
				}
				for _, id := range []string{added.ID, other} {
					if stopped(t, database, id) {
						t.Fatalf("item %s stopped after failure %d", id, failure)
					}
				}
				continue
			}
			for _, id := range []string{added.ID, other} {
				if !stopped(t, database, id) {
					t.Fatalf("item %s not stopped after the 5th failure", id)
				}
			}
		}
	})

	t.Run("a stale lock after four failures is recovered", func(t *testing.T) {
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		profile := filepath.Join(t.TempDir(), "profile")
		lock := leaveLockDirectory(t, profile)
		const id = "B0PROF0073"
		insertAt(t, database, id, "amazon", scheduleDay(20, 0))
		now := newClock(scheduleDay(14, 0))
		service := startHuman(t, database, dir, now, profileCollector(t, profile, server.URL), 73)
		output := captureProfileOutput(t)
		ctx := context.Background()
		for failure := 1; failure <= 4; failure++ {
			now.Set(scheduleDay(14, failure))
			service.Collect(ctx, id)
			waitStartFailures(t, output, failure)
		}
		if got := amazonFailures(t, database); got != 4 {
			t.Fatalf("consecutive failure count %d after 4 failures", got)
		}
		if err := os.RemoveAll(lock); err != nil {
			t.Fatal(err)
		}
		amazontest.StaleLock(t, profile, "otherhost")
		now.Set(scheduleDay(14, 30))
		service.Collect(ctx, id)
		assertFixturePriceRecorded(t, database, id, scheduleDay(14, 30))
		if got := amazonFailures(t, database); got != 0 {
			t.Fatalf("consecutive failure count %d after the recovered check, want 0", got)
		}
	})
}

const amazonCheckFailedText = "Amazon could not be reached for a price check."

// TC-009
func TestAmazonBrowserProfile_TC009_ScheduledCheckAndRefreshAfterKilledBrowsers(t *testing.T) {
	t.Run("scheduled check then Refresh price", func(t *testing.T) {
		server, _, _ := profileServer(t, "")
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		const id = "B0PROF0091"
		insertAt(t, database, id, "amazon", scheduleDay(7, 0))
		recordEarlierPrice(t, database, id)
		if err := database.SaveAmazonFailures(context.Background(), 2); err != nil {
			t.Fatal(err)
		}
		profile := filepath.Join(t.TempDir(), "profile")
		amazontest.KillBrowserOnProfile(t, profile)
		now := newClock(scheduleDay(7, 5))
		service := startHuman(t, database, dir, now, profileCollector(t, profile, server.URL), 9)
		output := captureProfileOutput(t)
		ctx := context.Background()

		service.collectDue(ctx)
		assertFixturePriceRecorded(t, database, id, scheduleDay(7, 5))
		if got := amazonFailures(t, database); got != 0 {
			t.Fatalf("consecutive failure count %d after the scheduled check, want 0", got)
		}
		next := nextCheck(t, database, id)
		if windowOf(next, paris) != evening || localDate(next, paris) != "2030-03-05" {
			t.Fatalf("next check at %s, want the evening window", next.In(paris))
		}

		amazontest.KillBrowserOnProfile(t, profile)
		now.Set(scheduleDay(10, 0))
		if _, err := service.RefreshItem(ctx, id); err != nil {
			t.Fatal(err)
		}
		waitChecksDone(t, service)
		assertFixturePriceRecorded(t, database, id, scheduleDay(10, 0))
		if got := amazonFailures(t, database); got != 0 {
			t.Fatalf("consecutive failure count %d after Refresh price, want 0", got)
		}
		if got := nextCheck(t, database, id); !got.Equal(next) {
			t.Fatalf("Refresh price moved the next check from %s to %s", next.In(paris), got.In(paris))
		}
		if logged := output.String(); strings.Contains(logged, "browser could not start") {
			t.Fatalf("a browser start failure was logged:\n%s", logged)
		}
	})

	t.Run("Refresh price during a blocked scheduled check", func(t *testing.T) {
		const scheduled, refreshed = "B0PROF0092", "B0PROF0093"
		server, arrived, release := profileServer(t, scheduled)
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		insertAt(t, database, scheduled, "amazon", scheduleDay(7, 0))
		insertAt(t, database, refreshed, "amazon", scheduleDay(20, 0))
		profile := filepath.Join(t.TempDir(), "profile")
		amazontest.KillBrowserOnProfile(t, profile)
		now := newClock(scheduleDay(7, 5))
		service := startHuman(t, database, dir, now, profileCollector(t, profile, server.URL), 92)
		ctx := context.Background()
		done := make(chan struct{})
		go func() { service.collectDue(ctx); close(done) }()
		select {
		case asin := <-arrived:
			if asin != scheduled {
				release()
				t.Fatalf("first page request for %s, want %s", asin, scheduled)
			}
		case <-time.After(time.Minute):
			release()
			t.Fatal("the scheduled check did not reach the server")
		}
		if _, err := service.RefreshItem(ctx, refreshed); err != nil {
			release()
			t.Fatal(err)
		}
		time.Sleep(500 * time.Millisecond)
		release()
		select {
		case <-done:
		case <-time.After(2 * time.Minute):
			t.Fatal("the scheduled check did not end")
		}
		waitChecksDone(t, service)
		assertFixturePriceRecorded(t, database, scheduled, scheduleDay(7, 5))
		assertFixturePriceRecorded(t, database, refreshed, scheduleDay(7, 5))
		if got := amazonFailures(t, database); got != 0 {
			t.Fatalf("consecutive failure count %d, want 0", got)
		}
	})

	t.Run("protected mode with items of two owners", func(t *testing.T) {
		server, _, _ := profileServer(t, "")
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		insertOwned(t, database, 1, "B0PROF0094", "amazon", scheduleDay(7, 0))
		insertOwned(t, database, 2, "B0PROF0095", "amazon", scheduleDay(7, 0))
		profile := filepath.Join(t.TempDir(), "profile")
		amazontest.KillBrowserOnProfile(t, profile)
		now := newClock(scheduleDay(7, 5))
		service := startHuman(t, database, dir, now, profileCollector(t, profile, server.URL), 94)
		service.collectDue(context.Background())
		assertFixturePriceRecorded(t, database, "B0PROF0094", scheduleDay(7, 5))
		assertFixturePriceRecorded(t, database, "B0PROF0095", scheduleDay(7, 5))
	})
}

func countLines(text string, match func(string) bool) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		if match(line) {
			count++
		}
	}
	return count
}
