import { NextRequest, NextResponse } from "next/server";
import { destroySession } from "@/lib/auth/session";

/**
 * GET /auth/sign-out
 *
 * Destroys the session cookie and redirects to the landing page.
 */
export async function GET(request: NextRequest) {
  await destroySession();
  return NextResponse.redirect(new URL("/", request.url));
}
