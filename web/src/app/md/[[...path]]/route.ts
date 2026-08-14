import {
  canonicalPathFromMarkdownSegments,
  resolvePublicContent,
} from "@/lib/public-content";
import {
  CANONICAL_PATH_HEADER,
  NEGOTIATED_MARKDOWN_HEADER,
  representationLinkHeader,
} from "@/lib/public-http";

export const revalidate = 3600;

type Context = {
  params: Promise<{ path?: string[] }>;
};

function appendVary(headers: Headers, value: string) {
  const values = new Set(
    (headers.get("Vary") ?? "")
      .split(",")
      .map((entry) => entry.trim())
      .filter(Boolean),
  );
  values.add(value);
  headers.set("Vary", [...values].join(", "));
}

async function markdownResponse(request: Request, context: Context, head: boolean) {
  const startedAt = Date.now();
  const { path = [] } = await context.params;
  const rewrittenCanonical = request.headers.get(CANONICAL_PATH_HEADER);
  const canonicalPath = rewrittenCanonical ?? canonicalPathFromMarkdownSegments(path);
  const negotiated = request.headers.get(NEGOTIATED_MARKDOWN_HEADER) === "1";
  const content = canonicalPath ? resolvePublicContent(canonicalPath) : null;

  if (!content) {
    console.log(
      JSON.stringify({
        level: "info",
        event: "agent_readable_response",
        path: canonicalPath ?? "/invalid",
        representation: "markdown",
        status: 404,
        bytes: 0,
        duration_ms: Date.now() - startedAt,
        request_id: request.headers.get("x-vercel-id"),
      }),
    );
    return new Response(head ? null : "Not found", {
      status: 404,
      headers: {
        "Content-Type": "text/plain; charset=utf-8",
        "X-Robots-Tag": "noindex, follow",
      },
    });
  }

  const body = content.renderMarkdown();
  const headers = new Headers({
    "Content-Type": "text/markdown; charset=utf-8",
    "Content-Language": "en",
    "Content-Location": content.markdownPath,
    "X-Content-Type-Options": "nosniff",
    "Cache-Control":
      content.kind === "publication"
        ? "no-store"
        : "public, max-age=0, s-maxage=3600, stale-while-revalidate=86400",
    Link: representationLinkHeader(content.canonicalPath, "https://www.agentclash.dev"),
  });
  appendVary(headers, "Accept");
  if (!negotiated) headers.set("X-Robots-Tag", "noindex, follow");

  console.log(
    JSON.stringify({
      level: "info",
      event: "agent_readable_response",
      path: content.canonicalPath,
      content_kind: content.kind,
      representation: "markdown",
      status: 200,
      bytes: Buffer.byteLength(body, "utf8"),
      duration_ms: Date.now() - startedAt,
      request_id: request.headers.get("x-vercel-id"),
    }),
  );
  return new Response(head ? null : body, { headers });
}

export function GET(request: Request, context: Context) {
  return markdownResponse(request, context, false);
}

export function HEAD(request: Request, context: Context) {
  return markdownResponse(request, context, true);
}
