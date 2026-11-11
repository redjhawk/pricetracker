# UC-TI-06 — Review tracked items full width on a phone

**Primary actor:** Operator, regular user or administrator
**Trigger:** The user opens the tracked items list on a phone, at a viewport 671 px wide or less (for example 320 or 360 px).
**Origin:** Owner follow-up on issue #140 ("que la lista ocupe todo el ancho en móvil, sin márgenes laterales"). Review finding REV-001: the first implementation ignored the Carbon tab panel padding, so a 16 px margin was still visible on each side.

## Preconditions

- Tracked items exist on the Amazon and LeBoncoin tabs.

## Main flow

1. The user opens the tracked items list on the Amazon tab.
2. The item cards span the whole screen width. No page background shows to the left or right of the cards.
3. Inside each card, the labels, values and row actions keep their own inner padding, so nothing touches the screen edge.
4. The user switches to the LeBoncoin tab and sees the same full-width cards.

## Alternatives and errors

- The header, page title, tabs, Amazon-stop warning, search field and empty states keep their current margins.
- At 672 px or more, and on every other screen, the layout is unchanged.

## Postconditions

- The cards run edge to edge, their text stays inset, and the page does not scroll sideways (FR-RESPONSIVE-014 to 017).

Regression test: `320px tracked-items list spans the full width` and `360px tracked-items list spans the full width` in `tests/responsive-layout.spec.ts`.
