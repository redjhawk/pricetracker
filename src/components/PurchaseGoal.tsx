import { useEffect, useMemo, useState } from "react";
import { Button, InlineLoading, InlineNotification } from "@carbon/react";
import { savePurchaseGoal } from "../api/items";
import { choiceFor, goalToSave, usePurchaseGoals, withGoal, type GoalChoice, type GoalsState } from "../purchaseGoalChoice";
import PurchaseGoalChoice from "./PurchaseGoalChoice";

interface Props {
  itemId: string;
  goal: string;
  onSaved: () => void;
}

export default function PurchaseGoal({ itemId, goal, onSaved }: Props) {
  const goals = usePurchaseGoals(true);
  const [choice, setChoice] = useState<GoalChoice>(() => choiceFor(goal));
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [success, setSuccess] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    if (!dirty) setChoice(choiceFor(goal));
  }, [goal, dirty]);

  // The item's own goal is always an option, even if the list is stale.
  const state = useMemo<GoalsState>(
    () => goals.state.status === "ready" ? { status: "ready", goals: withGoal(goals.state.goals, goal) } : goals.state,
    [goals.state, goal],
  );
  const draft = goalToSave(state, choice);

  async function save() {
    setSaving(true);
    setSuccess("");
    setError("");
    try {
      const result = await savePurchaseGoal(itemId, draft);
      setChoice(choiceFor(result.purchaseGoal));
      setDirty(false);
      setSuccess(result.reviewStarted ? "Purchase goal saved. A new AI review was requested." : "Purchase goal saved.");
      onSaved();
      goals.reload();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The purchase goal could not be saved.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="purchase-goal" aria-label="Purchase goal">
      <PurchaseGoalChoice
        id="purchase-goal"
        label="Purchase goal"
        helperText="Sent to the AI review, e.g. what you will use the item for."
        goals={state}
        value={choice}
        onChange={(value) => {
          setChoice(value);
          setDirty(true);
          setSuccess("");
        }}
        disabled={saving}
      />
      {state.status !== "loading" && (
        <div className="purchase-goal-actions">
          <Button kind="secondary" size="sm" onClick={() => void save()} disabled={saving || draft.trim() === goal}>
            Save goal
          </Button>
          {saving && <InlineLoading description="Saving the purchase goal…" />}
        </div>
      )}
      {success && (
        <InlineNotification kind="success" title={success} lowContrast className="detail-notification" onCloseButtonClick={() => setSuccess("")} />
      )}
      {error && (
        <InlineNotification kind="error" title="Could not save purchase goal" subtitle={error} lowContrast className="detail-notification" />
      )}
    </section>
  );
}
