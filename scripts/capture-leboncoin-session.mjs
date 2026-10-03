#!/usr/bin/env node
// Keep this module parseable by older Node.js versions so the runtime check can explain itself.
import fs from 'node:fs/promises';
import { constants } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import net from 'node:net';
import { spawn } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { setTimeout as delay } from 'node:timers/promises';

const usage = 'Usage: node scripts/capture-leboncoin-session.mjs --url <listing-url> [--browser <executable-path>] [--timeout-seconds <1-3600>]';
const outputRemoved = '--output is no longer supported. The session is now printed here; paste it into Settings › LeBonCoin session in PriceFollower.';
const startupMilliseconds = 20000;
const settleMilliseconds = 1000;
const noCookieMilliseconds = 15000;
const timeoutReasons = {
  challenge: 'LeBoncoin was still showing a verification page',
  'not-listing': 'the page was not the requested active listing (check the URL or try another active listing)',
  'navigation-error': 'the listing page could not be loaded',
  'no-cookie': 'the listing loaded but LeBoncoin had not set a usable datadome cookie',
  none: 'no page loaded yet',
};
class CaptureError extends Error {}
const fail = message => { throw new CaptureError(message); };

export function parseArguments(args) {
  if (args.some(arg => arg === '--output' || arg.startsWith('--output='))) fail(outputRemoved);
  if (args.length === 1 && args[0] === '--help') return { help: true };
  const options = { timeoutSeconds: 600 };
  const names = { '--url': 'url', '--browser': 'browser', '--timeout-seconds': 'timeoutSeconds' };
  const seen = new Set();
  for (let i = 0; i < args.length; i += 2) {
    const key = names[args[i]];
    if (!key || seen.has(key) || !args[i + 1] || args[i + 1].startsWith('--')) fail(usage);
    seen.add(key);
    options[key] = args[i + 1];
  }
  if (!options.url) fail(usage);
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

// Returns the only text printed on stdout: the single applicable datadome cookie.
export function sessionLine(cookies, listing, now = new Date()) {
  const target = new URL(listing.url);
  const candidates = cookies.filter(cookie => cookie.name === 'datadome');
  if (candidates.length !== 1) fail('A single applicable datadome cookie is required; complete verification or retry.');
  const cookie = candidates[0];
  const domains = ['leboncoin.fr', '.leboncoin.fr', 'www.leboncoin.fr', '.www.leboncoin.fr'];
  const domainApplies = cookie.domain?.startsWith('.') ? target.hostname === cookie.domain.slice(1) || target.hostname.endsWith(cookie.domain) : target.hostname === cookie.domain;
  const pathApplies = typeof cookie.path === 'string' && cookie.path.startsWith('/') && !/[\x00-\x1f\x7f]/.test(cookie.path) && (target.pathname === cookie.path || target.pathname.startsWith(cookie.path.endsWith('/') ? cookie.path : cookie.path + '/'));
  if (!domains.includes(cookie.domain) || !domainApplies || !pathApplies || typeof cookie.value !== 'string' || cookie.value.length > 4096 || !/^[\x21\x23-\x2b\x2d-\x3a\x3c-\x5b\x5d-\x7e]+$/.test(cookie.value)) fail('The applicable datadome cookie has invalid scope or value.');
  if (cookie.expires !== -1) {
    const expiry = new Date(cookie.expires * 1000);
    if (typeof cookie.expires !== 'number' || !Number.isFinite(expiry.getTime()) || expiry <= now) fail('The datadome cookie is expired or has invalid expiry metadata.');
  }
  return `datadome=${cookie.value}`;
}

export function checkRuntime(version) {
  const [major, minor] = version.split('.').map(Number);
  if (major < 20 || (major === 20 && minor < 19)) fail(`Node.js 20.19 or newer is required (found v${version}). Install a newer Node.js, then retry.`);
}

export async function loadPlaywright(importer = () => import('@playwright/test')) {
  try { return (await importer()).chromium; }
  catch { fail('Repository dependencies are missing. Run npm ci in the repository, then retry.'); }
}

export function checkDisplay(env, platform) {
  if (platform === 'win32') fail('This helper supports Linux and macOS desktops only.');
  if (platform === 'linux' && !env.DISPLAY && !env.WAYLAND_DISPLAY) fail('No graphical display was found (DISPLAY and WAYLAND_DISPLAY are not set). Run the helper from a terminal in your desktop session.');
}

const isExecutable = candidate => fs.access(candidate, constants.X_OK);

export async function findBrowser(requested, { env, platform, access = isExecutable, chromium }) {
  const usable = async candidate => { try { await access(candidate); return true; } catch { return false; } };
  if (requested) {
    if (await usable(requested)) return requested;
    fail(`The browser at ${requested} was not found or is not executable.`);
  }
  const directories = (env.PATH || '').split(path.delimiter).filter(Boolean);
  const candidates = platform === 'darwin'
    ? ['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome']
    : ['google-chrome', 'google-chrome-stable', 'chromium', 'chromium-browser'].flatMap(name => directories.map(directory => path.join(directory, name)));
  try { candidates.push(chromium.executablePath()); } catch {}
  for (const candidate of candidates) if (candidate && await usable(candidate)) return candidate;
  fail('No Chrome or Chromium browser was found. Install Google Chrome or Chromium, or pass --browser <path>.');
}

const describeExit = exit => exit?.signal ? `signal ${exit.signal}` : exit ? `exit code ${exit.code}` : 'it could not be started';

export function browserEndMessage({ pageClosed, exit, error }) {
  if (error || (exit && (exit.code !== 0 || exit.signal))) return `The browser exited unexpectedly${exit ? ` (${describeExit(exit)})` : ''} before verification finished. Retry; if it repeats, start the browser manually to check it works.`;
  if (pageClosed) return 'The browser window was closed before verification finished. Run the helper again and leave the window open until it reports success.';
  if (exit) return 'The browser was quit before verification finished.';
  return 'Lost the connection to the browser before verification finished. Retry the capture.';
}

export const timeoutMessage = (seconds, reason) => `Capture timed out after ${seconds} seconds: ${timeoutReasons[reason]}.`;

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

// timers/promises rejects with a generic AbortError; surface the abort reason instead.
const pauseFor = (milliseconds, signal) => delay(milliseconds, undefined, { signal }).catch(error => { signal.throwIfAborted(); throw error; });

// Un-minimize and raise the window; focus-stealing prevention may still keep it behind.
async function showWindow(context, page, signal) {
  const session = await abortable(context.newCDPSession(page), signal);
  const { windowId } = await abortable(session.send('Browser.getWindowForTarget'), signal);
  if (!windowId) fail('The browser started but did not open a window within 20 seconds.');
  await abortable(session.send('Browser.setWindowBounds', { windowId, bounds: { windowState: 'normal' } }), signal).catch(() => {});
  await abortable(page.bringToFront(), signal).catch(() => {});
  await session.detach().catch(() => {});
}

export async function openBrowser(executablePath, signal, dependencies = {}) {
  const profile = await fs.mkdtemp(path.join(os.tmpdir(), 'pricefollower-capture-'));
  let child, browser, closing, settleTimer;
  let exit = null, spawnFailed = false, connected = false;
  const ending = {};
  const ended = new AbortController();
  // Collect the rest of the end sequence briefly, then report one classified reason.
  const noteEnd = change => {
    if (!connected || closing) return;
    Object.assign(ending, change);
    if (settleTimer === undefined) settleTimer = setTimeout(() => ended.abort(new CaptureError(browserEndMessage(ending))), settleMilliseconds);
  };
  const terminate = signalName => {
    try {
      if (child?.pid) process.kill(-child.pid, signalName);
      else child?.kill(signalName);
    } catch (error) { if (error.code !== 'ESRCH') throw error; }
  };
  const running = () => child && !exit && !spawnFailed;
  const close = () => {
    if (closing) return closing;
    return closing = (async () => {
      clearTimeout(settleTimer);
      if (browser) await Promise.race([browser.close().catch(() => {}), delay(1000)]);
      if (running()) {
        terminate('SIGTERM');
        await Promise.race([new Promise(resolve => child.once('exit', resolve)), delay(1500)]);
        if (running()) {
          terminate('SIGKILL');
          await Promise.race([new Promise(resolve => child.once('exit', resolve)), delay(1000)]);
        }
      }
      // Chrome descendants share this task-owned process group.
      terminate('SIGKILL');
      await fs.rm(profile, { recursive: true, force: true });
    })();
  };
  const startupFailure = () => fail(`The browser exited during startup (${describeExit(exit)}). Check that it starts from this desktop session, or pass another --browser.`);
  try {
    await fs.chmod(profile, 0o700);
    signal.throwIfAborted();
    const port = await (dependencies.availablePort || availablePort)();
    const marker = `data:text/plain,pricefollower-${randomUUID()}`;
    child = (dependencies.spawn || spawn)(executablePath, [`--user-data-dir=${profile}`, '--no-first-run', '--no-default-browser-check', '--remote-debugging-address=127.0.0.1', `--remote-debugging-port=${port}`, marker], { stdio: 'ignore', detached: true });
    child.on('exit', (code, signalName) => { exit = { code, signal: signalName }; noteEnd({ exit }); });
    child.on('error', () => { spawnFailed = true; noteEnd({ error: true }); });
    const startupEnd = Date.now() + startupMilliseconds;
    const startupSignal = () => AbortSignal.any([signal, AbortSignal.timeout(Math.max(1, startupEnd - Date.now()))]);
    while (!browser && Date.now() < startupEnd) {
      signal.throwIfAborted();
      if (exit || spawnFailed) startupFailure();
      try {
        const requestSignal = AbortSignal.any([signal, AbortSignal.timeout(Math.min(1000, Math.max(1, startupEnd - Date.now())))]);
        const response = await (dependencies.fetch || fetch)(`http://127.0.0.1:${port}/json/list`, { signal: requestSignal });
        const targets = await response.json();
        // A fresh random startup target proves this endpoint belongs to our child.
        if (!Array.isArray(targets) || !targets.some(target => target.type === 'page' && target.url === marker)) fail('Browser debugging port collision; retry capture.');
        browser = await dependencies.connect(`http://127.0.0.1:${port}`, { timeout: Math.min(1000, Math.max(1, startupEnd - Date.now())) });
      } catch (error) {
        if (error instanceof CaptureError) throw error;
        await pauseFor(200, signal);
      }
    }
    signal.throwIfAborted();
    if (exit || spawnFailed) startupFailure();
    if (!browser) fail('The browser started but could not be controlled within 20 seconds. Retry the capture.');
    const context = browser.contexts()[0];
    const page = context?.pages().find(candidate => candidate.url() === marker);
    if (!page) fail('Could not confirm the isolated browser session.');
    connected = true;
    browser.on('disconnected', () => noteEnd({ disconnected: true }));
    page.on('close', () => noteEnd({ pageClosed: true }));
    try { await showWindow(context, page, startupSignal()); }
    catch (error) {
      signal.throwIfAborted();
      if (exit || spawnFailed) startupFailure();
      if (error instanceof CaptureError) throw error;
      fail('The browser started but did not open a window within 20 seconds.');
    }
    const waitSignal = AbortSignal.any([signal, ended.signal]);
    return { close, waitForSession: (listing, options) => waitForSession(page, context, listing, waitSignal, options) };
  } catch (error) { await close(); throw error; }
}

function captchaFrame(page) {
  return page.frames().some(frame => {
    try { return /(^|\.)captcha-delivery\.com$/.test(new URL(frame.url()).hostname); } catch { return false; }
  });
}

// Classify a polled page that is not (yet) the verified listing.
function observe(document, page) {
  if (document?.status === 403 || captchaFrame(page)) return 'challenge';
  if (!document?.ready) return undefined;
  if (document.status >= 500) return 'navigation-error';
  return 'not-listing';
}

export async function waitForSession(page, context, listing, signal, { observation, log, now = Date.now, pause = milliseconds => pauseFor(milliseconds, signal) }) {
  let document = null, noCookieSince;
  const mainDocument = request => request.isNavigationRequest() && request.frame() === page.mainFrame();
  page.on('request', request => { if (mainDocument(request)) document = { request, ready: false }; });
  page.on('response', response => {
    if (response.request() === document?.request) document = { request: response.request(), url: response.url(), status: response.status(), ready: false };
  });
  page.on('requestfailed', request => { if (request === document?.request) { document = null; observation.reason = 'navigation-error'; } });
  page.on('domcontentloaded', () => {
    if (!document?.status) return;
    document.ready = true;
    try { if (/(^|\.)leboncoin\.fr\.?$/i.test(new URL(document.url).hostname)) log('Page loaded; checking the listing…'); } catch {}
  });
  for (const url of ['https://www.leboncoin.fr/', listing.url]) {
    try { await abortable(page.goto(url, { waitUntil: 'domcontentloaded', timeout: 15000 }), signal); }
    catch (error) {
      signal.throwIfAborted();
      if (error.name !== 'TimeoutError') { document = null; observation.reason = 'navigation-error'; }
    }
  }
  while (true) {
    signal.throwIfAborted();
    const observed = document;
    let snapshot;
    try {
      snapshot = await abortable(page.evaluate(() => ({ url: location.href, data: document.querySelector('script#__NEXT_DATA__')?.textContent })), signal);
    } catch { signal.throwIfAborted(); }
    let reason;
    if (snapshot && observed === document && verifiedListing(listing, observed, snapshot)) {
      let cookies;
      try { cookies = await abortable(context.cookies(listing.url), signal); } catch { signal.throwIfAborted(); }
      if (cookies && observed === document && page.url() === snapshot.url) {
        // Cookies may arrive just after the document; keep waiting within the grace period.
        try { return sessionLine(cookies, listing, new Date(now())); } catch (error) { if (!(error instanceof CaptureError)) throw error; }
        reason = 'no-cookie';
      }
    } else if (snapshot && observed === document) reason = observe(observed, page);
    if (reason) observation.reason = reason;
    if (reason !== 'no-cookie') noCookieSince = undefined;
    else {
      if (noCookieSince === undefined) noCookieSince = now();
      if (now() - noCookieSince >= noCookieMilliseconds) fail('The listing loaded but LeBoncoin did not set a usable datadome cookie. Retry, or try another active listing.');
    }
    await pause(1000);
  }
}

export async function capture(options, listing, executablePath, dependencies) {
  const { log, chromium } = dependencies;
  const observation = { reason: 'none' };
  const controller = new AbortController();
  const signal = dependencies.signal ? AbortSignal.any([controller.signal, dependencies.signal]) : controller.signal;
  const deadline = new Date(Date.now() + options.timeoutSeconds * 1000);
  const timer = setTimeout(() => controller.abort(new CaptureError(timeoutMessage(options.timeoutSeconds, observation.reason))), options.timeoutSeconds * 1000);
  let browser;
  try {
    log(`Opening ${executablePath} with a temporary profile…`);
    browser = await (dependencies.openBrowser || openBrowser)(executablePath, signal, { connect: (...args) => chromium.connectOverCDP(...args) });
    log('A new browser window is open. If you do not see it, look for a ‘ready’ notification or switch windows (Alt+Tab / Activities).');
    log(`Waiting for you to complete any LeBoncoin verification in that window. Deadline: ${deadline.toLocaleTimeString('en-GB', { hour12: false })}. Press Ctrl+C to cancel.`);
    return await abortable(browser.waitForSession(listing, { observation, log }), signal);
  } finally {
    clearTimeout(timer);
    await browser?.close();
  }
}

export async function main(args = process.argv.slice(2), dependencies = {}) {
  const log = dependencies.error || (message => console.error(message));
  const print = dependencies.output || (line => process.stdout.write(line + '\n'));
  log('PriceFollower LeBoncoin session capture — checking prerequisites…');
  const controller = new AbortController();
  let exitCode = 1;
  const interrupt = () => { exitCode = 130; controller.abort(new CaptureError('Capture cancelled.')); };
  const terminate = () => { exitCode = 143; controller.abort(new CaptureError('Capture cancelled.')); };
  process.once('SIGINT', interrupt);
  process.once('SIGTERM', terminate);
  try {
    checkRuntime(dependencies.nodeVersion || process.versions.node);
    const options = parseArguments(args);
    if (options.help) { print(usage); return 0; }
    const listing = parseListingURL(options.url);
    const chromium = await loadPlaywright(dependencies.importPlaywright);
    const env = dependencies.env || process.env;
    const platform = dependencies.platform || process.platform;
    checkDisplay(env, platform);
    const executablePath = await findBrowser(options.browser, { env, platform, access: dependencies.access, chromium });
    controller.signal.throwIfAborted();
    const line = await capture(options, listing, executablePath, { log, chromium, signal: controller.signal, openBrowser: dependencies.openBrowser });
    log('Session captured. Paste the next line into Settings › LeBonCoin session in PriceFollower. Keep it private: do not share it in chats, tickets or logs.');
    print(line);
    return 0;
  } catch (error) {
    // Raw error messages may contain URLs or cookie material, so only fixed messages are shown.
    log(controller.signal.aborted ? 'Capture cancelled.' : error instanceof CaptureError ? error.message : `Capture failed because of an unexpected error (${error?.name || 'Error'}).`);
    return exitCode;
  } finally {
    process.removeListener('SIGINT', interrupt);
    process.removeListener('SIGTERM', terminate);
  }
}
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) process.exitCode = await main();
