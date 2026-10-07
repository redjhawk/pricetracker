import { useEffect, useRef, useState, type RefObject } from "react";
import { FeatureFlags, InlineLoading, InlineNotification, Modal, TextArea, TextInput } from "@carbon/react";
import { ApiError } from "../api/client";
import {
  getClaudeToken,
  getLeboncoinSession,
  saveSettings,
  type ClaudeTokenSettings,
  type LeboncoinSessionSettings,
} from "../api/settings";

const claudeErrorCodes = ["INVALID_CLAUDE_TOKEN", "CLAUDE_TOKEN_REJECTED", "CLAUDE_UNREACHABLE"];

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
  const [claudeToken, setClaudeToken] = useState<ClaudeTokenSettings | null>(null);
  const [claudeValue, setClaudeValue] = useState("");
  const [claudeInvalidMessage, setClaudeInvalidMessage] = useState("");
  const savingRef = useRef(false);

  useEffect(() => {
    if (!open) return;
    let active = true;
    Promise.all([getLeboncoinSession(), getClaudeToken()])
      .then(([loaded, loadedToken]) => {
        if (!active) return;
        setSettings(loaded);
        setValue(loaded.value ?? "");
        setClaudeToken(loadedToken);
        setClaudeValue(loadedToken.value ?? "");
      })
      .catch((error) => { if (active) setLoadError(errorMessage(error)); });
    return () => {
      active = false;
      setSettings(null);
      setValue("");
      setClaudeToken(null);
      setClaudeValue("");
      setClaudeInvalidMessage("");
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
    if (!settings || !claudeToken || changed || savingRef.current) return;
    savingRef.current = true;
    setSaving(true);
    setInvalidMessage("");
    setClaudeInvalidMessage("");
    setSaveError(null);
    const claudeChanged = claudeValue.trim() !== (claudeToken.value ?? "");
    // An unchanged session is not resent with a Claude-only change: resaving it resets its warnings (REV-002).
    const sessionChanged = value !== (settings.value ?? "");
    try {
      await saveSettings({
        ...(sessionChanged || !claudeChanged ? { leboncoinSession: { value, revision: settings.revision } } : {}),
        ...(claudeChanged ? { claudeToken: { value: claudeValue } } : {}),
      });
      onClose();
    } catch (error) {
      if (error instanceof ApiError && error.code === "INVALID_SESSION") setInvalidMessage(error.message);
      else if (error instanceof ApiError && claudeErrorCodes.includes(error.code)) setClaudeInvalidMessage(error.message);
      else if (error instanceof ApiError && error.code === "SESSION_CHANGED") setChanged(true);
      else setSaveError(errorMessage(error));
    } finally {
      savingRef.current = false;
      setSaving(false);
    }
  }

  const loaded = Boolean(settings && claudeToken);
  const loading = open && !loaded && !loadError;
  const claudeHelper = "Paste the token printed by claude setup-token (Claude Pro/Max subscription). It is used for Amazon and LeBoncoin AI reviews. Save an empty field to remove it."
    + (claudeToken && claudeToken.value === null ? " AI reviews are unavailable until a token is saved." : "");
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
        primaryButtonDisabled={!loaded || saving || changed}
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
        {saving && <InlineLoading description="Saving settings…" />}
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
            disabled={!loaded || saving}
            invalid={Boolean(invalidMessage)}
            invalidText={invalidMessage}
          />
        )}
        {!loadError && (
          <div className="settings-claude-token">
            {claudeToken?.lastRejectedAt && (
              <InlineNotification
                kind="warning"
                subtitle={`Claude rejected this token for an AI review on ${formatDate(claudeToken.lastRejectedAt)}. Replace it with a new token from claude setup-token.`}
                lowContrast
                hideCloseButton
                className="modal-notification"
              />
            )}
            <TextInput
              id="claude-token"
              labelText="Claude token"
              helperText={claudeHelper}
              type="text"
              autoComplete="off"
              spellCheck={false}
              value={claudeValue}
              onChange={(event) => {
                setClaudeValue(event.target.value);
                setClaudeInvalidMessage("");
              }}
              disabled={!loaded || saving}
              invalid={Boolean(claudeInvalidMessage)}
              invalidText={claudeInvalidMessage}
            />
          </div>
        )}
      </Modal>
    </FeatureFlags>
  );
}
