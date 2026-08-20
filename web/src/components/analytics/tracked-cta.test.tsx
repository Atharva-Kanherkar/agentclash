import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const posthog = vi.hoisted(() => ({
  init: vi.fn(),
  capture: vi.fn(),
  identify: vi.fn(),
  reset: vi.fn(),
  get_session_id: vi.fn(() => "session-1"),
}));
vi.mock("posthog-js", () => ({ default: posthog }));

import { TrackedLink } from "./tracked-cta";
import {
  initPostHog,
  resetPostHogModuleForTests,
} from "@/lib/analytics/posthog-client";
import { attributionSetOnce } from "@/lib/analytics/attribution";

let root: Root | null = null;
let container: HTMLDivElement | null = null;

describe("TrackedLink", () => {
  beforeEach(() => {
    resetPostHogModuleForTests();
    vi.clearAllMocks();
    window.localStorage.clear();
    window.history.replaceState({}, "", "/pricing?secret=value");
    initPostHog({ apiKey: "phc_test", apiHost: "/ingest" });
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root?.unmount());
    container?.remove();
    root = null;
    container = null;
  });

  it("emits the typed safe CTA payload and stores acquisition intent", () => {
    act(() => {
      root?.render(
        <TrackedLink
          href="/auth/login?mode=signup&returnTo=/dashboard"
          ctaId="pricing.closing.start_free"
          intent="start_free"
          placement="closing"
          onClick={(event) => event.preventDefault()}
        >
          Start free
        </TrackedLink>,
      );
    });
    act(() => {
      container
        ?.querySelector("a")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }));
    });

    expect(posthog.capture).toHaveBeenCalledWith(
      "web.marketing.cta.clicked",
      {
        cta_id: "pricing.closing.start_free",
        intent: "start_free",
        placement: "closing",
        source_path: "/pricing",
        destination_kind: "auth",
        destination_path: "/auth/login",
      },
    );
    expect(attributionSetOnce().acquisition_cta_id).toBe(
      "pricing.closing.start_free",
    );
  });
});
