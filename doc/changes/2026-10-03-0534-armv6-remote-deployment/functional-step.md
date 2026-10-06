# Functional specification stage: ARMv6 remote deployment

Status: ready
Role: distinct functional specification expert

Read the role instructions, staged workflow and all four required project skills. Inspected the existing release, transfer and installer scripts and `README.md`/`DEPLOYMENT.md`.

One subject was identified: an operator command wrapping build, transfer and remote installation. Wrote [the functional specification](../../specifications/armv6-remote-deployment/functional.md), requirements FR-ARMV6-001 through FR-ARMV6-007, with success, failure, data preservation and literal-input acceptance criteria.

The user explicitly requested ARMv6, existing-script reuse and the default `pricefollower` staging directory. SSH target input is necessary to address the device and is supplied at execution time. Existing installer behavior supplies service settings and durable data placement. The requested work does not change frontend, business backend or the approved HTTP contract. No additional approval is claimed and no refactoring is proposed.

Handoff: technical specification may proceed; no unresolved product question blocks the script. Real-device QA requires an actual reachable SSH target and privileges and must be recorded as unexecuted if unavailable.

Review correction REV-001: clarified FR-ARMV6-007 to match the technical specification's conservative pathname allowlist. Accepted plain paths remain literal; spaces, shell metacharacters and other unsupported syntax are rejected before commands run. The earlier phrase describing accepted paths containing spaces or metacharacters conflicted with the documented rejection policy. This correction removes that ambiguity without adding arbitrary-path support or changing the requested default/simple-path behavior. No implementation change was required.
