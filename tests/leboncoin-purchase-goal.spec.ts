import { expect, test, type BrowserContext } from "@playwright/test";

const timestamp = "2026-10-04T08:00:00Z";
const baseItem = {
  id: "lbc-1", title: "LeBoncoin laptop", platform: "leboncoin", listingId: "1234567890", asin: null,
  marketplace: "leboncoin.fr", url: "https://www.leboncoin.fr/ad/ordinateurs/1234567890", thumbnailUrl: null,
  status: "active", latestPrice: { amountCents: 15000, currency: "EUR", timestamp },
  lastThreeDetections: [{ amountCents: 15000, currency: "EUR", timestamp }], secondHandOffer: null,
  lastAttempt: null, nextCheckAt: null, addedAt: timestamp, purchaseGoal: "",
};
const idleReview = { tokenConfigured: true, running: false, lastAttempt: null, latest: null, history: [] };

async function mockApi(context: BrowserContext, options: { goal?: string; platform?: string; failSave?: boolean } = {}) {
  const state = { goal: options.goal ?? "", running: false, puts: [] as string[], posts: [] as unknown[] };
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== "http://127.0.0.1:4173") return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    if (url.pathname === "/api/v1/auth/session") return route.fulfill({ json: { mode: "open", user: null } });
    const method = route.request().method();
    const item = () => ({ ...baseItem, platform: options.platform ?? "leboncoin", purchaseGoal: state.goal });
    if (method === "GET" && url.pathname === "/api/v1/items") return route.fulfill({ json: { items: [] } });
    if (method === "POST" && url.pathname === "/api/v1/items") {
      const body = route.request().postDataJSON();
      state.posts.push(body);
      state.goal = body.purchaseGoal.trim();
      return route.fulfill({ status: 201, json: item() });
    }
    if (method === "GET" && url.pathname === `/api/v1/items/${baseItem.id}`) {
      return route.fulfill({ json: { ...item(), aiReview: options.platform === "amazon" ? null : { ...idleReview, running: state.running } } });
    }
    if (method === "PUT" && url.pathname === `/api/v1/items/${baseItem.id}/purchase-goal`) {
      const goal = route.request().postDataJSON().purchaseGoal;
      state.puts.push(goal);
      if (options.failSave) {
        return route.fulfill({ status: 500, json: { error: { code: "INTERNAL_ERROR", message: "The server could not complete the request." } } });
      }
      state.goal = goal.trim();
      state.running = true;
      return route.fulfill({ json: { purchaseGoal: state.goal, changed: true, reviewStarted: true } });
    }
    return route.abort();
  });
  return state;
}

test("add form sends the purchase goal and the details page shows it", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  await page.getByRole("button", { name: "Add item", exact: true }).first().click();
  const dialog = page.getByRole("dialog");
  await dialog.getByRole("textbox", { name: "Listing URL" }).fill(baseItem.url);
  await dialog.getByRole("textbox", { name: "Purchase goal (optional)" }).fill("Install a light Linux distro");
  await dialog.getByRole("button", { name: "Add item", exact: true }).click();
  await expect(dialog).toBeHidden();
  expect(api.posts).toEqual([{ url: baseItem.url, purchaseGoal: "Install a light Linux distro" }]);
  await page.goto(`/items/${baseItem.id}`);
  await expect(page.getByRole("textbox", { name: "Purchase goal" })).toHaveValue("Install a light Linux distro");
});

test("saving a changed goal shows success and a pending review; clearing sends an empty goal", async ({ page, context }) => {
  const api = await mockApi(context, { goal: "Goal A" });
  await page.goto(`/items/${baseItem.id}`);
  const field = page.getByRole("textbox", { name: "Purchase goal" });
  const save = page.getByRole("button", { name: "Save goal" });
  await expect(field).toHaveValue("Goal A");
  await expect(save).toBeDisabled();
  await field.fill("Goal B");
  await save.click();
  await expect(page.getByText("Purchase goal saved. A new AI review was requested.")).toBeVisible();
  await expect(page.getByText("AI review in progress…")).toBeVisible();
  api.running = false;
  await field.fill("");
  await save.click();
  await expect(field).toHaveValue("");
  expect(api.puts).toEqual(["Goal B", ""]);
  await page.reload();
  await expect(page.getByRole("textbox", { name: "Purchase goal" })).toHaveValue("");
});

test("failed save shows an error and keeps the text; keyboard save works", async ({ page, context }) => {
  const api = await mockApi(context, { failSave: true });
  await page.goto(`/items/${baseItem.id}`);
  const field = page.getByRole("textbox", { name: "Purchase goal" });
  await field.fill("Long goal\nwith several paragraphs");
  await page.keyboard.press("Tab");
  await expect(page.getByRole("button", { name: "Save goal" })).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.getByText("Could not save purchase goal")).toBeVisible();
  await expect(field).toHaveValue("Long goal\nwith several paragraphs");
  expect(api.puts).toEqual(["Long goal\nwith several paragraphs"]);
});

test("Amazon details have no purchase goal field", async ({ page, context }) => {
  await mockApi(context, { platform: "amazon" });
  await page.goto(`/items/${baseItem.id}`);
  await expect(page.getByRole("heading", { name: "LeBoncoin laptop" })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Purchase goal" })).toHaveCount(0);
});
