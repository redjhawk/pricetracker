import type { ItemStatus, PriceObservation, SecondHandOffer, SecondHandOfferDetection, TrackedItem } from "../types";

interface ApiPriceObservation {
  amountCents: number;
  currency: "EUR";
  timestamp: string;
}

interface ApiTrackedItem {
  id: string;
  title: string | null;
  platform: TrackedItem["platform"];
  listingId: string;
  asin: string | null;
  marketplace: string;
  url: string;
  thumbnailUrl: string | null;
  status: ItemStatus;
  latestPrice: ApiPriceObservation | null;
  lastThreeDetections: ApiPriceObservation[];
  secondHandOffer: {
    status: SecondHandOffer["status"];
    latestDetection: (ApiPriceObservation & { condition: SecondHandOfferDetection["condition"]; conditionLabel: string }) | null;
    lastThreeDetections: (ApiPriceObservation & { condition: SecondHandOfferDetection["condition"]; conditionLabel: string })[];
    lastCheckedAt: string | null;
  } | null;
  nextCheckAt: string | null;
  addedAt: string;
}

interface ApiErrorBody {
  error?: { code?: string; message?: string };
}

export class ApiError extends Error {
  code: string;

  constructor(message: string, code: string) {
    super(message);
    this.name = "ApiError";
    this.code = code;
  }
}

function mapObservation(observation: ApiPriceObservation | null): PriceObservation | null {
  if (!observation) return null;
  return {
    amount: observation.amountCents / 100,
    currency: observation.currency,
    timestamp: observation.timestamp,
  };
}

function mapSecondHandDetection(
  observation: (ApiPriceObservation & { condition: SecondHandOfferDetection["condition"]; conditionLabel: string }) | null,
): SecondHandOfferDetection | null {
  if (!observation) return null;
  return {
    amount: observation.amountCents / 100,
    currency: observation.currency,
    condition: observation.condition,
    conditionLabel: observation.conditionLabel,
    timestamp: observation.timestamp,
  };
}

function mapItem(item: ApiTrackedItem): TrackedItem {
  return {
    ...item,
    latestPrice: mapObservation(item.latestPrice),
    lastThreeDetections: item.lastThreeDetections.map((observation) => mapObservation(observation)!),
    secondHandOffer: item.secondHandOffer
      ? {
          ...item.secondHandOffer,
          latestDetection: mapSecondHandDetection(item.secondHandOffer.latestDetection),
          lastThreeDetections: item.secondHandOffer.lastThreeDetections.map((observation) => mapSecondHandDetection(observation)!),
        }
      : null,
  };
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      Accept: "application/json",
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });

  if (response.status === 204) return undefined as T;

  const data = await response.json().catch(() => null) as ApiErrorBody | T | null;
  if (!response.ok) {
    const error = data && typeof data === "object" && "error" in data ? data.error : undefined;
    throw new ApiError(
      error?.message ?? `The request failed (${response.status}).`,
      error?.code ?? "REQUEST_FAILED",
    );
  }
  return data as T;
}

export async function listItems(): Promise<TrackedItem[]> {
  const response = await request<{ items: ApiTrackedItem[] }>("/api/v1/items");
  return response.items.map(mapItem);
}

export async function getItem(id: string): Promise<TrackedItem> {
  const item = await request<ApiTrackedItem>(`/api/v1/items/${encodeURIComponent(id)}`);
  return mapItem(item);
}

export async function addItem(url: string): Promise<TrackedItem> {
  const item = await request<ApiTrackedItem>("/api/v1/items", {
    method: "POST",
    body: JSON.stringify({ url }),
  });
  return mapItem(item);
}

export function deleteItem(id: string): Promise<void> {
  return request<void>(`/api/v1/items/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function refreshAllItems(): Promise<{ requestedAt: string; itemsQueued: number }> {
  return request<{ requestedAt: string; itemsQueued: number }>("/api/v1/items/refresh", { method: "POST" });
}

export function refreshItem(id: string): Promise<{ requestedAt: string; itemsQueued: number }> {
  return request<{ requestedAt: string; itemsQueued: number }>(`/api/v1/items/${encodeURIComponent(id)}/refresh`, { method: "POST" });
}
