import { NextRequest, NextResponse } from "next/server";
import { getWorkOSClient } from "@/lib/auth/workos";
import { getWorkOSConfig } from "@/lib/auth/config";
import { createWorkOSSession } from "@/lib/auth/session";

/**
 * GET /auth/callback
 *
 * Handles the WorkOS OAuth callback. Exchanges the authorization code
 * for access + refresh tokens, stores them in an encrypted session cookie,
 * and redirects to the dashboard.
 */
export async function GET(request: NextRequest) {
  const code = request.nextUrl.searchParams.get("code");
  if (!code) {
    return NextResponse.redirect(
      new URL("/auth/login?error=callback_failed", request.url),
    );
  }

  try {
    const workos = getWorkOSClient();
    const { clientId } = getWorkOSConfig();

    const authResponse = await workos.userManagement.authenticateWithCode({
      clientId,
      code,
    });

    // WorkOS returns accessToken, refreshToken, and user info.
    // We store tokens in the session; user info is fetched from our
    // backend via GET /v1/users/me (the backend is the source of truth).
    const expiresIn = 3600; // 1 hour (WorkOS default access token lifetime)
    await createWorkOSSession(
      authResponse.accessToken,
      authResponse.refreshToken,
      expiresIn,
    );

    return NextResponse.redirect(new URL("/dashboard", request.url));
  } catch {
    return NextResponse.redirect(
      new URL("/auth/login?error=callback_failed", request.url),
    );
  }
}
