"use client";

import { useCallback, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useAccessToken } from "@workos-inc/authkit-nextjs/components";
import useSWR from "swr";

import { createApiClient } from "@/lib/api/client";
import {
  downloadEvalSetExport,
  listEvalSetCases,
  searchEvalSetCases,
} from "@/lib/api/eval-sets";
import type { EvalSetCaseResult } from "@/lib/api/types";
import {
  comboRepeatLabel,
  displayRef,
  type RefLabelMap,
} from "@/lib/eval-sets";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export function CaseExplorer({
  evalSetId,
  labels,
}: {
  evalSetId: string;
  labels?: RefLabelMap | null;
}) {
  const { accessToken, getAccessToken } = useAccessToken();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const q = searchParams.get("q") ?? "";
  const verdict = searchParams.get("verdict") ?? "";
  const minScore = searchParams.get("min_score") ?? "";
  const [draftQ, setDraftQ] = useState(q);
  const [exporting, setExporting] = useState<"csv" | "jsonl" | null>(null);

  const setParam = useCallback(
    (key: string, value: string) => {
      const next = new URLSearchParams(searchParams.toString());
      if (value) next.set(key, value);
      else next.delete(key);
      next.set("tab", "cases");
      router.replace(`${pathname}?${next.toString()}`);
    },
    [pathname, router, searchParams],
  );

  const { data, error, isLoading } = useSWR(
    accessToken
      ? (["eval-set-cases", evalSetId, q, verdict, minScore] as const)
      : null,
    async () => {
      const token = (await getAccessToken()) ?? accessToken;
      if (!token) return [] as EvalSetCaseResult[];
      const api = createApiClient(token);
      const params: Record<string, string | number | undefined> = {
        verdict: verdict || undefined,
        min_score: minScore || undefined,
        limit: 50,
      };
      const res = q
        ? await searchEvalSetCases(api, evalSetId, { ...params, q })
        : await listEvalSetCases(api, evalSetId, params);
      return res.cases ?? [];
    },
  );
  const cases = data ?? [];

  async function onExport(format: "csv" | "jsonl") {
    setExporting(format);
    try {
      const token = (await getAccessToken()) ?? accessToken;
      if (!token) return;
      await downloadEvalSetExport(token, evalSetId, format);
    } finally {
      setExporting(null);
    }
  }

  return (
    <div className="space-y-4">
      <form
        className="flex flex-wrap items-end gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          setParam("q", draftQ.trim());
        }}
      >
        <div className="min-w-[12rem] flex-1">
          <label className="text-xs text-muted-foreground">
            Transcript search
          </label>
          <Input
            value={draftQ}
            onChange={(e) => setDraftQ(e.target.value)}
            placeholder="e.g. refund"
          />
        </div>
        <div>
          <label className="text-xs text-muted-foreground">Verdict</label>
          <Input
            value={verdict}
            onChange={(e) => setParam("verdict", e.target.value)}
            placeholder="pass / fail"
            className="w-28"
          />
        </div>
        <div>
          <label className="text-xs text-muted-foreground">Min score</label>
          <Input
            value={minScore}
            onChange={(e) => setParam("min_score", e.target.value)}
            placeholder="0.5"
            className="w-24"
          />
        </div>
        <Button type="submit" variant="secondary">
          Apply
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={exporting !== null}
          onClick={() => void onExport("csv")}
        >
          {exporting === "csv" ? "Exporting…" : "Export CSV"}
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={exporting !== null}
          onClick={() => void onExport("jsonl")}
        >
          {exporting === "jsonl" ? "Exporting…" : "Export JSONL"}
        </Button>
      </form>
      {error ? (
        <p className="text-sm text-destructive">
          {error instanceof Error ? error.message : "Failed to load cases"}
        </p>
      ) : null}
      {isLoading && !data ? (
        <p className="text-sm text-muted-foreground">Loading cases…</p>
      ) : null}
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Pack</TableHead>
              <TableHead>Case</TableHead>
              <TableHead>Repeat</TableHead>
              <TableHead>Verdict</TableHead>
              <TableHead>Score</TableHead>
              <TableHead>Snippet</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {cases.map((c) => (
              <TableRow key={c.id}>
                <TableCell
                  className="max-w-[10rem] truncate text-xs font-medium"
                  title={c.pack_ref}
                >
                  {displayRef(c.pack_ref, labels)}
                </TableCell>
                <TableCell className="font-mono text-xs">{c.case_key}</TableCell>
                <TableCell className="tabular-nums text-xs text-muted-foreground">
                  {comboRepeatLabel(c.matrix_key)}
                </TableCell>
                <TableCell>{c.verdict || "—"}</TableCell>
                <TableCell className="tabular-nums">{c.score ?? "—"}</TableCell>
                <TableCell className="max-w-md text-xs text-muted-foreground">
                  {highlightSnippet(
                    c.snippet || c.transcript_text?.slice(0, 160) || "—",
                    q,
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {cases.length === 0 && !error && !isLoading ? (
        <p className="text-sm text-muted-foreground">
          No cases match these filters.
        </p>
      ) : null}
    </div>
  );
}

function highlightSnippet(text: string, q: string) {
  if (!q.trim()) return text;
  const idx = text.toLowerCase().indexOf(q.toLowerCase());
  if (idx < 0) return text;
  return (
    <>
      {text.slice(0, idx)}
      <mark className="bg-amber-200/80 text-foreground dark:bg-amber-500/30">
        {text.slice(idx, idx + q.length)}
      </mark>
      {text.slice(idx + q.length)}
    </>
  );
}
