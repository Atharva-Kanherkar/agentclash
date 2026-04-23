"use client";

import type React from "react";
import Link from "next/link";
import { useAuth } from "@workos-inc/authkit-nextjs/components";
import { ArrowRight, LogIn, Star } from "lucide-react";

function ClashMark({ className = "" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 512 512"
      className={className}
      aria-label="AgentClash"
      role="img"
    >
      <polygon points="80,180 240,256 80,332" fill="#ffffff" opacity="0.95" />
      <polygon points="432,180 272,256 432,332" fill="#ffffff" opacity="0.5" />
    </svg>
  );
}

function BetaPill() {
  return (
    <span className="inline-flex items-center gap-2 rounded-full border border-white/[0.14] bg-white/[0.04] px-3.5 py-1.5 text-[11px] font-[family-name:var(--font-mono)] uppercase tracking-[0.22em] text-white/70">
      <span className="relative flex size-1.5">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-white opacity-60" />
        <span className="relative inline-flex size-1.5 rounded-full bg-white" />
      </span>
      Released in beta
    </span>
  );
}

/* ──────────────────────────────────────────────────────────────────
 * Hero animation — radial constellation.
 * A pulsing central engine radiates light streaks outward to eight
 * feature nodes arranged on a circle. Continuous, rotating.
 * ────────────────────────────────────────────────────────────────── */
function ShippingConstellation() {
  const CX = 300;
  const CY = 300;
  const R = 220;
  const CENTER_R = 40;
  const NODE_R = 15;
  const DURATION = 1.4;
  const COUNT = 8;

  const nodes = Array.from({ length: COUNT }, (_, i) => {
    const angle = (i / COUNT) * Math.PI * 2 - Math.PI / 2;
    return {
      i,
      x: CX + R * Math.cos(angle),
      y: CY + R * Math.sin(angle),
      cos: Math.cos(angle),
      sin: Math.sin(angle),
    };
  });

  const paths = nodes.map((n) => {
    const startX = CX + CENTER_R * n.cos;
    const startY = CY + CENTER_R * n.sin;
    const endX = CX + (R - NODE_R - 2) * n.cos;
    const endY = CY + (R - NODE_R - 2) * n.sin;
    return `M ${startX.toFixed(1)} ${startY.toFixed(1)} L ${endX.toFixed(1)} ${endY.toFixed(1)}`;
  });

  return (
    <div className="flex items-center justify-center py-6" aria-hidden>
      <svg viewBox="0 0 600 600" className="w-full max-w-[560px]" focusable="false">
        <defs>
          <filter id="center-glow" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur stdDeviation="8" result="blur" />
            <feComposite in="SourceGraphic" in2="blur" operator="over" />
          </filter>
          <radialGradient id="center-fill">
            <stop offset="0%" stopColor="rgba(255,255,255,0.28)" />
            <stop offset="70%" stopColor="rgba(255,255,255,0.04)" />
            <stop offset="100%" stopColor="rgba(255,255,255,0)" />
          </radialGradient>
        </defs>

        {/* Outer orbit guide */}
        <circle
          cx={CX}
          cy={CY}
          r={R}
          fill="none"
          stroke="rgba(255,255,255,0.06)"
          strokeWidth="1"
          strokeDasharray="2 6"
        />

        {/* Static connector lines */}
        {paths.map((d, i) => (
          <path
            key={`line-${i}`}
            d={d}
            fill="none"
            stroke="rgba(255,255,255,0.12)"
            strokeWidth="1"
          />
        ))}

        {/* Central engine — pulsing */}
        <circle
          cx={CX}
          cy={CY}
          r={CENTER_R + 18}
          fill="url(#center-fill)"
          className="animate-results-glow"
        />
        <circle
          cx={CX}
          cy={CY}
          r={CENTER_R}
          fill="rgba(255,255,255,0.04)"
          stroke="rgba(255,255,255,0.55)"
          strokeWidth="1.4"
          filter="url(#center-glow)"
        />
        <circle cx={CX} cy={CY} r="5" fill="white" opacity="0.9" />

        {/* Outer nodes */}
        {nodes.map((n) => (
          <g key={`node-${n.i}`}>
            <circle
              cx={n.x}
              cy={n.y}
              r={NODE_R + 6}
              fill="none"
              stroke="rgba(255,255,255,0.08)"
              strokeWidth="1"
            />
            <circle
              cx={n.x}
              cy={n.y}
              r={NODE_R}
              fill="#060606"
              stroke="rgba(255,255,255,0.32)"
              strokeWidth="1.2"
            />
            <circle
              cx={n.x}
              cy={n.y}
              r="3"
              fill="white"
              opacity="0.55"
            />
          </g>
        ))}

        {/* Light streaks radiating outward */}
        {paths.map((d, i) => (
          <line
            key={`streak-${i}`}
            x1="-7"
            y1="0"
            x2="7"
            y2="0"
            stroke="white"
            strokeWidth="2.2"
            strokeLinecap="round"
            className="animate-light-streak"
            style={{
              offsetPath: `path('${d}')`,
              animationDelay: `${(-(i / COUNT) * DURATION).toFixed(2)}s`,
            }}
          />
        ))}
      </svg>
    </div>
  );
}

/* ──────────────────────────────────────────────────────────────────
 * Eight feature glyphs.
 * Small 48-viewBox SVGs in the same editorial grammar as the landing.
 * ────────────────────────────────────────────────────────────────── */
type GlyphProps = { className?: string };
const GLYPH_BASE =
  "size-7 text-white/90 shrink-0";

function ArtifactsGlyph({ className = "" }: GlyphProps) {
  return (
    <svg
      viewBox="0 0 48 48"
      className={`${GLYPH_BASE} ${className}`}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      aria-hidden
    >
      <rect x="10" y="9" width="20" height="26" rx="1" opacity="0.55" />
      <rect x="14" y="13" width="20" height="26" rx="1" opacity="0.8" />
      <line x1="18" y1="20" x2="30" y2="20" opacity="0.6" />
      <line x1="18" y1="25" x2="30" y2="25" opacity="0.6" />
      <line x1="18" y1="30" x2="26" y2="30" opacity="0.6" />
    </svg>
  );
}

function RagGlyph({ className = "" }: GlyphProps) {
  return (
    <svg
      viewBox="0 0 48 48"
      className={`${GLYPH_BASE} ${className}`}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      aria-hidden
    >
      <circle cx="12" cy="14" r="3" />
      <circle cx="12" cy="24" r="3" />
      <circle cx="12" cy="34" r="3" />
      <circle cx="36" cy="24" r="4.5" />
      <line x1="15" y1="14" x2="33" y2="22" opacity="0.5" />
      <line x1="15" y1="24" x2="31" y2="24" opacity="0.55" />
      <line x1="15" y1="34" x2="33" y2="26" opacity="0.5" />
    </svg>
  );
}

function KeysGlyph({ className = "" }: GlyphProps) {
  return (
    <svg
      viewBox="0 0 48 48"
      className={`${GLYPH_BASE} ${className}`}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      aria-hidden
    >
      <path d="M 24 6 L 38 12 V 22 C 38 32 32 38 24 42 C 16 38 10 32 10 22 V 12 Z" />
      <circle cx="24" cy="22" r="3.5" opacity="0.85" />
      <line x1="24" y1="25.5" x2="24" y2="32" strokeWidth="1.6" opacity="0.85" />
    </svg>
  );
}

function TracingGlyph({ className = "" }: GlyphProps) {
  return (
    <svg
      viewBox="0 0 48 48"
      className={`${GLYPH_BASE} ${className}`}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      aria-hidden
    >
      <line x1="8" y1="12" x2="26" y2="12" />
      <line x1="12" y1="20" x2="32" y2="20" />
      <line x1="16" y1="28" x2="38" y2="28" />
      <line x1="12" y1="36" x2="30" y2="36" />
      <circle cx="26" cy="12" r="2" fill="currentColor" opacity="0.7" stroke="none" />
      <circle cx="32" cy="20" r="2" fill="currentColor" opacity="0.9" stroke="none" />
      <circle cx="38" cy="28" r="2" fill="currentColor" opacity="0.75" stroke="none" />
      <circle cx="30" cy="36" r="2" fill="currentColor" opacity="0.6" stroke="none" />
    </svg>
  );
}

function KnowledgeGlyph({ className = "" }: GlyphProps) {
  return (
    <svg
      viewBox="0 0 48 48"
      className={`${GLYPH_BASE} ${className}`}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      aria-hidden
    >
      <path d="M 24 10 C 18 10 12 11 8 13 V 37 C 12 35 18 34 24 34 C 30 34 36 35 40 37 V 13 C 36 11 30 10 24 10 Z" opacity="0.85" />
      <line x1="24" y1="10" x2="24" y2="34" opacity="0.45" />
      <line x1="12" y1="17" x2="20" y2="17" opacity="0.35" />
      <line x1="12" y1="22" x2="20" y2="22" opacity="0.35" />
      <line x1="28" y1="17" x2="36" y2="17" opacity="0.35" />
      <line x1="28" y1="22" x2="36" y2="22" opacity="0.35" />
    </svg>
  );
}

function RegressionGlyph({ className = "" }: GlyphProps) {
  return (
    <svg
      viewBox="0 0 48 48"
      className={`${GLYPH_BASE} ${className}`}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      aria-hidden
    >
      <path d="M 40 24 A 16 16 0 1 1 13 13" />
      <polyline points="13,6 13,13 20,13" strokeLinejoin="round" />
      <path d="M 17 22 L 22 27 L 32 17" strokeWidth="1.7" />
    </svg>
  );
}

function CompareGlyph({ className = "" }: GlyphProps) {
  return (
    <svg
      viewBox="0 0 48 48"
      className={`${GLYPH_BASE} ${className}`}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      aria-hidden
    >
      <rect x="7" y="10" width="14" height="28" rx="1" opacity="0.85" />
      <rect x="27" y="18" width="14" height="20" rx="1" opacity="0.7" />
      <line x1="10" y1="16" x2="18" y2="16" opacity="0.5" />
      <line x1="10" y1="22" x2="18" y2="22" opacity="0.5" />
      <line x1="30" y1="24" x2="38" y2="24" opacity="0.5" />
      <line x1="30" y1="30" x2="38" y2="30" opacity="0.5" />
    </svg>
  );
}

function CiCdGlyph({ className = "" }: GlyphProps) {
  return (
    <svg
      viewBox="0 0 48 48"
      className={`${GLYPH_BASE} ${className}`}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      aria-hidden
    >
      <circle cx="10" cy="24" r="4" />
      <circle cx="24" cy="24" r="4" opacity="0.85" />
      <circle cx="38" cy="24" r="4" />
      <line x1="14" y1="24" x2="20" y2="24" opacity="0.6" />
      <line x1="28" y1="24" x2="34" y2="24" opacity="0.6" />
      <path d="M 24 14 V 10 A 2 2 0 0 0 22 8" opacity="0.45" />
      <path d="M 24 34 V 38 A 2 2 0 0 1 26 40" opacity="0.45" />
    </svg>
  );
}

const FEATURES: Array<{
  label: string;
  title: string;
  body: string;
  glyph: React.ReactNode;
}> = [
  {
    label: "Artifacts",
    title: "Every run is a paper trail.",
    body:
      "Logs, output files, scorecards, diffs, agent manifests — everything an agent produced, sealed per run, addressable by ID. Inspect in the UI, stream from the API, or pipe to your own storage.",
    glyph: <ArtifactsGlyph />,
  },
  {
    label: "RAG testing",
    title: "Retrieval and generation, judged together.",
    body:
      "Feed your corpus. Watch what each model retrieved before it answered. Grounding, faithfulness, and citation coverage scored as first-class axes — not left as an afterthought of the answer.",
    glyph: <RagGlyph />,
  },
  {
    label: "Key security",
    title: "The agent never sees your keys.",
    body:
      "API keys, DB creds, OAuth tokens live in a scoped secret vault. Tools inject them into the sandbox at call time — never into the prompt, never into the trace, never into the replay. The agent uses the capability; it doesn't know the secret.",
    glyph: <KeysGlyph />,
  },
  {
    label: "Tracing",
    title: "Tracing like never before.",
    body:
      "OpenTelemetry-native. Every think, every tool call, every observation, every byte — with span trees, causal chains, per-step cost and latency. Not a transcript dump. A forensic record.",
    glyph: <TracingGlyph />,
  },
  {
    label: "Knowledge sources",
    title: "Your docs, wired in.",
    body:
      "Attach PDFs, wikis, Notion, codebases, your own APIs. Agents query them through a shared retriever with provenance on every fact — so when a model cites something, you can see exactly where it came from.",
    glyph: <KnowledgeGlyph />,
  },
  {
    label: "Regression suites",
    title: "Every failure becomes a test.",
    body:
      "When a model flunks, the failing trace freezes into a permanent regression. Next week's race replays it. The one after does too. The suite sharpens itself — by the time a new model arrives, it walks into a track paved by every mistake the last one made.",
    glyph: <RegressionGlyph />,
  },
  {
    label: "Comparison",
    title: "Diff two races, side by side.",
    body:
      "Same challenge, new model, or same model with a new prompt. See exactly what moved: completion, cost, latency, tool trajectory, scorecard axes. No guessing which upgrade mattered.",
    glyph: <CompareGlyph />,
  },
  {
    label: "CI/CD",
    title: "Gate the merge on the race.",
    body:
      "Trigger races from GitHub Actions, a webhook, or the CLI. Fail the build when your agent regresses on the scorecard you care about. Eval moves from a dashboard you visit to a check that blocks bad code.",
    glyph: <CiCdGlyph />,
  },
];

export default function FeaturesPage() {
  const { user, loading: authLoading } = useAuth();

  return (
    <main className="main min-h-screen flex flex-col">
      {/* ── Header ──────────────────────────────────────────────── */}
      <header className="px-8 sm:px-12 py-6 border-b border-white/[0.06]">
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
          <nav className="flex items-center gap-1 sm:gap-2 text-xs">
            <Link
              href="/features"
              className="inline-flex px-3 py-1.5 text-white/85 transition-colors"
            >
              Features
            </Link>
            <Link
              href="/docs"
              className="inline-flex px-3 py-1.5 text-white/55 hover:text-white/85 transition-colors"
            >
              Docs
            </Link>
            <Link
              href="/blog"
              className="hidden sm:inline-flex px-3 py-1.5 text-white/55 hover:text-white/85 transition-colors"
            >
              Blog
            </Link>
            <a
              href="https://github.com/agentclash/agentclash"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 rounded-md border border-white/[0.08] bg-white/[0.03] px-3 py-1.5 text-white/60 hover:text-white/85 hover:border-white/15 transition-colors"
            >
              <Star className="size-3.5" />
              GitHub
            </a>
            {authLoading ? (
              <span className="inline-flex h-[30px] w-[88px] rounded-md border border-white/[0.08] bg-white/[0.04]" />
            ) : user ? (
              <Link
                href="/dashboard"
                className="inline-flex items-center gap-1.5 rounded-md bg-white px-3 py-1.5 font-medium text-[#060606] hover:bg-white/90 transition-colors"
              >
                Dashboard
                <ArrowRight className="size-3" />
              </Link>
            ) : (
              <Link
                href="/auth/login"
                className="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.04] px-3 py-1.5 text-white/75 hover:text-white hover:border-white/25 transition-colors"
              >
                <LogIn className="size-3.5" />
                Sign in
              </Link>
            )}
          </nav>
        </div>
      </header>

      {/* ── Hero ────────────────────────────────────────────────── */}
      <section className="px-8 sm:px-12 pt-28 pb-16 sm:pt-40 sm:pb-24">
        <div className="mx-auto max-w-[1440px] grid gap-16 md:grid-cols-[1.3fr_1fr] md:gap-20 items-center">
          <div>
            <BetaPill />

            <h1 className="mt-8 font-[family-name:var(--font-display)] font-normal tracking-[-0.04em] leading-[0.95] text-[clamp(2.75rem,6.5vw,6.5rem)] max-w-[18ch]">
              We&apos;re shipping more
              <br />
              <span className="text-white/40">than you think.</span>
            </h1>

            <p className="mt-10 max-w-[48ch] text-lg sm:text-xl leading-[1.5] text-white/55">
              The race engine is the visible part. Under the hood sit eight
              capabilities most teams quietly want from an eval platform but
              rarely get in one place. Trust us — or better, scroll.
            </p>

            <div className="mt-10 flex flex-col sm:flex-row sm:flex-wrap gap-3">
              <Link
                href={user ? "/dashboard" : "/auth/login"}
                className="inline-flex items-center justify-center gap-2 rounded-md bg-white px-6 py-3 text-sm font-medium text-[#060606] hover:bg-white/90 transition-colors"
              >
                {user ? "Go to dashboard" : "Get started"}
                <ArrowRight className="size-4" />
              </Link>
              <Link
                href="/docs"
                className="inline-flex items-center justify-center gap-2 rounded-md border border-white/15 bg-white/[0.04] px-6 py-3 text-sm font-medium text-white/80 hover:text-white hover:border-white/30 transition-colors"
              >
                Read the docs
                <ArrowRight className="size-4" />
              </Link>
            </div>
          </div>

          <div>
            <ShippingConstellation />
          </div>
        </div>
      </section>

      {/* ── Feature grid ────────────────────────────────────────── */}
      <section className="border-t border-white/[0.06] px-8 sm:px-12 py-24 sm:py-32">
        <div className="mx-auto max-w-[1440px]">
          <div className="flex flex-col gap-10 md:flex-row md:items-end md:justify-between md:gap-16">
            <h2 className="font-[family-name:var(--font-display)] font-normal tracking-[-0.03em] leading-[1.02] text-[clamp(2.25rem,5vw,4.5rem)] max-w-[22ch]">
              Eight capabilities.
              <br />
              <span className="text-white/40">One engine.</span>
            </h2>
            <p className="max-w-[42ch] text-base leading-[1.6] text-white/50">
              Each of these is a feature we would normally stretch into its
              own landing section. They all ship in beta today. More landing
              every week.
            </p>
          </div>

          <ul className="mt-20 grid grid-cols-1 gap-px border-y border-white/[0.06] bg-white/[0.06] sm:grid-cols-2 lg:grid-cols-4">
            {FEATURES.map((feature) => (
              <li
                key={feature.label}
                className="group relative flex flex-col bg-[#060606] px-8 py-12 transition-colors hover:bg-white/[0.015]"
              >
                <div className="inline-flex size-12 items-center justify-center rounded-full border border-white/[0.12] bg-white/[0.02] transition-colors group-hover:border-white/25">
                  {feature.glyph}
                </div>

                <p className="mt-8 text-[11px] font-[family-name:var(--font-mono)] uppercase tracking-[0.2em] text-white/40">
                  {feature.label}
                </p>

                <h3 className="mt-3 font-[family-name:var(--font-display)] text-2xl leading-[1.15] tracking-[-0.02em] text-white/95">
                  {feature.title}
                </h3>

                <p className="mt-4 text-[14px] leading-[1.65] text-white/55">
                  {feature.body}
                </p>
              </li>
            ))}
          </ul>

          <p className="mt-10 text-sm text-white/40">
            Want something that isn&apos;t here?{" "}
            <a
              href="https://github.com/agentclash/agentclash/issues/new"
              target="_blank"
              rel="noopener noreferrer"
              className="text-white/70 underline decoration-white/20 underline-offset-4 transition-colors hover:text-white hover:decoration-white/50"
            >
              Open an issue
            </a>
            . We read every one.
          </p>
        </div>
      </section>

      {/* ── Closing CTA ─────────────────────────────────────────── */}
      <section className="border-t border-white/[0.06] px-8 sm:px-12 py-28 sm:py-40">
        <div className="mx-auto max-w-[1440px]">
          <h2 className="font-[family-name:var(--font-display)] font-normal tracking-[-0.04em] leading-[0.95] text-[clamp(2.5rem,5.5vw,5.5rem)] max-w-[18ch]">
            All of it, free for beta.
          </h2>
          <p className="mt-8 max-w-[46ch] text-lg leading-[1.6] text-white/55">
            Spin up a workspace, bring your own keys, race your first pair of
            models in under a minute. If something&apos;s broken, tell us — we
            fix fast.
          </p>

          <div className="mt-10 flex flex-col sm:flex-row sm:flex-wrap gap-3">
            <Link
              href={user ? "/dashboard" : "/auth/login"}
              className="inline-flex items-center justify-center gap-2 rounded-md bg-white px-7 py-3 text-sm font-medium text-[#060606] hover:bg-white/90 transition-colors"
            >
              {user ? "Go to dashboard" : "Start racing"}
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

      {/* ── Footer ──────────────────────────────────────────────── */}
      <footer className="mt-auto border-t border-white/[0.06] px-8 sm:px-12 py-10">
        <div className="mx-auto max-w-[1440px] flex flex-wrap items-center justify-between gap-4 text-[11px] font-[family-name:var(--font-mono)] text-white/35">
          <div className="flex items-center gap-6">
            <span className="font-medium text-white/55">AgentClash</span>
            <span className="text-white/40">Beta</span>
          </div>
          <div className="flex items-center gap-5">
            <Link href="/features" className="hover:text-white/70 transition-colors">
              Features
            </Link>
            <Link href="/blog" className="hover:text-white/70 transition-colors">
              Blog
            </Link>
            <Link href="/team" className="hover:text-white/70 transition-colors">
              Team
            </Link>
            <a
              href="https://github.com/agentclash/agentclash"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-white/70 transition-colors"
            >
              GitHub
            </a>
          </div>
        </div>
      </footer>
    </main>
  );
}
