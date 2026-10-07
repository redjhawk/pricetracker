import { InlineLoading, InlineNotification, Tag } from "@carbon/react";
import type { AiReviewContent, AiReviewState } from "../types";

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

const ratingLabels: Record<AiReviewContent["price"]["rating"], [string, "green" | "gray" | "red"]> = {
  good_deal: ["Good deal", "green"],
  fair: ["Fair", "gray"],
  overpriced: ["Overpriced", "red"],
};

const formatCents = (cents: number) => euro.format(cents / 100);

// Amazon reviews only contain a price rating; they run automatically for search items.
export default function AmazonAiReview({ state, currentPriceCents }: { state: AiReviewState; currentPriceCents: number | null }) {
  const { latest, lastAttempt } = state;
  const failed = !state.running && lastAttempt?.status === "failed" ? lastAttempt : null;
  const price = latest?.review?.price;

  return (
    <section className="ai-review" aria-labelledby="ai-review-heading">
      <h2 id="ai-review-heading">AI review</h2>
      <div aria-live="polite">{state.running && <InlineLoading description="AI review in progress" />}</div>
      {!state.tokenConfigured && <p>Configure a Claude token in Settings.</p>}
      {failed && (
        <p>The last AI review failed on {dateTime.format(new Date(failed.completedAt ?? failed.createdAt))}.</p>
      )}
      {!latest && !state.running && !failed && <p className="history-empty">No AI review yet.</p>}
      {latest && price && (
        <div className="ai-review-rating">
          <Tag type={ratingLabels[price.rating][1]} size="md">Price: {ratingLabels[price.rating][0]}</Tag>
          <p>{price.explanation}</p>
          <p className="summary-footnote">
            Reviewed on <time dateTime={latest.createdAt}>{dateTime.format(new Date(latest.createdAt))}</time>
            {latest.priceCents !== null && ` at ${formatCents(latest.priceCents)}`}
          </p>
          {currentPriceCents !== null && latest.priceCents !== null && currentPriceCents !== latest.priceCents && (
            <InlineNotification
              kind="info"
              title={`Price changed since last review (reviewed at ${formatCents(latest.priceCents)})`}
              lowContrast
              hideCloseButton
              className="detail-notification"
            />
          )}
        </div>
      )}
    </section>
  );
}
