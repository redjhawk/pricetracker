import { expect, test, type BrowserContext } from "@playwright/test";

const timestamp = "2026-10-04T08:00:00Z";
const baseItem = {
  id: "lbc-1", title: "LeBoncoin bicycle", platform: "leboncoin", listingId: "1234567890", asin: null,
  marketplace: "leboncoin.fr", url: "https://www.leboncoin.fr/ad/velos/1234567890", thumbnailUrl: null,
  status: "active", latestPrice: { amountCents: 15000, currency: "EUR", timestamp },
  lastThreeDetections: [{ amountCents: 15000, currency: "EUR", timestamp }], secondHandOffer: null,
  lastAttempt: null, nextCheckAt: null, addedAt: timestamp, purchaseGoal: "",
};
const content = {
  price: { rating: "fair", explanation: "Typical market price." },
  condition: { rating: null, explanation: "No photos to assess." },
  recommendation: { rating: "negotiate", explanation: "Offer a bit less." },
  fairPrice: { minCents: 12000, maxCents: 16000, suggestedOfferCents: 13000 },
  risks: [], missingInformation: ["Charger not shown"], sellerQuestions: ["Is the charger included?"],
  descriptionVsPhotos: { matches: null, explanation: "No photos.", mismatches: [] },
};
const review = (id: number, priceCents: number | null, createdAt: string) => ({
  id, status: "succeeded", priceCents, createdAt, completedAt: createdAt, review: content, errorMessage: null,
});
type State = { tokenConfigured: boolean; running: boolean; lastAttempt: unknown; latest: unknown; history: unknown[] };

async function mockApi(context: BrowserContext, aiReview: State | null, platform = "leboncoin") {
  const state = { aiReview, posts: 0 };
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== "http://127.0.0.1:4173") return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    if (url.pathname === "/api/v1/auth/session") return route.fulfill({ json: { mode: "open", user: null } });
    const method = route.request().method();
    if (method === "GET" && url.pathname === "/api/v1/items") return route.fulfill({ json: { items: [] } });
    if (method === "GET" && url.pathname === `/api/v1/items/${baseItem.id}`) {
      return route.fulfill({ json: { ...baseItem, platform, aiReview: state.aiReview } });
    }
    if (method === "POST" && url.pathname === `/api/v1/items/${baseItem.id}/ai-review`) {
      state.posts += 1;
      state.aiReview = { ...state.aiReview!, running: true };
      return route.fulfill({ status: 202, json: { requestedAt: timestamp, alreadyRunning: false } });
    }
    return route.abort();
  });
  return state;
}

test("without a token: message, refresh disabled", async ({ page, context }) => {
  await mockApi(context, { tokenConfigured: false, running: false, lastAttempt: null, latest: null, history: [] });
  await page.goto(`/items/${baseItem.id}`);
  const section = page.getByRole("region", { name: "AI review" });
  await expect(section.getByText("Configure a Claude token in Settings")).toBeVisible();
  await expect(section.getByText("No AI review yet.")).toBeVisible();
  await expect(section.getByRole("button", { name: "Refresh AI review" })).toBeDisabled();
});

test("refresh shows progress, polls, then shows the review and history", async ({ page, context }) => {
  const api = await mockApi(context, { tokenConfigured: true, running: false, lastAttempt: null, latest: null, history: [] });
  await page.goto(`/items/${baseItem.id}`);
  const section = page.getByRole("region", { name: "AI review" });
  await section.getByRole("button", { name: "Refresh AI review" }).click();
  await expect(section.getByText("AI review in progress…")).toBeVisible();
  await expect(section.getByRole("button", { name: "Refresh AI review" })).toBeDisabled();
  const latest = review(2, 15000, "2026-10-04T09:00:00Z");
  api.aiReview = { tokenConfigured: true, running: false, lastAttempt: latest, latest, history: [latest, review(1, 14000, timestamp)] };
  await expect(section.getByText("Price: Fair").first()).toBeVisible({ timeout: 10_000 });
  await expect(section.getByText("Condition: Not assessable").first()).toBeVisible();
  await expect(section.getByText("Recommendation: Negotiate").first()).toBeVisible();
  await expect(section.getByText("No scam or risk signs were found.").first()).toBeVisible();
  await expect(section.getByText(/older price/)).toHaveCount(0);
  await expect(section.getByRole("button", { name: /04 Oct 2026, 08:00 UTC — 140,00/ })).toBeVisible();
  expect(api.posts).toBe(1);
});

test("failed last attempt keeps the previous review; older price warning only with known prices", async ({ page, context }) => {
  const latest = review(1, 14000, timestamp);
  const failed = { id: 2, status: "failed", priceCents: null, createdAt: timestamp, completedAt: timestamp, review: null,
    errorMessage: "Claude could not be reached. Try again later." };
  await mockApi(context, { tokenConfigured: true, running: false, lastAttempt: failed, latest, history: [latest] });
  await page.goto(`/items/${baseItem.id}`);
  const section = page.getByRole("region", { name: "AI review" });
  await expect(section.getByText(/The last AI review failed on/)).toBeVisible();
  await expect(section.getByText("Claude could not be reached. Try again later.")).toBeVisible();
  await expect(section.getByText(/This review was made at an older price/)).toBeVisible();
});

test("unknown reviewed price shows no older price warning", async ({ page, context }) => {
  const latest = review(1, null, timestamp);
  await mockApi(context, { tokenConfigured: true, running: false, lastAttempt: latest, latest, history: [latest] });
  await page.goto(`/items/${baseItem.id}`);
  await expect(page.getByText(/at unknown/)).toBeVisible();
  await expect(page.getByText(/older price/)).toHaveCount(0);
});

test("Amazon items have no AI review section", async ({ page, context }) => {
  await mockApi(context, null, "amazon");
  await page.goto(`/items/${baseItem.id}`);
  await expect(page.getByRole("heading", { name: "LeBoncoin bicycle" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "AI review" })).toHaveCount(0);
});
