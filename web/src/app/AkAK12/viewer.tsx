"use client";

import { useMemo, useState } from "react";
import dynamic from "next/dynamic";
import { CENSUS_DATA, RELIGIONS, type CensusRow } from "./data";

const GlobeScene = dynamic(
  () => import("./globe-scene").then((m) => m.GlobeScene),
  { ssr: false },
);

function formatPopulation(millions: number): string {
  if (millions >= 1000) {
    return `${(millions / 1000).toFixed(2)} billion`;
  }
  return `${millions.toLocaleString()} million`;
}

function StackedBar({ row }: { row: CensusRow }) {
  return (
    <div className="w-full">
      <div className="relative flex h-10 w-full overflow-hidden rounded-full border border-white/10 bg-white/5 shadow-inner">
        {RELIGIONS.map((r) => {
          const pct = row[r.key];
          if (pct <= 0) return null;
          return (
            <div
              key={r.key}
              className="h-full transition-[width] duration-700 ease-out"
              style={{
                width: `${pct}%`,
                background: r.color,
                boxShadow:
                  r.key === "hindu" || r.key === "muslim"
                    ? `0 0 18px 0 ${r.color}66`
                    : undefined,
              }}
              title={`${r.label}: ${pct.toFixed(2)}%`}
            />
          );
        })}
      </div>
    </div>
  );
}

function ReligionCard({
  religion,
  pct,
}: {
  religion: (typeof RELIGIONS)[number];
  pct: number;
}) {
  return (
    <div className="rounded-lg border border-white/10 bg-white/[0.03] p-3 backdrop-blur-sm transition-colors hover:bg-white/[0.06]">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span
            className="h-3 w-3 shrink-0 rounded-full"
            style={{ background: religion.color, boxShadow: `0 0 10px ${religion.color}99` }}
          />
          <span className="text-sm font-medium text-white">
            {religion.label}
          </span>
        </div>
        <span
          className="font-mono text-sm tabular-nums text-white tracking-tight"
          style={{ color: religion.color }}
        >
          {pct.toFixed(2)}%
        </span>
      </div>
      <p className="mt-1.5 pl-5 text-[11px] leading-snug text-white/50">
        {religion.description}
      </p>
    </div>
  );
}

export function Viewer() {
  const [index, setIndex] = useState(CENSUS_DATA.length - 1);
  const row = CENSUS_DATA[index];

  const sortedReligions = useMemo(
    () =>
      [...RELIGIONS].sort((a, b) => row[b.key] - row[a.key]),
    [row],
  );

  const scopeLabel =
    row.scope === "british-india"
      ? "British India (undivided subcontinent)"
      : "Republic of India";

  return (
    <div className="relative min-h-screen w-full overflow-hidden bg-[#05060f] text-white">
      {/* ambient gradient background */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0"
        style={{
          background:
            "radial-gradient(60% 50% at 30% 20%, rgba(255,153,51,0.08), transparent 60%)," +
            "radial-gradient(50% 40% at 80% 80%, rgba(19,136,8,0.10), transparent 60%)," +
            "radial-gradient(80% 60% at 50% 50%, rgba(40,60,180,0.18), transparent 70%)",
        }}
      />

      <div className="relative mx-auto flex min-h-screen max-w-7xl flex-col gap-6 px-6 py-10 lg:py-14">
        <header className="flex flex-col gap-3">
          <p className="font-mono text-xs uppercase tracking-[0.2em] text-white/50">
            India · Religious Demographics · 1881 — 2011
          </p>
          <h1 className="text-3xl font-medium leading-tight tracking-tight md:text-5xl">
            One country, many faiths.
          </h1>
          <p className="max-w-2xl text-sm leading-relaxed text-white/65 md:text-base">
            India has been home to people of every major religion for centuries.
            This visualisation walks through 130 years of decennial census data
            from the Office of the Registrar General of India — the same
            primary source used by demographers and policy researchers — to show
            how that composition has changed, and how it has remained
            remarkably plural.
          </p>
        </header>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1.1fr_1fr]">
          {/* Globe */}
          <div className="relative h-[420px] overflow-hidden rounded-2xl border border-white/10 bg-black/40 md:h-[520px]">
            <GlobeScene />
            <div className="pointer-events-none absolute left-4 top-4 rounded-md border border-white/10 bg-black/40 px-3 py-1.5 backdrop-blur-md">
              <span className="font-mono text-[10px] uppercase tracking-wider text-white/60">
                Marker · Geographic centre of India
              </span>
            </div>
          </div>

          {/* Stats */}
          <div className="flex flex-col gap-5 rounded-2xl border border-white/10 bg-white/[0.02] p-6 backdrop-blur-sm">
            <div className="flex items-baseline justify-between gap-4">
              <div>
                <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-white/50">
                  Census of
                </div>
                <div className="mt-1 font-serif text-5xl font-light tracking-tight">
                  {row.year}
                </div>
              </div>
              <div className="text-right">
                <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-white/50">
                  Population
                </div>
                <div className="mt-1 font-mono text-lg tabular-nums text-white">
                  {formatPopulation(row.populationMillions)}
                </div>
              </div>
            </div>

            <div className="flex flex-wrap gap-2">
              <span className="rounded-full border border-white/10 bg-white/5 px-2.5 py-0.5 font-mono text-[10px] uppercase tracking-wider text-white/70">
                {scopeLabel}
              </span>
            </div>

            <StackedBar row={row} />

            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              {sortedReligions.map((r) => (
                <ReligionCard key={r.key} religion={r} pct={row[r.key]} />
              ))}
            </div>

            {row.note && (
              <p className="rounded-md border border-amber-300/20 bg-amber-300/[0.04] p-3 text-xs italic leading-relaxed text-amber-100/85">
                {row.note}
              </p>
            )}
          </div>
        </div>

        {/* Slider */}
        <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-5 backdrop-blur-sm">
          <div className="flex items-center justify-between text-xs text-white/55">
            <span className="font-mono uppercase tracking-wider">
              Drag to travel through the censuses
            </span>
            <span className="font-mono tabular-nums text-white/70">
              {index + 1} / {CENSUS_DATA.length}
            </span>
          </div>

          <input
            type="range"
            min={0}
            max={CENSUS_DATA.length - 1}
            step={1}
            value={index}
            onChange={(e) => setIndex(parseInt(e.target.value, 10))}
            aria-label="Census year"
            className="mt-4 h-2 w-full cursor-pointer appearance-none rounded-full bg-white/10 accent-orange-400 [&::-webkit-slider-thumb]:h-5 [&::-webkit-slider-thumb]:w-5 [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-orange-400 [&::-webkit-slider-thumb]:shadow-[0_0_18px_rgba(255,153,51,0.7)]"
          />

          <div className="mt-3 flex flex-wrap gap-1.5">
            {CENSUS_DATA.map((d, i) => (
              <button
                key={d.year}
                type="button"
                onClick={() => setIndex(i)}
                className={
                  "rounded-md px-2 py-1 font-mono text-[11px] tabular-nums transition-colors " +
                  (i === index
                    ? "bg-orange-400 text-black"
                    : d.scope === "british-india"
                      ? "border border-white/10 bg-white/[0.04] text-white/60 hover:bg-white/[0.08]"
                      : "border border-white/10 bg-white/[0.04] text-white/80 hover:bg-white/[0.08]")
                }
                aria-label={`Jump to census year ${d.year}`}
              >
                {d.year}
              </button>
            ))}
          </div>
        </div>

        <footer className="space-y-2 border-t border-white/10 pt-5 text-xs leading-relaxed text-white/50">
          <p>
            <span className="text-white/70">Data source.</span> Census of India,
            decennial reports 1881-2011 (Office of the Registrar General &amp;
            Census Commissioner, India). Pre-1947 figures cover undivided
            British India; 1951-2011 figures cover the Republic of India only.
            The 2021 census has been postponed.
          </p>
          <p>
            <span className="text-white/70">Methodology note.</span> The 1947
            Partition redrew the subcontinent and is the dominant cause of the
            shift between 1941 and 1951. The 1981 census omitted Assam and the
            1991 census omitted Jammu &amp; Kashmir; both are pro-rated. &ldquo;Other&rdquo;
            includes tribal religions, Parsis, Jews, Baha&rsquo;is, the
            religiously unaffiliated, and respondents who declined to state.
          </p>
          <p className="pt-2 text-white/40">
            Built as an open data illustration of India&rsquo;s religious
            plurality. No personal data, no tracking, no claims beyond the
            primary census record.
          </p>
        </footer>
      </div>
    </div>
  );
}
