"use client";

import Link from "next/link";
import { useState } from "react";
import { ArrowRight, Check, Sparkles } from "lucide-react";

type Period = "monthly" | "yearly";

type Price = {
  value: string;
  suffix: string;
  note?: string;
};

type Cta = {
  label: string;
  href: string;
  external?: boolean;
  primary?: boolean;
  sublabel?: string;
};

type Tier = {
  name: string;
  tag: string;
  prices: { monthly: Price; yearly: Price };
  blurb: string;
  cta: Cta;
  features: string[];
  highlight?: boolean;
};

const TIERS: Tier[] = [
  {
    name: "Open source",
    tag: "Self-host",
    prices: {
      monthly: { value: "$0", suffix: "forever" },
      yearly: { value: "$0", suffix: "forever" },
    },
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
    prices: {
      monthly: { value: "$0", suffix: "/ month" },
      yearly: { value: "$0", suffix: "/ month" },
    },
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
    prices: {
      monthly: {
        value: "$49",
        suffix: "/ seat / month",
        note: "Billed monthly",
      },
      yearly: {
        value: "$39",
        suffix: "/ seat / month",
        note: "Billed annually · $468 / seat / yr",
      },
    },
    blurb:
      "For teams running real evals against real production tasks. Five seats minimum.",
    cta: {
      label: "Start free 30-day trial",
      href: "/auth/login?plan=pro",
      primary: true,
      sublabel: "No credit card required",
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
    prices: {
      monthly: { value: "Custom", suffix: "" },
      yearly: { value: "Custom", suffix: "" },
    },
    blurb:
      "Compliance, SSO, audit, dedicated support. 30-day pilot available — no card needed.",
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

export function PricingTiers() {
  const [period, setPeriod] = useState<Period>("monthly");

  return (
    <section className="px-6 sm:px-12 pb-24 sm:pb-32">
      <div className="mx-auto max-w-[1440px]">
        <div className="mb-8 flex flex-col items-center gap-5">
          <div className="inline-flex items-center gap-2.5 rounded-full border border-white/15 bg-white/[0.04] px-4 py-1.5 text-xs backdrop-blur">
            <Sparkles
              className="size-3.5 text-white/70"
              aria-hidden
            />
            <span className="font-medium text-white/90">
              Free for 30 days
            </span>
            <span className="text-white/30" aria-hidden>
              ·
            </span>
            <span className="text-white/55">
              No credit card required
            </span>
          </div>

          <PeriodToggle period={period} onChange={setPeriod} />
        </div>

        <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
          {TIERS.map((tier) => (
            <TierCard key={tier.name} tier={tier} period={period} />
          ))}
        </div>

        <p className="mt-10 mx-auto max-w-[68ch] text-center text-sm leading-[1.6] text-white/45">
          BYOK on every tier — we never mark up tokens. Race quota pools at the
          workspace level. Hosted sandbox uses E2B under the hood; you can swap
          in your own provider on Enterprise.
        </p>
      </div>
    </section>
  );
}

function PeriodToggle({
  period,
  onChange,
}: {
  period: Period;
  onChange: (p: Period) => void;
}) {
  return (
    <div
      className="inline-flex items-center rounded-full border border-white/10 bg-white/[0.03] p-1"
      role="group"
      aria-label="Billing period"
    >
      <ToggleButton
        active={period === "monthly"}
        onClick={() => onChange("monthly")}
      >
        Monthly
      </ToggleButton>
      <ToggleButton
        active={period === "yearly"}
        onClick={() => onChange("yearly")}
      >
        <span>Yearly</span>
        <span
          className={`rounded-full px-1.5 py-0.5 font-mono text-[0.55rem] font-semibold uppercase tracking-[0.12em] ${
            period === "yearly"
              ? "bg-[#060606]/15 text-[#060606]"
              : "bg-emerald-400/15 text-emerald-300"
          }`}
        >
          -20%
        </span>
      </ToggleButton>
    </div>
  );
}

function ToggleButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`inline-flex items-center gap-2 rounded-full px-4 py-1.5 text-xs font-medium transition-colors ${
        active
          ? "bg-white text-[#060606]"
          : "text-white/60 hover:text-white/90"
      }`}
    >
      {children}
    </button>
  );
}

function TierCard({ tier, period }: { tier: Tier; period: Period }) {
  const surface = tier.highlight ? "glass-card-elevated" : "glass-card";
  const ring = tier.highlight ? "ring-1 ring-white/15" : "";
  const price = tier.prices[period];

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
          {price.value}
        </span>
        {price.suffix && (
          <span className="text-sm text-white/45">{price.suffix}</span>
        )}
      </div>
      <div className="mt-1 min-h-[1.25rem] font-mono text-[0.62rem] uppercase tracking-[0.18em] text-white/35">
        {price.note ?? " "}
      </div>

      <p className="mt-3 text-sm leading-[1.55] text-white/60">{tier.blurb}</p>

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

function CtaButton({ cta }: { cta: Cta }) {
  const base =
    "inline-flex items-center justify-center gap-2 rounded-md px-5 py-2.5 text-sm font-medium transition-colors";
  const primary = "bg-white text-[#060606] hover:bg-white/90";
  const secondary =
    "border border-white/15 bg-white/[0.04] text-white/85 hover:text-white hover:border-white/25";
  const className = `${base} ${cta.primary ? primary : secondary}`;

  const inner = (
    <>
      {cta.label}
      <ArrowRight className="size-3.5" />
    </>
  );

  return (
    <div className="mt-6 flex flex-col items-stretch">
      {cta.external || cta.href.startsWith("mailto:") ? (
        <a
          href={cta.href}
          target={cta.external ? "_blank" : undefined}
          rel={cta.external ? "noopener noreferrer" : undefined}
          className={className}
        >
          {inner}
        </a>
      ) : (
        <Link href={cta.href} className={className}>
          {inner}
        </Link>
      )}
      {cta.sublabel && (
        <p className="mt-2 text-center text-[11px] text-white/40">
          {cta.sublabel}
        </p>
      )}
    </div>
  );
}
