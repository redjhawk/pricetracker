import { expect, test, type BrowserContext, type Page } from "@playwright/test";

// Synthetic values only; never real tokens or cookies.
const timestamp = "2026-10-03T08:00:00Z";
const revokedSession = {
  value: "Synthetic~Value_123", revision: 3, updatedAt: timestamp, status: "revoked",
  expiresAt: null, revokedAt: "2026-10-03T21:00:00Z", lastAttempt: null,
};
type Token = { value: string | null; updatedAt: string | null; lastRejectedAt: string | null };
type Reply = { status: number; json: unknown };

async function mockApi(context: BrowserContext, token: Token) {
  const state = { token: { ...token }, puts: [] as Record<string, unknown>[], putReplies: [] as Reply[] };
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== "http://127.0.0.1:4173") return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    const method = route.request().method();
    if (method === "GET" && url.pathname === "/api/v1/items") return route.fulfill({ json: { items: [] } });
    if (method === "GET" && url.pathname === "/api/v1/settings/leboncoin-session") return route.fulfill({ json: { session: revokedSession } });
    if (method === "GET" && url.pathname === "/api/v1/settings/claude-token") return route.fulfill({ json: { claudeToken: state.token } });
    if (method === "PUT" && url.pathname === "/api/v1/settings") {
      const body = route.request().postDataJSON() as { claudeToken?: { value: string } };
      state.puts.push(body);
      const reply = state.putReplies.shift();
      if (reply) return route.fulfill(reply);
      if (body.claudeToken) state.token = { value: body.claudeToken.value.trim() || null, updatedAt: timestamp, lastRejectedAt: null };
      return route.fulfill({ json: { session: revokedSession, claudeToken: state.token } });
    }
    return route.abort();
  });
  return state;
}

async function openSettings(page: Page) {
  await page.goto("/");
  await page.getByRole("button", { name: "Application menu", exact: true }).click();
  await page.getByRole("menuitem", { name: "Settings", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Settings" });
  await expect(dialog.getByRole("textbox", { name: "Claude token" })).toBeEnabled();
  return dialog;
}

test("no token: helper says AI reviews are unavailable; saving a token sends only the Claude part", async ({ page, context }) => {
  const api = await mockApi(context, { value: null, updatedAt: null, lastRejectedAt: null });
  const dialog = await openSettings(page);
  await expect(dialog).toContainText("claude setup-token");
  await expect(dialog).toContainText("AI reviews are unavailable until a token is saved.");
  await dialog.getByRole("textbox", { name: "Claude token" }).fill("sk-ant-oat01-synthetic");
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog).toBeHidden();
  // REV-002: the unchanged LeBoncoin session is not resent, so its revoked warning is kept.
  expect(api.puts).toEqual([{ claudeToken: { value: "sk-ant-oat01-synthetic" } }]);
  const reopened = await openSettings(page);
  await expect(reopened.getByRole("textbox", { name: "Claude token" })).toHaveValue("sk-ant-oat01-synthetic");
  await expect(reopened.getByText(/LeBoncoin revoked this session/)).toBeVisible();
});

test("unchanged token is not sent; rejected-token warning shown", async ({ page, context }) => {
  const api = await mockApi(context, { value: "sk-ant-oat01-old", updatedAt: timestamp, lastRejectedAt: "2026-10-03T20:00:00Z" });
  const dialog = await openSettings(page);
  await expect(dialog.getByText(/Claude rejected this token for an AI review on 03 Oct 2026, 20:00 UTC/)).toBeVisible();
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toHaveLength(1);
  expect(api.puts[0].claudeToken).toBeUndefined();
});

test("server token error is shown on the Claude entry and cleared by editing", async ({ page, context }) => {
  const api = await mockApi(context, { value: null, updatedAt: null, lastRejectedAt: null });
  const message = "Claude refused this token. Create a new one with claude setup-token.";
  api.putReplies.push({ status: 422, json: { error: { code: "CLAUDE_TOKEN_REJECTED", message } } });
  const dialog = await openSettings(page);
  const field = dialog.getByRole("textbox", { name: "Claude token" });
  await field.fill("sk-ant-oat01-bad");
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(field).toHaveAttribute("aria-invalid", "true");
  await expect(dialog.getByText(message)).toBeVisible();
  await expect(field).toHaveValue("sk-ant-oat01-bad");
  await field.fill("sk-ant-oat01-good");
  await expect(field).not.toHaveAttribute("aria-invalid", "true");
});
