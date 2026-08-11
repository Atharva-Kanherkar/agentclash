"use client";

import { useAccessToken } from "@workos-inc/authkit-nextjs/components";
import { Loader2, Wand2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { toast } from "sonner";

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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { generateDraft } from "@/components/challenge-packs/lib/api";
import { createApiClient } from "@/lib/api/client";
import { ApiError } from "@/lib/api/errors";
import type { ProviderAccount, ProviderConnectionModel } from "@/lib/api/types";

interface GeneratePackDialogProps {
  workspaceId: string;
}

// Matches the server's cap on the untrusted description.
const MAX_DESCRIPTION_LENGTH = 8000;

/**
 * Turns a plain-English description of an app into an editable challenge pack
 * draft and drops the author straight into the visual builder with it loaded.
 * Generation is billed to the workspace's own provider account (BYOK), so the
 * dialog picks the account and model before it can run.
 */
export function GeneratePackDialog({ workspaceId }: GeneratePackDialogProps) {
  const router = useRouter();
  const { getAccessToken } = useAccessToken();
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [accounts, setAccounts] = useState<ProviderAccount[]>([]);
  const [models, setModels] = useState<ProviderConnectionModel[]>([]);
  const [accountId, setAccountId] = useState("");
  const [model, setModel] = useState("");
  const [description, setDescription] = useState("");
  const [generating, setGenerating] = useState(false);

  useEffect(() => {
    if (!open) return;

    let cancelled = false;
    setLoading(true);
    setLoadError(null);

    (async () => {
      try {
        const token = await getAccessToken();
        const api = createApiClient(token);
        const response = await api.get<{ items: ProviderAccount[] }>(
          `/v1/workspaces/${workspaceId}/provider-accounts`,
        );
        if (cancelled) return;
        const active = response.items.filter(
          (account) => account.status === "active",
        );
        setAccounts(active);
        setAccountId((current) =>
          active.some((account) => account.id === current)
            ? current
            : (active[0]?.id ?? ""),
        );
      } catch (err) {
        if (cancelled) return;
        setLoadError(
          err instanceof ApiError || err instanceof Error
            ? err.message
            : "Failed to load provider accounts",
        );
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [getAccessToken, open, workspaceId]);

  useEffect(() => {
    if (!accountId) {
      setModels([]);
      setModel("");
      return;
    }

    let cancelled = false;
    setModels([]);
    setModel("");

    (async () => {
      try {
        const token = await getAccessToken();
        const api = createApiClient(token);
        const response = await api.get<{ items: ProviderConnectionModel[] }>(
          `/v1/provider-accounts/${accountId}/models`,
        );
        if (cancelled) return;
        setModels(response.items);
        setModel(response.items[0]?.id ?? "");
      } catch {
        // The live model list is optional — fall back to free-form entry.
        if (!cancelled) setModels([]);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [accountId, getAccessToken]);

  const ready =
    !loading &&
    !generating &&
    !!accountId &&
    !!model.trim() &&
    !!description.trim();

  async function handleGenerate() {
    if (!ready) return;

    setGenerating(true);
    try {
      const token = await getAccessToken();
      const draft = await generateDraft(token, workspaceId, {
        description: description.trim(),
        provider_account_id: accountId,
        model: model.trim(),
      });
      setOpen(false);
      router.push(
        `/workspaces/${draft.workspace_id}/challenge-packs/builder/${draft.id}`,
      );
    } catch (err) {
      toast.error(
        err instanceof ApiError || err instanceof Error
          ? err.message
          : "Couldn't generate a pack from that description",
      );
    } finally {
      setGenerating(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button size="sm" variant="outline" />}>
        <Wand2 data-icon="inline-start" className="size-4" />
        Describe your app
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Start a pack from a description</DialogTitle>
          <DialogDescription>
            Describe what your app does and who uses it. We draft one evaluation
            — the task, its inputs, and the checks that score it — then open it
            in the builder for you to edit before publishing.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-3 py-2">
          <label className="grid gap-1 text-xs font-medium text-muted-foreground">
            What does your app do?
            <textarea
              aria-label="App description"
              className="block w-full rounded-lg border border-input bg-transparent px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/50 disabled:opacity-50"
              rows={5}
              maxLength={MAX_DESCRIPTION_LENGTH}
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              disabled={generating}
              placeholder="A support inbox assistant that reads a customer email and drafts a refund reply, quoting the order total."
            />
          </label>

          {loadError ? (
            <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
              {loadError}
            </div>
          ) : loading ? (
            <div className="flex h-10 items-center rounded-lg border border-input px-3 text-sm text-muted-foreground">
              <Loader2 className="mr-2 size-4 animate-spin" />
              Loading provider accounts...
            </div>
          ) : accounts.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border px-3 py-3 text-sm text-muted-foreground">
              Add an active provider account in this workspace to generate a
              pack.
            </div>
          ) : (
            <>
              <label className="grid gap-1 text-xs font-medium text-muted-foreground">
                Provider account
                <Select
                  value={accountId}
                  onValueChange={(value) => value && setAccountId(value)}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select a provider account" />
                  </SelectTrigger>
                  <SelectContent>
                    {accounts.map((account) => (
                      <SelectItem key={account.id} value={account.id}>
                        {account.name} ({account.provider_key})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </label>

              <label className="grid gap-1 text-xs font-medium text-muted-foreground">
                Model
                {models.length > 0 ? (
                  <Select
                    value={model}
                    onValueChange={(value) => value && setModel(value)}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="Select a model" />
                    </SelectTrigger>
                    <SelectContent>
                      {models.map((providerModel) => (
                        <SelectItem
                          key={providerModel.id}
                          value={providerModel.id}
                        >
                          {providerModel.display_name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <Input
                    aria-label="Model"
                    value={model}
                    onChange={(event) => setModel(event.target.value)}
                    placeholder="e.g. gpt-4.1"
                    disabled={generating}
                  />
                )}
              </label>
            </>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => setOpen(false)}
            disabled={generating}
          >
            Cancel
          </Button>
          <Button onClick={handleGenerate} disabled={!ready}>
            {generating ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              "Draft it"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
