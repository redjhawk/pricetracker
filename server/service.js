import { randomUUID } from "node:crypto";
import { collectAmazonPrice, parseAmazonUrl } from "./amazon-collector.js";
import { config } from "./config.js";
import {
  db,
  deleteItem,
  getItem,
  insertItem,
  listItems,
  recordFailedCollection,
  recordSuccessfulCollection,
  setNextCheck,
} from "./database.js";

export class ServiceError extends Error {
  constructor(status, code, message) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

function datePartsInTimezone(date) {
  const parts = new Intl.DateTimeFormat("en-GB", {
    timeZone: config.timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  }).formatToParts(date);
  return Object.fromEntries(parts.filter((part) => part.type !== "literal").map(({ type, value }) => [type, Number(value)]));
}

function localSlotToUtc(year, month, day, hour, minute) {
  const requestedLocalAsUtc = Date.UTC(year, month - 1, day, hour, minute);
  const actual = datePartsInTimezone(new Date(requestedLocalAsUtc));
  const representedAsUtc = Date.UTC(actual.year, actual.month - 1, actual.day, actual.hour, actual.minute);
  return new Date(requestedLocalAsUtc - (representedAsUtc - requestedLocalAsUtc));
}

export function nextCheckAt(after = new Date()) {
  const local = datePartsInTimezone(after);
  const localMidnight = new Date(Date.UTC(local.year, local.month - 1, local.day));
  const scheduleSlots = config.checkTimes
    .map((slot) => slot.match(/^(\d{1,2}):(\d{2})$/))
    .filter(Boolean)
    .map((match) => ({ hour: Number(match[1]), minute: Number(match[2]) }))
    .filter(({ hour, minute }) => hour < 24 && minute < 60)
    .sort((a, b) => a.hour - b.hour || a.minute - b.minute);

  if (scheduleSlots.length === 0) throw new Error("PRICEFOLLOWER_CHECK_TIMES must contain valid HH:MM times.");

  for (let offset = 0; offset < 3; offset += 1) {
    const day = new Date(localMidnight.getTime() + offset * 24 * 60 * 60 * 1000);
    for (const slot of scheduleSlots) {
      const candidate = localSlotToUtc(day.getUTCFullYear(), day.getUTCMonth() + 1, day.getUTCDate(), slot.hour, slot.minute);
      if (candidate.getTime() > after.getTime()) return candidate.toISOString();
    }
  }

  throw new Error("Could not determine the next collection time.");
}

export const itemsService = {
  list() {
    return listItems().map(withInFlightAttempt);
  },

  get(id) {
    const item = getItem(id);
    return item ? withInFlightAttempt(item) : null;
  },

  add(rawUrl) {
    const parsed = parseAmazonUrl(rawUrl);
    if (parsed.kind === "invalid") {
      throw new ServiceError(400, "INVALID_URL", "Enter a valid HTTPS Amazon listing URL.");
    }
    if (parsed.kind === "unsupported") {
      throw new ServiceError(422, "UNSUPPORTED_LISTING", "This listing is outside the supported euro-priced Amazon marketplaces.");
    }

    const duplicate = db.prepare("SELECT id FROM items WHERE canonical_url = ?").get(parsed.canonicalUrl);
    if (duplicate) {
      throw new ServiceError(409, "ITEM_ALREADY_TRACKED", "This listing is already being tracked.");
    }

    const id = randomUUID();
    const addedAt = new Date().toISOString();
    insertItem({
      id,
      asin: parsed.asin,
      marketplace: parsed.marketplace,
      canonicalUrl: parsed.canonicalUrl,
      url: parsed.url,
      nextCheckAt: nextCheckAt(),
      addedAt,
    });

    // Return a pending item immediately; collection runs in the background.
    setImmediate(() => void this.collect(id).catch((error) => console.error("Initial collection failed:", error)));
    return getItem(id);
  },

  remove(id) {
    return deleteItem(id);
  },

  async collect(id) {
    if (inFlight.has(id)) return;
    const before = db.prepare("SELECT id, asin, marketplace, url FROM items WHERE id = ?").get(id);
    if (!before) return;

    const startedAt = new Date().toISOString();
    inFlight.set(id, startedAt);
    try {
      const result = await collectAmazonPrice(before);
      const timestamp = new Date().toISOString();
      if (!db.prepare("SELECT 1 FROM items WHERE id = ?").get(id)) return;

      if (result.result === "success") {
        recordSuccessfulCollection(id, {
          amountCents: result.amountCents,
          title: result.title,
          thumbnailUrl: result.thumbnailUrl,
          timestamp,
        });
      } else {
        recordFailedCollection(id, result.result, result.message, timestamp);
      }
    } finally {
      inFlight.delete(id);
    }
  },

  async collectDue() {
    const now = new Date().toISOString();
    const due = db.prepare("SELECT id FROM items WHERE next_check_at IS NOT NULL AND next_check_at <= ? ORDER BY next_check_at ASC")
      .all(now);

    for (const { id } of due) {
      if (inFlight.has(id)) continue;
      setNextCheck(id, nextCheckAt());
      await this.collect(id);
    }
  },
};

const inFlight = new Map();

function withInFlightAttempt(item) {
  const timestamp = inFlight.get(item.id);
  if (!timestamp) return item;
  return {
    ...item,
    status: "pending",
    lastAttempt: { result: "pending", timestamp, message: null },
  };
}
