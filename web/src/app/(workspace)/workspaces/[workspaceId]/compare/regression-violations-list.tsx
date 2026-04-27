"use client";

import Link from "next/link";
import { ArrowRightCircle } from "@/components/ui/nourico-icons";

import {
  REGRESSION_BLOCKING_RULES,
  regressionRuleLabel,
} from "@/lib/api/release-gates";
import type {
  ReleaseGateRegressionViolation,
  RegressionSeverity,
} from "@/lib/api/types";
import { Badge } from "@/components/ui/badge";

const severityVariant: Record<
  RegressionSeverity,
  "default" | "outline" | "destructive"
> = {
  info: "outline",
  warning: "default",
  blocking: "destructive",
};

function severityBadge(severity: string) {
  const variant =
    severity === "blocking" || severity === "warning" || severity === "info"
      ? severityVariant[severity as RegressionSeverity]
      : "outline";
  return <Badge variant={variant}>{severity}</Badge>;
}

interface RegressionViolationsListProps {
  workspaceId: string;
  candidateRunId?: string;
  candidateRunAgentId?: string;
  violations: ReleaseGateRegressionViolation[];
}

export function RegressionViolationsList({
  workspaceId,
  candidateRunId,
  candidateRunAgentId,
  violations,
}: RegressionViolationsListProps) {
  if (violations.length === 0) return null;

  return (
    <div className="mt-3 rounded-md border border-red-500/20 bg-red-500/5 p-3">
      <p className="mb-2 text-xs font-medium uppercase tracking-wide text-red-300/90">
        Regression violations
      </p>
      <ul className="space-y-2">
        {violations.map((v, idx) => {
          const isBlocking = REGRESSION_BLOCKING_RULES.has(v.rule);
          const caseHref = `/workspaces/${workspaceId}/regression-suites/${v.suite_id}/cases/${v.regression_case_id}`;
          const scorecardHref =
            candidateRunId && candidateRunAgentId
              ? `/workspaces/${workspaceId}/runs/${candidateRunId}/agents/${candidateRunAgentId}/scorecard#${v.evidence.scoring_result_id}`
              : null;
          return (
            <li
              key={`${v.rule}-${v.regression_case_id}-${idx}`}
              className="flex flex-wrap items-center gap-2 text-xs"
            >
              <span className="font-medium text-foreground">
                {regressionRuleLabel(v.rule)}
              </span>
              {severityBadge(v.severity)}
              {isBlocking && (
                <Badge variant="destructive">blocking rule</Badge>
              )}
              {typeof v.observed_count === "number" && (
                <span className="text-muted-foreground">
                  observed {v.observed_count}
                </span>
              )}
              <Link
                href={caseHref}
                className="inline-flex items-center gap-1 text-foreground hover:underline underline-offset-4"
              >
                Open case
                <ArrowRightCircle className="size-3" />
              </Link>
              {scorecardHref && (
                <Link
                  href={scorecardHref}
                  className="inline-flex items-center gap-1 text-muted-foreground hover:text-foreground transition-colors"
                >
                  Scoring result
                  <ArrowRightCircle className="size-3" />
                </Link>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
