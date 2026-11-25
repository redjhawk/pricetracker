# Technical specification: choose an existing purchase goal

Status: ready
Functional specification: [functional.md](functional.md) (ready, FR-GOAL-CHOICE-001–016)
Test cases: [testcases.md](../../changes/2026-11-24-0833-issue-304-ai-purchase-goal-let-the-user-to-choose/testcases.md) (TC-001–TC-014)
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "LeBoncoin purchase goal (2026-10-04)" → "List the existing purchase goals (issue #304)"
Extends: [LeBoncoin purchase goal technical specification](../leboncoin-purchase-goal/technical.md)

## Summary

The list of existing goals is not stored: the server derives it at each request from the requester's items (LeBoncoin and Amazon) (`GET /api/v1/purchase-goals`). Choosing a goal is purely client-side: the chosen or edited text is sent through the unchanged `POST /api/v1/items` and `PUT /api/v1/items/{id}/purchase-goal`. Copy semantics (FR-GOAL-CHOICE-007) and list maintenance (FR-GOAL-CHOICE-008) follow from this without new writes, tables or migrations.

## Requirement mapping

| Technical ID | Functional IDs | Technical solution |
| --- | --- | --- |
| TS-GOAL-CHOICE-001 | FR-GOAL-CHOICE-005, -006, -008, -009, -011 | Store `PurchaseGoals(ctx, ownerID)` selects the distinct non-empty goals of the owner's tracked items (both platforms since issue #305) (exact match, `BINARY` collation). Service `PurchaseGoals(ctx)` uses `ownerFrom(ctx)` and sorts alphabetically: base letters (case and accents ignored), then `strings.ToLower` comparison, then exact byte order. Derived at each call, so additions, deletions and goal changes are reflected at once and a goal disappears with its last item. |
| TS-GOAL-CHOICE-002 | FR-GOAL-CHOICE-001, -003, -013 | `GET /api/v1/purchase-goals` → `200 {"purchaseGoals": [...]}` (`[]` when none), behind the existing access checks of `handleAPI`; other methods `405` with `Allow: GET`; storage error `500 INTERNAL_ERROR`. |
| TS-GOAL-CHOICE-003 | FR-GOAL-CHOICE-001, -008, -012, -013 | Frontend `listPurchaseGoals()` in `src/api/items.ts` and hook `usePurchaseGoals(active)` with states `loading` / `ready` / `failed`; fetched each time the add-item form opens and when the details goal entry mounts or after a successful save. |
| TS-GOAL-CHOICE-004 | FR-GOAL-CHOICE-001, -002, -004, -006, -014, -015, -016 | Presentational `PurchaseGoalChoice` component: Carbon `Dropdown` ("No goal", goals in server order, "New goal") plus a `TextArea` for the goal text; selection rules of FR-GOAL-CHOICE-014; four-word labels (`goalLabel`). |
| TS-GOAL-CHOICE-005 | FR-GOAL-CHOICE-001, -002, -003, -009, -012, -013, -014 | `AddItemModal` uses `usePurchaseGoals(open)` and `PurchaseGoalChoice`; sends `goalToSave(...)` as `purchaseGoal` for every URL. |
| TS-GOAL-CHOICE-006 | FR-GOAL-CHOICE-003, -010, -013, -014, -015 | Details `PurchaseGoal` uses `PurchaseGoalChoice`, merges the item's own goal into the list, enables **Save goal** only when the goal to save differs from the stored one, reloads the goals after a successful save. |
| TS-GOAL-CHOICE-007 | FR-GOAL-CHOICE-007, -015 | No backend change: each item keeps its own `purchase_goal` column value; the existing `SetPurchaseGoal` writes one item and starts a review only for it. Selecting an existing goal copies its text into the request. |
| TS-GOAL-CHOICE-008 | FR-GOAL-CHOICE-011 | Shared items do not exist yet (FR-SHARE-004 not implemented): every item returned by the API is owned by the requester (`ownedListing`). The goal choice is rendered only in the owner's editable goal entry; when sharing adds its read-only marker to the item response, `ItemDetail` must render the goal as read-only text instead of `PurchaseGoal`. The list itself never contains other users' goals (TS-GOAL-CHOICE-001). |
| TS-GOAL-CHOICE-009 | FR-GOAL-CHOICE-004, -016 | Accessibility and narrow screens: visible labels (`titleText`, `labelText`), downshift keyboard model of the Carbon `Dropdown`, `autoAlign` floating menu, Carbon's one-line option labels, existing `role="status"` success notification. |

## API

Defined in `API_SPECIFICATION.md` ("List the existing purchase goals (issue #304)"):

- `GET /api/v1/purchase-goals` → `200 { "purchaseGoals": string[] }`, ordered and merged by the server; `[]`, never `null`.
- Order (review R1-02): alphabetical, in three steps. First by the base letters: the text in Unicode lower case, with the Latin letters with diacritics mapped to their base letters (`à á â ã ä å` → `a`, `ç` → `c`, `ð` → `d`, `è é ê ë` → `e`, `ì í î ï` → `i`, `ñ` → `n`, `ò ó ô õ ö ø` → `o`, `ù ú û ü` → `u`, `ý ÿ` → `y`) and the ligatures expanded (`æ` → `ae`, `œ` → `oe`, `ß` → `ss`), compared in byte order. Then by the lower-case text. Then by the exact text (byte order). So `"écran"` comes between `"dessin"` and `"zoom"`, `"ecran"` before `"écran"`, and `"Gaming"` before `"gaming"`. For ASCII texts this is the previous order.
- Errors: `401 AUTH_REQUIRED` (protected mode, no session), `403 FORBIDDEN` (admin), `405 METHOD_NOT_ALLOWED` (`Allow: GET`), `500 INTERNAL_ERROR`.
- `POST /api/v1/items` and `PUT /api/v1/items/{id}/purchase-goal`: unchanged.
- Compatibility: additive endpoint; existing clients unaffected. A path outside `/api/v1/items/` is used so it cannot collide with item ids in the `/api/v1/items/{id}` routes.

## Backend

### Store (`internal/store/store.go`)

```go
// PurchaseGoals returns the distinct non-empty purchase goals of the owner's items, unordered.
func (s *Store) PurchaseGoals(ctx context.Context, ownerID int64) ([]string, error)
```

SQL: `SELECT DISTINCT purchase_goal FROM items WHERE owner_id = ? AND tracked = 1 AND purchase_goal <> ''`. The column has the default `BINARY` collation, so `DISTINCT` merges only identical texts (case, spacing and line breaks kept). Rows are scanned into a slice initialised as `[]string{}`. No schema change, no migration, no index (one owner has few items; the query is a scan of `items` like `List`).

### Service (`internal/service/purchase_goal.go`)

```go
// PurchaseGoals returns the requester's existing purchase goals in display order.
func (s *Service) PurchaseGoals(ctx context.Context) ([]string, error)
```

- Calls `s.store.PurchaseGoals(ctx, ownerFrom(ctx))`. Owner 0 in open mode holds all items (FR-MODE-001/-002), as for `List`.
- Drops entries for which `strings.TrimSpace` is empty (defence for rows written outside the service; the service already stores trimmed goals).
- Sorts with `slices.SortFunc(goals, compareGoals)`, the order of `API_SPECIFICATION.md`: compare the lower-case texts with Latin diacritics mapped to their base letters and ligatures expanded (`baseLetters`, a `strings.Replacer`); on equality compare `strings.ToLower(a)` with `strings.ToLower(b)`; then `a` with `b` (`strings.Compare`). Gives "browsing", "écran", "Gaming", "gaming", … deterministically, with accented first letters among their base letters (FR-GOAL-CHOICE-006 for French goals). The mapping covers the Latin-1 letters and `œ`; `golang.org/x/text/collate` would cover all scripts but adds a dependency, which this change avoids.
- Read only; no transaction, lock or interaction with reviews. Concurrent goal changes are seen by the next request.
- Unchanged: `Add` (trims and stores the goal for both platforms since issue #305, so an Amazon add with a goal joins the list), `SetPurchaseGoal` (one item, review only for that item, unchanged text → no write, no review), `Remove` (owner-checked; a refused removal leaves the list as is).

### HTTP (`internal/httpapi/purchase_goal.go`, `internal/httpapi/server.go`)

- `handleData` routes `path == "/api/v1/purchase-goals"` to new `handlePurchaseGoals(w, r)` before the item routes; it therefore runs after `handleAPI`'s login, mode and admin checks.
- `handlePurchaseGoals`: method other than GET → `Allow: GET` + `405 METHOD_NOT_ALLOWED`; service error → `serverError` (`500`); success → `writeJSON(w, 200, map[string]any{"purchaseGoals": goals})`.

## Frontend

### API module (`src/api/items.ts`)

```ts
export async function listPurchaseGoals(): Promise<string[]>  // GET /api/v1/purchase-goals → response.purchaseGoals
```

### Goal choice helpers (`src/purchaseGoalChoice.ts`, new)

```ts
export type GoalOption = { kind: "none" } | { kind: "existing"; goal: string } | { kind: "new" };
export interface GoalChoice { option: GoalOption; text: string }
export type GoalsState = { status: "loading" } | { status: "ready"; goals: string[] } | { status: "failed" };
```

- `goalLabel(goal)`: `goal.trim().split(/\s+/)`; more than four words → first four joined by one space + `"…"`; otherwise the words joined by one space (one line, FR-GOAL-CHOICE-016).
- `choiceFor(goal)`: `""` → `{ option: none, text: "" }`; otherwise `{ option: existing goal, text: goal }`.
- `goalToSave(goals, choice)`: `loading` → `""`; free-text mode (`failed`, or `ready` with an empty list) → `choice.text`; choice mode → `""` for "No goal", otherwise `choice.text`. The server trims.
- `withGoal(goals, goal)`: returns `goals` plus `goal` when it is non-empty and absent, sorted with a mirror of the server comparator (same base-letter table, then lower-case compare, then exact). Used by the details page so the item's own goal is always an option even if the list is stale.
- `usePurchaseGoals(active: boolean): { state: GoalsState; reload(): void }`: when `active` becomes true, sets `loading` and calls `listPurchaseGoals()`; success → `ready`, any error (HTTP or network) → `failed`. Answers arriving after `active` became false or after a newer request are ignored (request counter in a ref). `reload()` refetches without going back to `loading`; on failure it keeps the previous state.

### `PurchaseGoalChoice` component (`src/components/PurchaseGoalChoice.tsx`, new, presentational)

Props: `id`, `label`, `helperText`, `goals: GoalsState`, `value: GoalChoice`, `onChange(value: GoalChoice)`, `disabled`.

- `loading`: `InlineLoading` "Loading your purchase goals…" (no field; FR-GOAL-CHOICE-012).
- Free-text mode (`failed`, or `ready` with no goal): the current `TextArea` exactly as today: `id={id}`, `labelText={label}`, `helperText`, `rows={3}`, value `value.text`; typing changes `text`, and an option "No goal" becomes "New goal" (`goalToSave` is the same in free-text mode), so if polling makes the list non-empty while the user types, the choice mode shows "New goal" with the typed text instead of hiding it under "No goal" and saving `""` (FR-GOAL-CHOICE-003, -013, -014). Existing ids and labels are kept, so existing e2e tests keep passing.
- Choice mode (`ready` with at least one goal):
  - Carbon `Dropdown` `id={`${id}-choice`}`, `titleText={label}`, `helperText`, `label="No goal"`, `autoAlign`, `items` = memoised option objects `[none, ...goals.map(existing), new]` (stable references; `selectedItem` is the option object matching `value.option`), `itemToString`: "No goal" / `goalLabel(goal)` / "New goal". Two goals with the same label remain separate objects in server order.
  - Selection (`onChange`): existing goal → `{ option, text: goal }` (full text); "New goal" → `{ option, text: "" }`; "No goal" → `{ option, text: "" }`. Re-selecting the already selected option does not fire `onChange` in downshift; it is detected with `downshiftProps.onStateChange` (selection types `ItemClick`, `ToggleButtonKeyDownEnter`, `ToggleButtonKeyDownSpaceButton` with no `selectedItem` in the changes and the menu closing) and re-applies the same rule, so "Gaming" → edit → "Gaming" restores "Gaming" (TC-008). `downshift` (9.4, installed through `@carbon/react`) is not a direct dependency and must not be added to `package.json`: compare `type` with the values in local constants. Downshift uses strings in development builds (`"__item_click__"`, `"__togglebutton_keydown_enter__"`, `"__togglebutton_keydown_space_button__"`) and numbers in production builds (`14`, `7`, `8` for `useSelect` in 9.4), so both are listed. A keyboard re-selection also requires `highlightedIndex` in the changes, so Enter with no highlighted option does not reset the text.
  - When the option is not "No goal": `TextArea` `id={id}`, `labelText="Goal text"`, `helperText="Changes apply to this item only."`, `rows={3}`, value `value.text`; typing changes `text` and keeps the option (the drop-down still shows the selected goal, FR-GOAL-CHOICE-015).
  - `disabled` disables both controls.

### Add-item form (`src/components/AddItemModal.tsx`)

- `const goals = usePurchaseGoals(open)`; state `choice: GoalChoice`, initially and after close/success `{ option: none, text: "" }`. The list is fetched again at every opening (FR-GOAL-CHOICE-008).
- Renders `PurchaseGoalChoice` with `id="add-item-purchase-goal"`, `label="Purchase goal (optional)"`, the current helper text, `disabled={submitting}`.
- Submit sends `onAdded(url.trim(), goalToSave(goals.state, choice))` for any URL (Amazon included, FR-GOAL-CHOICE-009; stored since issue #305). While loading the add action stays enabled and sends `""`. On failure the dialog, selection and text stay.
- `App.tsx` `handleAdd` is unchanged.

### Details page (`src/components/PurchaseGoal.tsx`, `src/components/ItemDetail.tsx`)

- `ItemDetail` renders `<PurchaseGoal key={item.id} …>` so state and the goals request reset per item. Shown for LeBoncoin and Amazon items (issue #305).
- `PurchaseGoal` keeps its props (`itemId`, `goal`, `onSaved`) and the existing `dirty` pattern: `choice` is re-synced to `choiceFor(goal)` while not dirty (polling may update `goal`); selecting or typing sets `dirty`.
- `const goals = usePurchaseGoals(true)`; the state passed to `PurchaseGoalChoice` is `goals.state` with `withGoal(goals, goal)` applied when `ready`. So the details page shows the choice once any goal exists, including right after the first goal is saved on a page that showed only the free-text field (TC-009).
- `PurchaseGoalChoice` with `id="purchase-goal"`, `label="Purchase goal"`, the current helper text, `disabled={saving}`.
- While `loading`: only the loading indicator (no **Save goal**). Otherwise **Save goal** as today, disabled when `saving || goalToSave(...).trim() === goal`.
- Save: `savePurchaseGoal(itemId, goalToSave(...))`. Success: as today (`dirty` false, success notification, `onSaved()`), plus `goals.reload()`. Failure: error notification; selection and text kept.

### Read-only shared items

No change now (TS-GOAL-CHOICE-008). TC-014 cannot be exercised until FR-SHARE-004 exists.

### Styles (`src/index.css`)

Carbon defaults suffice for width: the `Dropdown` is full width, its option labels are single-line with ellipsis, and four-word labels keep them short. One rule, `.purchase-goal-choice` (grid, `gap: var(--cds-spacing-05)`, `min-width: 0`), spaces the drop-down and the "Goal text" field in both places; no hard-coded sizes or colours.

### UI states

| State | Add-item form | Details page |
| --- | --- | --- |
| Goals loading | Loading indicator; **Add item** enabled, no goal sent | Loading indicator; no **Save goal** |
| Goals failed | Free-text "Purchase goal (optional)" | Free-text "Purchase goal" + **Save goal** |
| No goal exists | Same as failed | Same as failed |
| Goals available | Drop-down (+ "Goal text" unless "No goal") | Drop-down with the item's goal selected (+ "Goal text") + **Save goal** |
| Saving / adding | Controls disabled, existing progress indicator | Controls disabled, existing progress indicator |
| Save error | — | Error notification; selection and text kept |

## Tests

| Test case | Test file | Notes |
| --- | --- | --- |
| TC-001 | `internal/service/purchase_goal_choice_test.go` (new) | `PurchaseGoals` with `WithPrincipal`; Amazon row with a goal inserted through `database.Insert` or SQL, listed like a LeBoncoin one. |
| TC-002 | `internal/httpapi/purchase_goal_choice_test.go` (new) | `newSettingsServer`; protected and open mode; `[]` checked on the raw body. |
| TC-003 | `internal/service/purchase_goal_choice_test.go` | Uses `Add`, `Remove`, `SetPurchaseGoal`. |
| TC-004 | `internal/service/purchase_goal_choice_test.go` | `newReviewService(t, true)` and the fake reviewer. |
| TC-005 | `internal/httpapi/purchase_goal_choice_test.go` | Open mode, existing listing fetch double. |
| TC-006 – TC-013 | `tests/leboncoin-purchase-goal.spec.ts` (extended) | New mock helper serving `/api/v1/purchase-goals`; existing tests in the file unchanged. |
| TC-014 | `tests/leboncoin-purchase-goal.spec.ts` | `test.fixme` naming TC-014 and the reason (sharing not implemented, TS-GOAL-CHOICE-008). |

The test cases name a new `tests/purchase-goal-choice.spec.ts` added to `testMatch`. `playwright.config.ts` is outside the paths the test and development stages may change, so the e2e tests go into `tests/leboncoin-purchase-goal.spec.ts`, which `testMatch` already lists. No manual follow-up is needed. Moving them to their own file later would need an owner change to `testMatch`.

Accessible names used by the e2e tests: drop-down `combobox` named "Purchase goal (optional)" (add form) or "Purchase goal" (details); options "No goal", goal labels, "New goal"; goal text `textbox` "Goal text" in choice mode and "Purchase goal (optional)" / "Purchase goal" in free-text mode; button "Save goal".

## Files to change

- Backend: `internal/store/store.go`, `internal/service/purchase_goal.go`, `internal/httpapi/purchase_goal.go`, `internal/httpapi/server.go`.
- Frontend: `src/api/items.ts`, `src/purchaseGoalChoice.ts` (new), `src/components/PurchaseGoalChoice.tsx` (new), `src/components/AddItemModal.tsx`, `src/components/PurchaseGoal.tsx`, `src/components/ItemDetail.tsx`, `src/index.css` (only if needed).
- Tests: as in "Tests".
- Documentation: this file; `API_SPECIFICATION.md` (done in this stage).

## Refactoring

The add-form `TextArea` moves into `PurchaseGoalChoice` (free-text mode) so that both places share one goal entry. Reason: FR-GOAL-CHOICE-003/-010 require the same choice in both places. Scope: `AddItemModal.tsx`, `PurchaseGoal.tsx`. Risk: low; ids, labels and helper texts of the free-text field stay the same, and the existing e2e tests (which abort the new request and so get the free-text field) guard it. No other refactoring.

## Risks

- Carbon `Dropdown` inside the `Modal`: without `autoAlign` the menu could be clipped by the modal body; `autoAlign` floats it.
- Re-selection detection relies on downshift state-change types; covered by TC-008.
- Existing e2e specs that abort unknown routes see a failed goals request and must still pass (free-text fallback).

## Verification

`go vet ./...`, `go test ./...`, `npm run build`, `npx playwright test`.
