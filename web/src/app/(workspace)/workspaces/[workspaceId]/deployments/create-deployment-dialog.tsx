"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useAccessToken } from "@workos-inc/authkit-nextjs/components";
import { createApiClient } from "@/lib/api/client";
import { ApiError } from "@/lib/api/errors";
import type {
  AgentBuild,
  AgentBuildDetail,
  AgentBuildVersion,
  AgentDeploymentCreateResponse,
} from "@/lib/api/types";
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
import { JsonField } from "@/components/ui/json-field";
import { toast } from "sonner";
import { Loader2, Plus } from "lucide-react";

interface CreateDeploymentDialogProps {
  workspaceId: string;
}

export function CreateDeploymentDialog({
  workspaceId,
}: CreateDeploymentDialogProps) {
  const router = useRouter();
  const { getAccessToken } = useAccessToken();

  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [selectedBuildId, setSelectedBuildId] = useState("");
  const [selectedVersionId, setSelectedVersionId] = useState("");
  const [runtimeProfileId, setRuntimeProfileId] = useState("");
  const [providerAccountId, setProviderAccountId] = useState("");
  const [modelAliasId, setModelAliasId] = useState("");
  const [deploymentConfig, setDeploymentConfig] = useState("");
  const [submitting, setSubmitting] = useState(false);

  // Loaded data
  const [builds, setBuilds] = useState<AgentBuild[]>([]);
  const [readyVersions, setReadyVersions] = useState<AgentBuildVersion[]>([]);
  const [loadingBuilds, setLoadingBuilds] = useState(false);
  const [loadingVersions, setLoadingVersions] = useState(false);

  // Load builds when dialog opens
  const loadBuilds = useCallback(async () => {
    setLoadingBuilds(true);
    try {
      const token = await getAccessToken();
      const api = createApiClient(token);
      const res = await api.get<{ items: AgentBuild[] }>(
        `/v1/workspaces/${workspaceId}/agent-builds`,
      );
      setBuilds(res.items);
    } catch {
      toast.error("Failed to load builds");
    } finally {
      setLoadingBuilds(false);
    }
  }, [getAccessToken, workspaceId]);

  useEffect(() => {
    if (open) loadBuilds();
  }, [open, loadBuilds]);

  // Load versions when build is selected
  const loadVersions = useCallback(async (buildId: string) => {
    setLoadingVersions(true);
    setReadyVersions([]);
    setSelectedVersionId("");
    try {
      const token = await getAccessToken();
      const api = createApiClient(token);
      const build = await api.get<AgentBuildDetail>(
        `/v1/agent-builds/${buildId}`,
      );
      const ready = build.versions.filter(
        (v) => v.version_status === "ready",
      );
      setReadyVersions(ready);
      if (ready.length === 1) setSelectedVersionId(ready[0].id);
    } catch {
      toast.error("Failed to load versions");
    } finally {
      setLoadingVersions(false);
    }
  }, [getAccessToken]);

  function handleBuildChange(buildId: string) {
    setSelectedBuildId(buildId);
    if (buildId) {
      loadVersions(buildId);
    } else {
      setReadyVersions([]);
      setSelectedVersionId("");
    }
  }

  async function handleCreate() {
    if (!name.trim() || !selectedBuildId || !selectedVersionId || !runtimeProfileId.trim()) return;

    let configJson: unknown = undefined;
    if (deploymentConfig.trim()) {
      try {
        configJson = JSON.parse(deploymentConfig);
      } catch {
        toast.error("Invalid JSON in deployment config");
        return;
      }
    }

    setSubmitting(true);
    try {
      const token = await getAccessToken();
      const api = createApiClient(token);
      await api.post<AgentDeploymentCreateResponse>(
        `/v1/workspaces/${workspaceId}/agent-deployments`,
        {
          name: name.trim(),
          agent_build_id: selectedBuildId,
          build_version_id: selectedVersionId,
          runtime_profile_id: runtimeProfileId.trim(),
          provider_account_id: providerAccountId.trim() || undefined,
          model_alias_id: modelAliasId.trim() || undefined,
          deployment_config: configJson,
        },
      );
      toast.success(`Deployed "${name.trim()}"`);
      setOpen(false);
      resetForm();
      router.refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to create deployment");
    } finally {
      setSubmitting(false);
    }
  }

  function resetForm() {
    setName("");
    setSelectedBuildId("");
    setSelectedVersionId("");
    setRuntimeProfileId("");
    setProviderAccountId("");
    setModelAliasId("");
    setDeploymentConfig("");
    setReadyVersions([]);
  }

  const canSubmit = name.trim() && selectedBuildId && selectedVersionId && runtimeProfileId.trim();

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button size="sm" />}>
        <Plus data-icon="inline-start" className="size-4" />
        New Deployment
      </DialogTrigger>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>New Deployment</DialogTitle>
          <DialogDescription>
            Deploy a ready agent build version to make it runnable.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2 max-h-[60vh] overflow-y-auto">
          {/* Name */}
          <div>
            <label className="mb-1.5 block text-sm font-medium">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. code-review-prod"
              autoFocus
              className="block w-full rounded-lg border border-input bg-transparent px-3 py-2 text-sm placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/50"
            />
          </div>

          {/* Build selector */}
          <div>
            <label className="mb-1.5 block text-sm font-medium">
              Agent Build
            </label>
            <select
              value={selectedBuildId}
              onChange={(e) => handleBuildChange(e.target.value)}
              disabled={loadingBuilds}
              className="block w-full rounded-lg border border-input bg-transparent px-3 py-2 text-sm focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/50 disabled:opacity-50"
            >
              <option value="">
                {loadingBuilds ? "Loading..." : "Select a build"}
              </option>
              {builds.map((b) => (
                <option key={b.id} value={b.id}>
                  {b.name}
                </option>
              ))}
            </select>
          </div>

          {/* Version selector */}
          <div>
            <label className="mb-1.5 block text-sm font-medium">
              Build Version{" "}
              <span className="text-muted-foreground font-normal">
                (only ready versions)
              </span>
            </label>
            <select
              value={selectedVersionId}
              onChange={(e) => setSelectedVersionId(e.target.value)}
              disabled={!selectedBuildId || loadingVersions}
              className="block w-full rounded-lg border border-input bg-transparent px-3 py-2 text-sm focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/50 disabled:opacity-50"
            >
              <option value="">
                {loadingVersions
                  ? "Loading..."
                  : readyVersions.length === 0 && selectedBuildId
                    ? "No ready versions"
                    : "Select a version"}
              </option>
              {readyVersions.map((v) => (
                <option key={v.id} value={v.id}>
                  v{v.version_number} — {v.agent_kind}
                </option>
              ))}
            </select>
          </div>

          {/* Runtime profile ID */}
          <div>
            <label className="mb-1.5 block text-sm font-medium">
              Runtime Profile ID
            </label>
            <input
              type="text"
              value={runtimeProfileId}
              onChange={(e) => setRuntimeProfileId(e.target.value)}
              placeholder="UUID of the runtime profile"
              className="block w-full rounded-lg border border-input bg-transparent px-3 py-2 text-sm font-[family-name:var(--font-mono)] placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/50"
            />
            <p className="mt-1 text-xs text-muted-foreground">
              Runtime profile CRUD is not yet available — enter a UUID directly.
            </p>
          </div>

          {/* Provider account ID (optional) */}
          <div>
            <label className="mb-1.5 block text-sm font-medium">
              Provider Account ID{" "}
              <span className="text-muted-foreground font-normal">(optional)</span>
            </label>
            <input
              type="text"
              value={providerAccountId}
              onChange={(e) => setProviderAccountId(e.target.value)}
              placeholder="UUID"
              className="block w-full rounded-lg border border-input bg-transparent px-3 py-2 text-sm font-[family-name:var(--font-mono)] placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/50"
            />
          </div>

          {/* Model alias ID (optional) */}
          <div>
            <label className="mb-1.5 block text-sm font-medium">
              Model Alias ID{" "}
              <span className="text-muted-foreground font-normal">(optional)</span>
            </label>
            <input
              type="text"
              value={modelAliasId}
              onChange={(e) => setModelAliasId(e.target.value)}
              placeholder="UUID"
              className="block w-full rounded-lg border border-input bg-transparent px-3 py-2 text-sm font-[family-name:var(--font-mono)] placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/50"
            />
          </div>

          {/* Deployment config (optional) */}
          <JsonField
            label="Deployment Config (optional)"
            value={deploymentConfig}
            onChange={setDeploymentConfig}
            rows={4}
            description="Free-form JSON configuration for this deployment."
          />
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => setOpen(false)}
            disabled={submitting}
          >
            Cancel
          </Button>
          <Button disabled={!canSubmit || submitting} onClick={handleCreate}>
            {submitting ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              "Deploy"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
