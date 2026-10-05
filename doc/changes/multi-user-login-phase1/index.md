# Multiple local users and login: phase 1

Stage: functional, technical and API specifications complete; implementation next.\
Change slug: `multi-user-login-phase1`. Started 2026-10-04. GitHub issue #15.

## Scope

Phase 1 of [multi-user-login](../../specifications/multi-user-login/functional.md), as defined in the [delivery plan](../../specifications/multi-user-login/delivery-plan.md): device script for the `admin` account, open and protected modes, login and session, administrator page (user list and user creation), item ownership, per-user settings, and inheritance of existing data by the first user. The deploy scripts and the deploy documentation are updated to cover the device script (user instruction).

## User request (2026-10-04)

> implement everything. Remember to update the deploy scripts and deploy md information file.

## User decisions

- D-1 to D-18: recorded in [functional.md](../../specifications/multi-user-login/functional.md), user request 2026-10-04.
- D-19 to D-26: phase 1 decisions in [delivery-plan.md](../../specifications/multi-user-login/delivery-plan.md), user request 2026-10-04. Where they differ from D-1 to D-18 or a requirement, D-19 to D-26 apply.
- Precedence recorded: FR-ADMIN-001 and D-1 say the operator types the administrator password on the device; D-22 overrides this in phase 1: the device script generates a random password and prints it. The script creates `admin` and resets the `admin` password only.
- Deploy scripts and deploy documentation must be updated (user instruction, 2026-10-04).

## Subject specifications

| Subject | Functional file | Phase 1 requirement IDs | Status |
| --- | --- | --- | --- |
| Multiple local users and login | [functional](../../specifications/multi-user-login/functional.md), [delivery plan](../../specifications/multi-user-login/delivery-plan.md) | see [functional-step.md](functional-step.md) | ready, user-approved |

## Stage records

1. [Functional handoff](functional-step.md): ready.
2. [Technical specification](technical-step.md): ready, [technical.md](../../specifications/multi-user-login/technical.md).
3. [API contract](api-step.md): ready.
4. Implementation, 5. review, 6. review decisions, 8. commit, 7. QA: pending.

## API contract changes

New auth endpoints (`/auth/session`, `/auth/login`, `/auth/logout`) and admin endpoints (`/admin/users`), new 401/403/429 codes, and per-user scoping of existing endpoints in protected mode (another user's item is 404). Rationale: the SPA needs the mode and user to choose its screen, and scoping implements D-19 and FR-SHARE-005 without changing existing shapes; open mode keeps the old contract (D-25). Details in [api-step.md](api-step.md).

## Unresolved questions

Functional: none blocking. Precedence notes are recorded in [functional-step.md](functional-step.md).

## Handoffs

- Functional specifier (separate agent, 2026-10-04): verified phase 1 is complete and unambiguous from existing specifications and user decisions; no specification files changed; see [functional-step.md](functional-step.md).
