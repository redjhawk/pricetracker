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
  secondHandOffer: SecondHandOffer | null;
  lastAttempt?: {
    result: "pending" | "success" | "request_error" | "price_not_found" | "unavailable";
    timestamp: string;
    message: string | null;
  } | null;
  nextCheckAt: string | null;
  addedAt: string;
}
