import { expect, test, type BrowserContext, type Page } from "@playwright/test";

// FR-APP-INFO-001..013. Synthetic session data only.
type SessionReply = { status: number; json: unknown; delayMs?: number };

async function mockApi(context: BrowserContext, replies: SessionReply[]) {
  const state = { sessionCalls: 0 };
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== "http://127.0.0.1:4173") return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    if (url.pathname === "/api/v1/auth/session") {
      const reply = replies[Math.min(state.sessionCalls, replies.length - 1)];
      state.sessionCalls += 1;
      if (reply.delayMs) await new Promise((resolve) => setTimeout(resolve, reply.delayMs));
      return route.fulfill({ status: reply.status, json: reply.json });
    }
    if (url.pathname === "/api/v1/items") return route.fulfill({ json: { items: [] } });
    if (url.pathname === "/api/v1/amazon-searches") return route.fulfill({ json: { searches: [], amazonRequests: [] } });
    if (url.pathname === "/api/v1/admin/users") return route.fulfill({ json: { users: [] } });
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
  await mockApi(context, [userSession, ended]);
  await page.goto("/");
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
  const api = await mockApi(context, [openSession, failure, openSession]);
  await page.goto("/");
  const dialog = await openInformation(page);
  await expect(dialog.getByText("The version could not be loaded.")).toBeVisible();
  await expect(dialog).not.toContainText(version);
  await dialog.getByRole("button", { name: "Close", exact: true }).first().click();
  await expect(dialog).toBeHidden();
  const reopened = await openInformation(page);
  await expect(reopened).toContainText(version);
  expect(api.sessionCalls).toBe(3);
});

test("Close button, close icon and Escape close the modal and return focus to the menu", async ({ page, context }) => {
  await mockApi(context, [openSession]);
  await page.goto("/");
  const menu = page.getByRole("button", { name: "Application menu", exact: true });
  const footerClose = page.locator(".cds--modal-footer").getByRole("button", { name: "Close" });
  const iconClose = page.locator(".cds--modal-close");
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
  const box = await page.locator(".cds--modal-container").boundingBox();
  expect(box && box.x >= 0 && box.x + box.width <= 400).toBe(true);
});
