# Change: issue #140 follow-up, full-width tracked-items list on phones

- Issue: #140 (owner follow-up: “implementa que la lista ocupe todo el ancho en móvil, sin márgenes laterales”).
- Scope: at phone width (≤ 671 px) the tracked-items list cards (Amazon and LeBoncoin tabs) span the full viewport width with small inner card padding; no other screen, tablet, or desktop change.
- Requirements: FR-RESPONSIVE-014 to FR-RESPONSIVE-017 in [responsive-layout functional spec](../../specifications/responsive-layout/functional.md).
- Use case: [UC-TI-06](../../use-cases/tracked-items/UC-TI-06-review-tracked-items-full-width-on-phone.md) (REV-001).
- Previous change: [2026-11-11-1142-issue-140-responsive-layout](../2026-11-11-1142-issue-140-responsive-layout/index.md).

## Stage status

| Stage | Status |
| --- | --- |
| 1. Functional specification | done |
| 2. Technical specification | done (TS-RESPONSIVE-011 to 014 in [technical.md](../../specifications/responsive-layout/technical.md)) |
| 3. API specification | not affected: frontend CSS only; `API_SPECIFICATION.md` unchanged, backend unchanged |
| 4. Development | done ([implementation.md](implementation.md)); Playwright could not run in this environment |
| 5. Review | pending |
| 6. Review adjudication | pending |
| 7. QA | pending |
| 8. Commit / PR | pending |
