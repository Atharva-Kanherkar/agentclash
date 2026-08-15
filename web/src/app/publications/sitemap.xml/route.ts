import { listAllPublicPublications } from "@/lib/publication-data";
import { PUBLIC_ORIGIN } from "@/lib/public-content";

export const dynamic = "force-dynamic";
export const revalidate = 0;

function xml(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;");
}

export async function GET() {
  let publications: Awaited<ReturnType<typeof listAllPublicPublications>>;
  try {
    publications = await listAllPublicPublications();
  } catch (error) {
    console.error(
      JSON.stringify({
        level: "error",
        event: "publications_sitemap_unavailable",
        status: 503,
        error: error instanceof Error ? error.message : String(error),
      }),
    );
    return new Response("Publication sitemap temporarily unavailable", {
      status: 503,
      headers: {
        "Cache-Control": "no-store",
        "Retry-After": "60",
        "X-Content-Type-Options": "nosniff",
      },
    });
  }
  const urls = publications.map(
    (item) => `  <url>\n    <loc>${xml(`${PUBLIC_ORIGIN}${item.publication.canonical_path}`)}</loc>\n    <lastmod>${xml(item.publication.updated_at)}</lastmod>\n  </url>`,
  );
  const body = [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
    ...urls,
    "</urlset>",
  ].join("\n");
  return new Response(body, {
    headers: {
      "Content-Type": "application/xml; charset=utf-8",
      "Cache-Control": "no-store",
      "X-Content-Type-Options": "nosniff",
    },
  });
}
