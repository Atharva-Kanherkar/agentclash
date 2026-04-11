import { cookies } from "next/headers";
import { getIronSession } from "iron-session";
import { getApiUrl } from "@/lib/auth/config";
import { getSessionOptions } from "@/lib/auth/session";
import type { SessionData } from "@/lib/auth/session";
import type { ApiErrorResponse, SessionResponse, UserMeResponse } from "./types";

/**
 * Error thrown when the backend returns a non-OK response.
 * Callers can inspect `status` to decide how to handle it
 * (e.g. redirect on 401, show message on 4xx).
 */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/**
 * Build auth headers from the current session.
 * Returns headers appropriate for the session mode:
 *   - WorkOS: Authorization Bearer header
 *   - Dev: X-Agentclash-* identity headers
 */
async function getAuthHeaders(): Promise<Record<string, string>> {
  const cookieStore = await cookies();
  const session = await getIronSession<{ data?: SessionData }>(
    cookieStore,
    getSessionOptions(),
  );

  const data = session.data;
  if (!data) return {};

  if (data.mode === "workos") {
    return { Authorization: `Bearer ${data.accessToken}` };
  }

  // Dev mode — inject identity headers that the Go DevelopmentAuthenticator reads.
  const headers: Record<string, string> = {
    "X-Agentclash-User-Id": data.userId,
    "X-Agentclash-User-Email": data.email,
    "X-Agentclash-User-Display-Name": data.displayName,
  };
  if (data.orgMemberships) {
    headers["X-Agentclash-Org-Memberships"] = data.orgMemberships;
  }
  if (data.workspaceMemberships) {
    headers["X-Agentclash-Workspace-Memberships"] = data.workspaceMemberships;
  }
  return headers;
}

/**
 * Server-side fetch wrapper for the Go backend API.
 *
 * Usage:
 *   const user = await apiFetch<UserMeResponse>("/users/me");
 *   const orgs = await apiFetch<ListOrgsResponse>("/organizations?limit=10");
 *
 * Throws `ApiError` on non-OK responses.
 * Caller is responsible for handling auth errors (401) — typically
 * by destroying the session and redirecting to login.
 */
export async function apiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const baseUrl = getApiUrl();
  const authHeaders = await getAuthHeaders();

  const url = `${baseUrl}/v1${path}`;
  const response = await fetch(url, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...authHeaders,
      ...init?.headers,
    },
    // Disable Next.js fetch caching for API calls by default.
    // Individual callers can override with { next: { revalidate: N } }.
    cache: init?.cache ?? "no-store",
  });

  if (!response.ok) {
    let code = "unknown_error";
    let message = `API request failed: ${response.status}`;
    try {
      const body: ApiErrorResponse = await response.json();
      code = body.error.code;
      message = body.error.message;
    } catch {
      // Response wasn't JSON — keep the generic message.
    }
    throw new ApiError(response.status, code, message);
  }

  return response.json() as Promise<T>;
}

// --- Convenience wrappers for common endpoints ---

export async function getSessionFromAPI(): Promise<SessionResponse> {
  return apiFetch<SessionResponse>("/auth/session");
}

export async function getUserMe(): Promise<UserMeResponse> {
  return apiFetch<UserMeResponse>("/users/me");
}
