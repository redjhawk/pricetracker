export type ItemStatus =
  | "active"
  | "stale"
  | "retrieval_error"
  | "unavailable"
  | "pending";

export interface PriceObservation {
  amount: number;
  currency: "EUR";
  timestamp: string;
}

export type SecondHandOfferStatus = "pending" | "available" | "not_found" | "check_error";

export interface SecondHandOfferDetection extends PriceObservation {
  condition: "like_new" | "very_good" | "good" | "acceptable" | "unknown";
  conditionLabel: string;
}

export interface SecondHandOffer {
  status: SecondHandOfferStatus;
  latestDetection: SecondHandOfferDetection | null;
  lastThreeDetections: SecondHandOfferDetection[];
  priceHistory?: SecondHandOfferDetection[];
  lastCheckedAt: string | null;
}

export interface TrackedItem {
  id: string;
  title: string | null;
  platform: "amazon" | "leboncoin";
  listingId: string;
  asin: string | null;
  marketplace: string;
  url: string;
  thumbnailUrl: string | null;
  status: ItemStatus;
  latestPrice: PriceObservation | null;
  lastThreeDetections: PriceObservation[];
  priceHistory?: PriceObservation[];
  secondHandOffer: SecondHandOffer | null;
  lastAttempt?: {
    result: "pending" | "success" | "request_error" | "price_not_found" | "unavailable";
    timestamp: string;
    message: string | null;
  } | null;
  nextCheckAt: string | null;
  addedAt: string;
  purchaseGoal: string;
  aiReview?: AiReviewState | null;
}

export interface AiReviewContent {
  price: { rating: "good_deal" | "fair" | "overpriced"; explanation: string };
  condition: { rating: "excellent" | "good" | "fair" | "poor" | null; explanation: string };
  recommendation: { rating: "buy" | "negotiate" | "avoid"; explanation: string };
  fairPrice: { minCents: number; maxCents: number; suggestedOfferCents: number };
  risks: string[];
  missingInformation: string[];
  sellerQuestions: string[];
  descriptionVsPhotos: { matches: boolean | null; explanation: string; mismatches: string[] };
}

export interface AiReview {
  id: number;
  status: "pending" | "succeeded" | "failed";
  priceCents: number | null;
  createdAt: string;
  completedAt: string | null;
  review: AiReviewContent | null;
  errorMessage: string | null;
}

export interface AiReviewState {
  tokenConfigured: boolean;
  running: boolean;
  lastAttempt: AiReview | null;
  latest: AiReview | null;
  history: AiReview[];
}
