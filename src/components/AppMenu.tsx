import type { RefObject } from "react";
import { OverflowMenu, OverflowMenuItem } from "@carbon/react";
import { UserAvatar } from "@carbon/icons-react";

interface Props {
  onOpenInformation?: () => void;
  onOpenSettings?: () => void;
  onLogout?: () => void;
  triggerRef?: RefObject<HTMLButtonElement | null>;
}

export default function AppMenu({ onOpenInformation, onOpenSettings, onLogout, triggerRef }: Props) {
  return (
    <OverflowMenu
      innerRef={triggerRef}
      className="app-menu"
      renderIcon={UserAvatar}
      aria-label="Application menu"
      iconDescription="Application menu"
      flipped
      size="lg"
    >
      {onOpenInformation && <OverflowMenuItem itemText="Information" onClick={onOpenInformation} />}
      {onOpenSettings && <OverflowMenuItem itemText="Settings" onClick={onOpenSettings} />}
      {onLogout && <OverflowMenuItem itemText="Log out" onClick={onLogout} />}
    </OverflowMenu>
  );
}
