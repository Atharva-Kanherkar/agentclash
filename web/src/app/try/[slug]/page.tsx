import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { TryCliDemoClient } from "@/components/try-cli/demo-client";
import { getPublicTryCliDemo } from "@/lib/public-page-data";
import { markdownAlternate } from "@/lib/seo";

interface Props {
  params: Promise<{ slug: string }>;
}

export default async function TryCliDemoPage({ params }: Props) {
  const { slug } = await params;
  const demo = await getPublicTryCliDemo(slug);
  if (!demo) notFound();
  return <TryCliDemoClient slug={slug} initialDemo={demo} />;
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params;
  const demo = await getPublicTryCliDemo(slug);
  if (!demo) return { title: "CLI demo not found | AgentClash", robots: { index: false } };
  const canonicalPath = `/try/${demo.slug}`;
  return {
    title: `Try ${demo.name} in browser | AgentClash`,
    description:
      demo.tagline ?? `Run ${demo.name} in a disposable browser sandbox. No install required.`,
    alternates: {
      canonical: canonicalPath,
      types: markdownAlternate(canonicalPath),
    },
    openGraph: {
      title: `Try ${demo.name} in browser | AgentClash`,
      description:
        demo.tagline ?? `Run ${demo.name} in a disposable browser sandbox.`,
      url: canonicalPath,
    },
  };
}
