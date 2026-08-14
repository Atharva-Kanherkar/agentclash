import { readFile } from "node:fs/promises";
import path from "node:path";

export const revalidate = 3600;

const SCHEMA_ROOT = path.join(process.cwd(), "..", "docs", "schemas");
const ALLOWED_SCHEMAS = new Set([
  "prompt-eval-result.schema.json",
  "prompt-eval.schema.json",
  "voice-artifact-manifest.schema.json",
  "voice-live-continuity-report.schema.json",
  "voice-source-separation-report.schema.json",
  "voice-video-sync-report.schema.json",
]);

type Context = { params: Promise<{ path: string[] }> };

async function schemaResponse(request: Request, context: Context, head: boolean) {
  const startedAt = Date.now();
  const { path: segments } = await context.params;
  const relativePath = segments.join("/");
  if (!ALLOWED_SCHEMAS.has(relativePath)) {
    console.log(JSON.stringify({
      level: "info",
      event: "agent_readable_response",
      path: "/schemas/{unrecognized}",
      representation: "machine",
      status: 404,
      bytes: 0,
      duration_ms: Date.now() - startedAt,
      request_id: request.headers.get("x-vercel-id"),
    }));
    return new Response(head ? null : "Not found", {
      status: 404,
      headers: { "X-Robots-Tag": "noindex, follow" },
    });
  }

  try {
    const body = await readFile(path.join(SCHEMA_ROOT, relativePath), "utf8");
    JSON.parse(body);
    console.log(
      JSON.stringify({
        level: "info",
        event: "agent_readable_response",
        path: `/schemas/${relativePath}`,
        representation: "machine",
        status: 200,
        bytes: Buffer.byteLength(body, "utf8"),
        duration_ms: Date.now() - startedAt,
        request_id: request.headers.get("x-vercel-id"),
      }),
    );
    return new Response(head ? null : body, {
      headers: {
        "Content-Type": "application/schema+json; charset=utf-8",
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
        path: `/schemas/${relativePath}`,
        representation: "machine",
        status: 500,
        duration_ms: Date.now() - startedAt,
        request_id: request.headers.get("x-vercel-id"),
        error: error instanceof Error ? error.message : String(error),
      }),
    );
    return new Response(head ? null : "Schema unavailable", { status: 500 });
  }
}

export function GET(request: Request, context: Context) {
  return schemaResponse(request, context, false);
}

export function HEAD(request: Request, context: Context) {
  return schemaResponse(request, context, true);
}
