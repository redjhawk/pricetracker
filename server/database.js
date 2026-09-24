import fs from "node:fs";
import path from "node:path";
import Database from "better-sqlite3";
import { config, databasePath } from "./config.js";

fs.mkdirSync(path.dirname(databasePath), { recursive: true });

export const db = new Database(databasePath);
db.pragma("journal_mode = WAL");
db.pragma("foreign_keys = ON");
db.pragma("busy_timeout = 5000");

db.exec(`
  CREATE TABLE IF NOT EXISTS items (
    id TEXT PRIMARY KEY,
    asin TEXT NOT NULL,
    marketplace TEXT NOT NULL,
    canonical_url TEXT NOT NULL UNIQUE,
    url TEXT NOT NULL,
    title TEXT,
    thumbnail_url TEXT,
    next_check_at TEXT,
    added_at TEXT NOT NULL
  );

  CREATE TABLE IF NOT EXISTS price_observations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    amount_cents INTEGER NOT NULL CHECK(amount_cents >= 0),
    currency TEXT NOT NULL DEFAULT 'EUR' CHECK(currency = 'EUR'),
    observed_at TEXT NOT NULL
  );

  CREATE INDEX IF NOT EXISTS price_observations_item_time
    ON price_observations(item_id, observed_at DESC);

  CREATE TABLE IF NOT EXISTS collection_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    result TEXT NOT NULL CHECK(result IN ('success', 'request_error', 'price_not_found', 'unavailable')),
    attempted_at TEXT NOT NULL,
    message TEXT
  );

  CREATE INDEX IF NOT EXISTS collection_attempts_item_time
    ON collection_attempts(item_id, attempted_at DESC);

  CREATE INDEX IF NOT EXISTS items_next_check
    ON items(next_check_at);
`);

const selectItem = db.prepare("SELECT * FROM items WHERE id = ?");
const selectLatestPrice = db.prepare(
  "SELECT amount_cents, currency, observed_at FROM price_observations WHERE item_id = ? ORDER BY observed_at DESC, id DESC LIMIT 1",
);
const selectLastThree = db.prepare(
  "SELECT amount_cents, currency, observed_at FROM price_observations WHERE item_id = ? ORDER BY observed_at DESC, id DESC LIMIT 3",
);
const selectLastAttempt = db.prepare(
  "SELECT result, attempted_at, message FROM collection_attempts WHERE item_id = ? ORDER BY attempted_at DESC, id DESC LIMIT 1",
);

function mapObservation(row) {
  if (!row) return null;
  return { amountCents: row.amount_cents, currency: row.currency, timestamp: row.observed_at };
}

function statusFor(latestPrice, lastAttempt) {
  if (!lastAttempt) return "pending";
  if (lastAttempt.result === "unavailable") return "unavailable";
  if (lastAttempt.result !== "success") return "retrieval_error";
  if (!latestPrice) return "pending";
  const age = Date.now() - Date.parse(latestPrice.timestamp);
  return age > config.staleAfterHours * 60 * 60 * 1000 ? "stale" : "active";
}

export function getItem(id) {
  const row = selectItem.get(id);
  if (!row) return null;

  const latestPrice = mapObservation(selectLatestPrice.get(id));
  const lastAttemptRow = selectLastAttempt.get(id);
  const lastAttempt = lastAttemptRow
    ? {
        result: lastAttemptRow.result,
        timestamp: lastAttemptRow.attempted_at,
        message: lastAttemptRow.message,
      }
    : null;

  return {
    id: row.id,
    title: row.title,
    asin: row.asin,
    marketplace: row.marketplace,
    url: row.url,
    thumbnailUrl: row.thumbnail_url,
    status: statusFor(latestPrice, lastAttempt),
    latestPrice,
    lastThreeDetections: selectLastThree.all(id).map(mapObservation),
    lastAttempt,
    nextCheckAt: row.next_check_at,
    addedAt: row.added_at,
  };
}

export function listItems() {
  return db
    .prepare("SELECT id FROM items ORDER BY added_at DESC, id DESC")
    .all()
    .map(({ id }) => getItem(id));
}

export function insertItem({ id, asin, marketplace, canonicalUrl, url, nextCheckAt, addedAt }) {
  db.prepare(`
    INSERT INTO items (id, asin, marketplace, canonical_url, url, next_check_at, added_at)
    VALUES (?, ?, ?, ?, ?, ?, ?)
  `).run(id, asin, marketplace, canonicalUrl, url, nextCheckAt, addedAt);
}

export function setNextCheck(id, nextCheckAt) {
  db.prepare("UPDATE items SET next_check_at = ? WHERE id = ?").run(nextCheckAt, id);
}

export function updateItemMetadata(id, { title, thumbnailUrl }) {
  db.prepare("UPDATE items SET title = ?, thumbnail_url = ? WHERE id = ?").run(
    title ?? null,
    thumbnailUrl ?? null,
    id,
  );
}

export function recordSuccessfulCollection(id, { amountCents, title, thumbnailUrl, timestamp }) {
  const commit = db.transaction(() => {
    db.prepare("INSERT INTO price_observations (item_id, amount_cents, currency, observed_at) VALUES (?, ?, 'EUR', ?)")
      .run(id, amountCents, timestamp);
    db.prepare("INSERT INTO collection_attempts (item_id, result, attempted_at) VALUES (?, 'success', ?)")
      .run(id, timestamp);
    db.prepare("UPDATE items SET title = COALESCE(?, title), thumbnail_url = COALESCE(?, thumbnail_url) WHERE id = ?")
      .run(title, thumbnailUrl, id);
  });
  commit();
}

export function recordFailedCollection(id, result, message, timestamp) {
  db.prepare("INSERT INTO collection_attempts (item_id, result, attempted_at, message) VALUES (?, ?, ?, ?)")
    .run(id, result, timestamp, message ?? null);
}

export function deleteItem(id) {
  return db.prepare("DELETE FROM items WHERE id = ?").run(id).changes > 0;
}

function timestampHoursAgo(hours) {
  return new Date(Date.now() - hours * 60 * 60 * 1000).toISOString();
}

function seedDevelopmentItems(nextCheckAt) {
  if (!config.development || db.prepare("SELECT COUNT(*) AS total FROM items").get().total > 0) return;

  const samples = [
    {
      id: "sample-sony-headphones", asin: "B09XS7JWHH", marketplace: "amazon.de",
      url: "https://www.amazon.de/dp/B09XS7JWHH", title: "Sony WH-1000XM5 Wireless Noise Canceling Headphones",
      thumbnail: "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=120&h=120&fit=crop&auto=format",
      prices: [[27900, 3], [29900, 27], [29900, 51]], attempt: ["success", 3],
    },
    {
      id: "sample-kindle-paperwhite", asin: "B0CFPJYX4M", marketplace: "amazon.fr",
      url: "https://www.amazon.fr/dp/B0CFPJYX4M", title: "Kindle Paperwhite (16 Go) — Éclairage réglable chaud/froid",
      thumbnail: "https://images.unsplash.com/photo-1544716278-ca5e3f4abd8c?w=120&h=120&fit=crop&auto=format",
      prices: [[13999, 4], [13999, 28], [14999, 52]], attempt: ["success", 4],
    },
    {
      id: "sample-lego-land-rover", asin: "B07STGQK2S", marketplace: "amazon.es",
      url: "https://www.amazon.es/dp/B07STGQK2S", title: "LEGO Technic Land Rover Defender 42110",
      thumbnail: "https://images.unsplash.com/photo-1587654780291-39c9404d746b?w=120&h=120&fit=crop&auto=format",
      prices: [], attempt: null,
    },
    {
      id: "sample-nike-air-max", asin: "B07D9FKTTN", marketplace: "amazon.it",
      url: "https://www.amazon.it/dp/B07D9FKTTN", title: "Nike Air Max 270 Uomo",
      thumbnail: "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=120&h=120&fit=crop&auto=format",
      prices: [[9495, 60], [10995, 84], [10995, 108]], attempt: ["success", 60],
    },
    {
      id: "sample-samsung-monitor", asin: "B0CXMQZKNR", marketplace: "amazon.de",
      url: "https://www.amazon.de/dp/B0CXMQZKNR", title: "Samsung 27-Zoll-Monitor S27C432GAU",
      thumbnail: "https://images.unsplash.com/photo-1527443224154-c4a3942d3acf?w=120&h=120&fit=crop&auto=format",
      prices: [[21900, 8], [22900, 32], [24900, 56]], attempt: ["request_error", 2],
    },
    {
      id: "sample-title-unavailable", asin: "B0D2XWJ7XY", marketplace: "amazon.de",
      url: "https://www.amazon.de/dp/B0D2XWJ7XY", title: null, thumbnail: null,
      prices: [[4590, 120]], attempt: ["unavailable", 30],
    },
  ];

  const insert = db.prepare(`
    INSERT INTO items (id, asin, marketplace, canonical_url, url, title, thumbnail_url, next_check_at, added_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
  `);
  const addObservation = db.prepare(
    "INSERT INTO price_observations (item_id, amount_cents, currency, observed_at) VALUES (?, ?, 'EUR', ?)",
  );
  const addAttempt = db.prepare(
    "INSERT INTO collection_attempts (item_id, result, attempted_at, message) VALUES (?, ?, ?, ?)",
  );
  const seed = db.transaction(() => {
    for (const item of samples) {
      const addedAt = timestampHoursAgo(item.prices.at(-1)?.[1] ?? 12);
      insert.run(
        item.id, item.asin, item.marketplace,
        `https://${item.marketplace}/dp/${item.asin}`, item.url, item.title,
        item.thumbnail, nextCheckAt(new Date()), addedAt,
      );
      for (const [amountCents, hoursAgo] of item.prices) {
        addObservation.run(item.id, amountCents, timestampHoursAgo(hoursAgo));
      }
      if (item.attempt) {
        const [result, hoursAgo] = item.attempt;
        addAttempt.run(
          item.id,
          result,
          timestampHoursAgo(hoursAgo),
          result === "request_error" ? "Sample upstream request error" : null,
        );
      }
    }
  });

  seed();
  console.info(`Seeded ${samples.length} sample items into ${databasePath}`);
}

export { seedDevelopmentItems };
