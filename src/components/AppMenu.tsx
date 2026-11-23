import { useState, type KeyboardEvent, type RefObject } from "react";
import { OverflowMenu, OverflowMenuItem } from "@carbon/react";
import { UserAvatar } from "@carbon/icons-react";

interface Props {
  onOpenInformation?: () => void;
  onOpenSettings?: () => void;
  onLogout?: () => void;
  triggerRef?: RefObject<HTMLButtonElement | null>;
}

export default function AppMenu({ onOpenInformation, onOpenSettings, onLogout, triggerRef }: Props) {
  const [open, setOpen] = useState(false);

  // Carbon moves focus into the menu only once it is placed; Escape pressed earlier reaches the trigger.
  function closeOnEscape(event: KeyboardEvent) {
    if (event.key === "Escape") setOpen(false);
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
      onKeyDown={closeOnEscape}
    >
      {onOpenInformation && <OverflowMenuItem itemText="Information" onClick={onOpenInformation} />}
      {onOpenSettings && <OverflowMenuItem itemText="Settings" onClick={onOpenSettings} />}
      {onLogout && <OverflowMenuItem itemText="Log out" onClick={onLogout} />}
    </OverflowMenu>
  );
}
