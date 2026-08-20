"use client";

import { useEffect, useRef } from "react";
import {
  useAccessToken,
  useAuth,
} from "@workos-inc/authkit-nextjs/components";
import { createApiClient } from "@/lib/api/client";
import type { SessionResponse } from "@/lib/api/types";
import {
  captureWebEvent,
  getAnalyticsIdentityGeneration,
  getPostHogSessionID,
  identifyUser,
  runWhenPostHogReady,
} from "@/lib/analytics/posthog-client";
import {
  analyticsSessionGuardKey,
  attributionSetOnce,
} from "@/lib/analytics/attribution";
import { consumeAuthCompletedMarker } from "@/lib/analytics/auth-marker";
import { WEB_EVENTS } from "@/lib/analytics/events";

const INTERNAL_USER_IDS = new Set(
  (process.env.NEXT_PUBLIC_ANALYTICS_INTERNAL_USER_IDS ?? "")
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean),
);

const MAX_SESSION_ATTEMPTS = 3;

export function identifyPostHogSession(session: SessionResponse): void {
  const traits = {
    org_ids: session.organization_memberships.map(
      (membership) => membership.organization_id,
    ),
    workspace_ids: session.workspace_memberships.map(
      (membership) => membership.workspace_id,
    ),
    ...(INTERNAL_USER_IDS.size > 0
      ? { is_internal: INTERNAL_USER_IDS.has(session.user_id) }
      : {}),
  };
  identifyUser(session.user_id, traits, attributionSetOnce());

  if (consumeAuthCompletedMarker()) {
    captureWebEvent(WEB_EVENTS.AUTH_COMPLETED, { provider: "workos" });
  }

  runWhenPostHogReady(() => {
    const posthogSessionID = getPostHogSessionID();
    if (!posthogSessionID) return;
    const guard = analyticsSessionGuardKey(session.user_id, posthogSessionID);
    try {
      if (window.sessionStorage.getItem(guard)) return;
      window.sessionStorage.setItem(guard, "1");
    } catch {
      // Storage may be disabled; capture still remains non-blocking.
    }
    captureWebEvent(WEB_EVENTS.APP_SESSION_STARTED, {
      posthog_session_id: posthogSessionID,
    });
  });
}

/**
 * Resolves the internal UUID from the canonical session endpoint on every
 * authenticated destination, including onboarding and device/invite landings.
 */
export function PostHogIdentityBridge() {
  const { user, loading } = useAuth();
  const { getAccessToken } = useAccessToken();
  const identifiedWorkOSUser = useRef<string | null>(null);

  useEffect(() => {
    const workOSUserID = user?.id;
    if (loading) return;
    if (!workOSUserID) {
      identifiedWorkOSUser.current = null;
      return;
    }
    if (identifiedWorkOSUser.current === workOSUserID) return;
    const authenticatedWorkOSUserID: string = workOSUserID;
    const identityGeneration = getAnalyticsIdentityGeneration();

    let cancelled = false;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;

    async function resolve(attempt: number) {
      try {
        const token = await getAccessToken();
        if (!token) throw new Error("authenticated access token unavailable");
        const session = await createApiClient(token).get<SessionResponse>(
          "/v1/auth/session",
        );
        if (
          cancelled ||
          identityGeneration !== getAnalyticsIdentityGeneration()
        ) {
          return;
        }
        identifyPostHogSession(session);
        identifiedWorkOSUser.current = authenticatedWorkOSUserID;
      } catch {
        if (!cancelled && attempt < MAX_SESSION_ATTEMPTS) {
          retryTimer = setTimeout(() => resolve(attempt + 1), attempt * 500);
        }
      }
    }

    void resolve(1);
    return () => {
      cancelled = true;
      if (retryTimer) clearTimeout(retryTimer);
    };
  }, [getAccessToken, loading, user?.id]);

  return null;
}
