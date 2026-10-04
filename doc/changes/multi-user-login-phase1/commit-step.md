# Commit step: multi-user login, phase 1

Status: ready
Owner: coordinator

## Gates

- Specifications and the API contract match the implementation (technical spec updated for REV-002).
- Every review finding has a recorded decision in [decisions.md](decisions.md); no critical finding is open (REV-001 and REV-003 are fixed and verified, REV-007 is deferred, REV-006 is rejected).
- Staging used explicit paths; `git diff --cached --check` was clean.

## Stacked pull requests

They must be merged in order. Each PR targets the previous PR's branch; the first one targets `master`.

| # | Branch | Content | Changed lines |
| --- | --- | --- | --- |
| 1 | `ai-dev/issue-15-20261004-2056` | functional open mode, technical spec, API contract | 299 |
| 2 | `ai-dev/issue-15-store` | users, sessions, owner columns, migrations | 438 |
| 3 | `ai-dev/issue-15-auth` | auth service, `admin-password` command, deploy scripts, DEPLOYMENT.md, REV-001/002/004/006 fixes | 471 |
| 4 | `ai-dev/issue-15-settings` | per-user Claude token and LeBoncoin session | 372 |
| 5 | `ai-dev/issue-15-items` | items scoped to their owner | 106 |
| 6 | `ai-dev/issue-15-api` | login, session and admin endpoints, access middleware | 365 |
| 7 | `ai-dev/issue-15-web` | login page, admin page, log out, REV-005 fix | 432 |
| 8 | `ai-dev/issue-15-docs` | README, implementation, review, decisions and commit records | see PR |

Accepted intermediate state (REV-003): after PR 6 merges and before PR 7, a user created through the raw API makes the web interface answer 401 with no login page. No data is exposed.

## Verification

At each part boundary, `go vet ./...` and `go test ./...` passed, and from PR 7 on `npm run build` passed (details in [implementation.md](implementation.md)). At the tip, `npx playwright test` passed 48 of 48 tests. Each PR was measured with `scripts/pr-size.sh <base>` before it was opened.

## Limitations

- `bash -n` on the deploy scripts and a manual run of `pricefollower admin-password` were blocked in the sandbox.
- Login latency on the ARMv6 device has not been measured.
