import { expect, test, type BrowserContext } from "@playwright/test";

const timestamp = "2026-10-07T12:00:00Z";
const search = {
  id: "s1",
  url: "https://www.amazon.fr/joursprime/?x=1",
  label: "amazon.fr/joursprime/",
  addedAt: timestamp,
  capturedAt: timestamp,
  itemCount: 1,
  state: "stopped",
  waitingUntil: null,
  lastError: null,
};
const price = { amountCents: 9000, currency: "EUR", timestamp, oldPrice: false };
const item = {
  id: "i1", title: "Prime headphones", platform: "amazon", listingId: "B000000001", asin: "B000000001",
  marketplace: "amazon.fr", url: "https://www.amazon.fr/dp/B000000001", thumbnailUrl: null, status: "unavailable",
  latestPrice: price, lastThreeDetections: [price], secondHandOffer: null, nextCheckAt: null, addedAt: timestamp,
  purchaseGoal: "", tracked: false, position: 1,
  aiReviewSummary: { status: "available", priceRating: "good_deal", priceCents: 10000 },
};

async function mockApi(context: BrowserContext) {
  const state = { searches: [] as typeof search[], tracked: false, requests: [] as string[] };
  const amazonRequests = { stopped: true, stoppedAt: timestamp, consecutiveFailures: 5 };
  await context.route("**/*", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (url.origin !== "http://127.0.0.1:4173") return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    const key = `${request.method()} ${url.pathname}`;
    state.requests.push(key);
    if (url.pathname === "/api/v1/auth/session") return route.fulfill({ json: { mode: "open", user: null } });
    if (key === "GET /api/v1/items") return route.fulfill({ json: { items: [] } });
    if (key === "GET /api/v1/amazon/requests") return route.fulfill({ json: { amazonRequests } });
    if (key === "GET /api/v1/amazon-searches") return route.fulfill({ json: { searches: state.searches, amazonRequests } });
    if (key === "POST /api/v1/amazon-searches") {
      if (request.postDataJSON().url !== search.url) {
        return route.fulfill({ status: 422, json: { error: { code: "UNSUPPORTED_SEARCH", message: "Only searches on the supported Amazon euro marketplaces can be added." } } });
      }
      state.searches = [search];
      return route.fulfill({ status: 201, json: search });
    }
    if (key === "GET /api/v1/amazon-searches/s1") {
      return route.fulfill({ json: { search, amazonRequests, items: [{ ...item, tracked: state.tracked }] } });
    }
    if (key === "POST /api/v1/amazon-searches/s1/items/i1/track") {
      state.tracked = true;
      return route.fulfill({ json: { ...item, tracked: true } });
    }
    return route.fulfill({ status: 500, json: { error: { code: "UNEXPECTED", message: key } } });
  });
  return state;
}

test("add a search, see the stop line, open it and move an item to tracked", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  await expect(page.getByText("Amazon requests are stopped since")).toBeVisible();
  await page.getByRole("tab", { name: "Amazon searches" }).click();
  await expect(page.getByText("No Amazon search yet. Add a results URL from Amazon.")).toBeVisible();

  await page.getByLabel("Amazon search URL").fill("https://example.com/");
  await page.getByRole("button", { name: "Add search" }).click();
  await expect(page.getByText("Only searches on the supported Amazon euro marketplaces can be added.")).toBeVisible();

  await page.getByLabel("Amazon search URL").fill(search.url);
  await page.getByRole("button", { name: "Add search" }).click();
  await expect(page.getByRole("button", { name: search.label })).toBeVisible();
  await expect(page.getByRole("button", { name: "Refresh", exact: true })).toBeVisible();
  await expect(page.getByText("Amazon requests were stopped after repeated failures on")).toBeVisible();

  await page.getByRole("button", { name: search.label }).click();
  await expect(page).toHaveURL(/\/searches\/s1$/);
  await expect(page.getByRole("cell", { name: "Unavailable" })).toBeVisible();
  await expect(page.getByText("Good deal · Price changed")).toBeVisible();
  await page.getByRole("button", { name: "Move to tracked Amazon items" }).click();
  await expect(page.getByText("Tracked", { exact: true })).toBeVisible();
  expect(api.requests).toContain("POST /api/v1/amazon-searches/s1/items/i1/track");
});
