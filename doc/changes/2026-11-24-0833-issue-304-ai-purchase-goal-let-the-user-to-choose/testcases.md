# Test cases: choose an existing purchase goal (issue #304)

Sources: functional specification [purchase-goal-choice](../../specifications/purchase-goal-choice/functional.md) (FR-GOAL-CHOICE-001 to FR-GOAL-CHOICE-016) and use cases [UC-GOAL-CHOICE-001 to UC-GOAL-CHOICE-008](../../use-cases/purchase-goal-choice/).

Conventions:

- **Service tests** (`internal/service`, package-internal) use a real SQLite database with the normal migrations (`newReviewService` / `newSessionService`, `insertListing`, the fake reviewer that records review inputs). "Existing goals" is the service call that the technical specification defines for the list; "owner" contexts use `WithPrincipal`.
- **HTTP tests** (`internal/httpapi`) call `server.Handler()` on a real service and SQLite database (`newSettingsServer`). The goals endpoint, its path and its JSON shape are defined in `API_SPECIFICATION.md` by the technical specification; adding an item (`POST /api/v1/items`) and saving a goal (`PUT /api/v1/items/{id}/purchase-goal`) keep their current contracts.
- **E2E tests** (Playwright, new spec file `tests/purchase-goal-choice.spec.ts`, added to `testMatch` in `playwright.config.ts`) mock the API with `context.route`, as `tests/leboncoin-purchase-goal.spec.ts` does, and record the bodies sent by `POST /api/v1/items` and `PUT …/purchase-goal`. The goals route returns the list as the server would (already filtered, merged and ordered); the server rules are proved by TC-001 to TC-005.
- **Drop-down:** the labelled "Purchase goal (optional)" control of FR-GOAL-CHOICE-001; **field:** the goal text field; **Save goal:** the details page save action.
- Existing e2e specs that abort unknown API routes (for example `leboncoin-purchase-goal.spec.ts`, `responsive-layout.spec.ts`) see the goals request fail and get the free-text field (FR-GOAL-CHOICE-013); they keep their assertions and must still pass.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-GOAL-CHOICE-001 | TC-001, TC-002, TC-006 |
| FR-GOAL-CHOICE-002 | TC-007, TC-008 |
| FR-GOAL-CHOICE-003 | TC-002, TC-009 |
| FR-GOAL-CHOICE-004 | TC-012 |
| FR-GOAL-CHOICE-005 | TC-001, TC-003, TC-006, TC-007 |
| FR-GOAL-CHOICE-006 | TC-001, TC-006 |
| FR-GOAL-CHOICE-007 | TC-004, TC-008, TC-011 |
| FR-GOAL-CHOICE-008 | TC-003, TC-005, TC-007, TC-009 |
| FR-GOAL-CHOICE-009 | TC-003, TC-013 |
| FR-GOAL-CHOICE-010 | TC-009, TC-011 |
| FR-GOAL-CHOICE-011 | TC-001, TC-014 |
| FR-GOAL-CHOICE-012 | TC-010 |
| FR-GOAL-CHOICE-013 | TC-010 |
| FR-GOAL-CHOICE-014 | TC-007, TC-008, TC-011 |
| FR-GOAL-CHOICE-015 | TC-004, TC-006, TC-008, TC-011 |
| FR-GOAL-CHOICE-016 | TC-006, TC-011, TC-012 |

| Use case | Test cases |
| --- | --- |
| UC-GOAL-CHOICE-001 | TC-001, TC-002, TC-006, TC-008, TC-010, TC-012 |
| UC-GOAL-CHOICE-002 | TC-003, TC-005, TC-007, TC-008 |
| UC-GOAL-CHOICE-003 | TC-002, TC-003, TC-009 |
| UC-GOAL-CHOICE-004 | TC-004, TC-005, TC-010, TC-011, TC-012 |
| UC-GOAL-CHOICE-005 | TC-004, TC-008, TC-011 |
| UC-GOAL-CHOICE-006 | TC-003, TC-005 |
| UC-GOAL-CHOICE-007 | TC-003, TC-013 |
| UC-GOAL-CHOICE-008 | TC-001, TC-014 |

## Cases

### TC-001 — Existing goals: distinct exact texts, alphabetical ignoring case, own LeBoncoin items only

- Covers: FR-GOAL-CHOICE-001, FR-GOAL-CHOICE-005, FR-GOAL-CHOICE-006, FR-GOAL-CHOICE-011 (goals of others not offered); UC-GOAL-CHOICE-001, UC-GOAL-CHOICE-008
- Level: integration (Go service, real SQLite)
- Given protected mode with users `alice` and `bob`; alice's LeBoncoin items have the goals "photo editing", "Gaming", "Gaming", "gaming", "browsing" and "install a light Linux distro for my parents"; bob's LeBoncoin item has the goal "Office"
- When alice's existing goals are read
- Then the list is exactly "browsing", "Gaming", "gaming", "install a light Linux distro for my parents", "photo editing" ("Gaming" and "gaming" adjacent, each once); it contains full texts, not shortened ones.
- Corner cases:
  - spacing: goals "Gaming on Linux" and "Gaming  on Linux" (two spaces) are two entries;
  - multi-line goal "line one\nline two" is listed once with its line break;
  - empty goals (items without a goal) are not listed;
  - an Amazon row with a non-empty goal inserted directly in the database is not listed;
  - bob's list is exactly "Office"; a user with no item gets an empty list (not an error);
  - open mode (no regular user): the list holds the goals of all LeBoncoin items;
  - repeated reads return the same order.

### TC-002 — Goals endpoint over HTTP

- Covers: FR-GOAL-CHOICE-001, FR-GOAL-CHOICE-003 (empty list); UC-GOAL-CHOICE-001, UC-GOAL-CHOICE-003
- Level: integration (Go HTTP handler, real service and SQLite)
- Given protected mode, `alice` logged in, her LeBoncoin items with goals "Gaming" and "browsing", and `bob`'s item with goal "Office"
- When alice requests the goals endpoint
- Then the answer is 200 with "browsing", "Gaming" in that order and without "Office".
- Corner cases:
  - no goal on any of alice's items: 200 with an empty JSON array (not `null`);
  - without a session cookie in protected mode: 401 as for the other item endpoints;
  - open mode: answered without login.

### TC-003 — The list follows item additions, deletions and goal changes

- Covers: FR-GOAL-CHOICE-008, FR-GOAL-CHOICE-005, FR-GOAL-CHOICE-009; UC-GOAL-CHOICE-002, UC-GOAL-CHOICE-003, UC-GOAL-CHOICE-006, UC-GOAL-CHOICE-007
- Level: integration (Go service, real SQLite)
- Given LeBoncoin item A with goal "Gaming" and item B with goal "Office", no other goal
- When a LeBoncoin item C is added with goal "photo editing", then A is removed
- Then after the addition the list is "Gaming", "Office", "photo editing"; after the removal it is "Office", "photo editing".
- Corner cases:
  - goal changed: setting A's goal to "Office" removes "Gaming" and leaves "Office" once;
  - goal cleared: setting A's goal to "" (or whitespace only) removes "Gaming";
  - goal still used: with C also on "Gaming", removing A keeps "Gaming";
  - last goal gone: removing every item with a goal gives an empty list;
  - identical text: adding a LeBoncoin item with goal "Gaming" while A has it keeps "Gaming" once;
  - case variant: adding an item with "gaming" adds a separate entry;
  - Amazon: adding an Amazon item with goal "Gaming" or "Brand new text" stores no goal on it and leaves the list unchanged;
  - removal refused (another user's item, not removed): the list is unchanged.

### TC-004 — A copied goal is independent per item; only the changed item is reviewed

- Covers: FR-GOAL-CHOICE-007, FR-GOAL-CHOICE-015; UC-GOAL-CHOICE-004, UC-GOAL-CHOICE-005
- Level: integration (Go service, real SQLite, fake reviewer, Claude token saved)
- Given LeBoncoin items A and B, both with goal "Gaming", and no review running
- When A's goal is set to "Gaming on Linux"
- Then A's goal is "Gaming on Linux", B's goal is still "Gaming", one review is started for A with goal "Gaming on Linux" and none for B; the list holds "Gaming" and "Gaming on Linux".
- Corner cases:
  - A's goal cleared: B keeps "Gaming", the list still holds "Gaming" (used by B), only A is reviewed;
  - A's goal set to "Office" (existing goal of item C): C is not reviewed and its goal is unchanged; the list holds "Office" once;
  - setting A's goal to its current text (with surrounding spaces) changes nothing and starts no review.

### TC-005 — HTTP flow: add with a goal, change, delete, list reflects each step

- Covers: FR-GOAL-CHOICE-008; UC-GOAL-CHOICE-002, UC-GOAL-CHOICE-004, UC-GOAL-CHOICE-006
- Level: integration (Go HTTP handler, real service and SQLite; the listing fetch is the existing test double)
- Given open mode and no item
- When `POST /api/v1/items` adds a LeBoncoin item with `purchaseGoal` "  Gaming  ", then the goals endpoint is read; then `PUT …/purchase-goal` sets "Office"; then the goals endpoint is read; then `DELETE` removes the item; then the goals endpoint is read
- Then the reads return "Gaming" (trimmed), then "Office", then an empty array.
- Corner cases: an add that fails (invalid URL, 400) leaves the list unchanged.

### TC-006 — Add form: choose an existing goal; order and four-word display

- Covers: FR-GOAL-CHOICE-001, FR-GOAL-CHOICE-005, FR-GOAL-CHOICE-006, FR-GOAL-CHOICE-015, FR-GOAL-CHOICE-016; UC-GOAL-CHOICE-001
- Level: e2e (Playwright, mocked API)
- Given the goals route returns "browsing", "Gaming", "gaming", "install a light Linux distro for my parents", "photo editing"
- When the operator opens the add-item form, enters a LeBoncoin URL, opens the drop-down and selects "install a light Linux…", then adds the item
- Then before selecting, the drop-down is labelled "Purchase goal (optional)", shows "No goal", and no goal text field is shown; the options are, in order, "No goal", "browsing", "Gaming", "gaming", "install a light Linux…", "photo editing", "New goal"; after selecting, the drop-down shows "install a light Linux…" and the field shows "install a light Linux distro for my parents" and is editable; the POST body has `purchaseGoal` "install a light Linux distro for my parents"; the item's details page shows that goal.
- Corner cases:
  - exactly four words ("set up a server") is shown in full, without "…"; five words shows four and "…";
  - two goals with the same first four words ("install a light Linux distro" and "install a light Linux laptop") are two options, both "install a light Linux…", in list order; selecting the second puts its own full text in the field;
  - a goal with line breaks and tabs ("install\ta\nlight Linux distro") is shown on one line as "install a light Linux…", and the field shows the full text with its line break;
  - a one-word goal ("Gaming") is shown as is;
  - "No goal" kept: the POST body has an empty `purchaseGoal`.

### TC-007 — Add form: new goal when no existing goal fits

- Covers: FR-GOAL-CHOICE-002, FR-GOAL-CHOICE-005, FR-GOAL-CHOICE-008, FR-GOAL-CHOICE-014; UC-GOAL-CHOICE-002
- Level: e2e (Playwright, mocked API; the mocked goals list adds the goal of each posted LeBoncoin item)
- Given the goals route returns "Gaming", "install a light Linux distro"
- When the operator selects "New goal", types "photo editing" and adds a LeBoncoin item, then opens the add-item form again
- Then after selecting "New goal" the field is shown and empty; the POST body has `purchaseGoal` "photo editing"; the reopened form fetches the goals again and offers "No goal", "Gaming", "install a light Linux…", "photo editing", "New goal" with "No goal" selected and no field shown.
- Corner cases:
  - whitespace-only text under "New goal": the POST body's `purchaseGoal` trims to empty and the item has no goal;
  - text "Gaming" (identical to an existing goal): posted as "Gaming"; the reopened list offers "Gaming" once;
  - text "gaming": the reopened list offers "Gaming" and "gaming";
  - add fails (mocked 409 duplicate): the error is shown as today, the dialog stays with "New goal" selected and the typed text kept.

### TC-008 — Selecting an option replaces the field; abandoned text is not saved

- Covers: FR-GOAL-CHOICE-014, FR-GOAL-CHOICE-002, FR-GOAL-CHOICE-007, FR-GOAL-CHOICE-015; UC-GOAL-CHOICE-001, UC-GOAL-CHOICE-002, UC-GOAL-CHOICE-005
- Level: e2e (Playwright, mocked API)
- Given the goals route returns "Gaming", "install a light Linux distro for my parents"
- When the operator selects "New goal", types "x", then selects "Gaming" and adds a LeBoncoin item
- Then the field shows "Gaming" and the POST body has `purchaseGoal` "Gaming".
- Corner cases (each a fresh form):
  - "Gaming" selected, text changed to "Gaming!", then "No goal" selected: the field is hidden and the POST body's `purchaseGoal` is empty;
  - "Gaming" modified, then "New goal" selected: the field is empty (not "Gaming!");
  - "No goal" → "New goal" → type "a" → "No goal" → "New goal": the field is empty;
  - existing goal adapted: "install a light Linux…" selected, text changed to "install a light Linux distro for my sister"; the drop-down still shows "install a light Linux…"; the POST body has the adapted text (the other item's goal is not sent or changed);
  - existing goal selected and its text cleared (or left as spaces only): the item is added with no goal;
  - same option selected again after editing ("Gaming" → edit → "Gaming"): the field shows "Gaming".

### TC-009 — No existing goal: free-text field as before; first goal saved on the details page

- Covers: FR-GOAL-CHOICE-003, FR-GOAL-CHOICE-008, FR-GOAL-CHOICE-010; UC-GOAL-CHOICE-003
- Level: e2e (Playwright, mocked API; the goals route returns an empty list until a goal is saved)
- Given the goals route returns an empty list
- When the operator opens the add-item form
- Then it shows the empty multi-line textbox "Purchase goal (optional)" and no drop-down; typing "Gaming" and adding posts `purchaseGoal` "Gaming".
- And given a LeBoncoin item without goal and an empty goals list, when its details page opens, then it shows the empty "Purchase goal" textbox and **Save goal** (disabled) and no drop-down; when the operator types "Gaming" and saves, then the success message is announced and the page shows the drop-down with "Gaming" selected, the field showing "Gaming" and **Save goal** disabled.
- Corner cases:
  - nothing typed in the add form: the POST body's `purchaseGoal` is empty and the reopened form still has no drop-down;
  - save fails (mocked 500) on the details page: the error is shown, "Gaming" stays in the textbox and no drop-down appears.

### TC-010 — Goals loading and goals not loadable

- Covers: FR-GOAL-CHOICE-012, FR-GOAL-CHOICE-013; UC-GOAL-CHOICE-001, UC-GOAL-CHOICE-004
- Level: e2e (Playwright, mocked API)
- Given the goals route is held unanswered
- When the operator opens the add-item form, enters a LeBoncoin URL and clicks **Add item**
- Then the goal entry shows a loading indicator, the add action is enabled, and the POST body has an empty `purchaseGoal`.
- And given the goals route answers 500, when the add-item form opens, then the free-text "Purchase goal (optional)" textbox is shown without a drop-down, and a typed "Gaming" is posted.
- Corner cases:
  - details page while loading: the goal entry shows a loading indicator; once the held route answers, the drop-down appears with the item's goal selected;
  - details page with the goals route failing: item A with goal "Gaming" shows "Gaming" in the "Purchase goal" textbox with **Save goal**; changing it to "Office" and saving sends a PUT with "Office";
  - network abort instead of 500: same fallback as 500.

### TC-011 — Details page: change, remove, adapt and save the goal

- Covers: FR-GOAL-CHOICE-010, FR-GOAL-CHOICE-007, FR-GOAL-CHOICE-014, FR-GOAL-CHOICE-015, FR-GOAL-CHOICE-016; UC-GOAL-CHOICE-004, UC-GOAL-CHOICE-005
- Level: e2e (Playwright, mocked API, Claude token configured)
- Given item A with goal "Gaming" and the goals route returning "Gaming", "Office"
- When A's details page opens, the operator selects "Office" and clicks **Save goal**, then reloads
- Then on opening "Gaming" is selected, the field shows "Gaming" and **Save goal** is disabled; after selecting "Office" the field shows "Office" and **Save goal** is enabled; saving sends one PUT for A with "Office", announces success and shows a pending review; after reload "Office" is selected and shown in the field. No PUT is sent for any other item.
- Corner cases (each from A with "Gaming"):
  - "No goal" selected: the field is hidden, **Save goal** is enabled; saving sends "" and, after reload, "No goal" is selected;
  - "New goal", type "photo editing", save: PUT "photo editing"; after reload the drop-down offers "photo editing" and has it selected;
  - "New goal" with no text typed: the goal to save is empty, different from "Gaming", so **Save goal** is enabled and saving clears the goal;
  - field text changed to "Gaming on Linux": the drop-down still shows "Gaming", **Save goal** is enabled, saving sends "Gaming on Linux";
  - back to current: "Office" then "Gaming" selected again, or the text changed and typed back to "Gaming" (or "Gaming " with a trailing space): **Save goal** is disabled;
  - item without goal and a non-empty list: "No goal" selected, no field, **Save goal** disabled;
  - save fails (mocked 500): the error is shown, the selection ("Office") and the field text stay, and after reload "Gaming" is still selected;
  - long goal "install a light Linux distro for my parents": the selected value reads "install a light Linux…" and the field shows the full text.

### TC-012 — Keyboard, labels, announcements and narrow screen

- Covers: FR-GOAL-CHOICE-004, FR-GOAL-CHOICE-016 (one line); UC-GOAL-CHOICE-001, UC-GOAL-CHOICE-004
- Level: e2e (Playwright, mocked API)
- Given the goals route returns "Gaming", "install a light Linux distro for my parents"
- When, in the add-item form, the operator uses only the keyboard: Tab to the drop-down, open it, move with the arrow keys to "Gaming", confirm with Enter, Tab to the field, type " on Linux", and submit
- Then the drop-down and the field each have a visible label and an accessible name; the drop-down exposes the selected option ("Gaming") to assistive technologies (accessible value or selected option state); the POST body has "Gaming on Linux".
- And on a details page, using only the keyboard, the operator selects "install a light Linux…", reaches **Save goal** with Tab and activates it with Enter; the PUT is sent and the result is announced (live region).
- Corner cases:
  - Escape closes the open drop-down without changing the selection;
  - narrow screen (320 px wide): in the add-item form and on the details page, the drop-down, its open options and the field fit inside the viewport without horizontal page scroll, and each option label stays on one line.

### TC-013 — Amazon URL: goal choice offered, goal not saved

- Covers: FR-GOAL-CHOICE-009; UC-GOAL-CHOICE-007
- Level: e2e (Playwright, mocked API) and integration (Go service, real SQLite, in TC-003)
- Given the goals route returns "Gaming", and `POST /api/v1/items` answers 201 with an Amazon item whose `purchaseGoal` is empty
- When the operator enters an Amazon URL, selects "Gaming" and adds the item
- Then the drop-down is shown as for a LeBoncoin URL, the item is added (dialog closes, item listed), and its details page shows no purchase goal entry.
- Corner cases: "New goal" with typed text and an Amazon URL is added normally; the server side (no goal stored, list unchanged) is proved by TC-003.

### TC-014 — Item shared by another user: goal read-only

- Covers: FR-GOAL-CHOICE-011; UC-GOAL-CHOICE-008
- Level: e2e (Playwright, mocked API)
- Given a LeBoncoin item owned by another user and shared with the logged-in user, with goal "Office", and the goals route returning "Gaming"
- When its details page opens
- Then the goal "Office" is shown read-only; no drop-down, no editable goal field and no **Save goal** are offered.
- Corner cases: a shared item without goal shows no goal and no way to set one.
- Note: item sharing (FR-SHARE-004) is not yet in the API or the UI on the base branch. The technical specification states how a shared item is recognised (the existing read-only marker once sharing exists); until then this case cannot be exercised end to end and is recorded as pending, while TC-001 proves that goals of other users' items are never offered.
