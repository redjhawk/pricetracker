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

// Issue #267 (TC-014): the requests the interface sends pass the API's same-origin and JSON content-type checks.
const appOrigin = "http://127.0.0.1:4173";
const originNotAllowed = "Requests from another site are not allowed.";
const idleAiReview = { tokenConfigured: true, running: false, lastAttempt: null, latest: null, history: [] };
const noLeboncoinSession = { saved: false, hint: null, revision: 0, updatedAt: null, status: "none", expiresAt: null, revokedAt: null, lastAttempt: null };
type RecordedRequest = { method: string; path: string; contentType: string | null; origin: string | null; body: string | null };
type SessionUser = { username: string; role: "admin" | "user" };

// mockProtectedApi serves a protected-mode API and records every API request with its checked headers and body.
async function mockProtectedApi(context: BrowserContext, options: { user?: SessionUser; rejectAdd?: boolean } = {}) {
  const state = {
    user: (options.user ?? null) as SessionUser | null,
    items: [] as (FixtureItem & { purchaseGoal: string })[],
    users: [] as { username: string; lastLoginAt: string | null }[],
    claudeToken: { saved: true, hint: "wxyz", updatedAt: timestamp, lastRejectedAt: null },
    recorded: [] as RecordedRequest[],
    unexpected: [] as string[],
  };
  await context.route("**/*", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (url.origin !== appOrigin) return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    const method = request.method();
    const path = url.pathname;
    state.recorded.push({
      method, path, contentType: await request.headerValue("content-type"), origin: await request.headerValue("origin"), body: request.postData(),
    });
    const key = `${method} ${path}`;
    const itemMatch = path.match(/^\/api\/v1\/items\/([^/]+)(\/[a-z-]+)?$/);
    const item = itemMatch ? state.items.find((candidate) => candidate.id === decodeURIComponent(itemMatch[1])) : undefined;
    const action = `${method} ${itemMatch?.[2] ?? ""}`;
    if (key === "GET /api/v1/auth/session") return route.fulfill({ json: { mode: "protected", user: state.user, version: "v1" } });
    if (key === "POST /api/v1/auth/login") {
      const username = String(request.postDataJSON().username);
      state.user = { username, role: username === "admin" ? "admin" : "user" };
      return route.fulfill({ json: { user: state.user } });
    }
    if (key === "POST /api/v1/auth/logout") {
      state.user = null;
      return route.fulfill({ status: 204 });
    }
    if (key === "GET /api/v1/admin/users") return route.fulfill({ json: { users: state.users } });
    if (key === "POST /api/v1/admin/users") {
      const user = { username: String(request.postDataJSON().username), lastLoginAt: null };
      state.users.push(user);
      return route.fulfill({ status: 201, json: { user } });
    }
    if (key === "GET /api/v1/items") return route.fulfill({ json: { items: state.items } });
    if (key === "POST /api/v1/items") {
      if (options.rejectAdd) return route.fulfill({ status: 403, json: { error: { code: "ORIGIN_NOT_ALLOWED", message: originNotAllowed } } });
      const added = { ...leboncoin, purchaseGoal: String(request.postDataJSON().purchaseGoal).trim() };
      state.items = [added, ...state.items];
      return route.fulfill({ status: 201, json: added });
    }
    if (key === "POST /api/v1/items/refresh") return route.fulfill({ status: 202, json: { requestedAt: timestamp, itemsQueued: state.items.length } });
    if (key === "GET /api/v1/purchase-goals") {
      return route.fulfill({ json: { purchaseGoals: [...new Set(state.items.map((entry) => entry.purchaseGoal).filter(Boolean))] } });
    }
    if (item && action === "GET ") return route.fulfill({ json: { ...item, aiReview: idleAiReview } });
    if (item && action === "DELETE ") {
      state.items = state.items.filter((candidate) => candidate !== item);
      return route.fulfill({ status: 204 });
    }
    if (item && action === "PUT /purchase-goal") {
      item.purchaseGoal = String(request.postDataJSON().purchaseGoal).trim();
      return route.fulfill({ json: { purchaseGoal: item.purchaseGoal, changed: true, reviewStarted: false } });
    }
    if (item && action === "POST /refresh") return route.fulfill({ status: 202, json: { requestedAt: timestamp, itemsQueued: 1 } });
    if (item && action === "POST /ai-review") return route.fulfill({ status: 202, json: { requestedAt: timestamp, alreadyRunning: false } });
    if (key === "GET /api/v1/settings/leboncoin-session") return route.fulfill({ json: { session: noLeboncoinSession } });
    if (key === "GET /api/v1/settings/claude-token") return route.fulfill({ json: { claudeToken: state.claudeToken } });
    if (key === "PUT /api/v1/settings") {
      const value = request.postDataJSON().claudeToken?.value;
      if (typeof value === "string" && value.trim()) state.claudeToken = { ...state.claudeToken, hint: value.trim().slice(-4) };
      return route.fulfill({ json: { session: noLeboncoinSession, claudeToken: state.claudeToken } });
    }
    state.unexpected.push(key);
    return route.abort();
  });
  return state;
}

async function logInAs(page: Page, username: string) {
  await page.getByRole("textbox", { name: "Username" }).fill(username);
  await page.locator("#login-password").fill("synthetic-password-123");
  await page.getByRole("button", { name: "Log in", exact: true }).click();
}

test("TC-014 (#267) the interface sends requests that pass the same-origin and JSON content-type checks", async ({ page, context }) => {
  const api = await mockProtectedApi(context);
  const label = (entry: RecordedRequest) => `${entry.method} ${entry.path}`;
  const sent = (key: string) => api.recorded.some((entry) => label(entry) === key);
  const itemPath = `/api/v1/items/${leboncoin.id}`;
  await page.goto("/");
  await logInAs(page, "alice");

  await page.getByRole("button", { name: "Add item", exact: true }).first().click();
  const addDialog = page.getByRole("dialog", { name: "Add tracked item" });
  await addDialog.getByRole("textbox", { name: "Listing URL" }).fill(leboncoin.url);
  await addDialog.getByRole("button", { name: "Add item", exact: true }).click();
  await expect(addDialog).toBeHidden();
  await page.getByRole("tab", { name: "LeBoncoin" }).click();
  await page.getByRole("button", { name: "LeBoncoin bicycle", exact: true }).click();
  await expect(page.getByRole("heading", { name: "LeBoncoin bicycle" })).toBeVisible();

  await page.getByRole("textbox", { name: "Purchase goal" }).fill("Commute to work");
  await page.getByRole("button", { name: "Save goal" }).click();
  await expect(page.getByText("Purchase goal saved.", { exact: true })).toBeVisible();

  const refreshPrice = page.getByRole("button", { name: "Refresh price", exact: true });
  await refreshPrice.click();
  await expect.poll(() => sent(`POST ${itemPath}/refresh`)).toBe(true);
  await expect(refreshPrice).toBeEnabled();

  const refreshReview = page.getByRole("region", { name: "AI review" }).getByRole("button", { name: "Refresh AI review" });
  await refreshReview.click();
  await expect.poll(() => sent(`POST ${itemPath}/ai-review`)).toBe(true);
  await expect(refreshReview).toBeEnabled();

  await page.getByRole("button", { name: "Application menu", exact: true }).click();
  await page.getByRole("menuitem", { name: "Settings", exact: true }).click();
  const settings = page.getByRole("dialog", { name: "Settings" });
  await settings.getByRole("textbox", { name: "Claude token" }).fill("sk-ant-oat01-synthetic-new-uvst");
  await settings.getByRole("button", { name: "Save", exact: true }).click();
  await expect(settings).toBeHidden();

  await page.getByRole("button", { name: "Tracked items", exact: true }).click();
  await page.getByRole("button", { name: "Refresh all prices" }).click();
  await expect.poll(() => sent("POST /api/v1/items/refresh")).toBe(true);
  await expect(page.getByText(/Refreshing prices for/)).toHaveCount(0);

  await openMenu(page, "LeBoncoin bicycle");
  await page.getByRole("menuitem", { name: "Delete", exact: true }).click();
  const deleteDialog = page.getByRole("dialog", { name: "Delete tracked item" });
  await deleteDialog.getByRole("button", { name: "Delete", exact: true }).click();
  await expect(deleteDialog).toBeHidden();
  await expect(page.getByRole("heading", { name: "No items tracked yet" })).toBeVisible();

  await page.getByRole("button", { name: "Application menu", exact: true }).click();
  await page.getByRole("menuitem", { name: "Log out", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Log in to Price follower" })).toBeVisible();

  await logInAs(page, "admin");
  await expect(page.getByRole("heading", { name: "Users" })).toBeVisible();
  await page.getByRole("button", { name: "Add user" }).click();
  const userDialog = page.getByRole("dialog", { name: "Add user" });
  await userDialog.getByRole("textbox", { name: "Username" }).fill("bob");
  await userDialog.locator("#add-user-password").fill("synthetic-password-456");
  await userDialog.getByRole("button", { name: "Add user" }).click();
  await expect(page.getByText("bob can now log in.")).toBeVisible();

  expect(api.recorded.filter((entry) => entry.method !== "GET").map(label)).toEqual([
    "POST /api/v1/auth/login",
    "POST /api/v1/items",
    `PUT ${itemPath}/purchase-goal`,
    `POST ${itemPath}/refresh`,
    `POST ${itemPath}/ai-review`,
    "PUT /api/v1/settings",
    "POST /api/v1/items/refresh",
    `DELETE ${itemPath}`,
    "POST /api/v1/auth/logout",
    "POST /api/v1/auth/login",
    "POST /api/v1/admin/users",
  ]);
  const withBody = api.recorded.filter((entry) => entry.body);
  expect(withBody.map(label)).toEqual([
    "POST /api/v1/auth/login", "POST /api/v1/items", `PUT ${itemPath}/purchase-goal`, "PUT /api/v1/settings",
    "POST /api/v1/auth/login", "POST /api/v1/admin/users",
  ]);
  for (const entry of withBody) {
    expect(entry.contentType, label(entry)).toBe("application/json");
    expect(() => JSON.parse(entry.body!), label(entry)).not.toThrow();
  }
  const bodiless = api.recorded.filter((entry) => /^(POST \/api\/v1\/(items\/[^/]+\/(refresh|ai-review)|items\/refresh|auth\/logout)|DELETE .*)$/.test(label(entry)));
  expect(bodiless).toHaveLength(5);
  for (const entry of bodiless) expect(entry.body ?? "", label(entry)).toBe("");
  for (const entry of api.recorded) if (entry.origin !== null) expect(entry.origin, label(entry)).toBe(appOrigin);
  // The Origin check above is not vacuous: the browser sends Origin on state-changing requests.
  expect(api.recorded.some((entry) => entry.method !== "GET" && entry.origin === appOrigin)).toBe(true);
  expect(api.unexpected).toEqual([]);
});

test("TC-014 (#267) a 403 ORIGIN_NOT_ALLOWED answer to adding an item shows the server message in the add form", async ({ page, context }) => {
  const api = await mockProtectedApi(context, { user: { username: "alice", role: "user" }, rejectAdd: true });
  await page.goto("/");
  await page.getByRole("button", { name: "Add item", exact: true }).first().click();
  const dialog = page.getByRole("dialog", { name: "Add tracked item" });
  const field = dialog.getByRole("textbox", { name: "Listing URL" });
  await field.fill(leboncoin.url);
  await dialog.getByRole("button", { name: "Add item", exact: true }).click();
  const notification = dialog.locator(".cds--inline-notification").filter({ hasText: "Could not add item" });
  await expect(notification).toContainText(originNotAllowed);
  await expect(dialog.locator(".cds--form-requirement")).toHaveText(originNotAllowed);
  await expect(dialog).toBeVisible();
  await expect(field).toHaveValue(leboncoin.url);
  await expect(page.locator(".cds--inline-notification")).toHaveCount(1);
  await expect(page.getByText(originNotAllowed, { exact: true })).toHaveCount(2);
  await expect(page.getByRole("heading", { name: "Log in to Price follower" })).toHaveCount(0);
  expect(api.recorded.filter((entry) => entry.method !== "GET").map((entry) => `${entry.method} ${entry.path}`)).toEqual(["POST /api/v1/items"]);
  expect(api.unexpected).toEqual([]);
});
