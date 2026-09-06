"use client";

import { useState } from "react";
import { Check, HelpCircle, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { CaseResult } from "@/lib/vibe";
import { SafeMarkdown } from "./safe-markdown";

export function CaseEvidence({
  summary,
  load,
}: {
  summary: CaseResult;
  load: (key: string) => Promise<CaseResult>;
}) {
  const [evidence, setEvidence] = useState<CaseResult | null>(
    summary.input ? summary : null,
  );
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  async function fetchEvidence() {
    if (loading) return;
    setLoading(true);
    setError("");
    try {
      setEvidence(await load(summary.case_key));
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }
  return (
    <details
      className="group py-3"
      onToggle={(e) => {
        if (
          e.currentTarget.open &&
          (!evidence || evidence.verdict !== summary.verdict)
        )
          void fetchEvidence();
      }}
    >
      <summary className="flex cursor-pointer list-none items-center gap-3 text-sm">
        {summary.verdict === "PASS" ? (
          <Check size={16} />
        ) : summary.verdict === "FAIL" ? (
          <X size={16} className="text-builder-warn" />
        ) : (
          <HelpCircle size={16} className="text-builder-fg-muted" />
        )}
        <span className="flex-1">{summary.case_key.replaceAll("-", " ")}</span>
        <span className="font-mono text-[10px] text-builder-fg-muted">
          {summary.verdict}
        </span>
      </summary>
      <div className="mt-4 space-y-4 pl-7">
        {loading && (
          <p role="status" className="text-xs text-builder-fg-muted">
            Loading saved evidence…
          </p>
        )}
        {error && (
          <p role="alert" className="text-xs text-builder-warn">
            {error}
          </p>
        )}
        {evidence && (
          <>
            <div>
              <p className="mb-1 text-xs text-builder-fg-muted">Test input</p>
              <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-all font-mono text-xs">
                {JSON.stringify(evidence.input, null, 2)}
              </pre>
            </div>
            {evidence.output && (
              <div>
                <p className="text-xs text-builder-fg-muted">Agent response</p>
                <SafeMarkdown>{evidence.output}</SafeMarkdown>
              </div>
            )}
            {evidence.error && (
              <p className="text-sm text-builder-warn">
                {evidence.error.message}
              </p>
            )}
            {evidence.checks.map((check) => (
              <div key={check.key}>
                <p className="text-xs font-medium">
                  {check.key} · {check.verdict}
                </p>
                <SafeMarkdown>
                  {check.error?.message || check.evidence}
                </SafeMarkdown>
              </div>
            ))}
          </>
        )}
        {(error || summary.verdict === "UNKNOWN") && (
          <Button
            size="sm"
            variant="ghost"
            disabled={loading}
            onClick={fetchEvidence}
          >
            Reload saved evidence
          </Button>
        )}
      </div>
    </details>
  );
}
