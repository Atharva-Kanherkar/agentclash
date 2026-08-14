import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { PublicShareRenderer } from "@/components/share/public-share-renderers";
import { ApiError } from "@/lib/api/errors";
import { getPublicPublication } from "@/lib/publication-data";
import { publicationDescription, publicationTitle } from "@/lib/publications";
import { markdownAlternate } from "@/lib/seo";

export const dynamic = "force-dynamic";
export const revalidate = 0;

type Props = { params: Promise<{ id: string }> };

async function loadPublication(id: string) {
  try {
    return await getPublicPublication(id);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) notFound();
    if (error instanceof Error && error.message === "Invalid publication ID") notFound();
    throw error;
  }
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id } = await params;
  const publication = await loadPublication(id);
  const canonicalPath = publication.publication.canonical_path;
  return {
    title: `${publicationTitle(publication)} | AgentClash`,
    description: publicationDescription(publication),
    alternates: {
      canonical: canonicalPath,
      types: markdownAlternate(canonicalPath),
    },
    openGraph: {
      title: publicationTitle(publication),
      description: publicationDescription(publication),
      url: canonicalPath,
    },
  };
}

export default async function PublicationPage({ params }: Props) {
  const { id } = await params;
  const publication = await loadPublication(id);

  return (
    <main className="min-h-screen bg-background text-foreground">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-8 sm:px-6 lg:px-8">
        <header className="border-b border-border pb-5">
          <Link
            href="/publications"
            className="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground hover:text-foreground"
          >
            AgentClash publications
          </Link>
          <h1 className="mt-3 text-3xl font-sans font-semibold tracking-tight">
            {publicationTitle(publication)}
          </h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
            User-published content. This read-only artifact passed an allowlisted,
            redacted public serializer.
          </p>
          <div className="mt-4 flex flex-wrap gap-4 text-xs text-muted-foreground">
            <span>Type: {publication.publication.resource_type.replaceAll("_", " ")}</span>
            <span>Updated: {new Date(publication.publication.updated_at).toISOString()}</span>
            <Link
              href={`/md${publication.publication.canonical_path}`}
              className="underline-offset-4 hover:text-foreground hover:underline"
            >
              Markdown version
            </Link>
          </div>
        </header>

        <PublicShareRenderer resource={publication.resource as Record<string, unknown>} />
      </div>
    </main>
  );
}
