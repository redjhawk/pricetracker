# Independent review decisions: ARMv6 deployment

Adjudicator: `/root`, distinct from developer `marketplace_developer` and reviewer `workflow_adjudicator` acting as reviewer only.
Inputs: [review](review.md), [functional](../../specifications/armv6-remote-deployment/functional.md), [technical](../../specifications/armv6-remote-deployment/technical.md), [implementation](implementation.md) and script.

## REV-001 — Accepted path wording conflicts with supported syntax

- Evidence: the functional acceptance text previously said accepted paths remain literal even when containing spaces/metacharacters, while the technical specification, wrapper and deployment guide explicitly reject those unsupported characters.
- Impact: an operator or tester could expect unsupported arbitrary paths to work; the documents did not express the same acceptance boundary.
- Criticality: noncritical. This is a specification consistency problem, not an injection flaw: actual validation rejects those inputs before building or contacting a host. Default and ordinary relative/absolute paths meet the requested operation. The user did not request arbitrary shell/path syntax.
- Disposition: fix the functional wording; do not change the implementation to accept new inputs.
- Reason: the original functional requirement already permits unsupported forms to be rejected, and its corner-case section delegates precise syntax to the technical design. Clarifying that accepted plain paths stay literal and unsupported spaces/metacharacters fail restores consistency with the documented scope without inventing functionality.
- Resolution: functional author updated FR-ARMV6-007 and its stage record. Independent reviewer rechecked and verified the correction in review.md; fixed with no remaining review blocker.
- Owner: functional specification agent and independent reviewer.
- Follow-up: none after independent recheck. No intentionally deferred or rejected review comment.

## Other implementation choices

Lexical path normalization guards root and the existing install/data directories while preserving the actual staging pathname. Symlink resolution and rollback are explicitly outside scope; operator-selected ordinary staging directories are required. The wrapper reuses all three existing scripts; no refactoring, API change, database transfer or real-device operation is introduced.

Required next stage: isolated command-line QA. Real ARMv6 cross-compilation passed, but real SSH transfer, sudo prompting and service installation cannot be claimed without a supplied device. This limitation is consistent with the request to create a script, not execute a deployment on an unspecified host.

## QA adjudication

The independent CLI tester reports 43 passed isolated scenarios, including actual wrapper/build/copy script execution with controlled tools, failure propagation, malformed inputs and conditional remote installer/service-check execution in a fixture home. No new findings require severity or disposition. Real-device checks remain explicitly unexecuted; the requested script has been validated without inventing a host or installing a service elsewhere. See [qa.md](qa.md) for actual commands, results and limits.
