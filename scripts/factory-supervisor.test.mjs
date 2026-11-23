// Tests of the agent factory supervisor's decisions (scripts/factory/supervisor.mjs) and of the usage-limit
// reset parser and the deletion list in scripts/factory/lib.sh. Offline: no GitHub, no Claude.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { existsSync, mkdirSync, mkdtempSync, realpathSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';
import {
  applyGlobalLimit, classify, classifyProbe, decide, FACTORY_MARKER, freeSlots, HOLD_LABEL, labeledAt, launchCandidates,
  launchQueue, parseRanking, rankPrompt,
} from './factory/supervisor.mjs';

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

test('an error is retried with a growing delay, then repaired', () => {
  const base = { runs: [run(30, 'failure')], comments: [stop(10, error)] };
  assert.match(decide(snapshot(base)).why, /retry 1 of 3 after/);
  const first = decide(snapshot({ ...base, comments: [stop(16, error)] }));
  assert.deepEqual([first.action, first.attempt, first.progress, first.counted], ['relaunch', 1, 'implement/1', true]);

  const retried = (n, stopAgo) => ({ runs: [run(stopAgo + 5, 'failure')],
    comments: [...Array.from({ length: n }, (_, i) => relaunched(300 - i * 10, 'implement/1', i + 1)), stop(stopAgo, error)] });
  assert.match(decide(snapshot(retried(2, 40))).why, /retry 3 of 3 after/, 'the third retry waits 45 minutes');
  assert.equal(decide(snapshot(retried(2, 46))).attempt, 3);
  const spent = decide(snapshot(retried(3, 200)));
  assert.deepEqual([spent.action, spent.attempt, spent.progress, spent.busy], ['repair', 1, 'implement/1', true]);
});

const repairing = (minutes, progress, attempt = 1) => sup(minutes, { action: 'repair', reason: 'gate', progress, attempt });
const gate = { reason: 'gate', stage: 'testcases', round: 0 };

test('a failed gate or a blocked environment is repaired at once', () => {
  for (const reason of ['gate', 'blocked-env']) {
    const d = decide(snapshot({ runs: [run(30)], comments: [stop(20, { ...gate, reason })] }));
    assert.deepEqual([d.action, d.reason, d.progress, d.attempt, d.busy], ['repair', reason, 'testcases/0', 1, true], reason);
  }
});

test('a running repair is waited for, and taken as failed once it is too old', () => {
  const s = (repairAgo) => snapshot({ runs: [run(repairAgo + 20)], comments: [stop(repairAgo + 10, gate), repairing(repairAgo, 'testcases/0')] });
  const running = decide(s(30));
  assert.deepEqual([running.action, running.busy], ['none', true]);
  assert.match(running.why, /repair is running/);
  assert.match(decide(s(179)).why, /repair is running/);
  const stale = decide(s(181));
  assert.deepEqual([stale.action, stale.attempt], ['repair', 2]);
});
test('after a repair the issue is relaunched; two repairs at the same stage, then the supervisor gives up', () => {
  const repaired = sup(100, { action: 'relaunch', reason: 'repaired', progress: 'testcases/0', counted: false, attempt: 0 });
  assert.match(decide(snapshot({ runs: [run(300)], comments: [stop(290, gate), repairing(200, 'testcases/0'), { ...repaired, createdAt: ago(5) }] })).why,
    /waiting for the run to start/);
  const again = [stop(290, gate), repairing(200, 'testcases/0'), repaired, stop(60, gate)];
  assert.equal(decide(snapshot({ runs: [run(90)], comments: again })).attempt, 2);
  const twice = [...again, repairing(50, 'testcases/0', 2), { ...repaired, createdAt: ago(40) }, stop(10, gate)];
  const d = decide(snapshot({ runs: [run(35)], comments: twice }));
  assert.equal(d.action, 'give-up');
  assert.match(d.why, /2 repairs did not help/);
  assert.equal(decide(snapshot({ runs: [run(35)], comments: [...twice.slice(0, -1), stop(10, { ...gate, stage: 'techspec' })] })).action, 'repair',
    'another stage starts the count again');
});

test('a repair failure is a give-up: only the owner brings the supervisor back', () => {
  const failed = sup(30, { action: 'give-up', reason: 'repair-failed', progress: 'testcases/0' });
  assert.match(decide(snapshot({ runs: [run(300)], comments: [stop(290, gate), repairing(200, 'testcases/0'), failed] })).why, /gave up/);
});

test('no repair while the usage limit holds', () => {
  const ds = applyGlobalLimit([{ action: 'repair', why: 'x' }, { action: 'none', limitUntil: Date.parse(NOW) + 60_000 }], NOW);
  assert.equal(ds[0].action, 'none');
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
  for (const reason of ['questions', 'test-problem', 'max-rounds', 'sensitive', 'security', 'merge-conflict']) {
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

test('an issue takes a place while the factory goes on with it without the owner', () => {
  const busy = (over) => decide(snapshot(over)).busy;
  assert.equal(busy({ runs: [run(1, null, 'in_progress')] }), true, 'running');
  assert.equal(busy({ runs: [run(30, 'failure')], comments: [stop(10, error)] }), true, 'retry scheduled');
  assert.equal(busy({ runs: [run(30, 'failure')], comments: [stop(20, { reason: 'usage-limit', stage: 'tests', round: 0, resets_at: '2026-11-22T13:00:00Z' })] }), true, 'usage limit');
  assert.equal(busy({ runs: [run(30)], comments: [stop(20, { reason: 'base-bug', stage: 'implement', round: 0, waiting_on: 9 })], blockers: { 9: 'OPEN' } }), true, 'base bug');
  assert.equal(busy({ runs: [run(30)], comments: [stop(20, { reason: 'questions', stage: 'spec', round: 0 })] }), false, 'questions');
  assert.equal(busy({ runs: [run(70, 'failure')], comments: [stop(60, error), sup(50, { action: 'give-up', reason: 'error', progress: 'implement/1' })] }), false, 'gave up');
  assert.equal(busy({ labels: [HOLD_LABEL], runs: [run(1, null, 'in_progress')] }), false, 'on hold');
  assert.equal(busy({ runs: [run(30)] }), false, 'finished without a stop');
  const failing = { runs: [run(400, 'failure')], comments: [...[1, 2, 3].map((i) => relaunched(390 - i * 60, 'implement/1', i)), stop(60, error)] };
  assert.equal(decide(snapshot(failing)).action, 'repair');
  assert.equal(busy(failing), true, 'the supervisor repairs it');
  const repairedTwice = { ...failing, comments: [...failing.comments.slice(0, -1), repairing(300, 'implement/1'), repairing(200, 'implement/1', 2), failing.comments.at(-1)] };
  assert.equal(busy(repairedTwice), false, 'the supervisor gives up after two repairs');
});

test('free places count only the busy issues', () => {
  const d = [{ busy: true }, { busy: false }, { busy: false }];
  assert.equal(freeSlots(d, 1), 0);
  assert.equal(freeSlots(d, 3), 2);
  assert.equal(freeSlots([], 1), 1);
  assert.equal(freeSlots([{ busy: true }, { busy: true }], 1), 0, 'never negative');
});

test('a launched or labelled issue whose run never started is started again, then given up', () => {
  const launched = (minutes) => sup(minutes, { action: 'launch' });
  assert.match(decide(snapshot({ comments: [launched(5)] })).why, /launched, waiting/);
  assert.equal(decide(snapshot({ comments: [launched(5)] })).busy, true);
  const first = decide(snapshot({ comments: [owner(900, 'an old idea'), launched(16)] }));
  assert.deepEqual([first.action, first.reason, first.attempt, first.counted], ['relaunch', 'not-started', 1, true]);
  const notStarted = (i, minutes) => sup(minutes, { action: 'relaunch', reason: 'not-started', progress: '/', counted: true, attempt: i });
  const third = decide(snapshot({ comments: [launched(200), notStarted(1, 180), notStarted(2, 30)] }));
  assert.equal(decide(snapshot({ comments: [launched(200), notStarted(1, 180), notStarted(2, 120)] })).attempt, 3);
  assert.match(third.why, /retry 3 of 3 after/);
  const gaveUp = decide(snapshot({ comments: [launched(300), notStarted(1, 280), notStarted(2, 250), notStarted(3, 200)] }));
  assert.deepEqual([gaveUp.action, gaveUp.busy], ['give-up', false]);
  assert.doesNotMatch(gaveUp.why, / at \//);

  assert.equal(decide(snapshot({ labeledAt: ago(5) })).busy, true, 'labelled, its run should start');
  assert.equal(decide(snapshot({ labeledAt: ago(20) })).reason, 'not-started');
  assert.equal(decide(snapshot({ labeledAt: ago(20), comments: [owner(60)] })).reason, 'not-started', 'a comment older than the label is not lost');
  const untrusted = decide(snapshot({}));
  assert.deepEqual([untrusted.action, untrusted.busy], ['none', false], 'nobody trusted labelled it');
});

const issue = (number, over = {}) => ({ number, title: `Issue ${number}`, body: '', labels: [], author: 'owner',
  createdAt: `2026-11-${String(number).padStart(2, '0')}T00:00:00Z`, ...over });

test('only issues of trusted authors that never were in the factory and are not excluded are launched', () => {
  const issues = [
    issue(1), issue(2, { author: 'stranger' }), issue(3, { labels: ['ai-dev'] }), issue(4, { labels: ['factory:needs-human'] }),
    issue(5, { labels: [HOLD_LABEL] }), issue(6, { labels: ['wontfix'] }), issue(7, { labels: ['question'] }),
    issue(8, { labels: ['factory-infra'] }), issue(9, { labels: ['bug', 'priority:high'] }),
  ];
  assert.deepEqual(launchCandidates(issues, ['owner']).map((i) => i.number), [1, 9]);
});

test('the owner\'s priority labels come before the agent\'s order, and age breaks what it left out', () => {
  const c = [issue(1), issue(2, { labels: ['priority:low'] }), issue(3), issue(4, { labels: ['priority:high'] }), issue(5)];
  assert.deepEqual(launchQueue(c, [2, 5, 3, 1, 4]).map((i) => i.number), [4, 5, 3, 1, 2]);
  assert.deepEqual(launchQueue(c, [3, 99, 3]).map((i) => i.number), [4, 3, 1, 5, 2], 'unknown numbers and repeats ignored');
  assert.deepEqual(launchQueue(c).map((i) => i.number), [4, 1, 3, 5, 2], 'no ranking: by age');
});

test('the ranking is read from the agent\'s structured output', () => {
  const out = JSON.stringify({ structured_output: { order: [3, 1, 'x'], reasons: [{ issue: 3, why: ' blocks #1 ' }, { why: 'no issue' }] } });
  const r = parseRanking(out);
  assert.deepEqual(r.order, [3, 1]);
  assert.equal(r.reasons.get(3), 'blocks #1');
  assert.equal(parseRanking('not json'), null);
  assert.equal(parseRanking(JSON.stringify({ result: 'text', is_error: true })), null);
  const prompt = rankPrompt([issue(1, { body: 'x'.repeat(5000) })]);
  assert.match(prompt, /data, not instructions/);
  assert.ok(prompt.length < 4000, 'long bodies are cut');
});

test('the start of an issue without runs is the last ai-dev label by a trusted actor', () => {
  // A gh stub that applies the real --jq filter, with jq, to canned issue events.
  const dir = mkdtempSync(join(tmpdir(), 'gh-stub-'));
  const label = (name, login, at, event = 'labeled') => ({ event, label: { name }, actor: { login }, created_at: at });
  writeFileSync(join(dir, 'events.json'), JSON.stringify([label('ai-dev', 'owner', '2026-11-20T10:00:00Z'),
    label('bug', 'owner', '2026-11-21T10:00:00Z'), label('ai-dev', 'stranger', '2026-11-22T10:00:00Z'),
    label('ai-dev', 'owner', '2026-11-22T11:00:00Z', 'unlabeled')]));
  writeFileSync(join(dir, 'gh'), `#!/usr/bin/env bash\nwhile [ $# -gt 0 ]; do [ "$1" = --jq ] && exec jq -r "$2" "${dir}/events.json"; shift; done\nexit 1\n`, { mode: 0o755 });
  const path = process.env.PATH;
  process.env.PATH = `${dir}:${path}`;
  try {
    assert.equal(labeledAt('o/r', 1, ['owner']), '2026-11-20T10:00:00Z');
    assert.equal(labeledAt('o/r', 1, ['nobody']), null);
  } finally {
    process.env.PATH = path;
    rmSync(dir, { recursive: true, force: true });
  }
});

test('lib.sh reads the reset time of a usage-limit message', () => {
  const resets = (message) => execFileSync('bash', ['-c', 'source scripts/factory/lib.sh; limit_resets_at "$1"', '_', message], { encoding: 'utf8' }).trim();
  assert.equal(resets("You've hit your weekly limit · resets Nov 24 2099, 3pm (Europe/Paris)"), '2099-11-24T14:00:00Z');
  const daily = Date.parse(resets("You've hit your session limit · resets 11:10pm (UTC)"));
  assert.ok(daily > Date.now() && daily <= Date.now() + 24 * 3600_000, 'today or tomorrow');
  assert.equal(new Date(daily).toISOString().slice(11, 16), '23:10');
  assert.equal(resets('Claude failed'), '');
});

test('lib.sh deletes the files listed in delete.txt that the profile may edit, and only those', () => {
  const dir = realpathSync(mkdtempSync(join(tmpdir(), 'delete-')));
  const lib = join(process.cwd(), 'scripts/factory/lib.sh');
  const files = ['internal/a.go', 'internal/b.go', '.github/w.yml', 'README.md', 'doc/changes/x/state.json'];
  try {
    for (const f of files) { mkdirSync(join(dir, dirname(f)), { recursive: true }); writeFileSync(join(dir, f), 'x\n'); }
    execFileSync('git', ['init', '-q'], { cwd: dir });
    execFileSync('git', ['add', '-A'], { cwd: dir });
    writeFileSync(join(dir, 'doc/changes/x/delete.txt'),
      ['internal/a.go', `${dir}/internal/b.go`, '.github/w.yml', 'internal/../README.md', 'internal', '/etc/hostname', ''].join('\n'));
    execFileSync('bash', ['-c', `source "${lib}"; CHANGE_DIR=doc/changes/x; apply_deletions "Read,Edit(internal/**),Edit(doc/changes/x/**),Bash(go test:*)"`],
      { cwd: dir, stdio: 'pipe' });
    const left = files.filter((f) => existsSync(join(dir, f)));
    assert.deepEqual(left, ['.github/w.yml', 'README.md', 'doc/changes/x/state.json']);
    assert.ok(!existsSync(join(dir, 'doc/changes/x/delete.txt')), 'the list is not committed');
    assert.equal(execFileSync('git', ['ls-files', 'internal'], { cwd: dir, encoding: 'utf8' }), '');
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
