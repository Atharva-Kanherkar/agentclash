"use client";

import { Clock, Coins, Hash } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { scoreColor } from "@/lib/scores";

interface KpiStripProps {
  latencyMs?: number;
  totalTokens?: number;
  costUsd?: number | null;
  dimensions?: Record<string, number | null>;
  loading?: boolean;
}

function formatLatency(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function formatCost(usd: number | null | undefined): string {
  if (usd == null) return "N/A";
  return `$${usd.toFixed(4)}`;
}

export function KpiStrip({
  latencyMs,
  totalTokens,
  costUsd,
  dimensions,
  loading,
}: KpiStripProps) {
  if (loading) {
    return (
      <div className="flex items-center gap-4">
        <Skeleton className="h-4 w-16" />
        <Skeleton className="h-4 w-16" />
        <Skeleton className="h-4 w-16" />
      </div>
    );
  }

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
      {latencyMs != null && (
        <span className="flex items-center gap-1">
          <Clock className="size-3" />
          {formatLatency(latencyMs)}
        </span>
      )}
      {totalTokens != null && (
        <span className="flex items-center gap-1">
          <Hash className="size-3" />
          {totalTokens.toLocaleString()} tokens
        </span>
      )}
      {costUsd !== undefined && (
        <span className="flex items-center gap-1">
          <Coins className="size-3" />
          {formatCost(costUsd)}
        </span>
      )}
      {dimensions &&
        Object.entries(dimensions).map(([name, score]) => (
          <span
            key={name}
            className={`rounded-full border px-2 py-0.5 text-xs font-medium ${scoreColor(score ?? undefined)}`}
          >
            {name}: {score != null ? `${(score * 100).toFixed(0)}%` : "N/A"}
          </span>
        ))}
    </div>
  );
}
