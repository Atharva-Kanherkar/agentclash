import { getIronSession, sealData, type SessionOptions } from "iron-session";
import { cookies } from "next/headers";
import { getSessionSecret } from "./config";

/**
 * Session payload stored in the encrypted cookie.
 * Discriminated union on `mode` so TypeScript can narrow the type.
 */
export type SessionData =
  | WorkOSSessionData
  | DevSessionData;

export interface WorkOSSessionData {
  mode: "workos";
  accessToken: string;
  refreshToken: string;
  expiresAt: number; // Unix timestamp (seconds)
}

export interface DevSessionData {
  mode: "dev";
  userId: string;
  email: string;
  displayName: string;
  orgMemberships: string;       // "uuid:role,uuid:role"
  workspaceMemberships: string; // "uuid:role,uuid:role"
}

const COOKIE_NAME = "agentclash_session";
const SESSION_TTL = 60 * 60 * 8; // 8 hours

export function getSessionOptions(): SessionOptions {
  return {
    password: getSessionSecret(),
    cookieName: COOKIE_NAME,
    ttl: SESSION_TTL,
    cookieOptions: {
      secure: process.env.NODE_ENV === "production",
      httpOnly: true,
      sameSite: "lax" as const,
      path: "/",
    },
  };
}

/**
 * Read the current session from the request cookies.
 * Returns the session data, or null if no session exists.
 */
export async function getSession(): Promise<SessionData | null> {
  const cookieStore = await cookies();
  const session = await getIronSession<{ data?: SessionData }>(
    cookieStore,
    getSessionOptions(),
  );
  return session.data ?? null;
}

/**
 * Create a dev session from the login form.
 * Works in Server Actions where cookies() has write access.
 */
export async function createDevSession(input: {
  userId: string;
  email: string;
  displayName: string;
  orgMemberships: string;
  workspaceMemberships: string;
}): Promise<void> {
  const cookieStore = await cookies();
  const session = await getIronSession<{ data?: SessionData }>(
    cookieStore,
    getSessionOptions(),
  );
  session.data = {
    mode: "dev",
    ...input,
  };
  await session.save();
}

/**
 * Seal session data and set it as a cookie on a NextResponse.
 *
 * Use this in Route Handlers (callback, sign-out, refresh) because
 * they return NextResponse.redirect() and cookies() from next/headers
 * cannot write in that context.
 *
 * Uses sealData directly so the format is identical to what
 * getIronSession produces when calling session.save().
 */
export async function sealSessionToResponse(
  response: { cookies: { set: (name: string, value: string, opts?: Record<string, unknown>) => void } },
  data: SessionData,
): Promise<void> {
  const password = getSessionSecret();
  const sealed = await sealData({ data }, { password, ttl: SESSION_TTL });
  response.cookies.set(COOKIE_NAME, sealed, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: SESSION_TTL,
  });
}

/**
 * Delete the session cookie from a NextResponse.
 */
export function deleteSessionFromResponse(
  response: { cookies: { set: (name: string, value: string, opts?: Record<string, unknown>) => void } },
): void {
  response.cookies.set(COOKIE_NAME, "", {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 0,
  });
}

/**
 * Name of the session cookie. Used by proxy for existence checks
 * without decrypting.
 */
export const SESSION_COOKIE_NAME = COOKIE_NAME;
