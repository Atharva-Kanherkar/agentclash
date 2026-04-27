import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft, ArrowRight, Check, Star } from "lucide-react";
import { ClashMark } from "@/components/marketing/clash-mark";

export const metadata: Metadata = {
  title: "Pricing",
  description:
    "Free to self-host. Pay when you scale. AgentClash pricing for indie devs, teams, and enterprises.",
};

type Tier = {
  name: string;
  tag: string;
  price: string;
  priceSuffix?: string;
  blurb: string;
  cta: { label: string; href: string; external?: boolean; primary?: boolean };
  features: string[];
  highlight?: boolean;
};

const TIERS: Tier[] = [
  {
    name: "Open source",
    tag: "Self-host",
    price: "$0",
    priceSuffix: "forever",
    blurb:
      "Run the full engine on your own infra. No limits, no telemetry, no upsell.",
    cta: {
      label: "Star on GitHub",
      href: "https://github.com/agentclash/agentclash",
      external: true,
    },
    features: [
      "Full source on GitHub",
      "FSL-1.1-MIT license",
      "Bring your own Postgres, Temporal, sandbox",
      "Bring your own LLM keys",
      "Unlimited races, models, replays",
      "Community support (GitHub, Discord)",
    ],
  },
  {
    name: "Free",
    tag: "Hosted",
    price: "$0",
    priceSuffix: "/ month",
    blurb:
      "Hosted, no ops. Generous enough to actually evaluate the product on your task.",
    cta: { label: "Start your first race", href: "/auth/login" },
    features: [
      "1 seat, 1 workspace",
      "25 races / month",
      "Up to 4 models per race",
      "7-day replay retention",
      "BYOK LLM keys",
      "BYOK sandbox (E2B token)",
      "Community support",
    ],
  },
  {
    name: "Pro",
    tag: "For product teams",
    price: "$49",
    priceSuffix: "/ seat / month",
    blurb:
      "For teams running real evals against real production tasks. Five seats minimum.",
    cta: {
      label: "Start with Pro",
      href: "/auth/login?plan=pro",
      primary: true,
    },
    features: [
      "Everything in Free, plus:",
      "500 races / seat / month",
      "Up to 8 models per race",
      "30-day replay retention",
      "Hosted sandbox (with included credit)",
      "Private challenge packs",
      "CI integration (GitHub Actions, webhooks)",
      "3 concurrent races",
      "Email support, < 1 business day",
    ],
    highlight: true,
  },
  {
    name: "Enterprise",
    tag: "For orgs",
    price: "Custom",
    blurb:
      "Compliance, SSO, audit, dedicated support. Or self-host with a support contract.",
    cta: { label: "Talk to us", href: "mailto:hello@agentclash.dev" },
    features: [
      "Everything in Pro, plus:",
      "SSO / SAML",
      "Audit logs",
      "Unlimited replay retention",
      "99.9% uptime SLA",
      "Dedicated support channel",
      "Self-host option with support",
      "Custom MSA / billing terms",
    ],
  },
];

const FAQ: { q: string; a: string }[] = [
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
    a: "Pro is $49 per active seat per month, five seats minimum. Each seat gets its own race quota that pools at the workspace level. If your team needs more, Enterprise removes the per-seat metric entirely.",
  },
  {
    q: "Can I switch tiers later?",
    a: "Yes. Upgrades take effect immediately and prorate. Downgrades take effect at the next billing cycle so your team isn't surprised mid-month.",
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
            Free to self-host.
            <br />
            <span className="text-white/40">Pay when you scale.</span>
          </h1>
          <p className="mt-8 mx-auto max-w-[52ch] text-lg leading-[1.55] text-white/60">
            AgentClash is open source. Run it free on your own infra, or skip
            the ops with the hosted platform when you don&apos;t want to.
          </p>
        </div>
      </section>

      <section className="px-6 sm:px-12 pb-24 sm:pb-32">
        <div className="mx-auto max-w-[1440px] grid gap-5 md:grid-cols-2 lg:grid-cols-4">
          {TIERS.map((tier) => (
            <TierCard key={tier.name} tier={tier} />
          ))}
        </div>

        <p className="mt-10 mx-auto max-w-[68ch] text-center text-sm leading-[1.6] text-white/45">
          BYOK on every tier — we never mark up tokens. Race quota pools at the
          workspace level. Hosted sandbox uses E2B under the hood; you can swap
          in your own provider on Enterprise.
        </p>
      </section>

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
          <div className="mt-10 flex flex-col sm:flex-row gap-3 justify-center">
            <Link
              href="/auth/login"
              className="inline-flex items-center justify-center gap-2 rounded-md bg-white px-7 py-3 text-sm font-medium text-[#060606] hover:bg-white/90 transition-colors"
            >
              Start your first race
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

function TierCard({ tier }: { tier: Tier }) {
  const surface = tier.highlight ? "glass-card-elevated" : "glass-card";
  const ring = tier.highlight
    ? "ring-1 ring-white/15"
    : "";
  return (
    <div
      className={`${surface} glass-shine relative flex flex-col rounded-2xl p-7 ${ring}`}
    >
      {tier.highlight && (
        <span className="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full border border-white/20 bg-[#0b0e14]/95 px-3 py-1 font-mono text-[0.58rem] uppercase tracking-[0.22em] text-white/85 backdrop-blur">
          Recommended
        </span>
      )}

      <div className="font-mono text-[0.62rem] uppercase tracking-[0.22em] text-white/50">
        {tier.tag}
      </div>
      <div className="mt-3 font-[family-name:var(--font-display)] text-3xl tracking-[-0.02em] text-white">
        {tier.name}
      </div>

      <div className="mt-5 flex items-baseline gap-2">
        <span className="font-[family-name:var(--font-display)] text-[clamp(2.5rem,3.5vw,3.25rem)] tracking-[-0.03em] text-white leading-none">
          {tier.price}
        </span>
        {tier.priceSuffix && (
          <span className="text-sm text-white/45">{tier.priceSuffix}</span>
        )}
      </div>

      <p className="mt-4 text-sm leading-[1.55] text-white/60">{tier.blurb}</p>

      <CtaButton cta={tier.cta} />

      <div className="my-6 h-px bg-white/10" />

      <ul className="flex flex-col gap-3">
        {tier.features.map((feature) => (
          <li
            key={feature}
            className="flex items-start gap-2.5 text-[13.5px] leading-[1.5] text-white/75"
          >
            <Check
              className="mt-0.5 size-3.5 shrink-0 text-white/55"
              aria-hidden
            />
            <span>{feature}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function CtaButton({ cta }: { cta: Tier["cta"] }) {
  const base =
    "mt-6 inline-flex items-center justify-center gap-2 rounded-md px-5 py-2.5 text-sm font-medium transition-colors";
  const primary =
    "bg-white text-[#060606] hover:bg-white/90";
  const secondary =
    "border border-white/15 bg-white/[0.04] text-white/85 hover:text-white hover:border-white/25";
  const className = `${base} ${cta.primary ? primary : secondary}`;

  if (cta.external || cta.href.startsWith("mailto:")) {
    return (
      <a
        href={cta.href}
        target={cta.external ? "_blank" : undefined}
        rel={cta.external ? "noopener noreferrer" : undefined}
        className={className}
      >
        {cta.label}
        <ArrowRight className="size-3.5" />
      </a>
    );
  }
  return (
    <Link href={cta.href} className={className}>
      {cta.label}
      <ArrowRight className="size-3.5" />
    </Link>
  );
}
