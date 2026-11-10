import { request } from "./client";
import type { AmazonRequests } from "../types";

export async function getAmazonRequests(): Promise<AmazonRequests> {
  const response = await request<{ amazonRequests: AmazonRequests }>("/api/v1/amazon/requests");
  return response.amazonRequests;
}
