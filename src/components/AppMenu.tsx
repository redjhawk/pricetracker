import { useRef, type RefObject } from "react";
import { OverflowMenu, OverflowMenuItem } from "@carbon/react";
import { UserAvatar } from "@carbon/icons-react";

interface Props {
  onOpenInformation?: () => void;
  onOpenSettings?: () => void;
  onLogout?: () => void;
  triggerRef?: RefObject<HTMLButtonElement | null>;
}

export default function AppMenu({ onOpenInformation, onOpenSettings, onLogout, triggerRef }: Props) {
  const usedKeyboard = useRef(false);
  const markKeyboard = () => {
    usedKeyboard.current = true;
  };
  // Carbon can pull focus back into the menu just as a key closes it (a late re-render re-runs its
  // focus-on-open effect), so focus falls to the page once the menu unmounts: give it back to the button.
  const restoreFocus = () => {
    if (usedKeyboard.current && document.activeElement === document.body) triggerRef?.current?.focus();
  };

  return (
    <OverflowMenu
      innerRef={triggerRef}
      className="app-menu"
      renderIcon={UserAvatar}
      aria-label="Application menu"
      iconDescription="Application menu"
      flipped
      size="lg"
      onOpen={() => {
        usedKeyboard.current = false;
      }}
      onClose={restoreFocus}
    >
      {onOpenInformation && <OverflowMenuItem itemText="Information" onClick={onOpenInformation} onKeyDown={markKeyboard} />}
      {onOpenSettings && <OverflowMenuItem itemText="Settings" onClick={onOpenSettings} onKeyDown={markKeyboard} />}
      {onLogout && <OverflowMenuItem itemText="Log out" onClick={onLogout} onKeyDown={markKeyboard} />}
    </OverflowMenu>
  );
}
