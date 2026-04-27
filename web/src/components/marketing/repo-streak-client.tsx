"use client";

import { useMemo, useRef, useState } from "react";
import type { StreakData, StreakDay } from "@/lib/github-streak";

type HoverState = {
  day: StreakDay;
  x: number; // viewport coords
  y: number;
} | null;

const CELL = 12;
const GAP = 3;
const COL_STRIDE = CELL + GAP;
const ROW_STRIDE = CELL + GAP;

const MONTH_LABELS = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
];

function bucketFor(count: number): 0 | 1 | 2 | 3 | 4 {
  if (count === 0) return 0;
  if (count <= 2) return 1;
  if (count <= 5) return 2;
  if (count <= 10) return 3;
  return 4;
}

// Bucket fills tuned against the page's near-black background and the existing
// luminous-grid blue. Lowest level keeps the cell visible without competing
// with the rest of the page.
const FILL = ["#171a22", "#1f3a5a", "#2e6aa3", "#4f9bdb", "#7eb8e6"] as const;

const TIER_DOT: Record<string, string> = {
  breaking: "#ff8a4c",
  feat: "#7eb8e6",
  fix: "#cfd5e0",
};

export function RepoStreakClient({ data }: { data: StreakData }) {
  const [hover, setHover] = useState<HoverState>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);

  const { weeks, monthMarks, totalWeeks } = useMemo(() => {
    // Pad to start on a Sunday so columns are aligned weeks.
    const first = new Date(`${data.days[0].date}T00:00:00Z`);
    const padDays = first.getUTCDay(); // 0=Sun
    const padded: (StreakDay | null)[] = [
      ...Array.from({ length: padDays }, () => null),
      ...data.days,
    ];
    const weeks: (StreakDay | null)[][] = [];
    for (let i = 0; i < padded.length; i += 7) {
      weeks.push(padded.slice(i, i + 7));
    }
    // Tail-pad final column to 7 cells so the grid is rectangular.
    const last = weeks[weeks.length - 1];
    while (last.length < 7) last.push(null);

    const monthMarks: { x: number; label: string }[] = [];
    let lastMonth = -1;
    weeks.forEach((week, col) => {
      const firstReal = week.find((d): d is StreakDay => d !== null);
      if (!firstReal) return;
      const month = new Date(`${firstReal.date}T00:00:00Z`).getUTCMonth();
      if (month !== lastMonth) {
        monthMarks.push({ x: col * COL_STRIDE, label: MONTH_LABELS[month] });
        lastMonth = month;
      }
    });

    return { weeks, monthMarks, totalWeeks: weeks.length };
  }, [data]);

  const width = totalWeeks * COL_STRIDE - GAP;
  const height = 7 * ROW_STRIDE - GAP;

  return (
    <div className="relative">
      <div className="mb-6 flex flex-wrap items-baseline gap-x-8 gap-y-2 text-sm">
        <div className="font-mono text-[0.7rem] uppercase tracking-[0.22em] text-white/40">
          {data.owner}/{data.repo}
        </div>
        <Stat label="commits / yr" value={data.totalCommits.toLocaleString()} />
        <Stat label="active days" value={`${data.activeDays}`} />
        <Stat label="longest streak" value={`${data.longestStreak}d`} />
        <Stat
          label="busiest"
          value={
            data.busiest.count > 0
              ? `${data.busiest.count} on ${formatShort(data.busiest.date)}`
              : "—"
          }
        />
      </div>

      <div
        className="relative overflow-x-auto"
        onMouseLeave={() => setHover(null)}
      >
        <svg
          viewBox={`-26 -16 ${width + 32} ${height + 22}`}
          className="block min-w-[680px] w-full"
          role="img"
          aria-label={`Commit activity heatmap for ${data.owner}/${data.repo}`}
        >
          {monthMarks.map((m) => (
            <text
              key={`${m.x}-${m.label}`}
              x={m.x}
              y={-4}
              fill="rgba(255,255,255,0.35)"
              fontFamily="var(--font-mono)"
              fontSize="9"
              letterSpacing="0.12em"
            >
              {m.label.toUpperCase()}
            </text>
          ))}
          {["Mon", "Wed", "Fri"].map((label, i) => (
            <text
              key={label}
              x={-8}
              y={ROW_STRIDE * (i * 2 + 1) + CELL * 0.75}
              textAnchor="end"
              fill="rgba(255,255,255,0.3)"
              fontFamily="var(--font-mono)"
              fontSize="8"
              letterSpacing="0.1em"
            >
              {label.toUpperCase()}
            </text>
          ))}
          {weeks.map((week, col) =>
            week.map((day, row) => {
              if (!day) return null;
              const x = col * COL_STRIDE;
              const y = row * ROW_STRIDE;
              const fill = FILL[bucketFor(day.count)];
              const tier = day.landmark?.tier;
              return (
                <g
                  key={`${col}-${row}`}
                  onMouseEnter={(e) => {
                    setHover({
                      day,
                      x: e.clientX,
                      y: e.clientY,
                    });
                  }}
                  onMouseMove={(e) => {
                    setHover((h) =>
                      h && h.day.date === day.date
                        ? { day, x: e.clientX, y: e.clientY }
                        : h,
                    );
                  }}
                  style={{ cursor: day.count > 0 ? "pointer" : "default" }}
                >
                  <rect
                    x={x}
                    y={y}
                    width={CELL}
                    height={CELL}
                    rx={2.5}
                    fill={fill}
                    stroke={
                      hover?.day.date === day.date
                        ? "rgba(255,255,255,0.5)"
                        : "transparent"
                    }
                    strokeWidth={1}
                  />
                  {tier && tier !== "misc" && (
                    <circle
                      cx={x + CELL - 2.5}
                      cy={y + 2.5}
                      r={1.6}
                      fill={TIER_DOT[tier]}
                    />
                  )}
                </g>
              );
            }),
          )}
        </svg>
      </div>

      <div className="mt-5 flex flex-wrap items-center justify-between gap-y-3 font-mono text-[0.65rem] uppercase tracking-[0.18em] text-white/35">
        <div className="flex items-center gap-3">
          <span>Less</span>
          <div className="flex items-center gap-1">
            {FILL.map((c, i) => (
              <span
                key={c}
                aria-hidden
                style={{
                  width: 11,
                  height: 11,
                  borderRadius: 2,
                  background: c,
                  display: "inline-block",
                }}
              >
                <span className="sr-only">level {i}</span>
              </span>
            ))}
          </div>
          <span>More</span>
        </div>
        <div className="flex items-center gap-4">
          <LegendDot color={TIER_DOT.feat} label="feat" />
          <LegendDot color={TIER_DOT.fix} label="fix" />
          <LegendDot color={TIER_DOT.breaking} label="breaking" />
        </div>
      </div>

      {hover && (
        <div
          ref={tooltipRef}
          role="tooltip"
          style={{
            position: "fixed",
            left: hover.x + 14,
            top: hover.y + 14,
            zIndex: 50,
            pointerEvents: "none",
          }}
          className="max-w-[320px] rounded-md border border-white/15 bg-[#0b0e14]/95 px-3 py-2 text-xs leading-snug text-white/85 shadow-lg backdrop-blur"
        >
          <div className="font-mono text-[0.62rem] uppercase tracking-[0.2em] text-white/45">
            {formatLong(hover.day.date)}
          </div>
          <div className="mt-1 text-[0.78rem] text-white">
            {hover.day.count === 0
              ? "No commits"
              : hover.day.count === 1
                ? "1 commit"
                : `${hover.day.count} commits`}
          </div>
          {hover.day.landmark && (
            <div className="mt-2 border-t border-white/10 pt-2">
              <div className="font-mono text-[0.6rem] uppercase tracking-[0.18em] text-white/40">
                {hover.day.landmark.tier}
              </div>
              <div className="mt-0.5 line-clamp-3 text-[0.78rem] text-white/85">
                {hover.day.landmark.message}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline gap-2">
      <span className="font-[family-name:var(--font-display)] text-2xl tracking-[-0.02em] text-white">
        {value}
      </span>
      <span className="font-mono text-[0.62rem] uppercase tracking-[0.2em] text-white/40">
        {label}
      </span>
    </div>
  );
}

function LegendDot({ color, label }: { color: string; label: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span
        aria-hidden
        style={{
          width: 6,
          height: 6,
          borderRadius: 999,
          background: color,
          display: "inline-block",
        }}
      />
      {label}
    </span>
  );
}

function formatLong(iso: string): string {
  const d = new Date(`${iso}T00:00:00Z`);
  return d.toLocaleDateString("en-US", {
    weekday: "short",
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  });
}

function formatShort(iso: string): string {
  const d = new Date(`${iso}T00:00:00Z`);
  return d.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  });
}
