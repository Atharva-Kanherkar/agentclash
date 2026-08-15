import { buildPublicLlmsIndex } from "@/lib/public-content";

export const revalidate = 3600;

function llmsResponse(request: Request, head: boolean) {
  const startedAt = Date.now();
  const body = buildPublicLlmsIndex();
  console.log(JSON.stringify({
    level: "info",
    event: "agent_readable_response",
    path: "/llms.txt",
    representation: "machine",
    status: 200,
    bytes: Buffer.byteLength(body, "utf8"),
    duration_ms: Date.now() - startedAt,
    request_id: request.headers.get("x-vercel-id"),
  }));
  return new Response(head ? null : body, {
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
      "Cache-Control": "public, max-age=0, s-maxage=3600, stale-while-revalidate=86400",
      "X-Content-Type-Options": "nosniff",
    },
  });
}

export function GET(request: Request) {
  return llmsResponse(request, false);
}

export function HEAD(request: Request) {
  return llmsResponse(request, true);
}
