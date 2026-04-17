/**
 * Scorecard-local formatting helpers. Kept here (not in lib/scores) because
 * they're specific to the display surface, not the underlying data model.
 */

import type { LLMJudgeResult } from "@/lib/api/types";

export function formatDuration(start?: string, end?: string): string {
  if (!start || !end) return "—";
  const ms = new Date(end).getTime() - new Date(start).getTime();
  if (!Number.isFinite(ms) || ms < 0) return "—";
  if (ms < 1000) return `${ms}ms`;
  const s = ms / 1000;
  if (s < 60) return `${s.toFixed(1)}s`;
  const m = Math.floor(s / 60);
  const rs = Math.round(s % 60);
  return `${m}m ${rs}s`;
}

export function formatTimestamp(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function humanizeKey(key: string): string {
  return key
    .split(/[_\-]/)
    .map((s) => (s.length ? s[0].toUpperCase() + s.slice(1) : s))
    .join(" ");
}

export function signedDelta(delta?: number): string {
  if (delta == null || delta === 0) return "±0";
  const pct = (delta * 100).toFixed(1);
  return delta > 0 ? `+${pct}` : pct;
}

/**
 * Parse the LLM judge payload into a structured shape the UI can render.
 *
 * See backend/internal/workflow/judges.go for the producer contract:
 *   { mode, calls, model_scores, aggregated_score, warnings, reason?, unable_to_judge_count? }
 *
 * Payload is typed as Record<string, unknown> so we walk it defensively.
 */
export interface JudgeCall {
  model: string;
  providerKey?: string;
  sampleIndex?: number;
  score?: number;
  confidence?: string;
  error?: string;
  responseText?: string;
}

export interface ParsedJudgePayload {
  mode?: string;
  calls: JudgeCall[];
  modelScores: Array<{ model: string; score: number }>;
  aggregatedScore?: number;
  warnings: string[];
  reason?: string;
  unableToJudgeCount?: number;
  available: boolean;
}

export function parseJudgePayload(judge: LLMJudgeResult): ParsedJudgePayload {
  const p = judge.payload || {};
  const get = <T>(key: string): T | undefined => p[key] as T | undefined;

  const rawCalls = get<Array<Record<string, unknown>>>("calls") ?? [];
  const calls: JudgeCall[] = rawCalls.map((c) => ({
    model: String(c.model ?? ""),
    providerKey: asString(c.provider_key),
    sampleIndex: asNumber(c.sample_index),
    score: asNumber(c.score),
    confidence: asString(c.confidence),
    error: asString(c.error),
    responseText: asString(c.response_text),
  }));

  const rawModelScores =
    (get<Record<string, unknown>>("model_scores") as
      | Record<string, unknown>
      | undefined) ?? {};
  const modelScores: Array<{ model: string; score: number }> = Object.entries(
    rawModelScores,
  )
    .map(([model, v]) => ({ model, score: Number(v) }))
    .filter((e) => Number.isFinite(e.score));

  const rawWarnings = get<unknown[]>("warnings") ?? [];
  const warnings = rawWarnings
    .map((w) => (typeof w === "string" ? w : JSON.stringify(w)))
    .filter((w) => w && w.length > 0);

  const available =
    typeof p.available === "boolean"
      ? (p.available as boolean)
      : judge.normalized_score != null;

  return {
    mode: asString(get("mode")),
    calls,
    modelScores,
    aggregatedScore: asNumber(get("aggregated_score")),
    warnings,
    reason: asString(get("reason")),
    unableToJudgeCount: asNumber(get("unable_to_judge_count")),
    available,
  };
}

function asString(v: unknown): string | undefined {
  return typeof v === "string" && v.length > 0 ? v : undefined;
}

function asNumber(v: unknown): number | undefined {
  if (v == null) return undefined;
  const n = Number(v);
  return Number.isFinite(n) ? n : undefined;
}

/**
 * Sort legacy dims (correctness, reliability, latency, cost) first in canonical
 * order, then custom dimensions alphabetically. Kept in one place so every
 * scorecard surface presents dimensions the same way.
 */
const LEGACY_ORDER = ["correctness", "reliability", "latency", "cost"];

export function sortDimensionKeys(keys: string[]): string[] {
  return [...keys].sort((a, b) => {
    const ai = LEGACY_ORDER.indexOf(a);
    const bi = LEGACY_ORDER.indexOf(b);
    if (ai !== -1 && bi !== -1) return ai - bi;
    if (ai !== -1) return -1;
    if (bi !== -1) return 1;
    return a.localeCompare(b);
  });
}
