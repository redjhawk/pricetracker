import { Fragment, useCallback, useEffect, useState, type FormEvent } from "react";
import {
  Button,
  InlineLoading,
  InlineNotification,
  Modal,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableHeader,
  TableRow,
  Tag,
  TextInput,
} from "@carbon/react";
import type { AmazonRequests, AmazonSearch } from "../types";
import { addSearch, deleteSearch, listSearches, refreshSearch } from "../api/searches";

const dateTime = new Intl.DateTimeFormat("en-GB", {
  day: "2-digit",
  month: "short",
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "Europe/Paris",
});
const windowTime = new Intl.DateTimeFormat("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: "Europe/Paris" });

export const formatSearchDate = (value: string) => dateTime.format(new Date(value));

const stateTags: Record<AmazonSearch["state"], "gray" | "blue" | "green" | "red"> = {
  waiting: "gray",
  running: "blue",
  done: "green",
  stopped: "red",
};

export function searchStateText(search: AmazonSearch) {
  if (search.state === "waiting") {
    return search.waitingUntil ? `Waiting for next window (${windowTime.format(new Date(search.waitingUntil))})` : "Waiting";
  }
  return { running: "Running", done: "Done", stopped: "Stopped" }[search.state];
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "The server could not be reached.";
}

export default function AmazonSearchesPage({ onOpenSearch }: { onOpenSearch: (search: AmazonSearch) => void }) {
  const [searches, setSearches] = useState<AmazonSearch[]>([]);
  const [amazonRequests, setAmazonRequests] = useState<AmazonRequests | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [url, setUrl] = useState("");
  const [adding, setAdding] = useState(false);
  const [addError, setAddError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AmazonSearch | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const response = await listSearches();
      setSearches(response.searches);
      setAmazonRequests(response.amazonRequests);
      setError(null);
    } catch (loadError) {
      setError(errorMessage(loadError));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
    const poll = window.setInterval(() => void load(), 30_000);
    return () => window.clearInterval(poll);
  }, [load]);

  async function handleAdd(event: FormEvent) {
    event.preventDefault();
    setAdding(true);
    setAddError(null);
    try {
      await addSearch(url);
      setUrl("");
      await load();
    } catch (addFailure) {
      setAddError(errorMessage(addFailure));
    } finally {
      setAdding(false);
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    setActionError(null);
    try {
      await deleteSearch(deleteTarget.id);
      await load();
    } catch (deleteError) {
      setActionError(errorMessage(deleteError));
    } finally {
      setDeleteTarget(null);
    }
  }

  async function handleRefresh(search: AmazonSearch) {
    setActionError(null);
    try {
      await refreshSearch(search.id);
      await load();
    } catch (refreshError) {
      setActionError(errorMessage(refreshError));
    }
  }

  const stoppedAt = amazonRequests?.stoppedAt ?? null;

  return (
    <div className="amazon-searches">
      <form className="tracked-table-search" onSubmit={(event) => void handleAdd(event)}>
        <TextInput
          id="amazon-search-url"
          labelText="Amazon search URL"
          placeholder="https://www.amazon.fr/…"
          value={url}
          onChange={(event) => setUrl(event.target.value)}
          invalid={addError !== null}
          invalidText={addError ?? ""}
        />
        <Button type="submit" size="md" disabled={adding || url.trim() === ""}>Add search</Button>
      </form>

      {actionError && (
        <InlineNotification kind="error" title="Action failed" subtitle={actionError} lowContrast className="detail-notification" />
      )}

      {loading ? (
        <div className="empty-state"><InlineLoading description="Loading Amazon searches…" /></div>
      ) : error ? (
        <div className="page-feedback">
          <InlineNotification kind="error" title="Could not load Amazon searches" subtitle={error} lowContrast className="detail-notification" />
          <Button kind="tertiary" size="sm" onClick={() => void load()}>Retry</Button>
        </div>
      ) : searches.length === 0 ? (
        <div className="empty-state"><p>No Amazon search yet. Add a results URL from Amazon.</p></div>
      ) : (
        <TableContainer className="tracked-table-container">
          <Table size="lg" aria-label="Amazon searches">
            <TableHead>
              <TableRow>
                <TableHeader>Search</TableHeader>
                <TableHeader>Added</TableHeader>
                <TableHeader>Items</TableHeader>
                <TableHeader>State</TableHeader>
                <TableHeader aria-label="Actions" />
              </TableRow>
            </TableHead>
            <TableBody>
              {searches.map((search) => {
                const showStop = stoppedAt !== null && (search.state === "stopped" || search.state === "waiting");
                return (
                  <Fragment key={search.id}>
                    <TableRow>
                      <TableCell>
                        <Button kind="ghost" className="item-title" title={search.url} onClick={() => onOpenSearch(search)}>
                          {search.label}
                        </Button>
                      </TableCell>
                      <TableCell>{formatSearchDate(search.addedAt)}</TableCell>
                      <TableCell>{search.itemCount}</TableCell>
                      <TableCell>
                        <Tag type={stateTags[search.state]} size="sm">{searchStateText(search)}</Tag>
                        {search.lastError && <p className="muted">{search.lastError.message}</p>}
                      </TableCell>
                      <TableCell>
                        <div className="row-actions">
                          {showStop && amazonRequests?.stopped && (
                            <Button kind="tertiary" size="sm" onClick={() => void handleRefresh(search)}>Refresh</Button>
                          )}
                          <Button kind="danger--ghost" size="sm" onClick={() => setDeleteTarget(search)}>Delete</Button>
                        </div>
                      </TableCell>
                    </TableRow>
                    {showStop && (
                      <TableRow>
                        <TableCell colSpan={5}>
                          <InlineNotification
                            kind="warning"
                            title={`Amazon requests were stopped after repeated failures on ${formatSearchDate(stoppedAt)}.`}
                            lowContrast
                            hideCloseButton
                          />
                        </TableCell>
                      </TableRow>
                    )}
                  </Fragment>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Modal
        open={deleteTarget !== null}
        danger
        modalHeading={`Delete search ${deleteTarget?.label ?? ""}?`}
        primaryButtonText="Delete"
        secondaryButtonText="Cancel"
        onRequestClose={() => setDeleteTarget(null)}
        onRequestSubmit={() => void handleDelete()}
      >
        <p>Delete search? Items that are not tracked or in another search are deleted with their price history and reviews.</p>
      </Modal>
    </div>
  );
}
