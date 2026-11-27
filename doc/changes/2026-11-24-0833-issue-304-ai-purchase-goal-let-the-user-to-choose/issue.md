# Issue #304 AI purchase goal: let the user to choose from existing AI purchase goals

When ading a purchase goal, the user will be able to select an existing purchase goal.
User can add a new AI purchase goal if no option corresponda tothe current goal

## Comments by trusted authors

### redjhawk at 2026-11-24T14:38:55Z

What is an "existing purchase goal"? >> A 
Selecting a goal: copy or link? >> copy
Where can an existing goal be chosen? >> B
How are a choice and a new goal entered? >> B
Do goals of items shared by other users appear in the list? >>NO
Duplicate goals.  >> B: shown as separate options (only exactly identical texts are merged)?
Managing the list. Should the user be able to rename or remove a goal from the list itself (outside of editing an item)? >> NO.goal is removed with the last item that used that goal
Order and display of options. >> alphabetically 
Amazon URLs.  >> the field is always shown, you can always show the option

### redjhawk at 2026-11-24T15:51:07Z

Long goals in the drop-down. >> just the first 4 words
Changing the wording of the current goal on the details page >> the usercan select "new goal" and write a new one, or select an existing one and modify the text. Modifying the text of an existing one does not impact on other item goal description

### redjhawk at 2026-11-24T19:13:17Z

@claude can you continue?

### redjhawk at 2026-11-25T19:15:37Z

PUedes continuar 


### redjhawk at 2026-11-25T21:39:24Z

@claude Done: `API_SPECIFICATION.md` now has the three-step alphabetical order for `GET /api/v1/purchase-goals` and `technical.md` line 32 reads "Order (review R1-02):" (commit 6b0d452 on `ai-dev/issue-304`). Resume at implement for R2-01.

### redjhawk at 2026-11-25T22:42:25Z

Merge conflict resolved by hand on `ai-dev/issue-304` (commit 35b6627, `chore(factory): merge master into #304 (tests relocked)`).

Functional decision (owner, 2026-11-25): #305 gave Amazon items a purchase goal, so the goal choice of this issue works the same for Amazon items: the drop-down appears in the add-item form and on the details page of Amazon items, the chosen or typed goal is saved for them, and the list of existing goals includes the goals of Amazon items.

The functional and technical specifications, use cases, test cases, API_SPECIFICATION.md, the locked tests (TC-001, TC-003, TC-013) and the goals query follow it. `npm run build` and `npx playwright test` pass (208 passed, 1 skipped TC-014); Go tests pass except the three `amazon_browser_*` tests, which also fail on master on this machine (local browser environment).

Resume at implement for R2-01.

