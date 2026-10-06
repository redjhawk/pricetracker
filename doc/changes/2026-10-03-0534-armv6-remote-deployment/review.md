# Independent review: ARMv6 remote deployment

Status: one specification-consistency finding corrected and independently rechecked; QA pending.
Reviewer: separate reviewer invocation, runtime agent `workflow_adjudicator` (historical name; this invocation does not adjudicate its findings).
Reviewed snapshot: working tree over HEAD `14e45629291b0a7b52fd233191e284a208a54e3e`.

## Inputs and scope

Read reviewer instructions, all four mandatory project skills, [functional requirements](../../specifications/armv6-remote-deployment/functional.md), [technical requirements](../../specifications/armv6-remote-deployment/technical.md), [API preservation](api-step.md), [implementation evidence](implementation.md), wrapper, deployment documentation, and all three existing called scripts.

Only `scripts/deploy-armv6.sh`, the deployment guide and subject/workflow Markdown artifacts are in this change. Existing build, copy and installer scripts are unchanged. No application, backend, SQLite, HTTP API or dependency changes are present.

## REV-001: Staging-path acceptance wording conflicts with implemented syntax

- Requirement: FR-ARMV6-007; TS-ARMV6-002.
- Evidence: FR-ARMV6-007 says accepted paths remain literal “even when containing spaces or shell metacharacters.” Technical specifications expressly reject spaces and shell metacharacters. The wrapper path regex permits only ASCII letters, digits, underscores, dots, hyphens and slashes, and DEPLOYMENT.md documents rejection.
- Expected versus actual: the functional acceptance example implies support for these characters; the technical design and implementation intentionally fail such inputs before any deployment. The functional corner-case text also allows unsupported forms to be rejected and calls for precise documented syntax, making the acceptance wording internally ambiguous.
- Reproduction: a staging argument `my release` fails the wrapper regex with exit 2; no build or SSH stage occurs.
- Impact: specification-driven acceptance could reject the otherwise coherent implementation or lead operators to expect a supported directory form that is rejected. This is not an injection vulnerability: rejection is safe and clearly documented.
- Provisional severity: noncritical specification consistency issue. No device operation, data corruption or unapproved remote action occurred.
- Suggested resolution: reconcile FR-ARMV6-007 with the explicitly documented allowlist if restriction reflects the intended requirement. If arbitrary paths with spaces/metacharacters were intended, return that product choice to the user and update technical design and implementation before claiming compliance. Do not silently broaden accepted inputs or rewrite user intent.
- Independent recheck: functional author corrected FR-ARMV6-007 to accept plain paths literally and explicitly reject unsupported spaces/metacharacters before commands run, as defined by the technical allowlist. Read the revised requirement and functional handoff record; they now agree with TS-ARMV6-002, the regex and DEPLOYMENT.md. Correction verified; no implementation or API expansion occurred. Original evidence above is retained for traceability.

## Other requirement checks

- FR-001/002: validates arity and target/path before dependency checks; resolves project path from the script; invokes the existing build with `6`.
- FR-003/004: creates staging before invoking the existing copy script with a trailing slash, then uses `ssh -t` and `sudo bash` for the installer. Default staging contains its child binary and does not collide with it.
- FR-005: local strict Bash failure handling and remote `&&` sequence stop subsequent stages. The final active-service command must succeed before the success message.
- FR-006: upload inputs remain the binary and installer only; existing installer retains its data directory and no wrapper database removal exists.
- FR-007: destination allowlist excludes injected options and shell syntax; single-quoted remote paths contain no allowed quote characters. Parent segments and normalized root/dot/protected installed/data paths are rejected. The lexical guard handles repeated slash and dot separators; remote symlink aliases are explicitly documented operator limitations, not promised canonicalization.

No additional correctness, scope or readability findings identified. Straightforward sequential code reuses existing scripts without refactoring. Interactive SSH/sudo behavior and actual device installation remain unexecuted; `ssh -t` is inspected intent, not evidence of a real authentication prompt.

## Independent verification

Executed Bash syntax checks for wrapper and three called scripts: passed. Executed `git diff --check`: passed. Independently inspected `go version -m release/pricefollower`: confirms Linux ARM, `GOARM=6`, `CGO_ENABLED=0`, Go 1.25.0. The developer's successful real cross-build is recorded in implementation.md; this reviewer did not rerun it.

Command-line fixture QA must still verify ordering, default/explicit paths, invalid inputs, missing dependencies, repeated runs, failure propagation and inactive-service failure. No real SSH target was supplied; no device deployment or browser QA is claimed. Hand REV-001 to a distinct adjudicator before corrections and re-review.
