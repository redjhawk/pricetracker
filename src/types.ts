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

export interface TrackedItem {
  id: string;
  title: string | null;
  asin: string;
  marketplace: string;
  url: string;
  thumbnailUrl: string | null;
  status: ItemStatus;
  latestPrice: PriceObservation | null;
  lastThreeDetections: PriceObservation[];
  lastAttempt?: {
    result: "pending" | "success" | "request_error" | "price_not_found" | "unavailable";
    timestamp: string;
    message: string | null;
  } | null;
  nextCheckAt: string | null;
  addedAt: string;
}
