import { useEffect, useRef, useState, type RefObject } from "react";
import { Button, FeatureFlags, InlineLoading, InlineNotification, Modal, TextArea, TextInput } from "@carbon/react";
import { ApiError } from "../api/client";
import {
  getClaudeToken,
  getLeboncoinSession,
  saveSettings,
  type ClaudeTokenSettings,
  type LeboncoinSessionSettings,
  type SettingsChange,
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

// The state line of a secret entry: pending removal, saved (with its masked hint when the server gives one) or none.
// The hint is display text only; no request value is ever derived from it (issue #265).
function secretState(noun: "session" | "token", secret: { saved: boolean; hint: string | null }, removing: boolean) {
  if (removing) return `The saved ${noun} will be removed when you save.`;
  if (!secret.saved) return `No ${noun} is saved.`;
  return secret.hint === null ? `A ${noun} is saved.` : `A ${noun} is saved: ••••${secret.hint}.`;
}

function helperText(state: string | null, help: string) {
  return (
    <>
      {state && <span className="settings-secret-state" aria-live="polite">{state}</span>}
      {help}
    </>
  );
}

function sessionWarning(settings: LeboncoinSessionSettings): string | null {
  const notUsed = "It is no longer used. Capture a new session and save it here, or remove it.";
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
  const [removeSession, setRemoveSession] = useState(false);
  const [removeToken, setRemoveToken] = useState(false);
  const savingRef = useRef(false);

  useEffect(() => {
    if (!open) return;
    let active = true;
    // The inputs always start empty: the server never returns a saved secret.
    Promise.all([getLeboncoinSession(), getClaudeToken()])
      .then(([loaded, loadedToken]) => {
        if (!active) return;
        setSettings(loaded);
        setClaudeToken(loadedToken);
      })
      .catch((error) => { if (active) setLoadError(errorMessage(error)); });
    return () => {
      active = false;
      setSettings(null);
      setValue("");
      setRemoveSession(false);
      setClaudeToken(null);
      setClaudeValue("");
      setRemoveToken(false);
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
    // Only replacements (non-blank inputs) and removals are sent; an untouched entry is left out.
    const change: SettingsChange = {};
    if (removeSession) change.leboncoinSession = { remove: true, revision: settings.revision };
    else if (value.trim()) change.leboncoinSession = { value, revision: settings.revision };
    if (removeToken) change.claudeToken = { remove: true };
    else if (claudeValue.trim()) change.claudeToken = { value: claudeValue };
    if (!change.leboncoinSession && !change.claudeToken) {
      onClose();
      return;
    }
    savingRef.current = true;
    setSaving(true);
    setInvalidMessage("");
    setClaudeInvalidMessage("");
    setSaveError(null);
    try {
      await saveSettings(change);
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
  const claudeHelp = "Paste the token printed by claude setup-token (Claude Pro/Max subscription). It is used for Amazon and LeBoncoin AI reviews. Leave it empty to keep the saved token."
    + (claudeToken && !claudeToken.saved ? " AI reviews are unavailable until a token is saved." : "");
  const sessionHelp = "Paste the datadome value or a cookie string containing datadome=…. Leave it empty to keep the saved session.";
  const warning = settings ? sessionWarning(settings) : null;

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
        {warning && (
          <InlineNotification kind="warning" subtitle={warning} lowContrast hideCloseButton className="modal-notification" />
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
            helperText={helperText(settings && secretState("session", settings, removeSession), sessionHelp)}
            rows={4}
            value={value}
            onChange={(event) => {
              setValue(event.target.value);
              setInvalidMessage("");
            }}
            disabled={!loaded || saving || removeSession}
            invalid={Boolean(invalidMessage)}
            invalidText={invalidMessage}
          />
        )}
        {!loadError && settings?.saved && (
          <Button
            kind="ghost"
            size="sm"
            className="settings-secret-remove"
            disabled={!loaded || saving}
            onClick={() => {
              setRemoveSession(!removeSession);
              setValue("");
              setInvalidMessage("");
            }}
          >
            {removeSession ? "Undo session removal" : "Remove session"}
          </Button>
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
              helperText={helperText(claudeToken && secretState("token", claudeToken, removeToken), claudeHelp)}
              type="text"
              autoComplete="off"
              spellCheck={false}
              value={claudeValue}
              onChange={(event) => {
                setClaudeValue(event.target.value);
                setClaudeInvalidMessage("");
              }}
              disabled={!loaded || saving || removeToken}
              invalid={Boolean(claudeInvalidMessage)}
              invalidText={claudeInvalidMessage}
            />
            {claudeToken?.saved && (
              <Button
                kind="ghost"
                size="sm"
                className="settings-secret-remove"
                disabled={!loaded || saving}
                onClick={() => {
                  setRemoveToken(!removeToken);
                  setClaudeValue("");
                  setClaudeInvalidMessage("");
                }}
              >
                {removeToken ? "Undo token removal" : "Remove token"}
              </Button>
            )}
          </div>
        )}
      </Modal>
    </FeatureFlags>
  );
}
