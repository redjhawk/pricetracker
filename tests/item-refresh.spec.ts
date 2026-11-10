import { expect, test, type BrowserContext, type Page } from "@playwright/test";

const timestamp = "2026-10-03T08:00:00Z";
const price = { amountCents: 1299, currency: "EUR", timestamp };
const newPrice = { amountCents: 999, currency: "EUR", timestamp: "2026-10-03T09:00:00Z" };
const amazon = {
  id: "amazon-one", title: "Amazon headphones", platform: "amazon",
  listingId: "B09XS7JWHH", asin: "B09XS7JWHH", marketplace: "amazon.de",
  url: "https://www.amazon.de/dp/B09XS7JWHH", thumbnailUrl: null,
  status: "active", latestPrice: price, lastThreeDetections: [price],
  secondHandOffer: null,
  lastAttempt: { result: "success", timestamp, message: null },
  nextCheckAt: null, addedAt: timestamp,
};
const amazonTwo = { ...amazon, id: "amazon-two", title: "Amazon speaker", listingId: "B000000002", asin: "B000000002" };
const leboncoin = {
  ...amazon, id: "leboncoin-one", title: "LeBoncoin bicycle", platform: "leboncoin",
  listingId: "1234567890", asin: null, marketplace: "leboncoin.fr",
  url: "https://www.leboncoin.fr/ad/velos/1234567890",
};
type FixtureItem = Omit<typeof amazon, "asin" | "status" | "lastThreeDetections" | "latestPrice"> & {
  asin: string | null;
  status: string;
  latestPrice: typeof price;
  lastThreeDetections: (typeof price)[];
};

async function mockApi(context: BrowserContext, initialItems: FixtureItem[] = [amazon, amazonTwo, leboncoin]) {
  const state = {
    items: structuredClone(initialItems),
    itemRefreshStatus: 202,
    refreshGate: Promise.resolve(),
    requests: [] as string[],
    unexpected: [] as string[],
  };
  await context.route("**/*", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (url.origin !== "http://127.0.0.1:4173") {
      await route.abort();
      return;
    }
    if (!url.pathname.startsWith("/api/")) {
      await route.continue();
      return;
    }
    if (url.pathname === "/api/v1/auth/session") return route.fulfill({ json: { mode: "open", user: null } });
    const method = request.method();
    const path = url.pathname;
    state.requests.push(`${method} ${path}`);
    const refreshMatch = path.match(/^\/api\/v1\/items\/([^/]+)\/refresh$/);
    if (method === "GET" && path === "/api/v1/items") {
      await route.fulfill({ json: { items: state.items } });
    } else if (method === "POST" && path === "/api/v1/items/refresh") {
      state.items = state.items.map((item) => ({ ...item, status: "pending" }));
      await route.fulfill({ status: 202, json: { requestedAt: timestamp, itemsQueued: state.items.length } });
    } else if (method === "POST" && refreshMatch) {
      await state.refreshGate;
      if (state.itemRefreshStatus === 404) {
        await route.fulfill({ status: 404, json: { error: { code: "ITEM_NOT_FOUND", message: "Item not found." } } });
      } else if (state.itemRefreshStatus === 500) {
        await route.fulfill({ status: 500, json: { error: { code: "TEST_FAILURE", message: "Fixture refresh failed." } } });
      } else {
        state.items = state.items.map((item) => item.id === refreshMatch[1] ? { ...item, status: "pending" } : item);
        await route.fulfill({ status: 202, json: { requestedAt: timestamp, itemsQueued: 1 } });
      }
    } else {
      state.unexpected.push(`${method} ${path}`);
      await route.abort();
    }
  });
  return state;
}

function row(page: Page, title: string) {
  return page.getByRole("row").filter({ has: page.getByRole("button", { name: title, exact: true }) });
}

async function openMenu(page: Page, title: string) {
  await row(page, title).getByRole("button", { name: "Options", exact: true }).click();
}

test("menu lists Refresh price before Delete on both tabs", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  await openMenu(page, "Amazon headphones");
  await expect(page.getByRole("menuitem")).toHaveText(["View details", "Open on Amazon", "Refresh price", "Delete"]);
  await page.keyboard.press("Escape");
  await page.getByRole("tab", { name: "LeBoncoin" }).click();
  await openMenu(page, "LeBoncoin bicycle");
  await expect(page.getByRole("menuitem")).toHaveText(["View details", "Open on LeBoncoin", "Refresh price", "Delete"]);
  expect(api.unexpected).toEqual([]);
});

test("refreshes only the selected item, shows pending then updated row", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  await openMenu(page, "Amazon headphones");
  await page.getByRole("menuitem", { name: "Refresh price" }).click();
  await expect(row(page, "Amazon headphones").getByText("Pending", { exact: true })).toBeVisible();
  await expect(row(page, "Amazon speaker").getByText("Pending", { exact: true })).toHaveCount(0);
  expect(api.requests.filter((entry) => entry.startsWith("POST"))).toEqual(["POST /api/v1/items/amazon-one/refresh"]);
  await expect(page.getByText(/Refreshing prices for/)).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Refresh all prices" })).toBeEnabled();

  await openMenu(page, "Amazon headphones");
  await expect(page.getByRole("menuitem", { name: "Refresh price" })).toBeDisabled();
  await page.keyboard.press("Escape");
  await openMenu(page, "Amazon speaker");
  await expect(page.getByRole("menuitem", { name: "Refresh price" })).toBeEnabled();
  await page.keyboard.press("Escape");

  api.items = api.items.map((item) => item.id === "amazon-one"
    ? { ...item, status: "active", latestPrice: newPrice, lastThreeDetections: [newPrice, price] }
    : item);
  await expect(row(page, "Amazon headphones")).toContainText("9,99", { timeout: 10000 });
  await expect(row(page, "Amazon headphones").getByText("Active", { exact: true })).toBeVisible();
  await expect(page.getByRole("tab", { name: "Amazon", exact: true })).toHaveAttribute("aria-selected", "true");
  expect(api.unexpected).toEqual([]);
});

test("failed collection shows failure status and keeps previous price", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  await openMenu(page, "Amazon headphones");
  await page.getByRole("menuitem", { name: "Refresh price" }).click();
  await expect(row(page, "Amazon headphones").getByText("Pending", { exact: true })).toBeVisible();
  api.items = api.items.map((item) => item.id === "amazon-one" ? { ...item, status: "retrieval_error" } : item);
  await expect(row(page, "Amazon headphones").getByText("Retrieval error", { exact: true })).toBeVisible({ timeout: 10000 });
  await expect(row(page, "Amazon headphones")).toContainText("12,99");
});

test("request errors name the item, keep prices, and clear on later success", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  for (const status of [404, 500]) {
    api.itemRefreshStatus = status;
    await openMenu(page, "Amazon headphones");
    await page.getByRole("menuitem", { name: "Refresh price" }).click();
    await expect(page.getByText("Could not refresh price", { exact: true })).toBeVisible();
    await expect(page.getByText(/^Amazon headphones: /)).toBeVisible();
    await expect(row(page, "Amazon headphones")).toContainText("12,99");
    await expect(row(page, "Amazon speaker")).toContainText("12,99");
  }
  api.itemRefreshStatus = 202;
  await openMenu(page, "Amazon speaker");
  await page.getByRole("menuitem", { name: "Refresh price" }).click();
  await expect(page.getByText("Could not refresh price", { exact: true })).toHaveCount(0);
  await expect(row(page, "Amazon speaker").getByText("Pending", { exact: true })).toBeVisible();
});

test("action is disabled while its request is in flight and works by keyboard", async ({ page, context }) => {
  const api = await mockApi(context);
  let release!: () => void;
  api.refreshGate = new Promise<void>((resolve) => { release = resolve; });
  await page.goto("/");
  await row(page, "Amazon headphones").getByRole("button", { name: "Options", exact: true }).focus();
  await page.keyboard.press("Enter");
  const refreshAction = page.getByRole("menuitem", { name: "Refresh price" });
  await expect(page.getByRole("menuitem", { name: "View details" })).toBeFocused();
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("ArrowDown");
  await expect(refreshAction).toBeFocused();
  await page.keyboard.press("Enter");
  await openMenu(page, "Amazon headphones");
  await expect(refreshAction).toBeDisabled();
  await page.keyboard.press("Escape");
  release();
  await expect(row(page, "Amazon headphones").getByText("Pending", { exact: true })).toBeVisible();
});

test("refresh-all in progress keeps its indicator and pending items disabled", async ({ page, context }) => {
  await mockApi(context);
  await page.goto("/");
  await page.getByRole("button", { name: "Refresh all prices" }).click();
  await expect(page.getByText("Refreshing prices for 3 items…", { exact: true })).toBeVisible();
  await openMenu(page, "Amazon speaker");
  await expect(page.getByRole("menuitem", { name: "Refresh price" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Refresh all prices" })).toBeDisabled();
});
