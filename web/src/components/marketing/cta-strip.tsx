import Link from "next/link";
import { ArrowRight, Star } from "lucide-react";
import { DemoButton } from "./demo-button";
import { TrackedLink, type CTAIntent } from "@/components/analytics/tracked-cta";

type Variant = "demo-first" | "cli-first" | "github-first";

type Props = {
  variant?: Variant;
  demoLabel?: string;
  primaryLabel?: string;
  primaryHref?: string;
  secondaryLabel?: string;
  secondaryHref?: string;
  showGithub?: boolean;
  trackingPrefix?: string;
};

function conversionIntent(href: string): CTAIntent | null {
  if (href.startsWith("mailto:")) return "sales";
  if (href.startsWith("/auth/login")) return "start_free";
  if (href.startsWith("/try")) return "tryout";
  return null;
}

export function CTAStrip({
  variant = "demo-first",
  demoLabel,
  primaryLabel,
  primaryHref,
  secondaryLabel,
  secondaryHref,
  showGithub = true,
  trackingPrefix = "cta-strip",
}: Props) {
  const primaryIntent = primaryHref ? conversionIntent(primaryHref) : null;
  const secondaryIntent = secondaryHref ? conversionIntent(secondaryHref) : null;
  const primaryCTA =
    variant === "demo-first" ? (
      <DemoButton
        label={demoLabel}
        ctaId={`${trackingPrefix}.primary.demo`}
        placement="primary"
      />
    ) : primaryHref ? (
      primaryIntent ? (
        <TrackedLink
          href={primaryHref}
          ctaId={`${trackingPrefix}.primary.${primaryIntent}`}
          intent={primaryIntent}
          placement="primary"
          className="inline-flex items-center justify-center gap-2 rounded-md bg-white px-6 py-3 text-sm font-medium text-[#060606] hover:bg-white/90 transition-colors"
        >
          {primaryLabel ?? "Get started"}
          <ArrowRight className="size-4" />
        </TrackedLink>
      ) : (
        <Link
          href={primaryHref}
          className="inline-flex items-center justify-center gap-2 rounded-md bg-white px-6 py-3 text-sm font-medium text-[#060606] hover:bg-white/90 transition-colors"
        >
          {primaryLabel ?? "Get started"}
          <ArrowRight className="size-4" />
        </Link>
      )
    ) : null;

  const secondaryCTA = secondaryHref ? (
    secondaryIntent ? (
      <TrackedLink
        href={secondaryHref}
        ctaId={`${trackingPrefix}.secondary.${secondaryIntent}`}
        intent={secondaryIntent}
        placement="secondary"
        className="inline-flex items-center justify-center gap-2 rounded-md border border-white/15 bg-white/[0.04] px-6 py-3 text-sm font-medium text-white/80 hover:text-white hover:border-white/30 transition-colors"
      >
        {secondaryLabel ?? "Learn more"}
        <ArrowRight className="size-4" />
      </TrackedLink>
    ) : (
      <Link
        href={secondaryHref}
        className="inline-flex items-center justify-center gap-2 rounded-md border border-white/15 bg-white/[0.04] px-6 py-3 text-sm font-medium text-white/80 hover:text-white hover:border-white/30 transition-colors"
      >
        {secondaryLabel ?? "Learn more"}
        <ArrowRight className="size-4" />
      </Link>
    )
  ) : variant === "demo-first" ? (
    <TrackedLink
      href="/auth/login"
      ctaId={`${trackingPrefix}.secondary.start_free`}
      intent="start_free"
      placement="secondary"
      className="inline-flex items-center justify-center gap-2 rounded-md border border-white/15 bg-white/[0.04] px-6 py-3 text-sm font-medium text-white/80 hover:text-white hover:border-white/30 transition-colors"
    >
      Get started
      <ArrowRight className="size-4" />
    </TrackedLink>
  ) : null;

  return (
    <div className="flex flex-col sm:flex-row sm:flex-wrap sm:items-center gap-3">
      {primaryCTA}
      {secondaryCTA}
      {showGithub ? (
        <a
          href="https://github.com/agentclash/agentclash"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center justify-center gap-2 rounded-md border border-white/[0.08] bg-white/[0.02] px-6 py-3 text-sm font-medium text-white/60 hover:text-white/90 hover:border-white/20 transition-colors"
        >
          <Star className="size-4" />
          GitHub
        </a>
      ) : null}
    </div>
  );
}

export function CLIInstallStrip({
  command = "npm i -g agentclash",
  learnMoreHref = "/docs/getting-started/self-host",
  learnMoreLabel = "self-host",
}: {
  command?: string;
  learnMoreHref?: string;
  learnMoreLabel?: string;
}) {
  return (
    <div className="inline-flex items-center gap-3 rounded-md border border-white/[0.06] bg-white/[0.02] px-4 py-2.5 font-[family-name:var(--font-mono)] text-xs text-white/55">
      <span className="text-white/30 select-none">$</span>
      <code className="text-white/85">{command}</code>
      <span className="text-white/20">·</span>
      <Link
        href={learnMoreHref}
        className="text-white/45 hover:text-white/80 transition-colors"
      >
        {learnMoreLabel}
      </Link>
    </div>
  );
}
