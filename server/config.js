import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const development = process.env.NODE_ENV === "development";
const configuredDataDirectory = process.env.PRICEFOLLOWER_DATA_DIR;

export const config = {
  root,
  development,
  host: process.env.HOST ?? (development ? "127.0.0.1" : "0.0.0.0"),
  port: Number(process.env.PORT ?? 3001),
  timezone: process.env.PRICEFOLLOWER_TIMEZONE ?? "Europe/Paris",
  checkTimes: (process.env.PRICEFOLLOWER_CHECK_TIMES ?? "08:00,20:00")
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean),
  staleAfterHours: Number(process.env.PRICEFOLLOWER_STALE_AFTER_HOURS ?? 36),
  dataDirectory:
    configuredDataDirectory ??
    (development ? path.join(root, ".data") : "/var/lib/pricefollower"),
  userAgent:
    process.env.AMAZON_USER_AGENT ??
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36",
};

export const databasePath = path.join(config.dataDirectory, "pricefollower.sqlite");
