# Questions for issue #304: choose an existing purchase goal

The issue says that, when adding a purchase goal, the user can select an existing goal or add a new one if none fits. The answers below are needed before the behaviour can be specified.

1. **What is an "existing purchase goal"?**
   - A: the distinct goal texts currently set on the user's own LeBoncoin items (a goal disappears from the list when no item uses it any more).
   - B: a separate list of saved goals that keeps a goal after the last item using it is deleted or changed.
   - C: other (please describe).

2. **Selecting a goal: copy or link?**
   - A: copy. The selected text becomes the item's own goal. Later editing that item's goal changes only that item.
   - B: link. Several items share one goal. Editing the goal changes it for every item that uses it and starts a new AI review for each of them (when a Claude token is saved).

3. **Where can an existing goal be chosen?**
   - A: only in the add-item form.
   - B: in the add-item form and in the purchase goal field on the LeBoncoin item details page.

4. **How are a choice and a new goal entered?**
   - A: one field. The user types freely and existing goals matching the typed text are suggested; choosing a suggestion fills the field.
   - B: a drop-down list of existing goals plus a "New goal" option that shows the free-text field.
   - C: other (please describe).

5. **Do goals of items shared by other users appear in the list?** (protected mode, [multi-user login](../../specifications/multi-user-login/functional.md))
   - A: no, only the user's own goals.
   - B: yes, also goals of items shared by others that the user can see.

6. **Duplicate goals.** When two items have goals that differ only in letter case or surrounding/inner spaces (for example "Light Linux distro" and "light linux distro"), are they:
   - A: shown as one option (which spelling is shown?), or
   - B: shown as separate options (only exactly identical texts are merged)?

7. **Managing the list.** Should the user be able to rename or remove a goal from the list itself (outside of editing an item)?
   - A: no.
   - B: yes (please say which actions, and what happens to items using that goal).

8. **Order and display of options.** How should options be ordered (alphabetically, most recently used first, most used first), and how is a long multi-line goal shown in the list (in full, or shortened to its first line or first N characters)?

9. **Amazon URLs.** The add-item form shows the purchase goal field for any URL, but goals are saved only for LeBoncoin items (FR-PURCHASE-GOAL-001). Should the goal choice keep this rule (offered in the form, ignored for Amazon), or be hidden/disabled when the URL is an Amazon URL?
