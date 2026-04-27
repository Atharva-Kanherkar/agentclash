"use client";

import { Badge } from "@/components/ui/badge";
import { KpiStrip } from "./kpi-strip";
import { AlertTriangle } from "@/components/ui/nourico-icons";
import type { PlaygroundExperimentResult } from "@/lib/api/types";

function statusVariant(
  status: string,
): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "completed":
      return "default";
    case "running":
      return "secondary";
    case "failed":
      return "destructive";
    default:
      return "outline";
  }
}

interface ExperimentResultsProps {
  results: PlaygroundExperimentResult[];
}

export function ExperimentResults({ results }: ExperimentResultsProps) {
  if (results.length === 0) {
    return (
      <p className="py-4 text-center text-sm text-muted-foreground">
        No results yet.
      </p>
    );
  }

  return (
    <div className="space-y-3">
      {results.map((result) => (
        <div
          key={result.id}
          className="rounded-lg border border-border p-4 space-y-3"
        >
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">{result.case_key}</span>
            <Badge variant={statusVariant(result.status)}>
              {result.status}
            </Badge>
          </div>

          <KpiStrip
            latencyMs={result.latency_ms}
            totalTokens={result.total_tokens}
            costUsd={result.cost_usd}
            dimensions={result.dimension_scores}
          />

          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Rendered Prompt
              </p>
              <pre className="max-h-48 overflow-auto rounded-md bg-muted/50 p-3 text-xs leading-relaxed whitespace-pre-wrap">
                {result.rendered_prompt}
              </pre>
            </div>
            <div>
              <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Output
              </p>
              <pre className="max-h-48 overflow-auto rounded-md bg-muted/50 p-3 text-xs leading-relaxed whitespace-pre-wrap">
                {result.actual_output ||
                  result.error_message ||
                  "No output"}
              </pre>
            </div>
          </div>

          {result.warnings && result.warnings.length > 0 && (
            <div className="flex items-start gap-2 rounded-md border border-amber-500/20 bg-amber-500/5 p-3">
              <AlertTriangle className="mt-0.5 size-3.5 shrink-0 text-amber-500" />
              <div className="space-y-0.5 text-xs text-amber-700 dark:text-amber-400">
                {result.warnings.map((w, i) => (
                  <p key={i}>{w}</p>
                ))}
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
