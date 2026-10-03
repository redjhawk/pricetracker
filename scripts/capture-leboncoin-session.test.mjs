import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { parseListingURL, verifiedListing, sessionExport, validateDestination, writeSession, capture } from './capture-leboncoin-session.mjs';
const url = 'https://www.leboncoin.fr/ad/voitures/3245888872';
const listing = { url, id: '3245888872' };
const cookie = { name: 'datadome', value: 'synthetic-secret', domain: '.leboncoin.fr', path: '/', secure: true, expires: -1 };
const data = JSON.stringify({ props: { pageProps: { ad: { list_id: 3245888872, status: 'active' } } } });
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
test('exports only one valid applicable datadome cookie and minimal metadata', () => {
  const result = sessionExport([cookie, { ...cookie, name: 'login' }], listing, new Date('2026-10-03T12:00:00Z'));
  assert.deepEqual(result, { version: 1, capturedAt: '2026-10-03T12:00:00.000Z', cookie: { name: 'datadome', value: cookie.value, domain: cookie.domain, path: '/', secure: true, expiresAt: null } });
  for (const cookies of [[], [cookie, cookie], [{ ...cookie, expires: 1 }], [{ ...cookie, value: 'bad;value' }], [{ ...cookie, domain: 'other.fr' }], [{ ...cookie, path: '/ad/voiture' }]]) {
    assert.throws(() => sessionExport(cookies, listing), /cookie/);
  }
});
test('rejects unsafe destinations and preserves old file on failed atomic write', async t => {
  const dir = await temporary(t);
  const output = path.join(dir, 'session.json');
  await fs.writeFile(output, 'old', { mode: 0o600 });
  const aborted = AbortSignal.abort();
  await assert.rejects(writeSession(output, { cookie: 'new' }, aborted));
  assert.equal(await fs.readFile(output, 'utf8'), 'old');
  await fs.chmod(output, 0o644);
  await assert.rejects(validateDestination(output));
  await fs.chmod(output, 0o600);
  const link = path.join(dir, 'link');
  await fs.symlink(output, link);
  await assert.rejects(validateDestination(link));
  await fs.chmod(dir, 0o755);
  await assert.rejects(validateDestination(output));
});
test('atomic renewal writes a private complete file', async t => {
  const dir = await temporary(t);
  const output = path.join(dir, 'session.json');
  await fs.writeFile(output, 'old', { mode: 0o600 });
  const value = sessionExport([cookie], listing);
  await writeSession(output, value);
  assert.deepEqual(JSON.parse(await fs.readFile(output, 'utf8')), value);
  assert.equal((await fs.stat(output)).mode & 0o777, 0o600);
  assert.deepEqual(await fs.readdir(dir), ['session.json']);
});
test('capture always cleans up its browser after unsuccessful verification', async t => {
  const dir = await temporary(t);
  let cleaned = false;
  await assert.rejects(capture({ url, output: path.join(dir, 'session.json'), timeoutSeconds: 1 }, {
    openBrowser: async () => ({ waitForSession: async () => { throw Error('synthetic-secret'); }, close: async () => { cleaned = true; } }),
    log: () => {},
  }));
  assert.equal(cleaned, true);
});

test('CLI options reject missing, unknown and duplicate values', async () => {
  const { parseArguments } = await import('./capture-leboncoin-session.mjs');
  assert.deepEqual(parseArguments(['--help']), { help: true });
  assert.equal(parseArguments(['--url', url, '--output', '/private/file']).timeoutSeconds, 600);
  for (const args of [[], ['--unknown', 'x'], ['--url'], ['--url', url, '--url', url], ['--url', url, '--output', '/x', '--timeout-seconds', '0'], ['--url', url, '--output', '/x', '--timeout-seconds', '3601'], ['--url', url, '--output', '/x', '--timeout-seconds', '1.5']]) assert.throws(() => parseArguments(args));
});
test('rejects repository destinations, symlink parents and nonregular outputs', async t => {
  const dir = await temporary(t);
  await assert.rejects(validateDestination(path.resolve('session.json')));
  const nested = path.join(dir, 'nested');
  await fs.mkdir(nested, { mode: 0o700 });
  const linked = path.join(dir, 'linked');
  await fs.symlink(nested, linked);
  await assert.rejects(validateDestination(path.join(linked, 'session.json')));
  await assert.rejects(validateDestination(nested));
});
test('failed writes and renames retain existing export and remove temporary output', async t => {
  const dir = await temporary(t);
  const output = path.join(dir, 'session.json');
  await fs.writeFile(output, 'previous', { mode: 0o600 });
  for (const stage of ['writeFile', 'sync', 'rename']) {
    const io = {
      open: async (...args) => {
        const handle = await fs.open(...args);
        return { writeFile: stage === 'writeFile' ? async () => { throw Error('synthetic-secret'); } : handle.writeFile.bind(handle), sync: stage === 'sync' ? async () => { throw Error('disk full'); } : handle.sync.bind(handle), close: handle.close.bind(handle) };
      },
      rename: stage === 'rename' ? async () => { throw Error('rename failure'); } : fs.rename,
    };
    await assert.rejects(writeSession(output, sessionExport([cookie], listing), undefined, io));
    assert.equal(await fs.readFile(output, 'utf8'), 'previous');
    assert.deepEqual(await fs.readdir(dir), ['session.json']);
  }
});
test('success, cancellation and deadline each clean up once; failures preserve old export', async t => {
  const dir = await temporary(t);
  const output = path.join(dir, 'session.json');
  for (const outcome of ['success', 'cancel', 'timeout']) {
    await fs.writeFile(output, 'previous', { mode: 0o600 });
    let closed = 0;
    const controller = new AbortController();
    const logs = [];
    const operation = capture({ url, output, timeoutSeconds: outcome === 'timeout' ? 0.02 : 1 }, {
      signal: controller.signal,
      log: message => logs.push(message),
      openBrowser: async () => ({ close: async () => { closed++; }, waitForSession: async () => {
        if (outcome === 'success') return sessionExport([cookie], listing);
        if (outcome === 'cancel') controller.abort();
        return new Promise(() => {});
      } }),
    });
    if (outcome === 'success') await operation;
    else { await assert.rejects(operation); assert.equal(await fs.readFile(output, 'utf8'), 'previous'); }
    assert.equal(closed, 1);
    assert.equal(logs.join('\n').includes(cookie.value), false);
  }
});

test('startup uses own visible isolated target; collision and failed spawn remove profiles', async () => {
  const { EventEmitter } = await import('node:events');
  const { openBrowser } = await import('./capture-leboncoin-session.mjs');
  const markers = [];
  for (const outcome of ['success', 'collision', 'spawn-error', 'cdp-error', 'cancel']) {
    let profile, args, spawnOptions, closes = 0, killed = [];
    const controller = new AbortController();
    const child = new EventEmitter();
    child.kill = signal => { killed.push(signal); queueMicrotask(() => child.emit('exit', 0)); };
    let marker;
    const session = openBrowser({ browser: process.execPath }, controller.signal, {
      availablePort: async () => 19876,
      spawn: (executable, values, options) => {
        args = values; spawnOptions = options;
        profile = values.find(value => value.startsWith('--user-data-dir=')).split('=')[1];
        marker = values.at(-1);
        markers.push(marker);
        if (outcome === 'spawn-error') queueMicrotask(() => child.emit('error', Error('synthetic-secret')));
        return child;
      },
      fetch: async () => {
        if (outcome === 'cancel') controller.abort();
        return { json: async () => [{ type: 'page', url: outcome === 'collision' ? 'chrome://newtab/' : marker }] };
      },
      connect: async () => {
        if (outcome === 'cdp-error') { controller.abort(); throw Error('synthetic-secret'); }
        return { contexts: () => [{ pages: () => [{ url: () => marker }] }], close: async () => { closes++; } };
      },
    });
    if (outcome === 'success') {
      const browser = await session;
      assert.equal((await fs.stat(profile)).mode & 0o777, 0o700);
      assert.equal(spawnOptions.stdio, 'ignore');
      assert.ok(args.includes('--remote-debugging-port=19876'));
      assert.ok(args.includes('--remote-debugging-address=127.0.0.1'));
      assert.equal(args.some(value => /headless|enable-automation/.test(value)), false);
      await browser.close(); await browser.close();
      assert.equal(closes, 1);
    } else await assert.rejects(session);
    await assert.rejects(fs.stat(profile), { code: 'ENOENT' });
    assert.ok(killed.length > 0);
  }
  for (const value of markers) assert.match(value, /^data:text\/plain,pricefollower-[0-9a-f-]{36}$/);
  assert.equal(new Set(markers).size, markers.length);
});

test('actual polling gate waits through challenge and only exports a matching loaded document', async () => {
  const { EventEmitter } = await import('node:events');
  const { waitForSession } = await import('./capture-leboncoin-session.mjs');
  const page = new EventEmitter();
  const frame = {};
  const controller = new AbortController();
  let currentURL, reads = 0;
  page.mainFrame = () => frame;
  page.url = () => currentURL;
  page.isClosed = () => false;
  const navigate = (target, status) => {
    currentURL = target;
    const request = { isNavigationRequest: () => true, frame: () => frame };
    page.emit('request', request);
    page.emit('response', { request: () => request, url: () => target, status: () => status });
    page.emit('domcontentloaded');
  };
  page.goto = async target => navigate(target, 403);
  page.evaluate = async () => {
    reads++;
    if (reads === 1) { queueMicrotask(() => navigate(url, 200)); return { url, data: '{}' }; }
    return { url, data };
  };
  const result = await waitForSession(page, { cookies: async target => { assert.equal(target, url); return [cookie]; } }, listing, controller.signal);
  assert.equal(reads, 2);
  assert.equal(result.cookie.value, cookie.value);
});

test('real CLI handles SIGINT and SIGTERM without exporting or leaking child profiles', async t => {
  const { spawn } = await import('node:child_process');
  const dir = await temporary(t);
  const executable = path.join(dir, 'browser');
  const marker = path.join(dir, 'profile-path');
  // This local stand-in only records the temporary profile path and sleeps.
  await fs.writeFile(executable, '#!/bin/sh\nfor arg in "$@"; do\n case "$arg" in --user-data-dir=*) printf "%s" "${arg#*=}" > "' + marker + '";; esac\ndone\nexec sleep 30\n', { mode: 0o700 });
  for (const [signal, code] of [['SIGINT', 130], ['SIGTERM', 143]]) {
    await fs.rm(marker, { force: true });
    const output = path.join(dir, 'session.json');
    await fs.writeFile(output, 'previous', { mode: 0o600 });
    const child = spawn(process.execPath, ['scripts/capture-leboncoin-session.mjs', '--url', url, '--output', output, '--browser', executable], { stdio: ['ignore', 'pipe', 'pipe'] });
    let logs = '';
    child.stdout.on('data', chunk => { logs += chunk; });
    child.stderr.on('data', chunk => { logs += chunk; });
    const finished = new Promise(resolve => child.once('exit', (exitCode, killed) => resolve({ exitCode, killed })));
    const end = Date.now() + 5000;
    let profile;
    while (!profile && Date.now() < end) {
      try { profile = await fs.readFile(marker, 'utf8'); } catch {}
      if (!profile) await new Promise(resolve => setTimeout(resolve, 30));
    }
    if (!profile) { child.kill('SIGKILL'); await finished; assert.fail('Test browser did not start: ' + logs); }
    child.kill(signal);
    const result = await finished;
    assert.deepEqual(result, { exitCode: code, killed: null });
    assert.match(logs, /Capture cancelled/);
    assert.equal(logs.includes(cookie.value), false);
    assert.equal(await fs.readFile(output, 'utf8'), 'previous');
    await assert.rejects(fs.stat(profile), { code: 'ENOENT' });
  }
});
