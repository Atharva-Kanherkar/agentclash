import { NextRequest, NextResponse } from "next/server";
import { getWorkOSClient } from "@/lib/auth/workos";
import { getWorkOSConfig } from "@/lib/auth/config";
import { sealSessionToResponse } from "@/lib/auth/session";

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

    const response = NextResponse.redirect(new URL("/dashboard", request.url));
    await sealSessionToResponse(response, {
      mode: "workos",
      accessToken: authResponse.accessToken,
      refreshToken: authResponse.refreshToken,
      expiresAt: Math.floor(Date.now() / 1000) + 3600,
    });

    return response;
  } catch {
    return NextResponse.redirect(
      new URL("/auth/login?error=callback_failed", request.url),
    );
  }
}
