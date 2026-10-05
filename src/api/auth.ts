import { request } from "./client";

export interface SessionUser {
  username: string;
  role: "admin" | "user";
}

export interface SessionState {
  mode: "open" | "protected";
  user: SessionUser | null;
}

export interface UserSummary {
  username: string;
  lastLoginAt: string | null;
}

export function getSession(): Promise<SessionState> {
  return request<SessionState>("/api/v1/auth/session");
}

export async function login(username: string, password: string): Promise<SessionUser> {
  const response = await request<{ user: SessionUser }>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
  return response.user;
}

export function logout(): Promise<void> {
  return request<void>("/api/v1/auth/logout", { method: "POST" });
}

export async function listUsers(): Promise<UserSummary[]> {
  const response = await request<{ users: UserSummary[] }>("/api/v1/admin/users");
  return response.users;
}

export async function createUser(username: string, password: string): Promise<UserSummary> {
  const response = await request<{ user: UserSummary }>("/api/v1/admin/users", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
  return response.user;
}
