import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { EventEmitter } from 'node:events';
import { spawn } from 'node:child_process';
import * as helper from './capture-leboncoin-session.mjs';

const {
  parseArguments, parseListingURL, verifiedListing, sessionLine, checkRuntime, loadPlaywright, checkDisplay,
  findBrowser, browserEndMessage, timeoutMessage, openBrowser, waitForSession, capture, main,
} = helper;

const url = 'https://www.leboncoin.fr/ad/voitures/3245888872';
const listing = { url, id: '3245888872' };
const cookie = { name: 'datadome', value: 'synthetic-secret', domain: '.leboncoin.fr', path: '/', secure: true, expires: -1 };
const data = JSON.stringify({ props: { pageProps: { ad: { list_id: 3245888872, status: 'active' } } } });
const linuxEnv = { DISPLAY: ':1', PATH: '/usr/bin' };

async function temporary(t) {
  const dir = await fs.mkdtemp(path.join(os.tmpdir(), 'capture-test-'));
  await fs.chmod(dir, 0o700);
  t.after(() => fs.rm(dir, { recursive: true, force: true }));
  return dir;
}

test('canonicalizes supported listings without WHATWG normalization broadening acceptance', () => {
  assert.deepEqual(parseListingURL(' HTTPS://LEBONCOIN.FR.:443/ad/Voitures/3245888872,?x=y#z '), listing);
  assert.deepEqual(parseListingURL('https://www.leboncoin.fr/AD/Voitures/3245888872'), listing);
  assert.equal(parseListingURL('https://leboncoin.fr/ad/%76oitures/42').id, '42');
  for (const raw of ['http://leboncoin.fr/ad/a/1', 'https://other.fr/ad/a/1', 'https://@leboncoin.fr/ad/a/1', 'https://leboncoin.fr:444/ad/a/1', 'https://leboncoin.fr/a/../ad/a/1', 'https://leboncoin.fr/ad/a/../a/1', 'https://leboncoin.fr\\ad/a/1', 'https://leboncoin.fr/ad/a/1/']) {
    assert.throws(() => parseListingURL(raw), /listing URL/);
  }
});

test('requires matching successful active document, never a challenge or rounded ID', () => {
  assert.equal(verifiedListing(listing, { url, status: 200, ready: true }, { url, data }), true);
  for (const document of [null, { url, status: 403, ready: true }, { url, status: 200, ready: false }, { url: url + '9', status: 200, ready: true }]) {
    assert.equal(verifiedListing(listing, document, { url, data }), false);
  }
  for (const badData of ['{}', data.replace('3245888872', '12'), data.replace('active', 'inactive'), data.replace('3245888872', '9007199254740993')]) {
    assert.equal(verifiedListing(listing, { url, status: 200, ready: true }, { url, data: badData }), false);
  }
});

test('session line contains only the single valid applicable datadome cookie', () => {
  assert.equal(sessionLine([cookie, { ...cookie, name: 'login', value: 'account' }], listing, new Date('2026-10-03T12:00:00Z')), 'datadome=synthetic-secret');
  assert.equal(sessionLine([{ ...cookie, expires: Date.parse('2027-01-01T00:00:00Z') / 1000 }], listing, new Date('2026-10-03T12:00:00Z')), 'datadome=synthetic-secret');
  for (const cookies of [[], [cookie, cookie], [{ ...cookie, expires: 1 }], [{ ...cookie, value: 'bad;value' }], [{ ...cookie, value: 'bad"value' }], [{ ...cookie, value: 'x'.repeat(4097) }], [{ ...cookie, domain: 'other.fr' }], [{ ...cookie, path: '/ad/voiture' }]]) {
    assert.throws(() => sessionLine(cookies, listing), /cookie/);
  }
});

test('CLI options: URL required, no output option, bounded timeout', () => {
  assert.deepEqual(parseArguments(['--help']), { help: true });
  assert.deepEqual(parseArguments(['--url', url]), { url, timeoutSeconds: 600 });
  assert.deepEqual(parseArguments(['--url', url, '--browser', '/b', '--timeout-seconds', '30']), { url, browser: '/b', timeoutSeconds: 30 });
  for (const args of [[], ['--unknown', 'x'], ['--url'], ['--url', url, '--url', url], ['--url', url, '--timeout-seconds', '0'], ['--url', url, '--timeout-seconds', '3601'], ['--url', url, '--timeout-seconds', '1.5'], ['--browser', '/b']]) {
    assert.throws(() => parseArguments(args), /Usage|Timeout/);
  }
  for (const args of [['--url', url, '--output', '/private/file'], ['--output=/private/file', '--url', url], ['--output']]) {
    assert.throws(() => parseArguments(args), { message: '--output is no longer supported. The session is now printed here; paste it into Settings › LeBonCoin session in PriceFollower.' });
  }
});

test('runtime check requires Node.js 20.19 or newer', () => {
  for (const version of ['20.19.0', '20.19.2', '21.0.0', '22.23.3']) checkRuntime(version);
  for (const version of ['20.18.3', '18.20.0', '16.0.0']) {
    assert.throws(() => checkRuntime(version), { message: `Node.js 20.19 or newer is required (found v${version}). Install a newer Node.js, then retry.` });
  }
});

test('missing repository dependencies produce an npm ci instruction', async () => {
  const chromium = {};
  assert.equal(await loadPlaywright(async () => ({ chromium })), chromium);
  await assert.rejects(loadPlaywright(async () => { const error = Error('Cannot find package'); error.code = 'ERR_MODULE_NOT_FOUND'; throw error; }), { message: 'Repository dependencies are missing. Run npm ci in the repository, then retry.' });
});

test('a graphical display is required on Linux', () => {
  checkDisplay({ DISPLAY: ':1' }, 'linux');
  checkDisplay({ WAYLAND_DISPLAY: 'wayland-0' }, 'linux');
  checkDisplay({}, 'darwin');
  for (const env of [{}, { DISPLAY: '', WAYLAND_DISPLAY: '' }]) {
    assert.throws(() => checkDisplay(env, 'linux'), { message: 'No graphical display was found (DISPLAY and WAYLAND_DISPLAY are not set). Run the helper from a terminal in your desktop session.' });
  }
});

test('browser discovery honours --browser, then installed Chrome/Chromium in order, then Playwright', async () => {
  const accessFor = executables => async candidate => { if (!executables.includes(candidate)) throw Object.assign(Error('missing'), { code: 'ENOENT' }); };
  const chromium = { executablePath: () => '/cache/chromium' };
  assert.equal(await findBrowser('/opt/chrome', { env: linuxEnv, platform: 'linux', access: accessFor(['/opt/chrome']), chromium }), '/opt/chrome');
  await assert.rejects(findBrowser('/nope', { env: linuxEnv, platform: 'linux', access: accessFor(['/usr/bin/google-chrome']), chromium }), { message: 'The browser at /nope was not found or is not executable.' });
  const env = { PATH: '/a:/b' };
  assert.equal(await findBrowser(undefined, { env, platform: 'linux', access: accessFor(['/a/chromium', '/b/google-chrome']), chromium }), '/b/google-chrome');
  assert.equal(await findBrowser(undefined, { env, platform: 'linux', access: accessFor(['/a/chromium-browser', '/b/google-chrome-stable']), chromium }), '/b/google-chrome-stable');
  assert.equal(await findBrowser(undefined, { env, platform: 'linux', access: accessFor(['/a/chromium-browser']), chromium }), '/a/chromium-browser');
  assert.equal(await findBrowser(undefined, { env, platform: 'linux', access: accessFor(['/cache/chromium']), chromium }), '/cache/chromium');
  assert.equal(await findBrowser(undefined, { env: {}, platform: 'darwin', access: accessFor(['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome']), chromium }), '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome');
  await assert.rejects(findBrowser(undefined, { env, platform: 'linux', access: accessFor([]), chromium }), { message: 'No Chrome or Chromium browser was found. Install Google Chrome or Chromium, or pass --browser <path>.' });
});

test('end of browser is classified by observed events', () => {
  const closed = 'The browser window was closed before verification finished. Run the helper again and leave the window open until it reports success.';
  assert.equal(browserEndMessage({ pageClosed: true }), closed);
  assert.equal(browserEndMessage({ pageClosed: true, disconnected: true, exit: { code: 0, signal: null } }), closed);
  assert.equal(browserEndMessage({ disconnected: true, exit: { code: 1, signal: null } }), 'The browser exited unexpectedly (exit code 1) before verification finished. Retry; if it repeats, start the browser manually to check it works.');
  assert.equal(browserEndMessage({ disconnected: true, exit: { code: null, signal: 'SIGSEGV' } }), 'The browser exited unexpectedly (signal SIGSEGV) before verification finished. Retry; if it repeats, start the browser manually to check it works.');
  assert.equal(browserEndMessage({ disconnected: true, exit: { code: 0, signal: null } }), 'The browser was quit before verification finished.');
  assert.equal(browserEndMessage({ disconnected: true }), 'Lost the connection to the browser before verification finished. Retry the capture.');
});

test('timeout message names the last observed state', () => {
  assert.equal(timeoutMessage(30, 'challenge'), 'Capture timed out after 30 seconds: LeBoncoin was still showing a verification page.');
  assert.equal(timeoutMessage(30, 'not-listing'), 'Capture timed out after 30 seconds: the page was not the requested active listing (check the URL or try another active listing).');
  assert.equal(timeoutMessage(30, 'navigation-error'), 'Capture timed out after 30 seconds: the listing page could not be loaded.');
  assert.equal(timeoutMessage(30, 'none'), 'Capture timed out after 30 seconds: no page loaded yet.');
});

function fakeBrowser(marker, options = {}) {
  const calls = { closes: 0, cdp: [], front: 0 };
  const browser = new EventEmitter();
  const page = new EventEmitter();
  page.url = () => marker;
  page.bringToFront = async () => { calls.front++; };
  page.goto = () => new Promise(() => {});
  const context = {
    pages: () => [page],
    newCDPSession: async () => ({
      send: async (method, params) => {
        calls.cdp.push({ method, params });
        if (method === 'Browser.getWindowForTarget') {
          if (options.noWindow) throw Error('synthetic-secret');
          return { windowId: 7 };
        }
        return {};
      },
      detach: async () => {},
    }),
  };
  browser.contexts = () => [context];
  browser.close = async () => { calls.closes++; };
  return { browser, page, context, calls };
}

function startBrowser(outcome, dependencies = {}) {
  const controller = new AbortController();
  const child = new EventEmitter();
  const state = { killed: [], profile: undefined, args: undefined, spawnOptions: undefined, marker: undefined };
  child.pid = undefined;
  child.kill = signal => { state.killed.push(signal); queueMicrotask(() => child.emit('exit', null, signal)); };
  const session = openBrowser('/usr/bin/fake-chrome', controller.signal, {
    availablePort: async () => 19876,
    spawn: (executable, values, options) => {
      state.args = values; state.spawnOptions = options;
      state.profile = values.find(value => value.startsWith('--user-data-dir=')).split('=')[1];
      state.marker = values.at(-1);
      if (outcome === 'spawn-error') queueMicrotask(() => child.emit('error', Error('synthetic-secret')));
      if (outcome === 'startup-exit') queueMicrotask(() => child.emit('exit', 1, null));
      return child;
    },
    fetch: async () => {
      if (outcome === 'cancel') controller.abort();
      if (outcome === 'spawn-error' || outcome === 'startup-exit') throw Error('connection refused');
      return { json: async () => [{ type: 'page', url: outcome === 'collision' ? 'chrome://newtab/' : state.marker }] };
    },
    connect: async () => {
      state.fake = fakeBrowser(state.marker, { noWindow: outcome === 'no-window' });
      return state.fake.browser;
    },
    ...dependencies,
  });
  return { session, child, state, controller };
}

test('startup confirms, restores and raises its own isolated window', async () => {
  const { session, state } = startBrowser('success');
  const browser = await session;
  assert.equal((await fs.stat(state.profile)).mode & 0o777, 0o700);
  assert.equal(state.spawnOptions.stdio, 'ignore');
  assert.ok(state.args.includes('--remote-debugging-port=19876'));
  assert.ok(state.args.includes('--remote-debugging-address=127.0.0.1'));
  assert.equal(state.args.some(value => /headless|enable-automation|ozone/.test(value)), false);
  assert.match(state.marker, /^data:text\/plain,pricefollower-[0-9a-f-]{36}$/);
  assert.deepEqual(state.fake.calls.cdp.map(call => call.method), ['Browser.getWindowForTarget', 'Browser.setWindowBounds']);
  assert.deepEqual(state.fake.calls.cdp[1].params, { windowId: 7, bounds: { windowState: 'normal' } });
  assert.equal(state.fake.calls.front, 1);
  await browser.close(); await browser.close();
  assert.equal(state.fake.calls.closes, 1);
  await assert.rejects(fs.stat(state.profile), { code: 'ENOENT' });
});

test('startup failures have distinct messages and remove the profile', async () => {
  const expectations = {
    collision: /debugging port collision/,
    'spawn-error': /^The browser exited during startup \(it could not be started\)\. Check that it starts from this desktop session, or pass another --browser\.$/,
    'startup-exit': /^The browser exited during startup \(exit code 1\)\. Check that it starts from this desktop session, or pass another --browser\.$/,
    'no-window': /^The browser started but did not open a window within 20 seconds\.$/,
    cancel: /aborted|cancel/i,
  };
  for (const [outcome, expected] of Object.entries(expectations)) {
    const { session, state } = startBrowser(outcome);
    await assert.rejects(session, error => { assert.match(error.message, expected); assert.equal(error.message.includes('synthetic-secret'), false); return true; });
    await assert.rejects(fs.stat(state.profile), { code: 'ENOENT' });
  }
});

test('waiting ends with the classified reason after browser events settle', async () => {
  const sequences = {
    'window closed': [['page', 'close'], ['browser', 'disconnected'], ['child', 'exit', 0, null], /window was closed/],
    crash: [['browser', 'disconnected'], ['child', 'exit', null, 'SIGSEGV'], /exited unexpectedly \(signal SIGSEGV\)/],
    'lost connection': [['browser', 'disconnected'], /Lost the connection/],
  };
  for (const [name, sequence] of Object.entries(sequences)) {
    const expected = sequence.pop();
    const { session, child, state } = startBrowser('success');
    const browser = await session;
    const waiting = browser.waitForSession(listing, { observation: { reason: 'none' }, log: () => {} });
    for (const [source, event, ...values] of sequence) {
      const emitter = source === 'page' ? state.fake.page : source === 'browser' ? state.fake.browser : child;
      emitter.emit(event, ...values);
    }
    await assert.rejects(waiting, error => { assert.match(error.message, expected, name); return true; });
    await browser.close();
    await assert.rejects(fs.stat(state.profile), { code: 'ENOENT' });
  }
});

function fakePage() {
  const page = new EventEmitter();
  const frame = { url: () => 'about:blank' };
  page.frameList = [frame];
  page.mainFrame = () => frame;
  page.frames = () => page.frameList;
  page.current = undefined;
  page.url = () => page.current;
  page.navigate = (target, status) => {
    page.current = target;
    const request = { isNavigationRequest: () => true, frame: () => frame };
    page.emit('request', request);
    page.emit('response', { request: () => request, url: () => target, status: () => status });
    page.emit('domcontentloaded');
  };
  page.goto = async target => page.navigate(target, 403);
  return page;
}

test('polling waits through a challenge, logs loaded pages and prints only a matching listing session', async () => {
  const page = fakePage();
  let reads = 0;
  const logs = [];
  const observation = { reason: 'none' };
  page.evaluate = async () => {
    reads++;
    if (reads === 1) { queueMicrotask(() => page.navigate(url, 200)); return { url, data: undefined }; }
    return { url, data };
  };
  const context = { cookies: async target => { assert.equal(target, url); return [cookie]; } };
  const result = await waitForSession(page, context, listing, new AbortController().signal, { observation, log: message => logs.push(message), pause: async () => {} });
  assert.equal(result, 'datadome=synthetic-secret');
  assert.equal(reads, 2);
  assert.ok(logs.length >= 2 && logs.every(message => message === 'Page loaded; checking the listing…'));
});

test('last observation is recorded for the timeout message', async () => {
  const cases = {
    challenge: page => { page.evaluate = async () => ({ url, data: undefined }); },
    'captcha frame': page => {
      page.goto = async target => page.navigate(target, 200);
      page.frameList = [page.mainFrame(), { url: () => 'https://geo.captcha-delivery.com/captcha/' }];
      page.evaluate = async () => ({ url, data: undefined });
    },
    'not-listing': page => { page.goto = async target => page.navigate(target, 200); page.evaluate = async () => ({ url, data: data.replace('active', 'deleted') }); },
    'navigation-error': page => {
      page.goto = async () => {
        const request = { isNavigationRequest: () => true, frame: () => page.mainFrame() };
        page.emit('request', request); page.emit('requestfailed', request);
        throw Error('net::ERR_NAME_NOT_RESOLVED');
      };
      page.evaluate = async () => { throw Error('no page'); };
    },
  };
  const expected = { challenge: 'challenge', 'captcha frame': 'challenge', 'not-listing': 'not-listing', 'navigation-error': 'navigation-error' };
  for (const [name, setup] of Object.entries(cases)) {
    const page = fakePage();
    setup(page);
    const controller = new AbortController();
    const observation = { reason: 'none' };
    let pauses = 0;
    const pause = async () => { if (++pauses === 3) controller.abort(Error('deadline')); };
    await assert.rejects(waitForSession(page, { cookies: async () => [] }, listing, controller.signal, { observation, log: () => {}, pause }), /deadline/);
    assert.equal(observation.reason, expected[name], name);
  }
});

test('a verified listing without a usable cookie for 15 seconds ends early', async () => {
  const page = fakePage();
  page.goto = async target => page.navigate(target, 200);
  page.evaluate = async () => ({ url, data });
  let clock = 0, pauses = 0;
  const pause = async () => { pauses++; clock += 1000; };
  await assert.rejects(
    waitForSession(page, { cookies: async () => [{ ...cookie, value: 'bad;value' }] }, listing, new AbortController().signal, { observation: { reason: 'none' }, log: () => {}, pause, now: () => clock }),
    { message: 'The listing loaded but LeBoncoin did not set a usable datadome cookie. Retry, or try another active listing.' },
  );
  assert.equal(pauses, 15);
});

test('capture times out with the last observed reason and closes the browser once', async () => {
  let closed = 0;
  await assert.rejects(capture({ url, timeoutSeconds: 0.05 }, listing, '/usr/bin/fake-chrome', {
    log: () => {},
    openBrowser: async () => ({ close: async () => { closed++; }, waitForSession: async (target, { observation }) => { observation.reason = 'challenge'; return new Promise(() => {}); } }),
  }), { message: 'Capture timed out after 0.05 seconds: LeBoncoin was still showing a verification page.' });
  assert.equal(closed, 1);
});

function runMain(args, overrides = {}) {
  const stderr = [], stdout = [], events = [];
  const dependencies = {
    nodeVersion: '20.19.2', env: linuxEnv, platform: 'linux',
    importPlaywright: async () => ({ chromium: { executablePath: () => '/missing' } }),
    access: async candidate => { if (candidate !== '/usr/bin/google-chrome') throw Error('missing'); },
    error: message => { stderr.push(message); events.push('stderr'); },
    output: line => { stdout.push(line); events.push('stdout'); },
    openBrowser: async () => ({ close: async () => { events.push('close'); }, waitForSession: async () => 'datadome=synthetic-secret' }),
    ...overrides,
  };
  return main(args, dependencies).then(code => ({ code, stderr, stdout, events }));
}

test('success prints instructions on stderr, then exactly one datadome line on stdout after cleanup', async () => {
  const { code, stderr, stdout, events } = await runMain(['--url', url, '--timeout-seconds', '30']);
  assert.equal(code, 0);
  assert.deepEqual(stdout, ['datadome=synthetic-secret']);
  assert.equal(stderr[0], 'PriceFollower LeBoncoin session capture — checking prerequisites…');
  assert.ok(stderr.includes('Opening /usr/bin/google-chrome with a temporary profile…'));
  assert.ok(stderr.includes('A new browser window is open. If you do not see it, look for a ‘ready’ notification or switch windows (Alt+Tab / Activities).'));
  assert.ok(stderr.some(message => /^Waiting for you to complete any LeBoncoin verification in that window\. Deadline: \d\d:\d\d:\d\d\. Press Ctrl\+C to cancel\.$/.test(message)));
  assert.equal(stderr.at(-1), 'Session captured. Paste the next line into Settings › LeBonCoin session in PriceFollower. Keep it private: do not share it in chats, tickets or logs.');
  assert.equal(stderr.join('\n').includes('synthetic-secret'), false);
  assert.deepEqual(events.slice(-3), ['close', 'stderr', 'stdout']);
});

test('every failure prints the first message, one final cause and nothing on stdout', async () => {
  const cases = [
    [['--bogus'], {}, /^Usage:/],
    [['--url', url, '--output', '/x'], {}, /--output is no longer supported/],
    [['--url', 'http://example.com/'], {}, /supported HTTPS LeBoncoin listing URL/],
    [['--url', url], { nodeVersion: '18.19.0' }, /Node\.js 20\.19 or newer is required \(found v18\.19\.0\)/],
    [['--url', url], { importPlaywright: async () => { throw Error('missing'); } }, /Run npm ci/],
    [['--url', url], { env: { PATH: '/usr/bin' } }, /No graphical display/],
    [['--url', url, '--browser', '/nope'], {}, /The browser at \/nope was not found/],
    [['--url', url], { openBrowser: async () => { throw Error('synthetic-secret https://x'); } }, /^Capture failed because of an unexpected error \(Error\)\.$/],
    [['--url', url], { openBrowser: async () => ({ close: async () => {}, waitForSession: async () => { throw Error('synthetic-secret'); } }) }, /unexpected error \(Error\)/],
  ];
  for (const [args, overrides, expected] of cases) {
    const { code, stderr, stdout } = await runMain(args, overrides);
    assert.equal(code, 1, String(expected));
    assert.deepEqual(stdout, []);
    assert.equal(stderr[0], 'PriceFollower LeBoncoin session capture — checking prerequisites…');
    assert.match(stderr.at(-1), expected);
    assert.equal(stderr.join('\n').includes('synthetic-secret'), false);
  }
  const help = await runMain(['--help']);
  assert.equal(help.code, 0);
  assert.match(help.stdout.join('\n'), /^Usage: node scripts\/capture-leboncoin-session\.mjs --url <listing-url> \[--browser <executable-path>\] \[--timeout-seconds <1-3600>\]$/);
});

function runCLI(args, env) {
  const child = spawn(process.execPath, ['scripts/capture-leboncoin-session.mjs', ...args], { stdio: ['ignore', 'pipe', 'pipe'], env: { ...process.env, ...env } });
  const result = { stdout: '', stderr: '' };
  child.stdout.on('data', chunk => { result.stdout += chunk; });
  child.stderr.on('data', chunk => { result.stderr += chunk; });
  result.finished = new Promise(resolve => child.once('exit', (exitCode, signal) => resolve({ exitCode, signal })));
  result.child = child;
  return result;
}

test('real CLI rejects --output and a missing display before launching a browser', async () => {
  const output = runCLI(['--url', url, '--output', '/tmp/x'], { DISPLAY: ':1' });
  assert.deepEqual(await output.finished, { exitCode: 1, signal: null });
  assert.equal(output.stdout, '');
  assert.match(output.stderr, /^PriceFollower LeBoncoin session capture — checking prerequisites…\n--output is no longer supported/);
  const display = runCLI(['--url', url], { DISPLAY: '', WAYLAND_DISPLAY: '' });
  assert.deepEqual(await display.finished, { exitCode: 1, signal: null });
  assert.equal(display.stdout, '');
  assert.match(display.stderr, /No graphical display was found/);
});

test('real CLI handles SIGINT and SIGTERM without printing a session or leaking profiles', async t => {
  const dir = await temporary(t);
  const executable = path.join(dir, 'browser');
  const marker = path.join(dir, 'profile-path');
  // This local stand-in only records the temporary profile path and sleeps.
  await fs.writeFile(executable, '#!/bin/sh\nfor arg in "$@"; do\n case "$arg" in --user-data-dir=*) printf "%s" "${arg#*=}" > "' + marker + '";; esac\ndone\nexec sleep 30\n', { mode: 0o700 });
  for (const [signal, code] of [['SIGINT', 130], ['SIGTERM', 143]]) {
    await fs.rm(marker, { force: true });
    const run = runCLI(['--url', url, '--browser', executable], { DISPLAY: ':99' });
    const end = Date.now() + 5000;
    let profile;
    while (!profile && Date.now() < end) {
      try { profile = await fs.readFile(marker, 'utf8'); } catch {}
      if (!profile) await new Promise(resolve => setTimeout(resolve, 30));
    }
    if (!profile) { run.child.kill('SIGKILL'); await run.finished; assert.fail('Test browser did not start: ' + run.stderr); }
    run.child.kill(signal);
    assert.deepEqual(await run.finished, { exitCode: code, signal: null });
    assert.match(run.stderr, /Opening .* with a temporary profile…/);
    assert.match(run.stderr, /Capture cancelled\.\n$/);
    assert.equal(run.stdout, '');
    await assert.rejects(fs.stat(profile), { code: 'ENOENT' });
  }
});

test('an abort during the polling pause reports its own reason, not a generic AbortError', async () => {
  const page = fakePage();
  page.evaluate = async () => ({ url, data: undefined });
  const controller = new AbortController();
  const waiting = waitForSession(page, { cookies: async () => [] }, listing, controller.signal, { observation: { reason: 'none' }, log: () => {} });
  setTimeout(() => controller.abort(Object.assign(Error('The browser exited unexpectedly (signal SIGSEGV) before verification finished.'), { name: 'CaptureError' })), 100);
  await assert.rejects(waiting, { message: /exited unexpectedly \(signal SIGSEGV\)/ });
});

test('source avoids logical assignment so older Node.js can parse it and report the runtime message', async () => {
  const source = await fs.readFile(new URL('./capture-leboncoin-session.mjs', import.meta.url), 'utf8');
  assert.equal(/\?\?=|\|\|=|&&=/.test(source), false);
});
