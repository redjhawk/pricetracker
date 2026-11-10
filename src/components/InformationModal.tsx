import { useEffect, useState, type RefObject } from "react";
import { InlineLoading, InlineNotification, Modal } from "@carbon/react";
import { getSession } from "../api/auth";
import { authRequiredEvent } from "../api/client";

interface Props {
  open: boolean;
  username: string | null;
  onClose: () => void;
  launcherButtonRef: RefObject<HTMLElement | null>;
}

type VersionState = "loading" | "error" | { version: string };

export default function InformationModal({ open, username, onClose, launcherButtonRef }: Props) {
  const [versionState, setVersionState] = useState<VersionState>("loading");
  const [sessionEnded, setSessionEnded] = useState(false);

  useEffect(() => {
    if (!open) return;
    let active = true;
    setVersionState("loading");
    getSession()
      .then((session) => {
        if (!active) return;
        if (session.mode === "protected" && !session.user) {
          // The session ended: hide the stale username and use the existing logged-out flow (login page).
          setSessionEnded(true);
          window.dispatchEvent(new Event(authRequiredEvent));
        }
        setVersionState({ version: session.version });
      })
      .catch(() => active && setVersionState("error"));
    return () => {
      active = false;
    };
  }, [open]);

  return (
    <Modal
      open={open}
      modalHeading="Information"
      primaryButtonText="Close"
      onRequestSubmit={onClose}
      onRequestClose={onClose}
      launcherButtonRef={launcherButtonRef}
      size="sm"
    >
      <dl className="information-list">
        {username && !sessionEnded && (
          <>
            <dt>Username</dt>
            <dd>{username}</dd>
          </>
        )}
        <dt>Version</dt>
        <dd>
          {versionState === "loading" ? (
            <InlineLoading description="Loading version" />
          ) : versionState === "error" ? (
            <InlineNotification kind="error" title="The version could not be loaded." lowContrast hideCloseButton />
          ) : (
            versionState.version
          )}
        </dd>
      </dl>
    </Modal>
  );
}
