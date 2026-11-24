import { expect, test, type BrowserContext, type Locator, type Page, type Route } from "@playwright/test";

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

// Issue #304: choose an existing purchase goal (TC-006 to TC-014).

type GoalsReply = "ok" | "hold" | "fail" | "abort";
type MockItem = { id: string; platform: "leboncoin" | "amazon"; goal: string };

const lbcUrl = baseItem.url;
const otherLbcUrl = "https://www.leboncoin.fr/ad/ordinateurs/2222222222";
const amazonUrl = "https://www.amazon.fr/dp/B012345678";

// The server order: letter case ignored, then the exact text.
function byGoal(a: string, b: string) {
  const x = a.toLowerCase();
  const y = b.toLowerCase();
  if (x !== y) return x < y ? -1 : 1;
  return a < b ? -1 : a > b ? 1 : 0;
}

// Held goal requests must be answered or aborted before the test ends: removing the routes
// would resume them to the dev server proxy, whose backend is not running.
const heldGoalRoutes: Route[] = [];
test.afterEach(async () => {
  await Promise.all(heldGoalRoutes.splice(0).map((route) => route.abort().catch(() => {})));
});

// mockGoalsApi serves the goals list as the server derives it: the goals of other items
// (otherGoals) plus the current goals of the mocked LeBoncoin items, merged and ordered.
async function mockGoalsApi(context: BrowserContext, options: {
  otherGoals?: string[];
  items?: MockItem[];
  goalsReply?: GoalsReply;
  failSave?: boolean;
  postStatus?: number;
} = {}) {
  const state = {
    items: [...(options.items ?? [])],
    goalsReply: options.goalsReply ?? ("ok" as GoalsReply),
    goalRequests: 0,
    running: false,
    posts: [] as { url: string; purchaseGoal: string }[],
    puts: [] as { id: string; purchaseGoal: string }[],
    goals() {
      const all = [...(options.otherGoals ?? []), ...state.items.filter((item) => item.platform === "leboncoin").map((item) => item.goal)];
      return [...new Set(all.filter((goal) => goal.trim() !== ""))].sort(byGoal);
    },
    async releaseGoals() {
      state.goalsReply = "ok";
      await Promise.all(heldGoalRoutes.splice(0).map((route) => route.fulfill({ json: { purchaseGoals: state.goals() } })));
    },
  };
  const json = (item: MockItem) => ({
    ...baseItem, id: item.id, platform: item.platform, purchaseGoal: item.goal,
    title: item.platform === "amazon" ? "Amazon lamp" : baseItem.title,
  });
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== "http://127.0.0.1:4173") return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    if (url.pathname === "/api/v1/auth/session") return route.fulfill({ json: { mode: "open", user: null } });
    const method = route.request().method();
    if (method === "GET" && url.pathname === "/api/v1/purchase-goals") {
      state.goalRequests++;
      if (state.goalsReply === "hold") {
        heldGoalRoutes.push(route);
        return;
      }
      if (state.goalsReply === "abort") return route.abort();
      if (state.goalsReply === "fail") {
        return route.fulfill({ status: 500, json: { error: { code: "INTERNAL_ERROR", message: "The server could not complete the request." } } });
      }
      return route.fulfill({ json: { purchaseGoals: state.goals() } });
    }
    if (method === "GET" && url.pathname === "/api/v1/items") return route.fulfill({ json: { items: state.items.map(json) } });
    if (method === "POST" && url.pathname === "/api/v1/items") {
      const body = route.request().postDataJSON();
      state.posts.push(body);
      if (options.postStatus) {
        return route.fulfill({ status: options.postStatus, json: { error: { code: "ITEM_ALREADY_TRACKED", message: "This listing is already being tracked." } } });
      }
      const platform = body.url.includes("amazon.") ? "amazon" : "leboncoin";
      const item: MockItem = { id: `added-${state.posts.length}`, platform, goal: platform === "amazon" ? "" : body.purchaseGoal.trim() };
      state.items.push(item);
      return route.fulfill({ status: 201, json: json(item) });
    }
    const item = state.items.find((candidate) => url.pathname.startsWith(`/api/v1/items/${candidate.id}`));
    if (item && method === "GET" && url.pathname === `/api/v1/items/${item.id}`) {
      return route.fulfill({ json: { ...json(item), aiReview: item.platform === "amazon" ? null : { ...idleReview, running: state.running } } });
    }
    if (item && method === "PUT" && url.pathname === `/api/v1/items/${item.id}/purchase-goal`) {
      const goal = route.request().postDataJSON().purchaseGoal;
      state.puts.push({ id: item.id, purchaseGoal: goal });
      if (options.failSave) {
        return route.fulfill({ status: 500, json: { error: { code: "INTERNAL_ERROR", message: "The server could not complete the request." } } });
      }
      const changed = goal.trim() !== item.goal;
      item.goal = goal.trim();
      state.running = changed;
      return route.fulfill({ json: { purchaseGoal: item.goal, changed, reviewStarted: changed } });
    }
    return route.abort();
  });
  return state;
}

const itemA = (goal: string): MockItem => ({ id: baseItem.id, platform: "leboncoin", goal });

async function openAddForm(page: Page, url = lbcUrl) {
  await page.getByRole("button", { name: "Add item", exact: true }).first().click();
  const dialog = page.getByRole("dialog");
  await dialog.getByRole("textbox", { name: "Listing URL" }).fill(url);
  return dialog;
}

async function submitAddForm(dialog: Locator) {
  await dialog.getByRole("button", { name: "Add item", exact: true }).click();
  await expect(dialog).toBeHidden();
}

const addChoice = (scope: Page | Locator) => scope.getByRole("combobox", { name: "Purchase goal (optional)", exact: true });
const detailChoice = (page: Page) => page.getByRole("combobox", { name: "Purchase goal", exact: true });
const goalText = (scope: Page | Locator) => scope.getByRole("textbox", { name: "Goal text", exact: true });
const saveGoal = (page: Page) => page.getByRole("button", { name: "Save goal" });
// The value shown by the Carbon drop-down, without the title of its menu icon.
const selectedValue = (combobox: Locator) => combobox.locator(".cds--list-box__label");

async function choose(page: Page, combobox: Locator, label: string, index = 0) {
  await combobox.click();
  await page.getByRole("option", { name: label, exact: true }).nth(index).click();
  await expect(page.getByRole("option")).toHaveCount(0);
}

async function expectOptions(page: Page, combobox: Locator, labels: string[]) {
  await combobox.click();
  await expect(page.getByRole("option")).toHaveText(labels);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("option")).toHaveCount(0);
}

// highlightWithArrows opens the focused drop-down with ArrowDown and moves to the option.
async function highlightWithArrows(page: Page, combobox: Locator, label: string) {
  await page.keyboard.press("ArrowDown");
  const option = page.getByRole("option", { name: label, exact: true }).first();
  await expect(option).toBeVisible();
  const id = (await option.getAttribute("id")) ?? "";
  for (let step = 0; step < 10 && (await combobox.getAttribute("aria-activedescendant")) !== id; step++) {
    await page.keyboard.press("ArrowDown");
  }
  await expect(combobox).toHaveAttribute("aria-activedescendant", id);
}

async function tabTo(page: Page, target: Locator) {
  for (let step = 0; step < 6 && !(await target.evaluate((element) => element === document.activeElement)); step++) {
    await page.keyboard.press("Tab");
  }
  await expect(target).toBeFocused();
}

// expectOneLine checks that the visible text of element is laid out on a single line.
async function expectOneLine(element: Locator, what: string) {
  const spread = await element.evaluate((root) => {
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    const tops: number[] = [];
    for (let node = walker.nextNode(); node; node = walker.nextNode()) {
      if (node.parentElement?.closest("svg")) continue;
      const range = document.createRange();
      range.selectNodeContents(node);
      for (const rect of range.getClientRects()) if (rect.width > 0) tops.push(rect.top);
    }
    return tops.length ? Math.max(...tops) - Math.min(...tops) : 0;
  });
  expect(spread, `${what} wraps`).toBeLessThan(4);
}

async function expectFitsNarrowScreen(page: Page, combobox: Locator, field: Locator) {
  const width = page.viewportSize()!.width;
  const inside = async (locator: Locator, what: string) => {
    const box = await locator.boundingBox();
    expect(box, what).not.toBeNull();
    expect(box!.x, `${what} left`).toBeGreaterThanOrEqual(-0.5);
    expect(box!.x + box!.width, `${what} right`).toBeLessThanOrEqual(width + 0.5);
  };
  await inside(combobox, "drop-down");
  await inside(field, "goal text");
  await expectOneLine(selectedValue(combobox), "selected value");
  await combobox.click();
  const options = page.getByRole("option");
  await expect(options.first()).toBeVisible();
  for (const [index, option] of (await options.all()).entries()) {
    await inside(option, `option ${index}`);
    await expectOneLine(option, `option ${index}`);
  }
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  await page.keyboard.press("Escape");
}

test.describe("issue #304 add form", () => {
  test("TC-006 choosing an existing goal in the add form, in order and with four-word labels", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { otherGoals: ["browsing", "Gaming", "gaming", "install a light Linux distro for my parents", "photo editing"] });
    await page.goto("/");
    const dialog = await openAddForm(page);
    const choice = addChoice(dialog);
    await expect(choice).toBeVisible();
    await expect(dialog.getByText("Purchase goal (optional)", { exact: true })).toBeVisible();
    await expect(selectedValue(choice)).toHaveText("No goal");
    await expect(goalText(dialog)).toHaveCount(0);
    await expect(dialog.getByRole("textbox", { name: "Purchase goal (optional)" })).toHaveCount(0);
    await expectOptions(page, choice, ["No goal", "browsing", "Gaming", "gaming", "install a light Linux…", "photo editing", "New goal"]);
    await choose(page, choice, "install a light Linux…");
    await expect(selectedValue(choice)).toHaveText("install a light Linux…");
    await expect(goalText(dialog)).toHaveValue("install a light Linux distro for my parents");
    await expect(goalText(dialog)).toBeEditable();
    await submitAddForm(dialog);
    expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: "install a light Linux distro for my parents" }]);
    await page.goto(`/items/${api.items[0].id}`);
    await expect(selectedValue(detailChoice(page))).toHaveText("install a light Linux…");
    await expect(goalText(page)).toHaveValue("install a light Linux distro for my parents");
  });

  test("TC-006 labels: four words in full, five words shortened, one word as is", async ({ page, context }) => {
    await mockGoalsApi(context, { otherGoals: ["Gaming", "set up a home server", "set up a server"] });
    await page.goto("/");
    const dialog = await openAddForm(page);
    await expectOptions(page, addChoice(dialog), ["No goal", "Gaming", "set up a home…", "set up a server", "New goal"]);
    await choose(page, addChoice(dialog), "set up a server");
    await expect(selectedValue(addChoice(dialog))).toHaveText("set up a server");
    await choose(page, addChoice(dialog), "set up a home…");
    await expect(selectedValue(addChoice(dialog))).toHaveText("set up a home…");
    await expect(goalText(dialog)).toHaveValue("set up a home server");
    await choose(page, addChoice(dialog), "Gaming");
    await expect(selectedValue(addChoice(dialog))).toHaveText("Gaming");
  });

  test("TC-006 goals with the same first four words stay separate options", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { otherGoals: ["install a light Linux distro", "install a light Linux laptop"] });
    await page.goto("/");
    const dialog = await openAddForm(page);
    await expectOptions(page, addChoice(dialog), ["No goal", "install a light Linux…", "install a light Linux…", "New goal"]);
    await choose(page, addChoice(dialog), "install a light Linux…", 1);
    await expect(goalText(dialog)).toHaveValue("install a light Linux laptop");
    await choose(page, addChoice(dialog), "install a light Linux…", 0);
    await expect(goalText(dialog)).toHaveValue("install a light Linux distro");
    await choose(page, addChoice(dialog), "install a light Linux…", 1);
    await submitAddForm(dialog);
    expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: "install a light Linux laptop" }]);
  });

  test("TC-006 a goal with tabs and line breaks is shown on one line", async ({ page, context }) => {
    await mockGoalsApi(context, { otherGoals: ["install\ta\nlight Linux distro"] });
    await page.goto("/");
    const dialog = await openAddForm(page);
    await expectOptions(page, addChoice(dialog), ["No goal", "install a light Linux…", "New goal"]);
    await choose(page, addChoice(dialog), "install a light Linux…");
    await expect(selectedValue(addChoice(dialog))).toHaveText("install a light Linux…");
    await expectOneLine(selectedValue(addChoice(dialog)), "selected value");
    await expect(goalText(dialog)).toHaveValue("install\ta\nlight Linux distro");
  });

  test("TC-006 keeping No goal adds the item without a goal", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { otherGoals: ["Gaming"] });
    await page.goto("/");
    const dialog = await openAddForm(page);
    await expect(selectedValue(addChoice(dialog))).toHaveText("No goal");
    await submitAddForm(dialog);
    expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: "" }]);
  });

  test("TC-007 New goal shows an empty field; the reopened form offers the new goal", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { otherGoals: ["Gaming", "install a light Linux distro"] });
    await page.goto("/");
    let dialog = await openAddForm(page);
    await choose(page, addChoice(dialog), "New goal");
    await expect(selectedValue(addChoice(dialog))).toHaveText("New goal");
    await expect(goalText(dialog)).toBeVisible();
    await expect(goalText(dialog)).toHaveValue("");
    await goalText(dialog).fill("photo editing");
    await submitAddForm(dialog);
    expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: "photo editing" }]);
    const requests = api.goalRequests;
    dialog = await openAddForm(page, otherLbcUrl);
    await expect(selectedValue(addChoice(dialog))).toHaveText("No goal");
    expect(api.goalRequests).toBeGreaterThan(requests);
    await expect(goalText(dialog)).toHaveCount(0);
    await expectOptions(page, addChoice(dialog), ["No goal", "Gaming", "install a light Linux…", "photo editing", "New goal"]);
  });

  test("TC-007 a whitespace-only new goal adds the item without a goal", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { otherGoals: ["Gaming"] });
    await page.goto("/");
    let dialog = await openAddForm(page);
    await choose(page, addChoice(dialog), "New goal");
    await goalText(dialog).fill("   ");
    await submitAddForm(dialog);
    expect(api.posts).toHaveLength(1);
    expect(api.posts[0].purchaseGoal.trim()).toBe("");
    expect(api.items[0].goal).toBe("");
    dialog = await openAddForm(page, otherLbcUrl);
    await expectOptions(page, addChoice(dialog), ["No goal", "Gaming", "New goal"]);
  });

  for (const [typed, reopened] of [
    ["Gaming", ["No goal", "Gaming", "install a light Linux…", "New goal"]],
    ["gaming", ["No goal", "Gaming", "gaming", "install a light Linux…", "New goal"]],
  ] as const) {
    test(`TC-007 new goal "${typed}" next to the existing goal "Gaming"`, async ({ page, context }) => {
      const api = await mockGoalsApi(context, { otherGoals: ["Gaming", "install a light Linux distro"] });
      await page.goto("/");
      let dialog = await openAddForm(page);
      await choose(page, addChoice(dialog), "New goal");
      await goalText(dialog).fill(typed);
      await submitAddForm(dialog);
      expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: typed }]);
      dialog = await openAddForm(page, otherLbcUrl);
      await expectOptions(page, addChoice(dialog), [...reopened]);
    });
  }

  test("TC-007 a failed add keeps the dialog, New goal and the typed text", async ({ page, context }) => {
    await mockGoalsApi(context, { otherGoals: ["Gaming"], postStatus: 409 });
    await page.goto("/");
    const dialog = await openAddForm(page);
    await choose(page, addChoice(dialog), "New goal");
    await goalText(dialog).fill("photo editing");
    await dialog.getByRole("button", { name: "Add item", exact: true }).click();
    await expect(dialog.getByText("This listing is already being tracked.").first()).toBeVisible();
    await expect(dialog).toBeVisible();
    await expect(selectedValue(addChoice(dialog))).toHaveText("New goal");
    await expect(goalText(dialog)).toHaveValue("photo editing");
  });

  test("TC-008 selecting an option replaces the text typed under New goal", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { otherGoals: ["Gaming", "install a light Linux distro for my parents"] });
    await page.goto("/");
    const dialog = await openAddForm(page);
    await choose(page, addChoice(dialog), "New goal");
    await goalText(dialog).fill("x");
    await choose(page, addChoice(dialog), "Gaming");
    await expect(goalText(dialog)).toHaveValue("Gaming");
    await submitAddForm(dialog);
    expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: "Gaming" }]);
  });

  test("TC-008 a modified goal abandoned for No goal is not sent", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { otherGoals: ["Gaming"] });
    await page.goto("/");
    const dialog = await openAddForm(page);
    await choose(page, addChoice(dialog), "Gaming");
    await goalText(dialog).fill("Gaming!");
    await choose(page, addChoice(dialog), "No goal");
    await expect(goalText(dialog)).toHaveCount(0);
    await submitAddForm(dialog);
    expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: "" }]);
  });

  test("TC-008 New goal after a modified goal or abandoned text starts empty", async ({ page, context }) => {
    await mockGoalsApi(context, { otherGoals: ["Gaming"] });
    await page.goto("/");
    const dialog = await openAddForm(page);
    await choose(page, addChoice(dialog), "Gaming");
    await goalText(dialog).fill("Gaming!");
    await choose(page, addChoice(dialog), "New goal");
    await expect(goalText(dialog)).toHaveValue("");

    await page.goto("/");
    const fresh = await openAddForm(page);
    await choose(page, addChoice(fresh), "New goal");
    await goalText(fresh).fill("a");
    await choose(page, addChoice(fresh), "No goal");
    await choose(page, addChoice(fresh), "New goal");
    await expect(goalText(fresh)).toHaveValue("");
  });

  test("TC-008 an adapted existing goal is sent for the new item only", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { otherGoals: ["Gaming"], items: [{ id: "other", platform: "leboncoin", goal: "install a light Linux distro for my parents" }] });
    await page.goto("/");
    const dialog = await openAddForm(page);
    await choose(page, addChoice(dialog), "install a light Linux…");
    await goalText(dialog).fill("install a light Linux distro for my sister");
    await expect(selectedValue(addChoice(dialog))).toHaveText("install a light Linux…");
    await submitAddForm(dialog);
    expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: "install a light Linux distro for my sister" }]);
    expect(api.puts).toEqual([]);
    expect(api.items.find((item) => item.id === "other")?.goal).toBe("install a light Linux distro for my parents");
  });

  for (const cleared of ["", "   "]) {
    test(`TC-008 an existing goal whose text is cleared to "${cleared}" adds no goal`, async ({ page, context }) => {
      const api = await mockGoalsApi(context, { otherGoals: ["Gaming"] });
      await page.goto("/");
      const dialog = await openAddForm(page);
      await choose(page, addChoice(dialog), "Gaming");
      await goalText(dialog).fill(cleared);
      await submitAddForm(dialog);
      expect(api.posts).toHaveLength(1);
      expect(api.posts[0].purchaseGoal.trim()).toBe("");
    });
  }

  test("TC-008 selecting the same goal again restores its text", async ({ page, context }) => {
    await mockGoalsApi(context, { otherGoals: ["Gaming"] });
    await page.goto("/");
    const dialog = await openAddForm(page);
    await choose(page, addChoice(dialog), "Gaming");
    await goalText(dialog).fill("Gaming on weekends");
    await choose(page, addChoice(dialog), "Gaming");
    await expect(goalText(dialog)).toHaveValue("Gaming");
  });

  test("TC-009 without existing goals the add form keeps the free-text field", async ({ page, context }) => {
    const api = await mockGoalsApi(context);
    await page.goto("/");
    const dialog = await openAddForm(page);
    const field = dialog.getByRole("textbox", { name: "Purchase goal (optional)" });
    await expect(field).toBeVisible();
    await expect(field).toHaveValue("");
    expect(await field.evaluate((element) => element.tagName)).toBe("TEXTAREA");
    await expect(addChoice(dialog)).toHaveCount(0);
    await field.fill("Gaming");
    await submitAddForm(dialog);
    expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: "Gaming" }]);
  });

  test("TC-009 nothing typed without existing goals sends no goal; the reopened form has no drop-down", async ({ page, context }) => {
    const api = await mockGoalsApi(context);
    await page.goto("/");
    let dialog = await openAddForm(page);
    await expect(dialog.getByRole("textbox", { name: "Purchase goal (optional)" })).toBeVisible();
    await submitAddForm(dialog);
    expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: "" }]);
    dialog = await openAddForm(page, otherLbcUrl);
    await expect(dialog.getByRole("textbox", { name: "Purchase goal (optional)" })).toBeVisible();
    await expect(addChoice(dialog)).toHaveCount(0);
  });

  test("TC-010 while the goals load, the add form shows a loading indicator and adds without a goal", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { otherGoals: ["Gaming"], goalsReply: "hold" });
    await page.goto("/");
    const dialog = await openAddForm(page);
    await expect(dialog.getByText("Loading your purchase goals…")).toBeVisible();
    await expect(addChoice(dialog)).toHaveCount(0);
    await expect(dialog.getByRole("button", { name: "Add item", exact: true })).toBeEnabled();
    await submitAddForm(dialog);
    expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: "" }]);
  });

  for (const reply of ["fail", "abort"] as const) {
    test(`TC-010 goals not loadable (${reply}): the add form offers the free-text field`, async ({ page, context }) => {
      const api = await mockGoalsApi(context, { otherGoals: ["Gaming"], goalsReply: reply });
      await page.goto("/");
      const dialog = await openAddForm(page);
      const field = dialog.getByRole("textbox", { name: "Purchase goal (optional)" });
      await expect(field).toBeVisible();
      await expect(addChoice(dialog)).toHaveCount(0);
      await field.fill("Gaming");
      await submitAddForm(dialog);
      expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: "Gaming" }]);
    });
  }

  test("TC-012 keyboard only: choose a goal, adapt it and add the item", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { otherGoals: ["Gaming", "install a light Linux distro for my parents"] });
    await page.goto("/");
    const dialog = await openAddForm(page);
    const choice = addChoice(dialog);
    await expect(choice).toBeVisible();
    await expect(dialog.getByText("Purchase goal (optional)", { exact: true })).toBeVisible();
    await dialog.getByRole("textbox", { name: "Listing URL" }).focus();
    await page.keyboard.press("Tab");
    await expect(choice).toBeFocused();
    await highlightWithArrows(page, choice, "Gaming");
    await page.keyboard.press("Enter");
    await expect(page.getByRole("option")).toHaveCount(0);
    await expect(selectedValue(choice)).toHaveText("Gaming");
    // The selected option is exposed to assistive technologies.
    await page.keyboard.press("ArrowDown");
    await expect(page.getByRole("option", { name: "Gaming", exact: true, selected: true })).toBeVisible();
    await expect(page.getByRole("option", { selected: true })).toHaveCount(1);
    // Escape closes the options without changing the selection.
    await page.keyboard.press("Escape");
    await expect(page.getByRole("option")).toHaveCount(0);
    await expect(selectedValue(choice)).toHaveText("Gaming");
    await expect(choice).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(goalText(dialog)).toBeFocused();
    await expect(goalText(dialog)).toHaveValue("Gaming");
    await expect(dialog.getByText("Goal text", { exact: true })).toBeVisible();
    await page.keyboard.press("ControlOrMeta+End");
    await page.keyboard.type(" on Linux");
    await tabTo(page, dialog.getByRole("button", { name: "Add item", exact: true }));
    await page.keyboard.press("Enter");
    await expect(dialog).toBeHidden();
    expect(api.posts).toEqual([{ url: lbcUrl, purchaseGoal: "Gaming on Linux" }]);
  });

  test("TC-012 narrow screen: the add form goal choice fits 320 px", async ({ page, context }) => {
    await page.setViewportSize({ width: 320, height: 640 });
    await mockGoalsApi(context, { otherGoals: ["Gaming", "install a light Linux distro for my parents"] });
    await page.goto("/");
    const dialog = await openAddForm(page);
    await choose(page, addChoice(dialog), "install a light Linux…");
    await expectFitsNarrowScreen(page, addChoice(dialog), goalText(dialog));
  });

  test("TC-013 an Amazon URL offers the goal choice and adds the item without a goal", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { otherGoals: ["Gaming"] });
    await page.goto("/");
    const dialog = await openAddForm(page, amazonUrl);
    await choose(page, addChoice(dialog), "Gaming");
    await expect(goalText(dialog)).toHaveValue("Gaming");
    await submitAddForm(dialog);
    expect(api.posts).toHaveLength(1);
    await expect(page.getByText("Amazon lamp").first()).toBeVisible();
    await page.goto(`/items/${api.items[0].id}`);
    await expect(page.getByRole("heading", { name: "Amazon lamp" })).toBeVisible();
    await expect(page.getByRole("combobox", { name: "Purchase goal" })).toHaveCount(0);
    await expect(page.getByRole("textbox", { name: "Purchase goal" })).toHaveCount(0);
    await expect(goalText(page)).toHaveCount(0);
    await expect(saveGoal(page)).toHaveCount(0);
  });

  test("TC-013 an Amazon URL with a new goal is added normally", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { otherGoals: ["Gaming"] });
    await page.goto("/");
    const dialog = await openAddForm(page, amazonUrl);
    await choose(page, addChoice(dialog), "New goal");
    await goalText(dialog).fill("Brand new text");
    await submitAddForm(dialog);
    expect(api.posts).toHaveLength(1);
    expect(api.items[0]).toMatchObject({ platform: "amazon", goal: "" });
    await expect(page.getByText("Amazon lamp").first()).toBeVisible();
  });
});

test.describe("issue #304 details page", () => {
  test("TC-009 the first goal saved on a page without existing goals shows the drop-down", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { items: [itemA("")] });
    await page.goto(`/items/${baseItem.id}`);
    const field = page.getByRole("textbox", { name: "Purchase goal" });
    await expect(field).toHaveValue("");
    await expect(saveGoal(page)).toBeDisabled();
    await expect(detailChoice(page)).toHaveCount(0);
    await field.fill("Gaming");
    await saveGoal(page).click();
    await expect(page.getByText("Purchase goal saved. A new AI review was requested.")).toBeVisible();
    await expect(selectedValue(detailChoice(page))).toHaveText("Gaming");
    await expect(goalText(page)).toHaveValue("Gaming");
    await expect(saveGoal(page)).toBeDisabled();
    expect(api.puts).toEqual([{ id: baseItem.id, purchaseGoal: "Gaming" }]);
  });

  test("TC-009 a failed first save keeps the text and shows no drop-down", async ({ page, context }) => {
    await mockGoalsApi(context, { items: [itemA("")], failSave: true });
    await page.goto(`/items/${baseItem.id}`);
    const field = page.getByRole("textbox", { name: "Purchase goal" });
    await field.fill("Gaming");
    await saveGoal(page).click();
    await expect(page.getByText("Could not save purchase goal")).toBeVisible();
    await expect(field).toHaveValue("Gaming");
    await expect(detailChoice(page)).toHaveCount(0);
  });

  test("TC-010 details page: a loading indicator, then the drop-down with the item's goal", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { items: [itemA("Gaming")], otherGoals: ["Office"], goalsReply: "hold" });
    await page.goto(`/items/${baseItem.id}`);
    await expect(page.getByText("Loading your purchase goals…").filter({ visible: true })).toBeVisible();
    await expect(saveGoal(page)).toHaveCount(0);
    await expect(detailChoice(page)).toHaveCount(0);
    await api.releaseGoals();
    await expect(selectedValue(detailChoice(page))).toHaveText("Gaming");
    await expect(goalText(page)).toHaveValue("Gaming");
  });

  test("TC-010 details page with goals not loadable keeps the free-text goal and saves it", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { items: [itemA("Gaming")], otherGoals: ["Office"], goalsReply: "fail" });
    await page.goto(`/items/${baseItem.id}`);
    const field = page.getByRole("textbox", { name: "Purchase goal" });
    await expect(field).toHaveValue("Gaming");
    await expect(detailChoice(page)).toHaveCount(0);
    await expect(saveGoal(page)).toBeDisabled();
    await field.fill("Office");
    await saveGoal(page).click();
    await expect(page.getByText("Purchase goal saved. A new AI review was requested.")).toBeVisible();
    expect(api.puts).toEqual([{ id: baseItem.id, purchaseGoal: "Office" }]);
  });

  test("TC-011 select another existing goal, save and reload", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { items: [itemA("Gaming")], otherGoals: ["Office"] });
    await page.goto(`/items/${baseItem.id}`);
    await expect(selectedValue(detailChoice(page))).toHaveText("Gaming");
    await expect(goalText(page)).toHaveValue("Gaming");
    await expect(saveGoal(page)).toBeDisabled();
    await expectOptions(page, detailChoice(page), ["No goal", "Gaming", "Office", "New goal"]);
    await choose(page, detailChoice(page), "Office");
    await expect(goalText(page)).toHaveValue("Office");
    await expect(saveGoal(page)).toBeEnabled();
    await saveGoal(page).click();
    await expect(page.getByText("Purchase goal saved. A new AI review was requested.")).toBeVisible();
    await expect(page.getByText("AI review in progress…")).toBeVisible();
    expect(api.puts).toEqual([{ id: baseItem.id, purchaseGoal: "Office" }]);
    api.running = false;
    await page.reload();
    await expect(selectedValue(detailChoice(page))).toHaveText("Office");
    await expect(goalText(page)).toHaveValue("Office");
    expect(api.puts).toEqual([{ id: baseItem.id, purchaseGoal: "Office" }]);
  });

  test("TC-011 No goal hides the field and saving clears the goal", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { items: [itemA("Gaming")], otherGoals: ["Office"] });
    await page.goto(`/items/${baseItem.id}`);
    await choose(page, detailChoice(page), "No goal");
    await expect(goalText(page)).toHaveCount(0);
    await expect(saveGoal(page)).toBeEnabled();
    await saveGoal(page).click();
    await expect(page.getByText(/^Purchase goal saved\./)).toBeVisible();
    expect(api.puts).toEqual([{ id: baseItem.id, purchaseGoal: "" }]);
    api.running = false;
    await page.reload();
    await expect(selectedValue(detailChoice(page))).toHaveText("No goal");
    await expect(goalText(page)).toHaveCount(0);
  });

  test("TC-011 a New goal typed and saved is offered and selected after reload", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { items: [itemA("Gaming")], otherGoals: ["Office"] });
    await page.goto(`/items/${baseItem.id}`);
    await choose(page, detailChoice(page), "New goal");
    await expect(goalText(page)).toHaveValue("");
    await goalText(page).fill("photo editing");
    await saveGoal(page).click();
    await expect(page.getByText(/^Purchase goal saved\./)).toBeVisible();
    expect(api.puts).toEqual([{ id: baseItem.id, purchaseGoal: "photo editing" }]);
    api.running = false;
    await page.reload();
    await expect(selectedValue(detailChoice(page))).toHaveText("photo editing");
    await expectOptions(page, detailChoice(page), ["No goal", "Office", "photo editing", "New goal"]);
  });

  test("TC-011 New goal with no text clears the goal", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { items: [itemA("Gaming")], otherGoals: ["Office"] });
    await page.goto(`/items/${baseItem.id}`);
    await choose(page, detailChoice(page), "New goal");
    await expect(saveGoal(page)).toBeEnabled();
    await saveGoal(page).click();
    await expect(page.getByText(/^Purchase goal saved\./)).toBeVisible();
    expect(api.puts).toEqual([{ id: baseItem.id, purchaseGoal: "" }]);
  });

  test("TC-011 modified goal text keeps the selection and is saved", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { items: [itemA("Gaming")], otherGoals: ["Office"] });
    await page.goto(`/items/${baseItem.id}`);
    await goalText(page).fill("Gaming on Linux");
    await expect(selectedValue(detailChoice(page))).toHaveText("Gaming");
    await expect(saveGoal(page)).toBeEnabled();
    await saveGoal(page).click();
    await expect(page.getByText(/^Purchase goal saved\./)).toBeVisible();
    expect(api.puts).toEqual([{ id: baseItem.id, purchaseGoal: "Gaming on Linux" }]);
  });

  test("TC-011 back to the current goal disables Save goal", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { items: [itemA("Gaming")], otherGoals: ["Office"] });
    await page.goto(`/items/${baseItem.id}`);
    await choose(page, detailChoice(page), "Office");
    await expect(saveGoal(page)).toBeEnabled();
    await choose(page, detailChoice(page), "Gaming");
    await expect(goalText(page)).toHaveValue("Gaming");
    await expect(saveGoal(page)).toBeDisabled();
    await goalText(page).fill("Gaming x");
    await expect(saveGoal(page)).toBeEnabled();
    await goalText(page).fill("Gaming");
    await expect(saveGoal(page)).toBeDisabled();
    await goalText(page).fill("Gaming ");
    await expect(saveGoal(page)).toBeDisabled();
    expect(api.puts).toEqual([]);
  });

  test("TC-011 an item without goal and existing goals: No goal selected, Save goal disabled", async ({ page, context }) => {
    await mockGoalsApi(context, { items: [itemA("")], otherGoals: ["Office"] });
    await page.goto(`/items/${baseItem.id}`);
    await expect(selectedValue(detailChoice(page))).toHaveText("No goal");
    await expect(goalText(page)).toHaveCount(0);
    await expect(saveGoal(page)).toBeDisabled();
  });

  test("TC-011 a failed save keeps the selection and the text; the stored goal is unchanged", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { items: [itemA("Gaming")], otherGoals: ["Office"], failSave: true });
    await page.goto(`/items/${baseItem.id}`);
    await choose(page, detailChoice(page), "Office");
    await saveGoal(page).click();
    await expect(page.getByText("Could not save purchase goal")).toBeVisible();
    await expect(selectedValue(detailChoice(page))).toHaveText("Office");
    await expect(goalText(page)).toHaveValue("Office");
    expect(api.puts).toEqual([{ id: baseItem.id, purchaseGoal: "Office" }]);
    await page.reload();
    await expect(selectedValue(detailChoice(page))).toHaveText("Gaming");
  });

  test("TC-011 a long goal reads four words in the drop-down and in full in the field", async ({ page, context }) => {
    await mockGoalsApi(context, { items: [itemA("install a light Linux distro for my parents")], otherGoals: ["Office"] });
    await page.goto(`/items/${baseItem.id}`);
    await expect(selectedValue(detailChoice(page))).toHaveText("install a light Linux…");
    await expect(goalText(page)).toHaveValue("install a light Linux distro for my parents");
  });

  test("TC-012 keyboard only on the details page: select a goal and save it", async ({ page, context }) => {
    const api = await mockGoalsApi(context, { items: [itemA("Gaming")], otherGoals: ["install a light Linux distro for my parents"] });
    await page.goto(`/items/${baseItem.id}`);
    const choice = detailChoice(page);
    await expect(selectedValue(choice)).toHaveText("Gaming");
    await expect(page.getByText("Purchase goal", { exact: true })).toBeVisible();
    await expect(page.getByText("Goal text", { exact: true })).toBeVisible();
    await choice.focus();
    await highlightWithArrows(page, choice, "install a light Linux…");
    await page.keyboard.press("Enter");
    await expect(selectedValue(choice)).toHaveText("install a light Linux…");
    await expect(goalText(page)).toHaveValue("install a light Linux distro for my parents");
    await tabTo(page, saveGoal(page));
    await page.keyboard.press("Enter");
    await expect(page.getByRole("status").filter({ hasText: "Purchase goal saved" })).toBeVisible();
    expect(api.puts).toEqual([{ id: baseItem.id, purchaseGoal: "install a light Linux distro for my parents" }]);
  });

  test("TC-012 narrow screen: the details goal choice fits 320 px", async ({ page, context }) => {
    await page.setViewportSize({ width: 320, height: 640 });
    await mockGoalsApi(context, { items: [itemA("install a light Linux distro for my parents")], otherGoals: ["Gaming"] });
    await page.goto(`/items/${baseItem.id}`);
    await expect(selectedValue(detailChoice(page))).toHaveText("install a light Linux…");
    await expectFitsNarrowScreen(page, detailChoice(page), goalText(page));
  });

  // Item sharing (FR-SHARE-004) is not in the API or the UI yet, so no shared item can be shown
  // (TS-GOAL-CHOICE-008). TC-001 proves that goals of other users' items are never offered.
  test.fixme("TC-014 a shared item's goal is read-only, without drop-down or Save goal", async () => {});
});
