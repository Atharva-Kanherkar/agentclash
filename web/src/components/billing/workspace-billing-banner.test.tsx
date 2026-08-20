import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { WorkspaceEntitlementsResponse } from "@/lib/api/types";
import { WorkspaceBillingBanner } from "./workspace-billing-banner";

const { queryState } = vi.hoisted(() => ({
  queryState: {
    data: undefined as WorkspaceEntitlementsResponse | undefined,
  },
}));

vi.mock("next/link", () => ({
  default: ({
    href,
    children,
    ...props
  }: React.AnchorHTMLAttributes<HTMLAnchorElement> & { href: string }) =>
    React.createElement("a", { href, ...props }, children),
}));

vi.mock("@/lib/api/swr", () => ({
  useApiQuery: () => ({ data: queryState.data }),
}));

vi.mock("@/components/ui/badge", () => ({
  Badge: ({
    children,
    ...props
  }: React.HTMLAttributes<HTMLSpanElement>) =>
    React.createElement("span", props, children),
}));

let container: HTMLDivElement;
let root: Root;

function renderBanner() {
  act(() => {
    root.render(
      React.createElement(WorkspaceBillingBanner, {
        workspaceId: "workspace-1",
        orgSlug: "acme",
      }),
    );
  });
}

function entitlementResponse(
  planKey: "free" | "pro",
): WorkspaceEntitlementsResponse {
  return {
    organization_id: "organization-1",
    workspace_id: "workspace-1",
    entitlements: {
      plan_key: planKey,
      billing_period: "monthly",
      status: "active",
      seat_quantity: 1,
      races_per_workspace_month: planKey === "free" ? 25 : 500,
      feature_flags: {},
    },
    usage: {
      workspace_id: "workspace-1",
      race_count: 0,
      active_runs: 0,
      window_start: "2026-08-01T00:00:00Z",
      window_end: "2026-09-01T00:00:00Z",
    },
    gates: {
      run: {
        allowed: true,
        plan_key: planKey,
      },
    },
  };
}

beforeEach(() => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
  queryState.data = undefined;
});

describe("WorkspaceBillingBanner", () => {
  it("keeps billing discoverable for a new Free workspace", () => {
    queryState.data = entitlementResponse("free");

    renderBanner();

    expect(container.textContent).toContain("Free Active");
    expect(container.textContent).toContain("0 / 25 runs used this month");
    const billingLink = container.querySelector<HTMLAnchorElement>(
      'a[href="/orgs/acme/billing"]',
    );
    expect(billingLink?.textContent).toBe("Billing");
  });

  it("does not show the Free usage banner for an active paid plan", () => {
    queryState.data = entitlementResponse("pro");

    renderBanner();

    expect(container.innerHTML).toBe("");
  });
});
