import { getDocsSearchIndex } from "@/lib/docs";

export const revalidate = 3600;

export function GET() {
  return Response.json(getDocsSearchIndex(), {
    headers: {
      "Cache-Control": "public, max-age=3600, stale-while-revalidate=86400",
    },
  });
}
