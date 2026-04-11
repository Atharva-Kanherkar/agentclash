import { NextRequest, NextResponse } from "next/server";
import { SESSION_COOKIE_NAME } from "@/lib/auth/session";

/**
 * Next.js proxy (formerly middleware) for route protection.
 *
 * Runs on Edge — does NOT decrypt the session cookie (iron-session
 * may need Node APIs). Instead, checks for cookie *existence* only.
 * Actual session validation happens in the dashboard layout (Node runtime).
 */
export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const hasSession = request.cookies.has(SESSION_COOKIE_NAME);

  // Already signed-in users visiting /auth/login → redirect to dashboard.
  if (pathname === "/auth/login" && hasSession) {
    return NextResponse.redirect(new URL("/dashboard", request.url));
  }

  // Protected routes: require session cookie.
  if (pathname.startsWith("/dashboard")) {
    if (!hasSession) {
      const loginUrl = new URL("/auth/login", request.url);
      return NextResponse.redirect(loginUrl);
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    // Auth routes (for signed-in redirect).
    "/auth/login",
    // Protected routes.
    "/dashboard/:path*",
  ],
};
