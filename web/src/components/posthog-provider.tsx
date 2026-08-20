"use client";

/**
 * Mounts the one browser collector at the application root and captures App
 * Router pageviews. The adapter queues this component's first pageview even
 * when React runs its child effect before this provider's initialization
 * effect.
 */

import { Suspense, useEffect, type ReactNode } from "react";
import { usePathname, useSearchParams } from "next/navigation";
import {
  capturePageView,
  initPostHog,
} from "@/lib/analytics/posthog-client";
import { recordFirstTouch } from "@/lib/analytics/attribution";

const POSTHOG_KEY = process.env.NEXT_PUBLIC_POSTHOG_KEY ?? "";
// Default to the first-party reverse proxy (see next.config.ts rewrites).
const POSTHOG_HOST = process.env.NEXT_PUBLIC_POSTHOG_HOST ?? "/ingest";

export function PostHogProvider({ children }: { children: ReactNode }) {
  useEffect(() => {
    recordFirstTouch();
    initPostHog({ apiKey: POSTHOG_KEY, apiHost: POSTHOG_HOST });
  }, []);

  return (
    <>
      <Suspense fallback={null}>
        <PostHogPageView />
      </Suspense>
      {children}
    </>
  );
}

function PostHogPageView() {
  const pathname = usePathname();
  const searchParams = useSearchParams();

  useEffect(() => {
    if (!pathname) return;
    capturePageView(window.location.href);
  }, [pathname, searchParams]);

  return null;
}
