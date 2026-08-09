"use client";

import Link from "next/link";
import { Grid3x3, Loader2 } from "lucide-react";

import type { EvalSetStatus, ListEvalSetsResponse } from "@/lib/api/types";
import { useApiQuery } from "@/lib/api/swr";
import { EVAL_SET_ACTIVE, evalSetStatusLabel } from "@/lib/eval-sets";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/ui/empty-state";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const POLL_MS = 5000;

const SET_STATUS_BADGE: Partial<
  Record<EvalSetStatus, "default" | "secondary" | "destructive" | "outline">
> = {
  queued: "outline",
  expanding: "secondary",
  running: "default",
  aggregating: "secondary",
  completed: "secondary",
  failed: "destructive",
  cancelled: "outline",
  budget_exceeded: "destructive",
};

export function EvalSetsClient({ workspaceId }: { workspaceId: string }) {
  const { data, error, isLoading } = useApiQuery<ListEvalSetsResponse>(
    "/v1/eval-sets",
    { workspace_id: workspaceId },
    {
      refreshInterval: (response) =>
        response?.eval_sets?.some((s) => EVAL_SET_ACTIVE.includes(s.status))
          ? POLL_MS
          : 0,
    },
  );
  const sets = data?.eval_sets ?? [];

  if (isLoading && !data) {
    return <p className="text-sm text-muted-foreground">Loading eval sets…</p>;
  }
  if (error) {
    return (
      <EmptyState
        icon={<Grid3x3 className="h-8 w-8" />}
        title="Could not load eval sets"
        description={
          error instanceof Error ? error.message : "Failed to load eval sets"
        }
      />
    );
  }
  if (sets.length === 0) {
    return (
      <EmptyState
        icon={<Grid3x3 className="h-8 w-8" />}
        title="No eval sets yet"
        description="Submit a manifest with agentclash evalset submit sweep.yaml to create one."
      />
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Eval Sets</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Multi-pack matrices filling in live as combinations complete.
        </p>
      </div>
      <div className="overflow-x-auto rounded-xl border border-border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Combinations</TableHead>
              <TableHead>Created</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sets.map((set) => {
              const live = EVAL_SET_ACTIVE.includes(set.status);
              return (
                <TableRow key={set.id}>
                  <TableCell>
                    <Link
                      href={`/workspaces/${workspaceId}/eval-sets/${set.id}`}
                      className="font-medium text-foreground underline-offset-4 hover:underline"
                    >
                      {set.name}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={SET_STATUS_BADGE[set.status] ?? "outline"}
                      className="capitalize"
                    >
                      {live ? (
                        <Loader2
                          data-icon="inline-start"
                          className="size-3 animate-spin"
                        />
                      ) : null}
                      {evalSetStatusLabel(set.status)}
                    </Badge>
                  </TableCell>
                  <TableCell className="tabular-nums">
                    {set.combination_count}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {new Date(set.created_at).toLocaleString()}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
