"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useAccessToken } from "@workos-inc/authkit-nextjs/components";
import { createApiClient } from "@/lib/api/client";
import type { UserMeResponse } from "@/lib/api/types";

interface OrganizationContext {
  organizationId: string;
  orgSlug: string;
  orgName: string;
  role: string;
}

interface UseOrganizationReturn {
  organization: OrganizationContext | null;
  loading: boolean;
  error: Error | null;
}

/**
 * Derives the current organization context from the URL `orgSlug` param
 * cross-referenced with the /v1/users/me organization list.
 *
 * Expects the route to contain an `[orgSlug]` dynamic segment.
 */
export function useOrganization(): UseOrganizationReturn {
  const params = useParams<{ orgSlug?: string }>();
  const { getAccessToken } = useAccessToken();
  const [userMe, setUserMe] = useState<UserMeResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchUserMe = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const token = await getAccessToken();
      const api = createApiClient(token);
      const data = await api.get<UserMeResponse>("/v1/users/me");
      setUserMe(data);
    } catch (err) {
      setError(err instanceof Error ? err : new Error("Failed to fetch user"));
    } finally {
      setLoading(false);
    }
  }, [getAccessToken]);

  useEffect(() => {
    fetchUserMe();
  }, [fetchUserMe]);

  const organization = useMemo(() => {
    if (!userMe || !params?.orgSlug) return null;

    const org = userMe.organizations.find((o) => o.slug === params.orgSlug);
    if (!org) return null;

    return {
      organizationId: org.id,
      orgSlug: org.slug,
      orgName: org.name,
      role: org.role,
    };
  }, [userMe, params?.orgSlug]);

  return { organization, loading, error };
}
