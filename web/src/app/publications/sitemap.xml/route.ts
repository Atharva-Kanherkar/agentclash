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
  const publications = await listAllPublicPublications().catch(() => []);
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
