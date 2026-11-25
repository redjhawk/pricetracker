import { expect, test, type BrowserContext, type Page } from "@playwright/test";

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
type State = {
  tokenConfigured: boolean; running: boolean; lastAttempt: unknown; latest: unknown; history: unknown[];
  unavailableReason?: "stopped" | "no_current_price" | null;
};

async function mockApi(context: BrowserContext, aiReview: State | null) {
  const state = { aiReview, posts: 0 };
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== "http://127.0.0.1:4173") return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    if (url.pathname === "/api/v1/auth/session") return route.fulfill({ json: { mode: "open", user: null } });
    const method = route.request().method();
    if (method === "GET" && url.pathname === "/api/v1/items") return route.fulfill({ json: { items: [] } });
    if (method === "GET" && url.pathname === `/api/v1/items/${baseItem.id}`) {
      return route.fulfill({ json: { ...baseItem, aiReview: state.aiReview } });
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

test("TC-029 LeBoncoin details keep their review parts and goal field, without the Amazon messages", async ({ page, context }) => {
  const latest = review(1, 15000, timestamp);
  await mockApi(context, { tokenConfigured: true, running: false, lastAttempt: latest, latest, history: [latest], unavailableReason: null });
  await page.goto(`/items/${baseItem.id}`);
  await expect(page.getByRole("textbox", { name: "Purchase goal" })).toBeVisible();
  const section = page.getByRole("region", { name: "AI review" });
  await expect(section.getByText("Condition: Not assessable")).toBeVisible();
  await expect(section.getByText("Recommendation: Negotiate")).toBeVisible();
  await expect(section.getByText("Suggested offer", { exact: true })).toBeVisible();
  await expect(section.getByText("Risk signs", { exact: true })).toBeVisible();
  await expect(section.getByText(/Overall recommendation|Based on price|Amazon checks are stopped|No current price/)).toHaveCount(0);
  await expect(section.getByRole("button", { name: "Refresh AI review" })).toBeEnabled();
});

// Amazon item AI review and purchase goal (issue #305): TC-019 to TC-026 and TC-028.
const amazonTimestamp = "2026-11-02T08:00:00Z";
const amazonPrice = (amountCents: number) => ({ amountCents, currency: "EUR", timestamp: amazonTimestamp, oldPrice: false });
const usedOffer = { amountCents: 9500, currency: "EUR", timestamp: amazonTimestamp, condition: "very_good", conditionLabel: "Used - Very good" };
const amazonItem = {
  id: "amz-1", title: "Synthetic headphones", platform: "amazon", listingId: "B012345678", asin: "B012345678",
  marketplace: "amazon.fr", url: "https://www.amazon.fr/dp/B012345678", thumbnailUrl: null, status: "active",
  latestPrice: amazonPrice(12999), lastThreeDetections: [amazonPrice(12999)], priceHistory: [amazonPrice(12999)],
  secondHandOffer: { status: "available", latestDetection: usedOffer, lastThreeDetections: [usedOffer], priceHistory: [usedOffer], lastCheckedAt: amazonTimestamp },
  lastAttempt: { result: "success", timestamp: amazonTimestamp, message: null } as unknown,
  nextCheckAt: null, addedAt: amazonTimestamp, purchaseGoal: "gift for a 10-year-old", amazonStoppedAt: null as string | null,
};
const amazonContent = {
  price: { rating: "good_deal", explanation: "Below the usual price for these headphones." },
  fairPrice: { minCents: 11000, maxCents: 13000 },
  recommendation: { rating: "wait", explanation: "The price often drops before the holidays." },
};
const olderAmazonContent = {
  price: { rating: "fair", explanation: "A usual price." },
  fairPrice: { minCents: 12000, maxCents: 14000 },
  recommendation: { rating: "buy", explanation: "A good time to buy." },
};
const newestAmazonContent = {
  price: { rating: "overpriced", explanation: "Above the recent prices." },
  fairPrice: { minCents: 10000, maxCents: 12000 },
  recommendation: { rating: "avoid", explanation: "Better products cost less." },
};
const amazonReview = (id: number, priceCents: number, createdAt: string, content: unknown = amazonContent) => ({
  id, status: "succeeded", priceCents, createdAt, completedAt: createdAt, review: content, errorMessage: null,
});
const failedAmazonAttempt = (id: number, createdAt: string) => ({
  id, status: "failed", priceCents: null, createdAt, completedAt: createdAt, review: null, errorMessage: "Claude could not be reached. Try again later.",
});
const pendingAmazonAttempt = { id: 99, status: "pending", priceCents: null, createdAt: "2026-11-06T09:59:00Z", completedAt: null, review: null, errorMessage: null };
const amazonState = (overrides: Partial<State> = {}): State => ({
  tokenConfigured: true, running: false, lastAttempt: null, latest: null, history: [], unavailableReason: null, ...overrides,
});
// reviewed is the state of an item whose newest review is history[0].
const reviewed = (history: unknown[], overrides: Partial<State> = {}) => amazonState({ lastAttempt: history[0], latest: history[0], history, ...overrides });

const stoppedMessage = "Amazon checks are stopped: restart the item with Refresh price to request an AI review.";
const noPriceMessage = "No current price: an AI review needs a current Amazon price.";
const leboncoinOnlyParts = ["Condition", "Suggested offer", "Risk signs", "Missing accessories", "Questions to ask the seller", "Description vs photos"];

type AmazonDetails = Record<string, unknown> & { id: string; purchaseGoal: string; aiReview: State | null };
type AmazonApi = {
  addPosts: unknown[];
  reviewPosts: string[];
  goalPuts: string[];
  refreshPosts: string[];
  failGoalSave: boolean;
  onReviewPost?: (id: string) => void;
  onRefreshPost?: (id: string) => void;
  onDetailsGet?: (id: string) => void;
};

// mockAmazonApi serves the given item details (aiReview included) and records the requests.
async function mockAmazonApi(context: BrowserContext, details: Record<string, AmazonDetails>) {
  const api: AmazonApi = { addPosts: [], reviewPosts: [], goalPuts: [], refreshPosts: [], failGoalSave: false };
  const listed = (item: AmazonDetails) => {
    const { aiReview: _aiReview, ...rest } = item;
    return rest;
  };
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== "http://127.0.0.1:4173") return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    if (url.pathname === "/api/v1/auth/session") return route.fulfill({ json: { mode: "open", user: null } });
    const method = route.request().method();
    if (method === "GET" && url.pathname === "/api/v1/items") return route.fulfill({ json: { items: Object.values(details).map(listed) } });
    if (method === "POST" && url.pathname === "/api/v1/items") {
      const body = route.request().postDataJSON();
      api.addPosts.push(body);
      const added = Object.values(details)[0];
      added.purchaseGoal = String(body.purchaseGoal ?? "").trim();
      return route.fulfill({ status: 201, json: listed(added) });
    }
    const match = url.pathname.match(/^\/api\/v1\/items\/([^/]+)(\/[a-z-]+)?$/);
    const item = match ? details[decodeURIComponent(match[1])] : undefined;
    if (!match || !item) return route.abort();
    const action = match[2] ?? "";
    if (method === "GET" && action === "") {
      api.onDetailsGet?.(item.id);
      return route.fulfill({ json: item });
    }
    if (method === "POST" && action === "/ai-review") {
      api.reviewPosts.push(item.id);
      api.onReviewPost?.(item.id);
      return route.fulfill({ status: 202, json: { requestedAt: amazonTimestamp, alreadyRunning: false } });
    }
    if (method === "POST" && action === "/refresh") {
      api.refreshPosts.push(item.id);
      api.onRefreshPost?.(item.id);
      return route.fulfill({ status: 202, json: { requestedAt: amazonTimestamp, itemsQueued: 1 } });
    }
    if (method === "PUT" && action === "/purchase-goal") {
      const goal = route.request().postDataJSON().purchaseGoal as string;
      api.goalPuts.push(goal);
      if (api.failGoalSave) {
        return route.fulfill({ status: 500, json: { error: { code: "INTERNAL_ERROR", message: "The server could not complete the request." } } });
      }
      const changed = goal.trim() !== item.purchaseGoal;
      item.purchaseGoal = goal.trim();
      return route.fulfill({ json: { purchaseGoal: item.purchaseGoal, changed, reviewStarted: false } });
    }
    return route.abort();
  });
  return api;
}

const aiReviewSection = (page: Page) => page.getByRole("region", { name: "AI review" });
const refreshReviewButton = (page: Page) => aiReviewSection(page).getByRole("button", { name: "Refresh AI review" });

test("TC-019 add an Amazon item with a goal; its details page has the goal field and an AI review section", async ({ page, context }) => {
  const item: AmazonDetails = { ...amazonItem, purchaseGoal: "", aiReview: amazonState() };
  const api = await mockAmazonApi(context, { [item.id]: item });
  await page.goto("/");
  await page.getByRole("button", { name: "Add item", exact: true }).first().click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).not.toContainText("LeBoncoin only");
  await dialog.getByRole("textbox", { name: "Listing URL" }).fill(amazonItem.url);
  await dialog.getByRole("textbox", { name: "Purchase goal (optional)" }).fill("gift for a 10-year-old");
  await dialog.getByRole("button", { name: "Add item", exact: true }).click();
  await expect(dialog).toBeHidden();
  expect(api.addPosts).toEqual([{ url: amazonItem.url, purchaseGoal: "gift for a 10-year-old" }]);
  await page.goto(`/items/${item.id}`);
  await expect(page.getByRole("heading", { name: "Synthetic headphones" })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Purchase goal" })).toHaveValue("gift for a 10-year-old");
  const section = aiReviewSection(page);
  await expect(section.getByRole("heading", { name: "AI review" })).toBeVisible();
  await expect(section.getByText("No AI review yet.")).toBeVisible();
  await expect(refreshReviewButton(page)).toBeEnabled();
  expect(api.reviewPosts).toEqual([]);
});

test("TC-020 edit, clear and fail to save an Amazon goal without starting a review", async ({ page, context }) => {
  const earlier = amazonReview(1, 12999, "2026-11-04T08:00:00Z", olderAmazonContent);
  const item: AmazonDetails = { ...amazonItem, purchaseGoal: "", aiReview: reviewed([earlier]) };
  const api = await mockAmazonApi(context, { [item.id]: item });
  await page.goto(`/items/${item.id}`);
  const field = page.getByRole("textbox", { name: "Purchase goal" });
  const save = page.getByRole("button", { name: "Save goal" });
  const section = aiReviewSection(page);
  await expect(field).toHaveValue("");
  await expect(section.getByText("Price: Fair")).toBeVisible();
  await field.fill("Goal B");
  await page.keyboard.press("Tab");
  await expect(save).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("status").filter({ hasText: "Purchase goal saved." })).toBeVisible();
  await expect(page.getByText(/A new AI review was requested/)).toHaveCount(0);
  await expect(page.getByText("AI review in progress…")).toHaveCount(0);
  await expect(section.getByText("Price: Fair")).toBeVisible();
  await expect(field).toHaveValue("Goal B");
  await field.fill("");
  await save.click();
  await expect(save).toBeDisabled();
  await expect(field).toHaveValue("");
  await expect.poll(() => api.goalPuts).toEqual(["Goal B", ""]);

  api.failGoalSave = true;
  await field.fill("Goal C\nwith a second paragraph");
  await save.click();
  await expect(page.getByRole("status").filter({ hasText: "Could not save purchase goal" })).toBeVisible();
  await expect(field).toHaveValue("Goal C\nwith a second paragraph");
  await expect(section.getByText("Price: Fair")).toBeVisible();
  await expect(page.getByText("AI review in progress…")).toHaveCount(0);
  await expect.poll(() => api.goalPuts).toEqual(["Goal B", "", "Goal C\nwith a second paragraph"]);
  expect(api.reviewPosts).toEqual([]);
});

test("TC-021 Refresh AI review shows progress, then the labelled review and its history", async ({ page, context }) => {
  const older = amazonReview(1, 13999, "2026-11-04T08:00:00Z", olderAmazonContent);
  const first = amazonReview(2, 12999, "2026-11-05T09:00:00Z");
  const second = amazonReview(3, 12999, "2026-11-06T10:00:00Z", newestAmazonContent);
  const item: AmazonDetails = { ...amazonItem, aiReview: reviewed([older]) };
  const api = await mockAmazonApi(context, { [item.id]: item });
  const runs = [[first, older], [second, first, older]];
  let runningAnswers = 0;
  api.onReviewPost = () => {
    item.aiReview = { ...item.aiReview!, running: true, lastAttempt: pendingAmazonAttempt };
    runningAnswers = 2;
  };
  // After each request the details answer "running" twice, then the new review.
  api.onDetailsGet = () => {
    if (!item.aiReview?.running) return;
    if (runningAnswers > 0) {
      runningAnswers -= 1;
      return;
    }
    item.aiReview = reviewed(runs.shift()!);
  };
  await page.goto(`/items/${item.id}`);
  const section = aiReviewSection(page);
  const button = refreshReviewButton(page);
  const progress = section.locator('[aria-live="polite"]');

  await button.dblclick();
  await expect(button).toBeDisabled();
  await expect(progress).toContainText("AI review in progress…");
  await expect(section.getByText(/Reviewed on 05 Nov 2026, 09:00 UTC · Based on price 129,99\s€/)).toBeVisible({ timeout: 15_000 });
  expect(api.reviewPosts).toEqual([item.id]);
  await expect(button).toBeEnabled();
  await expect(progress).not.toContainText("AI review in progress…");
  // Carbon repeats a truncated tag label in a tooltip, hence first().
  await expect(section.getByText("Price: Good deal").first()).toBeVisible();
  await expect(section.getByText("Below the usual price for these headphones.")).toBeVisible();
  await expect(section.getByText("Fair price range").first()).toBeVisible();
  await expect(section.getByText(/110,00\s€ – 130,00\s€/)).toBeVisible();
  await expect(section.getByText("Overall recommendation: Wait for a lower price").first()).toBeVisible();
  await expect(section.getByText("The price often drops before the holidays.")).toBeVisible();
  for (const part of leboncoinOnlyParts) await expect(section.getByText(part)).toHaveCount(0);

  await button.focus();
  await page.keyboard.press("Enter");
  await expect(progress).toContainText("AI review in progress…");
  await expect(button).toBeDisabled();
  await expect(section.getByText(/Reviewed on 06 Nov 2026, 10:00 UTC · Based on price 129,99\s€/)).toBeVisible({ timeout: 15_000 });
  expect(api.reviewPosts).toEqual([item.id, item.id]);
  await expect(section.getByText("Price: Overpriced").first()).toBeVisible();
  await expect(section.getByText("Overall recommendation: Avoid").first()).toBeVisible();

  const history = section.getByRole("button", { name: /— Based on price/ });
  await expect(history).toHaveCount(2);
  await expect(history.nth(0)).toHaveText(/05 Nov 2026, 09:00 UTC — Based on price 129,99\s€ — Wait for a lower price/);
  await expect(history.nth(1)).toHaveText(/04 Nov 2026, 08:00 UTC — Based on price 139,99\s€ — Buy/);
  await history.nth(1).focus();
  await expect(history.nth(1)).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(history.nth(1)).toHaveAttribute("aria-expanded", "true");
  await expect(section.getByText("A good time to buy.")).toBeVisible();
  await expect(section).not.toContainText("gift for a 10-year-old");
  for (const part of leboncoinOnlyParts) await expect(section.getByText(part)).toHaveCount(0);
  expect(api.reviewPosts).toEqual([item.id, item.id]);
});

test("TC-022 a review made at another price is labelled with its own price", async ({ page, context }) => {
  const latest = amazonReview(1, 10000, "2026-11-04T08:00:00Z");
  const item: AmazonDetails = {
    ...amazonItem, latestPrice: amazonPrice(8000), lastThreeDetections: [amazonPrice(8000), amazonPrice(10000)],
    priceHistory: [amazonPrice(8000), amazonPrice(10000)], aiReview: reviewed([latest]),
  };
  await mockAmazonApi(context, { [item.id]: item });
  await page.goto(`/items/${item.id}`);
  await expect(page.locator(".summary-price").first()).toHaveText(/80,00\s€/);
  const section = aiReviewSection(page);
  await expect(section.getByText(/Based on price 100,00\s€/)).toBeVisible();
  await expect(section.getByText(/Based on price 80,00/)).toHaveCount(0);
  await expect(section.getByText(/older price/)).toHaveCount(0);
  await expect(section.getByText("AI review in progress…")).toHaveCount(0);
});

test("TC-023 a failed Amazon review shows a dated error and keeps the previous review", async ({ page, context }) => {
  const previous = amazonReview(1, 12999, "2026-11-04T08:00:00Z");
  const withReview: AmazonDetails = { ...amazonItem, aiReview: amazonState({ lastAttempt: failedAmazonAttempt(2, "2026-11-06T10:00:00Z"), latest: previous, history: [previous] }) };
  const withoutReview: AmazonDetails = { ...amazonItem, id: "amz-2", aiReview: amazonState({ lastAttempt: failedAmazonAttempt(3, "2026-11-06T11:00:00Z") }) };
  await mockAmazonApi(context, { [withReview.id]: withReview, [withoutReview.id]: withoutReview });

  await page.goto(`/items/${withReview.id}`);
  const section = aiReviewSection(page);
  const error = section.getByRole("status").filter({ hasText: "The last AI review failed on 06 Nov 2026, 10:00 UTC" });
  await expect(error).toBeVisible();
  await expect(error).toContainText("Claude could not be reached. Try again later.");
  await expect(section.getByText(/Reviewed on 04 Nov 2026, 08:00 UTC · Based on price 129,99\s€/)).toBeVisible();
  await expect(section.getByText("Price: Good deal")).toBeVisible();
  await expect(refreshReviewButton(page)).toBeEnabled();

  await page.goto(`/items/${withoutReview.id}`);
  await expect(section.getByRole("status").filter({ hasText: "The last AI review failed on 06 Nov 2026, 11:00 UTC" })).toBeVisible();
  await expect(section.getByText("No AI review yet.")).toBeVisible();
  await expect(refreshReviewButton(page)).toBeEnabled();
});

test("TC-024 without a Claude token: message, earlier review and disabled button", async ({ page, context }) => {
  const earlier = amazonReview(1, 12999, "2026-11-04T08:00:00Z");
  const item: AmazonDetails = { ...amazonItem, aiReview: reviewed([earlier], { tokenConfigured: false }) };
  await mockAmazonApi(context, { [item.id]: item });
  await page.goto(`/items/${item.id}`);
  await expect(page.locator(".summary-price").first()).toHaveText(/129,99\s€/);
  const section = aiReviewSection(page);
  await expect(section.getByText("Configure a Claude token in Settings")).toBeVisible();
  await expect(section.getByText("Price: Good deal")).toBeVisible();
  await expect(refreshReviewButton(page)).toBeDisabled();
});

for (const tokenConfigured of [true, false]) {
  test(`TC-025 stopped Amazon item ${tokenConfigured ? "with" : "without"} a token: review, message and no button; back after Refresh price`, async ({ page, context }) => {
    const earlier = amazonReview(1, 12999, "2026-11-04T08:00:00Z");
    const item: AmazonDetails = {
      ...amazonItem, status: "retrieval_error", amazonStoppedAt: "2026-11-03T07:26:00Z",
      lastAttempt: { result: "request_error", timestamp: "2026-11-03T07:26:00Z", message: "Amazon could not be reached for a price check." },
      aiReview: reviewed([earlier], { tokenConfigured, unavailableReason: "stopped" }),
    };
    const api = await mockAmazonApi(context, { [item.id]: item });
    api.onRefreshPost = () => Object.assign(item, {
      status: "active", amazonStoppedAt: null, lastAttempt: { result: "success", timestamp: "2026-11-06T08:00:00Z", message: null },
      aiReview: reviewed([earlier], { tokenConfigured }),
    });
    await page.goto(`/items/${item.id}`);
    const section = aiReviewSection(page);
    await expect(section.getByText("Price: Good deal")).toBeVisible();
    await expect(section.getByText(stoppedMessage)).toBeVisible();
    await expect(refreshReviewButton(page)).toHaveCount(0);
    await expect(section.getByText("Configure a Claude token in Settings")).toHaveCount(0);
    await expect(section.getByText(noPriceMessage)).toHaveCount(0);

    await page.getByRole("button", { name: "Refresh price", exact: true }).click();
    await expect(section.getByText(stoppedMessage)).toHaveCount(0);
    await expect(refreshReviewButton(page)).toBeVisible();
    if (tokenConfigured) {
      await expect(refreshReviewButton(page)).toBeEnabled();
    } else {
      await expect(refreshReviewButton(page)).toBeDisabled();
      await expect(section.getByText("Configure a Claude token in Settings")).toBeVisible();
    }
    await expect(section.getByText("AI review in progress…")).toHaveCount(0);
    await expect(section.getByText(/Reviewed on 04 Nov 2026, 08:00 UTC · Based on price 129,99\s€/)).toBeVisible();
    expect(api.refreshPosts).toEqual([item.id]);
    expect(api.reviewPosts).toEqual([]);
  });
}

test("TC-025 stopped Amazon item without a current price shows only the stopped message", async ({ page, context }) => {
  const earlier = amazonReview(1, 12999, "2026-11-04T08:00:00Z");
  const item: AmazonDetails = {
    ...amazonItem, status: "retrieval_error", amazonStoppedAt: "2026-11-03T07:26:00Z",
    lastAttempt: { result: "price_not_found", timestamp: "2026-11-03T07:26:00Z", message: "Amazon did not show a detectable euro price." },
    aiReview: reviewed([earlier], { unavailableReason: "stopped" }),
  };
  await mockAmazonApi(context, { [item.id]: item });
  await page.goto(`/items/${item.id}`);
  const section = aiReviewSection(page);
  await expect(section.getByText(stoppedMessage)).toBeVisible();
  await expect(section.getByText(noPriceMessage)).toHaveCount(0);
  await expect(refreshReviewButton(page)).toHaveCount(0);
});

test("TC-026 Amazon item without a current price: review, message and no button; back after a price", async ({ page, context }) => {
  const earlier = amazonReview(1, 12999, "2026-11-04T08:00:00Z");
  const item: AmazonDetails = {
    ...amazonItem, status: "retrieval_error",
    lastAttempt: { result: "price_not_found", timestamp: "2026-11-05T08:00:00Z", message: "Amazon did not show a detectable euro price." },
    aiReview: reviewed([earlier], { unavailableReason: "no_current_price" }),
  };
  const api = await mockAmazonApi(context, { [item.id]: item });
  api.onRefreshPost = () => Object.assign(item, {
    status: "active", latestPrice: amazonPrice(11999), lastThreeDetections: [amazonPrice(11999), amazonPrice(12999)],
    lastAttempt: { result: "success", timestamp: "2026-11-06T08:00:00Z", message: null }, aiReview: reviewed([earlier]),
  });
  await page.goto(`/items/${item.id}`);
  const section = aiReviewSection(page);
  await expect(section.getByText("Price: Good deal")).toBeVisible();
  await expect(section.getByText(noPriceMessage)).toBeVisible();
  await expect(refreshReviewButton(page)).toHaveCount(0);

  await page.getByRole("button", { name: "Refresh price", exact: true }).click();
  await expect(section.getByText(noPriceMessage)).toHaveCount(0);
  await expect(refreshReviewButton(page)).toBeEnabled();
  await expect(section.getByText(/Reviewed on 04 Nov 2026, 08:00 UTC · Based on price 129,99\s€/)).toBeVisible();
  await expect(section.getByText("AI review in progress…")).toHaveCount(0);
  expect(api.reviewPosts).toEqual([]);
});

test("TC-026 the no-current-price message also applies to a never-priced and an unavailable Amazon item", async ({ page, context }) => {
  const neverPriced: AmazonDetails = {
    ...amazonItem, status: "pending", latestPrice: null, lastThreeDetections: [], priceHistory: [], lastAttempt: null,
    aiReview: amazonState({ unavailableReason: "no_current_price" }),
  };
  const unavailable: AmazonDetails = {
    ...amazonItem, id: "amz-2", status: "unavailable",
    lastAttempt: { result: "unavailable", timestamp: "2026-11-05T08:00:00Z", message: "The listing appears to be unavailable." },
    aiReview: reviewed([amazonReview(1, 12999, "2026-11-04T08:00:00Z")], { unavailableReason: "no_current_price" }),
  };
  await mockAmazonApi(context, { [neverPriced.id]: neverPriced, [unavailable.id]: unavailable });
  for (const item of [neverPriced, unavailable]) {
    await page.goto(`/items/${item.id}`);
    const section = aiReviewSection(page);
    await expect(section.getByText(noPriceMessage)).toBeVisible();
    await expect(section.getByText(stoppedMessage)).toHaveCount(0);
    await expect(refreshReviewButton(page)).toHaveCount(0);
  }
});

test.describe("phone width", () => {
  test.use({ viewport: { width: 390, height: 844 } });

  test("TC-028 the Amazon AI review section is usable at phone width", async ({ page, context }) => {
    const longContent = {
      price: { rating: "good_deal", explanation: "B".repeat(180) },
      fairPrice: { minCents: 11000, maxCents: 13000 },
      recommendation: { rating: "wait", explanation: "The price of these headphones often drops before the holidays. ".repeat(6) },
    };
    const history = [
      amazonReview(4, 12999, "2026-11-07T08:00:00Z", longContent),
      amazonReview(3, 13499, "2026-11-06T08:00:00Z", longContent),
      amazonReview(2, 13999, "2026-11-05T08:00:00Z", olderAmazonContent),
      amazonReview(1, 14999, "2026-11-04T08:00:00Z", newestAmazonContent),
    ];
    const longGoal = "A gift for my nephew who loves music and travels a lot. ".repeat(8).trim();
    const item: AmazonDetails = { ...amazonItem, purchaseGoal: longGoal, aiReview: reviewed(history) };
    await mockAmazonApi(context, { [item.id]: item });
    await page.goto(`/items/${item.id}`);
    const section = aiReviewSection(page);
    const goal = page.getByRole("textbox", { name: "Purchase goal" });
    await expect(goal).toBeVisible();
    await expect(goal).toHaveValue(longGoal);
    await expect(section.getByRole("heading", { name: "AI review" })).toBeVisible();
    await expect(section.getByText("Price: Good deal").first()).toBeVisible();
    await expect(section.getByText("Overall recommendation: Wait for a lower price").first()).toBeVisible();
    await expect(section.getByText(/Based on price 129,99\s€/).first()).toBeVisible();
    const entries = section.getByRole("button", { name: /— Based on price/ });
    await expect(entries).toHaveCount(3);
    await page.waitForTimeout(100);
    const layout = await page.evaluate(() => ({ scrollWidth: document.documentElement.scrollWidth, width: document.documentElement.clientWidth }));
    expect(layout.scrollWidth, "the page scrolls horizontally").toBeLessThanOrEqual(layout.width);
    await goal.focus();
    await expect(goal).toBeFocused();
    for (let index = 0; index < 3; index += 1) {
      const entry = entries.nth(index);
      await entry.focus();
      await expect(entry).toBeFocused();
      await expect(entry).toBeInViewport();
    }
    await refreshReviewButton(page).focus();
    await expect(refreshReviewButton(page)).toBeFocused();
  });
});
