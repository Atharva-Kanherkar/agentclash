"use client";

import { useState, useEffect, useCallback } from "react";
import { useAccessToken } from "@workos-inc/authkit-nextjs/components";
import { createApiClient } from "@/lib/api/client";
import type {
  Run,
  RunAgent,
  ScorecardResponse,
  ValidatorDetail,
  MetricDetail,
  LLMJudgeResult,
} from "@/lib/api/types";
import { Badge } from "@/components/ui/badge";
import {
  Loader2,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  MinusCircle,
  Target,
  Shield,
  Zap,
  DollarSign,
  BarChart3,
  ChevronDown,
  ChevronRight,
  Activity,
  FlaskConical,
  Gauge,
  Bot,
} from "lucide-react";
import { scorePercent, scoreColor, barWidth, barColor } from "@/lib/scores";

const POLL_MS = 5000;

const LEGACY_DIM_META: Record<string, { label: string; icon: typeof Target }> = {
  correctness: { label: "Correctness", icon: Target },
  reliability: { label: "Reliability", icon: Shield },
  latency: { label: "Latency", icon: Zap },
  cost: { label: "Cost", icon: DollarSign },
};

function dimLabel(key: string): string {
  return LEGACY_DIM_META[key]?.label ?? key.charAt(0).toUpperCase() + key.slice(1).replace(/_/g, " ");
}

function DimIcon({ dimKey, className }: { dimKey: string; className?: string }) {
  const Icon = LEGACY_DIM_META[dimKey]?.icon ?? BarChart3;
  return <Icon className={className} />;
}

// --- Score Ring ---

function ScoreRing({ score, size = 120 }: { score?: number; size?: number }) {
  const r = (size - 12) / 2;
  const circumference = 2 * Math.PI * r;
  const pct = score ?? 0;
  const offset = circumference * (1 - pct);
  const color = score == null ? "text-muted-foreground" : score >= 0.8 ? "text-emerald-400" : score >= 0.5 ? "text-amber-400" : "text-red-400";

  return (
    <div className="relative inline-flex items-center justify-center" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="currentColor" strokeWidth={6} className="text-muted/40" />
        {score != null && (
          <circle
            cx={size / 2} cy={size / 2} r={r} fill="none" stroke="currentColor" strokeWidth={6}
            strokeDasharray={circumference} strokeDashoffset={offset} strokeLinecap="round"
            className={color}
          />
        )}
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className={`text-2xl font-bold ${color}`}>{scorePercent(score)}</span>
      </div>
    </div>
  );
}

// --- Verdict Badge ---

function VerdictBadge({ verdict, state }: { verdict: string; state: string }) {
  if (state === "error") {
    return <Badge variant="destructive" className="text-[10px] gap-1"><MinusCircle className="size-3" />Error</Badge>;
  }
  if (state === "unavailable") {
    return <Badge variant="secondary" className="text-[10px] gap-1"><MinusCircle className="size-3" />N/A</Badge>;
  }
  if (verdict === "pass") {
    return <Badge className="text-[10px] gap-1 bg-emerald-500/15 text-emerald-400 border-emerald-500/30 hover:bg-emerald-500/20"><CheckCircle2 className="size-3" />Pass</Badge>;
  }
  if (verdict === "fail") {
    return <Badge className="text-[10px] gap-1 bg-red-500/15 text-red-400 border-red-500/30 hover:bg-red-500/20"><XCircle className="size-3" />Fail</Badge>;
  }
  return <Badge variant="outline" className="text-[10px]">{verdict || state}</Badge>;
}

// --- Validator Row ---

function ValidatorRow({ v }: { v: ValidatorDetail }) {
  const [expanded, setExpanded] = useState(false);
  const hasDetail = !!v.reason || v.normalized_score != null;

  return (
    <div className="border-b border-border last:border-0">
      <button
        className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-muted/30 transition-colors"
        onClick={() => hasDetail && setExpanded(!expanded)}
        disabled={!hasDetail}
      >
        <VerdictBadge verdict={v.verdict} state={v.state} />
        <div className="flex-1 min-w-0">
          <span className="text-sm font-medium">{v.key}</span>
          <span className="text-xs text-muted-foreground ml-2">{v.type.replace(/_/g, " ")}</span>
        </div>
        {v.normalized_score != null && (
          <span className={`text-xs font-mono ${scoreColor(v.normalized_score)}`}>
            {scorePercent(v.normalized_score)}
          </span>
        )}
        {hasDetail && (
          expanded
            ? <ChevronDown className="size-3.5 text-muted-foreground shrink-0" />
            : <ChevronRight className="size-3.5 text-muted-foreground shrink-0" />
        )}
      </button>
      {expanded && v.reason && (
        <div className="px-4 pb-3 pl-20">
          <p className="text-xs text-muted-foreground bg-muted/40 rounded-md px-3 py-2">{v.reason}</p>
        </div>
      )}
    </div>
  );
}

// --- Metric Row ---

function MetricRow({ m }: { m: MetricDetail }) {
  const displayValue = m.numeric_value != null
    ? m.numeric_value.toLocaleString()
    : m.boolean_value != null
      ? (m.boolean_value ? "true" : "false")
      : m.text_value ?? "—";

  return (
    <div className="flex items-center gap-3 px-4 py-3 border-b border-border last:border-0">
      <Activity className="size-3.5 text-muted-foreground shrink-0" />
      <div className="flex-1 min-w-0">
        <span className="text-sm font-medium">{m.key}</span>
        <span className="text-xs text-muted-foreground ml-2">{m.collector.replace(/_/g, " ")}</span>
      </div>
      <Badge variant={m.state === "available" ? "outline" : "secondary"} className="text-[10px]">
        {m.state}
      </Badge>
      <span className="text-sm font-mono font-medium min-w-[60px] text-right">{displayValue}</span>
    </div>
  );
}

function llmJudgeIsAvailable(judge: LLMJudgeResult): boolean {
  const available = judge.payload?.available;
  if (typeof available === "boolean") return available;
  return judge.normalized_score != null;
}

function LLMJudgeRow({ judge }: { judge: LLMJudgeResult }) {
  const [expanded, setExpanded] = useState(false);
  const available = llmJudgeIsAvailable(judge);
  const payloadPreview = JSON.stringify(judge.payload, null, 2);

  return (
    <div className="border-b border-border last:border-0">
      <button
        className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-muted/30 transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        <Badge
          variant={available ? "outline" : "secondary"}
          className={`text-[10px] ${available ? "border-emerald-500/30 text-emerald-400" : ""}`}
        >
          {available ? "available" : "n/a"}
        </Badge>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">{judge.judge_key}</span>
            <span className="text-xs text-muted-foreground">{judge.mode.replace(/_/g, " ")}</span>
          </div>
          <div className="text-xs text-muted-foreground mt-0.5">
            {judge.sample_count} sample{judge.sample_count === 1 ? "" : "s"} across {judge.model_count} model{judge.model_count === 1 ? "" : "s"}
            {judge.confidence ? ` • ${judge.confidence} confidence` : ""}
          </div>
        </div>
        {judge.normalized_score != null && (
          <span className={`text-xs font-mono ${scoreColor(judge.normalized_score)}`}>
            {scorePercent(judge.normalized_score)}
          </span>
        )}
        {expanded
          ? <ChevronDown className="size-3.5 text-muted-foreground shrink-0" />
          : <ChevronRight className="size-3.5 text-muted-foreground shrink-0" />
        }
      </button>
      {expanded && (
        <div className="px-4 pb-4 pl-20 space-y-2">
          {judge.variance != null && (
            <p className="text-xs text-muted-foreground">variance: {judge.variance.toFixed(4)}</p>
          )}
          <pre className="overflow-x-auto rounded-md bg-muted/40 px-3 py-2 text-[11px] text-muted-foreground">
            {payloadPreview}
          </pre>
        </div>
      )}
    </div>
  );
}

// --- Main Component ---

interface ScorecardClientProps {
  initialScorecard: ScorecardResponse;
  run: Run;
  agent: RunAgent;
}

export function ScorecardClient({ initialScorecard, run, agent }: ScorecardClientProps) {
  const { getAccessToken } = useAccessToken();
  const [scorecard, setScorecard] = useState<ScorecardResponse>(initialScorecard);

  const isPending = scorecard.state === "pending";
  const isErrored = scorecard.state === "errored";
  const isReady = scorecard.state === "ready";
  const doc = scorecard.scorecard;

  const fetchScorecard = useCallback(async () => {
    try {
      const token = await getAccessToken();
      const api = createApiClient(token);
      const res = await api.get<ScorecardResponse>(`/v1/scorecards/${agent.id}`, { allowedStatuses: [202, 409] });
      setScorecard(res);
    } catch { /* Silently fail on poll */ }
  }, [getAccessToken, agent.id]);

  useEffect(() => {
    if (!isPending) return;
    const interval = setInterval(fetchScorecard, POLL_MS);
    return () => clearInterval(interval);
  }, [isPending, fetchScorecard]);

  // Sort dimensions: legacy first in canonical order, then custom alphabetically.
  const dimKeys = doc?.dimensions
    ? Object.keys(doc.dimensions).sort((a, b) => {
        const order = ["correctness", "reliability", "latency", "cost"];
        const ai = order.indexOf(a);
        const bi = order.indexOf(b);
        if (ai !== -1 && bi !== -1) return ai - bi;
        if (ai !== -1) return -1;
        if (bi !== -1) return 1;
        return a.localeCompare(b);
      })
    : [];

  const validators = doc?.validator_details ?? [];
  const metrics = doc?.metric_details ?? [];
  const llmJudges = scorecard.llm_judge_results ?? [];
  const passCount = validators.filter((v) => v.verdict === "pass").length;
  const failCount = validators.filter((v) => v.verdict === "fail").length;
  const errorCount = validators.filter((v) => v.state === "error").length;
  const availableJudgeCount = llmJudges.filter(llmJudgeIsAvailable).length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3 mb-1">
            <h1 className="text-lg font-semibold tracking-tight">{agent.label}</h1>
            <Badge variant="outline">{run.name}</Badge>
            {doc?.strategy && <Badge variant="outline" className="text-[10px]">{doc.strategy}</Badge>}
          </div>
        </div>
        <Badge variant={isReady ? "default" : isErrored ? "destructive" : "secondary"}>
          {isPending && <Loader2 data-icon="inline-start" className="size-3 animate-spin" />}
          {scorecard.state}
        </Badge>
      </div>

      {/* Pending / Error states */}
      {isPending && (
        <div className="rounded-lg border border-border p-8 text-center text-sm text-muted-foreground">
          <Loader2 className="size-6 animate-spin mx-auto mb-3" />
          <p>Evaluation in progress...</p>
        </div>
      )}
      {isErrored && (
        <div className="rounded-lg border border-destructive/20 bg-destructive/5 p-8 text-center text-sm text-destructive">
          <AlertTriangle className="size-6 mx-auto mb-3" />
          <p className="font-medium">Scorecard unavailable</p>
          <p className="text-xs mt-1 text-destructive/70">{scorecard.message || "An error occurred during evaluation."}</p>
        </div>
      )}

      {/* Ready state */}
      {isReady && (
        <>
          {/* Top row: Score ring + pass/fail summary + dimension bars */}
          <div className="grid gap-6 md:grid-cols-[auto_1fr]">
            {/* Score ring + verdict */}
            <div className="rounded-lg border border-border p-6 flex flex-col items-center gap-3">
              <ScoreRing score={scorecard.overall_score} />
              {doc?.passed != null && (
                <div className="flex items-center gap-1.5">
                  {doc.passed
                    ? <><CheckCircle2 className="size-4 text-emerald-400" /><span className="text-sm text-emerald-400 font-medium">Passed</span></>
                    : <><XCircle className="size-4 text-red-400" /><span className="text-sm text-red-400 font-medium">Failed</span></>
                  }
                </div>
              )}
              {doc?.overall_reason && (
                <p className="text-[10px] text-muted-foreground text-center max-w-[180px]">{doc.overall_reason}</p>
              )}
              {/* Quick stats */}
              <div className="flex gap-4 mt-2 text-center">
                <div>
                  <div className="text-lg font-bold text-emerald-400">{passCount}</div>
                  <div className="text-[10px] text-muted-foreground">pass</div>
                </div>
                <div>
                  <div className="text-lg font-bold text-red-400">{failCount}</div>
                  <div className="text-[10px] text-muted-foreground">fail</div>
                </div>
                {errorCount > 0 && (
                  <div>
                    <div className="text-lg font-bold text-amber-400">{errorCount}</div>
                    <div className="text-[10px] text-muted-foreground">error</div>
                  </div>
                )}
              </div>
            </div>

            {/* Dimension breakdown */}
            <div className="rounded-lg border border-border p-6">
              <h2 className="text-sm font-semibold mb-4 flex items-center gap-2">
                <Gauge className="size-4" />
                Dimensions
              </h2>
              <div className="space-y-4">
                {dimKeys.map((key) => {
                  const dim = doc!.dimensions[key];
                  return (
                    <div key={key}>
                      <div className="flex items-center justify-between mb-1.5">
                        <div className="flex items-center gap-2 text-sm">
                          <DimIcon dimKey={key} className="size-3.5 text-muted-foreground" />
                          <span>{dimLabel(key)}</span>
                          {dim.better_direction && (
                            <span className="text-[10px] text-muted-foreground/50">{dim.better_direction === "higher" ? "higher is better" : "lower is better"}</span>
                          )}
                        </div>
                        <div className="flex items-center gap-2">
                          {dim.state !== "available" && (
                            <Badge variant={dim.state === "error" ? "destructive" : "secondary"} className="text-[10px] h-4">{dim.state === "unavailable" ? "n/a" : dim.state}</Badge>
                          )}
                          <span className={`text-sm font-medium tabular-nums ${scoreColor(dim.score)}`}>{scorePercent(dim.score)}</span>
                        </div>
                      </div>
                      <div className="h-2 rounded-full bg-muted overflow-hidden">
                        <div className={`h-full rounded-full transition-all ${barColor(dim.score)}`} style={{ width: barWidth(dim.score) }} />
                      </div>
                      {dim.reason && <p className="text-xs text-muted-foreground mt-1">{dim.reason}</p>}
                    </div>
                  );
                })}
                {dimKeys.length === 0 && <p className="text-sm text-muted-foreground">No dimensions declared.</p>}
              </div>
            </div>
          </div>

          {/* Validators */}
          {validators.length > 0 && (
            <div className="rounded-lg border border-border">
              <div className="px-4 py-3 border-b border-border flex items-center gap-2">
                <FlaskConical className="size-4 text-muted-foreground" />
                <h2 className="text-sm font-semibold">Validators</h2>
                <span className="text-xs text-muted-foreground ml-auto">
                  {passCount}/{validators.length} passed
                </span>
              </div>
              {validators.map((v) => <ValidatorRow key={v.key} v={v} />)}
            </div>
          )}

          {/* Metrics */}
          {metrics.length > 0 && (
            <div className="rounded-lg border border-border">
              <div className="px-4 py-3 border-b border-border flex items-center gap-2">
                <Activity className="size-4 text-muted-foreground" />
                <h2 className="text-sm font-semibold">Metrics</h2>
              </div>
              {metrics.map((m) => <MetricRow key={m.key} m={m} />)}
            </div>
          )}

          {llmJudges.length > 0 && (
            <div className="rounded-lg border border-border">
              <div className="px-4 py-3 border-b border-border flex items-center gap-2">
                <Bot className="size-4 text-muted-foreground" />
                <h2 className="text-sm font-semibold">LLM Judges</h2>
                <span className="text-xs text-muted-foreground ml-auto">
                  {availableJudgeCount}/{llmJudges.length} available
                </span>
              </div>
              {llmJudges.map((judge) => <LLMJudgeRow key={judge.id} judge={judge} />)}
            </div>
          )}

          {/* Warnings */}
          {doc?.warnings && doc.warnings.length > 0 && (
            <div className="rounded-lg border border-amber-500/20 bg-amber-500/5 p-4">
              <h3 className="text-sm font-medium text-amber-400 flex items-center gap-2 mb-2">
                <AlertTriangle className="size-4" />
                Warnings
              </h3>
              <ul className="space-y-1">
                {doc.warnings.map((w, i) => <li key={i} className="text-xs text-amber-400/80">{w}</li>)}
              </ul>
            </div>
          )}
        </>
      )}
    </div>
  );
}
