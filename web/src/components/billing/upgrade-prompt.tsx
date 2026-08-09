"use client";

import { useCallback, useMemo, useSyncExternalStore } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useApiQuery } from "@/lib/api/swr";
import type {
  BillingOverviewResponse,
  BillingPlansResponse,
  WorkspaceEntitlementsResponse,
} from "@/lib/api/types";
import { isFreeActive } from "@/lib/billing";
import { useWorkspaceReadiness } from "@/lib/workspace-readiness";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface UpgradePromptProps {
  workspaceId: string;
  orgId?: string;
  orgSlug?: string;
  isOrgAdmin?: boolean;
}

/**
 * Why the prompt is currently eligible to open. Dismissal is keyed per
 * reason (see storageKey/isDismissed below) so dismissing an early
 * "you're active now" prompt can't permanently silence a later, more
 * urgent "near your quota" prompt.
 */
type PromptReason = "activated" | "quota";

// Fraction of the monthly run quota at which the prompt becomes eligible to
// open even for a workspace that hasn't otherwise activated.
const QUOTA_GATE_RATIO = 0.8;

function storageKey(orgId: string, reason: PromptReason) {
  return `agentclash:billing-upgrade-prompt-dismissed:${orgId}:${reason}`;
}

const DISMISSED_EVENT = "agentclash:billing-upgrade-prompt-dismissed";

function subscribeDismissed(callback: () => void) {
  window.addEventListener("storage", callback);
  window.addEventListener(DISMISSED_EVENT, callback);
  return () => {
    window.removeEventListener("storage", callback);
    window.removeEventListener(DISMISSED_EVENT, callback);
  };
}

function isDismissed(
  orgId: string | undefined,
  isOrgAdmin: boolean,
  reason: PromptReason | null,
) {
  if (!orgId || !isOrgAdmin || !reason) return true;
  return window.localStorage.getItem(storageKey(orgId, reason)) === "true";
}

export function UpgradePrompt({
  workspaceId,
  orgId,
  orgSlug,
  isOrgAdmin = false,
}: UpgradePromptProps) {
  const pathname = usePathname();
  const router = useRouter();
  const shouldFetch = Boolean(orgId && isOrgAdmin);
  const { data: overview } = useApiQuery<BillingOverviewResponse>(
    shouldFetch ? `/v1/organizations/${orgId}/billing` : null,
  );
  const { data: plansData } = useApiQuery<BillingPlansResponse>(
    shouldFetch ? "/v1/billing/plans" : null,
  );
  // Same endpoint WorkspaceBillingBanner already fetches in this layout —
  // SWR dedupes both callers into a single request.
  const { data: entitlementsData } = useApiQuery<WorkspaceEntitlementsResponse>(
    shouldFetch ? `/v1/workspaces/${workspaceId}/entitlements` : null,
  );
  // Same hook ActivationBanner already mounts in this layout — SWR dedupes
  // the underlying provider/deployment/pack/run requests.
  const { hasRun } = useWorkspaceReadiness(workspaceId);

  const upgradePlans = useMemo(
    () => (plansData?.items ?? []).filter((plan) => plan.key === "pro" || plan.key === "team"),
    [plansData],
  );
  const onBillingPage = pathname.includes("/billing");
  const isFree = isFreeActive(overview?.entitlements);

  const usage = entitlementsData?.usage;
  const monthlyLimit = entitlementsData?.entitlements.races_per_workspace_month;
  const approachingQuota =
    isFree &&
    usage != null &&
    typeof monthlyLimit === "number" &&
    monthlyLimit > 0 &&
    usage.race_count / monthlyLimit >= QUOTA_GATE_RATIO;

  // Quota takes priority when both are true — it's the more urgent,
  // time-boxed reason to show the prompt.
  const activeReason: PromptReason | null = !isFree
    ? null
    : approachingQuota
      ? "quota"
      : hasRun
        ? "activated"
        : null;

  const dismissed = useSyncExternalStore(
    subscribeDismissed,
    useCallback(
      () => isDismissed(orgId, isOrgAdmin, activeReason),
      [isOrgAdmin, orgId, activeReason],
    ),
    () => true,
  );

  const open =
    Boolean(orgId && isOrgAdmin) && activeReason !== null && !dismissed && !onBillingPage;

  function dismiss() {
    if (orgId && activeReason) {
      window.localStorage.setItem(storageKey(orgId, activeReason), "true");
    }
    window.dispatchEvent(new Event(DISMISSED_EVENT));
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) dismiss();
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Need more AgentClash runs?</DialogTitle>
          <DialogDescription>
            You are on Free. Keep using it, or upgrade when you need more run
            volume, replay retention, and governance controls.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-2">
          {upgradePlans.map((plan) => (
            <button
              key={plan.key}
              type="button"
              onClick={() => {
                dismiss();
                if (orgSlug) {
                  router.push(`/orgs/${orgSlug}/billing?plan=${plan.key}`);
                }
              }}
              className="rounded-lg border border-white/[0.08] bg-white/[0.03] p-3 text-left transition-colors hover:bg-white/[0.06] disabled:opacity-60"
            >
              <div className="flex items-center justify-between gap-3">
                <span className="text-sm font-medium">
                  View {plan.display_name}
                </span>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">
                {plan.limits.max_models_per_race.value ?? "Unlimited"} models
                per run, {plan.limits.concurrent_races.value ?? "unlimited"}{" "}
                concurrent runs.
              </p>
            </button>
          ))}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={dismiss}>
            Keep Free
          </Button>
          {orgSlug && (
            <Button
              variant="secondary"
              onClick={() => {
                dismiss();
                router.push(`/orgs/${orgSlug}/billing`);
              }}
            >
              View Plans
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
