import { request } from "./client";

export interface LeboncoinSessionSettings {
  value: string | null;
  revision: number;
  updatedAt: string | null;
  status: "none" | "active" | "expired" | "revoked";
  expiresAt: string | null;
  revokedAt: string | null;
  lastAttempt: { outcome: "accepted" | "rejected" | "failed"; attemptedAt: string } | null;
}

const sessionPath = "/api/v1/settings/leboncoin-session";

export async function getLeboncoinSession(): Promise<LeboncoinSessionSettings> {
  const response = await request<{ session: LeboncoinSessionSettings }>(sessionPath);
  return response.session;
}

export async function saveLeboncoinSession(value: string, revision: number): Promise<LeboncoinSessionSettings> {
  const response = await request<{ session: LeboncoinSessionSettings }>(sessionPath, {
    method: "PUT",
    body: JSON.stringify({ value, revision }),
  });
  return response.session;
}
