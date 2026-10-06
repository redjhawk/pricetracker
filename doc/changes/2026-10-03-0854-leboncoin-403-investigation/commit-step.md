# Investigation documentation commit

Status: prepared after independent review, adjudication and verification passed.
Coordinator: `/root`.

Scope: only `doc/changes/2026-10-03-0854-leboncoin-403-investigation/`. No application, dependency, API, credential, raw browser profile or probe binary belongs in the commit. The temporary module-local probe was removed after verification and comparison with its preserved source; its source and binary remain under the private `/tmp/pricefollower-leboncoin-diagnostic/` directory for the experiment handoff.

Intended message: `docs(leboncoin): record verified session access experiments`.

This is documentation of the requested investigation, not a collector feature. New product specifications and interface changes are not applicable. Proposed cookie integration remains future work under the normal specification-led workflow.

## Passed gates

- Investigation scope and API preservation are explicit in the report; no new application behavior or approval is asserted.
- Reviewer found one factual wording error; adjudicator required correction. Reviewer verified the corrected report and adjudicator marked REV-001 resolved. No findings remain open.
- Independent QA repeated the live pair at 08:51 UTC with explicit assertions: baseline HTTP/2 403/request_error, validated cookie HTTP/2 200/success/2690000 cents and expected title. No extra CAPTCHA interaction occurred in this repeat.
- QA verified the final reviewed report hash, ARMv6 build metadata, twenty local Markdown links and unchanged tracked production/API/dependency files.
- Coordinator removed the owned temporary module-local probe, explicitly staged only the six investigation Markdown records and inspected their scope and contents. No unrelated changes are staged or left in the working tree.
- `git diff --cached --check` passed. All twenty local Markdown links resolve; the report fingerprint matches the final reviewer/QA revision. A scan found no numeric network addresses in the staged documentation; no cookie values were added.
- This final preparation-record edit is staged and checked again immediately before committing.

This pre-commit record cannot contain its own resulting hash. Git's successful output and the coordinator's final handoff establish the actual outcome. No push is requested.
