import { request } from "./client";
import { mapItem, type ApiTrackedItem } from "./items";
import type { AiReviewSummary, AmazonRequests, AmazonSearch, SearchItem, TrackedItem } from "../types";

type ApiSearchItem = ApiTrackedItem & { position: number; aiReviewSummary: AiReviewSummary };

const searchPath = (id: string) => `/api/v1/amazon-searches/${encodeURIComponent(id)}`;

export function listSearches(): Promise<{ searches: AmazonSearch[]; amazonRequests: AmazonRequests }> {
  return request("/api/v1/amazon-searches");
}

export function addSearch(url: string): Promise<AmazonSearch> {
  return request("/api/v1/amazon-searches", { method: "POST", body: JSON.stringify({ url }) });
}

export async function getSearch(id: string): Promise<{ search: AmazonSearch; amazonRequests: AmazonRequests; items: SearchItem[] }> {
  const response = await request<{ search: AmazonSearch; amazonRequests: AmazonRequests; items: ApiSearchItem[] }>(searchPath(id));
  return {
    ...response,
    items: response.items.map((item) => ({ ...mapItem(item), position: item.position, aiReviewSummary: item.aiReviewSummary })),
  };
}

export function deleteSearch(id: string): Promise<void> {
  return request<void>(searchPath(id), { method: "DELETE" });
}

export async function trackSearchItem(searchId: string, itemId: string): Promise<TrackedItem> {
  const item = await request<ApiTrackedItem>(`${searchPath(searchId)}/items/${encodeURIComponent(itemId)}/track`, { method: "POST" });
  return mapItem(item);
}

export function refreshSearch(id: string): Promise<{ requestedAt: string; search: AmazonSearch }> {
  return request(`${searchPath(id)}/refresh`, { method: "POST" });
}

export async function getAmazonRequests(): Promise<AmazonRequests> {
  const response = await request<{ amazonRequests: AmazonRequests }>("/api/v1/amazon/requests");
  return response.amazonRequests;
}
