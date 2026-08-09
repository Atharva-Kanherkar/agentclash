"use client";

import type { MatrixCell, MatrixGrid as Grid } from "@/lib/eval-sets";
import { cn } from "@/lib/utils";

export function MatrixGridView({
  grid,
  onSelect,
  selectedKey,
}: {
  grid: Grid;
  onSelect?: (cell: MatrixCell) => void;
  selectedKey?: string | null;
}) {
  if (grid.rowLabels.length === 0 || grid.colLabels.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        Matrix will appear once combinations are expanded.
      </p>
    );
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-border">
      <table className="w-full min-w-[32rem] border-collapse text-sm">
        <thead>
          <tr>
            <th className="sticky left-0 z-10 bg-background px-3 py-2 text-left text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
              {grid.rowAxis === "agent" ? "Agent \\ Pack" : "Pack \\ Agent"}
            </th>
            {grid.colLabels.map((col) => (
              <th
                key={col}
                className="px-3 py-2 text-left text-xs font-medium text-muted-foreground"
              >
                <span className="line-clamp-2 max-w-[8rem] font-mono text-[11px]">
                  {shortRef(col)}
                </span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {grid.rowLabels.map((row) => (
            <tr key={row} className="border-t border-border">
              <th className="sticky left-0 z-10 bg-background px-3 py-2 text-left font-mono text-[11px] font-medium">
                {shortRef(row)}
              </th>
              {grid.colLabels.map((col) => {
                const cell = grid.cells[`${row}\0${col}`];
                const state = cell?.state ?? "queued";
                const cellKey = cell
                  ? `${cell.agentRef}\0${cell.packRef}`
                  : null;
                const active = cellKey != null && cellKey === selectedKey;
                const passRate =
                  cell && cell.totalCount > 0
                    ? `${cell.passCount}/${cell.totalCount}`
                    : "—";
                return (
                  <td key={col} className="px-2 py-2">
                    <button
                      type="button"
                      disabled={!cell}
                      onClick={() => cell && onSelect?.(cell)}
                      className={cn(
                        "flex h-14 w-full min-w-[5.5rem] flex-col items-start justify-center rounded-md border px-2 text-left transition",
                        state === "queued" &&
                          "border-border/60 bg-muted/20 text-muted-foreground",
                        state === "running" &&
                          "border-amber-600/50 bg-amber-500/10 text-amber-950 dark:text-amber-100 animate-pulse",
                        state === "scored" &&
                          "border-emerald-600/40 bg-emerald-500/10",
                        state === "failed" && "border-red-600/40 bg-red-500/10",
                        active && "ring-2 ring-foreground/30",
                      )}
                    >
                      <span className="text-[10px] uppercase tracking-[0.14em] opacity-70">
                        {state}
                      </span>
                      <span className="mt-0.5 font-mono text-[11px]">
                        {passRate}
                      </span>
                    </button>
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function shortRef(ref: string): string {
  if (ref.length <= 18) return ref;
  return `${ref.slice(0, 8)}…${ref.slice(-6)}`;
}
