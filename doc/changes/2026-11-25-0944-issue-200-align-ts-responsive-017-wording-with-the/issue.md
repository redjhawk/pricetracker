# Issue #200 Align TS-RESPONSIVE-017 wording with the growing item cell

**Problem:** `doc/specifications/responsive-layout/technical.md:117` (TS-RESPONSIVE-017) states "Cells do not grow, so the free space left on a line is always smaller than the next cell's width", which contradicts TS-RESPONSIVE-023 (line 136) and `src/index.css` (`td:has(> .item-cell) { flex: 1 1 14rem }`, `td:has(> .row-actions) { margin-inline-start: auto }`).

**Suggested work:** Reword to e.g. "Value cells do not grow; only the item cell grows into the space left on its line, and the actions cell is pushed to the line end (TS-RESPONSIVE-023)."

From review comment R1-02 on #198 (`doc/todo/issue-165-r1-02.md`).

## Comments by trusted authors

