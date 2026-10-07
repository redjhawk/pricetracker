import { expect, test, type BrowserContext } from "@playwright/test";

const timestamp = "2026-10-03T08:00:00Z";
const price = { amountCents: 1299, currency: "EUR", timestamp };
const usedPrice = { ...price, amountCents: 899, condition: "good", conditionLabel: "Bon état" };
const amazon = {
  id: "amazon-one", title: "Amazon headphones", platform: "amazon",
  listingId: "B09XS7JWHH", asin: "B09XS7JWHH", marketplace: "amazon.de",
  url: "https://www.amazon.de/dp/B09XS7JWHH", thumbnailUrl: null,
  status: "active", latestPrice: price, lastThreeDetections: [price],
  secondHandOffer: {
    status: "available", latestDetection: usedPrice,
    lastThreeDetections: [usedPrice], lastCheckedAt: timestamp,
  },
  lastAttempt: { result: "success", timestamp, message: null },
  nextCheckAt: null, addedAt: timestamp, purchaseGoal: "",
};
const leboncoin = {
  ...amazon, id: "leboncoin-one", title: "LeBoncoin bicycle", platform: "leboncoin",
  listingId: "1234567890", asin: null, marketplace: "leboncoin.fr",
  url: "https://www.leboncoin.fr/ad/velos/1234567890", secondHandOffer: null,
};
type FixtureItem = Omit<typeof amazon, "title" | "asin" | "secondHandOffer" | "latestPrice"> & {
  title: string | null;
  asin: string | null;
  secondHandOffer: (Omit<typeof amazon.secondHandOffer, "latestDetection"> & {
    latestDetection: typeof usedPrice | null;
  }) | null;
  latestPrice: typeof price | null;
};

const unexpectedRequests = new WeakMap<BrowserContext, string[]>();

test.afterEach(async ({ context }) => {
  expect(unexpectedRequests.get(context)).toEqual([]);
});

async function mockApi(context: BrowserContext, initialItems: FixtureItem[] = [amazon, leboncoin]) {
  const state = {
    items: structuredClone(initialItems),
    listError: false,
    refreshError: false,
    listGate: Promise.resolve(),
    requests: [] as { method: string; path: string; body: string | null }[],
    unexpected: [] as string[],
    addedItem: { ...leboncoin, id: "leboncoin-added", title: "Added bicycle", status: "pending",
      listingId: "1234567891", url: "https://www.leboncoin.fr/ad/velos/1234567891",
      latestPrice: null, lastThreeDetections: [],
      lastAttempt: { result: "pending", timestamp, message: null } } as FixtureItem,
  };
  unexpectedRequests.set(context, state.unexpected);
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
    const amazonRequests = { stopped: false, stoppedAt: null, consecutiveFailures: 0 };
    if (url.pathname === "/api/v1/amazon/requests") return route.fulfill({ json: { amazonRequests } });
    if (url.pathname === "/api/v1/amazon-searches") return route.fulfill({ json: { searches: [], amazonRequests } });
    const method = request.method();
    const path = url.pathname + url.search;
    state.requests.push({ method, path, body: request.postData() });
    if (method === "GET" && path === "/api/v1/items") {
      await state.listGate;
      await route.fulfill(state.listError
        ? { status: 500, json: { error: { code: "TEST_FAILURE", message: "Fixture retrieval failed." } } }
        : { json: { items: state.items } });
    } else if (method === "POST" && path === "/api/v1/items/refresh") {
      if (state.refreshError) {
        await route.fulfill({ status: 500, json: { error: { code: "TEST_FAILURE", message: "Fixture refresh failed." } } });
      } else {
        state.items = state.items.map((item) => ({ ...item, status: "pending" }));
        await route.fulfill({ status: 202, json: { requestedAt: timestamp, itemsQueued: state.items.length } });
      }
    } else if (method === "POST" && path === "/api/v1/items") {
      expect(request.postDataJSON()).toEqual({ url: state.addedItem.url, purchaseGoal: "" });
      state.items.unshift(state.addedItem);
      await route.fulfill({ status: 201, json: state.addedItem });
    } else {
      const item = state.items.find((entry) => path === `/api/v1/items/${entry.id}`);
      if (item && method === "GET") {
        await route.fulfill({ json: { ...item, priceHistory: item.lastThreeDetections,
          secondHandOffer: item.secondHandOffer && { ...item.secondHandOffer, priceHistory: item.secondHandOffer.lastThreeDetections } } });
      } else if (item && method === "DELETE") {
        state.items = state.items.filter((entry) => entry.id !== item.id);
        await route.fulfill({ status: 204 });
      } else {
        state.unexpected.push(`${method} ${path}`);
        await route.abort();
      }
    }
  });
  return state;
}

test("core: separate platform rows and five/four-column tables", async ({ page, context }) => {
  const api = await mockApi(context, [amazon, leboncoin, { ...amazon, id: "amazon-two", title: "Second Amazon item" }]);
  await page.goto("/");
  await expect(page.getByRole("tab", { name: "Amazon", exact: true })).toBeVisible();
  await expect(page.getByRole("tab", { name: "Amazon", exact: true })).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("tab", { name: "LeBoncoin", exact: true })).toBeVisible();
  await expect(page.getByRole("columnheader")).toHaveCount(5);
  await expect(page.getByRole("table").getByRole("button", { name: /Amazon headphones|Second Amazon item/ })).toHaveText(["Amazon headphones", "Second Amazon item"]);
  await expect(page.getByRole("button", { name: "LeBoncoin bicycle", exact: true })).toHaveCount(0);
  await page.getByRole("tab", { name: "LeBoncoin", exact: true }).click();
  await expect(page.getByRole("columnheader")).toHaveCount(4);
  await expect(page.getByRole("columnheader", { name: "Amazon second-hand offer" })).toHaveCount(0);
  await expect(page.getByRole("row").nth(1).getByRole("cell")).toHaveCount(4);
  await expect(page.getByRole("button", { name: "LeBoncoin bicycle", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Amazon headphones", exact: true })).toHaveCount(0);
  expect(api.unexpected).toEqual([]);
});

test("search stays scoped and retained across tabs", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  const search = page.getByRole("textbox", { name: "Search tracked items" });
  for (const query of ["  HEADPHONES ", "B09XS7JWHH", "amazon.de", "AMAZON"]) {
    await search.fill(query);
    await expect(page.getByRole("button", { name: "Amazon headphones", exact: true })).toBeVisible();
  }
  await search.fill(" BICYCLE ");
  await expect(page.getByRole("cell", { name: "No items match your search." })).toHaveAttribute("colspan", "5");
  await page.getByRole("tab", { name: "LeBoncoin" }).click();
  await expect(search).toHaveValue(" BICYCLE ");
  await expect(page.getByRole("button", { name: "LeBoncoin bicycle", exact: true })).toBeVisible();
  await search.fill("B09XS7JWHH");
  await expect(page.getByRole("cell", { name: "No items match your search." })).toHaveAttribute("colspan", "4");
  await search.fill("1234567890");
  await expect(page.getByRole("button", { name: "LeBoncoin bicycle", exact: true })).toBeVisible();
  await search.fill("");
  await expect(page.getByRole("row")).toHaveCount(2);
  await expect(page.getByText("2 items tracked", { exact: false })).toBeVisible();
  expect(api.requests.every(({ method, path }) => method === "GET" && path === "/api/v1/items")).toBe(true);
});

test("both tabs remain available for empty platform and collection", async ({ page, context }) => {
  const api = await mockApi(context, [leboncoin]);
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "No Amazon items tracked yet" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Refresh all prices" })).toBeEnabled();
  await page.getByRole("tab", { name: "LeBoncoin" }).click();
  await expect(page.getByRole("table")).toBeVisible();
  api.items = [];
  await page.reload();
  await expect(page.getByRole("heading", { name: "No items tracked yet" })).toBeVisible();
  await expect(page.getByRole("tab")).toHaveCount(3);
  await expect(page.getByRole("button", { name: "Refresh all prices" })).toBeDisabled();
  await page.getByRole("tab", { name: "LeBoncoin" }).click();
  await expect(page.getByRole("button", { name: "Add your first item" })).toBeVisible();
});

test("loading and retrieval errors take precedence over platform emptiness", async ({ page, context }) => {
  const api = await mockApi(context, []);
  let release!: () => void;
  api.listGate = new Promise<void>((resolve) => { release = resolve; });
  await page.goto("/");
  await page.getByRole("tab", { name: "LeBoncoin" }).click();
  await expect(page.getByText("Loading tracked items…", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: /No .*items tracked/ })).toHaveCount(0);
  api.listError = true;
  release();
  await expect(page.getByText("Could not load tracked items", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: /No .*items tracked/ })).toHaveCount(0);
  api.listError = false;
  api.items = [leboncoin];
  await page.getByRole("button", { name: "Retry", exact: true }).click();
  await expect(page.getByRole("button", { name: "LeBoncoin bicycle", exact: true })).toBeVisible();
  await expect(page.getByRole("tab", { name: "LeBoncoin" })).toHaveAttribute("aria-selected", "true");
});

test("global refresh reports all queued items and polling preserves selection", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  await page.getByRole("tab", { name: "LeBoncoin" }).click();
  await page.getByRole("textbox", { name: "Search tracked items" }).fill("bicycle");
  api.refreshError = true;
  await page.getByRole("button", { name: "Refresh all prices" }).click();
  await expect(page.getByText("Could not refresh prices", { exact: true })).toBeVisible();
  api.refreshError = false;
  await page.getByRole("button", { name: "Refresh all prices" }).click();
  await expect(page.getByText("Refreshing prices for 2 items…", { exact: true })).toBeVisible();
  expect(api.requests.filter(({ method }) => method === "POST")).toEqual([
    { method: "POST", path: "/api/v1/items/refresh", body: null },
    { method: "POST", path: "/api/v1/items/refresh", body: null },
  ]);
  api.items = [amazon, { ...leboncoin, title: "Updated bicycle" }];
  await expect(page.getByRole("button", { name: "Updated bicycle", exact: true })).toBeVisible({ timeout: 10000 });
  await expect(page.getByRole("tab", { name: "LeBoncoin" })).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("textbox", { name: "Search tracked items" })).toHaveValue("bicycle");
  await expect(page.getByRole("button", { name: "Refresh all prices" })).toBeEnabled();
});

test("details, adding and deleting retain platform selection; reload resets it", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  await page.getByRole("button", { name: "Add item", exact: true }).first().click();
  const dialog = page.getByRole("dialog");
  await dialog.getByRole("textbox", { name: "Listing URL" }).fill(api.addedItem.url);
  await dialog.getByRole("button", { name: "Add item", exact: true }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByRole("tab", { name: "Amazon", exact: true })).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("button", { name: "Added bicycle", exact: true })).toHaveCount(0);
  await page.getByRole("tab", { name: "LeBoncoin" }).click();
  await expect(page.getByRole("button", { name: "Added bicycle", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "LeBoncoin bicycle", exact: true }).click();
  await expect(page).toHaveURL("/items/leboncoin-one");
  await page.getByRole("button", { name: "Tracked items", exact: true }).click();
  await expect(page.getByRole("tab", { name: "LeBoncoin" })).toHaveAttribute("aria-selected", "true");
  for (const title of ["Added bicycle", "LeBoncoin bicycle"]) {
    await page.getByRole("row").filter({ has: page.getByRole("button", { name: title, exact: true }) }).getByRole("button", { name: "Options", exact: true }).click();
    await page.getByRole("menuitem", { name: "Delete", exact: true }).click();
    await page.getByRole("dialog").getByRole("button", { name: "Delete", exact: true }).click();
    await expect(page.getByRole("dialog")).toBeHidden();
  }
  await expect(page.getByRole("heading", { name: "No LeBoncoin items tracked yet" })).toBeVisible();
  await page.reload();
  await expect(page.getByRole("tab", { name: "Amazon", exact: true })).toHaveAttribute("aria-selected", "true");
});

test("existing prices, statuses and Amazon offer states survive the split", async ({ page, context }) => {
  const offers = [
    { ...amazon, status: "stale" },
    { ...amazon, id: "pending", title: null, status: "pending", latestPrice: null, lastThreeDetections: [],
      secondHandOffer: { ...amazon.secondHandOffer, status: "pending", latestDetection: null } },
    { ...amazon, id: "unavailable", title: "Unavailable item", status: "unavailable",
      secondHandOffer: { ...amazon.secondHandOffer, status: "not_found" } },
    { ...amazon, id: "failed", title: "Failed item", status: "retrieval_error",
      secondHandOffer: { ...amazon.secondHandOffer, status: "check_error" } },
    { ...amazon, id: "never-found", title: "No offer yet",
      secondHandOffer: { ...amazon.secondHandOffer, status: "check_error", latestDetection: null } },
  ];
  await mockApi(context, [...offers, { ...leboncoin, latestPrice: { ...price, amountCents: 0 }, lastThreeDetections: [{ ...price, amountCents: 0 }] }]);
  await page.goto("/");
  for (const text of ["Bon état", "Checking Amazon offers…", "No Amazon offer", "Latest check failed", "Amazon offer check unavailable", "Awaiting first price", "Stale", "Pending", "Unavailable", "Retrieval error"]) {
    await expect(page.getByText(text, { exact: true })).toBeVisible();
  }
  await expect(page.getByRole("button", { name: "Title unavailable", exact: true })).toBeVisible();
  await expect(page.getByRole("row").filter({ has: page.getByRole("button", { name: "Failed item", exact: true }) })).toContainText("12,99");
  await page.getByRole("tab", { name: "LeBoncoin" }).click();
  await expect(page.getByText("Gratuit", { exact: true })).toBeVisible();
  await expect(page.getByRole("cell")).toHaveCount(4);
});

test("Carbon keyboard navigation and narrow table scrolling preserve access", async ({ page, context }) => {
  await mockApi(context);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  const amazonTab = page.getByRole("tab", { name: "Amazon", exact: true });
  const leboncoinTab = page.getByRole("tab", { name: "LeBoncoin" });
  await amazonTab.focus();
  await page.keyboard.press("ArrowRight");
  await expect(leboncoinTab).toBeFocused();
  await expect(leboncoinTab).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("tabpanel", { name: "LeBoncoin" })).toHaveAttribute("id", await leboncoinTab.getAttribute("aria-controls") ?? "");
  await page.keyboard.press("Home");
  await expect(amazonTab).toBeFocused();
  await page.keyboard.press("End");
  await expect(page.getByRole("tab", { name: "Amazon searches" })).toBeFocused();
  await page.keyboard.press("ArrowLeft");
  await expect(leboncoinTab).toBeFocused();
  expect(await leboncoinTab.evaluate((node) => getComputedStyle(node).outlineStyle)).not.toBe("none");
  await page.keyboard.press("Tab");
  await expect(page.getByRole("textbox", { name: "Search tracked items" })).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("button", { name: "LeBoncoin bicycle", exact: true })).toBeFocused();
  const identity = page.getByText("leboncoin.fr · 1234567890", { exact: true });
  await expect(identity).toBeVisible();
  expect(await identity.evaluate((node) => node.scrollWidth <= node.clientWidth)).toBe(true);
  const container = page.locator(".tracked-table-container .cds--data-table-content");
  expect(await container.evaluate((node) => node.scrollWidth > node.clientWidth)).toBe(true);
  await page.getByRole("button", { name: "Options", exact: true }).click();
  await expect(page.getByRole("menuitem", { name: "View details", exact: true })).toBeVisible();
});
