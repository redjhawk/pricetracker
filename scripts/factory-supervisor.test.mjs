// Tests of the agent factory supervisor's decisions (scripts/factory/supervisor.mjs) and of the usage-limit
// reset parser in scripts/factory/lib.sh. Offline: no GitHub, no Claude.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import test from 'node:test';
import { applyGlobalLimit, classify, classifyProbe, decide, FACTORY_MARKER, HOLD_LABEL } from './factory/supervisor.mjs';

const NOW = '2026-11-22T12:00:00Z';
const ago = (minutes) => new Date(Date.parse(NOW) - minutes * 60_000).toISOString();
const stop = (minutes, marker) => ({ author: 'owner', createdAt: ago(minutes),
  body: `${FACTORY_MARKER}\n<!-- factory-stop ${JSON.stringify(marker)} -->\nStopped.` });
const owner = (minutes, body = 'continue') => ({ author: 'owner', createdAt: ago(minutes), body });
const sup = (minutes, meta) => ({ author: 'owner', createdAt: ago(minutes), body: `<!-- factory-supervisor ${JSON.stringify(meta)} -->\nRelaunching.` });
const run = (minutes, conclusion = 'success', status = 'completed') => ({ createdAt: ago(minutes), updatedAt: ago(minutes - 1), conclusion, status });
const relaunched = (minutes, progress, attempt) => sup(minutes, { action: 'relaunch', reason: 'error', progress, counted: true, attempt });
const snapshot = (over = {}) => ({ now: NOW, labels: [], comments: [], runs: [], runsSince: null, trusted: ['owner'], blockers: {}, ...over });
const error = { reason: 'error', stage: 'implement', round: 1 };

test('markers count only in comments of trusted authors', () => {
  const forged = { ...stop(5, error), author: 'stranger' };
  assert.equal(classify(forged, ['owner']).kind, 'other');
  assert.equal(classify(stop(5, error), ['owner']).kind, 'stop');
  assert.equal(classify(owner(5), ['owner']).kind, 'owner');
  assert.equal(classify({ ...owner(5), body: `${FACTORY_MARKER}\nquestions` }, ['owner']).kind, 'factory');
});

test('an issue on hold or with a run in progress is left alone', () => {
  const s = { comments: [stop(60, error)], runs: [run(70, 'failure')] };
  assert.equal(decide(snapshot({ ...s, labels: [HOLD_LABEL] })).action, 'none');
  assert.match(decide(snapshot({ ...s, runs: [run(70, 'failure'), run(1, null, 'in_progress')] })).why, /in progress/);
});

test('a usage limit is relaunched after its reset, without counting as a retry', () => {
  const limit = (resetsAt) => snapshot({ runs: [run(30, 'failure')], comments: [stop(20, { reason: 'usage-limit', stage: 'tests', round: 0, resets_at: resetsAt })] });
  const waiting = decide(limit('2026-11-22T13:10:00Z'));
  assert.equal(waiting.action, 'none');
  assert.equal(waiting.limitUntil, Date.parse('2026-11-22T13:15:00Z'));
  assert.equal(decide(limit('2026-11-22T11:57:00Z')).action, 'none', 'within the 5-minute margin');
  const after = decide(limit('2026-11-22T11:40:00Z'));
  assert.deepEqual([after.action, after.reason, after.counted], ['relaunch', 'usage-limit', false]);
  assert.equal(decide(limit(undefined)).action, 'relaunch', 'unknown reset: the probe decides');
});

test('an error is retried with a growing delay, then the supervisor gives up', () => {
  const base = { runs: [run(30, 'failure')], comments: [stop(10, error)] };
  assert.match(decide(snapshot(base)).why, /retry 1 of 3 after/);
  const first = decide(snapshot({ ...base, comments: [stop(16, error)] }));
  assert.deepEqual([first.action, first.attempt, first.progress, first.counted], ['relaunch', 1, 'implement/1', true]);

  const retried = (n, stopAgo) => ({ runs: [run(stopAgo + 5, 'failure')],
    comments: [...Array.from({ length: n }, (_, i) => relaunched(300 - i * 10, 'implement/1', i + 1)), stop(stopAgo, error)] });
  assert.match(decide(snapshot(retried(2, 40))).why, /retry 3 of 3 after/, 'the third retry waits 45 minutes');
  assert.equal(decide(snapshot(retried(2, 46))).attempt, 3);
  assert.equal(decide(snapshot(retried(3, 200))).action, 'give-up');
});

test('progress to another stage or round, or an owner comment, starts the retry count again', () => {
  const comments = [relaunched(300, 'implement/0', 1), relaunched(290, 'implement/0', 2), relaunched(280, 'implement/0', 3)];
  assert.equal(decide(snapshot({ runs: [run(70, 'failure')], comments: [...comments, stop(60, error)] })).attempt, 1);
  const answered = [...comments.map((c) => ({ ...c, body: c.body.replace('implement/0', 'implement/1') })), owner(100), stop(60, error)];
  assert.equal(decide(snapshot({ runs: [run(70, 'failure')], comments: answered })).attempt, 1);
});

test('after giving up, only an owner comment brings the supervisor back', () => {
  const gaveUp = sup(50, { action: 'give-up', reason: 'error', progress: 'implement/1' });
  assert.match(decide(snapshot({ runs: [run(70, 'failure')], comments: [stop(60, error), gaveUp] })).why, /gave up/);
  const resumed = [stop(60, error), gaveUp, owner(40), stop(20, error)];
  assert.equal(decide(snapshot({ runs: [run(39, 'failure')], comments: resumed })).attempt, 1);
});

test('a run that failed without reporting a stop is a crash; a success or a cancellation is not', () => {
  const s = (conclusion) => snapshot({ labels: ['ai-dev', 'factory:review'], runs: [run(30, conclusion)], comments: [stop(500, error)] });
  const crash = decide(s('failure'));
  assert.deepEqual([crash.action, crash.reason, crash.progress], ['relaunch', 'crash', 'review/']);
  assert.equal(decide(s('success')).action, 'none');
  assert.equal(decide(s('cancelled')).action, 'none');
});

test('a base-bug stop waits until the blocking issue is closed', () => {
  const s = (state) => snapshot({ runs: [run(30)], comments: [stop(20, { reason: 'base-bug', stage: 'implement', round: 0, waiting_on: 99 })], blockers: { 99: state } });
  assert.match(decide(s('OPEN')).why, /waiting for #99/);
  const resumed = decide(s('CLOSED'));
  assert.deepEqual([resumed.action, resumed.reason, resumed.counted], ['relaunch', 'base-bug', true]);
});

test('stops that need the owner are not relaunched', () => {
  for (const reason of ['questions', 'gate', 'test-problem', 'max-rounds', 'sensitive', 'security', 'merge-conflict']) {
    assert.equal(decide(snapshot({ runs: [run(300)], comments: [stop(200, { reason, stage: 'x', round: 0 })] })).action, 'none', reason);
  }
});

test('an owner comment that started no run is relaunched once its grace period is over', () => {
  const s = (commentAgo, runs) => snapshot({ runs, comments: [stop(100, { reason: 'gate', stage: 'testcases', round: 0 }), owner(commentAgo)] });
  assert.equal(decide(s(5, [run(110)])).action, 'none');
  const lost = decide(s(20, [run(110)]));
  assert.deepEqual([lost.action, lost.reason], ['relaunch', 'owner-comment']);
  assert.equal(decide(s(20, [run(110), run(19)])).action, 'none', 'its run happened');
  const relaunchedOnce = snapshot({ runs: [run(110)], comments: [owner(60), sup(40, { action: 'relaunch', reason: 'owner-comment', progress: '' })] });
  assert.notEqual(decide(relaunchedOnce).reason, 'owner-comment', 'relaunched once only');
  assert.equal(decide({ ...s(20, [run(10)]), runsSince: ago(15) }).action, 'none', 'older than the runs fetched');
});

test('a relaunch whose run has not started yet is not repeated', () => {
  const s = (relaunchAgo) => snapshot({ runs: [run(80, 'failure')], comments: [stop(70, error), relaunched(relaunchAgo, 'implement/1', 1)] });
  assert.match(decide(s(5)).why, /waiting for the run to start/);
  assert.equal(decide(s(40)).attempt, 2, 'the run never came: it counts as a retry');
});

test('while one issue waits for the usage limit, no other issue is relaunched', () => {
  const decisions = [
    { issue: 1, action: 'none', why: 'usage limit', limitUntil: Date.parse('2026-11-22T13:00:00Z') },
    { issue: 2, action: 'relaunch', reason: 'error', why: 'stopped with error' },
    { issue: 3, action: 'give-up', reason: 'error', why: 'tried' },
  ];
  assert.deepEqual(applyGlobalLimit(decisions, NOW).map((d) => d.action), ['none', 'none', 'give-up']);
  assert.deepEqual(applyGlobalLimit(decisions, '2026-11-22T14:00:00Z').map((d) => d.action), ['none', 'relaunch', 'give-up']);
});

test('the probe passes only when the agent printed the nonce it read with Bash', () => {
  assert.deepEqual(classifyProbe({ stdout: 'abc123\n' }, 'abc123'), { ok: true });
  const limit = classifyProbe({ stdout: "You've hit your session limit · resets 3pm (UTC)" }, 'abc123');
  assert.deepEqual([limit.ok, limit.limit], [false, true]);
  const broken = classifyProbe({ stdout: 'bwrap: Failed to make / slave: Permission denied' }, 'abc123');
  assert.deepEqual([broken.ok, broken.limit], [false, false]);
  assert.match(broken.detail, /bwrap/);
  assert.match(classifyProbe({ error: new Error('spawnSync claude ENOENT') }, 'x').detail, /ENOENT/);
});

test('lib.sh reads the reset time of a usage-limit message', () => {
  const resets = (message) => execFileSync('bash', ['-c', 'source scripts/factory/lib.sh; limit_resets_at "$1"', '_', message], { encoding: 'utf8' }).trim();
  assert.equal(resets("You've hit your weekly limit · resets Nov 24 2099, 3pm (Europe/Paris)"), '2099-11-24T14:00:00Z');
  const daily = Date.parse(resets("You've hit your session limit · resets 11:10pm (UTC)"));
  assert.ok(daily > Date.now() && daily <= Date.now() + 24 * 3600_000, 'today or tomorrow');
  assert.equal(new Date(daily).toISOString().slice(11, 16), '23:10');
  assert.equal(resets('Claude failed'), '');
});
