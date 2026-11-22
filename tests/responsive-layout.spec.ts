import { expect, test, type BrowserContext, type Locator, type Page, type Route } from "@playwright/test";

// FR-RESPONSIVE-001..013 (issue #140) and 014..024 (issue #165). Synthetic data only.
const timestamp = "2026-11-10T08:00:00Z";
const price = { amountCents: 123456789, currency: "EUR", timestamp };
const usedPrice = { ...price, amountCents: 99999999, condition: "good", conditionLabel: "Used - Very good condition" };
const longTitle = "A".repeat(200);
const longUsername = "u".repeat(80);
const amazon = {
  id: "amazon-one", title: longTitle, platform: "amazon",
  listingId: "B09XS7JWHH", asin: "B09XS7JWHH", marketplace: "amazon.de",
  url: `https://www.amazon.de/dp/B09XS7JWHH/${"x".repeat(150)}`, thumbnailUrl: null,
  status: "active", latestPrice: price, lastThreeDetections: [price, price, price],
  secondHandOffer: { status: "available", latestDetection: usedPrice, lastThreeDetections: [usedPrice], lastCheckedAt: timestamp },
  lastAttempt: { result: "success", timestamp, message: null },
  nextCheckAt: null, addedAt: timestamp, purchaseGoal: "", amazonStoppedAt: timestamp,
};
const content = {
  price: { rating: "fair", explanation: "B".repeat(180) },
  condition: { rating: null, explanation: "No photos to assess." },
  recommendation: { rating: "negotiate", explanation: "Offer a bit less." },
  fairPrice: { minCents: 12000, maxCents: 16000, suggestedOfferCents: 13000 },
  risks: [], missingInformation: ["Charger not shown"], sellerQuestions: ["Is the charger included?"],
  descriptionVsPhotos: { matches: null, explanation: "No photos.", mismatches: [] },
};
const review = { id: 1, status: "succeeded", priceCents: 15000, createdAt: timestamp, completedAt: timestamp, review: content, errorMessage: null };
const leboncoin = {
  ...amazon, id: "leboncoin-one", title: "LeBoncoin bicycle", platform: "leboncoin",
  listingId: "1234567890", asin: null, marketplace: "leboncoin.fr",
  url: `https://www.leboncoin.fr/ad/velos/1234567890?${"q".repeat(150)}`, secondHandOffer: null,
  purchaseGoal: "C".repeat(200), amazonStoppedAt: null,
};
const aiReview = { tokenConfigured: true, running: false, lastAttempt: review, latest: review, history: [review] };
const openSession = { mode: "open", user: null, version: "v1" };
const adminSession = { mode: "protected", user: { username: "admin", role: "admin" }, version: "v1" };

// Issue #165: items shaped like the development seed data (short titles, three prices).
const seedTimes = ["2026-11-10T08:00:00Z", "2026-11-09T20:00:00Z", "2026-11-09T08:00:00Z"];
const seedPrices = (cents: number[]) => cents.map((amountCents, index) => ({ amountCents, currency: "EUR", timestamp: seedTimes[index], oldPrice: false }));
const seedUsed = { amountCents: 24900, currency: "EUR", timestamp, condition: "very_good", conditionLabel: "Used - Very good" };
const sampleAmazon = {
  ...amazon, id: "amazon-sample", title: "Sony WH-1000XM5 Wireless Noise Canceling Headphones",
  marketplace: "amazon.fr", url: "https://www.amazon.fr/dp/B09XS7JWHH",
  latestPrice: seedPrices([29900])[0], lastThreeDetections: seedPrices([29900, 31900, 34900]),
  secondHandOffer: { status: "available", latestDetection: seedUsed, lastThreeDetections: [seedUsed], lastCheckedAt: timestamp },
  amazonStoppedAt: null,
};
const sampleAmazonNoOffer = {
  ...sampleAmazon, id: "amazon-sample-no-offer", title: "Kindle Paperwhite 16 GB",
  listingId: "B0CFPJYX7P", asin: "B0CFPJYX7P", url: "https://www.amazon.fr/dp/B0CFPJYX7P",
  secondHandOffer: { status: "not_found", latestDetection: null, lastThreeDetections: [], lastCheckedAt: timestamp },
};
const sampleLeboncoin = {
  ...leboncoin, id: "leboncoin-sample", title: "Vélo de route Btwin Triban RC 520",
  listingId: "2876543210", url: "https://www.leboncoin.fr/ad/velos/2876543210",
  latestPrice: seedPrices([45000])[0], lastThreeDetections: seedPrices([45000, 48000, 50000]), purchaseGoal: "",
};
const sampleItems = [sampleAmazon, sampleAmazonNoOffer, sampleLeboncoin];
const allItems = [amazon, sampleAmazon, sampleAmazonNoOffer, leboncoin, sampleLeboncoin];
const olderReview = { ...review, id: 2, priceCents: 16000, createdAt: "2026-11-01T08:00:00Z", completedAt: "2026-11-01T08:00:00Z" };
const itemDetails = {
  [amazon.id]: {
    ...amazon, aiReview: null, priceHistory: [price, price, price],
    secondHandOffer: { ...amazon.secondHandOffer, priceHistory: [usedPrice, usedPrice] },
  },
  [leboncoin.id]: { ...leboncoin, aiReview: { ...aiReview, history: [review, olderReview] } },
};

// Requests held by "hold" replies. Unrouting resumes paused requests to the network, which
// the dev server would proxy to a backend that is not running, so abort them first.
const heldRoutes: Route[] = [];
test.afterEach(() => abortHeldRoutes());

type MockOptions = {
  session?: unknown;
  items?: unknown[];
  itemsReply?: "hold" | "error";
  itemDetails?: Record<string, unknown>;
  itemReply?: "hold";
  users?: unknown[];
};

function hold(route: Route) {
  heldRoutes.push(route);
}

async function abortHeldRoutes() {
  await Promise.all(heldRoutes.splice(0).map((route) => route.abort().catch(() => {})));
}

async function mockApi(context: BrowserContext, options: MockOptions = {}) {
  await context.addInitScript(layoutProbe);
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== "http://127.0.0.1:4173") return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    const path = url.pathname;
    if (path === "/api/v1/auth/session") return route.fulfill({ json: options.session ?? openSession });
    if (path === "/api/v1/auth/login") {
      return route.fulfill({ status: 401, json: { error: { code: "INVALID_CREDENTIALS", message: "Invalid username or password." } } });
    }
    if (path === "/api/v1/items") {
      // "hold" never answers, so the page stays in its loading state.
      if (options.itemsReply === "hold") return hold(route);
      if (options.itemsReply === "error") {
        return route.fulfill({ status: 500, json: { error: { code: "INTERNAL", message: "The tracked items could not be loaded." } } });
      }
      return route.fulfill({ json: { items: options.items ?? [amazon, leboncoin] } });
    }
    if (path.startsWith("/api/v1/items/")) {
      if (options.itemReply === "hold") return hold(route);
      const detail = options.itemDetails?.[decodeURIComponent(path.slice("/api/v1/items/".length))];
      if (detail) return route.fulfill({ json: detail });
    }
    if (path === `/api/v1/items/${amazon.id}`) return route.fulfill({ json: { ...amazon, aiReview: null } });
    if (path === `/api/v1/items/${leboncoin.id}`) return route.fulfill({ json: { ...leboncoin, aiReview } });
    if (path === "/api/v1/admin/users") {
      return route.fulfill({ json: { users: options.users ?? [{ username: longUsername, role: "user", lastLoginAt: timestamp }] } });
    }
    if (path === "/api/v1/settings/leboncoin-session") {
      return route.fulfill({ json: { session: { value: null, revision: 0, updatedAt: null, status: "none", expiresAt: null, revokedAt: null, lastAttempt: null } } });
    }
    if (path === "/api/v1/settings/claude-token") {
      return route.fulfill({ json: { claudeToken: { value: null, updatedAt: null, lastRejectedAt: null } } });
    }
    return route.fulfill({ status: 500, json: { error: { code: "UNEXPECTED", message: path } } });
  });
}

async function expectNoHorizontalScroll(page: Page) {
  await page.waitForTimeout(100);
  const result = await page.evaluate(() => {
    const width = document.documentElement.clientWidth;
    const wider = [...document.querySelectorAll("body *")]
      .filter((element) => element.getBoundingClientRect().right > width + 0.5)
      .filter((element) => !element.parentElement || element.parentElement.getBoundingClientRect().right <= width + 0.5)
      .slice(0, 5)
      .map((element) => `${element.tagName.toLowerCase()}.${[...element.classList].join(".")} [${element.parentElement?.className}] ${Math.round(element.getBoundingClientRect().left)}-${Math.round(element.getBoundingClientRect().right)} "${element.textContent?.slice(0, 30)}"`);
    return { fits: document.documentElement.scrollWidth <= width, wider };
  });
  expect(result.fits, `elements wider than the viewport: ${result.wider.join(", ")}`).toBe(true);
}

async function openMenuEntry(page: Page, name: string) {
  await page.getByRole("button", { name: "Application menu", exact: true }).click();
  await page.getByRole("menuitem", { name, exact: true }).click();
}

for (const viewport of [{ width: 320, height: 640 }, { width: 360, height: 740 }]) {
  test.describe(`${viewport.width}px`, () => {
    test.use({ viewport });

    test("tracked items list on both tabs", async ({ page, context }) => {
      await mockApi(context);
      await page.goto("/");
      const table = page.getByRole("table", { name: "Tracked items" });
      await expect(table.getByRole("button", { name: longTitle })).toBeVisible();
      await expectNoHorizontalScroll(page);
      for (const label of ["Prices", "Amazon second-hand offer", "Status"]) {
        await expect(table.locator(`td[data-label="${label}"]`)).toBeVisible();
      }
      await expect(page.getByText(/Amazon checks were stopped/)).toBeVisible();
      for (const control of [
        page.getByRole("tab", { name: "Amazon" }),
        page.getByRole("tab", { name: "LeBoncoin" }),
        page.getByRole("textbox", { name: "Search tracked items" }),
        page.getByRole("button", { name: "Refresh all prices" }),
        page.locator(".page-heading-actions").getByRole("button", { name: "Add item" }),
      ]) {
        await control.scrollIntoViewIfNeeded();
        await expect(control).toBeInViewport();
      }
      await page.getByRole("tab", { name: "LeBoncoin" }).click();
      await expect(table.getByRole("button", { name: "LeBoncoin bicycle" })).toBeVisible();
      await expectNoHorizontalScroll(page);
      await page.getByRole("textbox", { name: "Search tracked items" }).fill("nothing-matches");
      await expect(page.getByText("No items match your search.")).toBeVisible();
      await expectNoHorizontalScroll(page);
    });

    test("empty list", async ({ page, context }) => {
      await mockApi(context, { items: [] });
      await page.goto("/");
      await expect(page.getByText("No items tracked yet")).toBeVisible();
      await expectNoHorizontalScroll(page);
    });

    test("item details for Amazon and LeBoncoin", async ({ page, context }) => {
      await mockApi(context);
      await page.goto(`/items/${amazon.id}`);
      await expect(page.getByRole("heading", { name: longTitle })).toBeVisible();
      await expectNoHorizontalScroll(page);
      await page.goto(`/items/${leboncoin.id}`);
      const section = page.getByRole("region", { name: "AI review" });
      await expect(section.getByText("Price: Fair").first()).toBeVisible();
      await expect(page.getByRole("region", { name: "Purchase goal" })).toBeVisible();
      await expectNoHorizontalScroll(page);
      for (const name of ["Refresh price", "Delete"]) {
        const button = page.locator(".detail-actions").getByRole("button", { name });
        await button.scrollIntoViewIfNeeded();
        await expect(button).toBeInViewport();
      }
    });

    test("administrator page with a long username", async ({ page, context }) => {
      await mockApi(context, { session: { mode: "protected", user: { username: "admin", role: "admin" }, version: "v1" } });
      await page.goto("/");
      await expect(page.getByText(longUsername)).toBeVisible();
      await expect(page.getByRole("button", { name: "Add user" })).toBeInViewport();
      await expectNoHorizontalScroll(page);
      await page.getByRole("button", { name: "Add user" }).click();
      const dialog = page.getByRole("dialog", { name: "Add user" });
      await expect(dialog.getByRole("button", { name: "Add user" })).toBeInViewport();
      await expectNoHorizontalScroll(page);
    });

    test("login page and header menu", async ({ page, context }) => {
      await mockApi(context, { session: { mode: "protected", user: null, version: "v1" } });
      await page.goto("/");
      await expect(page.getByRole("button", { name: "Log in" })).toBeInViewport();
      await expectNoHorizontalScroll(page);
      await page.unrouteAll({ behavior: "ignoreErrors" });
      await context.unrouteAll({ behavior: "ignoreErrors" });
      await mockApi(context);
      await page.goto("/");
      await page.getByRole("button", { name: "Application menu", exact: true }).click();
      for (const item of await page.getByRole("menuitem").all()) await expect(item).toBeInViewport();
      await expectNoHorizontalScroll(page);
    });

    test("modals", async ({ page, context }) => {
      await mockApi(context);
      await page.goto("/");
      await expect(page.getByRole("table", { name: "Tracked items" })).toBeVisible();
      const checkModal = async (name: string, buttons: string[]) => {
        const dialog = page.getByRole("dialog", { name });
        await expect(dialog).toBeVisible();
        await expectNoHorizontalScroll(page);
        for (const button of buttons) await expect(dialog.locator(".cds--modal-footer").getByRole("button", { name: button })).toBeInViewport();
        await page.keyboard.press("Escape");
        await expect(dialog).toBeHidden();
      };
      await page.locator(".page-heading-actions").getByRole("button", { name: "Add item" }).click();
      await checkModal("Add tracked item", ["Cancel"]);
      const row = page.getByRole("row").filter({ has: page.getByRole("button", { name: longTitle, exact: true }) });
      await row.getByRole("button", { name: "Options", exact: true }).click();
      const deleteEntry = page.getByRole("menuitem", { name: "Delete", exact: true });
      await expect(deleteEntry).toBeVisible();
      await expectNoHorizontalScroll(page);
      await deleteEntry.click();
      await checkModal("Delete tracked item", ["Cancel"]);
      await openMenuEntry(page, "Settings");
      await checkModal("Settings", []);
      await openMenuEntry(page, "Information");
      await checkModal("Information", ["Close"]);
    });
  });
}

// REV-001: intermediate widths between phone and desktop.
for (const width of [700, 800, 1000]) {
  test(`${width}px list on both tabs`, async ({ page, context }) => {
    await page.setViewportSize({ width, height: 800 });
    await mockApi(context);
    await page.goto("/");
    await expect(page.getByRole("table", { name: "Tracked items" }).getByRole("button", { name: longTitle })).toBeVisible();
    await expectNoHorizontalScroll(page);
    await page.getByRole("tab", { name: "LeBoncoin" }).click();
    await expect(page.getByRole("button", { name: "LeBoncoin bicycle" })).toBeVisible();
    await expectNoHorizontalScroll(page);
  });

  test(`QA-F001 ${width}px tables do not scroll inside and keep Status and actions on screen`, async ({ page, context }) => {
    await page.setViewportSize({ width, height: 800 });
    await mockApi(context);
    await page.goto("/");
    for (const tab of ["Amazon", "LeBoncoin"]) {
      await page.getByRole("tab", { name: tab }).click();
      const table = page.getByRole("table", { name: "Tracked items" });
      await expect(table.locator("tbody tr").first()).toBeVisible();
      for (const scroller of await page.locator(".cds--data-table-content, .tracked-table-container").all()) {
        expect(await scroller.evaluate((node) => node.scrollWidth <= node.clientWidth)).toBe(true);
      }
      for (const cell of await table.locator('td[data-label="Status"], .row-actions').all()) {
        const box = (await cell.boundingBox())!;
        expect(box.x + box.width).toBeLessThanOrEqual(width);
      }
    }
  });

  test(`${width}px admin page`, async ({ page, context }) => {
    await page.setViewportSize({ width, height: 800 });
    await mockApi(context, { session: { mode: "protected", user: { username: "admin", role: "admin" }, version: "v1" } });
    await page.goto("/");
    await expect(page.getByText(longUsername)).toBeVisible();
    await expectNoHorizontalScroll(page);
  });
}

test("resize keeps the tab, search text and modal input", async ({ page, context }) => {
  await page.setViewportSize({ width: 360, height: 740 });
  await mockApi(context);
  await page.goto("/");
  await page.getByRole("tab", { name: "LeBoncoin" }).click();
  await page.getByRole("textbox", { name: "Search tracked items" }).fill("bicycle");
  await page.setViewportSize({ width: 1056, height: 800 });
  await page.setViewportSize({ width: 360, height: 740 });
  await expect(page.getByRole("tab", { name: "LeBoncoin" })).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("textbox", { name: "Search tracked items" })).toHaveValue("bicycle");
  await page.locator(".page-heading-actions").getByRole("button", { name: "Add item" }).click();
  const input = page.getByRole("dialog", { name: "Add tracked item" }).getByRole("textbox").first();
  await input.fill("https://www.leboncoin.fr/ad/velos/1");
  await page.setViewportSize({ width: 1056, height: 800 });
  await page.setViewportSize({ width: 360, height: 740 });
  await expect(input).toHaveValue("https://www.leboncoin.fr/ad/velos/1");
});

test("desktop keeps the tracked table header row", async ({ page, context }) => {
  await page.setViewportSize({ width: 1056, height: 800 });
  await mockApi(context);
  await page.goto("/");
  const header = page.getByRole("table", { name: "Tracked items" }).locator("thead");
  await expect(header).toBeVisible();
  expect(await header.evaluate((element) => getComputedStyle(element).display)).toBe("table-header-group");
  await expect(page.getByRole("columnheader", { name: "Prices" })).toBeVisible();
});

// FR-RESPONSIVE-014..017: the tracked-items list spans the full viewport width on phones.
async function listBox(page: Page) {
  return page.locator(".tracked-table-container").evaluate((element) => {
    const box = element.getBoundingClientRect();
    const textLeft = Math.min(...[...element.querySelectorAll("tbody td")].map((td) => {
      const range = document.createRange();
      range.selectNodeContents(td);
      return range.getBoundingClientRect().left;
    }));
    return { left: box.left, right: box.right, textLeft, width: document.documentElement.clientWidth };
  });
}

// Regression test for UC-TI-06 (REV-001: the Carbon tab panel padding left 16 px margins).
for (const width of [320, 360]) {
  test(`${width}px tracked-items list spans the full width`, async ({ page, context }) => {
    await page.setViewportSize({ width, height: 740 });
    await mockApi(context);
    await page.goto("/");
    await expect(page.getByRole("table", { name: "Tracked items" }).getByRole("button", { name: longTitle })).toBeVisible();
    for (const tab of ["Amazon", "LeBoncoin"]) {
      await page.getByRole("tab", { name: tab }).click();
      const box = await listBox(page);
      expect(Math.abs(box.left)).toBeLessThanOrEqual(1);
      expect(Math.abs(box.right - box.width)).toBeLessThanOrEqual(1);
      expect(box.textLeft).toBeGreaterThanOrEqual(4);
      await expectNoHorizontalScroll(page);
    }
  });
}

// Issue #165 (FR-RESPONSIVE-014..024): below 1056 px every page uses the full width.
const GUTTER = 16.5; // At most 16 px between a screen edge and the nearest content, plus rounding.
const EDGE = 0.5; // Text or a control at x <= 0.5 or right >= width - 0.5 touches the edge.
const cardWidths = [320, 360, 671, 672, 700, 775, 1055];
const tabs = ["Amazon", "LeBoncoin"] as const;
const cardLabels = { Amazon: ["Prices", "Amazon second-hand offer", "Status"], LeBoncoin: ["Prices", "Status"] };

// Baseline measured on master before issue #165: at >= 1056 px the page inset is Carbon's .cds--content
// padding (2rem) plus the .page-shell inset (spacing-07, 2rem, since the --cds-spacing-* tokens are
// declared on :root), and the tracked-items tab panel adds 1rem. The login form is 24rem wide and centred.
const DESKTOP_PAGE_OFFSET = 64;
const DESKTOP_TABLE_OFFSET = 80;
const LOGIN_FORM_WIDTH = 384;
// Carbon header: 3rem high; the name link has a 2px border and 1rem start padding.
const HEADER_HEIGHT = 48;
const HEADER_BRAND_LEFT = 18;
// Carbon "sm" modal container: full width below 672 px, 60% of the overlay from 672 to 1055 px.
const modalWidthRatio = (width: number) => (width < 672 ? 1 : 0.6);

type Rect = { left: number; right: number; top: number; bottom: number; viewport: number };
type Scan = { viewport: number; leftGap: number; rightGap: number; touching: string[] };
type CardCell = {
  label: string; left: number; right: number; top: number; bottom: number; width: number;
  labelShown: boolean; clipped: boolean; visible: boolean;
};

declare global {
  interface Window {
    layoutProbe: {
      hidden(element: Element, cache?: Map<Element, boolean>): boolean;
      contentRect(element: Element): Rect;
      scan(): Scan;
      insetProblems(container: Element): string[];
      labelShown(cell: Element): boolean;
      cells(row: Element): { right: number; cells: CardCell[] };
    };
  }
}

// Runs in the page (context.addInitScript): measures the page as a person sees it.
// "Content" is a visible text line, a control, or the outer box of a filled or bordered element.
function layoutProbe() {
  const controls = "button, a[href], input:not([type=hidden]), textarea, select, [role=tab]";
  const transparent = (color: string) => color === "transparent" || /^rgba\(.*,\s*0\)$/.test(color);
  // Inside a box of at most 1 px that clips its content (the visually hidden pattern).
  const clipped = (element: Element | null, cache: Map<Element, boolean>): boolean => {
    if (!element || element === document.body) return false;
    const known = cache.get(element);
    if (known !== undefined) return known;
    const box = element.getBoundingClientRect();
    const style = getComputedStyle(element);
    const clips = style.overflowX !== "visible" || style.overflowY !== "visible" || style.clip !== "auto";
    const result = ((box.width <= 1 || box.height <= 1) && clips) || clipped(element.parentElement, cache);
    cache.set(element, result);
    return result;
  };
  const hidden = (element: Element, cache = new Map<Element, boolean>()) =>
    !element.checkVisibility({ opacityProperty: true, visibilityProperty: true }) || clipped(element, cache);
  const painted = (element: Element) => {
    const style = getComputedStyle(element);
    if (!transparent(style.backgroundColor) && style.backgroundColor !== getComputedStyle(document.body).backgroundColor) return true;
    return ["top", "right", "bottom", "left"].some((side) =>
      parseFloat(style.getPropertyValue(`border-${side}-width`)) > 0
      && style.getPropertyValue(`border-${side}-style`) !== "none"
      && !transparent(style.getPropertyValue(`border-${side}-color`)));
  };
  const textRects = (root: Element, cache: Map<Element, boolean>, skip = (_element: Element) => false) => {
    const rects: { left: number; right: number; text: string }[] = [];
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    for (let node = walker.nextNode(); node; node = walker.nextNode()) {
      const text = (node.textContent ?? "").trim();
      const parent = node.parentElement;
      if (!text || !parent || skip(parent) || hidden(parent, cache)) continue;
      const range = document.createRange();
      range.selectNodeContents(node);
      for (const rect of range.getClientRects()) {
        if (rect.width > 0 && rect.height > 0) rects.push({ left: rect.left, right: rect.right, text: text.slice(0, 40) });
      }
    }
    return rects;
  };
  // The outer box of a control or a filled/bordered element, otherwise the box of its text.
  const contentRect = (element: Element) => {
    let box: DOMRect;
    if (element.matches(controls) || painted(element)) {
      box = element.getBoundingClientRect();
    } else {
      const range = document.createRange();
      range.selectNodeContents(element);
      box = range.getBoundingClientRect();
    }
    return { left: box.left, right: box.right, top: box.top, bottom: box.bottom, viewport: document.documentElement.clientWidth };
  };
  // Page content outside the application header and modals.
  const scan = () => {
    const viewport = document.documentElement.clientWidth;
    const cache = new Map<Element, boolean>();
    const outside = (element: Element) => Boolean(element.closest(".cds--header, .cds--modal"));
    const boxes: { kind: string; left: number; right: number; label: string }[] = [];
    const root = document.getElementById("root")!;
    for (const element of root.querySelectorAll("*")) {
      if (outside(element) || hidden(element, cache)) continue;
      const box = element.getBoundingClientRect();
      if (box.width < 1 || box.height < 1) continue;
      const kind = element.matches(controls) ? "control" : painted(element) ? "block" : "";
      const label = `${element.tagName.toLowerCase()}.${[...element.classList].join(".")} "${(element.textContent ?? "").trim().slice(0, 30)}"`;
      if (kind) boxes.push({ kind, left: box.left, right: box.right, label });
    }
    for (const rect of textRects(root, cache, outside)) boxes.push({ kind: "text", left: rect.left, right: rect.right, label: `"${rect.text}"` });
    return {
      viewport,
      leftGap: Math.min(...boxes.map((box) => box.left)),
      rightGap: Math.min(...boxes.map((box) => viewport - box.right)),
      touching: boxes
        .filter((box) => box.kind !== "block" && (box.left <= 0.5 || box.right >= viewport - 0.5))
        .map((box) => `${box.kind} ${box.label} ${Math.round(box.left)}-${Math.round(box.right)}`),
    };
  };
  // Text, controls and cell labels closer than 4 px to a screen edge.
  const insetProblems = (container: Element) => {
    const viewport = document.documentElement.clientWidth;
    const cache = new Map<Element, boolean>();
    const problems: string[] = [];
    const check = (what: string, left: number, right: number) => {
      if (left < 4 || right > viewport - 4) problems.push(`${what} ${Math.round(left)}-${Math.round(right)}`);
    };
    for (const rect of textRects(container, cache)) check(`text "${rect.text}"`, rect.left, rect.right);
    for (const element of container.querySelectorAll(controls)) {
      if (hidden(element, cache)) continue;
      const box = element.getBoundingClientRect();
      check(`control "${element.getAttribute("aria-label") ?? (element.textContent ?? "").trim().slice(0, 30)}"`, box.left, box.right);
    }
    for (const cell of container.querySelectorAll("td[data-label]")) {
      const box = cell.getBoundingClientRect();
      const style = getComputedStyle(cell);
      check(`label "${cell.getAttribute("data-label")}"`, box.left + parseFloat(style.paddingLeft), box.right - parseFloat(style.paddingRight));
    }
    return problems;
  };
  // A value is identifiable by its visible label, or by a visible column header.
  const labelShown = (cell: Element) => {
    const label = cell.getAttribute("data-label");
    if (!label) return false;
    const before = getComputedStyle(cell, "::before");
    if (before.display !== "none" && (before.content === JSON.stringify(label) || before.content === "attr(data-label)")) return true;
    const head = cell.closest("table")?.querySelector("thead");
    if (!head || hidden(head)) return false;
    const index = [...cell.parentElement!.children].indexOf(cell);
    return head.querySelectorAll("th")[index]?.textContent?.trim() === label;
  };
  // The cells of a card (a table row), in DOM order, and the right edge of the card's content box.
  const cells = (row: Element) => {
    const box = row.getBoundingClientRect();
    const style = getComputedStyle(row);
    return {
      right: box.right - parseFloat(style.paddingRight) - parseFloat(style.borderRightWidth),
      cells: [...row.children].map((cell) => {
        const cellBox = cell.getBoundingClientRect();
        return {
          label: cell.getAttribute("data-label") ?? "",
          left: cellBox.left, right: cellBox.right, top: cellBox.top, bottom: cellBox.bottom, width: cellBox.width,
          labelShown: labelShown(cell), clipped: cell.scrollWidth > cell.clientWidth + 1, visible: !hidden(cell),
        };
      }),
    };
  };
  window.layoutProbe = { hidden, contentRect, scan, insetProblems, labelShown, cells };
}

async function edgeGaps(locator: Locator) {
  const rect = await locator.evaluate((element) => window.layoutProbe.contentRect(element));
  return { left: rect.left, right: rect.viewport - rect.right };
}

async function boxGaps(locator: Locator) {
  return locator.evaluate((element) => {
    const box = element.getBoundingClientRect();
    return { left: box.left, right: document.documentElement.clientWidth - box.right, width: box.width };
  });
}

async function expectEdgeToEdge(locator: Locator, what: string) {
  const gaps = await boxGaps(locator);
  expect(Math.abs(gaps.left), `${what}: left edge`).toBeLessThanOrEqual(1);
  expect(Math.abs(gaps.right), `${what}: right edge`).toBeLessThanOrEqual(1);
}

function expectGapsWithinGutter(gaps: { left: number; right: number }, what: string) {
  expect(gaps.left, `${what}: left gap`).toBeLessThanOrEqual(GUTTER);
  expect(gaps.right, `${what}: right gap`).toBeLessThanOrEqual(GUTTER);
}

async function expectFullWidth(locator: Locator, what: string) {
  expectGapsWithinGutter(await edgeGaps(locator), what);
}

function expectNotTouching(gaps: { left: number; right: number }, what: string) {
  expect(gaps.left, `${what} touches the left edge`).toBeGreaterThan(EDGE);
  expect(gaps.right, `${what} touches the right edge`).toBeGreaterThan(EDGE);
}

async function scanPage(page: Page) {
  const scan = await page.evaluate(() => window.layoutProbe.scan());
  expect(scan.touching, "text or controls touching a screen edge").toEqual([]);
  return scan;
}

async function expectPageFullWidth(page: Page, what: string) {
  const scan = await scanPage(page);
  expectGapsWithinGutter({ left: scan.leftGap, right: scan.rightGap }, what);
}

async function expectMenuOnScreen(page: Page) {
  const menu = page.getByRole("menu");
  await expect(menu).toBeVisible();
  const gaps = await boxGaps(menu);
  expect(gaps.left).toBeGreaterThanOrEqual(-EDGE);
  expect(gaps.right).toBeGreaterThanOrEqual(-EDGE);
  for (const item of await page.getByRole("menuitem").all()) await expect(item).toBeInViewport();
}

const sameLine = (a: CardCell, b: CardCell) => a.top < b.bottom - 1 && b.top < a.bottom - 1;
const followsOnLine = (previous: CardCell, cell: CardCell) => sameLine(previous, cell) && cell.left >= previous.right - 1;

test("TC-001 tracked-item cards run edge to edge", async ({ page, context }) => {
  await mockApi(context, { items: allItems });
  await page.setViewportSize({ width: 320, height: 800 });
  await page.goto("/");
  const table = page.getByRole("table", { name: "Tracked items" });
  await expect(table.getByRole("button", { name: longTitle })).toBeVisible();
  for (const width of cardWidths) {
    await page.setViewportSize({ width, height: 800 });
    for (const tab of tabs) {
      await page.getByRole("tab", { name: tab }).click();
      const rows = table.locator("tbody tr");
      await expect(rows).toHaveCount(tab === "Amazon" ? 3 : 2);
      await expectEdgeToEdge(page.locator(".tracked-table-container"), `${width}px ${tab} list`);
      for (const [index, row] of (await rows.all()).entries()) await expectEdgeToEdge(row, `${width}px ${tab} card ${index}`);
      await expectNoHorizontalScroll(page);
    }
  }
});

test("TC-002 card content stays inset and readable", async ({ page, context }) => {
  await mockApi(context, { items: allItems, itemDetails });
  await page.setViewportSize({ width: 320, height: 800 });
  await page.goto("/");
  const table = page.getByRole("table", { name: "Tracked items" });
  await expect(table.getByRole("button", { name: longTitle })).toBeVisible();
  for (const width of [320, 775, 1055]) {
    await page.setViewportSize({ width, height: 800 });
    for (const tab of tabs) {
      await page.getByRole("tab", { name: tab }).click();
      const rows = table.locator("tbody tr");
      await expect(rows).toHaveCount(tab === "Amazon" ? 3 : 2);
      for (const [index, row] of (await rows.all()).entries()) {
        const what = `${width}px ${tab} card ${index}`;
        for (const label of cardLabels[tab]) {
          const cell = row.locator(`td[data-label="${label}"]`);
          await expect(cell, `${what}: ${label}`).toBeVisible();
          expect(await cell.evaluate((element) => window.layoutProbe.labelShown(element)), `${what}: "${label}" label shown`).toBe(true);
        }
        if (tab === "LeBoncoin") await expect(row.locator('td[data-label="Amazon second-hand offer"]')).toHaveCount(0);
        await expect(row.locator(".item-title")).toBeVisible();
        await expect(row.getByRole("button", { name: "Options", exact: true })).toBeVisible();
        expect(await row.evaluate((element) => window.layoutProbe.insetProblems(element)), `${what}: content inset`).toEqual([]);
      }
      // Opening the item and its Options menu work.
      const title = (await rows.first().locator(".item-title").textContent())!.trim();
      await rows.first().locator(".item-title").click();
      await expect(page.getByRole("heading", { level: 1, name: title, exact: true })).toBeVisible();
      await page.goBack();
      await expect(page.getByRole("tab", { name: tab })).toHaveAttribute("aria-selected", "true");
      await rows.first().getByRole("button", { name: "Options", exact: true }).click();
      await expect(page.getByRole("menuitem", { name: "View details", exact: true })).toBeVisible();
      await expectMenuOnScreen(page);
      await page.keyboard.press("Escape");
      await expect(page.getByRole("menu")).toBeHidden();
    }
  }
});

test("TC-003 card layout and full width switch together at 1056px", async ({ page, context }) => {
  await mockApi(context, { items: allItems });
  await page.setViewportSize({ width: 1055, height: 800 });
  await page.goto("/");
  const table = page.getByRole("table", { name: "Tracked items" });
  const search = page.getByRole("textbox", { name: "Search tracked items" });
  await page.getByRole("tab", { name: "LeBoncoin" }).click();
  await search.fill("bicycle");
  for (const width of [1055, 1056, 1055, 360, 775]) {
    await page.setViewportSize({ width, height: 800 });
    await expect(page.getByRole("tab", { name: "LeBoncoin" })).toHaveAttribute("aria-selected", "true");
    await expect(search).toHaveValue("bicycle");
    await expect(table.getByRole("button", { name: "LeBoncoin bicycle" })).toBeVisible();
    await expect(table.getByRole("button", { name: sampleLeboncoin.title })).toHaveCount(0);
    const head = await table.locator("thead").evaluate((element) => ({
      shown: !window.layoutProbe.hidden(element),
      display: getComputedStyle(element).display,
    }));
    const container = page.locator(".tracked-table-container");
    if (width >= 1056) {
      expect(head.display, `${width}px header row`).toBe("table-header-group");
      await expect(page.getByRole("columnheader", { name: "Prices" })).toBeVisible();
      await expect(page.getByRole("columnheader", { name: "Status" })).toBeVisible();
      const gaps = await boxGaps(container);
      expect(Math.abs(gaps.left - DESKTOP_TABLE_OFFSET), `${width}px table left offset ${gaps.left}`).toBeLessThanOrEqual(1);
      expect(Math.abs(gaps.right - DESKTOP_TABLE_OFFSET), `${width}px table right offset ${gaps.right}`).toBeLessThanOrEqual(1);
      expect(gaps.left).toBeGreaterThan(16);
    } else {
      expect(head.shown, `${width}px header row hidden`).toBe(false);
      await expectEdgeToEdge(container, `${width}px list`);
      for (const row of await table.locator("tbody tr").all()) await expectEdgeToEdge(row, `${width}px card`);
    }
    await expectNoHorizontalScroll(page);
  }
});

test("TC-004 no-match message inside the full-width list", async ({ page, context }) => {
  await mockApi(context, { items: allItems });
  await page.setViewportSize({ width: 360, height: 800 });
  await page.goto("/");
  const table = page.getByRole("table", { name: "Tracked items" });
  const search = page.getByRole("textbox", { name: "Search tracked items" });
  await expect(table.getByRole("button", { name: longTitle })).toBeVisible();
  for (const width of [360, 775]) {
    await page.setViewportSize({ width, height: 800 });
    for (const tab of tabs) {
      const what = `${width}px ${tab}`;
      await page.getByRole("tab", { name: tab }).click();
      await search.fill("nothing-matches");
      const message = page.locator(".tracked-table-container").getByText("No items match your search.");
      await expect(message).toBeVisible();
      await expectEdgeToEdge(page.locator(".tracked-table-container"), `${what} list`);
      const gaps = await edgeGaps(message);
      expect(gaps.left, `${what}: message left inset`).toBeGreaterThanOrEqual(4);
      expect(gaps.left, `${what}: message left gap`).toBeLessThanOrEqual(GUTTER);
      expect(gaps.right, `${what}: message right inset`).toBeGreaterThanOrEqual(4);
      await expectNoHorizontalScroll(page);
      await search.fill("");
      const rows = table.locator("tbody tr");
      await expect(rows).toHaveCount(tab === "Amazon" ? 3 : 2);
      for (const row of await rows.all()) await expectEdgeToEdge(row, `${what} card after clearing the search`);
    }
  }
});

test("TC-005 card values side by side", async ({ page, context }) => {
  await mockApi(context, { items: sampleItems });
  await page.setViewportSize({ width: 775, height: 800 });
  await page.goto("/");
  const table = page.getByRole("table", { name: "Tracked items" });
  await expect(table.getByRole("button", { name: sampleAmazon.title })).toBeVisible();
  for (const width of [775, 1055]) {
    await page.setViewportSize({ width, height: 800 });
    for (const tab of tabs) {
      await page.getByRole("tab", { name: tab }).click();
      const rows = table.locator("tbody tr");
      await expect(rows).toHaveCount(tab === "Amazon" ? 2 : 1);
      for (const [index, row] of (await rows.all()).entries()) {
        const what = `${width}px ${tab} card ${index}`;
        const card = await row.evaluate((element) => window.layoutProbe.cells(element));
        const cell = (label: string) => card.cells.find((candidate) => candidate.label === label)!;
        const labels = cardLabels[tab];
        for (const [position, first] of labels.entries()) {
          for (const second of labels.slice(position + 1)) {
            expect(sameLine(cell(first), cell(second)), `${what}: ${first} and ${second} on the same line`).toBe(true);
          }
          expect(cell(first).labelShown, `${what}: ${first} labelled`).toBe(true);
        }
        // A value starts a new line only when it does not fit in the space left on the previous one.
        card.cells.forEach((current, position) => {
          if (position === 0 || followsOnLine(card.cells[position - 1], current)) return;
          const free = card.right - card.cells[position - 1].right;
          expect(free, `${what}: free space before value ${current.label || position} (width ${current.width})`).toBeLessThan(current.width + 1);
        });
      }
    }
  }
});

test("TC-006 phone card values in reading and focus order", async ({ page, context }) => {
  await mockApi(context, { items: allItems });
  await page.setViewportSize({ width: 320, height: 640 });
  await page.goto("/");
  const table = page.getByRole("table", { name: "Tracked items" });
  await expect(table.getByRole("button", { name: longTitle })).toBeVisible();
  for (const tab of tabs) {
    await page.getByRole("tab", { name: tab }).click();
    for (const [index, row] of (await table.locator("tbody tr").all()).entries()) {
      const what = `${tab} card ${index}`;
      const card = await row.evaluate((element) => window.layoutProbe.cells(element));
      card.cells.forEach((cell, position) => {
        const name = `${what} value ${cell.label || position}`;
        expect(cell.visible, `${name} visible`).toBe(true);
        expect(cell.clipped, `${name} clipped`).toBe(false);
        expect(cell.left, `${name} left`).toBeGreaterThanOrEqual(-EDGE);
        expect(cell.right, `${name} right`).toBeLessThanOrEqual(320 + EDGE);
        if (cell.label) expect(cell.labelShown, `${name} labelled`).toBe(true);
        if (position === 0) return;
        // DOM order is the visual order: right of the previous value, or below every previous value.
        const below = card.cells.slice(0, position).every((previous) => cell.top >= previous.bottom - 1);
        expect(followsOnLine(card.cells[position - 1], cell) || below, `${name} in reading order`).toBe(true);
      });
      expect(await row.evaluate((element) => window.layoutProbe.insetProblems(element)), `${what}: content inset`).toEqual([]);
      await expect(row.getByRole("button", { name: "Options", exact: true })).toBeVisible();
    }
    await expectNoHorizontalScroll(page);
  }

  // Keyboard: Tab from the search field goes through each card's title and Options button in order.
  await page.getByRole("tab", { name: "Amazon" }).click();
  const rowCount = await table.locator("tbody tr").count();
  await page.getByRole("textbox", { name: "Search tracked items" }).focus();
  const stops: { kind: string; row: number; top: number; inViewport: boolean; focusVisible: boolean }[] = [];
  for (let press = 0; press < rowCount * 4 + 4; press += 1) {
    await page.keyboard.press("Tab");
    const stop = await page.evaluate(() => {
      const element = document.activeElement as HTMLElement;
      const rows = [...document.querySelectorAll('table[aria-label="Tracked items"] tbody tr')];
      const row = element.closest("tbody tr");
      const kind = element.classList.contains("item-title") ? "title"
        : element.closest(".row-actions") && (element.hasAttribute("aria-haspopup") || element.getAttribute("aria-label") === "Options") ? "options"
          : "other";
      const box = element.getBoundingClientRect();
      const style = getComputedStyle(element);
      return {
        kind,
        row: row ? rows.indexOf(row) : -1,
        top: box.top + window.scrollY,
        inViewport: box.left >= -0.5 && box.top >= -0.5 && box.right <= document.documentElement.clientWidth + 0.5 && box.bottom <= window.innerHeight + 0.5,
        focusVisible: (style.outlineStyle !== "none" && parseFloat(style.outlineWidth) > 0) || style.boxShadow !== "none",
      };
    });
    if (stop.row < 0 && stops.some((previous) => previous.row >= 0)) break;
    stops.push(stop);
  }
  const sequence = stops.filter((stop) => stop.kind !== "other");
  expect(sequence.map((stop) => `${stop.kind} ${stop.row}`)).toEqual(
    Array.from({ length: rowCount }, (_, row) => [`title ${row}`, `options ${row}`]).flat(),
  );
  sequence.forEach((stop, index) => {
    expect(stop.inViewport, `${stop.kind} ${stop.row} fully in the viewport when focused`).toBe(true);
    expect(stop.focusVisible, `${stop.kind} ${stop.row} focus indicator`).toBe(true);
    if (index > 0) expect(stop.top, `${stop.kind} ${stop.row} below the previous stop`).toBeGreaterThanOrEqual(sequence[index - 1].top - 1);
  });

  // Enter on the Options button opens the row menu within the screen.
  await table.locator("tbody tr").first().getByRole("button", { name: "Options", exact: true }).focus();
  await page.keyboard.press("Enter");
  await expectMenuOnScreen(page);
  await expectNoHorizontalScroll(page);
});

test("TC-007 elements above the list use the full width", async ({ page, context }) => {
  await mockApi(context, { items: allItems });
  await page.setViewportSize({ width: 360, height: 800 });
  await page.goto("/");
  const table = page.getByRole("table", { name: "Tracked items" });
  const search = page.getByRole("textbox", { name: "Search tracked items" });
  await expect(table.getByRole("button", { name: longTitle })).toBeVisible();
  for (const width of [360, 775, 1055]) {
    await page.setViewportSize({ width, height: 800 });
    // Heading rows (title, item count, Refresh all prices, Add item): the nearest content to each edge is within 16 px.
    const heading = [
      ["title", page.getByRole("heading", { level: 1, name: "Tracked items" })],
      ["item count", page.locator(".page-description")],
      ["Refresh all prices", page.getByRole("button", { name: "Refresh all prices" })],
      ["Add item", page.locator(".page-heading-actions").getByRole("button", { name: "Add item" })],
    ] as const;
    const headingGaps = [];
    for (const [name, locator] of heading) {
      const gaps = await edgeGaps(locator);
      expectNotTouching(gaps, `${width}px ${name}`);
      headingGaps.push(gaps);
    }
    expectGapsWithinGutter({
      left: Math.min(...headingGaps.map((gaps) => gaps.left)),
      right: Math.min(...headingGaps.map((gaps) => gaps.right)),
    }, `${width}px heading`);
    // Full-width blocks: tab bar, search field, Amazon-stop warning.
    const amazonTab = await edgeGaps(page.getByRole("tab", { name: "Amazon" }));
    const leboncoinTab = await edgeGaps(page.getByRole("tab", { name: "LeBoncoin" }));
    expectGapsWithinGutter({ left: amazonTab.left, right: leboncoinTab.right }, `${width}px tab bar`);
    expectNotTouching({ left: amazonTab.left, right: leboncoinTab.right }, `${width}px tab bar`);
    await expectFullWidth(search, `${width}px search field`);
    expectNotTouching(await edgeGaps(search), `${width}px search field`);
    await expectFullWidth(page.locator(".cds--inline-notification").filter({ hasText: "Amazon checks were stopped" }), `${width}px Amazon-stop warning`);
    await scanPage(page);
    await expectNoHorizontalScroll(page);
    // The search still filters the selected tab.
    await search.fill("Sony");
    await expect(table.getByRole("button", { name: sampleAmazon.title })).toBeVisible();
    await expect(table.getByRole("button", { name: longTitle })).toHaveCount(0);
    await search.fill("");
    await expect(table.getByRole("button", { name: longTitle })).toBeVisible();
  }
});

for (const state of [
  { name: "empty", options: { items: [] }, target: (page: Page) => page.locator(".empty-state").filter({ hasText: "No items tracked yet" }) },
  { name: "loading", options: { itemsReply: "hold" }, target: (page: Page) => page.locator(".empty-state").filter({ hasText: "Loading tracked items" }) },
  { name: "error", options: { itemsReply: "error" }, target: (page: Page) => page.locator(".cds--inline-notification").filter({ hasText: "Could not load tracked items" }) },
] satisfies { name: string; options: MockOptions; target: (page: Page) => Locator }[]) {
  test(`TC-008 tracked items ${state.name} state uses the full width`, async ({ page, context }) => {
    await mockApi(context, state.options);
    await page.setViewportSize({ width: 360, height: 800 });
    await page.goto("/");
    await expect(state.target(page)).toBeVisible();
    for (const width of [360, 775]) {
      await page.setViewportSize({ width, height: 800 });
      await expectFullWidth(state.target(page), `${width}px ${state.name} state`);
      await scanPage(page);
      await expectNoHorizontalScroll(page);
    }
    await abortHeldRoutes();
    await context.unrouteAll({ behavior: "ignoreErrors" });
  });
}

test("TC-009 item details use the full width", async ({ page, context }) => {
  await mockApi(context, { itemDetails });
  const pages = [
    {
      id: amazon.id, title: longTitle, source: "Open on Amazon",
      blocks: (current: Page) => [
        current.locator(".detail-summary"),
        current.locator(".detections-list").first(),
        current.locator(".detections-list").nth(1),
        current.locator(".detail-notification").filter({ hasText: "Amazon checks of this item are stopped" }),
      ],
    },
    {
      id: leboncoin.id, title: "LeBoncoin bicycle", source: "Open on LeBoncoin",
      blocks: (current: Page) => [
        current.locator(".detail-summary"),
        current.locator(".detections-list"),
        current.getByRole("textbox", { name: "Purchase goal" }),
        current.locator(".ai-review .detail-notification"),
        current.locator(".ai-review-history .cds--accordion__item").first(),
      ],
    },
  ];
  for (const width of [360, 775, 1055]) {
    await page.setViewportSize({ width, height: 800 });
    for (const detail of pages) {
      const what = `${width}px ${detail.id}`;
      await page.goto(`/items/${detail.id}`);
      const heading = page.getByRole("heading", { level: 1, name: detail.title, exact: true });
      await expect(heading).toBeVisible();
      if (detail.id === leboncoin.id) await expect(page.locator(".ai-review-history")).toBeVisible();
      for (const [index, block] of detail.blocks(page).entries()) await expectFullWidth(block, `${what} block ${index}`);
      expect((await edgeGaps(page.locator(".detail-breadcrumb"))).left, `${what} breadcrumb left gap`).toBeLessThanOrEqual(GUTTER);
      expect((await edgeGaps(heading)).left, `${what} title left gap`).toBeLessThanOrEqual(GUTTER);
      const actions = page.locator(".detail-actions");
      await expect(actions.getByRole("link", { name: detail.source })).toBeVisible();
      const first = await edgeGaps(actions.getByRole("button", { name: "Refresh price" }));
      const last = await edgeGaps(actions.getByRole("button", { name: "Delete" }));
      expect(Math.min(first.left, last.right), `${what} actions reach a gutter`).toBeLessThanOrEqual(GUTTER);
      await expectPageFullWidth(page, what);
      // Long title and purchase goal wrap.
      expect(await heading.evaluate((element) => element.scrollWidth <= element.clientWidth + 1), `${what} title wraps`).toBe(true);
      if (detail.id === leboncoin.id) {
        const goal = page.getByRole("textbox", { name: "Purchase goal" });
        expect(await goal.evaluate((element) => element.scrollWidth <= element.clientWidth + 1), `${what} purchase goal wraps`).toBe(true);
      }
      await expectNoHorizontalScroll(page);
    }
  }
});

test("TC-009 item details loading state uses the full width", async ({ page, context }) => {
  await mockApi(context, { itemReply: "hold" });
  await page.setViewportSize({ width: 775, height: 800 });
  await page.goto(`/items/${leboncoin.id}`);
  const loading = page.locator(".cds--inline-loading").filter({ hasText: "Loading item details" });
  await expect(loading).toBeVisible();
  // The indicator's box spans the content width; its spinner starts at the gutter.
  expectGapsWithinGutter(await boxGaps(loading), "loading indicator");
  expect((await boxGaps(loading.locator(".cds--inline-loading__animation"))).left, "loading spinner left gap").toBeLessThanOrEqual(GUTTER);
  await scanPage(page);
  await expectNoHorizontalScroll(page);
  await abortHeldRoutes();
  await context.unrouteAll({ behavior: "ignoreErrors" });
});

test("TC-009 item details error state uses the full width", async ({ page, context }) => {
  await mockApi(context);
  await page.setViewportSize({ width: 775, height: 800 });
  await page.goto("/items/unknown-item");
  const notification = page.locator(".cds--inline-notification").filter({ hasText: "Could not load item" });
  await expect(notification).toBeVisible();
  await expectFullWidth(notification, "error notification");
  await expect(page.getByRole("button", { name: "Back to tracked items" })).toBeVisible();
  await scanPage(page);
  await expectNoHorizontalScroll(page);
});

test("TC-010 administrator users list edge to edge", async ({ page, context }) => {
  await mockApi(context, { session: adminSession, users: [{ username: "alice", role: "user", lastLoginAt: timestamp }] });
  await page.setViewportSize({ width: 360, height: 800 });
  await page.goto("/");
  const table = page.getByRole("table", { name: "Users" });
  await expect(table.getByText("alice")).toBeVisible();
  const addUser = page.getByRole("button", { name: "Add user" });
  for (const width of [360, 775, 1055]) {
    await page.setViewportSize({ width, height: 800 });
    const row = table.locator("tbody tr");
    await expectEdgeToEdge(row, `${width}px user card`);
    const card = await row.evaluate((element) => window.layoutProbe.cells(element));
    const cell = (label: string) => card.cells.find((candidate) => candidate.label === label)!;
    for (const label of ["Username", "Last login"]) {
      expect(cell(label).visible, `${width}px ${label} visible`).toBe(true);
      expect(cell(label).clipped, `${width}px ${label} clipped`).toBe(false);
      expect(cell(label).labelShown, `${width}px ${label} labelled`).toBe(true);
    }
    expect(await row.evaluate((element) => window.layoutProbe.insetProblems(element)), `${width}px user card inset`).toEqual([]);
    if (width >= 775) expect(sameLine(cell("Username"), cell("Last login")), `${width}px Username and Last login on the same line`).toBe(true);
    const title = await edgeGaps(page.getByRole("heading", { level: 1, name: "Users" }));
    expectNotTouching(title, `${width}px title`);
    expect(title.left, `${width}px title left gap`).toBeLessThanOrEqual(GUTTER);
    const button = await edgeGaps(addUser);
    expectNotTouching(button, `${width}px Add user`);
    expect(Math.min(button.left, button.right), `${width}px Add user gap`).toBeLessThanOrEqual(GUTTER);
    await scanPage(page);
    await expectNoHorizontalScroll(page);
  }
  await addUser.click();
  const dialog = page.getByRole("dialog", { name: "Add user" });
  await expect(dialog).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
});

test("TC-010 administrator long username wraps inside its card at 320px", async ({ page, context }) => {
  await mockApi(context, { session: adminSession, users: [{ username: longUsername, role: "user", lastLoginAt: timestamp }] });
  await page.setViewportSize({ width: 320, height: 800 });
  await page.goto("/");
  const row = page.getByRole("table", { name: "Users" }).locator("tbody tr");
  await expect(row.getByText(longUsername)).toBeVisible();
  await expectEdgeToEdge(row, "user card");
  const username = await row.locator('td[data-label="Username"]').evaluate((element) => {
    const range = document.createRange();
    range.selectNodeContents(element);
    return { lines: range.getClientRects().length, right: element.getBoundingClientRect().right };
  });
  expect(username.lines, "long username wraps").toBeGreaterThan(1);
  expect(username.right).toBeLessThanOrEqual(320 + EDGE);
  const card = await row.evaluate((element) => window.layoutProbe.cells(element));
  const lastLogin = card.cells.find((cell) => cell.label === "Last login")!;
  expect(lastLogin.visible, "Last login visible").toBe(true);
  expect(lastLogin.clipped, "Last login clipped").toBe(false);
  expect(lastLogin.labelShown, "Last login labelled").toBe(true);
  expect(await row.evaluate((element) => window.layoutProbe.insetProblems(element)), "user card inset").toEqual([]);
  await expectNoHorizontalScroll(page);
});

test("TC-011 login page uses the full width", async ({ page, context }) => {
  await mockApi(context, { session: { mode: "protected", user: null, version: "v1" } });
  await page.setViewportSize({ width: 360, height: 800 });
  await page.goto("/");
  const fields = [
    ["username field", page.getByRole("textbox", { name: "Username" })],
    ["password field", page.locator("#login-password")],
    ["Log in button", page.getByRole("button", { name: "Log in", exact: true })],
  ] as const;
  await expect(fields[2][1]).toBeVisible();
  for (const width of [360, 775, 1055]) {
    await page.setViewportSize({ width, height: 800 });
    for (const [name, locator] of fields) {
      const gaps = await edgeGaps(locator);
      expectGapsWithinGutter(gaps, `${width}px ${name}`);
      expectNotTouching(gaps, `${width}px ${name}`);
    }
    await scanPage(page);
    await expectNoHorizontalScroll(page);
  }
  await page.setViewportSize({ width: 775, height: 800 });
  await fields[0][1].fill("alice");
  await fields[1][1].fill("wrong-password-123");
  await fields[2][1].click();
  const notification = page.locator(".cds--inline-notification").filter({ hasText: "Could not log in" });
  await expect(notification).toBeVisible();
  await expectFullWidth(notification, "775px login error");
  await scanPage(page);
  await expectNoHorizontalScroll(page);
});

test("TC-012 desktop tracked items layout unchanged", async ({ page, context }) => {
  await mockApi(context, { items: allItems });
  await page.setViewportSize({ width: 1056, height: 900 });
  await page.goto("/");
  const table = page.getByRole("table", { name: "Tracked items" });
  await expect(table.getByRole("button", { name: longTitle })).toBeVisible();
  for (const width of [1056, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    for (const tab of tabs) {
      const what = `${width}px ${tab}`;
      await page.getByRole("tab", { name: tab }).click();
      const shell = await boxGaps(page.locator("#main-content"));
      expect(Math.abs(shell.left - DESKTOP_PAGE_OFFSET), `${what} page left offset ${shell.left}`).toBeLessThanOrEqual(1);
      expect(Math.abs(shell.right - DESKTOP_PAGE_OFFSET), `${what} page right offset ${shell.right}`).toBeLessThanOrEqual(1);
      const list = await boxGaps(page.locator(".tracked-table-container"));
      expect(Math.abs(list.left - DESKTOP_TABLE_OFFSET), `${what} table left offset ${list.left}`).toBeLessThanOrEqual(1);
      expect(Math.abs(list.right - DESKTOP_TABLE_OFFSET), `${what} table right offset ${list.right}`).toBeLessThanOrEqual(1);
      const searchRow = await boxGaps(page.locator(".tracked-table-search"));
      expect(Math.abs(list.left - searchRow.left), `${what} table lines up with the search row`).toBeLessThanOrEqual(1);
      expect(Math.abs(list.right - searchRow.right), `${what} table lines up with the search row`).toBeLessThanOrEqual(1);
      expect(await table.locator("thead").evaluate((element) => getComputedStyle(element).display)).toBe("table-header-group");
      for (const name of ["Item", "Prices", "Status"]) await expect(page.getByRole("columnheader", { name, exact: true })).toBeVisible();
      await expectNoHorizontalScroll(page);
    }
  }
});

test("TC-012 desktop item details layout unchanged", async ({ page, context }) => {
  await mockApi(context, { itemDetails });
  for (const width of [1056, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    for (const detail of [{ id: amazon.id, title: longTitle }, { id: leboncoin.id, title: "LeBoncoin bicycle" }]) {
      const what = `${width}px ${detail.id}`;
      await page.goto(`/items/${detail.id}`);
      await expect(page.getByRole("heading", { level: 1, name: detail.title, exact: true })).toBeVisible();
      for (const selector of ["#main-content", ".detail-summary"]) {
        const gaps = await boxGaps(page.locator(selector));
        expect(Math.abs(gaps.left - DESKTOP_PAGE_OFFSET), `${what} ${selector} left offset ${gaps.left}`).toBeLessThanOrEqual(1);
        expect(Math.abs(gaps.right - DESKTOP_PAGE_OFFSET), `${what} ${selector} right offset ${gaps.right}`).toBeLessThanOrEqual(1);
      }
      await expectNoHorizontalScroll(page);
    }
  }
});

test("TC-012 desktop administrator page layout unchanged", async ({ page, context }) => {
  await mockApi(context, { session: adminSession, users: [{ username: "alice", role: "user", lastLoginAt: timestamp }] });
  await page.setViewportSize({ width: 1056, height: 900 });
  await page.goto("/");
  const table = page.getByRole("table", { name: "Users" });
  await expect(table.getByText("alice")).toBeVisible();
  for (const width of [1056, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    for (const selector of ["#main-content", ".responsive-table-container"]) {
      const gaps = await boxGaps(page.locator(selector));
      expect(Math.abs(gaps.left - DESKTOP_PAGE_OFFSET), `${width}px ${selector} left offset ${gaps.left}`).toBeLessThanOrEqual(1);
      expect(Math.abs(gaps.right - DESKTOP_PAGE_OFFSET), `${width}px ${selector} right offset ${gaps.right}`).toBeLessThanOrEqual(1);
    }
    expect(await table.locator("thead").evaluate((element) => getComputedStyle(element).display)).toBe("table-header-group");
    for (const name of ["Username", "Last login"]) await expect(page.getByRole("columnheader", { name, exact: true })).toBeVisible();
    await expectNoHorizontalScroll(page);
  }
});

test("TC-012 desktop login page layout unchanged", async ({ page, context }) => {
  await mockApi(context, { session: { mode: "protected", user: null, version: "v1" } });
  await page.setViewportSize({ width: 1056, height: 900 });
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Log in", exact: true })).toBeVisible();
  for (const width of [1056, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    const form = await boxGaps(page.locator(".login-form"));
    expect(Math.abs(form.width - LOGIN_FORM_WIDTH), `${width}px login form width ${form.width}`).toBeLessThanOrEqual(1);
    expect(Math.abs(form.left - form.right), `${width}px login form centred`).toBeLessThanOrEqual(1);
    await expectNoHorizontalScroll(page);
  }
});

async function expectHeaderBaseline(page: Page, what: string) {
  const header = page.locator("header.cds--header");
  const viewport = await page.evaluate(() => document.documentElement.clientWidth);
  const box = (await header.boundingBox())!;
  expect(Math.abs(box.x), `${what} header left`).toBeLessThanOrEqual(1);
  expect(Math.abs(box.y), `${what} header top`).toBeLessThanOrEqual(1);
  expect(Math.abs(box.width - viewport), `${what} header width`).toBeLessThanOrEqual(1);
  expect(Math.abs(box.height - HEADER_HEIGHT), `${what} header height`).toBeLessThanOrEqual(1);
  const brand = (await header.locator(".brand-mark").boundingBox())!;
  expect(Math.abs(brand.x - HEADER_BRAND_LEFT), `${what} brand left ${brand.x}`).toBeLessThanOrEqual(1);
  const items = [
    header.locator("a.cds--header__name"),
    header.getByRole("link", { name: "Log in" }),
    header.getByRole("button", { name: "Add item" }),
    header.getByRole("button", { name: "Application menu", exact: true }),
  ];
  const boxes = [];
  for (const item of items) if (await item.count()) boxes.push((await item.boundingBox())!);
  expect(Math.abs(boxes[0].x), `${what} header name left`).toBeLessThanOrEqual(1);
  boxes.forEach((item, index) => {
    expect(item.y, `${what} header item ${index} top`).toBeGreaterThanOrEqual(-1);
    expect(item.y + item.height, `${what} header item ${index} bottom`).toBeLessThanOrEqual(HEADER_HEIGHT + 1);
    if (index > 0) expect(item.x, `${what} header item ${index} after the previous one`).toBeGreaterThanOrEqual(boxes[index - 1].x + boxes[index - 1].width - 1);
  });
  const menu = boxes[boxes.length - 1];
  expect(Math.abs(menu.x + menu.width - viewport), `${what} menu trigger at the right edge`).toBeLessThanOrEqual(1);
}

async function expectModalBaseline(page: Page, name: string, width: number) {
  const dialog = page.getByRole("dialog", { name });
  await expect(dialog).toBeVisible();
  const boxes = await page.locator(".cds--modal.is-visible").evaluate((overlay) => {
    const container = overlay.querySelector(".cds--modal-container")!.getBoundingClientRect();
    const box = overlay.getBoundingClientRect();
    return { overlayLeft: box.left, overlayWidth: box.width, left: container.left, width: container.width };
  });
  const expectedWidth = boxes.overlayWidth * modalWidthRatio(width);
  expect(Math.abs(boxes.width - expectedWidth), `${width}px ${name} modal width ${boxes.width}`).toBeLessThanOrEqual(1);
  expect(Math.abs(boxes.left - (boxes.overlayLeft + (boxes.overlayWidth - expectedWidth) / 2)), `${width}px ${name} modal left ${boxes.left}`).toBeLessThanOrEqual(1);
  for (const button of await page.locator(".cds--modal.is-visible .cds--modal-footer button").all()) await expect(button).toBeInViewport();
  await expectNoHorizontalScroll(page);
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
}

test("TC-013 header and modals unchanged below desktop", async ({ page, context }) => {
  await mockApi(context, { items: allItems });
  await page.setViewportSize({ width: 360, height: 800 });
  await page.goto("/");
  const table = page.getByRole("table", { name: "Tracked items" });
  await expect(table.getByRole("button", { name: longTitle })).toBeVisible();
  for (const width of [360, 775]) {
    await page.setViewportSize({ width, height: 800 });
    await expectHeaderBaseline(page, `${width}px`);
    await page.getByRole("button", { name: "Application menu", exact: true }).click();
    await expectMenuOnScreen(page);
    await page.keyboard.press("Escape");
    await expect(page.getByRole("menu")).toBeHidden();
    await page.locator(".page-heading-actions").getByRole("button", { name: "Add item" }).click();
    await expectModalBaseline(page, "Add tracked item", width);
    await table.locator("tbody tr").first().getByRole("button", { name: "Options", exact: true }).click();
    await page.getByRole("menuitem", { name: "Delete", exact: true }).click();
    await expectModalBaseline(page, "Delete tracked item", width);
    await openMenuEntry(page, "Settings");
    await expectModalBaseline(page, "Settings", width);
    await openMenuEntry(page, "Information");
    await expectModalBaseline(page, "Information", width);
  }
});

test("TC-013 administrator header and Add user modal unchanged below desktop", async ({ page, context }) => {
  await mockApi(context, { session: adminSession, users: [{ username: "alice", role: "user", lastLoginAt: timestamp }] });
  await page.setViewportSize({ width: 360, height: 800 });
  await page.goto("/");
  await expect(page.getByRole("table", { name: "Users" }).getByText("alice")).toBeVisible();
  for (const width of [360, 775]) {
    await page.setViewportSize({ width, height: 800 });
    await expectHeaderBaseline(page, `${width}px administrator`);
    await page.getByRole("button", { name: "Add user" }).click();
    await expectModalBaseline(page, "Add user", width);
  }
});

test("TC-013 Information modal with a long username at 400px", async ({ page, context }) => {
  await mockApi(context, { session: { mode: "protected", user: { username: longUsername, role: "user" }, version: "v1" }, items: allItems });
  await page.setViewportSize({ width: 400, height: 800 });
  await page.goto("/");
  await openMenuEntry(page, "Information");
  const dialog = page.getByRole("dialog", { name: "Information" });
  await expect(dialog).toContainText(longUsername);
  await expect(dialog).toContainText("v1");
  const layout = await page.locator(".cds--modal.is-visible").evaluate((overlay) => {
    const container = overlay.querySelector(".cds--modal-container")!.getBoundingClientRect();
    const [username, version] = [...overlay.querySelectorAll(".information-list dd")].map((dd) => {
      const range = document.createRange();
      range.selectNodeContents(dd);
      const text = range.getBoundingClientRect();
      const box = dd.getBoundingClientRect();
      return { lines: range.getClientRects().length, left: text.left, right: text.right, top: box.top, bottom: box.bottom };
    });
    return { left: container.left, right: container.right, username, version };
  });
  expect(layout.username.lines, "long username wraps").toBeGreaterThan(1);
  for (const value of [layout.username, layout.version]) {
    expect(value.left).toBeGreaterThanOrEqual(layout.left);
    expect(value.right).toBeLessThanOrEqual(layout.right);
  }
  expect(layout.version.top, "version below the username").toBeGreaterThanOrEqual(layout.username.bottom - 0.5);
  await expectModalBaseline(page, "Information", 400);
});
