import { useCallback, useEffect, useState } from "react";
import {
  Button,
  InlineLoading,
  InlineNotification,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableHeader,
  TableRow,
  Tag,
} from "@carbon/react";
import { ArrowLeft, Launch } from "@carbon/icons-react";
import type { AmazonRequests, AmazonSearch, SearchItem } from "../types";
import { getSearch, trackSearchItem } from "../api/searches";
import { formatSearchDate, formatWindowTime, searchStateText } from "./AmazonSearchesPage";
import StatusTag from "./StatusTag";

interface Props {
  searchId: string;
  onBack: () => void;
  onViewDetail: (item: SearchItem) => void;
  onTracked: () => void;
}

const euro = new Intl.NumberFormat("de-DE", { style: "currency", currency: "EUR" });
const ratingLabels = { good_deal: "Good deal", fair: "Fair", overpriced: "Overpriced" };

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "The server could not be reached.";
}

function reviewText(item: SearchItem) {
  const summary = item.aiReviewSummary;
  if (summary.status === "pending") return "Pending";
  if (summary.status === "failed") return "Failed";
  if (summary.status === "no_token") return "No token";
  if (!summary.priceRating) return "—";
  const currentCents = item.latestPrice ? Math.round(item.latestPrice.amount * 100) : null;
  const changed = currentCents !== null && summary.priceCents !== null && currentCents !== summary.priceCents;
  return ratingLabels[summary.priceRating] + (changed ? " · Price changed" : "");
}

export default function AmazonSearchItems({ searchId, onBack, onViewDetail, onTracked }: Props) {
  const [search, setSearch] = useState<AmazonSearch | null>(null);
  const [amazonRequests, setAmazonRequests] = useState<AmazonRequests | null>(null);
  const [items, setItems] = useState<SearchItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [trackError, setTrackError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const response = await getSearch(searchId);
      setSearch(response.search);
      setAmazonRequests(response.amazonRequests);
      setItems(response.items);
      setError(null);
    } catch (loadError) {
      setError(errorMessage(loadError));
    } finally {
      setLoading(false);
    }
  }, [searchId]);

  useEffect(() => {
    setLoading(true);
    void load();
    const poll = window.setInterval(() => void load(), 30_000);
    return () => window.clearInterval(poll);
  }, [load]);

  async function handleTrack(item: SearchItem) {
    setTrackError(null);
    try {
      await trackSearchItem(searchId, item.id);
      onTracked();
      await load();
    } catch (trackFailure) {
      setTrackError(`${item.title ?? item.listingId}: ${errorMessage(trackFailure)}`);
    }
  }

  return (
    <section className="tracked-page" aria-labelledby="search-heading">
      <Button kind="ghost" size="sm" renderIcon={ArrowLeft} onClick={onBack}>Back to Amazon searches</Button>
      {loading ? (
        <InlineLoading description="Loading search items…" />
      ) : error || !search ? (
        <div className="page-feedback">
          <InlineNotification kind="error" title="Could not load the search" subtitle={error ?? ""} lowContrast className="detail-notification" />
          <Button kind="tertiary" size="sm" onClick={() => void load()}>Retry</Button>
        </div>
      ) : (
        <>
          <div className="page-heading">
            <div>
              <h1 id="search-heading" title={search.url}>{search.label}</h1>
              <p className="page-description">{searchStateText(search)} · {search.itemCount} items</p>
            </div>
          </div>
          {amazonRequests?.stoppedAt && (
            <InlineNotification
              kind="warning"
              title={`Amazon requests were stopped after repeated failures on ${formatSearchDate(amazonRequests.stoppedAt)}.`}
              lowContrast
              hideCloseButton
              className="detail-notification"
            />
          )}
          {search.lastError && (
            <InlineNotification kind="error" title={search.lastError.message} lowContrast hideCloseButton className="detail-notification" />
          )}
          {trackError && (
            <InlineNotification kind="error" title="Could not move the item" subtitle={trackError} lowContrast className="detail-notification" />
          )}
          {items.length === 0 ? (
            <div className="empty-state">
              <p>
                {search.waitingUntil
                  ? `Waiting for the first retrieval at ${formatWindowTime(search.waitingUntil)}`
                  : "No items retrieved yet."}
              </p>
            </div>
          ) : (
            <TableContainer className="tracked-table-container">
              <Table size="lg" aria-label="Search items">
                <TableHead>
                  <TableRow>
                    <TableHeader>Item</TableHeader>
                    <TableHeader>Prices</TableHeader>
                    <TableHeader>Status</TableHeader>
                    <TableHeader>AI review</TableHeader>
                    <TableHeader aria-label="Actions" />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {items.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell>
                        <div className="item-identity">
                          <Button kind="ghost" className="item-title" onClick={() => onViewDetail(item)}>
                            {item.title ?? "Title unavailable"}
                          </Button>
                          <span className="item-asin">{item.marketplace} · {item.listingId}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        {item.lastThreeDetections.length ? (
                          <ul className="price-history-list">
                            {item.lastThreeDetections.slice(0, 3).map((observation, index) => (
                              <li key={`${observation.timestamp}-${index}`}>
                                <span>{euro.format(observation.amount)}</span>
                                {observation.timestamp && (
                                  <time dateTime={observation.timestamp}>{formatSearchDate(observation.timestamp)}</time>
                                )}
                              </li>
                            ))}
                          </ul>
                        ) : <span className="muted">Awaiting first price</span>}
                      </TableCell>
                      <TableCell><StatusTag status={item.status} /></TableCell>
                      <TableCell>{reviewText(item)}</TableCell>
                      <TableCell>
                        <div className="row-actions">
                          <Button kind="ghost" size="sm" onClick={() => onViewDetail(item)}>Details</Button>
                          <Button
                            kind="ghost"
                            size="sm"
                            hasIconOnly
                            renderIcon={Launch}
                            iconDescription="Open Amazon listing in a new tab"
                            href={item.url}
                            target="_blank"
                            rel="noopener noreferrer"
                          />
                          {item.tracked ? (
                            <Tag type="green" size="sm">Tracked</Tag>
                          ) : (
                            <Button kind="ghost" size="sm" onClick={() => void handleTrack(item)}>Move to tracked Amazon items</Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </>
      )}
    </section>
  );
}
