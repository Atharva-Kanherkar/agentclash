import { readFile } from "node:fs/promises";
import path from "node:path";

export const revalidate = 3600;

const OPENAPI_PATH = path.join(
  process.cwd(),
  "..",
  "docs",
  "api-server",
  "openapi.yaml",
);

async function openApiResponse(request: Request, head: boolean) {
  const startedAt = Date.now();
  try {
    const body = await readFile(OPENAPI_PATH, "utf8");
    console.log(
      JSON.stringify({
        level: "info",
        event: "agent_readable_response",
        path: "/openapi.yaml",
        representation: "machine",
        status: 200,
        bytes: Buffer.byteLength(body, "utf8"),
        duration_ms: Date.now() - startedAt,
        request_id: request.headers.get("x-vercel-id"),
      }),
    );
    return new Response(head ? null : body, {
      headers: {
        "Content-Type": "application/yaml; charset=utf-8",
        "Cache-Control": "public, max-age=0, s-maxage=3600, stale-while-revalidate=86400",
        "X-Content-Type-Options": "nosniff",
        "X-Robots-Tag": "noindex, follow",
      },
    });
  } catch (error) {
    console.error(
      JSON.stringify({
        level: "error",
        event: "agent_readable_response",
        path: "/openapi.yaml",
        representation: "machine",
        status: 500,
        duration_ms: Date.now() - startedAt,
        request_id: request.headers.get("x-vercel-id"),
        error: error instanceof Error ? error.message : String(error),
      }),
    );
    return new Response(head ? null : "OpenAPI contract unavailable", { status: 500 });
  }
}

export function GET(request: Request) {
  return openApiResponse(request, false);
}

export function HEAD(request: Request) {
  return openApiResponse(request, true);
}
