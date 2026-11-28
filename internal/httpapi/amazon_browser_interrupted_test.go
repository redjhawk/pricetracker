package httpapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/service"
)

// An Amazon check stopped with the server while its answer is offered as a download writes no file
// (issue #233): a real browser collector with its persistent profile in the data directory.

// A server stop during a check answered with a download writes nothing in the data directory; the
// next Refresh price of each owner records the normal page.
// TC-005
func TestAmazonBrowserInterrupted_TC005_ServerStopDuringRefreshWritesNoFile(t *testing.T) {
	ids := []string{"B0INTR0051", "B0INTR0052"}
	owners := []int64{1, 2}
	api := openNavigationAPI(t, ids, owners...)
	home := amazontest.IsolatedHome(t)
	before := dataFiles(t, api.dir)

	api.fixture.answer.Store(answerAttachmentHeld)
	output := amazontest.CaptureOutput(t)
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
	if lines := linesWith(output(), "Amazon browser cache could not be cleared"); len(lines) > 0 {
		t.Fatalf("cache clear failure logged:\n%s", strings.Join(lines, "\n"))
	}
	assertNoDownloadWritten(t, api, home)
	assertNoNewDataFile(t, api, before, "after the server stop")

	api.fixture.answer.Store(answerPage)
	for index, owner := range owners {
		id, at := ids[index], humanTime(14, 10+index)
		api.clock.Set(at)
		refreshOutput := amazontest.CaptureOutput(t)
		principal := service.WithPrincipal(context.Background(), service.Principal{UserID: owner, Role: "user"})
		if _, err := api.items.RefreshItem(principal, id); err != nil {
			t.Fatalf("Refresh price of %s by owner %d: %v", id, owner, err)
		}
		waitStoredCheck(t, api.database, id, at)
		// FR-AMZ-HCHK-016: the Refresh price logs no cache clear failure, in either wording (issues #240 and #358).
		if lines := linesWith(refreshOutput(), "Amazon browser cache could not be cleared"); len(lines) > 0 {
			t.Fatalf("cache clear failure logged during Refresh price of %s:\n%s", id, strings.Join(lines, "\n"))
		}
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
