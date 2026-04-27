"use client";

import Link from "next/link";
import { useState } from "react";
import { ArrowRight } from "lucide-react";
import { TiltCard } from "@/app/auth/login/tilt-card";
import { ShaderLines } from "./shader-lines";

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
  prices: { monthly: Price; yearly: Price };
  blurb: string;
  cta: Cta;
  features: string[];
};

const TIERS: Tier[] = [
  {
    name: "Free",
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
      label: "Start free 45-day trial",
      href: "/auth/login?plan=pro",
      primary: true,
      sublabel: "No credit card required",
    },
    features: [
      "Everything in Free, plus:",
      "500 races / seat / month",
      "Up to 8 models per race",
      "30-day replay retention",
      "Hosted sandbox with included credit",
      "Private challenge packs",
      "CI integration (GitHub Actions, webhooks)",
      "3 concurrent races",
      "Email support, < 1 business day",
    ],
  },
  {
    name: "Team",
    prices: {
      monthly: {
        value: "$100",
        suffix: "/ seat / month",
        note: "Billed monthly",
      },
      yearly: {
        value: "$80",
        suffix: "/ seat / month",
        note: "Billed annually · $960 / seat / yr",
      },
    },
    blurb:
      "For teams running evals across multiple products and surfaces.",
    cta: {
      label: "Start free 45-day trial",
      href: "/auth/login?plan=team",
      sublabel: "No credit card required",
    },
    features: [
      "Everything in Pro, plus:",
      "2,000 races / seat / month",
      "Up to 12 models per race",
      "90-day replay retention",
      "10 concurrent races",
      "Multiple workspaces",
      "Workspace-level audit log",
      "Slack notifications",
      "Priority email support, < 4 business hours",
    ],
  },
  {
    name: "Enterprise",
    prices: {
      monthly: { value: "Custom", suffix: "" },
      yearly: { value: "Custom", suffix: "" },
    },
    blurb:
      "Compliance, SSO, dedicated support. 45-day pilot available — no card needed.",
    cta: { label: "Talk to us", href: "mailto:hello@agentclash.dev" },
    features: [
      "Everything in Team, plus:",
      "SSO / SAML",
      "Org-wide audit logs",
      "Unlimited replay retention",
      "99.9% uptime SLA",
      "Dedicated support channel",
      "Custom MSA / billing terms",
    ],
  },
];

export function PricingBlock() {
  const [period, setPeriod] = useState<Period>("monthly");

  return (
    <section
      id="pricing"
      className="relative isolate border-t border-white/[0.06] py-32 sm:py-44 overflow-hidden"
    >
      <div className="absolute inset-0 -z-10">
        <ShaderLines
          colorA="#ffffff"
          colorB="#ffffff"
          colorIntensity={0.5}
          animationSpeed={0.035}
          mosaicScale={{ x: 7, y: 3.5 }}
          centerFade={1}
        />
      </div>
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 top-0 h-72 -z-[5] bg-gradient-to-b from-[#060606] via-[#060606]/60 to-transparent"
      />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 bottom-0 h-48 -z-[5] bg-gradient-to-b from-transparent to-[#060606]"
      />

      <div className="relative px-6 sm:px-12">
        <div className="mx-auto max-w-[1440px]">
          <div className="text-center">
            <h2 className="text-3xl sm:text-5xl font-semibold tracking-tight text-white">
              Free for 45 days.
            </h2>
            <p className="mt-4 mx-auto max-w-[44ch] text-sm leading-6 text-white/75">
              No credit card. Self-host the engine for free, or skip the ops
              with hosted.
            </p>

            <div className="mt-10 flex justify-center">
              <PeriodToggle period={period} onChange={setPeriod} />
            </div>
          </div>

          <div className="mt-14 grid gap-5 md:grid-cols-2 lg:grid-cols-4">
            {TIERS.map((tier) => (
              <TierCard key={tier.name} tier={tier} period={period} />
            ))}
          </div>

          <p className="mt-10 mx-auto max-w-[64ch] text-center text-sm leading-6 text-white/60">
            BYOK on every tier — we never mark up tokens. Race quota pools at
            the workspace level.
          </p>
        </div>
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
      className="inline-flex items-center rounded-full border border-white/10 bg-white/[0.04] p-1 backdrop-blur"
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
          className={`rounded-full px-1.5 py-0.5 text-[10px] font-semibold tracking-tight ${
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
        active ? "bg-white text-[#060606]" : "text-white/65 hover:text-white"
      }`}
    >
      {children}
    </button>
  );
}

function TierCard({ tier, period }: { tier: Tier; period: Period }) {
  const price = tier.prices[period];

  return (
    <TiltCard className="h-full">
      <div
        className="glass-card glass-shine relative flex h-full flex-col rounded-2xl p-6 sm:p-7"
        // Override the global glass-card bg (2.5% white) with a denser surface
        // so shader streaks behind the card don't wash out the text.
        style={{ backgroundColor: "rgba(255, 255, 255, 0.06)" }}
      >
        <h3 className="text-2xl font-semibold text-white">{tier.name}</h3>
        <p className="mt-2 text-sm leading-6 text-white/75">{tier.blurb}</p>

        <div className="mt-6 flex items-baseline gap-2">
          <span className="text-4xl font-semibold tracking-tight text-white">
            {price.value}
          </span>
          {price.suffix && (
            <span className="text-sm text-white/55">{price.suffix}</span>
          )}
        </div>
        <div className="mt-1 min-h-[1.25rem] text-xs text-white/55">
          {price.note ?? " "}
        </div>

        <CtaButton cta={tier.cta} />

        <div className="my-6 h-px bg-white/10" />

        <ul className="flex flex-col gap-2.5 text-[14px] leading-6 text-white/85">
          {tier.features.map((feature) => (
            <li key={feature} className="flex items-start gap-2.5">
              <span
                aria-hidden
                className="select-none text-white/45 leading-6"
              >
                —
              </span>
              <span>{feature}</span>
            </li>
          ))}
        </ul>
      </div>
    </TiltCard>
  );
}

function CtaButton({ cta }: { cta: Cta }) {
  const base =
    "inline-flex w-full items-center justify-center gap-2 rounded-md px-5 py-2.5 text-sm font-medium transition-colors";
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
    <div className="mt-6 flex flex-col">
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
