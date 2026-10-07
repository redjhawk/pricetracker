import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  InlineNotification,
  StructuredListBody,
  StructuredListCell,
  StructuredListHead,
  StructuredListRow,
  StructuredListWrapper,
  Tag,
  InlineLoading,
} from "@carbon/react";
import { Launch, Renew, TrashCan } from "@carbon/icons-react";
import type { TrackedItem } from "../types";
import AiReview from "./AiReview";
import AmazonAiReview from "./AmazonAiReview";
import PurchaseGoal from "./PurchaseGoal";
import StatusTag from "./StatusTag";

interface Props {
  item: TrackedItem;
  onBack: () => void;
  onDelete: (item: TrackedItem) => void;
  onRefresh: () => void;
  refreshing: boolean;
  refreshError: string | null;
  onAiReviewRefresh: () => void;
  aiReviewRequesting: boolean;
  aiReviewError: string | null;
  onPurchaseGoalSaved: () => void;
}

const euro = new Intl.NumberFormat("de-DE", { style: "currency", currency: "EUR" });
const priceLabel = (amount: number) => amount === 0 ? "Gratuit" : euro.format(amount);
const dateTime = new Intl.DateTimeFormat("en-GB", {
  day: "2-digit",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "UTC",
  timeZoneName: "short",
});

export default function ItemDetail({
  item,
  onBack,
  onDelete,
  onRefresh,
  refreshing,
  refreshError,
  onAiReviewRefresh,
  aiReviewRequesting,
  aiReviewError,
  onPurchaseGoalSaved,
}: Props) {
  const title = item.title ?? `Listing ${item.listingId}`;
  const priceHistory = item.priceHistory ?? item.lastThreeDetections;
  const secondHandHistory = item.secondHandOffer?.priceHistory ?? item.secondHandOffer?.lastThreeDetections ?? [];

  return (
    <section className="detail-page" aria-labelledby="detail-heading">
      <Breadcrumb noTrailingSlash className="detail-breadcrumb">
        <BreadcrumbItem>
          <button className="breadcrumb-button" onClick={onBack}>Tracked items</button>
        </BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>{title}</BreadcrumbItem>
      </Breadcrumb>

      <div className="detail-heading">
        {item.thumbnailUrl && <img className="detail-thumbnail" src={item.thumbnailUrl} alt="" />}
        <div className="detail-heading-copy">
          <h1 id="detail-heading">{title}</h1>
          <div className="detail-tags">
            <Tag type="cool-gray" size="sm">{item.marketplace}</Tag>
            <Tag type="cool-gray" size="sm">{item.listingId}</Tag>
            <StatusTag status={item.status} />
          </div>
        </div>
        <div className="detail-actions">
          {item.tracked && (
            <Button kind="tertiary" size="sm" renderIcon={Renew} onClick={onRefresh} disabled={refreshing}>
              Refresh price
            </Button>
          )}
          <Button
            kind="ghost"
            size="sm"
            renderIcon={Launch}
            href={item.url}
            target="_blank"
            rel="noopener noreferrer"
          >Open on {item.platform === "amazon" ? "Amazon" : "LeBoncoin"}</Button>
          {item.tracked && (
            <Button kind="danger--ghost" size="sm" renderIcon={TrashCan} onClick={() => onDelete(item)}>
              Delete
            </Button>
          )}
        </div>
      </div>

      {refreshing && <InlineLoading description="Refreshing this item's price…" />}
      {refreshError && (
        <InlineNotification kind="error" title="Could not refresh price" subtitle={refreshError} lowContrast className="detail-notification" />
      )}

      {item.status === "stale" && (
        <InlineNotification
          kind="warning"
          title="Price may be stale"
          subtitle="Recent collection attempts have not returned a new price. Retries are continuing."
          lowContrast
          className="detail-notification"
        />
      )}
      {item.status === "retrieval_error" && (
        <InlineNotification
          kind="error"
          title="Retrieval error"
          subtitle="The latest collection attempt failed. The last successful price is preserved. Retries are continuing."
          lowContrast
          className="detail-notification"
        />
      )}
      {item.status === "unavailable" && (
        <InlineNotification
          kind="warning"
          title="Listing unavailable"
          subtitle="The listing appears to be unavailable. The last detected price is shown and checks will continue."
          lowContrast
          className="detail-notification"
        />
      )}
      {item.status === "pending" && (
        <InlineNotification
          kind="info"
          title="First price check in progress"
          subtitle="The item has been added. Its first price detection will appear here when the check completes."
          lowContrast
          className="detail-notification"
        />
      )}

      <div className="detail-summary">
        <article className="summary-tile">
          <p className="summary-label">Latest price</p>
          {item.latestPrice ? (
            <>
              <p className="summary-price">{priceLabel(item.latestPrice.amount)}</p>
              <p className="summary-footnote">
                {item.latestPrice.timestamp === null ? "Old price" : (
                  <>Detected <time dateTime={item.latestPrice.timestamp}>{dateTime.format(new Date(item.latestPrice.timestamp))}</time></>
                )}
              </p>
            </>
          ) : <p className="summary-empty">No successful price detection yet.</p>}
        </article>
        <article className="summary-tile">
          <p className="summary-label">Next scheduled check</p>
          <p className="summary-schedule">
            {item.nextCheckAt ? dateTime.format(new Date(item.nextCheckAt)) : "Not scheduled"}
          </p>
          <p className="summary-footnote">Added {dateTime.format(new Date(item.addedAt))}</p>
        </article>
        {item.secondHandOffer && <article className="summary-tile">
          <p className="summary-label">Amazon second-hand offer</p>
          {item.secondHandOffer.status === "available" && item.secondHandOffer.latestDetection ? (
            <>
              <p className="summary-price">{priceLabel(item.secondHandOffer.latestDetection.amount)}</p>
              <p className="summary-footnote">{item.secondHandOffer.latestDetection.conditionLabel}</p>
              <p className="summary-footnote">
                Detected <time dateTime={item.secondHandOffer.latestDetection.timestamp}>{dateTime.format(new Date(item.secondHandOffer.latestDetection.timestamp))}</time>
              </p>
            </>
          ) : item.secondHandOffer.status === "pending" ? (
            <p className="summary-empty">Checking Amazon offers…</p>
          ) : item.secondHandOffer.status === "not_found" ? (
            <p className="summary-empty">No Amazon-sold second-hand offer found.</p>
          ) : item.secondHandOffer.latestDetection ? (
            <>
              <p className="summary-price">{priceLabel(item.secondHandOffer.latestDetection.amount)}</p>
              <p className="summary-footnote">Last detected: {item.secondHandOffer.latestDetection.conditionLabel}</p>
              <p className="summary-footnote">The latest offer check could not be completed.</p>
            </>
          ) : <p className="summary-empty">Amazon offer check unavailable.</p>}
        </article>}
      </div>

      {item.platform === "leboncoin" && (
        <PurchaseGoal itemId={item.id} goal={item.purchaseGoal} onSaved={onPurchaseGoalSaved} />
      )}

      {item.platform === "amazon" && item.aiReview && (
        <AmazonAiReview
          state={item.aiReview}
          currentPriceCents={item.latestPrice ? Math.round(item.latestPrice.amount * 100) : null}
        />
      )}

      {item.platform === "leboncoin" && item.aiReview && (
        <AiReview
          state={item.aiReview}
          currentPriceCents={item.latestPrice ? Math.round(item.latestPrice.amount * 100) : null}
          onRefresh={onAiReviewRefresh}
          requesting={aiReviewRequesting}
          requestError={aiReviewError}
        />
      )}

      <div className="history-heading">
        <div>
          <h2>Price history</h2>
          <p>All successful item-price detections, newest first</p>
        </div>
      </div>

      {priceHistory.length === 0 ? (
        <p className="history-empty">No successful detections yet.</p>
      ) : (
        <StructuredListWrapper className="detections-list">
          <StructuredListHead>
            <StructuredListRow head>
              <StructuredListCell head>#</StructuredListCell>
              <StructuredListCell head>Detected price</StructuredListCell>
              <StructuredListCell head>Date and time</StructuredListCell>
            </StructuredListRow>
          </StructuredListHead>
          <StructuredListBody>
            {priceHistory.map((observation, index) => (
              <StructuredListRow key={`${observation.timestamp}-${index}`}>
                <StructuredListCell>{index + 1}</StructuredListCell>
                <StructuredListCell className="detection-price">{priceLabel(observation.amount)}</StructuredListCell>
                <StructuredListCell>
                  {observation.timestamp === null ? "Old price" : (
                    <time dateTime={observation.timestamp}>{dateTime.format(new Date(observation.timestamp))}</time>
                  )}
                </StructuredListCell>
              </StructuredListRow>
            ))}
          </StructuredListBody>
        </StructuredListWrapper>
      )}

      {item.secondHandOffer && <>
      <div className="history-heading second-hand-history-heading">
        <div>
          <h2>Amazon second-hand price history</h2>
          <p>All successful detections of offers sold by Amazon, newest first</p>
        </div>
      </div>

      {secondHandHistory.length === 0 ? (
        <p className="history-empty">No Amazon-sold second-hand offer has been detected yet.</p>
      ) : (
        <StructuredListWrapper className="detections-list">
          <StructuredListHead>
            <StructuredListRow head>
              <StructuredListCell head>#</StructuredListCell>
              <StructuredListCell head>Offer price</StructuredListCell>
              <StructuredListCell head>Condition</StructuredListCell>
              <StructuredListCell head>Date and time</StructuredListCell>
            </StructuredListRow>
          </StructuredListHead>
          <StructuredListBody>
            {secondHandHistory.map((detection, index) => (
              <StructuredListRow key={`${detection.timestamp}-${index}`}>
                <StructuredListCell>{index + 1}</StructuredListCell>
                <StructuredListCell className="detection-price">{euro.format(detection.amount)}</StructuredListCell>
                <StructuredListCell>{detection.conditionLabel}</StructuredListCell>
                <StructuredListCell>
                  <time dateTime={detection.timestamp}>{dateTime.format(new Date(detection.timestamp))}</time>
                </StructuredListCell>
              </StructuredListRow>
            ))}
          </StructuredListBody>
        </StructuredListWrapper>
      )}
      </>}
    </section>
  );
}
