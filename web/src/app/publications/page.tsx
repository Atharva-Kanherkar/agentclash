import type { Metadata } from "next";
import Link from "next/link";
import { MarketingShell } from "@/components/marketing/marketing-shell";
import { listPublicPublications } from "@/lib/publication-data";
import { publicationDescription, publicationTitle } from "@/lib/publications";
import { markdownAlternate } from "@/lib/seo";

export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "Published Agent Evaluation Artifacts | AgentClash",
  description:
    "Browse explicitly published, redacted AgentClash challenge packs, scorecards, replays, and agent tryouts.",
  alternates: {
    canonical: "/publications",
    types: markdownAlternate("/publications"),
  },
};

type Props = { searchParams: Promise<{ cursor?: string }> };

export default async function PublicationsPage({ searchParams }: Props) {
  const { cursor } = await searchParams;
  const result = await listPublicPublications({ cursor, limit: 50 }).catch(() => ({
    items: [],
    next_cursor: undefined,
  }));

  return (
    <MarketingShell>
      <section className="px-6 py-20 sm:px-12 sm:py-28">
        <div className="mx-auto max-w-[1080px]">
          <p className="font-mono text-2xs uppercase tracking-[0.14em] text-white/40">
            Public evidence
          </p>
          <h1 className="mt-5 max-w-[18ch] text-[clamp(2.5rem,6vw,4.75rem)] font-sans font-semibold leading-[1.04] tracking-tight text-white">
            Published agent evaluation artifacts
          </h1>
          <p className="mt-7 max-w-[64ch] text-lg leading-8 text-white/60">
            Every item here was explicitly opted into search indexing and passed
            its redacted public serializer. Capability-token shares never appear.
          </p>

          {result.items.length > 0 ? (
            <div className="mt-12 grid gap-px overflow-hidden rounded-xl border border-white/[0.08] bg-white/[0.08] md:grid-cols-2">
              {result.items.map((item) => (
                <article key={item.publication.id} className="bg-[#060606] p-6">
                  <p className="font-mono text-2xs uppercase tracking-[0.12em] text-white/35">
                    {item.publication.resource_type.replaceAll("_", " ")}
                  </p>
                  <h2 className="mt-3 text-xl font-sans font-semibold tracking-tight text-white">
                    <Link
                      href={item.publication.canonical_path}
                      className="underline-offset-4 hover:underline"
                    >
                      {publicationTitle(item)}
                    </Link>
                  </h2>
                  <p className="mt-3 text-sm leading-6 text-white/50">
                    {publicationDescription(item)}
                  </p>
                  <p className="mt-5 text-xs text-white/30">
                    Updated {new Date(item.publication.updated_at).toLocaleDateString("en", {
                      year: "numeric",
                      month: "short",
                      day: "numeric",
                      timeZone: "UTC",
                    })}
                  </p>
                </article>
              ))}
            </div>
          ) : (
            <div className="mt-12 rounded-xl border border-white/[0.08] bg-white/[0.025] p-8">
              <h2 className="text-lg font-semibold text-white">No active publications yet</h2>
              <p className="mt-3 max-w-[56ch] text-sm leading-6 text-white/50">
                Public artifacts appear here only after their owner enables search
                indexing. Private and token-based shares remain excluded.
              </p>
            </div>
          )}

          {result.next_cursor ? (
            <div className="mt-8">
              <Link
                href={`/publications?cursor=${encodeURIComponent(result.next_cursor)}`}
                rel="next"
                className="inline-flex min-h-11 items-center rounded-md border border-white/[0.12] px-5 text-sm font-medium text-white transition-colors hover:bg-white/[0.06]"
              >
                More publications
              </Link>
            </div>
          ) : null}
        </div>
      </section>
    </MarketingShell>
  );
}
