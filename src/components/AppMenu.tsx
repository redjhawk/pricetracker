import { useState, type KeyboardEvent, type RefObject } from "react";
import { OverflowMenu, OverflowMenuItem } from "@carbon/react";
import { UserAvatar } from "@carbon/icons-react";

interface Props {
  onOpenInformation?: () => void;
  onOpenSettings?: () => void;
  onLogout?: () => void;
  triggerRef?: RefObject<HTMLButtonElement | null>;
}

// Moves focus as an arrow key would once focus is on the first entry, where Carbon puts it.
function focusEntryFromTrigger(trigger: HTMLElement, step: 1 | -1) {
  const menuId = trigger.getAttribute("aria-controls");
  const menu = menuId ? document.getElementById(menuId) : null;
  if (!menu) return false;
  const entries = Array.from(menu.querySelectorAll<HTMLElement>("[role=menuitem]:not([disabled])"));
  if (entries.length === 0) return false;
  entries[(step + entries.length) % entries.length].focus();
  return true;
}

export default function AppMenu({ onOpenInformation, onOpenSettings, onLogout, triggerRef }: Props) {
  const [open, setOpen] = useState(false);

  // Carbon moves focus into the menu only once it is placed; keys pressed earlier reach the trigger.
  function handleTriggerKeyDown(event: KeyboardEvent<HTMLElement>) {
    if (event.key === "Escape") setOpen(false);
    const step = event.key === "ArrowDown" ? 1 : event.key === "ArrowUp" ? -1 : 0;
    if (step !== 0 && focusEntryFromTrigger(event.currentTarget, step)) event.preventDefault();
  }

  return (
    <OverflowMenu
      innerRef={triggerRef}
      className="app-menu"
      renderIcon={UserAvatar}
      aria-label="Application menu"
      iconDescription="Application menu"
      flipped
      size="lg"
      open={open}
      onOpen={() => setOpen(true)}
      onClose={() => setOpen(false)}
      onKeyDown={handleTriggerKeyDown}
    >
      {onOpenInformation && <OverflowMenuItem itemText="Information" onClick={onOpenInformation} />}
      {onOpenSettings && <OverflowMenuItem itemText="Settings" onClick={onOpenSettings} />}
      {onLogout && <OverflowMenuItem itemText="Log out" onClick={onLogout} />}
    </OverflowMenu>
  );
}
