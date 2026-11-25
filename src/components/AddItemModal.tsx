import { useState } from "react";
import { InlineLoading, InlineNotification, Modal, TextInput } from "@carbon/react";
import { goalToSave, noGoalChoice, usePurchaseGoals, type GoalChoice } from "../purchaseGoalChoice";
import PurchaseGoalChoice from "./PurchaseGoalChoice";

interface Props {
  open: boolean;
  onClose: () => void;
  onAdded: (url: string, purchaseGoal: string) => Promise<void>;
}

export default function AddItemModal({ open, onClose, onAdded }: Props) {
  const [url, setUrl] = useState("");
  const goals = usePurchaseGoals(open);
  const [choice, setChoice] = useState<GoalChoice>(noGoalChoice);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  function close() {
    if (submitting) return;
    setUrl("");
    setChoice(noGoalChoice);
    setError("");
    onClose();
  }

  async function submit() {
    if (!url.trim()) {
      setError("Enter an Amazon or LeBoncoin listing URL.");
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      await onAdded(url.trim(), goalToSave(goals.state, choice));
      setUrl("");
      setChoice(noGoalChoice);
      onClose();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The item could not be added.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      open={open}
      modalHeading="Add tracked item"
      primaryButtonText={submitting ? "Adding item…" : "Add item"}
      secondaryButtonText="Cancel"
      primaryButtonDisabled={submitting}
      onRequestSubmit={() => void submit()}
      onRequestClose={close}
      size="sm"
    >
      <p className="modal-intro">Paste the URL of an Amazon or LeBoncoin listing to add it to your tracked items.</p>
      {submitting && <InlineLoading description="Adding the item and starting its first price check…" />}
      {error && (
        <InlineNotification
          kind="error"
          title="Could not add item"
          subtitle={error}
          lowContrast
          className="modal-notification"
          onCloseButtonClick={() => setError("")}
        />
      )}
      <TextInput
        id="add-item-url"
        labelText="Listing URL"
        placeholder="https://www.amazon.fr/dp/B0XXXXXXXX or https://www.leboncoin.fr/ad/..."
        value={url}
        onChange={(event) => {
          setUrl(event.target.value);
          setError("");
        }}
        disabled={submitting}
        invalid={Boolean(error)}
        invalidText={error}
        helperText="Supported: European Amazon marketplaces and French LeBoncoin listings"
      />
      <PurchaseGoalChoice
        id="add-item-purchase-goal"
        label="Purchase goal (optional)"
        helperText="Sent to the AI review, e.g. what you will use the item for."
        goals={goals.state}
        value={choice}
        onChange={setChoice}
        disabled={submitting}
      />
    </Modal>
  );
}
