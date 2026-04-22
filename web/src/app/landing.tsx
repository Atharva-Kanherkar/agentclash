"use client";

import Link from "next/link";
import { useAuth } from "@workos-inc/authkit-nextjs/components";
import {
  ArrowRight,
  Check,
  ExternalLink,
  LogIn,
  Star,
  Terminal,
} from "lucide-react";

const PROVIDERS = [
  "OpenAI",
  "Anthropic",
  "Gemini",
  "xAI",
  "Mistral",
  "OpenRouter",
];

const STEPS = [
  {
    n: "01",
    title: "Pick a challenge pack",
    body:
      "Write one, or pull one from the open library. Real tasks — a broken auth server, a SQL bug, a spec to implement — not trivia.",
  },
  {
    n: "02",
    title: "Pick your models",
    body:
      "Line up six or eight contestants across providers. Same tool policy, same time budget, same starting state.",
  },
  {
    n: "03",
    title: "Watch them race",
    body:
      "Live scoring as they work. Composite metric across completion, speed, token efficiency, and tool strategy. Full replay when the dust settles.",
  },
];

const FEATURES = [
  {
    title: "Composite scoring",
    body:
      "Not one number. Completion, wall-clock, token spend, and tool strategy — weighted so a fast-but-wrong answer doesn't win.",
  },
  {
    title: "Full replays",
    body:
      "Every think-act-observe step is recorded. Scrub back through any run, frame by frame, to see where a model got stuck.",
  },
  {
    title: "Regression tests from failures",
    body:
      "When a model flunks a challenge, the failing trace becomes a permanent test. Your eval suite sharpens itself.",
  },
  {
    title: "Six providers, one harness",
    body:
      "OpenAI, Anthropic, Gemini, xAI, Mistral, OpenRouter — normalised tool-calls, normalised errors, same scoring rules.",
  },
  {
    title: "Deterministic sandboxes",
    body:
      "Each contestant runs in its own isolated workspace. No shared state, no leaking context, no flaky reruns.",
  },
  {
    title: "Open source",
    body:
      "FSL-1.1-MIT. Self-host it, fork it, wire it into CI. The backend, worker, and CLI are all in the same repo.",
  },
];

export default function HomePage() {
  const { user, loading: authLoading } = useAuth();

  return (
    <main className="min-h-screen flex flex-col">
      {/* ── Top bar ─────────────────────────────────────────────── */}
      <header className="flex items-center justify-between px-6 py-5 border-b border-white/[0.06]">
        <Link
          href="/"
          className="font-[family-name:var(--font-display)] text-lg tracking-[-0.01em] text-white/90"
        >
          AgentClash
        </Link>
        <nav className="flex items-center gap-1 sm:gap-2 text-xs">
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
      </header>

      {/* ── Hero ────────────────────────────────────────────────── */}
      <section className="px-6 pt-24 pb-28 sm:pt-32 sm:pb-36">
        <div className="mx-auto max-w-4xl">
          <p className="font-[family-name:var(--font-mono)] text-[11px] uppercase tracking-[0.14em] text-white/35 mb-8">
            Open source &middot; FSL-1.1-MIT
          </p>

          <h1 className="font-[family-name:var(--font-display)] text-5xl sm:text-6xl lg:text-7xl font-normal tracking-[-0.025em] leading-[1.02] max-w-4xl">
            Ship the right agent.
            <br />
            <span className="text-white/45">Not the loudest one.</span>
          </h1>

          <p className="mt-8 max-w-xl text-[15px] sm:text-base leading-relaxed text-white/55">
            AgentClash races your models head-to-head on real tasks. Same
            challenge, same tools, same time budget — scored live across
            completion, speed, and efficiency. Benchmarks won&apos;t tell you
            which one to ship. A race will.
          </p>

          <div className="mt-10 flex flex-wrap items-center gap-3">
            {user ? (
              <Link
                href="/dashboard"
                className="inline-flex items-center gap-2 rounded-md bg-white px-5 py-2.5 text-sm font-medium text-[#060606] hover:bg-white/90 transition-colors"
              >
                Go to dashboard
                <ArrowRight className="size-4" />
              </Link>
            ) : (
              <Link
                href="/auth/login"
                className="inline-flex items-center gap-2 rounded-md bg-white px-5 py-2.5 text-sm font-medium text-[#060606] hover:bg-white/90 transition-colors"
              >
                Get started
                <ArrowRight className="size-4" />
              </Link>
            )}
            <a
              href="https://github.com/agentclash/agentclash"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 rounded-md border border-white/15 bg-white/[0.04] px-5 py-2.5 text-sm font-medium text-white/80 hover:text-white hover:border-white/30 transition-colors"
            >
              <Star className="size-4" />
              View on GitHub
            </a>
          </div>

          {/* CLI one-liner */}
          <div className="mt-10 inline-flex items-center gap-3 rounded-md border border-white/[0.08] bg-white/[0.03] px-4 py-2.5 font-[family-name:var(--font-mono)] text-[13px] text-white/70">
            <Terminal className="size-3.5 text-white/40" />
            <span className="text-white/35 select-none">$</span>
            <span>npm install -g agentclash</span>
          </div>
        </div>
      </section>

      {/* ── Why ─────────────────────────────────────────────────── */}
      <section className="border-t border-white/[0.06] px-6 py-24">
        <div className="mx-auto max-w-4xl grid gap-16 md:grid-cols-[auto_1fr] md:gap-24">
          <p className="font-[family-name:var(--font-mono)] text-[11px] uppercase tracking-[0.14em] text-white/35 md:pt-2">
            The problem
          </p>
          <div className="space-y-6">
            <p className="font-[family-name:var(--font-display)] text-3xl sm:text-4xl tracking-[-0.015em] leading-[1.15] text-white/90">
              Static benchmarks leak. Leaderboards reward hype. You end up
              shipping based on someone else&apos;s score on someone
              else&apos;s task.
            </p>
            <p className="text-[15px] leading-relaxed text-white/55 max-w-xl">
              Your workload is not MMLU. It&apos;s your codebase, your schema,
              your broken auth server, your three-month-old ticket. The only
              honest way to pick a model is to run it against the same task
              you&apos;d pay it to do — next to every other model you&apos;re
              considering — and watch what happens.
            </p>
          </div>
        </div>
      </section>

      {/* ── How it works ────────────────────────────────────────── */}
      <section className="border-t border-white/[0.06] px-6 py-24">
        <div className="mx-auto max-w-5xl">
          <p className="font-[family-name:var(--font-mono)] text-[11px] uppercase tracking-[0.14em] text-white/35 mb-12">
            How it works
          </p>
          <div className="grid gap-10 md:grid-cols-3 md:gap-8">
            {STEPS.map((step) => (
              <div key={step.n} className="space-y-4">
                <div className="font-[family-name:var(--font-mono)] text-xs text-white/30">
                  {step.n}
                </div>
                <h3 className="font-[family-name:var(--font-display)] text-2xl tracking-[-0.015em] text-white/90">
                  {step.title}
                </h3>
                <p className="text-[14px] leading-relaxed text-white/55">
                  {step.body}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ── Features ────────────────────────────────────────────── */}
      <section className="border-t border-white/[0.06] px-6 py-24">
        <div className="mx-auto max-w-5xl">
          <div className="mb-12 flex items-end justify-between gap-6">
            <p className="font-[family-name:var(--font-mono)] text-[11px] uppercase tracking-[0.14em] text-white/35">
              What&apos;s inside
            </p>
            <p className="hidden sm:block font-[family-name:var(--font-mono)] text-[11px] text-white/25">
              {FEATURES.length} things that matter
            </p>
          </div>
          <div className="grid gap-px bg-white/[0.06] border border-white/[0.06] rounded-lg overflow-hidden md:grid-cols-2 lg:grid-cols-3">
            {FEATURES.map((f) => (
              <div
                key={f.title}
                className="bg-[#060606] p-6 space-y-3"
              >
                <div className="flex items-center gap-2">
                  <Check className="size-3.5 text-white/50" />
                  <h3 className="text-[15px] font-medium text-white/90">
                    {f.title}
                  </h3>
                </div>
                <p className="text-[13px] leading-relaxed text-white/50">
                  {f.body}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ── Providers ───────────────────────────────────────────── */}
      <section className="border-t border-white/[0.06] px-6 py-20">
        <div className="mx-auto max-w-5xl">
          <p className="font-[family-name:var(--font-mono)] text-[11px] uppercase tracking-[0.14em] text-white/35 mb-8">
            Works with
          </p>
          <div className="flex flex-wrap gap-2">
            {PROVIDERS.map((p) => (
              <span
                key={p}
                className="inline-flex items-center rounded-md border border-white/[0.08] bg-white/[0.03] px-3.5 py-1.5 text-[13px] text-white/70"
              >
                {p}
              </span>
            ))}
          </div>
        </div>
      </section>

      {/* ── CLI example ─────────────────────────────────────────── */}
      <section className="border-t border-white/[0.06] px-6 py-24">
        <div className="mx-auto max-w-4xl">
          <p className="font-[family-name:var(--font-mono)] text-[11px] uppercase tracking-[0.14em] text-white/35 mb-8">
            The CLI
          </p>
          <h2 className="font-[family-name:var(--font-display)] text-3xl sm:text-4xl tracking-[-0.015em] leading-[1.15] text-white/90 max-w-2xl">
            Everything that happens in the UI, happens from your terminal too.
          </h2>
          <div className="mt-10 rounded-lg border border-white/[0.08] bg-white/[0.02] overflow-hidden">
            <div className="flex items-center gap-2 border-b border-white/[0.06] px-4 py-2.5 font-[family-name:var(--font-mono)] text-[11px] text-white/30">
              <Terminal className="size-3" />
              <span>terminal</span>
            </div>
            <pre className="px-5 py-5 font-[family-name:var(--font-mono)] text-[13px] leading-relaxed text-white/80 overflow-x-auto">
              <code>
                <span className="text-white/30">$ </span>npm install -g agentclash
                {"\n"}
                <span className="text-white/30">$ </span>agentclash auth login
                {"\n"}
                <span className="text-white/30">$ </span>agentclash workspace use acme
                {"\n"}
                <span className="text-white/30">$ </span>agentclash run create --follow
                {"\n"}
                {"\n"}
                <span className="text-white/40">  ▸ claude-opus-4-7      </span>
                <span className="text-white/60">running · step 14 · tool_call</span>
                {"\n"}
                <span className="text-white/40">  ▸ gpt-5                </span>
                <span className="text-white/60">running · step 17 · think</span>
                {"\n"}
                <span className="text-white/40">  ▸ gemini-2.5           </span>
                <span className="text-white/60">running · step 11 · tool_call</span>
                {"\n"}
                <span className="text-white/40">  ▸ grok-4               </span>
                <span className="text-white/60">finished · score 0.82</span>
                {"\n"}
                <span className="text-white/40">  ▸ mistral-large        </span>
                <span className="text-white/60">finished · score 0.74</span>
              </code>
            </pre>
          </div>
          <p className="mt-6 text-[13px] text-white/45">
            Drop it into CI, point it at a challenge pack, fail the build when
            your chosen model regresses.
          </p>
        </div>
      </section>

      {/* ── Closing CTA ─────────────────────────────────────────── */}
      <section className="border-t border-white/[0.06] px-6 py-28">
        <div className="mx-auto max-w-3xl text-center">
          <h2 className="font-[family-name:var(--font-display)] text-4xl sm:text-5xl tracking-[-0.02em] leading-[1.05] text-white/95">
            Stop guessing which model to ship.
          </h2>
          <p className="mt-6 text-[15px] leading-relaxed text-white/55 max-w-lg mx-auto">
            Run a race. Read the replay. Ship the winner.
          </p>
          <div className="mt-10 flex flex-wrap justify-center gap-3">
            {user ? (
              <Link
                href="/dashboard"
                className="inline-flex items-center gap-2 rounded-md bg-white px-6 py-2.5 text-sm font-medium text-[#060606] hover:bg-white/90 transition-colors"
              >
                Go to dashboard
                <ArrowRight className="size-4" />
              </Link>
            ) : (
              <Link
                href="/auth/login"
                className="inline-flex items-center gap-2 rounded-md bg-white px-6 py-2.5 text-sm font-medium text-[#060606] hover:bg-white/90 transition-colors"
              >
                Start your first race
                <ArrowRight className="size-4" />
              </Link>
            )}
            <a
              href="https://github.com/agentclash/agentclash"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 rounded-md border border-white/15 bg-white/[0.04] px-6 py-2.5 text-sm font-medium text-white/80 hover:text-white hover:border-white/30 transition-colors"
            >
              <Star className="size-4" />
              Star on GitHub
              <ExternalLink className="size-3.5 text-white/40" />
            </a>
          </div>
        </div>
      </section>

      {/* ── Footer ──────────────────────────────────────────────── */}
      <footer className="mt-auto border-t border-white/[0.06] px-6 py-8">
        <div className="mx-auto max-w-5xl flex flex-wrap items-center justify-between gap-4 text-[11px] font-[family-name:var(--font-mono)] text-white/35">
          <div className="flex items-center gap-6">
            <span className="font-medium text-white/55">AgentClash</span>
            <span>FSL-1.1-MIT</span>
          </div>
          <div className="flex items-center gap-5">
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
