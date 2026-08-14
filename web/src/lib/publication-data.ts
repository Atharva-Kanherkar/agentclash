import { createApiClient } from "@/lib/api/client";
import type {
  PublicPublicationListResponse,
  PublicPublicationResponse,
} from "@/lib/api/types";
import type { PublicContentAdapter } from "@/lib/public-content";
import {
  publicationDescription,
  publicationTitle,
  renderPublicationMarkdown,
} from "@/lib/publications";

const PUBLICATION_ID = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export async function getPublicPublication(
  id: string,
): Promise<PublicPublicationResponse> {
  if (!PUBLICATION_ID.test(id)) throw new Error("Invalid publication ID");
  return createApiClient().get<PublicPublicationResponse>(
    `/public/publications/${encodeURIComponent(id)}`,
    { cache: "no-store" },
  );
}

export async function listPublicPublications(args: {
  cursor?: string;
  limit?: number;
} = {}): Promise<PublicPublicationListResponse> {
  return createApiClient().get<PublicPublicationListResponse>(
    "/public/publications",
    {
      params: { cursor: args.cursor, limit: args.limit ?? 50 },
      cache: "no-store",
    },
  );
}

export async function listAllPublicPublications(
  maximum = 10_000,
): Promise<PublicPublicationResponse[]> {
  const items: PublicPublicationResponse[] = [];
  const seenCursors = new Set<string>();
  let cursor: string | undefined;
  while (items.length < maximum) {
    const page = await listPublicPublications({ cursor, limit: Math.min(100, maximum - items.length) });
    items.push(...page.items);
    if (!page.next_cursor || seenCursors.has(page.next_cursor)) break;
    seenCursors.add(page.next_cursor);
    cursor = page.next_cursor;
  }
  return items;
}

export async function resolvePublicPublicationAdapter(
  canonicalPath: string | null,
): Promise<PublicContentAdapter | null> {
  const id = canonicalPath?.match(/^\/publications\/([^/]+)$/)?.[1];
  if (!id) return null;
  try {
    const publication = await getPublicPublication(id);
    return {
      canonicalPath: publication.publication.canonical_path,
      markdownPath: `/md${publication.publication.canonical_path}`,
      title: publicationTitle(publication),
      description: publicationDescription(publication),
      kind: "publication",
      lastModified: publication.publication.updated_at,
      indexable: true,
      includeIn: {
        sitemap: false,
        llms: false,
        llmsFull: false,
        search: false,
        indexNow: true,
      },
      renderMarkdown: (origin) => renderPublicationMarkdown(publication, origin),
    };
  } catch {
    return null;
  }
}
