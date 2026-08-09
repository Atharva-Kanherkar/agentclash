import type { ApiClient } from "./client";
import type {
  CompareEvalSetsResponse,
  EvalSessionDetail,
  EvalSetCaseResult,
  EvalSetReportResponse,
  GetEvalSetResponse,
  ListEvalSetsResponse,
} from "./types";

export function listEvalSets(
  api: ApiClient,
  workspaceId: string,
): Promise<ListEvalSetsResponse> {
  return api.get<ListEvalSetsResponse>("/v1/eval-sets", {
    params: { workspace_id: workspaceId },
  });
}

export function getEvalSet(
  api: ApiClient,
  evalSetId: string,
): Promise<GetEvalSetResponse> {
  return api.get<GetEvalSetResponse>(`/v1/eval-sets/${evalSetId}`);
}

export function getEvalSetReport(
  api: ApiClient,
  evalSetId: string,
): Promise<EvalSetReportResponse> {
  return api.get<EvalSetReportResponse>(`/v1/eval-sets/${evalSetId}/report`);
}

export function searchEvalSetCases(
  api: ApiClient,
  evalSetId: string,
  params: Record<string, string | number | undefined>,
): Promise<{ cases: EvalSetCaseResult[]; query: string }> {
  return api.get(`/v1/eval-sets/${evalSetId}/search`, { params });
}

export function listEvalSetCases(
  api: ApiClient,
  evalSetId: string,
  params: Record<string, string | number | undefined>,
): Promise<{ cases: EvalSetCaseResult[]; next_cursor?: string }> {
  return api.get(`/v1/eval-sets/${evalSetId}/cases`, { params });
}

export function compareEvalSets(
  api: ApiClient,
  a: string,
  b: string,
): Promise<CompareEvalSetsResponse> {
  return api.get<CompareEvalSetsResponse>("/v1/compare/eval-sets", {
    params: { a, b },
  });
}

export function getEvalSession(
  api: ApiClient,
  evalSessionId: string,
): Promise<EvalSessionDetail> {
  return api.get<EvalSessionDetail>(`/v1/eval-sessions/${evalSessionId}`);
}

export function evalSetExportUrl(
  evalSetId: string,
  format: "csv" | "jsonl",
): string {
  const base =
    typeof window === "undefined"
      ? process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL
      : process.env.NEXT_PUBLIC_API_URL;
  return `${(base ?? "").replace(/\/+$/, "")}/v1/eval-sets/${evalSetId}/export?format=${format}`;
}

/** Authenticated download for warehouse export (Bearer required). */
export async function downloadEvalSetExport(
  token: string,
  evalSetId: string,
  format: "csv" | "jsonl",
): Promise<void> {
  const url = evalSetExportUrl(evalSetId, format);
  const res = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    throw new Error(`Export failed (${res.status})`);
  }
  const blob = await res.blob();
  const objectUrl = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = objectUrl;
  a.download = `eval-set-${evalSetId}.${format}`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(objectUrl);
}
