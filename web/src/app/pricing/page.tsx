import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft, ArrowRight, Star } from "lucide-react";
import { ClashMark } from "@/components/marketing/clash-mark";
import { PricingTiers } from "./pricing-tiers";

export const metadata: Metadata = {
  title: "Pricing",
  description:
    "Free for 30 days, no credit card required. AgentClash pricing for indie devs, teams, and enterprises.",
};

const FAQ: { q: string; a: string }[] = [
  {
    q: "Is the free trial really no card?",
    a: "Right. Sign up with email, run real evals against real models for 30 days. We'll email you before the trial ends so you can decide whether to add a card or drop down to the Free tier. No silent charges.",
  },
  {
    q: "Is the open source version the same code that runs the hosted platform?",
    a: "Yes. AgentClash is FSL-1.1-MIT licensed; the hosted platform adds operations (managed Temporal, Postgres, sandbox), not exclusive features. If you can run Postgres and a worker, you can self-host the same engine.",
  },
  {
    q: "Can I bring my own LLM keys?",
    a: "Yes, on every tier. We never charge a markup on tokens — your provider bills you directly. Pro adds an optional managed sandbox so you don't have to plug in an E2B token; LLMs stay BYOK.",
  },
  {
    q: "What counts as a 'race'?",
    a: "One run with one or more contestants on one challenge pack. A race that fails before any model produces a token doesn't count against your quota.",
  },
  {
    q: "How does the seat-based pricing work for big teams?",
    a: "Pro is $49 per active seat per month (or $39 billed annually), five seats minimum. Each seat gets its own race quota that pools at the workspace level. If your team needs more, Enterprise removes the per-seat metric entirely.",
  },
  {
    q: "Discounts for OSS contributors, startups, or research?",
    a: "Yes — email hello@agentclash.dev. We have programs for early-stage startups, accepted YC companies, university research groups, and significant OSS contributors.",
  },
];

export default function PricingPage() {
  return (
    <main className="min-h-screen bg-[#060606] text-white">
      <header className="px-5 sm:px-12 py-5 sm:py-6 border-b border-white/[0.06]">
        <div className="mx-auto flex max-w-[1440px] items-center justify-between">
          <Link
            href="/"
            className="inline-flex items-center gap-2.5 text-white/90"
          >
            <ClashMark className="size-6" />
            <span className="font-[family-name:var(--font-display)] text-xl tracking-[-0.01em]">
              AgentClash
            </span>
          </Link>
          <Link
            href="/"
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs text-white/55 hover:text-white/85 transition-colors"
          >
            <ArrowLeft className="size-3.5" />
            Back
          </Link>
        </div>
      </header>

      <section className="px-8 sm:px-12 pt-24 pb-12 sm:pt-32 sm:pb-16">
        <div className="mx-auto max-w-[1440px] text-center">
          <p className="font-mono text-[0.66rem] uppercase tracking-[0.28em] text-white/45">
            Pricing
          </p>
          <h1 className="mt-6 font-[family-name:var(--font-display)] font-normal tracking-[-0.04em] leading-[0.95] text-[clamp(2.75rem,7vw,6rem)] mx-auto max-w-[18ch]">
            Free for 30 days.
            <br />
            <span className="text-white/40">No credit card.</span>
          </h1>
          <p className="mt-8 mx-auto max-w-[52ch] text-lg leading-[1.55] text-white/60">
            Self-host the engine for free, or skip the ops with hosted —
            full Pro on us for the first month while we&apos;re launching.
          </p>
        </div>
      </section>

      <PricingTiers />

      <section className="border-t border-white/[0.06] px-8 sm:px-12 py-24 sm:py-32">
        <div className="mx-auto max-w-[1440px]">
          <p className="font-mono text-[0.66rem] uppercase tracking-[0.28em] text-white/45">
            FAQ
          </p>
          <h2 className="mt-4 font-[family-name:var(--font-display)] font-normal tracking-[-0.03em] leading-[1.05] text-[clamp(2rem,4.5vw,3.5rem)] max-w-[22ch]">
            Questions you&apos;d ask in the demo.
          </h2>

          <dl className="mt-14 grid gap-x-12 gap-y-10 md:grid-cols-2">
            {FAQ.map(({ q, a }) => (
              <div key={q}>
                <dt className="font-[family-name:var(--font-display)] text-xl tracking-[-0.015em] text-white/95">
                  {q}
                </dt>
                <dd className="mt-3 text-[15px] leading-[1.65] text-white/55">
                  {a}
                </dd>
              </div>
            ))}
          </dl>
        </div>
      </section>

      <section className="border-t border-white/[0.06] px-8 sm:px-12 py-32 sm:py-40">
        <div className="mx-auto max-w-[1440px] text-center">
          <h2 className="font-[family-name:var(--font-display)] font-normal tracking-[-0.04em] leading-[0.95] text-[clamp(2.5rem,5.5vw,5rem)] max-w-[18ch] mx-auto">
            Stop guessing.
            <br />
            <span className="text-white/40">Start racing.</span>
          </h2>
          <p className="mt-6 text-sm text-white/45">
            30 days of Pro on us. No credit card required.
          </p>
          <div className="mt-10 flex flex-col sm:flex-row gap-3 justify-center">
            <Link
              href="/auth/login?plan=pro"
              className="inline-flex items-center justify-center gap-2 rounded-md bg-white px-7 py-3 text-sm font-medium text-[#060606] hover:bg-white/90 transition-colors"
            >
              Start free 30-day trial
              <ArrowRight className="size-4" />
            </Link>
            <a
              href="https://github.com/agentclash/agentclash"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center gap-2 rounded-md border border-white/[0.08] bg-white/[0.02] px-7 py-3 text-sm font-medium text-white/60 hover:text-white/90 hover:border-white/20 transition-colors"
            >
              <Star className="size-4" />
              Star on GitHub
            </a>
          </div>
        </div>
      </section>
    </main>
  );
}
