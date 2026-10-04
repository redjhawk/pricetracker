import { Accordion, AccordionItem, Button, InlineLoading, InlineNotification, Tag } from "@carbon/react";
import { Renew } from "@carbon/icons-react";
import type { AiReview as AiReviewAttempt, AiReviewContent, AiReviewState } from "../types";

interface Props {
  state: AiReviewState;
  currentPriceCents: number | null;
  onRefresh: () => void;
  requesting: boolean;
  requestError: string | null;
}

type TagType = "green" | "gray" | "red";

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

const formatDate = (value: string) => dateTime.format(new Date(value));
const formatCents = (cents: number) => euro.format(cents / 100);

function reviewedPrice(cents: number | null) {
  if (cents === null) return "unknown";
  return cents === 0 ? "Free" : formatCents(cents);
}

const priceLabels: Record<AiReviewContent["price"]["rating"], [string, TagType]> = {
  good_deal: ["Good deal", "green"],
  fair: ["Fair", "gray"],
  overpriced: ["Overpriced", "red"],
};

const conditionLabels: Record<NonNullable<AiReviewContent["condition"]["rating"]>, [string, TagType]> = {
  excellent: ["Excellent", "green"],
  good: ["Good", "green"],
  fair: ["Fair", "gray"],
  poor: ["Poor", "red"],
};

const recommendationLabels: Record<AiReviewContent["recommendation"]["rating"], [string, TagType]> = {
  buy: ["Buy", "green"],
  negotiate: ["Negotiate", "gray"],
  avoid: ["Avoid", "red"],
};

function conditionLabel(rating: AiReviewContent["condition"]["rating"]): [string, TagType] {
  return rating ? conditionLabels[rating] : ["Not assessable", "gray"];
}

function descriptionVsPhotosLabel(matches: boolean | null) {
  if (matches === null) return "Cannot be compared";
  return matches ? "Matches the photos" : "Does not match the photos";
}

function TextList({ values, empty }: { values: string[]; empty?: string }) {
  if (values.length === 0) return empty ? <p>{empty}</p> : null;
  return <ul className="ai-review-list">{values.map((value, index) => <li key={index}>{value}</li>)}</ul>;
}

function Rating({ name, label, explanation }: { name: string; label: [string, TagType]; explanation: string }) {
  return (
    <div className="ai-review-rating">
      <Tag type={label[1]} size="md">{name}: {label[0]}</Tag>
      <p>{explanation}</p>
    </div>
  );
}

function ReviewBody({ review }: { review: AiReviewContent }) {
  return (
    <div className="ai-review-body">
      <Rating name="Price" label={priceLabels[review.price.rating]} explanation={review.price.explanation} />
      <Rating name="Condition" label={conditionLabel(review.condition.rating)} explanation={review.condition.explanation} />
      <Rating
        name="Recommendation"
        label={recommendationLabels[review.recommendation.rating]}
        explanation={review.recommendation.explanation}
      />
      <dl className="ai-review-details">
        <dt>Fair price range</dt>
        <dd>{formatCents(review.fairPrice.minCents)} – {formatCents(review.fairPrice.maxCents)}</dd>
        <dt>Suggested offer</dt>
        <dd>{formatCents(review.fairPrice.suggestedOfferCents)}</dd>
        <dt>Risk signs</dt>
        <dd><TextList values={review.risks} empty="No scam or risk signs were found." /></dd>
        <dt>Missing accessories or information</dt>
        <dd><TextList values={review.missingInformation} empty="Nothing notable." /></dd>
        <dt>Questions to ask the seller</dt>
        <dd><TextList values={review.sellerQuestions} empty="None." /></dd>
        <dt>Description vs photos</dt>
        <dd>
          <p>{descriptionVsPhotosLabel(review.descriptionVsPhotos.matches)}</p>
          <p>{review.descriptionVsPhotos.explanation}</p>
          <TextList values={review.descriptionVsPhotos.mismatches} />
        </dd>
      </dl>
    </div>
  );
}

function historyTitle(attempt: AiReviewAttempt) {
  const recommendation = attempt.review ? recommendationLabels[attempt.review.recommendation.rating][0] : "";
  return `${formatDate(attempt.createdAt)} — ${reviewedPrice(attempt.priceCents)} — ${recommendation}`;
}

export default function AiReview({ state, currentPriceCents, onRefresh, requesting, requestError }: Props) {
  const inProgress = state.running || requesting;
  const { latest, lastAttempt } = state;
  const lastFailed = !inProgress && lastAttempt?.status === "failed" ? lastAttempt : null;
  const previous = state.history.filter((attempt) => attempt.id !== latest?.id);

  return (
    <section className="ai-review" aria-labelledby="ai-review-heading">
      <div className="history-heading">
        <h2 id="ai-review-heading">AI review</h2>
        <Button
          kind="tertiary"
          size="sm"
          renderIcon={Renew}
          onClick={onRefresh}
          disabled={!state.tokenConfigured || inProgress}
        >
          Refresh AI review
        </Button>
      </div>

      {!state.tokenConfigured && (
        <InlineNotification
          kind="info"
          title="Configure a Claude token in Settings"
          lowContrast
          hideCloseButton
          className="detail-notification"
        />
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
          No AI review yet.{state.tokenConfigured && " Use Refresh AI review to request one."}
        </p>
      )}
      {latest?.review && (
        <>
          <p className="summary-footnote">
            Reviewed on <time dateTime={latest.createdAt}>{formatDate(latest.createdAt)}</time> at {reviewedPrice(latest.priceCents)}
          </p>
          {currentPriceCents !== null && latest.priceCents !== null && latest.priceCents !== currentPriceCents && (
            <InlineNotification
              kind="warning"
              subtitle={`This review was made at an older price (${reviewedPrice(latest.priceCents)}); the current price is ${reviewedPrice(currentPriceCents)}.`}
              lowContrast
              hideCloseButton
              className="detail-notification"
            />
          )}
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
