package service

import (
	"context"
	"reflect"
	"testing"
	"time"

	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// insertGoal stores an item of owner with goal, bypassing the service rules
// (so an Amazon row may carry a goal).
func insertGoal(t *testing.T, database *store.Store, id, platform, goal string, owner int64) {
	t.Helper()
	url := "https://www.leboncoin.fr/ad/goal/" + id
	listing := model.Listing{ID: id, Platform: platform, ListingID: id, Marketplace: "leboncoin.fr", URL: url, PurchaseGoal: goal, OwnerID: owner}
	if platform == "amazon" {
		url = "https://www.amazon.fr/dp/" + id
		listing.ListingID, listing.ASIN, listing.Marketplace, listing.URL = id, id, "amazon.fr", url
	}
	if err := database.Insert(context.Background(), listing, url, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func assertGoals(t *testing.T, service *Service, ctx context.Context, want ...string) {
	t.Helper()
	goals, err := service.PurchaseGoals(ctx)
	if err != nil {
		t.Fatalf("purchase goals: %v", err)
	}
	if want == nil {
		want = []string{}
	}
	if goals == nil {
		goals = []string{}
	}
	if !reflect.DeepEqual(goals, want) {
		t.Fatalf("purchase goals %q, want %q", goals, want)
	}
}

func userContext(t *testing.T, service *Service, name string) (context.Context, int64) {
	t.Helper()
	user, err := service.CreateUser(context.Background(), name, "long enough pw")
	if err != nil {
		t.Fatal(err)
	}
	return WithPrincipal(context.Background(), Principal{UserID: user.ID, Role: "user"}), user.ID
}

func itemGoal(t *testing.T, service *Service, ctx context.Context, id string) string {
	t.Helper()
	item, err := service.Get(ctx, id)
	if err != nil {
		t.Fatalf("get %s: %v", id, err)
	}
	return item.PurchaseGoal
}

// TC-001: distinct exact texts, alphabetical ignoring case, own items of both platforms.
func TestPurchaseGoals_TC001_DistinctAlphabeticalOwnItems(t *testing.T) {
	service, database := newSessionService(t)
	alice, aliceID := userContext(t, service, "alice")
	bob, bobID := userContext(t, service, "bob")
	carol, carolID := userContext(t, service, "carol")
	dave, _ := userContext(t, service, "dave")
	for i, goal := range []string{"photo editing", "Gaming", "Gaming", "gaming", "browsing", "install a light Linux distro for my parents", ""} {
		insertGoal(t, database, "alice"+string(rune('a'+i)), "leboncoin", goal, aliceID)
	}
	insertGoal(t, database, "B000000001", "amazon", "Amazon goal", aliceID)
	insertGoal(t, database, "bob1", "leboncoin", "Office", bobID)

	want := []string{"Amazon goal", "browsing", "Gaming", "gaming", "install a light Linux distro for my parents", "photo editing"}
	for range 3 {
		assertGoals(t, service, alice, want...)
	}
	assertGoals(t, service, bob, "Office")
	assertGoals(t, service, dave)

	// Corner cases: spacing and line breaks kept, empty and whitespace-only goals dropped, Amazon goals kept.
	for i, goal := range []string{"Gaming on Linux", "Gaming  on Linux", "line one\nline two", "line one\nline two", "", "   "} {
		insertGoal(t, database, "carol"+string(rune('a'+i)), "leboncoin", goal, carolID)
	}
	insertGoal(t, database, "B000000002", "amazon", "Brand new text", carolID)
	assertGoals(t, service, carol, "Brand new text", "Gaming  on Linux", "Gaming on Linux", "line one\nline two")
}

// TC-001: open mode lists the goals of all items.
func TestPurchaseGoals_TC001_OpenModeListsAllGoals(t *testing.T) {
	service, database := newSessionService(t)
	insertGoal(t, database, "a", "leboncoin", "photo editing", 0)
	insertGoal(t, database, "b", "leboncoin", "Gaming", 0)
	insertGoal(t, database, "c", "leboncoin", "", 0)
	insertGoal(t, database, "B000000001", "amazon", "Amazon goal", 0)
	assertGoals(t, service, context.Background(), "Amazon goal", "Gaming", "photo editing")
}

// newGoalService has open-mode LeBoncoin items A ("Gaming") and B ("Office").
func newGoalService(t *testing.T) (*Service, *store.Store, context.Context) {
	t.Helper()
	service, database, _, _ := newReviewService(t, false)
	insertGoal(t, database, "A", "leboncoin", "Gaming", 0)
	insertGoal(t, database, "B", "leboncoin", "Office", 0)
	return service, database, context.Background()
}

// TC-003: the list follows item additions and removals.
func TestPurchaseGoals_TC003_ListFollowsAdditionsAndRemovals(t *testing.T) {
	service, _, ctx := newGoalService(t)
	assertGoals(t, service, ctx, "Gaming", "Office")
	if _, err := service.Add(ctx, "https://www.leboncoin.fr/ad/test/3031", "photo editing"); err != nil {
		t.Fatal(err)
	}
	assertGoals(t, service, ctx, "Gaming", "Office", "photo editing")
	if deleted, err := service.Remove(ctx, "A"); err != nil || !deleted {
		t.Fatalf("remove A: %v %v", deleted, err)
	}
	assertGoals(t, service, ctx, "Office", "photo editing")
}

// TC-003: a changed or cleared goal leaves the list.
func TestPurchaseGoals_TC003_GoalChangedOrCleared(t *testing.T) {
	t.Run("changed", func(t *testing.T) {
		service, _, ctx := newGoalService(t)
		if _, _, _, err := service.SetPurchaseGoal(ctx, "A", "Office"); err != nil {
			t.Fatal(err)
		}
		assertGoals(t, service, ctx, "Office")
	})
	for _, cleared := range []string{"", "   "} {
		t.Run("cleared "+`"`+cleared+`"`, func(t *testing.T) {
			service, _, ctx := newGoalService(t)
			if _, _, _, err := service.SetPurchaseGoal(ctx, "A", cleared); err != nil {
				t.Fatal(err)
			}
			assertGoals(t, service, ctx, "Office")
		})
	}
}

// TC-003: a goal still used stays; removing the last goal empties the list.
func TestPurchaseGoals_TC003_GoalStillUsedAndLastGoalGone(t *testing.T) {
	service, database, ctx := newGoalService(t)
	insertGoal(t, database, "C", "leboncoin", "Gaming", 0)
	if _, err := service.Remove(ctx, "A"); err != nil {
		t.Fatal(err)
	}
	assertGoals(t, service, ctx, "Gaming", "Office")
	for _, id := range []string{"B", "C"} {
		if _, err := service.Remove(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	assertGoals(t, service, ctx)
}

// TC-003: an identical text is listed once, a case variant separately.
func TestPurchaseGoals_TC003_IdenticalAndCaseVariantAdditions(t *testing.T) {
	service, _, ctx := newGoalService(t)
	if _, err := service.Add(ctx, "https://www.leboncoin.fr/ad/test/3032", "Gaming"); err != nil {
		t.Fatal(err)
	}
	assertGoals(t, service, ctx, "Gaming", "Office")
	if _, err := service.Add(ctx, "https://www.leboncoin.fr/ad/test/3033", "gaming"); err != nil {
		t.Fatal(err)
	}
	assertGoals(t, service, ctx, "Gaming", "gaming", "Office")
}

// TC-003: an Amazon item stores its goal, which joins the list (issue #305).
func TestPurchaseGoals_TC003_AmazonAdditionJoinsList(t *testing.T) {
	service, _, ctx := newGoalService(t)
	for url, goal := range map[string]string{"https://www.amazon.fr/dp/B012345678": "Gaming", "https://www.amazon.fr/dp/B087654321": "Brand new text"} {
		item, err := service.Add(ctx, url, goal)
		if err != nil {
			t.Fatal(err)
		}
		if stored := itemGoal(t, service, ctx, item.ID); stored != goal {
			t.Fatalf("Amazon item stored goal %q, want %q", stored, goal)
		}
	}
	assertGoals(t, service, ctx, "Brand new text", "Gaming", "Office")
}

// TC-003: a refused removal leaves the list unchanged.
func TestPurchaseGoals_TC003_RefusedRemovalLeavesListUnchanged(t *testing.T) {
	service, database, _, _ := newReviewService(t, false)
	alice, aliceID := userContext(t, service, "alice")
	bob, _ := userContext(t, service, "bob")
	insertGoal(t, database, "A", "leboncoin", "Gaming", aliceID)
	insertGoal(t, database, "B", "leboncoin", "Office", aliceID)
	if deleted, err := service.Remove(bob, "A"); err != nil || deleted {
		t.Fatalf("bob removed alice's item: %v %v", deleted, err)
	}
	assertGoals(t, service, alice, "Gaming", "Office")
	assertGoals(t, service, bob)
}

// newCopiedGoalService has LeBoncoin items A and B with goal "Gaming", C with "Office", and a Claude token.
func newCopiedGoalService(t *testing.T) (*Service, *fakeReviewer, context.Context) {
	t.Helper()
	service, database, reviewer, _ := newReviewService(t, true)
	insertGoal(t, database, "A", "leboncoin", "Gaming", 0)
	insertGoal(t, database, "B", "leboncoin", "Gaming", 0)
	insertGoal(t, database, "C", "leboncoin", "Office", 0)
	return service, reviewer, context.Background()
}

// assertReviewedOnlyA checks that one review ran, for item A with goal, and none for B or C.
func assertReviewedOnlyA(t *testing.T, service *Service, reviewer *fakeReviewer, ctx context.Context, goal string) {
	t.Helper()
	waitReviews(t, service, "A", false)
	if reviewer.calls() != 1 || reviewer.inputs[0].URL != "https://www.leboncoin.fr/ad/goal/A" || reviewer.inputs[0].PurchaseGoal != goal {
		t.Fatalf("review inputs %+v", reviewer.inputs)
	}
	for _, id := range []string{"B", "C"} {
		item, err := service.Get(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		state, err := service.AIReview(ctx, item)
		if err != nil {
			t.Fatal(err)
		}
		if state.Running || state.Latest != nil || len(state.History) != 0 {
			t.Fatalf("item %s reviewed: %+v", id, state)
		}
	}
}

// TC-004: a copied goal is independent per item; only the changed item is reviewed.
func TestPurchaseGoals_TC004_CopiedGoalIsIndependentPerItem(t *testing.T) {
	service, reviewer, ctx := newCopiedGoalService(t)
	goal, changed, started, err := service.SetPurchaseGoal(ctx, "A", "Gaming on Linux")
	if err != nil || goal != "Gaming on Linux" || !changed || !started {
		t.Fatalf("set: %q %v %v %v", goal, changed, started, err)
	}
	assertReviewedOnlyA(t, service, reviewer, ctx, "Gaming on Linux")
	if got := itemGoal(t, service, ctx, "A"); got != "Gaming on Linux" {
		t.Fatalf("A goal %q", got)
	}
	if got := itemGoal(t, service, ctx, "B"); got != "Gaming" {
		t.Fatalf("B goal %q", got)
	}
	assertGoals(t, service, ctx, "Gaming", "Gaming on Linux", "Office")
}

// TC-004: clearing A's goal keeps B's goal and the list entry.
func TestPurchaseGoals_TC004_ClearedCopyKeepsOtherItemGoal(t *testing.T) {
	service, reviewer, ctx := newCopiedGoalService(t)
	if _, changed, _, err := service.SetPurchaseGoal(ctx, "A", ""); err != nil || !changed {
		t.Fatalf("clear: %v %v", changed, err)
	}
	assertReviewedOnlyA(t, service, reviewer, ctx, "")
	if got := itemGoal(t, service, ctx, "B"); got != "Gaming" {
		t.Fatalf("B goal %q", got)
	}
	assertGoals(t, service, ctx, "Gaming", "Office")
}

// TC-004: taking another item's goal neither changes nor reviews that item.
func TestPurchaseGoals_TC004_ExistingGoalOfAnotherItem(t *testing.T) {
	service, reviewer, ctx := newCopiedGoalService(t)
	if _, changed, _, err := service.SetPurchaseGoal(ctx, "A", "Office"); err != nil || !changed {
		t.Fatalf("set: %v %v", changed, err)
	}
	assertReviewedOnlyA(t, service, reviewer, ctx, "Office")
	if got := itemGoal(t, service, ctx, "C"); got != "Office" {
		t.Fatalf("C goal %q", got)
	}
	assertGoals(t, service, ctx, "Gaming", "Office")
}

// TC-004: setting the current text again changes nothing and starts no review.
func TestPurchaseGoals_TC004_SameTextChangesNothing(t *testing.T) {
	service, reviewer, ctx := newCopiedGoalService(t)
	goal, changed, started, err := service.SetPurchaseGoal(ctx, "A", "  Gaming ")
	if err != nil || goal != "Gaming" || changed || started {
		t.Fatalf("same text: %q %v %v %v", goal, changed, started, err)
	}
	waitReviews(t, service, "A", false)
	if reviewer.calls() != 0 {
		t.Fatalf("review inputs %+v", reviewer.inputs)
	}
	assertGoals(t, service, ctx, "Gaming", "Office")
}
