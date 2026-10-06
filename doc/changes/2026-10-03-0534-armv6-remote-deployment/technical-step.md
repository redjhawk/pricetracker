# Technical specification stage

Status: ready. Role: distinct technical specification agent. Date: 2026-10-03.

Read the technical role, staged workflow, all four required project skills, ready functional subject, existing build/transfer/installer scripts, deployment guide and approved API contract. Prepared [technical specification](../../specifications/armv6-remote-deployment/technical.md) with requirement mappings, exact command interface, conservative input syntax, reuse of existing scripts, stage failure gates and isolated CLI QA.

Handoff scope: new `scripts/deploy-armv6.sh`, focused `DEPLOYMENT.md` instructions and workflow evidence. No frontend/backend implementation, data schema, HTTP API change or refactoring. No unresolved technical blocker. A real remote device is needed only to execute a real deployment; tests must record that limitation.

Pre-review clarification: protected-path and root/dot checks use a separate lexical check value with repeated slash and dot-separator normalization. This closes syntactic aliases such as `/opt//pricefollower`, `/opt/./pricefollower` and `/./` without resolving remote symlinks or changing the literal path used for deployment. This is necessary input validation within the existing scope, not new functionality or refactoring.
