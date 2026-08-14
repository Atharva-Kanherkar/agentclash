import { getPublicSearchIndex } from "@/lib/public-content";

export const revalidate = 3600;

export function GET() {
  return Response.json(getPublicSearchIndex(), {
    headers: {
      "Cache-Control": "public, max-age=3600, stale-while-revalidate=86400",
    },
  });
}
