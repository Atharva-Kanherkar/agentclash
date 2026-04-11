import { NextRequest, NextResponse } from "next/server";
import { unsealData } from "iron-session";
import { getWorkOSClient } from "@/lib/auth/workos";
import { getWorkOSConfig } from "@/lib/auth/config";
import { getSessionSecret } from "@/lib/auth/config";
import { sealSessionToResponse, SESSION_COOKIE_NAME } from "@/lib/auth/session";
import type { SessionData } from "@/lib/auth/session";

export async function GET(request: NextRequest) {
  const returnTo = request.nextUrl.searchParams.get("returnTo") || "/dashboard";

  try {
    const rawCookie = request.cookies.get(SESSION_COOKIE_NAME);
    if (!rawCookie?.value) {
      return NextResponse.redirect(new URL("/auth/sign-out", request.url));
    }

    const password = getSessionSecret();
    const unsealed = await unsealData<{ data?: SessionData }>(rawCookie.value, { password });
    const sessionData = unsealed.data;

    if (!sessionData || sessionData.mode !== "workos" || !sessionData.refreshToken) {
      return NextResponse.redirect(new URL("/auth/sign-out", request.url));
    }

    const workos = getWorkOSClient();
    const { clientId } = getWorkOSConfig();

    const refreshed = await workos.userManagement.authenticateWithRefreshToken({
      clientId,
      refreshToken: sessionData.refreshToken,
    });

    const response = NextResponse.redirect(new URL(returnTo, request.url));
    await sealSessionToResponse(response, {
      mode: "workos",
      accessToken: refreshed.accessToken,
      refreshToken: refreshed.refreshToken,
      expiresAt: Math.floor(Date.now() / 1000) + 3600,
    });

    return response;
  } catch {
    return NextResponse.redirect(new URL("/auth/sign-out", request.url));
  }
}
