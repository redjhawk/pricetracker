import { request } from "./client";

// The server never returns a saved secret: only whether one is saved and its masked hint (issue #265).
export interface LeboncoinSessionSettings {
  saved: boolean;
  hint: string | null;
  revision: number;
  updatedAt: string | null;
  status: "none" | "active" | "expired" | "revoked";
  expiresAt: string | null;
  revokedAt: string | null;
  lastAttempt: { outcome: "accepted" | "rejected" | "failed"; attemptedAt: string } | null;
}

export async function getLeboncoinSession(): Promise<LeboncoinSessionSettings> {
  const response = await request<{ session: LeboncoinSessionSettings }>("/api/v1/settings/leboncoin-session");
  return response.session;
}

export interface ClaudeTokenSettings {
  saved: boolean;
  hint: string | null;
  updatedAt: string | null;
  lastRejectedAt: string | null;
}

export async function getClaudeToken(): Promise<ClaudeTokenSettings> {
  const response = await request<{ claudeToken: ClaudeTokenSettings }>("/api/v1/settings/claude-token");
  return response.claudeToken;
}

// An absent entry is left unchanged; `value` replaces the secret and `remove` removes it.
export interface SettingsChange {
  leboncoinSession?: { value: string; revision: number } | { remove: true; revision: number };
  claudeToken?: { value: string } | { remove: true };
}

export function saveSettings(change: SettingsChange): Promise<{ session: LeboncoinSessionSettings; claudeToken: ClaudeTokenSettings }> {
  return request<{ session: LeboncoinSessionSettings; claudeToken: ClaudeTokenSettings }>("/api/v1/settings", {
    method: "PUT",
    body: JSON.stringify(change),
  });
}
