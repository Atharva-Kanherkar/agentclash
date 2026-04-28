"use client";

import { useRef, useState } from "react";
import { ChevronLeft, ChevronRight, Plus } from "lucide-react";
import { SpectraNoise } from "./spectra-noise";

export type UseCase = {
  label: string;
  brief: string;
  verdict: string;
  hueShift: number;
};

export function UseCasesCarousel({ items }: { items: UseCase[] }) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [expanded, setExpanded] = useState<number | null>(null);

  const scrollByCard = (direction: -1 | 1) => {
    const el = scrollRef.current;
    if (!el) return;
    el.scrollBy({
      left: direction * Math.min(420, el.clientWidth * 0.85),
      behavior: "smooth",
    });
  };

  return (
    <div className="relative">
      <div className="absolute right-0 -top-14 hidden gap-2 sm:flex">
        <CarouselButton onClick={() => scrollByCard(-1)} label="Previous">
          <ChevronLeft className="size-4" />
        </CarouselButton>
        <CarouselButton onClick={() => scrollByCard(1)} label="Next">
          <ChevronRight className="size-4" />
        </CarouselButton>
      </div>

      <div
        ref={scrollRef}
        className="flex snap-x snap-mandatory gap-5 overflow-x-auto pb-6 pl-px pr-6 [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden"
      >
        {items.map((u, i) => (
          <UseCaseCard
            key={u.label}
            useCase={u}
            index={i}
            total={items.length}
            expanded={expanded === i}
            onToggle={() => setExpanded(expanded === i ? null : i)}
          />
        ))}
      </div>
    </div>
  );
}

function CarouselButton({
  onClick,
  label,
  children,
}: {
  onClick: () => void;
  label: string;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      className="inline-flex size-9 items-center justify-center rounded-full border border-white/10 bg-white/[0.04] text-white/65 backdrop-blur transition-colors hover:border-white/25 hover:text-white"
    >
      {children}
    </button>
  );
}

function UseCaseCard({
  useCase,
  index,
  total,
  expanded,
  onToggle,
}: {
  useCase: UseCase;
  index: number;
  total: number;
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <article
      className={`group relative flex shrink-0 snap-start flex-col overflow-hidden rounded-2xl glass-card glass-shine transition-[width,box-shadow] duration-300 ${
        expanded
          ? "w-[88vw] max-w-[520px] sm:w-[460px]"
          : "w-[78vw] max-w-[360px] sm:w-[340px]"
      }`}
      style={{ backgroundColor: "rgba(255,255,255,0.04)" }}
    >
      <div className="relative h-24 overflow-hidden rounded-t-2xl">
        <SpectraNoise
          hueShift={useCase.hueShift}
          speed={0.4}
          warpAmount={0.35}
          noiseIntensity={0.08}
          resolutionScale={0.55}
        />
        <div
          aria-hidden
          className="absolute inset-0"
          style={{
            background:
              "linear-gradient(to bottom, rgba(6,6,6,0.0) 0%, rgba(6,6,6,0.35) 65%, rgba(6,6,6,0.85) 100%)",
          }}
        />
        <div className="absolute inset-x-0 bottom-0 z-10 flex items-end justify-between gap-3 px-5 pb-3">
          <p className="font-[family-name:var(--font-mono)] text-[11px] uppercase tracking-[0.22em] text-white/90">
            {useCase.label}
          </p>
          <span className="font-[family-name:var(--font-mono)] text-[10px] tabular-nums tracking-[0.2em] text-white/55">
            {String(index + 1).padStart(2, "0")} / {String(total).padStart(2, "0")}
          </span>
        </div>
      </div>

      <div className="relative z-10 flex flex-1 flex-col p-5 sm:p-6">
        <p
          className={`font-[family-name:var(--font-mono)] text-[13px] leading-[1.6] text-white/85 ${
            expanded ? "" : "line-clamp-3"
          }`}
        >
          <span aria-hidden className="text-white/30">&gt;&nbsp;</span>
          {useCase.brief}
        </p>

        <div
          className={`grid transition-[grid-template-rows,opacity,margin] duration-300 ease-out ${
            expanded ? "mt-5 grid-rows-[1fr] opacity-100" : "mt-0 grid-rows-[0fr] opacity-0"
          }`}
        >
          <div className="overflow-hidden">
            <div className="border-t border-white/10 pt-4">
              <p className="font-[family-name:var(--font-mono)] text-[10px] uppercase tracking-[0.2em] text-white/40">
                Verdict
              </p>
              <p className="mt-1.5 font-[family-name:var(--font-mono)] text-[12px] leading-[1.55] text-white/70">
                {useCase.verdict}
              </p>
            </div>
          </div>
        </div>

        <button
          type="button"
          onClick={onToggle}
          aria-expanded={expanded}
          className="mt-auto inline-flex items-center gap-2 self-start pt-5 font-[family-name:var(--font-mono)] text-[11px] uppercase tracking-[0.18em] text-white/45 transition-colors hover:text-white/85"
        >
          <Plus
            className={`size-3 transition-transform duration-300 ${
              expanded ? "rotate-45" : ""
            }`}
          />
          {expanded ? "Collapse" : "Verdict"}
        </button>
      </div>
    </article>
  );
}
