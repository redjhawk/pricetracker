#!/usr/bin/env node
import fs from 'node:fs/promises';
import { constants } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import net from 'node:net';
import { spawn } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { setTimeout as delay } from 'node:timers/promises';
import { chromium } from '@playwright/test';

const repository = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const usage = 'Usage: node scripts/capture-leboncoin-session.mjs --url <listing-url> --output <private-file> [--browser <executable-path>] [--timeout-seconds <1-3600>]';
class CaptureError extends Error {}
const fail = message => { throw new CaptureError(message); };

export function parseArguments(args) {
  if (args.length === 1 && args[0] === '--help') return { help: true };
  const options = { timeoutSeconds: 600 };
  const names = { '--url': 'url', '--output': 'output', '--browser': 'browser', '--timeout-seconds': 'timeoutSeconds' };
  const seen = new Set();
  for (let i = 0; i < args.length; i += 2) {
    const key = names[args[i]];
    if (!key || seen.has(key) || !args[i + 1] || args[i + 1].startsWith('--')) fail(usage);
    seen.add(key);
    options[key] = args[i + 1];
  }
  if (!options.url || !options.output) fail(usage);
  if (!/^\d+$/.test(String(options.timeoutSeconds)) || Number(options.timeoutSeconds) < 1 || Number(options.timeoutSeconds) > 3600) fail('Timeout must be an integer from 1 through 3600 seconds.');
  options.timeoutSeconds = Number(options.timeoutSeconds);
  return options;
}

export function parseListingURL(raw) {
  // Inspect raw authority/path before WHATWG can remove userinfo or dot segments.
  const match = /^https:\/\/((?:www\.)?leboncoin\.fr\.?)(?::443)?(\/[^?#]*)(?:\?[^#]*)?(?:#.*)?$/i.exec(raw.trim());
  if (!match || /[\\\x00-\x20\x7f]/.test(raw.trim())) fail('Use a supported HTTPS LeBoncoin listing URL.');
  let decoded;
  try { decoded = decodeURIComponent(match[2]); } catch { fail('Use a supported HTTPS LeBoncoin listing URL.'); }
  const listing = /^\/ad\/([a-zA-Z0-9_-]+)\/([0-9]+),?$/i.exec(decoded);
  if (!listing) fail('Use a supported HTTPS LeBoncoin listing URL.');
  return { url: `https://www.leboncoin.fr/ad/${listing[1].toLowerCase()}/${listing[2]}`, id: listing[2] };
}

export function verifiedListing(listing, document, snapshot) {
  if (!document?.ready || document.status < 200 || document.status >= 300) return false;
  try {
    if (parseListingURL(document.url).url !== listing.url || parseListingURL(snapshot.url).url !== listing.url) return false;
    const ad = JSON.parse(snapshot.data)?.props?.pageProps?.ad;
    return ad?.status === 'active' && Number.isSafeInteger(ad.list_id) && ad.list_id > 0 && BigInt(ad.list_id) === BigInt(listing.id);
  } catch { return false; }
}

export function sessionExport(cookies, listing, now = new Date()) {
  const target = new URL(listing.url);
  const candidates = cookies.filter(cookie => cookie.name === 'datadome');
  if (candidates.length !== 1) fail('A single applicable datadome cookie is required; complete verification or retry.');
  const cookie = candidates[0];
  const domains = ['leboncoin.fr', '.leboncoin.fr', 'www.leboncoin.fr', '.www.leboncoin.fr'];
  const domainApplies = cookie.domain?.startsWith('.') ? target.hostname === cookie.domain.slice(1) || target.hostname.endsWith(cookie.domain) : target.hostname === cookie.domain;
  const pathApplies = typeof cookie.path === 'string' && cookie.path.startsWith('/') && !/[\x00-\x1f\x7f]/.test(cookie.path) && (target.pathname === cookie.path || target.pathname.startsWith(cookie.path.endsWith('/') ? cookie.path : cookie.path + '/'));
  if (!domains.includes(cookie.domain) || !domainApplies || !pathApplies || typeof cookie.secure !== 'boolean' || typeof cookie.value !== 'string' || cookie.value.length > 4096 || !/^[\x21\x23-\x2b\x2d-\x3a\x3c-\x5b\x5d-\x7e]+$/.test(cookie.value)) fail('The applicable datadome cookie has invalid scope or metadata.');
  let expiresAt = null;
  if (cookie.expires !== -1) {
    const expiry = new Date(cookie.expires * 1000);
    if (typeof cookie.expires !== 'number' || !Number.isFinite(cookie.expires) || !Number.isFinite(expiry.getTime()) || expiry <= now || expiry.getUTCFullYear() > 9999) fail('The datadome cookie is expired or has invalid expiry metadata.');
    expiresAt = expiry.toISOString();
  }
  return { version: 1, capturedAt: now.toISOString(), cookie: { name: 'datadome', value: cookie.value, domain: cookie.domain, path: cookie.path, secure: cookie.secure, expiresAt } };
}

export async function validateDestination(output) {
  if (process.platform === 'win32' || !process.getuid) fail('Capture requires a POSIX desktop with private file permissions.');
  const destination = path.resolve(output);
  if (destination === repository || destination.startsWith(repository + path.sep)) fail('Keep the session file outside the repository.');
  const directory = path.dirname(destination);
  let current = path.parse(directory).root;
  for (const segment of directory.slice(current.length).split(path.sep).filter(Boolean)) {
    current = path.join(current, segment);
    const stat = await fs.lstat(current);
    if (!stat.isDirectory() || stat.isSymbolicLink()) fail('Output parent directories must be real directories, without symlinks.');
  }
  const parent = await fs.lstat(directory);
  if (parent.uid !== process.getuid() || (parent.mode & 0o777) !== 0o700) fail('Output directory must be owned by you with mode 0700.');
  try {
    const stat = await fs.lstat(destination);
    if (!stat.isFile() || stat.isSymbolicLink() || stat.uid !== process.getuid() || (stat.mode & 0o077) !== 0) fail('Existing output must be an owned private regular file, without symlinks.');
  } catch (error) { if (error.code !== 'ENOENT') throw error; }
  return destination;
}

export async function writeSession(output, value, signal, io = fs) {
  signal?.throwIfAborted();
  const destination = await validateDestination(output);
  const contents = JSON.stringify(value) + '\n';
  if (Buffer.byteLength(contents) > 16384) fail('Session export exceeds the supported size.');
  const temporary = path.join(path.dirname(destination), `.session-${randomUUID()}.tmp`);
  let handle;
  try {
    handle = await io.open(temporary, constants.O_WRONLY | constants.O_CREAT | constants.O_EXCL | constants.O_NOFOLLOW, 0o600);
    await handle.writeFile(contents, 'utf8');
    await handle.sync();
    await handle.close();
    handle = undefined;
    await validateDestination(destination);
    signal?.throwIfAborted();
    await io.rename(temporary, destination);
  } finally {
    await handle?.close().catch(() => {});
    await fs.rm(temporary, { force: true });
  }
}

async function availablePort() {
  const server = net.createServer();
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
  const port = server.address().port;
  await new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve()));
  return port;
}

// Race operations without losing their cleanup owner when the deadline expires.
async function abortable(operation, signal) {
  signal.throwIfAborted();
  let listener;
  const aborted = new Promise((_, reject) => {
    listener = () => reject(signal.reason);
    signal.addEventListener('abort', listener, { once: true });
  });
  try { return await Promise.race([operation, aborted]); }
  finally { signal.removeEventListener('abort', listener); }
}

export async function openBrowser(options, signal, dependencies = {}) {
  const executable = options.browser || chromium.executablePath();
  try { await fs.access(executable, constants.X_OK); }
  catch { fail('Browser executable unavailable. Run npx playwright install chromium or pass --browser <executable-path>.'); }
  const profile = await fs.mkdtemp(path.join(os.tmpdir(), 'pricefollower-capture-'));
  let child, browser, exited = false, closing;
  const terminate = signalName => {
    try {
      if (child?.pid) process.kill(-child.pid, signalName);
      else child?.kill(signalName);
    } catch (error) { if (error.code !== 'ESRCH') throw error; }
  };
  const close = () => closing ??= (async () => {
    if (browser) await Promise.race([browser.close().catch(() => {}), delay(1000)]);
    if (child && !exited) {
      terminate('SIGTERM');
      await Promise.race([new Promise(resolve => child.once('exit', resolve)), delay(1500)]);
      if (!exited) {
        terminate('SIGKILL');
        await Promise.race([new Promise(resolve => child.once('exit', resolve)), delay(1000)]);
      }
    }
    // Chrome descendants share this task-owned process group.
    terminate('SIGKILL');
    await fs.rm(profile, { recursive: true, force: true });
  })();
  try {
    await fs.chmod(profile, 0o700);
    signal.throwIfAborted();
    const port = await (dependencies.availablePort || availablePort)();
    const marker = `data:text/plain,pricefollower-${randomUUID()}`;
    child = (dependencies.spawn || spawn)(executable, [`--user-data-dir=${profile}`, '--no-first-run', '--no-default-browser-check', '--remote-debugging-address=127.0.0.1', `--remote-debugging-port=${port}`, marker], { stdio: 'ignore', detached: true });
    child.once('exit', () => { exited = true; });
    child.once('error', () => { exited = true; });
    const startupEnd = Date.now() + 20000;
    while (!browser && Date.now() < startupEnd) {
      signal.throwIfAborted();
      if (exited) fail('Browser exited before capture connected. Check your graphical desktop and browser setup.');
      try {
        const requestSignal = AbortSignal.any([signal, AbortSignal.timeout(Math.min(1000, Math.max(1, startupEnd - Date.now())))]);
        const response = await (dependencies.fetch || fetch)(`http://127.0.0.1:${port}/json/list`, { signal: requestSignal });
        const targets = await response.json();
        // A fresh random startup target proves this endpoint belongs to our child.
        if (!Array.isArray(targets) || !targets.some(target => target.type === 'page' && target.url === marker)) fail('Browser debugging port collision; retry capture.');
        browser = await (dependencies.connect || chromium.connectOverCDP.bind(chromium))(`http://127.0.0.1:${port}`, { timeout: Math.min(1000, Math.max(1, startupEnd - Date.now())) });
      } catch (error) {
        if (error instanceof CaptureError) throw error;
        await delay(200, undefined, { signal });
      }
    }
    signal.throwIfAborted();
    if (!browser || exited) fail('Browser connection did not start within 20 seconds.');
    const context = browser.contexts()[0];
    const page = context?.pages().find(candidate => candidate.url() === marker);
    if (!page) fail('Could not confirm the isolated browser session.');
    return { close, waitForSession: listing => waitForSession(page, context, listing, signal, () => exited || !browser.isConnected()) };
  } catch (error) { await close(); throw error; }
}

export async function waitForSession(page, context, listing, signal, disconnected = () => false) {
  let document = null;
  const mainDocument = request => request.isNavigationRequest() && request.frame() === page.mainFrame();
  page.on('request', request => { if (mainDocument(request)) document = { request, ready: false }; });
  page.on('response', response => {
    if (response.request() === document?.request) document = { request: response.request(), url: response.url(), status: response.status(), ready: false };
  });
  page.on('requestfailed', request => { if (request === document?.request) document = null; });
  page.on('domcontentloaded', () => { if (document?.status) document.ready = true; });
  for (const url of ['https://www.leboncoin.fr/', listing.url]) {
    try { await abortable(page.goto(url, { waitUntil: 'domcontentloaded', timeout: 15000 }), signal); }
    catch (error) { signal.throwIfAborted(); if (error.name !== 'TimeoutError') document = null; }
  }
  while (true) {
    signal.throwIfAborted();
    if (page.isClosed() || disconnected()) fail('Browser closed before a verified listing was captured.');
    const observed = document;
    let snapshot;
    try {
      snapshot = await abortable(page.evaluate(() => ({ url: location.href, data: document.querySelector('script#__NEXT_DATA__')?.textContent })), signal);
    } catch { signal.throwIfAborted(); }
    if (snapshot && observed === document && verifiedListing(listing, observed, snapshot)) {
      const cookies = await abortable(context.cookies(listing.url), signal);
      if (observed !== document || page.url() !== snapshot.url) continue;
      // Missing cookies may arrive just after the document; keep waiting safely.
      try { return sessionExport(cookies, listing); } catch (error) { if (!(error instanceof CaptureError)) throw error; }
    }
    await delay(1000, undefined, { signal });
  }
}

export async function capture(options, dependencies = {}) {
  const listing = parseListingURL(options.url);
  const output = await validateDestination(options.output);
  const controller = new AbortController();
  const signal = dependencies.signal ? AbortSignal.any([controller.signal, dependencies.signal]) : controller.signal;
  const timer = setTimeout(() => controller.abort(new CaptureError('Capture timed out. Complete verification and try again.')), options.timeoutSeconds * 1000);
  const log = dependencies.log || console.log;
  let browser;
  try {
    signal.throwIfAborted();
    log('Opening an isolated visible browser. Complete any LeBoncoin verification yourself. Press Ctrl+C to cancel.');
    browser = await (dependencies.openBrowser || openBrowser)(options, signal);
    signal.throwIfAborted();
    log('Waiting for the requested active listing and its verified session…');
    const exported = await abortable(browser.waitForSession(listing), signal);
    signal.throwIfAborted();
    await writeSession(output, exported, signal);
    log(`Session saved privately to ${output}. Server acceptance must be checked separately.`);
  } finally {
    clearTimeout(timer);
    await browser?.close();
  }
}

export async function main(args = process.argv.slice(2)) {
  const controller = new AbortController();
  let exitCode = 1;
  const interrupt = () => { exitCode = 130; controller.abort(new CaptureError('Capture cancelled.')); };
  const terminate = () => { exitCode = 143; controller.abort(new CaptureError('Capture cancelled.')); };
  process.once('SIGINT', interrupt);
  process.once('SIGTERM', terminate);
  try {
    const options = parseArguments(args);
    if (options.help) { console.log(usage); return 0; }
    await capture(options, { signal: controller.signal });
    return 0;
  } catch (error) {
    const diagnostic = controller.signal.aborted ? 'Capture cancelled.' : error instanceof CaptureError ? error.message : 'Capture failed. Check the browser setup and private output permissions, then retry.';
    console.error(diagnostic);
    return exitCode;
  } finally {
    process.removeListener('SIGINT', interrupt);
    process.removeListener('SIGTERM', terminate);
  }
}
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) process.exitCode = await main();
