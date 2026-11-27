# Issue #201 Merge card td display rule into the tbody td block

**Problem:** `src/index.css:664-666` keeps a standalone `.responsive-table td { display: block; }` inside the 1055.98px query; every matched td is also matched by `.responsive-table tbody tr td` at line 678.

**Suggested work:** Move `display: block` into the `.responsive-table tbody tr td` block and delete the standalone rule, so the card-cell styling lives in one place. Re-run `tests/responsive-layout.spec.ts` to confirm no layout change.

From review comment R1-03 on #198 (`doc/todo/issue-165-r1-03.md`).

## Comments by trusted authors

