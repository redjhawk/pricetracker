# Questions for issue #304: choose an existing purchase goal

Thank you for the answers to Q-1–Q-9; they are recorded in the [specification](../../specifications/purchase-goal-choice/functional.md). Two points are still open.

10. **Long goals in the drop-down.** (Second part of Q-8, not answered.) A goal can be long and span several lines. With the drop-down (Q-4 B), the free-text field is shown only for "New goal", so a selected existing goal is visible only in the drop-down. How should a long goal be shown?
    - A: in full: options and the selected value wrap over several lines.
    - B: shortened to its first line (or first N characters, please give N) with "…" in the list; once selected, the full text is shown read-only below the drop-down.
    - C: shortened in the list and in the selected value; the full text appears in a tooltip on hover/focus.
    - D: other (please describe).

11. **Changing the wording of the current goal on the details page.** Before this feature, the operator edited the goal text directly. Now the current goal is selected in the drop-down; to reword it, the operator selects "New goal". Should the free-text field then:
    - A: be pre-filled with the item's current goal, so it can be edited, or
    - B: start empty?
