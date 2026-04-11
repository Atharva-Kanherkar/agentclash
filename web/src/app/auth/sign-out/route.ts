import { NextRequest, NextResponse } from "next/server";
import { deleteSessionFromResponse } from "@/lib/auth/session";

export async function GET(request: NextRequest) {
  const response = NextResponse.redirect(new URL("/", request.url));
  deleteSessionFromResponse(response);
  return response;
}
