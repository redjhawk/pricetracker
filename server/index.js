import fs from "node:fs";
import path from "node:path";
import { createServer } from "node:http";
import { config } from "./config.js";
import { db, seedDevelopmentItems } from "./database.js";
import { itemsService, nextCheckAt, ServiceError } from "./service.js";

const distDirectory = path.join(config.root, "dist");
const mimeTypes = {
  ".css": "text/css; charset=utf-8",
  ".html": "text/html; charset=utf-8",
  ".ico": "image/x-icon",
  ".js": "text/javascript; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".png": "image/png",
  ".svg": "image/svg+xml",
  ".webp": "image/webp",
  ".woff2": "font/woff2",
};

if (config.development) seedDevelopmentItems(nextCheckAt);

function sendJson(response, status, body) {
  response.writeHead(status, {
    "Content-Type": "application/json; charset=utf-8",
    "Cache-Control": "no-store",
    "X-Content-Type-Options": "nosniff",
  });
  response.end(body === undefined ? undefined : JSON.stringify(body));
}

async function readJson(request) {
  const chunks = [];
  let length = 0;
  for await (const chunk of request) {
    length += chunk.length;
    if (length > 32_768) throw new ServiceError(413, "REQUEST_TOO_LARGE", "Request body is too large.");
    chunks.push(chunk);
  }
  try {
    return JSON.parse(Buffer.concat(chunks).toString("utf8"));
  } catch {
    throw new ServiceError(400, "INVALID_JSON", "Request body must be valid JSON.");
  }
}

function serveFrontend(request, response, pathname) {
  const requestedPath = pathname === "/" ? "/index.html" : pathname;
  const resolved = path.resolve(distDirectory, `.${decodeURIComponent(requestedPath)}`);
  if (!resolved.startsWith(`${distDirectory}${path.sep}`) && resolved !== path.join(distDirectory, "index.html")) {
    response.writeHead(400).end("Bad request");
    return;
  }

  let target = resolved;
  if (!fs.existsSync(target) || fs.statSync(target).isDirectory()) {
    // Serve the SPA shell for direct navigation to client-side item pages.
    target = path.join(distDirectory, "index.html");
  }
  if (!fs.existsSync(target)) {
    response.writeHead(404, { "Content-Type": "text/plain; charset=utf-8" }).end("Frontend build not found. Run npm run build.");
    return;
  }

  response.writeHead(200, {
    "Content-Type": mimeTypes[path.extname(target)] ?? "application/octet-stream",
    "X-Content-Type-Options": "nosniff",
  });
  fs.createReadStream(target).pipe(response);
}

async function handleApi(request, response, pathname) {
  if (request.method === "GET" && pathname === "/api/v1/items") {
    sendJson(response, 200, { items: itemsService.list() });
    return;
  }

  if (request.method === "POST" && pathname === "/api/v1/items") {
    const body = await readJson(request);
    if (!body || typeof body.url !== "string" || !body.url.trim()) {
      throw new ServiceError(400, "INVALID_URL", "A listing URL is required.");
    }
    sendJson(response, 201, itemsService.add(body.url.trim()));
    return;
  }

  const itemMatch = pathname.match(/^\/api\/v1\/items\/([^/]+)$/);
  if (itemMatch && request.method === "GET") {
    const id = decodeURIComponent(itemMatch[1]);
    const item = itemsService.get(id);
    if (!item) throw new ServiceError(404, "ITEM_NOT_FOUND", "Tracked item was not found.");
    sendJson(response, 200, item);
    return;
  }

  if (itemMatch && request.method === "DELETE") {
    const id = decodeURIComponent(itemMatch[1]);
    if (!itemsService.remove(id)) throw new ServiceError(404, "ITEM_NOT_FOUND", "Tracked item was not found.");
    response.writeHead(204, { "Cache-Control": "no-store" }).end();
    return;
  }

  sendJson(response, 404, { error: { code: "ROUTE_NOT_FOUND", message: "API route was not found." } });
}

const server = createServer(async (request, response) => {
  let pathname;
  try {
    pathname = new URL(request.url ?? "/", "http://localhost").pathname;
    if (pathname.startsWith("/api/")) {
      await handleApi(request, response, pathname);
    } else if (request.method === "GET" && !config.development && fs.existsSync(distDirectory)) {
      serveFrontend(request, response, pathname);
    } else {
      sendJson(response, 404, { error: { code: "ROUTE_NOT_FOUND", message: "Route was not found." } });
    }
  } catch (error) {
    if (response.headersSent) {
      response.destroy();
      return;
    }
    if (error instanceof ServiceError) {
      sendJson(response, error.status, {
        error: { code: error.code, message: error.message },
      });
      return;
    }
    console.error("Request failed:", error);
    sendJson(response, 500, {
      error: { code: "INTERNAL_ERROR", message: "The server could not complete the request." },
    });
  }
});

server.listen(config.port, config.host, () => {
  console.info(`Price follower API listening on http://${config.host}:${config.port}`);
  if (!config.development) {
    console.info(`Serving frontend from ${distDirectory} when a production build is present.`);
  }
});

const scheduler = setInterval(() => {
  itemsService.collectDue().catch((error) => console.error("Scheduled collection failed:", error));
}, 30_000);
itemsService.collectDue().catch((error) => console.error("Initial scheduled collection failed:", error));

function shutdown() {
  clearInterval(scheduler);
  server.close(() => {
    db.close();
    process.exit(0);
  });
  setTimeout(() => process.exit(1), 5_000).unref();
}

process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);
