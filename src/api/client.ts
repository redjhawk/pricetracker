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

export const authRequiredEvent = "pricefollower:auth-required";

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
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
    // The session ended or protected mode started: let the app reload the session state.
    if (error?.code === "AUTH_REQUIRED") window.dispatchEvent(new Event(authRequiredEvent));
    throw new ApiError(
      error?.message ?? `The request failed (${response.status}).`,
      error?.code ?? "REQUEST_FAILED",
    );
  }
  return data as T;
}
