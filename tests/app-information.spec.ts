import { expect, test, type BrowserContext, type Locator, type Page } from "@playwright/test";

// FR-APP-INFO-001..014. Synthetic session data only.
type SessionReply = { status: number; json: unknown; delayMs?: number; gate?: Promise<void> };

// The app calls the session endpoint at startup (twice in development, under React StrictMode).
// Tests that need later replies load the page first, then set state.replies and reset state.sessionCalls.
async function mockApi(context: BrowserContext, replies: SessionReply[]) {
  const state = { sessionCalls: 0, replies };
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== "http://127.0.0.1:4173") return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    if (url.pathname === "/api/v1/auth/session") {
      const reply = state.replies[Math.min(state.sessionCalls, state.replies.length - 1)];
      state.sessionCalls += 1;
      if (reply.delayMs) await new Promise((resolve) => setTimeout(resolve, reply.delayMs));
      if (reply.gate) await reply.gate;
      return route.fulfill({ status: reply.status, json: reply.json });
    }
    if (url.pathname === "/api/v1/items") return route.fulfill({ json: { items: [] } });
    if (url.pathname === "/api/v1/amazon-searches") return route.fulfill({ json: { searches: [], amazonRequests: [] } });
    if (url.pathname === "/api/v1/admin/users") return route.fulfill({ json: { users: [] } });
    // Issue #225: the Settings modal loads both secrets' state.
    if (url.pathname === "/api/v1/settings/leboncoin-session") {
      return route.fulfill({ json: { session: { saved: false, hint: null, revision: 0, updatedAt: null, status: "none", expiresAt: null, revokedAt: null, lastAttempt: null } } });
    }
    if (url.pathname === "/api/v1/settings/claude-token") {
      return route.fulfill({ json: { claudeToken: { saved: false, hint: null, updatedAt: null, lastRejectedAt: null } } });
    }
    return route.fulfill({ status: 500, json: { error: { code: "UNEXPECTED", message: url.pathname } } });
  });
  return state;
}

const version = "v20261109200945";
const openSession = { status: 200, json: { mode: "open", user: null, version } };
const userSession = { status: 200, json: { mode: "protected", user: { username: "alice", role: "user" }, version } };
const adminSession = { status: 200, json: { mode: "protected", user: { username: "admin", role: "admin" }, version } };
const failure = { status: 500, json: { error: { code: "INTERNAL", message: "boom" } } };

async function openInformation(page: Page) {
  await page.getByRole("button", { name: "Application menu", exact: true }).click();
  await page.getByRole("menuitem", { name: "Information", exact: true }).click();
  return page.getByRole("dialog", { name: "Information" });
}

test("regular user sees Information with Settings and Log out, and the modal shows username and version", async ({ page, context }) => {
  await mockApi(context, [userSession]);
  await page.goto("/");
  await page.getByRole("button", { name: "Application menu", exact: true }).click();
  await expect(page.getByRole("menuitem")).toHaveText(["Information", "Settings", "Log out"]);
  await page.getByRole("menuitem", { name: "Information", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Information" });
  await expect(dialog).toContainText("Username");
  await expect(dialog).toContainText("alice");
  await expect(dialog).toContainText(version);
  await expect(dialog).not.toContainText("@");
  expect(new URL(page.url()).pathname).toBe("/");
});

test("administrator sees Information and Log out", async ({ page, context }) => {
  await mockApi(context, [adminSession]);
  await page.goto("/");
  await page.getByRole("button", { name: "Application menu", exact: true }).click();
  await expect(page.getByRole("menuitem")).toHaveText(["Information", "Log out"]);
  await page.getByRole("menuitem", { name: "Information", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Information" });
  await expect(dialog).toContainText("admin");
  await expect(dialog).toContainText(version);
});

test("open mode shows only the version", async ({ page, context }) => {
  await mockApi(context, [openSession]);
  await page.goto("/");
  const dialog = await openInformation(page);
  await expect(dialog).toContainText(version);
  await expect(dialog).not.toContainText("Username");
});

test("REV-002: ended session shows no stale username and returns to the login page", async ({ page, context }) => {
  const ended = { status: 200, json: { mode: "protected", user: null, version } };
  const api = await mockApi(context, [userSession]);
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Application menu", exact: true })).toBeVisible();
  api.replies = [ended];
  await page.getByRole("button", { name: "Application menu", exact: true }).click();
  await page.getByRole("menuitem", { name: "Information", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Log in to Price follower" })).toBeVisible();
  await expect(page.getByText("alice")).toHaveCount(0);
});

test("loading indicator while the version is pending", async ({ page, context }) => {
  await mockApi(context, [openSession, { ...openSession, delayMs: 1500 }]);
  await page.goto("/");
  const dialog = await openInformation(page);
  await expect(dialog.getByText("Loading version")).toBeVisible();
  await expect(dialog).toContainText(version);
});

test("failure shows an error and reopening retries", async ({ page, context }) => {
  const api = await mockApi(context, [openSession]);
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Application menu", exact: true })).toBeVisible();
  api.replies = [failure, openSession];
  api.sessionCalls = 0;
  const dialog = await openInformation(page);
  await expect(dialog.getByText("The version could not be loaded.")).toBeVisible();
  await expect(dialog).not.toContainText(version);
  await dialog.getByRole("button", { name: "Close", exact: true }).first().click();
  await expect(dialog).toBeHidden();
  const reopened = await openInformation(page);
  await expect(reopened).toContainText(version);
  expect(api.sessionCalls).toBe(2);
});

test("Close button, close icon and Escape close the modal and return focus to the menu", async ({ page, context }) => {
  await mockApi(context, [openSession]);
  await page.goto("/");
  const menu = page.getByRole("button", { name: "Application menu", exact: true });
  const information = page.getByRole("dialog", { name: "Information" });
  const footerClose = information.locator(".cds--modal-footer").getByRole("button", { name: "Close" });
  const iconClose = information.locator(".cds--modal-close");
  for (const close of [() => footerClose.click(), () => iconClose.click(), () => page.keyboard.press("Escape")]) {
    const dialog = await openInformation(page);
    await expect(dialog).toContainText(version);
    await close();
    await expect(dialog).toBeHidden();
    await expect(menu).toBeFocused();
  }
});

test("keyboard opens the entry and focus stays inside the modal", async ({ page, context }) => {
  await mockApi(context, [openSession]);
  await page.goto("/");
  const menu = page.getByRole("button", { name: "Application menu", exact: true });
  await menu.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("menuitem", { name: "Information", exact: true })).toBeFocused();
  await page.keyboard.press("Enter");
  const dialog = page.getByRole("dialog", { name: "Information" });
  await expect(dialog).toContainText(version);
  // The Carbon modal moves focus inside only once it has opened; tabbing earlier starts from the menu.
  await expect.poll(() => dialog.evaluate((element) => element.contains(document.activeElement))).toBe(true);
  for (let i = 0; i < 4; i += 1) {
    await page.keyboard.press("Tab");
    expect(await dialog.evaluate((element) => element.contains(document.activeElement))).toBe(true);
  }
});

test("at 400 px a long username wraps without horizontal scrolling", async ({ page, context }) => {
  const longName = "a".repeat(32);
  await mockApi(context, [{ status: 200, json: { mode: "protected", user: { username: longName, role: "user" }, version } }]);
  await page.setViewportSize({ width: 400, height: 800 });
  await page.goto("/");
  const dialog = await openInformation(page);
  await expect(dialog).toContainText(longName);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  const box = await dialog.boundingBox();
  expect(box && box.x >= 0 && box.x + box.width <= 400).toBe(true);
});

// Issue #215 (FR-APP-MENU-007, FR-APP-INFO-014): keys pressed without waiting.
async function throttleCpu(page: Page, context: BrowserContext) {
  const cdp = await context.newCDPSession(page);
  await cdp.send("Emulation.setCPUThrottlingRate", { rate: 6 });
}

// Records every element that gets focus outside the dialog of the visible modal; the list must stay empty.
// The record starts empty at each call.
async function recordFocusOutsideModal(page: Page) {
  await page.evaluate(() => {
    const record = window as unknown as { focusOutsideModal?: string[] };
    const installed = record.focusOutsideModal !== undefined;
    record.focusOutsideModal = [];
    if (installed) return;
    document.addEventListener("focusin", (event) => {
      const dialog = document.querySelector(".cds--modal.is-visible [role='dialog']");
      const target = event.target as Element;
      if (dialog && !dialog.contains(target)) record.focusOutsideModal!.push(target.outerHTML.slice(0, 120));
    }, true);
  });
}

function focusOutsideModal(page: Page) {
  return page.evaluate(() => (window as unknown as { focusOutsideModal: string[] }).focusOutsideModal);
}

async function openInformationWithKeyboard(page: Page) {
  await page.getByRole("button", { name: "Application menu", exact: true }).focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("menuitem", { name: "Information", exact: true })).toBeFocused();
  await page.keyboard.press("Enter");
  const dialog = page.getByRole("dialog", { name: "Information" });
  await expect(dialog).toBeVisible();
  await expect.poll(() => dialog.evaluate((element) => element.contains(document.activeElement))).toBe(true);
  await recordFocusOutsideModal(page);
  return dialog;
}

// Numbers the dialog's tabbable controls in document order and returns their count.
function numberTabbables(dialog: Locator) {
  return dialog.evaluate((element) => {
    element.querySelectorAll("[data-tab-position]").forEach((control) => control.removeAttribute("data-tab-position"));
    const controls = [...element.querySelectorAll<HTMLElement>("a[href], button, input, select, textarea, [tabindex]")].filter((control) =>
      control.tabIndex >= 0 && !control.matches(":disabled") && control.getClientRects().length > 0 && getComputedStyle(control).visibility !== "hidden");
    controls.forEach((control, position) => control.setAttribute("data-tab-position", String(position)));
    return controls.length;
  });
}

// One synchronous read, no retry: the position of the focused control, -1 for another element inside
// the dialog, null when focus is outside the dialog.
function focusPosition(dialog: Locator) {
  return dialog.evaluate((element) => {
    if (!element.contains(document.activeElement)) return null;
    return Number(document.activeElement?.getAttribute("data-tab-position") ?? -1);
  });
}

// Presses the keys and checks after each one that focus is on the next (Tab) or previous (Shift+Tab)
// control, wrapping at both ends. Fast mode checks right after the press; slow mode waits for the focus.
async function pressAndFollow(page: Page, dialog: Locator, keys: string[], count: number, start: number, slow = false) {
  let position = start;
  for (const [index, key] of keys.entries()) {
    await page.keyboard.press(key);
    position = (position + (key === "Tab" ? 1 : count - 1)) % count;
    if (slow) await expect.poll(() => focusPosition(dialog), `focus after key ${index + 1} (${key})`).toBe(position);
    else expect(await focusPosition(dialog), `focus right after key ${index + 1} (${key})`).toBe(position);
  }
  return position;
}

async function startPosition(dialog: Locator) {
  const count = await numberTabbables(dialog);
  expect(count).toBeGreaterThanOrEqual(2);
  const start = (await focusPosition(dialog)) ?? -1;
  expect(start).toBeGreaterThanOrEqual(0);
  return { count, start };
}

test("TC-001 (#215) regular user: Escape at once after Enter or Space closes the menu", async ({ page, context }) => {
  await mockApi(context, [userSession]);
  await page.goto("/");
  const menu = page.getByRole("button", { name: "Application menu", exact: true });
  await expect(menu).toBeVisible();
  await page.waitForLoadState("networkidle");
  const requests: string[] = [];
  page.on("request", (request) => {
    if (new URL(request.url()).pathname.startsWith("/api/")) requests.push(request.url());
  });
  await menu.focus();
  for (const open of ["Enter", "Space"]) {
    await page.keyboard.press(open);
    await page.keyboard.press("Escape");
    await expect(page.getByRole("menuitem")).toHaveCount(0);
    await expect(menu).toBeFocused();
    await expect(menu).toHaveAttribute("aria-expanded", "false");
  }
  await page.keyboard.press("Enter");
  await expect(page.getByRole("menuitem")).toHaveText(["Information", "Settings", "Log out"]);
  await expect(page.getByRole("menuitem", { name: "Information", exact: true })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menuitem")).toHaveCount(0);
  await expect(menu).toBeFocused();
  await expect(menu).toHaveAttribute("aria-expanded", "false");
  await expect(page).toHaveURL("/");
  expect(requests).toEqual([]);
});

// Issue #225 (FR-APP-MENU-009): a regular user's page is ready, then every API request is recorded,
// the CPU throttled and the menu control focused.
async function userMenuReady(page: Page, context: BrowserContext) {
  await mockApi(context, [userSession]);
  await page.goto("/");
  const menu = page.getByRole("button", { name: "Application menu", exact: true });
  await expect(menu).toBeVisible();
  await page.waitForLoadState("networkidle");
  const requests: string[] = [];
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (url.pathname.startsWith("/api/")) requests.push(`${request.method()} ${url.pathname}`);
  });
  await throttleCpu(page, context);
  await menu.focus();
  return { menu, requests };
}

test("TC-002 (#225) slow machine, regular user: Enter, ArrowDown, Enter at once opens Settings in each of 5 rounds", async ({ page, context }) => {
  test.setTimeout(90_000);
  const { menu, requests } = await userMenuReady(page, context);
  const settings = page.getByRole("dialog", { name: "Settings" });
  for (let round = 1; round <= 5; round += 1) {
    await page.keyboard.press("Enter");
    await page.keyboard.press("ArrowDown");
    await page.keyboard.press("Enter");
    await expect(settings, `Settings opened in round ${round}`).toBeVisible();
    await expect(page.locator("[role=dialog]:visible")).toHaveCount(1);
    await expect(page.getByRole("menuitem")).toHaveCount(0);
    await expect(settings.getByRole("textbox", { name: "LeBonCoin session" })).toBeEnabled();
    await page.keyboard.press("Escape");
    await expect(settings).toBeHidden();
    await expect(menu).toBeFocused();
    await expect(menu).toHaveAttribute("aria-expanded", "false");
  }
  expect([...requests].sort()).toEqual([
    ...Array(5).fill("GET /api/v1/settings/claude-token"),
    ...Array(5).fill("GET /api/v1/settings/leboncoin-session"),
  ]);
  await page.keyboard.press("Enter");
  await expect(page.getByRole("menuitem")).toHaveText(["Information", "Settings", "Log out"]);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menuitem")).toHaveCount(0);
  await expect(menu).toBeFocused();
  await expect(page).toHaveURL("/");
});

test("TC-004 (#225) slow machine, regular user: Enter, ArrowUp at once focuses Log out in each of 5 rounds", async ({ page, context }) => {
  test.setTimeout(90_000);
  const { menu, requests } = await userMenuReady(page, context);
  for (let round = 1; round <= 5; round += 1) {
    await page.keyboard.press("Enter");
    await page.keyboard.press("ArrowUp");
    await expect(page.getByRole("menuitem")).toHaveText(["Information", "Settings", "Log out"]);
    await expect(menu).toHaveAttribute("aria-expanded", "true");
    await expect(page.getByRole("menuitem", { name: "Log out", exact: true }), `Log out focused in round ${round}`).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(page.getByRole("menuitem")).toHaveCount(0);
    await expect(menu).toBeFocused();
    await expect(menu).toHaveAttribute("aria-expanded", "false");
  }
  expect(requests).toEqual([]);
  await expect(page).toHaveURL("/");
});

const tabCases = [
  { name: "open mode", session: openSession },
  { name: "regular user", session: userSession },
  { name: "administrator", session: adminSession },
  { name: "open mode at 400 px", session: openSession, width: 400 },
  { name: "slow machine", session: openSession, slow: true },
];

for (const { name, session, width, slow } of tabCases) {
  test(`TC-004 (#215) ${name}: Tab pressed repeatedly with no wait stays inside and wraps to the first control`, async ({ page, context }) => {
    test.setTimeout(90_000);
    await mockApi(context, [session]);
    if (width) await page.setViewportSize({ width, height: 800 });
    await page.goto("/");
    await expect(page.getByRole("button", { name: "Application menu", exact: true })).toBeVisible();
    if (slow) await throttleCpu(page, context);
    // Close and reopen between rounds; the slow machine repeats the sequence 5 times.
    for (let round = 0; round < (slow ? 5 : 2); round += 1) {
      const dialog = await openInformationWithKeyboard(page);
      await expect(dialog).toContainText(version);
      const { count, start } = await startPosition(dialog);
      await pressAndFollow(page, dialog, Array(2 * count + 1).fill("Tab"), count, start);
      expect(await focusOutsideModal(page)).toEqual([]);
      await dialog.locator(".cds--modal-footer").getByRole("button", { name: "Close" }).click();
      await expect(dialog).toBeHidden();
    }
  });
}

for (const slow of [false, true]) {
  test(`TC-005 (#215) Shift+Tab from the first control goes directly to the last control${slow ? " on a slow machine" : ""}`, async ({ page, context }) => {
    test.setTimeout(90_000);
    await mockApi(context, [openSession]);
    await page.goto("/");
    await expect(page.getByRole("button", { name: "Application menu", exact: true })).toBeVisible();
    if (slow) await throttleCpu(page, context);
    const dialog = await openInformationWithKeyboard(page);
    await expect(dialog).toContainText(version);
    const { count } = await startPosition(dialog);
    for (let round = 0; round < (slow ? 5 : 1); round += 1) {
      await dialog.locator('[data-tab-position="0"]').focus();
      expect(await focusPosition(dialog)).toBe(0);
      const last = await pressAndFollow(page, dialog, ["Shift+Tab"], count, 0);
      expect(last).toBe(count - 1);
      const end = await pressAndFollow(page, dialog, Array(count).fill("Shift+Tab"), count, last);
      await pressAndFollow(page, dialog, ["Tab", "Tab", "Shift+Tab", "Shift+Tab", "Shift+Tab"], count, end);
    }
    expect(await focusOutsideModal(page)).toEqual([]);
  });
}

test("TC-006 (#215) focus stays inside while the version is loading and when it arrives during tabbing", async ({ page, context }) => {
  const api = await mockApi(context, [openSession]);
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Application menu", exact: true })).toBeVisible();
  await page.waitForLoadState("networkidle");
  let release!: () => void;
  api.replies = [{ ...openSession, gate: new Promise<void>((resolve) => { release = resolve; }) }];
  api.sessionCalls = 0;
  const dialog = await openInformationWithKeyboard(page);
  await expect(dialog.getByText("Loading version")).toBeVisible();
  const { count, start } = await startPosition(dialog);
  const keys = [...Array(count + 1).fill("Tab"), ...Array(count + 1).fill("Shift+Tab")];
  const fastEnd = await pressAndFollow(page, dialog, keys, count, start);
  await expect(dialog.getByText("Loading version")).toBeVisible();
  await dialog.locator(`[data-tab-position="${start}"]`).focus();
  expect(await pressAndFollow(page, dialog, keys, count, start, true)).toBe(fastEnd);
  expect(await focusOutsideModal(page)).toEqual([]);

  let position = await pressAndFollow(page, dialog, ["Tab", "Tab"], count, fastEnd);
  release();
  for (let presses = 0; presses < 200 && !(await dialog.textContent())?.includes(version); presses += 1) {
    position = await pressAndFollow(page, dialog, ["Tab"], count, position);
  }
  await expect(dialog).toContainText(version);
  await expect(dialog.getByText("Loading version")).toHaveCount(0);
  await pressAndFollow(page, dialog, [...Array(count + 1).fill("Tab"), ...Array(count + 1).fill("Shift+Tab")], count, position);
  expect(await focusOutsideModal(page)).toEqual([]);
});

test("TC-006 (#215) focus stays inside when the version could not be loaded", async ({ page, context }) => {
  const api = await mockApi(context, [openSession]);
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Application menu", exact: true })).toBeVisible();
  await page.waitForLoadState("networkidle");
  api.replies = [failure];
  api.sessionCalls = 0;
  const dialog = await openInformationWithKeyboard(page);
  await expect(dialog.getByText("The version could not be loaded.")).toBeVisible();
  const { count, start } = await startPosition(dialog);
  const keys = [...Array(count + 1).fill("Tab"), ...Array(count + 1).fill("Shift+Tab")];
  const fastEnd = await pressAndFollow(page, dialog, keys, count, start);
  await dialog.locator(`[data-tab-position="${start}"]`).focus();
  expect(await pressAndFollow(page, dialog, keys, count, start, true)).toBe(fastEnd);
  await expect(dialog.getByText("The version could not be loaded.")).toBeVisible();
  expect(await focusOutsideModal(page)).toEqual([]);
});

test("TC-007 (#215) closing after fast tabbing returns focus to the menu control", async ({ page, context }) => {
  test.setTimeout(90_000);
  await mockApi(context, [openSession]);
  await page.goto("/");
  const menu = page.getByRole("button", { name: "Application menu", exact: true });
  const emptyList = page.getByRole("heading", { name: "No items tracked yet" });
  await expect(emptyList).toBeVisible();
  const information = page.getByRole("dialog", { name: "Information" });
  const footerClose = information.locator(".cds--modal-footer").getByRole("button", { name: "Close" });
  const iconClose = information.locator(".cds--modal-close");
  const closers = [() => page.keyboard.press("Escape"), () => footerClose.click(), () => iconClose.click()];
  for (let round = 0; round < 3; round += 1) {
    for (const close of closers) {
      const dialog = await openInformationWithKeyboard(page);
      const count = await numberTabbables(dialog);
      expect(count).toBeGreaterThanOrEqual(2);
      for (let presses = 0; presses <= count; presses += 1) await page.keyboard.press("Tab");
      await close();
      await expect(dialog).toBeHidden();
      await expect(menu).toBeFocused();
      await expect(page.getByRole("menuitem")).toHaveCount(0);
      await expect(page).toHaveURL("/");
      await expect(emptyList).toBeVisible();
      expect(await focusOutsideModal(page)).toEqual([]);
    }
  }
});
