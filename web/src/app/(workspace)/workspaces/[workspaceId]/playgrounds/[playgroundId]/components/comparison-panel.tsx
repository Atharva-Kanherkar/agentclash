"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { EmptyState } from "@/components/ui/empty-state";
import {
  ArrowUpRight,
  ArrowDownRight,
  Minus,
  GitCompare,
} from "lucide-react";
import type {
  PlaygroundExperiment,
  PlaygroundExperimentComparison,
  PlaygroundDimensionDelta,
} from "@/lib/api/types";

function statusVariant(
  status: string,
): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "completed":
      return "default";
    case "failed":
      return "destructive";
    default:
      return "outline";
  }
}

function DeltaIndicator({ delta }: { delta: PlaygroundDimensionDelta }) {
  if (delta.state === "missing") {
    return <span className="text-xs text-muted-foreground">N/A</span>;
  }
  if (delta.state === "partial" || delta.delta == null) {
    return (
      <span className="flex items-center gap-1 text-xs text-muted-foreground">
        <Minus className="size-3" />
        partial
      </span>
    );
  }
  const d = delta.delta;
  if (Math.abs(d) < 0.001) {
    return (
      <span className="flex items-center gap-1 text-xs text-muted-foreground">
        <Minus className="size-3" />
        same
      </span>
    );
  }
  if (d > 0) {
    return (
      <span className="flex items-center gap-1 text-xs text-emerald-500">
        <ArrowUpRight className="size-3" />+{(d * 100).toFixed(1)}%
      </span>
    );
  }
  return (
    <span className="flex items-center gap-1 text-xs text-red-500">
      <ArrowDownRight className="size-3" />
      {(d * 100).toFixed(1)}%
    </span>
  );
}

interface ComparisonPanelProps {
  workspaceId: string;
  playgroundId: string;
  experiments: PlaygroundExperiment[];
  comparison: PlaygroundExperimentComparison | null;
  initialBaselineId: string | null;
  initialCandidateId: string | null;
}

export function ComparisonPanel({
  workspaceId,
  playgroundId,
  experiments,
  comparison,
  initialBaselineId,
  initialCandidateId,
}: ComparisonPanelProps) {
  const router = useRouter();
  const completedExperiments = experiments.filter(
    (e) => e.status === "completed",
  );

  const [baselineId, setBaselineId] = useState(
    initialBaselineId ?? completedExperiments[0]?.id ?? "",
  );
  const [candidateId, setCandidateId] = useState(
    initialCandidateId ?? completedExperiments[1]?.id ?? completedExperiments[0]?.id ?? "",
  );

  function handleCompare() {
    if (!baselineId || !candidateId) return;
    const params = new URLSearchParams();
    params.set("tab", "compare");
    params.set("baseline", baselineId);
    params.set("candidate", candidateId);
    router.push(
      `/workspaces/${workspaceId}/playgrounds/${playgroundId}?${params.toString()}`,
    );
  }

  if (completedExperiments.length < 2) {
    return (
      <EmptyState
        icon={<GitCompare className="size-10" />}
        title="Not enough experiments"
        description="Complete at least two experiments to compare their results side-by-side."
      />
    );
  }

  return (
    <div className="space-y-6">
      <div className="rounded-lg border border-border p-4 space-y-4">
        <h3 className="text-sm font-medium">Select Experiments</h3>
        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Baseline
            </label>
            <Select
              value={baselineId}
              onValueChange={(v) => v && setBaselineId(v)}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Select baseline" />
              </SelectTrigger>
              <SelectContent>
                {completedExperiments.map((exp) => (
                  <SelectItem key={exp.id} value={exp.id}>
                    {exp.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Candidate
            </label>
            <Select
              value={candidateId}
              onValueChange={(v) => v && setCandidateId(v)}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Select candidate" />
              </SelectTrigger>
              <SelectContent>
                {completedExperiments.map((exp) => (
                  <SelectItem key={exp.id} value={exp.id}>
                    {exp.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <div className="flex justify-end">
          <Button
            onClick={handleCompare}
            disabled={!baselineId || !candidateId || baselineId === candidateId}
          >
            <GitCompare className="mr-2 size-4" />
            Compare
          </Button>
        </div>
      </div>

      {comparison && (
        <div className="space-y-6">
          <div>
            <h3 className="mb-3 text-sm font-medium">Aggregated Dimensions</h3>
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {Object.entries(comparison.aggregated_dimension_deltas).map(
                ([dimension, delta]) => (
                  <div
                    key={dimension}
                    className="rounded-lg border border-border p-3 space-y-2"
                  >
                    <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                      {dimension}
                    </p>
                    <div className="flex items-center justify-between">
                      <div className="text-xs text-muted-foreground">
                        <span>
                          B:{" "}
                          {delta.baseline_value != null
                            ? `${(delta.baseline_value * 100).toFixed(0)}%`
                            : "N/A"}
                        </span>
                        <span className="mx-1.5">/</span>
                        <span>
                          C:{" "}
                          {delta.candidate_value != null
                            ? `${(delta.candidate_value * 100).toFixed(0)}%`
                            : "N/A"}
                        </span>
                      </div>
                      <DeltaIndicator delta={delta} />
                    </div>
                  </div>
                ),
              )}
            </div>
          </div>

          <div>
            <h3 className="mb-3 text-sm font-medium">Per-Case Comparison</h3>
            <div className="space-y-3">
              {comparison.per_case.map((item) => (
                <div
                  key={item.case_key}
                  className="rounded-lg border border-border p-4 space-y-3"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium">
                      {item.case_key}
                    </span>
                    <div className="flex gap-2">
                      <Badge variant={statusVariant(item.baseline_status)}>
                        B: {item.baseline_status}
                      </Badge>
                      <Badge variant={statusVariant(item.candidate_status)}>
                        C: {item.candidate_status}
                      </Badge>
                    </div>
                  </div>
                  <div className="grid gap-4 md:grid-cols-2">
                    <div>
                      <p className="mb-1 text-xs font-medium text-muted-foreground">
                        Baseline Output
                      </p>
                      <pre className="max-h-40 overflow-auto rounded-md bg-muted/50 p-3 text-xs whitespace-pre-wrap">
                        {item.baseline_output ||
                          item.baseline_error_message ||
                          "No output"}
                      </pre>
                    </div>
                    <div>
                      <p className="mb-1 text-xs font-medium text-muted-foreground">
                        Candidate Output
                      </p>
                      <pre className="max-h-40 overflow-auto rounded-md bg-muted/50 p-3 text-xs whitespace-pre-wrap">
                        {item.candidate_output ||
                          item.candidate_error_message ||
                          "No output"}
                      </pre>
                    </div>
                  </div>
                  {Object.keys(item.dimension_deltas).length > 0 && (
                    <div className="flex flex-wrap gap-3">
                      {Object.entries(item.dimension_deltas).map(
                        ([dim, delta]) => (
                          <div key={dim} className="flex items-center gap-1.5">
                            <span className="text-xs text-muted-foreground">
                              {dim}:
                            </span>
                            <DeltaIndicator delta={delta} />
                          </div>
                        ),
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
