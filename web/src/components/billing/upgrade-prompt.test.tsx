import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { UpgradePrompt } from "./upgrade-prompt";

// Controllable mock state for every input the gating logic reads.
const { state } = vi.hoisted(() => ({
  state: {
    entitlements: { plan_key: "free", status: "active" } as {
      plan_key: string;
      status: string;
    },
    hasRun: false,
    raceCount: 0,
    monthlyLimit: 25 as number | null,
    pathname: "/workspaces/ws_1",
  },
}));

vi.mock("next/navigation", () => ({
  usePathname: () => state.pathname,
  useRouter: () => ({ push: vi.fn() }),
}));

vi.mock("@/lib/api/swr", () => ({
  useApiQuery: (path: string | null) => {
    if (!path) return { data: undefined };
    if (path.endsWith("/billing")) {
      return { data: { entitlements: state.entitlements } };
    }
    if (path === "/v1/billing/plans") {
      return { data: { items: [] } };
    }
    if (path.endsWith("/entitlements")) {
      return {
        data: {
          entitlements: { races_per_workspace_month: state.monthlyLimit },
          usage: { race_count: state.raceCount },
        },
      };
    }
    return { data: undefined };
  },
}));

vi.mock("@/lib/workspace-readiness", () => ({
  useWorkspaceReadiness: () => ({ hasRun: state.hasRun }),
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement>) =>
    React.createElement("button", props, children),
}));

vi.mock("@/components/ui/dialog", async () => {
  const React = await import("react");
  const DialogOpenContext = React.createContext(false);
  return {
    Dialog: ({ open, children }: { open: boolean; children: React.ReactNode }) =>
      React.createElement(DialogOpenContext.Provider, { value: open }, children),
    DialogContent: ({ children }: { children: React.ReactNode }) => {
      const open = React.useContext(DialogOpenContext);
      return open
        ? React.createElement("div", { "data-testid": "dialog-content" }, children)
        : null;
    },
    DialogDescription: ({ children }: { children: React.ReactNode }) =>
      React.createElement("p", null, children),
    DialogFooter: ({ children }: { children: React.ReactNode }) =>
      React.createElement("div", null, children),
    DialogHeader: ({ children }: { children: React.ReactNode }) =>
      React.createElement("div", null, children),
    DialogTitle: ({ children }: { children: React.ReactNode }) =>
      React.createElement("h1", null, children),
  };
});

let container: HTMLDivElement;
let root: Root;

function renderPrompt(
  props: Partial<React.ComponentProps<typeof UpgradePrompt>> = {},
): boolean {
  act(() => {
    root.render(
      React.createElement(UpgradePrompt, {
        workspaceId: "ws_1",
        orgId: "org_1",
        orgSlug: "acme",
        isOrgAdmin: true,
        ...props,
      }),
    );
  });
  return container.querySelector('[data-testid="dialog-content"]') !== null;
}

function clickKeepFree() {
  const button = Array.from(container.querySelectorAll("button")).find(
    (b) => b.textContent === "Keep Free",
  );
  act(() => {
    button?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
}

beforeEach(() => {
  window.localStorage.clear();
  state.entitlements = { plan_key: "free", status: "active" };
  state.hasRun = false;
  state.raceCount = 0;
  state.monthlyLimit = 25;
  state.pathname = "/workspaces/ws_1";
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
});

describe("UpgradePrompt", () => {
  it("stays closed for a brand-new workspace: free, zero runs, zero usage", () => {
    expect(renderPrompt()).toBe(false);
  });

  it("opens once the workspace has activated (>=1 run)", () => {
    state.hasRun = true;
    expect(renderPrompt()).toBe(true);
  });

  it("opens once usage crosses the quota gate, even with no run recorded yet", () => {
    state.raceCount = 20; // 20 / 25 = 80%
    expect(renderPrompt()).toBe(true);
  });

  it("stays closed just under the quota gate", () => {
    state.raceCount = 19; // 19 / 25 = 76%
    expect(renderPrompt()).toBe(false);
  });

  it("treats a missing or zero monthly limit as ungated rather than dividing by zero", () => {
    state.monthlyLimit = null;
    state.raceCount = 999;
    expect(renderPrompt()).toBe(false);
  });

  it("never opens for a non-admin, regardless of activation", () => {
    state.hasRun = true;
    expect(renderPrompt({ isOrgAdmin: false })).toBe(false);
  });

  it("never opens on the billing page", () => {
    state.hasRun = true;
    state.pathname = "/orgs/acme/billing";
    expect(renderPrompt()).toBe(false);
  });

  it("never opens for a plan that isn't free-active", () => {
    state.hasRun = true;
    state.entitlements = { plan_key: "pro", status: "active" };
    expect(renderPrompt()).toBe(false);
  });

  it("dismissing the activation prompt does not silence a later quota prompt", () => {
    state.hasRun = true;
    expect(renderPrompt()).toBe(true);

    clickKeepFree();
    expect(renderPrompt()).toBe(false);

    // Cross the quota gate too -- a different reason, so it re-arms.
    state.raceCount = 20;
    expect(renderPrompt()).toBe(true);
  });

  it("dismissing stays dismissed on re-render while the reason is unchanged", () => {
    state.hasRun = true;
    expect(renderPrompt()).toBe(true);

    clickKeepFree();
    expect(renderPrompt()).toBe(false);
  });
});
