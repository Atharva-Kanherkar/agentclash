"use client";

import { useEffect, useState } from "react";
import { useAccessToken } from "@workos-inc/authkit-nextjs/components";
import { createApiClient } from "@/lib/api/client";
import type { OrgMember } from "@/lib/api/types";
import { useOrgContext } from "../org-context";
import { OrgMembersClient } from "./org-members-client";
import { Loader2 } from "lucide-react";

export function OrgMembersLoader() {
  const { orgId, isAdmin, currentUserId } = useOrgContext();
  const { getAccessToken } = useAccessToken();
  const [members, setMembers] = useState<OrgMember[] | null>(null);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const token = await getAccessToken();
        if (!token) return;
        const api = createApiClient(token);
        const res = await api.get<{ items: OrgMember[]; total: number }>(
          `/v1/organizations/${orgId}/memberships`,
          { params: { limit: 50, offset: 0 } },
        );
        if (!cancelled) {
          setMembers(res.items);
          setTotal(res.total);
        }
      } catch {
        if (!cancelled) setMembers([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [getAccessToken, orgId]);

  if (!members) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="size-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <OrgMembersClient
      orgId={orgId}
      isAdmin={isAdmin}
      currentUserId={currentUserId}
      initialMembers={members}
      initialTotal={total}
    />
  );
}
