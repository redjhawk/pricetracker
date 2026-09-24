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
} from "@carbon/react";
import { Launch, TrashCan } from "@carbon/icons-react";
import type { TrackedItem } from "../types";
import StatusTag from "./StatusTag";

interface Props {
  item: TrackedItem;
  onBack: () => void;
  onDelete: (item: TrackedItem) => void;
}

const euro = new Intl.NumberFormat("de-DE", { style: "currency", currency: "EUR" });
const dateTime = new Intl.DateTimeFormat("en-GB", {
  day: "2-digit",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "UTC",
  timeZoneName: "short",
});

export default function ItemDetail({ item, onBack, onDelete }: Props) {
  const title = item.title ?? `Listing ${item.asin}`;

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
            <Tag type="cool-gray" size="sm">{item.asin}</Tag>
            <StatusTag status={item.status} />
          </div>
        </div>
        <div className="detail-actions">
          <Button
            kind="ghost"
            size="sm"
            renderIcon={Launch}
            href={item.url}
            target="_blank"
            rel="noopener noreferrer"
          >
            Open on Amazon
          </Button>
          <Button kind="danger--ghost" size="sm" renderIcon={TrashCan} onClick={() => onDelete(item)}>
            Delete
          </Button>
        </div>
      </div>

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
              <p className="summary-price">{euro.format(item.latestPrice.amount)}</p>
              <p className="summary-footnote">
                Detected <time dateTime={item.latestPrice.timestamp}>{dateTime.format(new Date(item.latestPrice.timestamp))}</time>
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
      </div>

      <div className="history-heading">
        <div>
          <h2>Recent detections</h2>
          <p>Most recent successful item-price checks</p>
        </div>
      </div>

      {item.lastThreeDetections.length === 0 ? (
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
            {item.lastThreeDetections.slice(0, 3).map((observation, index) => (
              <StructuredListRow key={`${observation.timestamp}-${index}`}>
                <StructuredListCell>{index + 1}</StructuredListCell>
                <StructuredListCell className="detection-price">{euro.format(observation.amount)}</StructuredListCell>
                <StructuredListCell>
                  <time dateTime={observation.timestamp}>{dateTime.format(new Date(observation.timestamp))}</time>
                </StructuredListCell>
              </StructuredListRow>
            ))}
          </StructuredListBody>
        </StructuredListWrapper>
      )}
    </section>
  );
}
