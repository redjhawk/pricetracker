#!/usr/bin/env node
// Supervisor of the agent factory (doc/workflow/WORKFLOW.md, "Supervisor"). It runs every 15 minutes
// (.github/workflows/supervisor.yml), reads the open ai-dev issues and resumes the factory runs that
// stopped for a reason that goes away without the owner:
//
//   usage-limit                               after the reset (resets_at in the stop marker)
//   error, crash, blocked-env, merge-failed   after 15, 30, 45 minutes; at most 3 times per stage and round
//   base-bug                                  when the issue it waits for is closed
//   an owner comment that started no run      15 minutes after the comment
//
// Every other stop (functional questions, a failed gate, sensitive files...) waits for the owner.
// Before it relaunches anything, a probe agent checks that the runner can run agents with Bash. When it
// cannot, one factory-infra issue says so and nothing is relaunched; the issue is closed once the probe
// passes. The supervisor never edits code: it comments, and opens or closes the infra issue.
//
// A relaunch is a comment by the owner's token without the <!-- agent-factory --> marker, so fabrica.yml
// starts on it as on an owner comment; the factory does not pass it to the agents (lib.sh).
//
//   node scripts/factory/supervisor.mjs [--dry-run]

import { execFileSync, spawnSync } from 'node:child_process';
import { randomBytes } from 'node:crypto';
import { appendFileSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

export const FACTORY_MARKER = '<!-- agent-factory -->';
export const HOLD_LABEL = 'factory-hold';
export const INFRA_LABEL = 'factory-infra';
const STOP_RE = /<!-- factory-stop (\{.*?\}) -->/;
const SUPERVISOR_RE = /<!-- factory-supervisor (\{.*?\}) -->/;
const RETRIED = new Set(['error', 'crash', 'blocked-env', 'merge-failed']);
const MINUTE = 60_000;

export const DEFAULTS = {
  maxRetries: 3, // counted relaunches at the same stage and round before giving up
  backoffMinutes: 15, // the n-th retry waits n times this long after the stop
  limitMarginMinutes: 5, // after the usage limit's reset time
  graceMinutes: 15, // for a run to start after a comment or a relaunch
};

const iso = (ms) => new Date(ms).toISOString().replace(/\.\d{3}Z$/, 'Z');

function marker(body, re) {
  const m = re.exec(body);
  if (!m) return null;
  try { return JSON.parse(m[1]); } catch { return null; }
}

/** Kind of an issue comment. Markers count only in comments of trusted authors (the factory uses the owner's token). */
export function classify(comment, trusted) {
  const at = Date.parse(comment.createdAt);
  const body = comment.body ?? '';
  if (!trusted.includes(comment.author)) return { kind: 'other', at };
  const sup = marker(body, SUPERVISOR_RE);
  if (sup) return { kind: 'supervisor', at, marker: sup };
  const stop = marker(body, STOP_RE);
  if (stop) return { kind: 'stop', at, marker: stop };
  if (body.includes(FACTORY_MARKER)) return { kind: 'factory', at };
  return { kind: 'owner', at };
}

/**
 * What to do with one issue. Pure: everything it needs is in the snapshot.
 * snapshot: { now, labels[], comments[{author, body, createdAt}], runs[{status, conclusion, createdAt, updatedAt}]
 *   (fabrica runs of this issue whose job ran), runsSince (oldest run fetched, ISO, or null when all were),
 *   trusted[], blockers {issue number: 'OPEN'|'CLOSED'}, config }
 * Returns { action: 'none'|'relaunch'|'give-up', why, reason?, progress?, attempt?, counted?, limitUntil? }.
 */
export function decide(s) {
  const cfg = { ...DEFAULTS, ...s.config };
  const now = Date.parse(s.now);
  const grace = cfg.graceMinutes * MINUTE;
  const none = (why, extra = {}) => ({ action: 'none', why, ...extra });

  if (s.labels.includes(HOLD_LABEL)) return none(`label ${HOLD_LABEL}`);
  const runs = s.runs.map((r) => ({ ...r, at: Date.parse(r.createdAt) })).sort((a, b) => a.at - b.at);
  if (runs.some((r) => r.status !== 'completed')) return none('a run is queued or in progress');

  const events = s.comments.map((c) => classify(c, s.trusted)).sort((a, b) => a.at - b.at);
  const latest = (kind) => events.filter((e) => e.kind === kind).at(-1);
  const lastRun = runs.at(-1);
  const lastRunAt = lastRun?.at ?? -Infinity;
  const owner = latest('owner');
  const sups = events.filter((e) => e.kind === 'supervisor');
  const lastSup = sups.at(-1);

  if (lastSup?.marker.action === 'relaunch' && lastSup.at > lastRunAt && now - lastSup.at < grace) {
    return none('relaunched, waiting for the run to start');
  }

  // An owner comment restarts the factory through fabrica.yml. One that started no run (its run was cancelled
  // while queued, or never created) is relaunched here, once.
  const runsSince = s.runsSince ? Date.parse(s.runsSince) : -Infinity;
  if (owner && owner.at > lastRunAt && owner.at >= runsSince && !(lastSup && lastSup.at > owner.at)) {
    if (now - owner.at < grace) return none('owner comment, its run should start');
    return relaunch('owner-comment', '', false, 'the owner comment started no run');
  }

  if (lastSup?.marker.action === 'give-up' && !(owner && owner.at > lastSup.at)) return none('gave up, waiting for the owner');

  // The stop to act on: the factory's last stop marker, or a crash when the last run failed without reporting one.
  const stop = latest('stop');
  let reason, since, m;
  if (lastRun && (!stop || stop.at < lastRun.at)) {
    if (!['failure', 'timed_out'].includes(lastRun.conclusion)) return none(`last run ${lastRun.conclusion}, no stop`);
    reason = 'crash';
    since = Date.parse(lastRun.updatedAt ?? lastRun.createdAt);
    m = { stage: stageLabel(s.labels) };
  } else if (stop) {
    ({ reason } = stop.marker);
    since = stop.at;
    m = stop.marker;
  } else {
    return none('no stop recorded');
  }
  if (owner && owner.at > since) return none('the owner answered after the stop');

  const progress = `${m.stage ?? ''}/${m.round ?? ''}`;
  const tries = sups.filter((e) => e.marker.action === 'relaunch' && e.marker.counted &&
    e.marker.progress === progress && !(owner && owner.at > e.at)).length;

  if (reason === 'usage-limit') {
    const until = m.resets_at ? Date.parse(m.resets_at) + cfg.limitMarginMinutes * MINUTE : NaN;
    if (now < until) return none(`usage limit until ${iso(until)}`, { limitUntil: until });
    return relaunch(reason, progress, false, 'the usage limit has reset');
  }
  if (reason === 'base-bug') {
    if (s.blockers?.[m.waiting_on] !== 'CLOSED') return none(`waiting for #${m.waiting_on}`);
    return counted(since, `#${m.waiting_on} is closed`);
  }
  if (RETRIED.has(reason)) return counted(since + cfg.backoffMinutes * MINUTE * (tries + 1), `stopped with ${reason}`);
  return none(`needs the owner (${reason})`);

  function counted(notBefore, why) {
    if (tries >= cfg.maxRetries) {
      return { action: 'give-up', reason, progress, why: `${tries} relaunches at ${progress} did not help (${reason})` };
    }
    if (now < notBefore) return none(`retry ${tries + 1} of ${cfg.maxRetries} after ${iso(notBefore)}`);
    return { ...relaunch(reason, progress, true, why), attempt: tries + 1 };
  }
  function relaunch(why0, prog, isCounted, why) {
    return { action: 'relaunch', reason: why0, progress: prog, counted: isCounted, why };
  }
}

function stageLabel(labels) {
  return labels.map((l) => /^factory:(.+)$/.exec(l)?.[1]).find(Boolean) ?? '';
}

/** The usage limit is per account: while one issue waits for its reset, no other issue is relaunched. */
export function applyGlobalLimit(decisions, now) {
  const until = Math.max(-Infinity, ...decisions.map((d) => d.limitUntil ?? -Infinity));
  if (!(Date.parse(now) < until)) return decisions;
  return decisions.map((d) => (d.action === 'relaunch' ? { ...d, action: 'none', why: `usage limit until ${iso(until)}` } : d));
}

/** Result of the probe agent: ok when it printed the nonce it could only read with Bash. */
export function classifyProbe({ stdout = '', stderr = '', error } = {}, nonce) {
  const out = `${stdout}\n${stderr}`;
  if (out.includes(nonce)) return { ok: true };
  const detail = (out.trim() || String(error ?? 'no output')).split('\n').slice(-30).join('\n');
  return { ok: false, limit: /hit your (session|usage|weekly) limit/i.test(out), detail };
}

// ---- side effects ----

function gh(args, input) {
  return execFileSync('gh', args, { encoding: 'utf8', input, stdio: ['pipe', 'pipe', 'inherit'], maxBuffer: 64 * 1024 * 1024 });
}
const ghJson = (args) => JSON.parse(gh(args));

function comments(repo, issue) {
  const all = [];
  for (let page = 1; ; page++) {
    const batch = ghJson(['api', `repos/${repo}/issues/${issue}/comments?per_page=100&page=${page}`]);
    all.push(...batch.map((c) => ({ author: c.user.login, body: c.body, createdAt: c.created_at })));
    if (batch.length < 100) return all;
  }
}

const RUN_LIMIT = 300;
function fabricaRuns(repo) {
  const runs = ghJson(['run', 'list', '-R', repo, '--workflow', 'fabrica.yml', '--limit', String(RUN_LIMIT),
    '--json', 'databaseId,displayTitle,status,conclusion,createdAt,updatedAt']);
  const since = runs.length < RUN_LIMIT ? null : runs.map((r) => r.createdAt).sort()[0];
  const byIssue = new Map();
  for (const r of runs) {
    const n = Number(/^fabrica #(\d+)$/.exec(r.displayTitle)?.[1]);
    if (!n || r.conclusion === 'skipped') continue;
    if (!byIssue.has(n)) byIssue.set(n, []);
    byIssue.get(n).push(r);
  }
  return { byIssue, since };
}

// The runs decide() needs: those not finished and the latest one that ran its job. A run cancelled while
// queued (its concurrency group moved on) never ran it.
function relevantRuns(repo, runs) {
  const out = [];
  for (const r of [...runs].sort((a, b) => b.createdAt.localeCompare(a.createdAt))) {
    if (r.status !== 'completed') { out.push(r); continue; }
    if (r.conclusion !== 'cancelled' || ghJson(['run', 'view', String(r.databaseId), '-R', repo, '--json', 'jobs']).jobs.length > 0) {
      out.push(r);
      break;
    }
  }
  return out;
}

export function probe(cwd, model) {
  const nonce = randomBytes(8).toString('hex');
  const file = '.factory-probe';
  writeFileSync(join(cwd, file), `${nonce}\n`);
  // The same environment as run_agent in lib.sh: no GitHub token, credentials scrubbed from subprocesses.
  const env = { ...process.env, CLAUDE_CODE_SUBPROCESS_ENV_SCRUB: '1' };
  delete env.GH_TOKEN;
  delete env.GITHUB_TOKEN;
  try {
    const r = spawnSync('claude', ['-p', `Run \`cat ${file}\` with the Bash tool and reply with its output only.`,
      '--model', model, '--max-turns', '3', '--allowedTools', `Bash(cat ${file})`, '--output-format', 'text'],
    { cwd, env, encoding: 'utf8', timeout: 5 * MINUTE });
    return classifyProbe(r, nonce);
  } finally {
    rmSync(join(cwd, file), { force: true });
  }
}

const runLink = () => (process.env.GITHUB_RUN_ID
  ? `\nSupervisor run: ${process.env.GITHUB_SERVER_URL}/${process.env.GITHUB_REPOSITORY}/actions/runs/${process.env.GITHUB_RUN_ID}` : '');

function relaunchText(d) {
  switch (d.reason) {
    case 'owner-comment': return 'Your last comment started no factory run: the supervisor starts one now.';
    case 'usage-limit': return 'The Claude usage limit has reset: the supervisor resumes the factory.';
    case 'base-bug': return `${d.why}: the supervisor resumes the factory, which merges the base branch before implementing.`;
    default: {
      const [stage, round] = d.progress.split('/');
      return `The factory ${d.why}: the supervisor resumes it (retry ${d.attempt} of ${DEFAULTS.maxRetries} at stage \`${stage}\`${round ? `, round ${round}` : ''}).`;
    }
  }
}

function relaunchIssue(repo, issue, d) {
  const meta = { action: 'relaunch', reason: d.reason, progress: d.progress, counted: d.counted, attempt: d.attempt ?? 0 };
  // No <!-- agent-factory --> marker: this comment is what starts fabrica.yml.
  gh(['issue', 'comment', String(issue), '-R', repo, '--body-file', '-'],
    `<!-- factory-supervisor ${JSON.stringify(meta)} -->\n${relaunchText(d)}${runLink()}\n`);
}

function giveUp(repo, issue, d) {
  const meta = { action: 'give-up', reason: d.reason, progress: d.progress };
  gh(['issue', 'comment', String(issue), '-R', repo, '--body-file', '-'],
    `${FACTORY_MARKER}\n<!-- factory-supervisor ${JSON.stringify(meta)} -->\n` +
    `The supervisor stops retrying: ${d.why}. Look at the last stop above and answer in a comment to resume the factory.${runLink()}\n`);
  // One factory:* label at a time, as set_stage_label in lib.sh does.
  gh(['label', 'create', 'factory:needs-human', '--color', 'BFD4F2', '--force', '-R', repo]);
  const old = d.labels.filter((l) => l.startsWith('factory:') && l !== 'factory:needs-human');
  gh(['issue', 'edit', String(issue), '-R', repo, '--add-label', 'factory:needs-human', ...old.flatMap((l) => ['--remove-label', l])]);
}

function openInfraIssue(repo) {
  return ghJson(['issue', 'list', '-R', repo, '--label', INFRA_LABEL, '--state', 'open', '--json', 'number'])[0]?.number;
}

function reportBrokenRunner(repo, result) {
  if (openInfraIssue(repo)) return;
  gh(['label', 'create', INFRA_LABEL, '--color', 'B60205', '--description', 'The agent factory runner cannot run agents', '--force', '-R', repo]);
  gh(['issue', 'create', '-R', repo, '--label', INFRA_LABEL, '--title', 'Agent factory: the runner cannot run agents', '--body-file', '-'],
    'The supervisor\'s probe, a Claude agent that runs `cat` with the Bash tool like the factory stages do, failed on the `arc-dev` runner. ' +
    'The supervisor relaunches no factory run until the probe passes, and then closes this issue.\n\nProbe output:\n~~~\n' +
    `${result.detail}\n~~~\n${runLink()}\n`);
}

function closeInfraIssue(repo) {
  const n = openInfraIssue(repo);
  if (n) gh(['issue', 'close', String(n), '-R', repo, '--comment', `The supervisor's probe passes again: factory runs are relaunched.${runLink()}`]);
}

function main() {
  const dryRun = process.argv.includes('--dry-run');
  const repo = process.env.GITHUB_REPOSITORY || ghJson(['repo', 'view', '--json', 'nameWithOwner']).nameWithOwner;
  const trusted = (process.env.FACTORY_TRUSTED_ACTORS || repo.split('/')[0]).split(',');
  const now = new Date().toISOString();
  const { byIssue, since } = fabricaRuns(repo);
  const issues = ghJson(['issue', 'list', '-R', repo, '--label', 'ai-dev', '--state', 'open', '--limit', '100', '--json', 'number,labels']);

  let decisions = issues.map(({ number, labels }) => {
    const all = comments(repo, number);
    const lastStop = all.map((c) => classify(c, trusted)).filter((e) => e.kind === 'stop').sort((a, b) => a.at - b.at).at(-1);
    const blockers = {};
    const waitingOn = lastStop?.marker.waiting_on;
    if (waitingOn) blockers[waitingOn] = ghJson(['issue', 'view', String(waitingOn), '-R', repo, '--json', 'state']).state;
    const runs = relevantRuns(repo, byIssue.get(number) ?? []);
    const names = labels.map((l) => l.name);
    return { issue: number, labels: names, ...decide({ now, labels: names, comments: all, runs, runsSince: since, trusted, blockers }) };
  });
  decisions = applyGlobalLimit(decisions, now);

  const relaunches = decisions.filter((d) => d.action === 'relaunch');
  let probeLine = 'probe: not needed';
  if (!dryRun && (relaunches.length > 0 || openInfraIssue(repo))) {
    const result = probe(process.cwd(), process.env.FACTORY_MODEL || 'claude-opus-5-5');
    probeLine = `probe: ${result.ok ? 'ok' : result.limit ? 'usage limit' : 'runner broken'}`;
    if (result.ok) closeInfraIssue(repo);
    else if (!result.limit) reportBrokenRunner(repo, result);
    if (!result.ok) {
      decisions = decisions.map((d) => (d.action === 'relaunch' ? { ...d, action: 'none', why: `${probeLine}; ${d.why}` } : d));
    }
  }

  for (const d of decisions) {
    if (dryRun) continue;
    if (d.action === 'relaunch') relaunchIssue(repo, d.issue, d);
    if (d.action === 'give-up') giveUp(repo, d.issue, d);
  }

  const lines = [`supervisor ${now}${dryRun ? ' (dry run)' : ''}, ${probeLine}`,
    ...decisions.map((d) => `#${d.issue}\t${d.action}\t${d.reason ?? ''}\t${d.why}`)];
  console.log(lines.join('\n'));
  if (process.env.GITHUB_STEP_SUMMARY) appendFileSync(process.env.GITHUB_STEP_SUMMARY, `\`\`\`\n${lines.join('\n')}\n\`\`\`\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main();
