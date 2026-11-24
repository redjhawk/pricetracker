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
// Issue #265: the API returns whether each secret is saved and a hint (its last 4
// characters), never the secret itself.
const tokenT = "sk-ant-oat01-synthetic-token-wxyz"; // saved Claude token (hint wxyz)
const tokenU = "sk-ant-oat01-synthetic-new-uvst"; // new Claude token
const sessionN = "synthetic-renewed-or-new-n0p1"; // new LeBoncoin session
const noSession = {
  saved: false, hint: null as string | null, revision: 0, updatedAt: null as string | null,
  status: "none" as string, expiresAt: null as string | null, revokedAt: null as string | null,
  lastAttempt: null as { outcome: string; attemptedAt: string } | null,
};
// The saved session S ends with 9f3a.
const activeSession = { ...noSession, saved: true, hint: "9f3a" as string | null, revision: 7, updatedAt: timestamp, status: "active" };
const noClaudeToken = { saved: false, hint: null as string | null, updatedAt: null as string | null, lastRejectedAt: null as string | null };
const savedToken = { ...noClaudeToken, saved: true, hint: "wxyz", updatedAt: timestamp };
type Session = typeof noSession;
type Token = typeof noClaudeToken;
type PutReply = { status: number; json: unknown } | "echo";
type SaveBody = {
  leboncoinSession?: { value?: string; remove?: boolean; revision: number };
  claudeToken?: { value?: string; remove?: boolean };
};

const settingsPath = "/api/v1/settings/leboncoin-session";
const claudeTokenPath = "/api/v1/settings/claude-token";
const sessionChanged = { status: 409, json: { error: { code: "SESSION_CHANGED", message: "The LeBoncoin session changed after Settings was opened. Reopen Settings before saving." } } };
const serverFailure = { status: 500, json: { error: { code: "INTERNAL_ERROR", message: "The server could not complete the request." } } };
const invalidSession = { status: 400, json: { error: { code: "INVALID_SESSION", message: "No datadome cookie was found in the pasted text." } } };
const unexpectedRequests = new WeakMap<BrowserContext, string[]>();

test.afterEach(async ({ context }) => {
  expect(unexpectedRequests.get(context)).toEqual([]);
});

function extracted(value: string) {
  const match = value.match(/datadome=([^;\s]+)/);
  return match ? match[1] : value.trim();
}

function hintOf(value: string) {
  return value.length >= 16 ? value.slice(-4) : null;
}

async function mockApi(context: BrowserContext, session: Session = activeSession, token: Token = savedToken) {
  const state = {
    session: structuredClone(session),
    token: structuredClone(token),
    getError: false,
    getGate: Promise.resolve(),
    putGate: Promise.resolve(),
    putReplies: [] as PutReply[],
    listGate: Promise.resolve(),
    listError: false,
    gets: 0,
    puts: [] as SaveBody[],
    responses: [] as string[],
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
    const fulfill = (reply: { status?: number; json: unknown }) => {
      state.responses.push(JSON.stringify(reply.json));
      return route.fulfill(reply);
    };
    if (url.pathname === "/api/v1/auth/session") return fulfill({ json: { mode: "open", user: null } });
    const method = request.method();
    const path = url.pathname + url.search;
    if (method === "GET" && path === "/api/v1/items") {
      await state.listGate;
      await fulfill(state.listError
        ? { status: 500, json: { error: { code: "TEST_FAILURE", message: "Fixture retrieval failed." } } }
        : { json: { items: [item] } });
    } else if (method === "GET" && path === `/api/v1/items/${item.id}`) {
      await fulfill({ json: { ...item, priceHistory: item.lastThreeDetections } });
    } else if (method === "GET" && path === settingsPath) {
      state.gets += 1;
      await state.getGate;
      await fulfill(state.getError ? serverFailure : { json: { session: state.session } });
    } else if (method === "GET" && path === claudeTokenPath) {
      await state.getGate;
      await fulfill(state.getError ? serverFailure : { json: { claudeToken: state.token } });
    } else if (method === "PUT" && path === "/api/v1/settings") {
      const body = request.postDataJSON() as SaveBody;
      state.puts.push(body);
      await state.putGate;
      const reply = state.putReplies.shift() ?? "echo";
      if (reply === "echo") {
        const sessionPart = body.leboncoinSession;
        if (sessionPart?.remove) {
          state.session = { ...noSession, revision: state.session.revision + 1, updatedAt: timestamp };
        } else if (sessionPart?.value?.trim()) {
          state.session = { ...noSession, saved: true, hint: hintOf(extracted(sessionPart.value)), revision: state.session.revision + 1, updatedAt: timestamp, status: "active" };
        }
        const tokenPart = body.claudeToken;
        if (tokenPart?.remove) state.token = { ...noClaudeToken, updatedAt: timestamp };
        else if (tokenPart?.value?.trim()) state.token = { ...noClaudeToken, saved: true, hint: hintOf(tokenPart.value.trim()), updatedAt: timestamp };
        await fulfill({ json: { session: state.session, claudeToken: state.token } });
      } else {
        await fulfill(reply);
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

function tokenField(page: Page) {
  return page.getByRole("dialog", { name: "Settings" }).getByRole("textbox", { name: "Claude token" });
}

function dialogButton(page: Page, name: string) {
  return page.getByRole("dialog", { name: "Settings" }).getByRole("button", { name, exact: true });
}

const sessionHint = (hint: string) => `A session is saved: ••••${hint}.`;
const tokenHint = (hint: string) => `A token is saved: ••••${hint}.`;
const sessionPending = "The saved session will be removed when you save.";
const tokenPending = "The saved token will be removed when you save.";
const stateLine = /^(A (session|token) is saved|No (session|token) is saved|The saved (session|token) will be removed)/;

async function openLoaded(page: Page) {
  const dialog = await openSettings(page);
  await expect(sessionField(page)).toBeEnabled();
  return dialog;
}

// No remove or undo action can be used (loading, load failure or nothing saved).
async function expectNoUsableRemoveAction(page: Page) {
  const actions = page.getByRole("dialog", { name: "Settings" }).getByRole("button", { name: /^(Remove|Undo) (session|token)( removal)?$/ });
  for (const action of await actions.all()) await expect(action).toBeDisabled();
}

// The request bodies never carry a hint or the masked hint text.
function expectNoHintSent(puts: SaveBody[], ...hints: string[]) {
  const sent = JSON.stringify(puts);
  expect(sent).not.toContain("••••");
  for (const hint of hints) expect(sent).not.toContain(hint);
}

test("header menu: visible on every page, Information and Settings entries, keyboard and dismissal", async ({ page, context }) => {
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
  await expect(page.getByRole("menuitem")).toHaveText(["Information", "Settings"]);
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
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  await expect(page.getByRole("dialog", { name: "Settings" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "LeBoncoin bicycle" })).toBeAttached();
  await expect(page).toHaveURL(`/items/${item.id}`);
});

test("TC-012 (#265) settings shows the hints as text, empty inputs and help texts saying an empty input keeps the secret", async ({ page, context }) => {
  const api = await mockApi(context);
  let release!: () => void;
  api.getGate = new Promise<void>((resolve) => { release = resolve; });
  await page.goto("/");
  const dialog = await openSettings(page);
  await expect(dialog.getByText("Loading settings…")).toBeVisible();
  await expect(saveButton(page)).toBeDisabled();
  await expect(sessionField(page)).toBeDisabled();
  await expect(tokenField(page)).toBeDisabled();
  await expect(dialog.getByText(stateLine)).toHaveCount(0);
  await expectNoUsableRemoveAction(page);
  release();
  await expect(sessionField(page)).toBeEnabled();
  await expect(sessionField(page)).toHaveValue("");
  await expect(tokenField(page)).toHaveValue("");
  await expect(dialog.getByText(sessionHint("9f3a"), { exact: true })).toBeVisible();
  await expect(dialog.getByText(tokenHint("wxyz"), { exact: true })).toBeVisible();
  await expect(dialog).toContainText("Paste the datadome value or a cookie string containing datadome=…. Leave it empty to keep the saved session.");
  await expect(dialog).toContainText("Paste the token printed by claude setup-token (Claude Pro/Max subscription). It is used for LeBoncoin AI reviews. Leave it empty to keep the saved token.");
  await expect(dialog).not.toContainText("AI reviews are unavailable");
  await expect(dialog).not.toContainText(/empty field|save an empty/i);
  await expect(dialogButton(page, "Remove session")).toBeEnabled();
  await expect(dialogButton(page, "Remove token")).toBeEnabled();
  await expect(saveButton(page)).toBeEnabled();
  await expect(dialog.getByText(/rejected|failed|expired|revoked/)).toHaveCount(0);
  expect(api.puts).toEqual([]);
});

test("TC-012 (#265) nothing saved: each entry says none is saved and offers no remove action", async ({ page, context }) => {
  const api = await mockApi(context, noSession, noClaudeToken);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await expect(dialog.getByText("No session is saved.", { exact: true })).toBeVisible();
  await expect(dialog.getByText("No token is saved.", { exact: true })).toBeVisible();
  await expect(dialog).toContainText("AI reviews are unavailable until a token is saved.");
  await expect(dialog.getByRole("button", { name: /^Remove (session|token)$/ })).toHaveCount(0);
  await expect(sessionField(page)).toHaveValue("");
  await expect(tokenField(page)).toHaveValue("");
  expect(api.puts).toEqual([]);
});

test("TC-012 (#265) a short secret is shown as saved without any character", async ({ page, context }) => {
  await mockApi(context, { ...activeSession, hint: null }, { ...savedToken, hint: null });
  await page.goto("/");
  const dialog = await openLoaded(page);
  await expect(dialog.getByText("A session is saved.", { exact: true })).toBeVisible();
  await expect(dialog.getByText("A token is saved.", { exact: true })).toBeVisible();
  await expect(dialog).not.toContainText("••••");
  await expect(dialogButton(page, "Remove session")).toBeEnabled();
  await expect(dialogButton(page, "Remove token")).toBeEnabled();
});

test("TC-012 (#265) load failure disables saving; cancel closes and reopening retries", async ({ page, context }) => {
  const api = await mockApi(context);
  api.getError = true;
  await page.goto("/");
  const dialog = await openSettings(page);
  await expect(dialog.getByText("Could not load settings")).toBeVisible();
  await expect(dialog).toContainText("The server could not complete the request.");
  await expect(dialog.getByRole("textbox")).toHaveCount(0);
  await expect(dialog.getByText(stateLine)).toHaveCount(0);
  await expectNoUsableRemoveAction(page);
  await expect(saveButton(page)).toBeDisabled();
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(dialog).toBeHidden();
  await expect(menuTrigger(page)).toBeFocused();
  api.getError = false;
  await openLoaded(page);
  await expect(sessionField(page)).toHaveValue("");
  await expect(dialog.getByText(sessionHint("9f3a"), { exact: true })).toBeVisible();
  expect(api.gets).toBe(2);
  expect(api.puts).toEqual([]);
});

test("TC-013 (#265) saving a session sends only the typed session and its revision; reopening shows the new hint", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await sessionField(page).fill(`datadome=${sessionN}`);
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([{ leboncoinSession: { value: `datadome=${sessionN}`, revision: 7 } }]);
  expectNoHintSent(api.puts, "wxyz", "9f3a");
  await expect(menuTrigger(page)).toBeFocused();
  await openLoaded(page);
  await expect(sessionField(page)).toHaveValue("");
  await expect(tokenField(page)).toHaveValue("");
  await expect(dialog.getByText(sessionHint("n0p1"), { exact: true })).toBeVisible();
  await expect(dialog.getByText(tokenHint("wxyz"), { exact: true })).toBeVisible();
  expect(api.gets).toBe(2);
});

test("TC-013 (#265) a cookie string is sent as typed", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await sessionField(page).fill(`Cookie: a=1; datadome=${sessionN}; b=2`);
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([{ leboncoinSession: { value: `Cookie: a=1; datadome=${sessionN}; b=2`, revision: 7 } }]);
});

test("TC-013 (#265) both inputs empty or blank and no removal: no request, the modal closes and nothing changes", async ({ page, context }) => {
  const api = await mockApi(context, { ...activeSession, lastAttempt: { outcome: "rejected", attemptedAt: "2026-10-03T20:00:00Z" } },
    { ...savedToken, lastRejectedAt: "2026-10-03T20:00:00Z" });
  await page.goto("/");
  let dialog = await openLoaded(page);
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  dialog = await openLoaded(page);
  await sessionField(page).fill("   \n  ");
  await tokenField(page).fill("   ");
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([]);
  dialog = await openLoaded(page);
  await expect(dialog.getByText(sessionHint("9f3a"), { exact: true })).toBeVisible();
  await expect(dialog.getByText(tokenHint("wxyz"), { exact: true })).toBeVisible();
  await expect(dialog.getByText(/LeBoncoin rejected this session on 03 Oct 2026, 20:00 UTC/)).toBeVisible();
  await expect(dialog.getByText(/Claude rejected this token for an AI review on 03 Oct 2026, 20:00 UTC/)).toBeVisible();
});

test("TC-013 (#265) invalid session shows the server message on the field and keeps input", async ({ page, context }) => {
  const api = await mockApi(context);
  api.putReplies.push(invalidSession);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await sessionField(page).fill("a=1; b=2");
  await saveButton(page).click();
  await expect(sessionField(page)).toHaveAttribute("aria-invalid", "true");
  await expect(dialog.getByText("No datadome cookie was found in the pasted text.")).toBeVisible();
  await expect(sessionField(page)).toHaveValue("a=1; b=2");
  await expect(tokenField(page)).not.toHaveAttribute("aria-invalid", "true");
  await sessionField(page).fill(`a=1; b=2; datadome=${sessionN}`);
  await expect(sessionField(page)).not.toHaveAttribute("aria-invalid", "true");
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([
    { leboncoinSession: { value: "a=1; b=2", revision: 7 } },
    { leboncoinSession: { value: `a=1; b=2; datadome=${sessionN}`, revision: 7 } },
  ]);
});

test("TC-013 (#265) server failure keeps input and a retry sends the same request", async ({ page, context }) => {
  const api = await mockApi(context);
  api.putReplies.push(serverFailure);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await sessionField(page).fill(`datadome=${sessionN}`);
  await saveButton(page).click();
  await expect(dialog.getByText("Could not save settings")).toBeVisible();
  await expect(sessionField(page)).toHaveValue(`datadome=${sessionN}`);
  await expect(saveButton(page)).toBeEnabled();
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  const request = { leboncoinSession: { value: `datadome=${sessionN}`, revision: 7 } };
  expect(api.puts).toEqual([request, request]);
});

test("TC-015 (#265) changed session warns when replacing, keeps input, blocks save until reopened", async ({ page, context }) => {
  const api = await mockApi(context);
  api.putReplies.push(sessionChanged);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await sessionField(page).fill(`datadome=${sessionN}`);
  await saveButton(page).click();
  await expect(dialog.getByText("Session changed", { exact: true })).toBeVisible();
  await expect(dialog).toContainText("close and reopen Settings before saving");
  await expect(sessionField(page)).toHaveValue(`datadome=${sessionN}`);
  await expect(sessionField(page)).toBeEditable();
  await expect(saveButton(page)).toBeDisabled();
  api.session = { ...activeSession, hint: "y7k2", revision: 8, lastAttempt: { outcome: "accepted", attemptedAt: timestamp } };
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await openLoaded(page);
  await expect(dialog.getByText(sessionHint("y7k2"), { exact: true })).toBeVisible();
  await expect(dialog.getByText("Session changed", { exact: true })).toHaveCount(0);
  await expect(sessionField(page)).toHaveValue("");
  await sessionField(page).fill(`datadome=${sessionN}`);
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([
    { leboncoinSession: { value: `datadome=${sessionN}`, revision: 7 } },
    { leboncoinSession: { value: `datadome=${sessionN}`, revision: 8 } },
  ]);
});

test("TC-015 (#265) changed session warns when removing and keeps the pending removal", async ({ page, context }) => {
  const api = await mockApi(context);
  api.putReplies.push(sessionChanged);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await dialogButton(page, "Remove session").click();
  await saveButton(page).click();
  await expect(dialog.getByText("Session changed", { exact: true })).toBeVisible();
  await expect(dialog.getByText(sessionPending, { exact: true })).toBeVisible();
  await expect(dialogButton(page, "Undo session removal")).toBeVisible();
  await expect(saveButton(page)).toBeDisabled();
  api.session = { ...activeSession, hint: "y7k2", revision: 8 };
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await openLoaded(page);
  await expect(dialog.getByText(sessionHint("y7k2"), { exact: true })).toBeVisible();
  await expect(dialog.getByText(sessionPending, { exact: true })).toHaveCount(0);
  await dialogButton(page, "Remove session").click();
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([{ leboncoinSession: { remove: true, revision: 7 } }, { leboncoinSession: { remove: true, revision: 8 } }]);
});

test("TC-015 (#265) a token-only save asks for no session check and shows no warning", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await tokenField(page).fill(tokenU);
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([{ claudeToken: { value: tokenU } }]);
  await expect(page.getByText("Session changed", { exact: true })).toHaveCount(0);
});

test("TC-014 (#265) Remove session marks the session for removal; Save sends only the removal", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await dialogButton(page, "Remove session").click();
  await expect(dialog.getByText(sessionPending, { exact: true })).toBeVisible();
  await expect(dialog.getByText(sessionHint("9f3a"), { exact: true })).toHaveCount(0);
  await expect(sessionField(page)).toBeDisabled();
  await expect(dialogButton(page, "Undo session removal")).toBeEnabled();
  await expect(dialog.getByText(tokenHint("wxyz"), { exact: true })).toBeVisible();
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([{ leboncoinSession: { remove: true, revision: 7 } }]);
  await openLoaded(page);
  await expect(dialog.getByText("No session is saved.", { exact: true })).toBeVisible();
  await expect(dialog.getByRole("button", { name: /session/ })).toHaveCount(0);
  await expect(dialog.getByText(tokenHint("wxyz"), { exact: true })).toBeVisible();
});

test("TC-014 (#265) undoing a session removal makes the input usable and Save then sends no removal", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await dialogButton(page, "Remove session").click();
  await dialogButton(page, "Undo session removal").click();
  await expect(dialog.getByText(sessionPending, { exact: true })).toHaveCount(0);
  await expect(dialog.getByText(sessionHint("9f3a"), { exact: true })).toBeVisible();
  await expect(sessionField(page)).toBeEnabled();
  await expect(sessionField(page)).toHaveValue("");
  await expect(dialogButton(page, "Remove session")).toBeEnabled();
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts.filter((body) => body.leboncoinSession)).toEqual([]);
});

test("TC-014 (#265) Cancel or Escape discards a pending removal without a request", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  let dialog = await openLoaded(page);
  await dialogButton(page, "Remove session").click();
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(dialog).toBeHidden();
  dialog = await openLoaded(page);
  await expect(dialog.getByText(sessionHint("9f3a"), { exact: true })).toBeVisible();
  await expect(dialog.getByText(sessionPending, { exact: true })).toHaveCount(0);
  await dialogButton(page, "Remove session").click();
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  dialog = await openLoaded(page);
  await expect(dialog.getByText(sessionPending, { exact: true })).toHaveCount(0);
  await expect(dialogButton(page, "Remove session")).toBeEnabled();
  await expect(sessionField(page)).toBeEnabled();
  expect(api.puts).toEqual([]);
});

test("TC-014 (#265) removing the token and pasting a new session send one request with both", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await dialogButton(page, "Remove token").click();
  await sessionField(page).fill(`datadome=${sessionN}`);
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([{ leboncoinSession: { value: `datadome=${sessionN}`, revision: 7 }, claudeToken: { remove: true } }]);
});

test("TC-014 (#265) text typed before choosing Remove session is not sent", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await sessionField(page).fill(`datadome=${sessionN}`);
  await dialogButton(page, "Remove session").click();
  await expect(sessionField(page)).toHaveValue("");
  await expect(sessionField(page)).toBeDisabled();
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([{ leboncoinSession: { remove: true, revision: 7 } }]);
  expect(JSON.stringify(api.puts)).not.toContain(sessionN);
});

test("TC-014 (#265) a refused save keeps the pending removals for a retry", async ({ page, context }) => {
  const api = await mockApi(context);
  api.putReplies.push(invalidSession, serverFailure);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await dialogButton(page, "Remove token").click();
  await sessionField(page).fill("a=1; b=2");
  await saveButton(page).click();
  await expect(dialog.getByText("No datadome cookie was found in the pasted text.")).toBeVisible();
  await expect(dialog.getByText(tokenPending, { exact: true })).toBeVisible();
  await expect(dialogButton(page, "Undo token removal")).toBeEnabled();
  await expect(tokenField(page)).toBeDisabled();
  await dialogButton(page, "Remove session").click();
  await saveButton(page).click();
  await expect(dialog.getByText("Could not save settings")).toBeVisible();
  await expect(dialog.getByText(tokenPending, { exact: true })).toBeVisible();
  await expect(dialog.getByText(sessionPending, { exact: true })).toBeVisible();
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  const both = { leboncoinSession: { remove: true, revision: 7 }, claudeToken: { remove: true } };
  expect(api.puts).toEqual([{ leboncoinSession: { value: "a=1; b=2", revision: 7 }, claudeToken: { remove: true } }, both, both]);
});

test("TC-012 (#265) cancel and Escape discard typed text; reopening shows empty inputs", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  let dialog = await openLoaded(page);
  await sessionField(page).fill(`datadome=${sessionN}`);
  await tokenField(page).fill(tokenU);
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(dialog).toBeHidden();
  dialog = await openLoaded(page);
  await expect(sessionField(page)).toHaveValue("");
  await expect(tokenField(page)).toHaveValue("");
  await expect(dialog.getByText(sessionHint("9f3a"), { exact: true })).toBeVisible();
  await sessionField(page).fill(`datadome=${sessionN}`);
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(menuTrigger(page)).toBeFocused();
  await openLoaded(page);
  await expect(sessionField(page)).toHaveValue("");
  expect(api.puts).toEqual([]);
});

test("TC-013 (#265) saving in progress blocks closing, removal and duplicate saves", async ({ page, context }) => {
  const api = await mockApi(context);
  let release!: () => void;
  api.putGate = new Promise<void>((resolve) => { release = resolve; });
  await page.goto("/");
  const dialog = await openLoaded(page);
  await sessionField(page).fill(`datadome=${sessionN}`);
  await saveButton(page).click();
  await expect(saveButton(page)).toHaveText("Saving…");
  await expect(saveButton(page)).toBeDisabled();
  await expect(sessionField(page)).toBeDisabled();
  await expect(dialogButton(page, "Remove session")).toBeDisabled();
  await expect(dialogButton(page, "Remove token")).toBeDisabled();
  await saveButton(page).click({ force: true });
  await page.keyboard.press("Escape");
  await dialog.getByRole("button", { name: "Cancel" }).click({ force: true });
  await expect(dialog).toBeVisible();
  release();
  await expect(dialog).toBeHidden();
  expect(api.puts).toHaveLength(1);
});

// Issue #286: a whitespace-only input of any length leaves the saved secret untouched.
const blankW9000 = " ".repeat(9000);
const blankC1025 = " ".repeat(1025);
const tooLongSession = { status: 400, json: { error: { code: "INVALID_SESSION", message: "The pasted text is too long (maximum 8192 characters)." } } };
const warnedSession = { ...activeSession, lastAttempt: { outcome: "rejected", attemptedAt: "2026-10-03T20:00:00Z" } };
const warnedToken = { ...savedToken, lastRejectedAt: "2026-10-03T20:00:00Z" };

// No request replaces or removes a secret, or carries a long blank input.
function expectNoSecretChangeSent(puts: SaveBody[]) {
  for (const put of puts) {
    expect(put.leboncoinSession?.remove).toBeFalsy();
    expect(put.leboncoinSession?.value?.trim() ?? "").toBe("");
    expect(put.claudeToken?.remove).toBeFalsy();
    expect(put.claudeToken?.value?.trim() ?? "").toBe("");
  }
  expect(JSON.stringify(puts)).not.toContain(" ".repeat(1025));
}

async function expectUntouchedAfterReopening(page: Page) {
  const dialog = await openLoaded(page);
  await expect(sessionField(page)).toHaveValue("");
  await expect(tokenField(page)).toHaveValue("");
  await expect(dialog.getByText(sessionHint("9f3a"), { exact: true })).toBeVisible();
  await expect(dialog.getByText(tokenHint("wxyz"), { exact: true })).toBeVisible();
  await expect(dialog.getByText(/LeBoncoin rejected this session on 03 Oct 2026, 20:00 UTC/)).toBeVisible();
  await expect(dialog.getByText(/Claude rejected this token for an AI review on 03 Oct 2026, 20:00 UTC/)).toBeVisible();
}

for (const { name, session, token } of [
  { name: "a 9000-space session paste", session: blankW9000, token: "" },
  { name: "both inputs filled with long whitespace", session: blankW9000, token: blankC1025 },
]) {
  test(`long whitespace TC-005 ${name} saves without a "too long" message and changes nothing`, async ({ page, context }) => {
    const api = await mockApi(context, warnedSession, warnedToken);
    await page.goto("/");
    const dialog = await openLoaded(page);
    await sessionField(page).fill(session);
    await tokenField(page).fill(token);
    await saveButton(page).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByText(/too long/i)).toHaveCount(0);
    expectNoSecretChangeSent(api.puts);
    await expectUntouchedAfterReopening(page);
  });
}

test("long whitespace TC-005 a long paste that is not whitespace only shows the server's too long message", async ({ page, context }) => {
  const api = await mockApi(context);
  api.putReplies.push(tooLongSession);
  const value = `${blankW9000}abc`;
  await page.goto("/");
  const dialog = await openLoaded(page);
  await sessionField(page).fill(value);
  await saveButton(page).click();
  await expect(sessionField(page)).toHaveAttribute("aria-invalid", "true");
  await expect(dialog.getByText("The pasted text is too long (maximum 8192 characters).")).toBeVisible();
  await expect(dialog).toBeVisible();
  await expect(sessionField(page)).toHaveValue(value);
  await expect(tokenField(page)).not.toHaveAttribute("aria-invalid", "true");
  expect(api.puts).toEqual([{ leboncoinSession: { value, revision: 7 } }]);
});

test("long whitespace TC-005 Cancel after a 9000-space paste sends nothing; reopening shows an empty input", async ({ page, context }) => {
  const api = await mockApi(context, warnedSession, warnedToken);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await sessionField(page).fill(blankW9000);
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(dialog).toBeHidden();
  await expectUntouchedAfterReopening(page);
  expect(api.puts).toEqual([]);
});

const notUsed = "It is no longer used. Capture a new session and save it here, or remove it.";
const hintCases: { name: string; session: Session; state: string; text: string | RegExp | null }[] = [
  { name: "rejected", session: { ...activeSession, lastAttempt: { outcome: "rejected", attemptedAt: "2026-10-03T20:00:00Z" } }, state: sessionHint("9f3a"),
    text: /LeBoncoin rejected this session on 03 Oct 2026, 20:00 UTC\. Capture a new session and save it here\./ },
  { name: "failed", session: { ...activeSession, lastAttempt: { outcome: "failed", attemptedAt: "2026-10-03T20:00:00Z" } }, state: sessionHint("9f3a"),
    text: /The last LeBoncoin check using this session failed on 03 Oct 2026, 20:00 UTC for a reason other than a rejection\./ },
  { name: "expired", session: { ...activeSession, status: "expired", expiresAt: "2026-10-02T10:00:00Z",
    lastAttempt: { outcome: "rejected", attemptedAt: "2026-10-03T20:00:00Z" } }, state: sessionHint("9f3a"),
    text: `This session expired on 02 Oct 2026, 10:00 UTC. ${notUsed}` },
  { name: "revoked", session: { ...activeSession, status: "revoked", revokedAt: "2026-10-03T21:00:00Z" }, state: sessionHint("9f3a"),
    text: `LeBoncoin revoked this session on 03 Oct 2026, 21:00 UTC. ${notUsed}` },
  { name: "revoked without a remaining value", session: { ...noSession, revision: 8, status: "revoked", revokedAt: "2026-10-03T21:00:00Z" },
    state: "No session is saved.", text: `LeBoncoin revoked this session on 03 Oct 2026, 21:00 UTC. ${notUsed}` },
  { name: "renewed", session: { ...activeSession, hint: "y7k2", revision: 8, lastAttempt: { outcome: "accepted", attemptedAt: timestamp } },
    state: sessionHint("y7k2"), text: null },
  { name: "accepted", session: { ...activeSession, lastAttempt: { outcome: "accepted", attemptedAt: timestamp } }, state: sessionHint("9f3a"), text: null },
  { name: "unused", session: activeSession, state: sessionHint("9f3a"), text: null },
  { name: "none", session: noSession, state: "No session is saved.", text: null },
];

for (const { name, session, state, text } of hintCases) {
  test(`TC-016 (#265) hint and warning for ${name} session`, async ({ page, context }) => {
    await mockApi(context, session);
    await page.goto("/");
    const dialog = await openLoaded(page);
    await expect(sessionField(page)).toHaveValue("");
    await expect(dialog.getByText(state, { exact: true })).toBeVisible();
    await expect(dialog).not.toContainText(/empty field/i);
    if (text) {
      await expect(dialog.getByText(text)).toBeVisible();
      await expect(dialog.getByText(/rejected|failed|expired|revoked/)).toHaveCount(1);
    } else {
      await expect(dialog.getByText(/rejected|failed|expired|revoked/)).toHaveCount(0);
    }
  });
}

test("TC-016 (#265) after a successful replacement the reopened modal shows no warning", async ({ page, context }) => {
  await mockApi(context, { ...activeSession, lastAttempt: { outcome: "rejected", attemptedAt: "2026-10-03T20:00:00Z" } });
  await page.goto("/");
  const dialog = await openLoaded(page);
  await expect(dialog.getByText(/LeBoncoin rejected this session/)).toBeVisible();
  await sessionField(page).fill(`datadome=${sessionN}`);
  await saveButton(page).click();
  await expect(dialog).toBeHidden();
  await openLoaded(page);
  await expect(dialog.getByText(sessionHint("n0p1"), { exact: true })).toBeVisible();
  await expect(dialog.getByText(/rejected|failed|expired|revoked/)).toHaveCount(0);
});

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
  await mockApi(context, { ...activeSession, lastAttempt: { outcome: "rejected", attemptedAt: "2026-10-03T20:00:00Z" } });
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
  await expect(sessionField(page)).toBeEnabled();
  await expect(sessionField(page)).toHaveValue("");
  await expect(saveButton(page)).toBeInViewport();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});

// Everything the page keeps after the modal closed: storage, IndexedDB, cookies, DOM, form values and URL.
async function browserState(page: Page) {
  const pageState = await page.evaluate(async () => {
    const databases = indexedDB.databases ? await indexedDB.databases() : [];
    const values = [...document.querySelectorAll("input, textarea")].map((field) => (field as HTMLInputElement).value);
    return JSON.stringify({ local: { ...localStorage }, session: { ...sessionStorage }, databases, values })
      + document.documentElement.outerHTML + location.href;
  });
  return pageState + JSON.stringify(await page.context().cookies());
}

async function expectNoTypedSecret(page: Page) {
  await expect(page.getByRole("dialog", { name: "Settings" })).toBeHidden();
  const state = await browserState(page);
  for (const secret of [tokenU, sessionN]) expect(state).not.toContain(secret);
}

async function typeBothSecrets(page: Page) {
  await expect(sessionField(page)).toHaveValue("");
  await expect(tokenField(page)).toHaveValue("");
  await sessionField(page).fill(`datadome=${sessionN}`);
  await tokenField(page).fill(tokenU);
}

async function secretLifecycle(page: Page, api: Awaited<ReturnType<typeof mockApi>>) {
  let dialog = await openLoaded(page);
  await typeBothSecrets(page);
  await saveButton(page).click();
  await expectNoTypedSecret(page);
  dialog = await openLoaded(page);
  await typeBothSecrets(page);
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expectNoTypedSecret(page);
  api.putReplies.push(serverFailure);
  dialog = await openLoaded(page);
  await typeBothSecrets(page);
  await saveButton(page).click();
  await expect(dialog.getByText("Could not save settings")).toBeVisible();
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expectNoTypedSecret(page);
  await openLoaded(page);
  await expect(sessionField(page)).toHaveValue("");
  await expect(tokenField(page)).toHaveValue("");
  await page.keyboard.press("Escape");
  await expectNoTypedSecret(page);
}

test("TC-017 (#265) the browser holds no full secret after the modal closes", async ({ page, context }) => {
  const api = await mockApi(context);
  const apiRequests: string[] = [];
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (url.pathname.startsWith("/api/")) apiRequests.push(`${request.method()} ${url.pathname}`);
  });
  await page.goto("/");
  await secretLifecycle(page, api);
  expect(api.puts).toHaveLength(2);
  for (const response of api.responses) {
    for (const secret of [tokenT, tokenU, sessionN]) expect(response).not.toContain(secret);
  }
  expect(apiRequests.filter((request) => request.startsWith("PUT ") && request !== "PUT /api/v1/settings")).toEqual([]);
});

test("TC-017 (#265) narrow viewport and page reload give the same result", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.setViewportSize({ width: 400, height: 800 });
  await page.goto("/");
  await secretLifecycle(page, api);
  await page.reload();
  await expect(menuTrigger(page)).toBeVisible();
  await expectNoTypedSecret(page);
  await secretLifecycle(page, api);
  expect(api.puts).toHaveLength(4);
});

// Tabs forward until the focused element has the given accessible text.
async function tabTo(page: Page, name: string) {
  for (let i = 0; i < 25; i += 1) {
    await page.keyboard.press("Tab");
    const focused = await page.evaluate(() => (document.activeElement?.closest("[role=dialog]") ? document.activeElement.textContent?.trim() : null));
    if (focused === name) return;
  }
  throw new Error(`Tab did not reach "${name}" in the Settings dialog`);
}

test("TC-018 (#265) hints are the inputs' descriptions; remove and undo work from the keyboard", async ({ page, context }) => {
  const api = await mockApi(context);
  await page.goto("/");
  const dialog = await openLoaded(page);
  await expect(sessionField(page)).toHaveAccessibleDescription(/A session is saved: ••••9f3a\./);
  await expect(tokenField(page)).toHaveAccessibleDescription(/A token is saved: ••••wxyz\./);

  await tabTo(page, "Remove session");
  await expect(dialogButton(page, "Remove session")).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(dialogButton(page, "Undo session removal")).toBeFocused();
  await expect(sessionField(page)).toHaveAccessibleDescription(new RegExp(sessionPending.replace(/\./g, "\\.")));
  await expect(dialog.locator("[aria-live=polite]").filter({ hasText: sessionPending })).toHaveCount(1);
  await page.keyboard.press("Space");
  await expect(dialogButton(page, "Remove session")).toBeFocused();
  await expect(sessionField(page)).toHaveAccessibleDescription(/A session is saved: ••••9f3a\./);
  await expect(dialog.locator("[aria-live=polite]").filter({ hasText: sessionHint("9f3a") })).toHaveCount(1);

  await tabTo(page, "Remove token");
  await page.keyboard.press("Space");
  await expect(dialogButton(page, "Undo token removal")).toBeFocused();
  await expect(tokenField(page)).toHaveAccessibleDescription(new RegExp(tokenPending.replace(/\./g, "\\.")));
  await expect(dialog.locator("[aria-live=polite]").filter({ hasText: tokenPending })).toHaveCount(1);
  await page.keyboard.press("Enter");
  await expect(dialogButton(page, "Remove token")).toBeFocused();
  await expect(tokenField(page)).toBeEnabled();

  await expectTabbingStaysInDialog(page);
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  expect(api.puts).toEqual([]);
});

test("TC-018 (#265) at 400 px the hints, actions and inputs fit without horizontal scrolling", async ({ page, context }) => {
  await mockApi(context, { ...activeSession, status: "expired", expiresAt: "2026-10-02T10:00:00Z" }, { ...savedToken, lastRejectedAt: "2026-10-03T20:00:00Z" });
  await page.setViewportSize({ width: 400, height: 800 });
  await page.goto("/");
  const dialog = await openLoaded(page);
  await dialogButton(page, "Remove session").click();
  for (const target of [
    dialog.getByText(sessionPending, { exact: true }), dialog.getByText(tokenHint("wxyz"), { exact: true }),
    dialogButton(page, "Undo session removal"), dialogButton(page, "Remove token"), sessionField(page), tokenField(page),
  ]) {
    await target.scrollIntoViewIfNeeded();
    await expect(target).toBeVisible();
    const box = (await target.boundingBox())!;
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(400);
  }
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
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
  await page.keyboard.press("ArrowDown");
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

// Issue #215 (FR-APP-MENU-007): keys pressed without waiting act as the same keys pressed slowly.
async function throttleCpu(page: Page, context: BrowserContext) {
  const cdp = await context.newCDPSession(page);
  await cdp.send("Emulation.setCPUThrottlingRate", { rate: 6 });
}

// Loads the list page, then records every API request made afterwards.
async function listReady(page: Page) {
  await page.goto("/");
  await expect(page.getByText(/^1 item tracked/)).toBeVisible();
  await page.waitForLoadState("networkidle");
  const requests: string[] = [];
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (url.pathname.startsWith("/api/")) requests.push(`${request.method()} ${url.pathname}`);
  });
  return requests;
}

async function expectMenuClosed(page: Page) {
  await expect(page.getByRole("menuitem")).toHaveCount(0);
  await expect(menuTrigger(page)).toBeFocused();
  await expect(menuTrigger(page)).toHaveAttribute("aria-expanded", "false");
}

async function expectMenuOpen(page: Page) {
  await expect(page.getByRole("menuitem")).toHaveText(["Information", "Settings"]);
  await expect(menuTrigger(page)).toHaveAttribute("aria-expanded", "true");
}

async function expectMenuState(page: Page, open: boolean) {
  if (open) await expectMenuOpen(page);
  else await expectMenuClosed(page);
}

test("TC-001 (#215) Escape at once after Enter or Space closes the menu", async ({ page, context }) => {
  const api = await mockApi(context);
  const requests = await listReady(page);
  await menuTrigger(page).focus();
  for (const open of ["Enter", "Space"]) {
    await page.keyboard.press(open);
    await page.keyboard.press("Escape");
    await expectMenuClosed(page);
  }
  await page.keyboard.press("Enter");
  await expectMenuOpen(page);
  await expect(page.getByRole("menuitem", { name: "Information", exact: true })).toBeFocused();
  await page.keyboard.press("ArrowDown");
  await expect(page.getByRole("menuitem", { name: "Settings", exact: true })).toBeFocused();
  await page.keyboard.press("Escape");
  await expectMenuClosed(page);
  await expect(page).toHaveURL("/");
  await expect(page.locator("[role=dialog]:visible")).toHaveCount(0);
  expect(requests).toEqual([]);
  expect(api.gets).toBe(0);
});

test("TC-001 (#215) slow machine: Enter or Space then Escape closes the menu each time", async ({ page, context }) => {
  test.setTimeout(90_000);
  const api = await mockApi(context);
  const requests = await listReady(page);
  await throttleCpu(page, context);
  await menuTrigger(page).focus();
  for (let round = 0; round < 5; round += 1) {
    for (const open of ["Enter", "Space"]) {
      await page.keyboard.press(open);
      await page.keyboard.press("Escape");
      await expectMenuClosed(page);
    }
  }
  await page.keyboard.press("Enter");
  await expectMenuOpen(page);
  await page.keyboard.press("Escape");
  await expectMenuClosed(page);
  await expect(page).toHaveURL("/");
  expect(requests).toEqual([]);
  expect(api.gets).toBe(0);
});

// Each sequence alternates opening and closing keys; `open` is the state it leaves.
const quickSequences = [
  { keys: ["Enter", "Escape", "Enter"], open: true },
  { keys: ["Enter", "Escape", "Space"], open: true },
  { keys: ["Enter", "Escape", "Enter", "Escape"], open: false },
];

async function runQuickSequences(page: Page) {
  for (const { keys, open } of quickSequences) {
    for (const key of keys) await page.keyboard.press(key);
    await expectMenuState(page, open);
    if (open) {
      await page.keyboard.press("Escape");
      await expectMenuClosed(page);
    }
    for (const [index, key] of keys.entries()) {
      await page.keyboard.press(key);
      await expectMenuState(page, index % 2 === 0);
      if (index % 2 === 0) await expect(page.getByRole("menuitem", { name: "Information", exact: true })).toBeFocused();
    }
    await expectMenuState(page, open);
    if (open) {
      await page.keyboard.press("Escape");
      await expectMenuClosed(page);
    }
  }
}

test("TC-002 (#215) keys pressed in quick succession give the same result as slow keys", async ({ page, context }) => {
  const api = await mockApi(context);
  const requests = await listReady(page);
  await menuTrigger(page).focus();
  await runQuickSequences(page);
  await expect(page).toHaveURL("/");
  expect(requests).toEqual([]);
  expect(api.gets).toBe(0);
});

test("TC-002 (#215) slow machine: keys pressed in quick succession give the same result as slow keys", async ({ page, context }) => {
  test.setTimeout(90_000);
  await mockApi(context);
  await listReady(page);
  await throttleCpu(page, context);
  await menuTrigger(page).focus();
  await runQuickSequences(page);
});

test("TC-003 (#215) Escape after a pause, click then Escape, Escape while closed and the other ways to close are unchanged", async ({ page, context }) => {
  const api = await mockApi(context);
  const requests = await listReady(page);
  const trigger = menuTrigger(page);
  await trigger.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("menuitem", { name: "Information", exact: true })).toBeFocused();
  await page.keyboard.press("Escape");
  await expectMenuClosed(page);

  await trigger.click();
  await expectMenuOpen(page);
  await page.keyboard.press("Escape");
  await expectMenuClosed(page);

  await page.keyboard.press("Escape");
  await expectMenuClosed(page);
  await expect(page.locator("[role=dialog]:visible")).toHaveCount(0);
  await page.keyboard.press("Enter");
  await expectMenuOpen(page);
  await page.keyboard.press("Escape");
  await expectMenuClosed(page);

  await trigger.click();
  await expectMenuOpen(page);
  await trigger.click();
  await expect(page.getByRole("menuitem")).toHaveCount(0);
  await expect(trigger).toHaveAttribute("aria-expanded", "false");
  await expect(page).toHaveURL("/");
  expect(requests).toEqual([]);
  expect(api.gets).toBe(0);

  await page.goto(`/items/${item.id}`);
  const heading = page.getByRole("heading", { name: "LeBoncoin bicycle" });
  await expect(heading).toBeVisible();
  await menuTrigger(page).click();
  await expectMenuOpen(page);
  await heading.click();
  await expect(page.getByRole("menuitem")).toHaveCount(0);
  await expect(menuTrigger(page)).toHaveAttribute("aria-expanded", "false");
  await menuTrigger(page).focus();
  await page.keyboard.press("Enter");
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  await expect(page.getByRole("dialog", { name: "Settings" })).toBeVisible();
  await expect(heading).toBeAttached();
  await expect(page).toHaveURL(`/items/${item.id}`);
});
