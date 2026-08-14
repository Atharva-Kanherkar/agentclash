import type { Metadata } from "next";
import { Suspense } from "react";

import { PublicTryoutsClient } from "./tryouts-client";
import { markdownAlternate } from "@/lib/seo";
import { getPublicTryoutPageData } from "@/lib/public-page-data";

const baseMetadata: Metadata = {
  title: "Free AI Agent Tryout",
  description:
    "Write what you would reject in production, run a sandboxed tryout on real work, and get a scored verdict with outputs before you deploy.",
  keywords: [
    "AI agent evaluation",
    "integrate AI into business",
    "AI workflow automation",
    "customer support AI",
    "document AI extraction",
    "contract review AI",
    "AI agent pilot",
    "enterprise AI testing",
    "AI automation ROI",
    "agent quality bar",
  ],
  alternates: {
    canonical: "/tryouts",
    types: markdownAlternate("/tryouts"),
  },
  openGraph: {
    title: "Free AI Agent Tryout for Business Workflows | AgentClash",
    description:
      "Run a free sandboxed AI agent on support, finance, legal, and ops tasks. Set your quality bar and get a scored verdict before production.",
    url: "/tryouts",
  },
};

type Props = {
  searchParams: Promise<{ tryout?: string | string[] }>;
};

export async function generateMetadata({ searchParams }: Props): Promise<Metadata> {
  const { tryout } = await searchParams;
  if (!tryout) return baseMetadata;
  return {
    ...baseMetadata,
    robots: { index: false, follow: false },
  };
}

export default async function PublicTryoutsPage({ searchParams }: Props) {
  const value = (await searchParams).tryout;
  const tryoutId = typeof value === "string" ? value : undefined;
  const initialData = await getPublicTryoutPageData(tryoutId);

  return (
    <Suspense
      fallback={
        <main className="min-h-screen bg-[#131312] text-white">
          <div className="mx-auto flex min-h-screen max-w-6xl items-center justify-center px-4">
            <div className="flex items-center gap-1">
              {[0, 1, 2].map((index) => (
                <span
                  key={index}
                  className="size-1.5 rounded-full bg-white/30 animate-pulse"
                  style={{ animationDelay: `${index * 180}ms` }}
                />
              ))}
            </div>
          </div>
        </main>
      }
    >
      <PublicTryoutsClient
        initialTemplates={initialData.templates}
        initialTools={initialData.tools}
        initialTryout={initialData.tryout}
        initialEvents={initialData.events}
      />
    </Suspense>
  );
}
