import { resolvePublicContent } from "@/lib/public-content";
import { representationLinkHeader } from "@/lib/public-http";

type Context = {
  params: Promise<{
    slug?: string[];
  }>;
};

async function docsMarkdownResponse(_request: Request, context: Context, head: boolean) {
  const params = await context.params;
  const slug = params.slug ?? [];
  const canonicalPath = slug.length === 0 ? "/docs" : `/docs/${slug.join("/")}`;
  const doc = resolvePublicContent(canonicalPath);

  if (!doc) {
    return new Response("Not found", {
      status: 404,
      headers: {
        "Content-Type": "text/plain; charset=utf-8",
      },
    });
  }

  return new Response(head ? null : doc.renderMarkdown(), {
    headers: {
      "Content-Type": "text/markdown; charset=utf-8",
      "Content-Language": "en",
      "Content-Location": doc.markdownPath,
      "Cache-Control": "public, max-age=0, s-maxage=3600, stale-while-revalidate=86400",
      "Link": representationLinkHeader(doc.canonicalPath, "https://www.agentclash.dev"),
      "Vary": "Accept",
      "X-Content-Type-Options": "nosniff",
      "X-Robots-Tag": "noindex, follow",
    },
  });
}

export function GET(request: Request, context: Context) {
  return docsMarkdownResponse(request, context, false);
}

export function HEAD(request: Request, context: Context) {
  return docsMarkdownResponse(request, context, true);
}
