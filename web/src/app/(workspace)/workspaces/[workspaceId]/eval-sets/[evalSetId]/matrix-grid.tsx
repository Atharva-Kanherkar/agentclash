"use client";

import { Bot, Package } from "lucide-react";

import type {
  MatrixCell,
  MatrixCellState,
  MatrixGrid as Grid,
  RefLabelMap,
} from "@/lib/eval-sets";
import {
  countCellsByState,
  displayRef,
  MATRIX_CELL_STATES,
  matrixCellStateLabel,
} from "@/lib/eval-sets";
import { cn } from "@/lib/utils";

const STATE_DOT: Record<MatrixCellState, string> = {
  queued: "bg-muted-foreground/40",
  running: "bg-amber-500",
  scored: "bg-emerald-500",
  failed: "bg-red-500",
};

const STATE_CELL: Record<MatrixCellState, string> = {
  queued:
    "border-border/70 bg-muted/15 text-muted-foreground hover:bg-muted/30",
  running:
    "border-amber-500/45 bg-amber-500/[0.09] text-amber-950 dark:text-amber-50",
  scored:
    "border-emerald-500/40 bg-emerald-500/[0.08] text-emerald-950 dark:text-emerald-50",
  failed:
    "border-red-500/40 bg-red-500/[0.08] text-red-950 dark:text-red-50",
};

const STATE_BAR: Record<MatrixCellState, string> = {
  queued: "bg-muted-foreground/25",
  running: "bg-amber-500",
  scored: "bg-emerald-500",
  failed: "bg-red-500",
};

export function MatrixGridView({
  grid,
  onSelect,
  selectedKey,
  labels,
}: {
  grid: Grid;
  onSelect?: (cell: MatrixCell) => void;
  selectedKey?: string | null;
  labels?: RefLabelMap | null;
}) {
  if (grid.rowLabels.length === 0 || grid.colLabels.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-border bg-muted/10 px-6 py-10 text-center">
        <p className="text-sm font-medium">Matrix not ready yet</p>
        <p className="mt-1 text-sm text-muted-foreground">
          Combinations appear here once the eval set finishes expanding.
        </p>
      </div>
    );
  }

  const counts = countCellsByState(grid);
  const rowIsAgent = grid.rowAxis === "agent";
  const RowIcon = rowIsAgent ? Bot : Package;
  const ColIcon = rowIsAgent ? Package : Bot;

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">
          {grid.rowLabels.length} × {grid.colLabels.length} cells · click any
          cell for runs
        </p>
        <ul className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
          {MATRIX_CELL_STATES.map((state) => (
            <li key={state} className="inline-flex items-center gap-1.5">
              <span
                className={cn("size-1.5 rounded-full", STATE_DOT[state])}
                aria-hidden
              />
              <span>
                {matrixCellStateLabel(state)}
                {counts[state] > 0 ? (
                  <span className="tabular-nums text-foreground/70">
                    {" "}
                    {counts[state]}
                  </span>
                ) : null}
              </span>
            </li>
          ))}
        </ul>
      </div>

      <div className="overflow-x-auto rounded-xl border border-border bg-card/30 shadow-sm">
        <table className="w-full min-w-[36rem] border-collapse text-sm">
          <thead>
            <tr>
              <th className="sticky left-0 z-20 bg-background/95 px-3 py-3 text-left backdrop-blur">
                <span className="inline-flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-[0.14em] text-muted-foreground">
                  <RowIcon className="size-3 opacity-70" aria-hidden />
                  {rowIsAgent ? "Agent" : "Pack"}
                  <span className="opacity-40">/</span>
                  <ColIcon className="size-3 opacity-70" aria-hidden />
                  {rowIsAgent ? "Pack" : "Agent"}
                </span>
              </th>
              {grid.colLabels.map((col) => (
                <th
                  key={col}
                  title={col}
                  className="max-w-[9.5rem] px-2 py-3 text-left align-bottom"
                >
                  <span className="line-clamp-2 text-xs font-medium leading-snug text-foreground/90">
                    {displayRef(col, labels)}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {grid.rowLabels.map((row) => (
              <tr key={row} className="border-t border-border/80">
                <th
                  title={row}
                  className="sticky left-0 z-10 max-w-[11rem] bg-background/95 px-3 py-2 text-left align-middle backdrop-blur"
                >
                  <span className="line-clamp-2 text-xs font-medium leading-snug">
                    {displayRef(row, labels)}
                  </span>
                </th>
                {grid.colLabels.map((col) => {
                  const cell = grid.cells[`${row}\0${col}`];
                  const state = cell?.state ?? "queued";
                  const cellKey = cell
                    ? `${cell.agentRef}\0${cell.packRef}`
                    : null;
                  const active = cellKey != null && cellKey === selectedKey;
                  const progress =
                    cell && cell.totalCount > 0
                      ? Math.round((cell.doneCount / cell.totalCount) * 100)
                      : 0;
                  const passRate =
                    cell && cell.totalCount > 0
                      ? `${cell.passCount}/${cell.totalCount}`
                      : "—";
                  const agentName = cell
                    ? displayRef(cell.agentRef, labels)
                    : displayRef(rowIsAgent ? row : col, labels);
                  const packName = cell
                    ? displayRef(cell.packRef, labels)
                    : displayRef(rowIsAgent ? col : row, labels);

                  return (
                    <td key={col} className="p-1.5">
                      <button
                        type="button"
                        disabled={!cell}
                        onClick={() => cell && onSelect?.(cell)}
                        aria-label={`${agentName} on ${packName}: ${matrixCellStateLabel(state)}, ${passRate} completed`}
                        aria-pressed={active}
                        className={cn(
                          "relative flex h-[3.75rem] w-full min-w-[6.25rem] flex-col items-stretch justify-between overflow-hidden rounded-lg border px-2.5 py-2 text-left transition",
                          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60",
                          "disabled:cursor-default disabled:opacity-40",
                          STATE_CELL[state],
                          state === "running" && "animate-pulse",
                          active &&
                            "ring-2 ring-foreground/25 border-foreground/30",
                          cell && "hover:-translate-y-px hover:shadow-sm",
                        )}
                      >
                        <span className="flex items-center gap-1.5">
                          <span
                            className={cn(
                              "size-1.5 shrink-0 rounded-full",
                              STATE_DOT[state],
                              state === "running" && "animate-pulse",
                            )}
                            aria-hidden
                          />
                          <span className="text-[10px] font-medium uppercase tracking-[0.12em] opacity-75">
                            {matrixCellStateLabel(state)}
                          </span>
                        </span>
                        <span className="font-mono text-sm font-semibold tabular-nums tracking-tight">
                          {passRate}
                        </span>
                        <span
                          className="pointer-events-none absolute inset-x-0 bottom-0 h-0.5 bg-border/40"
                          aria-hidden
                        >
                          <span
                            className={cn(
                              "block h-full transition-[width] duration-500",
                              STATE_BAR[state],
                            )}
                            style={{ width: `${progress}%` }}
                          />
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
    </div>
  );
}
