"use client";

import { useState } from "react";
import { useAccessToken } from "@workos-inc/authkit-nextjs/components";
import { Activity, Loader2 } from "lucide-react";
import { toast } from "sonner";

import { createApiClient } from "@/lib/api/client";
import { ApiError } from "@/lib/api/errors";
import type {
  ProviderAccount,
  ProviderAccountSmokeTestResponse,
} from "@/lib/api/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

export function TestProviderAccountDialog({
  account,
}: {
  account: ProviderAccount;
}) {
  const { getAccessToken } = useAccessToken();
  const [open, setOpen] = useState(false);
  const [model, setModel] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<ProviderAccountSmokeTestResponse | null>(
    null,
  );
  const modelRequired = account.provider_key === "custom";
  const modelFieldId = `provider-account-test-model-${account.id}`;

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
    if (!nextOpen) {
      setModel("");
      setResult(null);
    }
  }

  async function handleTest() {
    const requestedModel = model.trim();
    if (modelRequired && !requestedModel) {
      toast.error("Model is required for custom providers");
      return;
    }

    setSubmitting(true);
    setResult(null);
    try {
      const token = await getAccessToken();
      const api = createApiClient(token);
      const response = await api.post<ProviderAccountSmokeTestResponse>(
        `/v1/provider-accounts/${account.id}/test`,
        requestedModel ? { model: requestedModel } : {},
      );
      setResult(response);
    } catch (error) {
      toast.error(
        error instanceof ApiError
          ? error.message
          : "Failed to test provider account",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger render={<Button variant="outline" size="sm" />}>
        <Activity data-icon="inline-start" className="size-4" />
        Test
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Test {account.name}</DialogTitle>
          <DialogDescription>
            Send a minimal prompt through this provider account and inspect the
            connection result.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div>
            <label
              htmlFor={modelFieldId}
              className="mb-1.5 block text-sm font-medium"
            >
              Model
              {!modelRequired && (
                <span className="font-normal text-muted-foreground">
                  {" "}(optional — uses the provider default)
                </span>
              )}
            </label>
            <input
              id={modelFieldId}
              value={model}
              onChange={(event) => {
                setModel(event.target.value);
                setResult(null);
              }}
              placeholder={
                modelRequired ? "e.g. controlled-model" : "Leave blank for default"
              }
              required={modelRequired}
              disabled={submitting}
              className="block w-full rounded-lg border border-input bg-transparent px-3 py-2 text-sm placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/50 disabled:opacity-50"
            />
          </div>

          {result && <ProviderAccountTestResult result={result} />}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={submitting}
          >
            Close
          </Button>
          <Button onClick={handleTest} disabled={submitting}>
            {submitting ? (
              <>
                <Loader2 className="size-4 animate-spin" />
                Testing…
              </>
            ) : (
              "Run Test"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ProviderAccountTestResult({
  result,
}: {
  result: ProviderAccountSmokeTestResponse;
}) {
  return (
    <section
      aria-live="polite"
      className="space-y-3 rounded-lg border border-border bg-muted/30 p-4"
    >
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm font-medium">Connection result</span>
        <Badge variant={result.passed ? "default" : "destructive"}>
          {result.passed ? "Passed" : "Failed"}
        </Badge>
      </div>

      <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
        <dt className="text-muted-foreground">Model</dt>
        <dd className="break-all font-[family-name:var(--font-mono)]">
          {result.model || "—"}
        </dd>
        <dt className="text-muted-foreground">Provider model</dt>
        <dd className="break-all font-[family-name:var(--font-mono)]">
          {result.provider_model_id || "—"}
        </dd>
        <dt className="text-muted-foreground">Duration</dt>
        <dd>{result.duration_ms} ms</dd>
        {result.code && (
          <>
            <dt className="text-muted-foreground">Code</dt>
            <dd className="break-all font-[family-name:var(--font-mono)]">
              {result.code}
            </dd>
          </>
        )}
      </dl>

      {result.message && (
        <p className="text-sm text-muted-foreground">{result.message}</p>
      )}
    </section>
  );
}
