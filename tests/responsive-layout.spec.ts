import { expect, test, type BrowserContext, type Page } from "@playwright/test";

// FR-RESPONSIVE-001..013 (issue #140). Synthetic data only.
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

async function mockApi(context: BrowserContext, options: { session?: unknown; items?: unknown[] } = {}) {
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== "http://127.0.0.1:4173") return route.abort();
    if (!url.pathname.startsWith("/api/")) return route.continue();
    const path = url.pathname;
    if (path === "/api/v1/auth/session") return route.fulfill({ json: options.session ?? openSession });
    if (path === "/api/v1/items") return route.fulfill({ json: { items: options.items ?? [amazon, leboncoin] } });
    if (path === `/api/v1/items/${amazon.id}`) return route.fulfill({ json: { ...amazon, aiReview: null } });
    if (path === `/api/v1/items/${leboncoin.id}`) return route.fulfill({ json: { ...leboncoin, aiReview } });
    if (path === "/api/v1/admin/users") {
      return route.fulfill({ json: { users: [{ username: longUsername, role: "user", lastLoginAt: timestamp }] } });
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
