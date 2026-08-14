import { buildPublicLlmsFull } from "@/lib/public-content";

export const revalidate = 3600;

export function GET() {
  return new Response(buildPublicLlmsFull(), {
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
      "Cache-Control": "public, max-age=0, s-maxage=3600, stale-while-revalidate=86400",
      "X-Content-Type-Options": "nosniff",
    },
  });
}
