"use client";

import { Suspense, useMemo, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useAccessToken } from "@workos-inc/authkit-nextjs/components";
import { Grid3x3 } from "lucide-react";
import useSWR from "swr";

import { createApiClient } from "@/lib/api/client";
import { compareEvalSets, getEvalSession } from "@/lib/api/eval-sets";
import type { CompareEvalSetsResponse, GetEvalSetResponse } from "@/lib/api/types";
import { useApiQuery } from "@/lib/api/swr";
import {
  EVAL_SET_ACTIVE,
  buildMatrixGrid,
  completionPercent,
  inFlightCount,
  type LiveStatusMap,
  type MatrixCell,
} from "@/lib/eval-sets";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { CaseExplorer } from "./case-explorer";
import { MatrixGridView } from "./matrix-grid";

const POLL_MS = 5000;

export function EvalSetDetailClient({
  workspaceId,
  evalSetId,
}: {
  workspaceId: string;
  evalSetId: string;
}) {
  return (
    <Suspense fallback={<p className="text-sm text-muted-foreground">Loading…</p>}>
      <EvalSetDetailInner workspaceId={workspaceId} evalSetId={evalSetId} />
    </Suspense>
  );
}

function EvalSetDetailInner({
  workspaceId,
  evalSetId,
}: {
  workspaceId: string;
  evalSetId: string;
}) {
  const { accessToken, getAccessToken } = useAccessToken();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const tab = searchParams.get("tab") ?? "matrix";
  const [selected, setSelected] = useState<MatrixCell | null>(null);
  const [compareId, setCompareId] = useState("");
  const [compare, setCompare] = useState<CompareEvalSetsResponse | null>(null);
  const [compareError, setCompareError] = useState<string | null>(null);

  const { data: detail, error, isLoading } = useApiQuery<GetEvalSetResponse>(
    `/v1/eval-sets/${evalSetId}`,
    undefined,
    {
      refreshInterval: (response) =>
        response && EVAL_SET_ACTIVE.includes(response.eval_set.status)
          ? POLL_MS
          : 0,
    },
  );

  const sessionIds = detail?.eval_session_ids ?? [];
  const active =
    !!detail && EVAL_SET_ACTIVE.includes(detail.eval_set.status);
  const liveKey =
    active && sessionIds.length > 0
      ? (["eval-set-live", evalSetId, ...sessionIds] as const)
      : null;

  const { data: live = {} } = useSWR<LiveStatusMap>(
    liveKey,
    async () => {
      const token = (await getAccessToken()) ?? accessToken;
      if (!token) return {};
      const api = createApiClient(token);
      const nextLive: LiveStatusMap = {};
      await Promise.all(
        sessionIds.map(async (sessionId) => {
          try {
            const session = await getEvalSession(api, sessionId);
            for (const run of session.runs ?? []) {
              if (!run.matrix_key) continue;
              nextLive[run.matrix_key] = {
                status: run.status,
                runId: run.id,
              };
            }
          } catch {
            // best-effort
          }
        }),
      );
      return nextLive;
    },
    {
      refreshInterval: active ? POLL_MS : 0,
      revalidateOnFocus: true,
    },
  );

  const grid = useMemo(
    () => (detail ? buildMatrixGrid(detail, live) : null),
    [detail, live],
  );

  const selectedLive = useMemo(() => {
    if (!selected || !grid) return selected;
    const key = `${selected.agentRef}\0${selected.packRef}`;
    return grid.cells[key] ?? selected;
  }, [selected, grid]);

  const setTab = (value: string) => {
    const next = new URLSearchParams(searchParams.toString());
    next.set("tab", value);
    router.replace(`${pathname}?${next.toString()}`);
  };

  if (error && !detail) {
    return (
      <EmptyState
        icon={<Grid3x3 className="h-8 w-8" />}
        title="Eval set unavailable"
        description={
          error instanceof Error ? error.message : "Failed to load eval set"
        }
      />
    );
  }
  if ((isLoading && !detail) || !detail || !grid) {
    return <p className="text-sm text-muted-foreground">Loading eval set…</p>;
  }

  const set = detail.eval_set;
  const pct = completionPercent(detail, live);
  const flight = inFlightCount(live);
  const runs = detail.result?.run_count ?? detail.result?.aggregate?.runs ?? 0;
  const sessions =
    detail.eval_session_ids?.length ?? detail.result?.session_count ?? 0;
  const selectedKey = selectedLive
    ? `${selectedLive.agentRef}\0${selectedLive.packRef}`
    : null;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0">
          <Link
            href={`/workspaces/${workspaceId}/eval-sets`}
            className="text-xs text-muted-foreground hover:underline"
          >
            ← Eval sets
          </Link>
          <h1 className="mt-2 text-2xl font-semibold tracking-tight">{set.name}</h1>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <Badge variant="outline">{set.status}</Badge>
            <span className="text-sm text-muted-foreground">
              {set.combination_count} combinations · {sessions} sessions · {runs}{" "}
              runs
            </span>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Stat label="Complete" value={`${pct}%`} />
          <Stat label="In flight" value={String(flight)} />
          <Stat
            label="Elapsed"
            value={
              set.started_at
                ? formatElapsed(set.started_at, set.finished_at)
                : "—"
            }
          />
          <Stat
            label="Budget"
            value={set.budget_usd != null ? `$${set.budget_usd}` : "—"}
          />
        </div>
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="matrix">Matrix</TabsTrigger>
          <TabsTrigger value="cases">Case explorer</TabsTrigger>
          <TabsTrigger value="compare">Compare</TabsTrigger>
        </TabsList>
        <TabsContent value="matrix" className="space-y-4">
          <MatrixGridView
            grid={grid}
            selectedKey={selectedKey}
            onSelect={setSelected}
          />
          {selectedLive ? (
            <div className="rounded-lg border border-border bg-card/40 p-4">
              <h2 className="text-sm font-semibold">
                {shortRef(selectedLive.agentRef)} × {shortRef(selectedLive.packRef)}
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                {selectedLive.passCount}/{selectedLive.totalCount} completed · state{" "}
                {selectedLive.state}
              </p>
              <ul className="mt-3 space-y-2">
                {selectedLive.combos.map((c) => (
                  <li
                    key={c.matrixKey}
                    className="flex flex-wrap items-center justify-between gap-2 font-mono text-xs"
                  >
                    <span className="text-muted-foreground">{c.matrixKey}</span>
                    <span className="flex items-center gap-2">
                      <Badge variant="outline">{c.status || "queued"}</Badge>
                      {c.runId ? (
                        <Link
                          href={`/workspaces/${workspaceId}/runs/${c.runId}`}
                          className="underline-offset-4 hover:underline"
                        >
                          Open run
                        </Link>
                      ) : null}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </TabsContent>
        <TabsContent value="cases">
          <CaseExplorer evalSetId={evalSetId} />
        </TabsContent>
        <TabsContent value="compare" className="space-y-4">
          <div className="flex flex-wrap items-end gap-2">
            <div className="min-w-[16rem] flex-1">
              <label className="text-xs text-muted-foreground">
                Compare with eval set ID
              </label>
              <Input
                value={compareId}
                onChange={(e) => setCompareId(e.target.value)}
                placeholder="previous eval set uuid"
              />
            </div>
            <Button
              type="button"
              onClick={async () => {
                if (!accessToken || !compareId.trim()) return;
                try {
                  const api = createApiClient(accessToken);
                  const res = await compareEvalSets(
                    api,
                    evalSetId,
                    compareId.trim(),
                  );
                  setCompare(res);
                  setCompareError(null);
                } catch (err) {
                  setCompareError(
                    err instanceof Error ? err.message : "Compare failed",
                  );
                }
              }}
            >
              Compare
            </Button>
          </div>
          {compareError ? (
            <p className="text-sm text-destructive">{compareError}</p>
          ) : null}
          {compare ? (
            <div className="space-y-3 overflow-x-auto">
              <p className="text-sm text-muted-foreground">
                {compare.shared_keys} shared keys · {compare.regressions.length}{" "}
                regressions
              </p>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Matrix</TableHead>
                    <TableHead>A</TableHead>
                    <TableHead>B</TableHead>
                    <TableHead>Delta</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {compare.regressions.map((r) => (
                    <TableRow key={`${r.matrix_key}:${r.case_key}`}>
                      <TableCell className="font-mono text-xs">
                        {r.matrix_key}
                      </TableCell>
                      <TableCell>{r.a_score}</TableCell>
                      <TableCell className="text-red-600 dark:text-red-400">
                        {r.b_score}
                      </TableCell>
                      <TableCell className="text-red-600 dark:text-red-400">
                        {r.delta.toFixed(3)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              {compare.regressions.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No score regressions.
                </p>
              ) : null}
            </div>
          ) : null}
        </TabsContent>
      </Tabs>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border bg-background/70 px-3 py-2">
      <div className="text-[10px] uppercase tracking-[0.14em] text-muted-foreground">
        {label}
      </div>
      <div className="mt-1 text-sm font-semibold">{value}</div>
    </div>
  );
}

function formatElapsed(startedAt: string, finishedAt?: string | null): string {
  const start = new Date(startedAt).getTime();
  const end = finishedAt ? new Date(finishedAt).getTime() : Date.now();
  const sec = Math.max(0, Math.round((end - start) / 1000));
  if (sec < 60) return `${sec}s`;
  const min = Math.floor(sec / 60);
  return `${min}m ${sec % 60}s`;
}

function shortRef(ref: string): string {
  if (ref.length <= 24) return ref;
  return `${ref.slice(0, 10)}…${ref.slice(-8)}`;
}
