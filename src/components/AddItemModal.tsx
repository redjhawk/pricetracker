import { useState } from "react";
import { InlineLoading, InlineNotification, Modal, TextInput } from "@carbon/react";

interface Props {
  open: boolean;
  onClose: () => void;
  onAdded: (url: string) => Promise<void>;
}

export default function AddItemModal({ open, onClose, onAdded }: Props) {
  const [url, setUrl] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  function close() {
    if (submitting) return;
    setUrl("");
    setError("");
    onClose();
  }

  async function submit() {
    if (!url.trim()) {
      setError("Enter an Amazon listing URL.");
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      await onAdded(url.trim());
      setUrl("");
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
      <p className="modal-intro">Paste the URL of an Amazon product listing to add it to your tracked items.</p>
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
        labelText="Amazon listing URL"
        placeholder="https://www.amazon.de/dp/B0XXXXXXXX"
        value={url}
        onChange={(event) => {
          setUrl(event.target.value);
          setError("");
        }}
        disabled={submitting}
        invalid={Boolean(error)}
        invalidText={error}
        helperText="Supported euro marketplaces: amazon.de, .fr, .es, .it, .nl, .be"
      />
    </Modal>
  );
}
