import type { RefObject } from "react";
import { OverflowMenu, OverflowMenuItem } from "@carbon/react";
import { UserAvatar } from "@carbon/icons-react";

interface Props {
  onOpenSettings: () => void;
  triggerRef: RefObject<HTMLButtonElement | null>;
}

export default function AppMenu({ onOpenSettings, triggerRef }: Props) {
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
      <OverflowMenuItem itemText="Settings" onClick={onOpenSettings} />
    </OverflowMenu>
  );
}
