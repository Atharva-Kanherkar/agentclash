"use client";

import { useEffect, useState } from "react";
import { useAccessToken } from "@workos-inc/authkit-nextjs/components";
import { createApiClient } from "@/lib/api/client";
import type { OrgWorkspace } from "@/lib/api/types";
import { useOrgContext } from "../org-context";
import { OrgWorkspacesClient } from "./org-workspaces-client";
import { Loader2 } from "lucide-react";

export function OrgWorkspacesLoader() {
  const { orgId, isAdmin } = useOrgContext();
  const { getAccessToken } = useAccessToken();
  const [workspaces, setWorkspaces] = useState<OrgWorkspace[] | null>(null);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const token = await getAccessToken();
        if (!token) return;
        const api = createApiClient(token);
        const res = await api.get<{ items: OrgWorkspace[]; total: number }>(
          `/v1/organizations/${orgId}/workspaces`,
          { params: { limit: 50, offset: 0 } },
        );
        if (!cancelled) {
          setWorkspaces(res.items);
          setTotal(res.total);
        }
      } catch {
        if (!cancelled) setWorkspaces([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [getAccessToken, orgId]);

  if (!workspaces) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="size-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <OrgWorkspacesClient
      orgId={orgId}
      isAdmin={isAdmin}
      initialWorkspaces={workspaces}
      initialTotal={total}
    />
  );
}
