# Workflow documentation QA

Status: executed for documentation; application interface QA not applicable.
Tester: `documentation_qa`, distinct from implementation, reviewer, and adjudicator.
Executed on: 2026-10-03 (Europe/Paris, user-provided session date).
Revision: working-tree documentation over Git HEAD `299996a431dc14fec2d8f39b3956992e8f1a3426`; later documentation edits require corresponding rechecks.
Environment: repository filesystem and Python 3; no application server or browser session used.

## Executed checks

| Case | Actual command or procedure | Expected | Observed / result |
| --- | --- | --- | --- |
| DOC-QA-001 | Python 3 heredoc using `pathlib`, `re.findall` for Markdown links, and relative-target `Path.exists()` across AGENTS.md, doc/README.md, doc/workflow, .agents/roles, and doc/changes/agent-workflow | All concrete local links resolve | Passed: 17 Markdown files, 45 concrete local links, zero missing targets. External links, anchors, and placeholder paths were excluded; link anchors and external availability were not tested. |
| DOC-QA-002 | Python assertions that all six named role Markdown files exist; read each role | Separate functional specifier, technical specifier, developer, reviewer, adjudicator, and QA instructions | Passed: six roles exist and identify distinct spawned agent invocations rather than native registration. Reviewer/adjudicator independence and sequential handoffs are explicit. |
| DOC-QA-003 | Python regex over numbered workflow headings plus manual comparison with the request | Functional → technical → API → implementation → review → decisions → QA | Passed: seven headings occur in this order; one functional and one technical file per subject, technical coverage of frontend/backend/API, and explicit artifact paths are documented. |
| DOC-QA-004 | Python phrase assertions and manual reading of workflow, technical role, developer role, and AGENTS.md | Contract changes require user confirmation before implementation; cross-tier work waits for confirmation of both tiers | Passed: confirmation gates preserved, unchanged approved APIs may be reused, unresolved product choices and refactoring decisions return to the user. |
| DOC-QA-005 | Python assertions of required sections in all four templates; manual inspection of tables and placeholders | Templates support requirements, technical tiers, reasoned decisions, and reproducible QA | Passed: functional requirements and acceptance table; technical tier/scope/refactoring sections; review IDs, criticality, fix/defer/reject and pending resolution separation; actual interface/listing URLs, ordered actions and outcomes in QA template. |
| DOC-QA-006 | Read review.md, decisions.md, and revised role/template text | Reported fixes exist and no invented application QA evidence | Passed for documentation consistency: five review corrections appear in final text and reviewer rechecks are recorded. The finalized adjudication record marks all five findings fixed with independent recheck links; no unresolved review blocker remains. |

Commands executed also included `git rev-parse HEAD`, `git status --short`, `cat`, and `find`. `rg` was unavailable; file discovery used `find` instead. All four required project skills and the QA role were read before these checks.

## Coverage and limitations

This change defines a development workflow and does not implement application behavior. Random interface actions, corner-case listing submissions, application builds, and interface URL execution are not applicable. No application URLs, screenshots, browser passes, or listing-input outcomes were invented. The [future feature QA template](../../workflow/templates/qa.md) requires real interface URLs and listing URLs with exact actions, results, and evidence when application functionality is implemented.

Checks cover documentation existence, concrete link targets, stage order, role separation, scope/approval rules, and template usability. They do not execute future agents or prove future application behavior. Placeholder example paths deliberately require feature-specific values. No test data was created and no cleanup was necessary.

## Handoff

No new QA findings. Independent review is in [review.md](review.md); criticality and final dispositions are owned by the adjudicator in [decisions.md](decisions.md). The finalized decision record reflects all independently verified corrections. No documentation QA blocker remains.
