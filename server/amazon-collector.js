import { config } from "./config.js";

export const AMAZON_HOSTS = new Set([
  "amazon.de",
  "amazon.fr",
  "amazon.es",
  "amazon.it",
  "amazon.nl",
  "amazon.be",
]);

export function parseAmazonUrl(rawValue) {
  let url;
  try {
    url = new URL(rawValue);
  } catch {
    return { kind: "invalid" };
  }

  const hostname = url.hostname.toLowerCase().replace(/^www\./, "");
  if (url.protocol !== "https:" || (url.port && url.port !== "443") || url.username || url.password || !AMAZON_HOSTS.has(hostname)) {
    return { kind: "unsupported" };
  }

  const asin = url.pathname.match(/\/(?:dp|gp\/product|gp\/aw\/d)\/([A-Z0-9]{10})(?:\/|$)/i)?.[1];
  if (!asin) return { kind: "unsupported" };

  const marketplace = hostname;
  return {
    kind: "valid",
    asin: asin.toUpperCase(),
    marketplace,
    url: `https://${marketplace}/dp/${asin.toUpperCase()}`,
    canonicalUrl: `https://${marketplace}/dp/${asin.toUpperCase()}`,
  };
}

function attribute(tag, name) {
  const match = tag.match(new RegExp(`\\b${name}\\s*=\\s*(?:"([^"]*)"|'([^']*)'|([^\\s>]+))`, "i"));
  return match?.[1] ?? match?.[2] ?? match?.[3] ?? null;
}

function decodeEntities(value) {
  return value
    .replace(/&nbsp;|&#160;/gi, " ")
    .replace(/&amp;/gi, "&")
    .replace(/&quot;/gi, '"')
    .replace(/&#39;|&apos;/gi, "'")
    .replace(/&euro;|&#8364;|&#x20ac;/gi, "€")
    .replace(/&#(\d+);/g, (_, number) => String.fromCodePoint(Number(number)))
    .replace(/&#x([\da-f]+);/gi, (_, number) => String.fromCodePoint(parseInt(number, 16)));
}

function textContent(fragment) {
  return decodeEntities(fragment.replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, " ")
    .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, " ")
    .replace(/<[^>]+>/g, " "))
    .replace(/[\u200e\u200f\u202a-\u202e]/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

function elementContentById(html, id) {
  const escaped = id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const match = html.match(new RegExp(`<([a-z][\\w:-]*)\\b[^>]*\\bid=["']${escaped}["'][^>]*>([\\s\\S]*?)<\\/\\1>`, "i"));
  return match?.[2] ?? "";
}

function metaContent(html, property) {
  const tags = html.match(/<meta\b[^>]*>/gi) ?? [];
  const tag = tags.find((candidate) => {
    const name = attribute(candidate, "property") ?? attribute(candidate, "name");
    return name?.toLowerCase() === property.toLowerCase();
  });
  const content = tag ? attribute(tag, "content") : null;
  return content ? textContent(content) : null;
}

function productTitle(html) {
  const fromTitleElement = textContent(elementContentById(html, "productTitle"));
  return fromTitleElement || metaContent(html, "og:title") || null;
}

function imageById(html, id) {
  const match = html.match(new RegExp(`<img\\b(?=[^>]*\\bid=["']${id}["'])[^>]*>`, "i"));
  if (!match) return null;
  return attribute(match[0], "data-old-hires") || attribute(match[0], "src") || null;
}

function parseEuroPrice(value) {
  const decoded = textContent(value);
  if (!/(?:€|\bEUR\b)/i.test(decoded)) return null;
  const numberMatch = decoded.match(/[0-9][0-9\s\u00a0.,]*[0-9]|[0-9]/);
  if (!numberMatch) return null;

  let digits = numberMatch[0].replace(/[\s\u00a0]/g, "");
  const comma = digits.lastIndexOf(",");
  const dot = digits.lastIndexOf(".");
  let decimalIndex = -1;
  if (comma >= 0 && dot >= 0) decimalIndex = Math.max(comma, dot);
  else if (comma >= 0 && digits.length - comma - 1 <= 2) decimalIndex = comma;
  else if (dot >= 0 && digits.length - dot - 1 <= 2) decimalIndex = dot;

  let euros;
  let cents;
  if (decimalIndex >= 0) {
    euros = digits.slice(0, decimalIndex).replace(/[.,]/g, "") || "0";
    cents = digits.slice(decimalIndex + 1).replace(/[.,]/g, "").padEnd(2, "0").slice(0, 2);
  } else {
    euros = digits.replace(/[.,]/g, "");
    cents = "00";
  }

  const amountCents = Number(euros) * 100 + Number(cents || "0");
  return Number.isSafeInteger(amountCents) && amountCents > 0 ? amountCents : null;
}

function pricesInRegion(region) {
  const candidates = [];
  const offscreenPattern = /<span\b[^>]*class=["']([^"']*\ba-offscreen\b[^"']*)["'][^>]*>([\s\S]*?)<\/span>/gi;
  for (const match of region.matchAll(offscreenPattern)) {
    const prefix = region.slice(Math.max(0, match.index - 2_000), match.index);
    const priceWrappers = [...prefix.matchAll(/<span\b[^>]*class=["']([^"']*\ba-price\b[^"']*)["'][^>]*>/gi)];
    const wrapperClasses = priceWrappers.at(-1)?.[1]?.toLowerCase() ?? "";
    if (!wrapperClasses) continue;
    if (/a-text-price|price-per-unit|unit-price|installment|saving|coupon|listprice/.test(wrapperClasses)) continue;

    const amountCents = parseEuroPrice(match[2]);
    if (amountCents !== null) {
      candidates.push({ amountCents, primary: /\bpricetopay\b|\bpriceblock_(?:our|deal)price\b/.test(wrapperClasses) });
    }
  }
  return candidates;
}

function productPrice(html) {
  const regions = [
    "corePriceDisplay_desktop_feature_div",
    "corePrice_feature_div",
    "price_inside_buybox",
    "apex_desktop",
    "desktop_buybox",
  ];
  const candidates = [];
  for (let priority = 0; priority < regions.length; priority += 1) {
    const id = regions[priority];
    const index = html.indexOf(`id="${id}"`);
    const alternateIndex = index < 0 ? html.indexOf(`id='${id}'`) : index;
    const start = index >= 0 ? index : alternateIndex;
    if (start < 0) continue;
    for (const candidate of pricesInRegion(html.slice(start, start + 20_000))) {
      candidates.push({ ...candidate, priority });
    }
  }

  candidates.sort((a, b) => Number(b.primary) - Number(a.primary) || a.priority - b.priority);
  return candidates[0]?.amountCents ?? null;
}

function productImage(html) {
  const oldHires = imageById(html, "landingImage");
  if (oldHires) return oldHires;
  const dynamicImage = html.match(/<img\b[^>]*data-a-dynamic-image=["']([^"']+)["'][^>]*>/i);
  if (dynamicImage) {
    const source = attribute(dynamicImage[0], "src");
    if (source) return source;
  }
  return metaContent(html, "og:image");
}

function isUnavailable(html) {
  return /currently unavailable|temporarily out of stock|no longer available|this item is not available/i.test(
    textContent(html.slice(0, 200_000)),
  );
}

async function fetchProductHtml(initialUrl) {
  let target = new URL(initialUrl);
  const localeByMarketplace = {
    "amazon.de": "de-DE,de;q=0.9,en;q=0.8",
    "amazon.fr": "fr-FR,fr;q=0.9,en;q=0.8",
    "amazon.es": "es-ES,es;q=0.9,en;q=0.8",
    "amazon.it": "it-IT,it;q=0.9,en;q=0.8",
    "amazon.nl": "nl-NL,nl;q=0.9,en;q=0.8",
    "amazon.be": "nl-BE,nl;q=0.9,fr-BE;q=0.8,en;q=0.7",
  };
  const chromeMajor = config.userAgent.match(/Chrome\/(\d+)/)?.[1] ?? "154";
  for (let redirects = 0; redirects <= 4; redirects += 1) {
    if (target.protocol !== "https:" || !AMAZON_HOSTS.has(target.hostname.toLowerCase().replace(/^www\./, ""))) {
      throw new Error("The Amazon listing redirected to an unsupported host.");
    }

    const response = await fetch(target, {
      method: "GET",
      redirect: "manual",
      signal: AbortSignal.timeout(15_000),
      headers: {
        "User-Agent": config.userAgent,
        Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
        "Accept-Language": localeByMarketplace[target.hostname.replace(/^www\./, "")] ?? "en-GB,en;q=0.9",
        "Accept-Encoding": "identity",
        "Upgrade-Insecure-Requests": "1",
        "Sec-CH-UA": `"Google Chrome";v="${chromeMajor}", "Chromium";v="${chromeMajor}", "Not_A Brand";v="99"`,
        "Sec-CH-UA-Mobile": "?0",
        "Sec-CH-UA-Platform": '"Linux"',
        "Sec-Fetch-Dest": "document",
        "Sec-Fetch-Mode": "navigate",
        "Sec-Fetch-Site": "none",
        "Sec-Fetch-User": "?1",
      },
    });

    if (response.status >= 300 && response.status < 400) {
      const location = response.headers.get("location");
      if (!location || redirects === 4) throw new Error("The Amazon listing could not be reached.");
      target = new URL(location, target);
      continue;
    }
    if (response.status === 404 || response.status === 410) return { kind: "unavailable" };
    if (!response.ok) throw new Error(`Amazon returned HTTP ${response.status}.`);
    const contentType = response.headers.get("content-type") ?? "";
    if (!contentType.includes("text/html")) throw new Error("Amazon returned an unexpected response.");
    const html = await response.text();
    if (/captcha|robot check|enter the characters you see below/i.test(html.slice(0, 200_000))) {
      throw new Error("Amazon blocked the product request.");
    }
    return { kind: "html", html };
  }
  throw new Error("The Amazon listing could not be reached.");
}

export async function collectAmazonPrice(item) {
  try {
    const fetched = await fetchProductHtml(item.url);
    if (fetched.kind === "unavailable") {
      return { result: "unavailable", message: "The listing is no longer available." };
    }

    const { html } = fetched;
    if (isUnavailable(html)) {
      return { result: "unavailable", message: "The listing is no longer available." };
    }

    const amountCents = productPrice(html);
    if (amountCents === null) {
      return { result: "price_not_found", message: "Amazon did not show a detectable euro price." };
    }

    return {
      result: "success",
      amountCents,
      title: productTitle(html),
      thumbnailUrl: productImage(html),
    };
  } catch (error) {
    console.warn(`Amazon price collection failed for ${item.marketplace}/${item.asin}: ${error.message}`);
    return { result: "request_error", message: "Amazon could not be reached for a price check." };
  }
}
