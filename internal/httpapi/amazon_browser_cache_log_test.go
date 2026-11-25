package httpapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/service"
)

// The neutral cache-clear log line (issue #240) through the application: a real browser collector with
// its persistent profile in the data directory confirms the clear, so no cache line is logged.

// assertNoCacheLine fails t when a line of output says the browser cache could not be cleared, in the
// neutral or the old wording.
func assertNoCacheLine(t *testing.T, output, step string) {
	t.Helper()
	for _, text := range []string{"Amazon browser cache could not be cleared", "after a page not opened"} {
		if lines := linesWith(output, text); len(lines) > 0 {
			t.Fatalf("%s: cache clear failure logged:\n%s", step, strings.Join(lines, "\n"))
		}
	}
}

// A check stopped with the server while the download is offered, then Refresh price by each owner on the
// normal page: the clearing succeeds, no cache line is logged and nothing else changes.
// TC-004
func TestAmazonBrowserCacheLog_TC004_SuccessfulClearThroughTheApplicationLogsNothing(t *testing.T) {
	ids := []string{"B0CLOG0041", "B0CLOG0042"}
	owners := []int64{1, 2}
	api := openNavigationAPI(t, ids, owners...)
	home := amazontest.IsolatedHome(t)
	before := dataFiles(t, api.dir)

	api.fixture.answer.Store(answerAttachmentHeld)
	output := captureCheckOutput(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopped := make(chan struct{})
	go func() {
		api.items.Collect(ctx, ids[0])
		close(stopped)
	}()
	select {
	case <-api.fixture.flushed:
	case <-time.After(60 * time.Second):
		t.Fatal("the product page was not answered with the download within 60 s")
	}
	select {
	case <-stopped:
		t.Fatal("the check ended before the server stop: the interruption point was missed")
	default:
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(20 * time.Second):
		t.Fatal("the stopped check did not return within 20 s")
	}
	assertNoCacheLine(t, output(), "after the server stop")
	assertNoDownloadWritten(t, api, home)
	assertNoNewDataFile(t, api, before, "after the server stop")

	api.fixture.answer.Store(answerPage)
	for index, owner := range owners {
		id, at := ids[index], humanTime(14, 10+index)
		api.clock.Set(at)
		output := captureCheckOutput(t)
		principal := service.WithPrincipal(context.Background(), service.Principal{UserID: owner, Role: "user"})
		if _, err := api.items.RefreshItem(principal, id); err != nil {
			t.Fatalf("Refresh price of %s by owner %d: %v", id, owner, err)
		}
		waitStoredCheck(t, api.database, id, at)
		assertNoCacheLine(t, output(), "after Refresh price of "+id)
		items, err := api.database.List(context.Background(), owner)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ID != id {
			t.Fatalf("items of owner %d: %+v, want only %s", owner, items, id)
		}
		if item := items[0]; item.LatestPrice == nil || item.LatestPrice.AmountCents != 12999 {
			t.Fatalf("owner %d, item %s: latest price %+v, want 129.99", owner, id, item.LatestPrice)
		}
		assertNoDownloadWritten(t, api, home)
		assertNoNewDataFile(t, api, before, "after Refresh price of "+id)
	}
}
