# Technical specification handoff

Status: ready. Role: distinct technical specification agent. Date: 2026-10-03.

Read the repository instructions, staged workflow, technical role, all four required project skills, ready functional specification, current App/list/CSS implementation, approved API, existing search use case, and installed Carbon tab types/source. Consulted the [official Carbon tabs guidance](https://www.carbondesignsystem.com/building-blocks/core/components/tabs/guidelines).

[Technical specification](../../specifications/platform-tabs/technical.md) maps FR-PLATFORM-TABS-001–008 to TS-PLATFORM-TABS-001–006 and defines App-owned tab state, existing shared search state, Carbon panels, platform columns, feedback precedence, and preserved global operations. No backend/API changes or refactoring are required.

Developer scope includes a focused durable Playwright harness to record RED before UI changes, GREEN after implementation, and the existing build. One exact-pinned browser-test dependency is justified by the requested test-first/browser verification; no broader testing framework is introduced. Minimal use-case/README updates keep user guidance accurate. Independent review, adjudication and QA remain later stages; no implementation or test execution is claimed here.

No unresolved questions. Proceed through [API preservation](api-step.md) to the separate developer role.
