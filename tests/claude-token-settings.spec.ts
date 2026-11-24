import { expect, test, type BrowserContext, type Page } from "@playwright/test";

// Synthetic values only; never real tokens or cookies.
// Issue #265: the API returns whether a secret is saved and its hint (last 4 characters), never the secret.
const timestamp = "2026-10-03T08:00:00Z";
const tokenT = "sk-ant-oat01-synthetic-token-wxyz"; // the saved token (hint wxyz)
const tokenU = "sk-ant-oat01-synthetic-new-uvst";
const revokedSession = {
  saved: true, hint: "9f3a", revision: 3, updatedAt: timestamp, status: "revoked",
  expiresAt: null, revokedAt: "2026-10-03T21:00:00Z", lastAttempt: null,
};
type Token = { saved: boolean; hint: string | null; updatedAt: string | null; lastRejectedAt: string | null };
type Reply = { status: number; json: unknown };
type SaveBody = { leboncoinSession?: unknown; claudeToken?: { value?: string; remove?: boolean } };
const noToken: Token = { saved: false, hint: null, updatedAt: null, lastRejectedAt: null };
const savedToken: Token = { saved: true, hint: "wxyz", updatedAt: timestamp, lastRejectedAt: null };

async function mockApi(context: BrowserContext, token: Token) {
  const state = { token: { ...token }, puts: [] as SaveBody[], putReplies: [] as Reply[], putGate: Promise.resolve() };
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== "http://127.0.0.1:4173") return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    if (url.pathname === "/api/v1/auth/session") return route.fulfill({ json: { mode: "open", user: null } });
    const method = route.request().method();
    if (method === "GET" && url.pathname === "/api/v1/items") return route.fulfill({ json: { items: [] } });
    if (method === "GET" && url.pathname === "/api/v1/settings/leboncoin-session") return route.fulfill({ json: { session: revokedSession } });
    if (method === "GET" && url.pathname === "/api/v1/settings/claude-token") return route.fulfill({ json: { claudeToken: state.token } });
    if (method === "PUT" && url.pathname === "/api/v1/settings") {
      const body = route.request().postDataJSON() as SaveBody;
      state.puts.push(body);
      await state.putGate;
      const reply = state.putReplies.shift();
      if (reply) return route.fulfill(reply);
      if (body.claudeToken?.remove) state.token = { ...noToken, updatedAt: timestamp };
      else if (body.claudeToken?.value?.trim()) state.token = { saved: true, hint: body.claudeToken.value.trim().slice(-4), updatedAt: timestamp, lastRejectedAt: null };
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

test("TC-012 (#265) no token: the entry says none is saved and AI reviews are unavailable; no remove action", async ({ page, context }) => {
  await mockApi(context, noToken);
  const dialog = await openSettings(page);
  await expect(dialog).toContainText("claude setup-token");
  await expect(dialog.getByText("No token is saved.", { exact: true })).toBeVisible();
  await expect(dialog).toContainText("AI reviews are unavailable until a token is saved.");
  await expect(dialog).toContainText("Leave it empty to keep the saved token.");
  await expect(dialog.getByRole("button", { name: "Remove token", exact: true })).toHaveCount(0);
  await expect(dialog.getByRole("textbox", { name: "Claude token" })).toHaveValue("");
});

test("TC-013 (#265) saving a token sends only the Claude part, shows progress, and reopening shows its hint", async ({ page, context }) => {
  const api = await mockApi(context, noToken);
  let release!: () => void;
  api.putGate = new Promise<void>((resolve) => { release = resolve; });
  const dialog = await openSettings(page);
  await dialog.getByRole("textbox", { name: "Claude token" }).fill(tokenU);
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog.getByRole("button", { name: "Saving…" })).toBeDisabled();
  await expect(dialog.getByText("Saving settings…")).toBeVisible();
  release();
  await expect(dialog).toBeHidden();
  // The untouched LeBoncoin session is not sent, so no session revision check is requested and its warning is kept.
  expect(api.puts).toEqual([{ claudeToken: { value: tokenU } }]);
  const reopened = await openSettings(page);
  await expect(reopened.getByRole("textbox", { name: "Claude token" })).toHaveValue("");
  await expect(reopened.getByText("A token is saved: ••••uvst.", { exact: true })).toBeVisible();
  await expect(reopened.getByText(/LeBoncoin revoked this session/)).toBeVisible();
  await expect(reopened).not.toContainText("AI reviews are unavailable");
});

test("TC-013 (#265) a blank session input next to a typed token sends only the token", async ({ page, context }) => {
  const api = await mockApi(context, savedToken);
  const dialog = await openSettings(page);
  await dialog.getByRole("textbox", { name: "LeBonCoin session" }).fill("  \n ");
  await dialog.getByRole("textbox", { name: "Claude token" }).fill(`  ${tokenU}  `);
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([{ claudeToken: { value: `  ${tokenU}  ` } }]);
});

// Issue #286: a whitespace-only input of any length leaves the saved secret untouched.
test("long whitespace TC-005 a 1025-space token input sends no replacement, shows no message and closes", async ({ page, context }) => {
  const api = await mockApi(context, { ...savedToken, lastRejectedAt: "2026-10-03T20:00:00Z" });
  const dialog = await openSettings(page);
  await dialog.getByRole("textbox", { name: "Claude token" }).fill(" ".repeat(1025));
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByText(/too long|Saving settings…/i)).toHaveCount(0);
  for (const put of api.puts) {
    expect(put.claudeToken?.remove).toBeFalsy();
    expect(put.claudeToken?.value?.trim() ?? "").toBe("");
  }
  expect(JSON.stringify(api.puts)).not.toContain(" ".repeat(1025));
  const reopened = await openSettings(page);
  await expect(reopened.getByRole("textbox", { name: "Claude token" })).toHaveValue("");
  await expect(reopened.getByText("A token is saved: ••••wxyz.", { exact: true })).toBeVisible();
  await expect(reopened.getByText(/Claude rejected this token for an AI review on 03 Oct 2026, 20:00 UTC/)).toBeVisible();
});

test("long whitespace TC-005 a 9000-space session input next to a typed token sends only the token", async ({ page, context }) => {
  const api = await mockApi(context, savedToken);
  const dialog = await openSettings(page);
  await dialog.getByRole("textbox", { name: "LeBonCoin session" }).fill(" ".repeat(9000));
  await dialog.getByRole("textbox", { name: "Claude token" }).fill(tokenU);
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByText(/too long/i)).toHaveCount(0);
  expect(api.puts).toEqual([{ claudeToken: { value: tokenU } }]);
});

test("TC-016 (#265) the saved token is shown by its hint with its rejected warning; an untouched Save sends nothing", async ({ page, context }) => {
  const api = await mockApi(context, { ...savedToken, lastRejectedAt: "2026-10-03T20:00:00Z" });
  const dialog = await openSettings(page);
  await expect(dialog.getByText("A token is saved: ••••wxyz.", { exact: true })).toBeVisible();
  await expect(dialog.getByText(/Claude rejected this token for an AI review on 03 Oct 2026, 20:00 UTC/)).toBeVisible();
  await expect(dialog.getByRole("textbox", { name: "Claude token" })).toHaveValue("");
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([]);
  const reopened = await openSettings(page);
  await expect(reopened.getByText(/Claude rejected this token for an AI review/)).toBeVisible();
});

test("TC-013 (#265) the same token pasted again is sent as a replacement; the warning is gone afterwards", async ({ page, context }) => {
  const api = await mockApi(context, { ...savedToken, lastRejectedAt: "2026-10-03T20:00:00Z" });
  const dialog = await openSettings(page);
  await dialog.getByRole("textbox", { name: "Claude token" }).fill(tokenT);
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([{ claudeToken: { value: tokenT } }]);
  const reopened = await openSettings(page);
  await expect(reopened.getByText("A token is saved: ••••wxyz.", { exact: true })).toBeVisible();
  await expect(reopened.getByText(/Claude rejected this token/)).toHaveCount(0);
});

for (const { code, status, message } of [
  { code: "INVALID_CLAUDE_TOKEN", status: 400, message: "The Claude token must not contain spaces, line breaks or other control characters." },
  { code: "CLAUDE_TOKEN_REJECTED", status: 422, message: "Claude refused this token. Create a new one with claude setup-token." },
  { code: "CLAUDE_UNREACHABLE", status: 502, message: "The token could not be verified because Claude could not be reached. Try again." },
]) {
  test(`TC-013 (#265) ${code} is shown on the Claude entry, keeps the input and a retry sends the same request`, async ({ page, context }) => {
    const api = await mockApi(context, savedToken);
    api.putReplies.push({ status, json: { error: { code, message } } });
    const dialog = await openSettings(page);
    const field = dialog.getByRole("textbox", { name: "Claude token" });
    await field.fill(tokenU);
    await dialog.getByRole("button", { name: "Save" }).click();
    await expect(field).toHaveAttribute("aria-invalid", "true");
    await expect(dialog.getByText(message)).toBeVisible();
    await expect(field).toHaveValue(tokenU);
    await expect(dialog.getByRole("textbox", { name: "LeBonCoin session" })).not.toHaveAttribute("aria-invalid", "true");
    await dialog.getByRole("button", { name: "Save" }).click();
    await expect(dialog).toBeHidden();
    expect(api.puts).toEqual([{ claudeToken: { value: tokenU } }, { claudeToken: { value: tokenU } }]);
  });
}

test("server token error is cleared by editing", async ({ page, context }) => {
  const api = await mockApi(context, noToken);
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

test("TC-014 (#265) Remove token marks the token for removal; Save removes it and leaves the session untouched", async ({ page, context }) => {
  const api = await mockApi(context, savedToken);
  const dialog = await openSettings(page);
  const field = dialog.getByRole("textbox", { name: "Claude token" });
  await field.fill(tokenU);
  await dialog.getByRole("button", { name: "Remove token", exact: true }).click();
  await expect(dialog.getByText("The saved token will be removed when you save.", { exact: true })).toBeVisible();
  await expect(dialog.getByText("A token is saved: ••••wxyz.", { exact: true })).toHaveCount(0);
  await expect(field).toBeDisabled();
  await expect(field).toHaveValue("");
  await expect(dialog.getByRole("button", { name: "Undo token removal", exact: true })).toBeEnabled();
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([{ claudeToken: { remove: true } }]);
  const reopened = await openSettings(page);
  await expect(reopened.getByText("No token is saved.", { exact: true })).toBeVisible();
  await expect(reopened.getByRole("button", { name: /token/ })).toHaveCount(0);
  await expect(reopened).toContainText("AI reviews are unavailable until a token is saved.");
  await expect(reopened.getByText(/LeBoncoin revoked this session/)).toBeVisible();
});

test("TC-014 (#265) undoing a token removal sends nothing", async ({ page, context }) => {
  const api = await mockApi(context, savedToken);
  const dialog = await openSettings(page);
  await dialog.getByRole("button", { name: "Remove token", exact: true }).click();
  await dialog.getByRole("button", { name: "Undo token removal", exact: true }).click();
  await expect(dialog.getByText("A token is saved: ••••wxyz.", { exact: true })).toBeVisible();
  await expect(dialog.getByRole("textbox", { name: "Claude token" })).toBeEnabled();
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([]);
});
