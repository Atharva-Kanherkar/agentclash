"use client";

import { useAccessToken } from "@workos-inc/authkit-nextjs/components";
import { Loader2, Sparkles } from "lucide-react";
import Link from "next/link";
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
import {
  listWorkspaceAgentTryouts,
  promoteAgentTryoutToEval,
} from "@/lib/api/agent-tryouts";
import { createApiClient } from "@/lib/api/client";
import { ApiError } from "@/lib/api/errors";
import type { AgentTryout } from "@/lib/api/types";
import { formatTryoutCost, tryoutModelLabel } from "@/lib/agent-tryout-status";

interface PromoteTryoutDialogProps {
  workspaceId: string;
}

// The workspace tryout list is paged and unfiltered, so the dialog scans pages
// for completed tryouts. The bounds keep a busy workspace from issuing an
// unbounded number of requests just to populate a picker.
const TRYOUT_PAGE_SIZE = 50;
const TRYOUT_OPTION_LIMIT = 50;
const TRYOUT_SCAN_LIMIT = 500;

function tryoutLabel(tryout: AgentTryout): string {
  const when = new Date(tryout.created_at).toLocaleDateString();
  return `${tryout.template_slug} · ${tryoutModelLabel(tryout)} · ${formatTryoutCost(tryout)} · ${when}`;
}

/**
 * Turns a completed agent tryout into an editable challenge pack draft and
 * drops the author straight into the visual builder with it loaded.
 */
export function PromoteTryoutDialog({ workspaceId }: PromoteTryoutDialogProps) {
  const router = useRouter();
  const { getAccessToken } = useAccessToken();
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [tryouts, setTryouts] = useState<AgentTryout[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [promoting, setPromoting] = useState(false);

  useEffect(() => {
    if (!open) return;

    let cancelled = false;
    setLoading(true);
    setLoadError(null);
    setTryouts([]);
    setSelectedId("");

    (async () => {
      try {
        const token = await getAccessToken();
        const api = createApiClient(token);
        // Only completed tryouts carry the evidence the backend promotes, and
        // the list endpoint has no status filter — so page through rather than
        // filtering a single page, which would report "none" for a workspace
        // whose completed tryouts are older than its newest page.
        const completed: AgentTryout[] = [];
        for (
          let offset = 0;
          offset < TRYOUT_SCAN_LIMIT;
          offset += TRYOUT_PAGE_SIZE
        ) {
          const page = await listWorkspaceAgentTryouts(api, workspaceId, {
            limit: TRYOUT_PAGE_SIZE,
            offset,
          });
          if (cancelled) return;
          completed.push(
            ...page.items.filter((tryout) => tryout.status === "completed"),
          );
          if (
            page.items.length < TRYOUT_PAGE_SIZE ||
            completed.length >= TRYOUT_OPTION_LIMIT
          ) {
            break;
          }
        }
        setTryouts(completed);
        setSelectedId(completed[0]?.id ?? "");
      } catch (err) {
        if (cancelled) return;
        setLoadError(
          err instanceof ApiError || err instanceof Error
            ? err.message
            : "Failed to load tryouts",
        );
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [getAccessToken, open, workspaceId]);

  async function handlePromote() {
    if (!selectedId || promoting) return;

    setPromoting(true);
    try {
      const token = await getAccessToken();
      const api = createApiClient(token);
      const result = await promoteAgentTryoutToEval(api, selectedId);
      setOpen(false);
      router.push(
        `/workspaces/${result.workspace_id}/challenge-packs/builder/${result.draft_id}`,
      );
    } catch (err) {
      toast.error(
        err instanceof ApiError || err instanceof Error
          ? err.message
          : "Couldn't promote this tryout",
      );
    } finally {
      setPromoting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button size="sm" variant="outline" />}>
        <Sparkles data-icon="inline-start" className="size-4" />
        From a tryout
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Start a pack from a tryout</DialogTitle>
          <DialogDescription>
            Turns a completed tryout into an editable pack draft — its task,
            input, tool policy, expected outputs, judges, and budget — then opens
            it in the builder.
          </DialogDescription>
        </DialogHeader>

        <div className="py-2">
          {loading ? (
            <div className="flex h-10 items-center rounded-lg border border-input px-3 text-sm text-muted-foreground">
              <Loader2 className="mr-2 size-4 animate-spin" />
              Loading tryouts...
            </div>
          ) : loadError ? (
            <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
              {loadError}
            </div>
          ) : tryouts.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border px-3 py-3 text-sm text-muted-foreground">
              <p>This workspace has no completed tryouts yet.</p>
              <Link
                href="/tryouts"
                className="mt-2 inline-flex font-medium text-foreground hover:underline underline-offset-4"
              >
                Run a tryout
              </Link>
            </div>
          ) : (
            <Select
              value={selectedId}
              onValueChange={(value) => value && setSelectedId(value)}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Select a tryout" />
              </SelectTrigger>
              <SelectContent>
                {tryouts.map((tryout) => (
                  <SelectItem key={tryout.id} value={tryout.id}>
                    {tryoutLabel(tryout)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => setOpen(false)}
            disabled={promoting}
          >
            Cancel
          </Button>
          <Button
            onClick={handlePromote}
            disabled={promoting || loading || !selectedId}
          >
            {promoting ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              "Open in builder"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
