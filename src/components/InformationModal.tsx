import { useEffect, useState, type RefObject } from "react";
import { InlineLoading, InlineNotification, Modal } from "@carbon/react";
import { getSession } from "../api/auth";

interface Props {
  open: boolean;
  username: string | null;
  onClose: () => void;
  launcherButtonRef: RefObject<HTMLElement | null>;
}

type VersionState = "loading" | "error" | { version: string };

export default function InformationModal({ open, username, onClose, launcherButtonRef }: Props) {
  const [versionState, setVersionState] = useState<VersionState>("loading");

  useEffect(() => {
    if (!open) return;
    let active = true;
    setVersionState("loading");
    getSession()
      .then((session) => active && setVersionState({ version: session.version }))
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
        {username && (
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
