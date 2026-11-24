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

export interface ClaudeTokenSettings {
  value: string | null;
  updatedAt: string | null;
  lastRejectedAt: string | null;
}

export async function getClaudeToken(): Promise<ClaudeTokenSettings> {
  const response = await request<{ claudeToken: ClaudeTokenSettings }>("/api/v1/settings/claude-token");
  return response.claudeToken;
}

export function saveSettings(input: {
  leboncoinSession?: { value: string; revision: number };
  claudeToken?: { value: string };
}): Promise<{ session: LeboncoinSessionSettings; claudeToken: ClaudeTokenSettings }> {
  return request<{ session: LeboncoinSessionSettings; claudeToken: ClaudeTokenSettings }>("/api/v1/settings", {
    method: "PUT",
    body: JSON.stringify(input),
  });
}
