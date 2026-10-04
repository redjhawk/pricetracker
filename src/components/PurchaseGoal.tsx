import { useEffect, useState } from "react";
import { Button, InlineLoading, InlineNotification, TextArea } from "@carbon/react";
import { savePurchaseGoal } from "../api/items";

interface Props {
  itemId: string;
  goal: string;
  onSaved: () => void;
}

export default function PurchaseGoal({ itemId, goal, onSaved }: Props) {
  const [draft, setDraft] = useState(goal);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [success, setSuccess] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    if (!dirty) setDraft(goal);
  }, [goal, dirty]);

  async function save() {
    setSaving(true);
    setSuccess("");
    setError("");
    try {
      const result = await savePurchaseGoal(itemId, draft);
      setDraft(result.purchaseGoal);
      setDirty(false);
      setSuccess(result.reviewStarted ? "Purchase goal saved. A new AI review was requested." : "Purchase goal saved.");
      onSaved();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The purchase goal could not be saved.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="purchase-goal" aria-label="Purchase goal">
      <TextArea
        id="purchase-goal"
        labelText="Purchase goal"
        helperText="Sent to the AI review, e.g. what you will use the item for."
        rows={3}
        value={draft}
        onChange={(event) => {
          setDraft(event.target.value);
          setDirty(true);
          setSuccess("");
        }}
        disabled={saving}
      />
      <div className="purchase-goal-actions">
        <Button kind="secondary" size="sm" onClick={() => void save()} disabled={saving || draft.trim() === goal}>
          Save goal
        </Button>
        {saving && <InlineLoading description="Saving the purchase goal…" />}
      </div>
      {success && (
        <InlineNotification kind="success" title={success} lowContrast className="detail-notification" onCloseButtonClick={() => setSuccess("")} />
      )}
      {error && (
        <InlineNotification kind="error" title="Could not save purchase goal" subtitle={error} lowContrast className="detail-notification" />
      )}
    </section>
  );
}
