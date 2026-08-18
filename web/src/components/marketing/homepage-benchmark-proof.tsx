import type { ReactNode } from "react";

import { BenchmarkScoreboard } from "@/components/marketing/benchmark-scoreboard";
import type { HomepageBenchmarkData } from "@/lib/homepage-benchmark";

export function HomepageBenchmarkProof({
  benchmark,
  actions,
}: {
  benchmark?: HomepageBenchmarkData;
  actions?: ReactNode;
}) {
  if (!benchmark || benchmark.results.length === 0) return null;

  return (
    <section
      aria-labelledby="homepage-benchmark-heading"
      className="border-t border-white/[0.06] px-8 py-24 sm:px-12 sm:py-32"
    >
      <div className="mx-auto grid max-w-[1440px] gap-12 lg:grid-cols-[0.72fr_1.28fr] lg:items-center lg:gap-16">
        <div>
          <p className="font-mono text-[11px] uppercase tracking-[0.14em] text-white/40">
            Measured benchmark
          </p>
          <h2
            id="homepage-benchmark-heading"
            className="mt-4 max-w-[18ch] font-sans text-[clamp(2.25rem,5vw,4.5rem)] font-semibold leading-[1.02] tracking-tight"
          >
            A real run. A result you can reproduce.
          </h2>
          <p className="mt-7 max-w-[48ch] text-base font-medium leading-7 text-white/85 sm:text-lg">
            {benchmark.title}
          </p>
          <p className="mt-4 max-w-[52ch] text-base leading-7 text-white/55">
            {benchmark.verdict}
          </p>
          {benchmark.challengePack ? (
            <p className="mt-5 font-mono text-xs leading-6 text-white/40">
              Challenge: {benchmark.challengePack}
            </p>
          ) : null}
          {actions ? (
            <div className="mt-8 flex flex-wrap items-center gap-x-6 gap-y-3 text-sm">
              {actions}
            </div>
          ) : null}
        </div>

        <BenchmarkScoreboard
          results={benchmark.results}
          featuredModel={benchmark.featuredModel}
        />
      </div>
    </section>
  );
}
