import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { readFileSync } from "node:fs";
import path from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  pathname: "/",
  search: "",
  capturePageView: vi.fn(),
  initPostHog: vi.fn(),
  recordFirstTouch: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  usePathname: () => mocks.pathname,
  useSearchParams: () => new URLSearchParams(mocks.search),
}));
vi.mock("@/lib/analytics/posthog-client", () => ({
  capturePageView: mocks.capturePageView,
  initPostHog: mocks.initPostHog,
}));
vi.mock("@/lib/analytics/attribution", () => ({
  recordFirstTouch: mocks.recordFirstTouch,
}));

import { PostHogProvider } from "./posthog-provider";

let root: Root | null = null;
let container: HTMLDivElement | null = null;

describe("PostHogProvider", () => {
  beforeEach(() => {
    mocks.pathname = "/";
    mocks.search = "";
    vi.clearAllMocks();
    window.history.replaceState({}, "", "/");
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

  it("captures the initial page and subsequent App Router navigation", () => {
    act(() => root?.render(<PostHogProvider>content</PostHogProvider>));
    expect(mocks.capturePageView).toHaveBeenCalledWith("http://localhost:3000/");
    expect(mocks.initPostHog).toHaveBeenCalledTimes(1);

    mocks.pathname = "/pricing";
    mocks.search = "utm_source=test";
    window.history.replaceState({}, "", "/pricing?utm_source=test");
    act(() => root?.render(<PostHogProvider>content</PostHogProvider>));
    expect(mocks.capturePageView).toHaveBeenLastCalledWith(
      "http://localhost:3000/pricing?utm_source=test",
    );
  });

  it("mounts exactly one PostHog provider in the provider tree", () => {
    const source = readFileSync(
      path.join(process.cwd(), "src/app/providers.tsx"),
      "utf8",
    );
    expect(source.match(/<PostHogProvider>/g)).toHaveLength(1);
  });
});
