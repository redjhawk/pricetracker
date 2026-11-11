# Commit step

- Branch: `ai-dev/issue-140-20261111-1512` → `master`, single PR (small CSS change, no split).
- Commits:
  1. `docs(responsive): specify the full-width tracked-items list on phones` (FR-RESPONSIVE-014..017, TS-RESPONSIVE-011..014, UC-TI-06).
  2. `feat(responsive): run the tracked-items list edge to edge on phones` (`src/index.css`, `tests/responsive-layout.spec.ts`).
  3. `docs(changes): record the full-width list change` (this folder).
- Checks: `npm run build` passed. Playwright could not run: Chromium fails to start (`libnspr4.so` missing, no sudo). Browser QA blocked (see qa.md).
- REV-001 (critical) is fixed in the code and was rechecked by the reviewer by reading it; it is confirmed only when `tests/responsive-layout.spec.ts` passes on Chromium. Do not merge before that.
- PR: see the issue #140 report.
