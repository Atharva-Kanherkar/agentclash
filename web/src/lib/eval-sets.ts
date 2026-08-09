import type { GetEvalSetResponse } from "@/lib/api/types";

export type EvalSetStatus =
  | "queued"
  | "expanding"
  | "running"
  | "aggregating"
  | "completed"
  | "failed"
  | "cancelled";

export const EVAL_SET_ACTIVE: EvalSetStatus[] = [
  "queued",
  "expanding",
  "running",
  "aggregating",
];

export type MatrixCellState = "queued" | "running" | "scored" | "failed";

export interface MatrixComboRef {
  matrixKey: string;
  status: string;
  runId?: string;
}

export interface MatrixCell {
  packRef: string;
  agentRef: string;
  state: MatrixCellState;
  /** Fraction of repeats/models that reached a terminal status. */
  doneCount: number;
  totalCount: number;
  passCount: number;
  combos: MatrixComboRef[];
}

export interface MatrixGrid {
  rowAxis: "agent" | "pack";
  rowLabels: string[];
  colLabels: string[];
  cells: Record<string, MatrixCell>; // key = `${row}\0${col}`
}

export type LiveStatusMap = Record<string, { status: string; runId?: string }>;

function terminal(status: string): boolean {
  return ["completed", "failed", "cancelled", "canceled"].includes(
    status.toLowerCase(),
  );
}

function runningish(status: string): boolean {
  return [
    "running",
    "provisioning",
    "scoring",
    "queued",
    "draft",
    "aggregating",
    "expanding",
  ].includes(status.toLowerCase());
}

function cellState(combos: MatrixComboRef[]): MatrixCellState {
  if (combos.length === 0) return "queued";
  if (combos.some((c) => runningish(c.status) && !terminal(c.status))) {
    // "queued" run status still means the set has work in flight for this cell
    // when the parent set is active; treat non-terminal as running except pure empty.
    const anyActive = combos.some((c) => {
      const s = c.status.toLowerCase();
      return s === "running" || s === "provisioning" || s === "scoring";
    });
    if (anyActive) return "running";
  }
  if (combos.every((c) => c.status.toLowerCase() === "completed")) return "scored";
  if (combos.some((c) => ["failed", "cancelled", "canceled"].includes(c.status.toLowerCase()))) {
    if (combos.every((c) => terminal(c.status))) return "failed";
    return "running";
  }
  if (combos.every((c) => terminal(c.status))) return "scored";
  if (combos.some((c) => c.status && c.status.toLowerCase() !== "queued")) {
    return "running";
  }
  return "queued";
}

/** Strip pack_ref prefix from matrix_key to recover agent_ref (and optional model). */
export function agentRefFromMatrixKey(matrixKey: string, packRef: string): string {
  const prefix = `${packRef}/`;
  let rest = matrixKey.startsWith(prefix)
    ? matrixKey.slice(prefix.length)
    : matrixKey;
  // drop trailing /repeat
  const last = rest.lastIndexOf("/");
  if (last > 0 && /^\d+$/.test(rest.slice(last + 1))) {
    rest = rest.slice(0, last);
  }
  // if model segment present, keep agent as first segment for grid axis
  const slash = rest.indexOf("/");
  if (slash > 0) return rest.slice(0, slash);
  return rest || "agent";
}

export function buildMatrixGrid(
  detail: GetEvalSetResponse,
  live?: LiveStatusMap,
): MatrixGrid {
  const expansionCombos = detail.eval_set.expansion?.combinations ?? [];
  const resultCombos = detail.result?.aggregate?.combinations ?? [];
  const statusByKey = new Map<string, { status: string; runId?: string }>();

  for (const c of resultCombos) {
    if (!c.matrix_key) continue;
    statusByKey.set(c.matrix_key, { status: c.status ?? "queued" });
  }
  if (live) {
    for (const [key, value] of Object.entries(live)) {
      statusByKey.set(key, value);
    }
  }

  type Seed = { matrix_key: string; pack_ref: string; agent_ref: string };
  const seeds: Seed[] = [];

  if (expansionCombos.length > 0) {
    for (const c of expansionCombos) {
      seeds.push({
        matrix_key: c.matrix_key,
        pack_ref: c.pack_ref,
        agent_ref: c.agent_ref,
      });
    }
  } else {
    for (const c of resultCombos) {
      if (!c.matrix_key) continue;
      const pack = c.pack_ref || "pack";
      seeds.push({
        matrix_key: c.matrix_key,
        pack_ref: pack,
        agent_ref: agentRefFromMatrixKey(c.matrix_key, pack),
      });
    }
  }

  const packs = new Set<string>();
  const agents = new Set<string>();
  const byCell = new Map<string, MatrixComboRef[]>();

  for (const c of seeds) {
    packs.add(c.pack_ref);
    agents.add(c.agent_ref);
    const liveStatus = statusByKey.get(c.matrix_key);
    const status = liveStatus?.status ?? "queued";
    const key = `${c.agent_ref}\0${c.pack_ref}`;
    const list = byCell.get(key) ?? [];
    list.push({
      matrixKey: c.matrix_key,
      status,
      runId: liveStatus?.runId,
    });
    byCell.set(key, list);
  }

  const packLabels = [...packs].sort();
  const agentLabels = [...agents].sort();
  const flip = packLabels.length > agentLabels.length;

  const cells: Record<string, MatrixCell> = {};
  for (const [key, combos] of byCell) {
    const [agentRef, packRef] = key.split("\0");
    const doneCount = combos.filter((c) => terminal(c.status)).length;
    const passCount = combos.filter(
      (c) => c.status.toLowerCase() === "completed",
    ).length;
    const cell: MatrixCell = {
      packRef,
      agentRef,
      state: cellState(combos),
      doneCount,
      totalCount: combos.length,
      passCount,
      combos,
    };
    if (flip) {
      cells[`${packRef}\0${agentRef}`] = cell;
    } else {
      cells[`${agentRef}\0${packRef}`] = cell;
    }
  }

  if (flip) {
    return {
      rowAxis: "pack",
      rowLabels: packLabels,
      colLabels: agentLabels,
      cells,
    };
  }
  return {
    rowAxis: "agent",
    rowLabels: agentLabels,
    colLabels: packLabels,
    cells,
  };
}

export function completionPercent(
  detail: GetEvalSetResponse,
  live?: LiveStatusMap,
): number {
  const total = detail.eval_set.combination_count || 0;
  if (total <= 0) return 0;

  if (live && Object.keys(live).length > 0) {
    const done = Object.values(live).filter((v) => terminal(v.status)).length;
    return Math.min(100, Math.round((done / total) * 100));
  }

  const combos = detail.result?.aggregate?.combinations ?? [];
  const withKeys = combos.filter((c) => c.matrix_key);
  const done = withKeys.filter((c) => terminal(c.status ?? "")).length;
  if (withKeys.length === 0) {
    // Fall back to set-level status for header until live map exists.
    if (detail.eval_set.status === "completed") return 100;
    return 0;
  }
  return Math.min(100, Math.round((done / total) * 100));
}

export function inFlightCount(live?: LiveStatusMap): number {
  if (!live) return 0;
  return Object.values(live).filter((v) => {
    const s = v.status.toLowerCase();
    return s === "running" || s === "provisioning" || s === "scoring";
  }).length;
}
