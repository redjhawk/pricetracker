import { Accordion, AccordionItem, Button, InlineLoading, InlineNotification, Tag } from "@carbon/react";
import { Renew } from "@carbon/icons-react";
import type { AiReview, AiReviewState, AmazonAiReviewContent } from "../types";
import { dateTime } from "../dateTime";

interface Props {
  state: AiReviewState<AmazonAiReviewContent>;
  onRefresh: () => void;
  requesting: boolean;
  requestError: string | null;
}

type TagType = "green" | "gray" | "red";

const euro = new Intl.NumberFormat("de-DE", { style: "currency", currency: "EUR" });
const formatDate = (value: string) => dateTime.format(new Date(value));
const formatCents = (cents: number | null) => cents === null ? "unknown" : euro.format(cents / 100);

const priceLabels: Record<AmazonAiReviewContent["price"]["rating"], [string, TagType]> = {
  good_deal: ["Good deal", "green"],
  fair: ["Fair", "gray"],
  overpriced: ["Overpriced", "red"],
};

const recommendationLabels: Record<AmazonAiReviewContent["recommendation"]["rating"], [string, TagType]> = {
  buy: ["Buy", "green"],
  wait: ["Wait for a lower price", "gray"],
  avoid: ["Avoid", "red"],
};

const unavailableMessages = {
  stopped: "Amazon checks are stopped: restart the item with Refresh price to request an AI review.",
  no_current_price: "No current price: an AI review needs a current Amazon price.",
};

function Rating({ name, label, explanation }: { name: string; label: [string, TagType]; explanation: string }) {
  return (
    <div className="ai-review-rating">
      <Tag type={label[1]} size="md">{name}: {label[0]}</Tag>
      <p>{explanation}</p>
    </div>
  );
}

function ReviewBody({ review }: { review: AmazonAiReviewContent }) {
  return (
    <div className="ai-review-body">
      <Rating name="Price" label={priceLabels[review.price.rating]} explanation={review.price.explanation} />
      <dl className="ai-review-details">
        <dt>Fair price range</dt>
        <dd>{formatCents(review.fairPrice.minCents)} – {formatCents(review.fairPrice.maxCents)}</dd>
      </dl>
      <Rating
        name="Overall recommendation"
        label={recommendationLabels[review.recommendation.rating]}
        explanation={review.recommendation.explanation}
      />
    </div>
  );
}

function historyTitle(attempt: AiReview<AmazonAiReviewContent>) {
  const recommendation = attempt.review ? recommendationLabels[attempt.review.recommendation.rating][0] : "";
  return `${formatDate(attempt.createdAt)} — Based on price ${formatCents(attempt.priceCents)} — ${recommendation}`;
}

export default function AmazonAiReview({ state, onRefresh, requesting, requestError }: Props) {
  const inProgress = state.running || requesting;
  const { latest, lastAttempt, unavailableReason } = state;
  const canRequest = unavailableReason === null;
  const lastFailed = !inProgress && lastAttempt?.status === "failed" ? lastAttempt : null;
  const previous = state.history.filter((attempt) => attempt.id !== latest?.id);
  const notice = unavailableReason ? unavailableMessages[unavailableReason]
    : !state.tokenConfigured ? "Configure a Claude token in Settings" : null;

  return (
    <section className="ai-review" aria-labelledby="ai-review-heading">
      <div className="history-heading">
        <h2 id="ai-review-heading">AI review</h2>
        {canRequest && (
          <Button
            kind="tertiary"
            size="sm"
            renderIcon={Renew}
            onClick={onRefresh}
            disabled={!state.tokenConfigured || inProgress}
          >
            Refresh AI review
          </Button>
        )}
      </div>

      {notice && (
        <InlineNotification kind="info" title={notice} lowContrast hideCloseButton className="detail-notification" />
      )}
      <div aria-live="polite">
        {inProgress && <InlineLoading description="AI review in progress…" />}
      </div>
      {requestError && (
        <InlineNotification
          kind="error"
          title="Could not start the AI review"
          subtitle={requestError}
          lowContrast
          className="detail-notification"
        />
      )}
      {lastFailed && (
        <InlineNotification
          kind="error"
          title={`The last AI review failed on ${formatDate(lastFailed.completedAt ?? lastFailed.createdAt)}`}
          subtitle={lastFailed.errorMessage ?? ""}
          lowContrast
          hideCloseButton
          className="detail-notification"
        />
      )}
      {!latest && !inProgress && (
        <p className="history-empty">
          No AI review yet.{canRequest && state.tokenConfigured && " Use Refresh AI review to request one."}
        </p>
      )}
      {latest?.review && (
        <>
          <p className="summary-footnote">
            Reviewed on <time dateTime={latest.createdAt}>{formatDate(latest.createdAt)}</time> · Based on price {formatCents(latest.priceCents)}
          </p>
          <ReviewBody review={latest.review} />
        </>
      )}
      {previous.length > 0 && (
        <Accordion className="ai-review-history">
          {previous.map((attempt) => (
            <AccordionItem key={attempt.id} title={historyTitle(attempt)}>
              {attempt.review && <ReviewBody review={attempt.review} />}
            </AccordionItem>
          ))}
        </Accordion>
      )}
    </section>
  );
}
