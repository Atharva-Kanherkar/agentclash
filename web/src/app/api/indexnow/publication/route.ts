import { withAuth } from "@workos-inc/authkit-nextjs";
import { NextResponse } from "next/server";
import { submitIndexNow } from "@/lib/indexnow";
import { PUBLIC_ORIGIN } from "@/lib/public-content";

const PUBLICATION_ID = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export async function POST(request: Request) {
  const { user } = await withAuth();
  if (!user) {
    return NextResponse.json({ ok: false, error: "unauthorized" }, { status: 401 });
  }

  const payload = await request.json().catch(() => null);
  const id = payload && typeof payload === "object"
    ? (payload as { publication_id?: unknown }).publication_id
    : null;
  if (typeof id !== "string" || !PUBLICATION_ID.test(id)) {
    return NextResponse.json({ ok: false, error: "invalid_publication_id" }, { status: 400 });
  }

  const result = await submitIndexNow([`${PUBLIC_ORIGIN}/publications/${id}`]);
  const accepted = result.status === 200 || result.status === 202;
  return NextResponse.json(
    { ok: accepted, indexnowStatus: result.status },
    { status: accepted ? 202 : 502 },
  );
}
