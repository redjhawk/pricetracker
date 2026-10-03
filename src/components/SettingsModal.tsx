import { useEffect, useRef, useState, type RefObject } from "react";
import { FeatureFlags, InlineLoading, InlineNotification, Modal, TextArea } from "@carbon/react";
import { ApiError } from "../api/client";
import { getLeboncoinSession, saveLeboncoinSession, type LeboncoinSessionSettings } from "../api/settings";

interface Props {
  open: boolean;
  onClose: () => void;
  launcherButtonRef: RefObject<HTMLElement | null>;
}

const dateTime = new Intl.DateTimeFormat("en-GB", {
  day: "2-digit",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "UTC",
  timeZoneName: "short",
});

const formatDate = (value: string) => dateTime.format(new Date(value));

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "The server could not be reached.";
}

function sessionHint(settings: LeboncoinSessionSettings): string | null {
  const notUsed = "It is no longer used. Capture a new session and save it here, or save an empty field to stop using it.";
  if (settings.status === "revoked" && settings.revokedAt) {
    return `LeBoncoin revoked this session on ${formatDate(settings.revokedAt)}. ${notUsed}`;
  }
  if (settings.status === "expired" && settings.expiresAt) {
    return `This session expired on ${formatDate(settings.expiresAt)}. ${notUsed}`;
  }
  if (settings.status !== "active" || !settings.lastAttempt) return null;
  const attemptedAt = formatDate(settings.lastAttempt.attemptedAt);
  if (settings.lastAttempt.outcome === "rejected") {
    return `LeBoncoin rejected this session on ${attemptedAt}. Capture a new session and save it here.`;
  }
  if (settings.lastAttempt.outcome === "failed") {
    return `The last LeBoncoin check using this session failed on ${attemptedAt} for a reason other than a rejection.`;
  }
  return null;
}

export default function SettingsModal({ open, onClose, launcherButtonRef }: Props) {
  const [settings, setSettings] = useState<LeboncoinSessionSettings | null>(null);
  const [value, setValue] = useState("");
  const [loadError, setLoadError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [invalidMessage, setInvalidMessage] = useState("");
  const [saveError, setSaveError] = useState<string | null>(null);
  const [changed, setChanged] = useState(false);
  const savingRef = useRef(false);

  useEffect(() => {
    if (!open) return;
    let active = true;
    getLeboncoinSession()
      .then((loaded) => {
        if (!active) return;
        setSettings(loaded);
        setValue(loaded.value ?? "");
      })
      .catch((error) => { if (active) setLoadError(errorMessage(error)); });
    return () => {
      active = false;
      setSettings(null);
      setValue("");
      setLoadError(null);
      setInvalidMessage("");
      setSaveError(null);
      setChanged(false);
    };
  }, [open]);

  function close() {
    if (savingRef.current) return;
    onClose();
  }

  async function save() {
    if (!settings || changed || savingRef.current) return;
    savingRef.current = true;
    setSaving(true);
    setInvalidMessage("");
    setSaveError(null);
    try {
      await saveLeboncoinSession(value, settings.revision);
      onClose();
    } catch (error) {
      if (error instanceof ApiError && error.code === "INVALID_SESSION") setInvalidMessage(error.message);
      else if (error instanceof ApiError && error.code === "SESSION_CHANGED") setChanged(true);
      else setSaveError(errorMessage(error));
    } finally {
      savingRef.current = false;
      setSaving(false);
    }
  }

  const loading = open && !settings && !loadError;
  const hint = settings ? sessionHint(settings) : null;

  return (
    // Carbon's keyboard-based focus wrap (no sentinel spans) keeps Tab/Shift+Tab inside this modal
    // synchronously, including while the field and Save are disabled.
    <FeatureFlags enableFocusWrapWithoutSentinels>
      <Modal
        open={open}
        modalHeading="Settings"
        primaryButtonText={saving ? "Saving…" : "Save"}
        secondaryButtonText="Cancel"
        primaryButtonDisabled={!settings || saving || changed}
        onRequestSubmit={() => void save()}
        onRequestClose={close}
        launcherButtonRef={launcherButtonRef}
        // Save and the field are disabled while loading, so Carbon's default target (Save) cannot take focus.
        // Cancel is enabled in every state reached on open. The close icon button is avoided because
        // its tooltip consumes the first Escape press.
        selectorPrimaryFocus=".cds--modal-footer .cds--btn--secondary"
        size="sm"
      >
        {loading && <InlineLoading description="Loading settings…" />}
        {loadError && (
          <InlineNotification
            kind="error"
            title="Could not load settings"
            subtitle={loadError}
            lowContrast
            hideCloseButton
            className="modal-notification"
          />
        )}
        {hint && (
          <InlineNotification kind="warning" subtitle={hint} lowContrast hideCloseButton className="modal-notification" />
        )}
        {changed && (
          <InlineNotification
            kind="warning"
            title="Session changed"
            subtitle="The LeBoncoin session changed after Settings was opened. Nothing was saved. Copy your text if needed, then close and reopen Settings before saving."
            lowContrast
            hideCloseButton
            className="modal-notification"
          />
        )}
        {saveError && (
          <InlineNotification
            kind="error"
            title="Could not save settings"
            subtitle={saveError}
            lowContrast
            hideCloseButton
            className="modal-notification"
          />
        )}
        {saving && <InlineLoading description="Saving the session…" />}
        {!loadError && (
          <TextArea
            id="leboncoin-session"
            labelText="LeBonCoin session"
            helperText="Paste the datadome value or a cookie string containing datadome=…. Save an empty field to remove the session."
            rows={4}
            value={value}
            onChange={(event) => {
              setValue(event.target.value);
              setInvalidMessage("");
            }}
            disabled={!settings || saving}
            invalid={Boolean(invalidMessage)}
            invalidText={invalidMessage}
          />
        )}
      </Modal>
    </FeatureFlags>
  );
}
