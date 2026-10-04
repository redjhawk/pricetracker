import { expect, test, type BrowserContext, type Page } from "@playwright/test";

// Synthetic session values only; never real cookies.
const timestamp = "2026-10-03T08:00:00Z";
const item = {
  id: "leboncoin-one", title: "LeBoncoin bicycle", platform: "leboncoin",
  listingId: "1234567890", asin: null, marketplace: "leboncoin.fr",
  url: "https://www.leboncoin.fr/ad/velos/1234567890", thumbnailUrl: null,
  status: "active", latestPrice: { amountCents: 1299, currency: "EUR", timestamp },
  lastThreeDetections: [{ amountCents: 1299, currency: "EUR", timestamp }],
  secondHandOffer: null, lastAttempt: { result: "success", timestamp, message: null },
  nextCheckAt: null, addedAt: timestamp, purchaseGoal: "",
};
const noSession = {
  value: null as string | null, revision: 0, updatedAt: null as string | null,
  status: "none" as string, expiresAt: null as string | null, revokedAt: null as string | null,
  lastAttempt: null as { outcome: string; attemptedAt: string } | null,
};
const activeSession = { ...noSession, value: "Synthetic~Value_123", revision: 3, updatedAt: timestamp, status: "active" };
type Session = typeof noSession;
type PutReply = { status: number; json: unknown } | "echo";

const settingsPath = "/api/v1/settings/leboncoin-session";
const claudeTokenPath = "/api/v1/settings/claude-token";
const noClaudeToken = { value: null, updatedAt: null, lastRejectedAt: null };
const unexpectedRequests = new WeakMap<BrowserContext, string[]>();

test.afterEach(async ({ context }) => {
  expect(unexpectedRequests.get(context)).toEqual([]);
});

function extracted(value: string) {
  const match = value.match(/datadome=([^;\s]+)/);
  return match ? match[1] : value.trim();
}

async function mockApi(context: BrowserContext, session: Session = activeSession) {
  const state = {
    session: structuredClone(session),
    getError: false,
    getGate: Promise.resolve(),
    putGate: Promise.resolve(),
    putReplies: [] as PutReply[],
    listGate: Promise.resolve(),
    listError: false,
    gets: 0,
    puts: [] as unknown[],
    unexpected: [] as string[],
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
    const method = request.method();
    const path = url.pathname + url.search;
    if (method === "GET" && path === "/api/v1/items") {
      await state.listGate;
      await route.fulfill(state.listError
        ? { status: 500, json: { error: { code: "TEST_FAILURE", message: "Fixture retrieval failed." } } }
        : { json: { items: [item] } });
    } else if (method === "GET" && path === `/api/v1/items/${item.id}`) {
      await route.fulfill({ json: { ...item, priceHistory: item.lastThreeDetections } });
    } else if (method === "GET" && path === settingsPath) {
      state.gets += 1;
      await state.getGate;
      await route.fulfill(state.getError
        ? { status: 500, json: { error: { code: "INTERNAL_ERROR", message: "The server could not complete the request." } } }
        : { json: { session: state.session } });
    } else if (method === "GET" && path === claudeTokenPath) {
      await state.getGate;
      await route.fulfill(state.getError
        ? { status: 500, json: { error: { code: "INTERNAL_ERROR", message: "The server could not complete the request." } } }
        : { json: { claudeToken: noClaudeToken } });
    } else if (method === "PUT" && path === "/api/v1/settings") {
      const body = (request.postDataJSON() as { leboncoinSession: { value: string; revision: number } }).leboncoinSession;
      state.puts.push(body);
      await state.putGate;
      const reply = state.putReplies.shift() ?? "echo";
      if (reply === "echo") {
        const value = extracted(body.value);
        state.session = value
          ? { ...noSession, value, revision: state.session.revision + 1, updatedAt: timestamp, status: "active" }
          : { ...noSession, revision: state.session.revision + 1, updatedAt: timestamp };
        await route.fulfill({ json: { session: state.session, claudeToken: noClaudeToken } });
      } else {
        await route.fulfill(reply);
      }
    } else {
      state.unexpected.push(`${method} ${path}`);
      await route.abort();
    }
  });
  return state;
}

function menuTrigger(page: Page) {
  return page.getByRole("button", { name: "Application menu", exact: true });
}

async function openSettings(page: Page) {
  await menuTrigger(page).click();
  await page.getByRole("menuitem", { name: "Settings", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Settings" });
  await expect(dialog).toBeVisible();
  return dialog;
}

function sessionField(page: Page) {
  return page.getByRole("dialog", { name: "Settings" }).getByRole("textbox", { name: "LeBonCoin session" });
}

function saveButton(page: Page) {
  return page.getByRole("dialog", { name: "Settings" }).getByRole("button", { name: /^Sav(e|ing…)$/ });
}

test("header menu: visible on every page, one Settings entry, keyboard and dismissal", async ({ page, context }) => {
  const api = await mockApi(context);
  let release!: () => void;
  api.listGate = new Promise<void>((resolve) => { release = resolve; });
  await page.goto("/");
  await expect(page.getByText("Loading tracked items…", { exact: true })).toBeVisible();
  await expect(menuTrigger(page)).toBeVisible();
  api.listError = true;
  release();
  await expect(page.getByText("Could not load tracked items", { exact: true })).toBeVisible();
  await expect(menuTrigger(page)).toBeVisible();
  await expect(page.getByRole("banner")).not.toContainText(/sign|user|profile|account/i);

  const trigger = menuTrigger(page);
  await expect(trigger).toHaveAttribute("aria-expanded", "false");
  await trigger.click();
  await expect(trigger).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByRole("menuitem")).toHaveCount(1);
  await expect(page.getByRole("menuitem")).toHaveText("Settings");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menuitem")).toHaveCount(0);
  await expect(trigger).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("menuitem", { name: "Settings" })).toBeVisible();
  await page.keyboard.press("Escape");
  await page.keyboard.press("Space");
  await expect(page.getByRole("menuitem", { name: "Settings" })).toBeVisible();
  await page.mouse.click(10, 300);
  await expect(page.getByRole("menuitem")).toHaveCount(0);
  await trigger.click();
  await trigger.click();
  await expect(page.getByRole("menuitem")).toHaveCount(0);
  expect(api.gets).toBe(0);

  api.listError = false;
  await page.goto(`/items/${item.id}`);
  await expect(page.getByRole("heading", { name: "LeBoncoin bicycle" })).toBeVisible();
  await expect(menuTrigger(page)).toBeVisible();
  await menuTrigger(page).focus();
  await page.keyboard.press("Enter");
  await page.keyboard.press("Enter");
  await expect(page.getByRole("dialog", { name: "Settings" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "LeBoncoin bicycle" })).toBeAttached();
  await expect(page).toHaveURL(`/items/${item.id}`);
});

test("settings modal: loading, full value, help text, session and Claude token fields", async ({ page, context }) => {
  const api = await mockApi(context);
  let release!: () => void;
  api.getGate = new Promise<void>((resolve) => { release = resolve; });
  await page.goto("/");
  const dialog = await openSettings(page);
  await expect(dialog.getByText("Loading settings…")).toBeVisible();
  await expect(saveButton(page)).toBeDisabled();
  await expect(sessionField(page)).toBeDisabled();
  await expect(sessionField(page)).toHaveValue("");
  release();
  await expect(sessionField(page)).toHaveValue("Synthetic~Value_123");
  await expect(sessionField(page)).toBeEnabled();
  await expect(dialog.getByRole("textbox")).toHaveCount(2);
  await expect(dialog).toContainText("datadome=");
  await expect(dialog).toContainText(/empty field to remove the session/);
  await expect(saveButton(page)).toBeEnabled();
  await expect(dialog.getByText(/rejected|failed|expired|revoked/)).toHaveCount(0);
});

test("load failure disables saving; cancel closes and reopening retries", async ({ page, context }) => {
  const api = await mockApi(context);
  api.getError = true;
  await page.goto("/");
  const dialog = await openSettings(page);
  await expect(dialog.getByText("Could not load settings")).toBeVisible();
  await expect(dialog).toContainText("The server could not complete the request.");
  await expect(dialog.getByRole("textbox")).toHaveCount(0);
  await expect(saveButton(page)).toBeDisabled();
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(dialog).toBeHidden();
  await expect(menuTrigger(page)).toBeFocused();
  api.getError = false;
  await openSettings(page);
  await expect(sessionField(page)).toHaveValue("Synthetic~Value_123");
  expect(api.gets).toBe(2);
  expect(api.puts).toEqual([]);
});

test("saving sends the text and revision, closes, and reopening reloads", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  await openSettings(page);
  await sessionField(page).fill("Cookie: a=1; datadome=New~Synthetic_456; b=2");
  await saveButton(page).click();
  await expect(page.getByRole("dialog", { name: "Settings" })).toBeHidden();
  expect(api.puts).toEqual([{ value: "Cookie: a=1; datadome=New~Synthetic_456; b=2", revision: 3 }]);
  await expect(menuTrigger(page)).toBeFocused();
  await openSettings(page);
  await expect(sessionField(page)).toHaveValue("New~Synthetic_456");
  expect(api.gets).toBe(2);
});

test("saving an empty field clears the session", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  await openSettings(page);
  await sessionField(page).fill("");
  await saveButton(page).click();
  await expect(page.getByRole("dialog", { name: "Settings" })).toBeHidden();
  expect(api.puts).toEqual([{ value: "", revision: 3 }]);
  await openSettings(page);
  await expect(sessionField(page)).toHaveValue("");
});

test("invalid session shows the server message on the field and keeps input", async ({ page, context }) => {
  const api = await mockApi(context);
  api.putReplies.push({ status: 400, json: { error: { code: "INVALID_SESSION", message: "No datadome cookie was found in the pasted text." } } });
  await page.goto("/");
  await openSettings(page);
  await sessionField(page).fill("a=1; b=2");
  await saveButton(page).click();
  await expect(sessionField(page)).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByRole("dialog", { name: "Settings" }).getByText("No datadome cookie was found in the pasted text.")).toBeVisible();
  await expect(sessionField(page)).toHaveValue("a=1; b=2");
  await sessionField(page).fill("a=1; b=2; datadome=Fixed~Synthetic");
  await expect(sessionField(page)).not.toHaveAttribute("aria-invalid", "true");
  await saveButton(page).click();
  await expect(page.getByRole("dialog", { name: "Settings" })).toBeHidden();
  expect(api.puts).toHaveLength(2);
});

test("server failure keeps input and allows retry", async ({ page, context }) => {
  const api = await mockApi(context);
  api.putReplies.push({ status: 500, json: { error: { code: "INTERNAL_ERROR", message: "The server could not complete the request." } } });
  await page.goto("/");
  const dialog = await openSettings(page);
  await sessionField(page).fill("Retry~Synthetic");
  await saveButton(page).click();
  await expect(dialog.getByText("Could not save settings")).toBeVisible();
  await expect(sessionField(page)).toHaveValue("Retry~Synthetic");
  await expect(saveButton(page)).toBeEnabled();
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([{ value: "Retry~Synthetic", revision: 3 }, { value: "Retry~Synthetic", revision: 3 }]);
});

test("changed session warns, keeps input, blocks save until reopened", async ({ page, context }) => {
  const api = await mockApi(context);
  api.putReplies.push({ status: 409, json: { error: { code: "SESSION_CHANGED", message: "The LeBoncoin session changed after Settings was opened. Reopen Settings before saving." } } });
  await page.goto("/");
  const dialog = await openSettings(page);
  await sessionField(page).fill("Mine~Synthetic");
  await saveButton(page).click();
  await expect(dialog.getByText("Session changed", { exact: true })).toBeVisible();
  await expect(dialog).toContainText("close and reopen Settings before saving");
  await expect(sessionField(page)).toHaveValue("Mine~Synthetic");
  await expect(sessionField(page)).toBeEditable();
  await expect(saveButton(page)).toBeDisabled();
  api.session = { ...activeSession, value: "Renewed~Synthetic", revision: 4 };
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await openSettings(page);
  await expect(sessionField(page)).toHaveValue("Renewed~Synthetic");
  await expect(dialog.getByText("Session changed", { exact: true })).toHaveCount(0);
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([{ value: "Mine~Synthetic", revision: 3 }, { value: "Renewed~Synthetic", revision: 4 }]);
});

test("cancel and Escape discard edits without saving", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  let dialog = await openSettings(page);
  await sessionField(page).fill("Discarded~One");
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(dialog).toBeHidden();
  dialog = await openSettings(page);
  await expect(sessionField(page)).toHaveValue("Synthetic~Value_123");
  await sessionField(page).fill("Discarded~Two");
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(menuTrigger(page)).toBeFocused();
  await openSettings(page);
  await expect(sessionField(page)).toHaveValue("Synthetic~Value_123");
  expect(api.puts).toEqual([]);
});

test("saving in progress blocks closing and duplicate saves", async ({ page, context }) => {
  const api = await mockApi(context);
  let release!: () => void;
  api.putGate = new Promise<void>((resolve) => { release = resolve; });
  await page.goto("/");
  const dialog = await openSettings(page);
  await sessionField(page).fill("Pending~Synthetic");
  await saveButton(page).click();
  await expect(saveButton(page)).toHaveText("Saving…");
  await expect(saveButton(page)).toBeDisabled();
  await expect(sessionField(page)).toBeDisabled();
  await saveButton(page).click({ force: true });
  await page.keyboard.press("Escape");
  await dialog.getByRole("button", { name: "Cancel" }).click({ force: true });
  await expect(dialog).toBeVisible();
  release();
  await expect(dialog).toBeHidden();
  expect(api.puts).toHaveLength(1);
});

const hintCases: { name: string; session: Session; text: RegExp | null }[] = [
  { name: "rejected", session: { ...activeSession, lastAttempt: { outcome: "rejected", attemptedAt: "2026-10-03T20:00:00Z" } },
    text: /LeBoncoin rejected this session on 03 Oct 2026, 20:00 UTC\. Capture a new session and save it here\./ },
  { name: "failed", session: { ...activeSession, lastAttempt: { outcome: "failed", attemptedAt: "2026-10-03T20:00:00Z" } },
    text: /The last LeBoncoin check using this session failed on 03 Oct 2026, 20:00 UTC for a reason other than a rejection\./ },
  { name: "expired", session: { ...activeSession, status: "expired", expiresAt: "2026-10-02T10:00:00Z",
    lastAttempt: { outcome: "rejected", attemptedAt: "2026-10-03T20:00:00Z" } },
    text: /This session expired on 02 Oct 2026, 10:00 UTC\. It is no longer used\./ },
  { name: "revoked", session: { ...activeSession, status: "revoked", revokedAt: "2026-10-03T21:00:00Z" },
    text: /LeBoncoin revoked this session on 03 Oct 2026, 21:00 UTC\. It is no longer used\./ },
  { name: "accepted", session: { ...activeSession, lastAttempt: { outcome: "accepted", attemptedAt: timestamp } }, text: null },
  { name: "unused", session: activeSession, text: null },
  { name: "none", session: noSession, text: null },
];

for (const { name, session, text } of hintCases) {
  test(`hint for ${name} session`, async ({ page, context }) => {
    await mockApi(context, session);
    await page.goto("/");
    const dialog = await openSettings(page);
    await expect(sessionField(page)).toHaveValue(session.value ?? "");
    if (text) {
      await expect(dialog.getByText(text)).toBeVisible();
      await expect(dialog.getByText(/rejected|failed|expired|revoked/)).toHaveCount(1);
    } else {
      await expect(dialog.getByText(/rejected|failed|expired|revoked/)).toHaveCount(0);
    }
  });
}

test("menu cannot be used while another modal is open", async ({ page, context }) => {
  await mockApi(context);
  await page.goto("/");
  const box = await menuTrigger(page).boundingBox();
  await page.getByRole("button", { name: "Add item", exact: true }).first().click();
  await expect(page.getByRole("dialog", { name: "Add tracked item" })).toBeVisible();
  await page.mouse.click(box!.x + box!.width / 2, box!.y + box!.height / 2);
  await expect(page.getByRole("menuitem", { name: "Settings" })).toHaveCount(0);
  await expect(page.getByRole("dialog", { name: "Settings" })).toHaveCount(0);
});

test("narrow viewport keeps the menu and modal usable without horizontal scroll", async ({ page, context }) => {
  await mockApi(context, { ...activeSession, value: "Long~Synthetic_".repeat(40) });
  await page.setViewportSize({ width: 400, height: 800 });
  await page.goto("/");
  const trigger = menuTrigger(page);
  await expect(trigger).toBeVisible();
  const triggerBox = (await trigger.boundingBox())!;
  for (const other of [page.getByRole("link", { name: /Price follower/ }), page.getByRole("button", { name: "Add item", exact: true }).first()]) {
    const box = (await other.boundingBox())!;
    expect(box.x + box.width <= triggerBox.x || triggerBox.x + triggerBox.width <= box.x).toBe(true);
  }
  expect(triggerBox.x + triggerBox.width).toBeLessThanOrEqual(400);
  await trigger.click();
  const menuBox = (await page.getByRole("menu").boundingBox())!;
  expect(menuBox.x).toBeGreaterThanOrEqual(0);
  expect(menuBox.x + menuBox.width).toBeLessThanOrEqual(400);
  await page.getByRole("menuitem", { name: "Settings" }).click();
  await expect(sessionField(page)).toHaveValue("Long~Synthetic_".repeat(40));
  await expect(saveButton(page)).toBeInViewport();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});

test("session value is not kept in browser storage", async ({ page, context }) => {
  await mockApi(context);
  await page.goto("/");
  await openSettings(page);
  await expect(sessionField(page)).toHaveValue("Synthetic~Value_123");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "Settings" })).toBeHidden();
  const stored = await page.evaluate(() => JSON.stringify({ ...localStorage }) + JSON.stringify({ ...sessionStorage }));
  expect(stored).not.toContain("Synthetic~Value_123");
  expect(await page.getByText("Synthetic~Value_123").count()).toBe(0);
});

// Focus may move asynchronously (Carbon initial focus), so each assertion waits for it to settle.
async function expectFocusInDialog(page: Page) {
  await expect.poll(() => page.evaluate(() => Boolean(document.activeElement?.closest("[role=dialog]")))).toBe(true);
}

async function expectTabbingStaysInDialog(page: Page) {
  for (const key of ["Tab", "Tab", "Tab", "Tab", "Tab", "Shift+Tab", "Shift+Tab", "Shift+Tab", "Shift+Tab", "Shift+Tab", "Shift+Tab"]) {
    await page.keyboard.press(key);
    await expectFocusInDialog(page);
  }
}

test("QA-SET-F01: focus enters and stays in the dialog while loading, loaded and on error", async ({ page, context }) => {
  const api = await mockApi(context);
  let release!: () => void;
  api.getGate = new Promise<void>((resolve) => { release = resolve; });
  await page.goto("/");
  await openSettings(page);
  await expect(page.getByText("Loading settings…")).toBeVisible();
  await expectFocusInDialog(page);
  await expectTabbingStaysInDialog(page);
  release();
  await expect(sessionField(page)).toBeEnabled();
  await expectFocusInDialog(page);
  await expectTabbingStaysInDialog(page);
  await page.keyboard.press("Escape");
  await expect(menuTrigger(page)).toBeFocused();
  api.getError = true;
  await menuTrigger(page).press("Enter");
  await page.keyboard.press("Enter");
  await expect(page.getByText("Could not load settings")).toBeVisible();
  await expectFocusInDialog(page);
  await expectTabbingStaysInDialog(page);
});

test("QA-SET-F02: header controls are unreachable behind Settings; no stacked modal", async ({ page, context }) => {
  await mockApi(context);
  await page.goto("/");
  await openSettings(page);
  await expect(sessionField(page)).toBeEnabled();
  for (let i = 0; i < 6; i += 1) {
    await page.keyboard.press("Tab");
    await expectFocusInDialog(page);
    expect(await page.evaluate(() => Boolean(document.activeElement?.closest("header")))).toBe(false);
    if (await page.evaluate(() => document.activeElement?.tagName !== "BUTTON")) continue;
    if (await page.evaluate(() => /Save|Cancel|Close/i.test(document.activeElement?.textContent + (document.activeElement?.getAttribute("aria-label") ?? "")))) continue;
    await page.keyboard.press("Enter");
  }
  await expect(page.locator("[role=dialog]:visible")).toHaveCount(1);
});
