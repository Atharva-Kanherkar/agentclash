import { NextResponse } from "next/server";
import { getSession } from "@/lib/auth/session";

/**
 * GET /api/auth/status
 *
 * Returns the current auth state. Used by client components
 * (like the landing page) to show signed-in indicators.
 */
export async function GET() {
  const session = await getSession();
  if (!session) {
    return NextResponse.json({ authenticated: false });
  }

  return NextResponse.json({
    authenticated: true,
    mode: session.mode,
    email: session.mode === "dev" ? session.email : undefined,
  });
}
