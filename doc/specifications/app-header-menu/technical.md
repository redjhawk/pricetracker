# Technical specification: application header menu

Status: ready (frontend-only subject; implemented together with the settings subject; API approved by the user 2026-10-03 (proposal revision 1 unchanged; see the change index and api-step.md); implementation authorized)\
Functional specification: [functional.md](functional.md), FR-APP-MENU-001 – 006 (ready)\
Companion: [LeBoncoin session settings](../leboncoin-session-settings/technical.md) (the modal this menu opens)

## Requirement mapping

`MENU-*` abbreviates `FR-APP-MENU-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-APP-MENU-001 | MENU-001, MENU-005 | `AppMenu` component in the existing `HeaderGlobalBar`, after the **Add item** button, on every page | `src/components/AppMenu.tsx` (new), `src/App.tsx`, `src/index.css` (only if alignment needs a class) | Playwright: icon visible on list, detail, loading and error states; 400 px layout |
| TS-APP-MENU-002 | MENU-002, MENU-003, MENU-004 | Carbon `OverflowMenu` with `UserAvatar` icon and one `OverflowMenuItem` “Settings”; selecting it opens `SettingsModal` | same | Playwright: click and Enter/Space open; Escape/outside click close; one item; selection opens modal |
| TS-APP-MENU-003 | MENU-004 | Focus return to the trigger via a ref passed to the modal's `launcherButtonRef` | `src/App.tsx`, `src/components/SettingsModal.tsx` | Playwright: focus on trigger after Escape and after modal close |
| TS-APP-MENU-004 | MENU-006 | Carbon modals are focus-trapped and cover the header; no extra code | — | Playwright: with Add item modal open, the trigger is not focusable/clickable |

## Frontend

`src/components/AppMenu.tsx`:

```tsx
interface Props {
  onOpenSettings: () => void;
  triggerRef: React.RefObject<HTMLButtonElement | null>;
}
```

Renders Carbon `OverflowMenu` (installed `@carbon/react` 1.117.0 exports it) with:

- `renderIcon={UserAvatar}` from `@carbon/icons-react` (profile-style appearance, FR-APP-MENU-001).
- `aria-label="Application menu"` and `iconDescription="Application menu"` — describes the menu, never a signed-in user (FR-APP-MENU-004). Carbon sets `aria-haspopup`/`aria-expanded` on the trigger, gives it a focus ring, opens on click/Enter/Space, moves focus into the menu, closes on Escape/outside click and returns focus to the trigger.
- `flipped` so the menu aligns to the right edge and stays on screen at about 400 px (FR-APP-MENU-005); `size="lg"` to match the 48 px header height.
- `ref={triggerRef}` (the component forwards the ref to its trigger button in this Carbon version; the developer confirms with the installed types or Carbon MCP before relying on it, otherwise wraps a `HeaderGlobalAction` + `OverflowMenu` alternative only after reporting back).
- One child: `<OverflowMenuItem itemText="Settings" onClick={onOpenSettings} />` (FR-APP-MENU-003). Carbon closes the menu on selection.

Header styling: the header uses Carbon's default dark UI-shell styles. If the icon colour/alignment does not match the header, add one class in `src/index.css` using Carbon tokens only (e.g. `fill: var(--cds-icon-on-color)`); no hard-coded colours or spacing.

`src/App.tsx`: `const [settingsOpen, setSettingsOpen] = useState(false); const menuButtonRef = useRef<HTMLButtonElement>(null);` Inside `HeaderGlobalBar`, after the existing **Add item** `Button`: `<AppMenu onOpenSettings={() => setSettingsOpen(true)} triggerRef={menuButtonRef} />`. The settings modal receives `launcherButtonRef={menuButtonRef}`. The header is outside the page branches, so the control appears on the list, detail, loading, error and missing-item views (FR-APP-MENU-001). No route, URL, server request or browser storage is involved; opening/closing the menu has no side effects.

FR-APP-MENU-006: Carbon `Modal` renders an overlay above the header with a focus trap, so the trigger is unreachable while the Add item, Delete or Settings modal is open. No additional state is needed.

## Backend

Not affected: the menu needs no data and calls no endpoint.

## API

Not affected: no request is made by the menu. The settings modal it opens uses the pending settings endpoints (see the settings subject).

## Scope and refactoring

Files: new `src/components/AppMenu.tsx`; edits in `src/App.tsx`; optionally one class in `src/index.css`. No new dependency (icons and components are installed). No refactoring proposed; extracting the whole header into its own component is not needed for this change.

## Verification and unresolved questions

- `npm run build` (TypeScript + Vite).
- Playwright spec (shared with the settings subject, `tests/leboncoin-session-settings.spec.ts`, mocked API): trigger visible on list and detail pages and while the list request is pending or failed; accessible name “Application menu” and no user wording; `aria-expanded` toggles; keyboard open with Enter and Space; exactly one menu item “Settings”; Escape closes and focuses the trigger; outside click closes; selecting Settings opens the dialog named “Settings”; closing the dialog returns focus to the trigger; with the Add item dialog open, clicking at the trigger position does not open the menu; at 400×800 the trigger is visible, does not overlap the brand or Add item button (bounding boxes), the menu fits in the viewport, and the page has no horizontal scroll.
- Manual QA: screen reader announcement of the trigger and menu (record as a limitation if unavailable).

No unresolved question.

## Revision note

2026-10-03, review amendment REV-SET-001: status updated to record the API approval of 2026-10-03.
