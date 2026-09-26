import { useState } from "react";
import { InlineNotification, Modal } from "@carbon/react";
import type { TrackedItem } from "../types";

interface Props {
  item: TrackedItem | null;
  onClose: () => void;
  onConfirm: (id: string) => Promise<void>;
}

export default function DeleteModal({ item, onClose, onConfirm }: Props) {
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  if (!item) return null;

  const label = item.title ?? `Listing ${item.listingId}`;
  const trackedItem = item;

  async function submit() {
    setSubmitting(true);
    setError("");
    try {
      await onConfirm(trackedItem.id);
      onClose();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The item could not be deleted.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      open
      danger
      modalHeading="Delete tracked item"
      primaryButtonText={submitting ? "Deleting…" : "Delete"}
      primaryButtonDisabled={submitting}
      secondaryButtonText="Cancel"
      onRequestSubmit={() => void submit()}
      onRequestClose={onClose}
      size="sm"
    >
      {error && (
        <InlineNotification
          kind="error"
          title="Could not delete item"
          subtitle={error}
          lowContrast
          className="modal-notification"
        />
      )}
      <p className="delete-copy">
        Permanently delete <strong>{label}</strong> and all of its price history? This action cannot be undone.
      </p>
    </Modal>
  );
}
