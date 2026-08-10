import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/lib/api/errors";
import type { AgentTryout } from "@/lib/api/types";
import { PromoteTryoutDialog } from "./promote-tryout-dialog";

const {
  mockPush,
  mockGetAccessToken,
  mockCreateApiClient,
  mockListWorkspaceAgentTryouts,
  mockPromoteAgentTryoutToEval,
  toast,
} = vi.hoisted(() => {
  return {
    mockPush: vi.fn(),
    mockGetAccessToken: vi.fn(),
    mockCreateApiClient: vi.fn(),
    mockListWorkspaceAgentTryouts: vi.fn(),
    mockPromoteAgentTryoutToEval: vi.fn(),
    toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn() }),
  };
});

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

vi.mock("@workos-inc/authkit-nextjs/components", () => ({
  useAccessToken: () => ({ getAccessToken: mockGetAccessToken }),
}));

vi.mock("sonner", () => ({ toast }));

vi.mock("@/lib/api/client", () => ({
  createApiClient: (...args: unknown[]) => mockCreateApiClient(...args),
}));

vi.mock("@/lib/api/agent-tryouts", () => ({
  listWorkspaceAgentTryouts: (...args: unknown[]) =>
    mockListWorkspaceAgentTryouts(...args),
  promoteAgentTryoutToEval: (...args: unknown[]) =>
    mockPromoteAgentTryoutToEval(...args),
}));

vi.mock("next/link", () => ({
  default: ({
    href,
    children,
    ...props
  }: React.AnchorHTMLAttributes<HTMLAnchorElement> & { href: string }) =>
    React.createElement("a", { href, ...props }, children),
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement>) =>
    React.createElement("button", props, children),
}));

// A minimal Dialog that still honours open/onOpenChange, so the test exercises
// the real "open the dialog, then load tryouts" sequence.
vi.mock("@/components/ui/dialog", () => {
  const Ctx = React.createContext<{
    open: boolean;
    setOpen: (value: boolean) => void;
  }>({ open: false, setOpen: () => {} });

  return {
    Dialog: ({
      open,
      onOpenChange,
      children,
    }: {
      open: boolean;
      onOpenChange: (value: boolean) => void;
      children: React.ReactNode;
    }) =>
      React.createElement(
        Ctx.Provider,
        { value: { open, setOpen: onOpenChange } },
        React.createElement("div", { "data-testid": "dialog-root" }, children),
      ),
    DialogTrigger: ({ children }: { children: React.ReactNode }) => {
      const { setOpen } = React.useContext(Ctx);
      return React.createElement(
        "button",
        { "data-testid": "dialog-trigger", onClick: () => setOpen(true) },
        children,
      );
    },
    DialogContent: ({ children }: { children: React.ReactNode }) => {
      const { open } = React.useContext(Ctx);
      return open ? React.createElement("div", null, children) : null;
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

vi.mock("@/components/ui/select", () => ({
  Select: ({
    value,
    onValueChange,
    children,
  }: {
    value: string;
    onValueChange: (value: string) => void;
    children: React.ReactNode;
  }) => {
    const options = React.Children.toArray(children).flatMap((child) => {
      if (!React.isValidElement(child)) return [];
      return React.Children.toArray(
        (child as React.ReactElement<{ children?: React.ReactNode }>).props
          .children,
      );
    });
    return React.createElement(
      "select",
      {
        "aria-label": "Tryout",
        value,
        onChange: (event: React.ChangeEvent<HTMLSelectElement>) =>
          onValueChange(event.target.value),
      },
      options,
    );
  },
  SelectTrigger: ({ children }: { children: React.ReactNode }) =>
    React.createElement(React.Fragment, null, children),
  SelectValue: ({ placeholder }: { placeholder?: string }) =>
    React.createElement("option", { value: "" }, placeholder ?? "Select"),
  SelectContent: ({ children }: { children: React.ReactNode }) =>
    React.createElement(React.Fragment, null, children),
  SelectItem: ({
    value,
    children,
  }: {
    value: string;
    children: React.ReactNode;
  }) => React.createElement("option", { value }, children),
}));

function makeTryout(overrides: Partial<AgentTryout> = {}): AgentTryout {
  return {
    id: "tryout-1",
    workspace_id: "ws-1",
    template_slug: "meeting-minutes",
    status: "completed",
    input_snapshot: {},
    template_snapshot: {},
    tool_policy_snapshot: {},
    evaluation_spec_snapshot: {},
    selected_model_policy: { mode: "hosted_default" },
    summary: {},
    redaction_status: "passed",
    cost_limit_usd: 10,
    max_duration_seconds: 120,
    created_at: "2026-08-01T00:00:00Z",
    updated_at: "2026-08-01T00:00:00Z",
    ...overrides,
  } as AgentTryout;
}

async function flushPromises() {
  await act(async () => {
    await Promise.resolve();
  });
}

async function waitFor(assertion: () => void, attempts = 20) {
  let lastError: unknown;
  for (let index = 0; index < attempts; index += 1) {
    try {
      assertion();
      return;
    } catch (error) {
      lastError = error;
      await flushPromises();
    }
  }
  throw lastError;
}

function clickElement(element: Element) {
  element.dispatchEvent(
    new MouseEvent("click", { bubbles: true, cancelable: true }),
  );
}

function findButton(container: HTMLElement, label: string) {
  return Array.from(container.querySelectorAll("button")).find(
    (button) => button.textContent === label,
  );
}

function renderDialog() {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root: Root = createRoot(container);

  act(() => {
    root.render(
      React.createElement(PromoteTryoutDialog, { workspaceId: "ws-1" }),
    );
  });

  // The dialog only loads tryouts once opened.
  act(() => {
    clickElement(container.querySelector('[data-testid="dialog-trigger"]')!);
  });

  return {
    container,
    unmount() {
      act(() => root.unmount());
      container.remove();
    },
  };
}

describe("PromoteTryoutDialog", () => {
  beforeEach(() => {
    mockPush.mockReset();
    mockGetAccessToken.mockReset();
    mockCreateApiClient.mockReset();
    mockListWorkspaceAgentTryouts.mockReset();
    mockPromoteAgentTryoutToEval.mockReset();
    toast.error.mockReset();

    mockGetAccessToken.mockResolvedValue("token");
    mockCreateApiClient.mockReturnValue({ client: true });
    mockListWorkspaceAgentTryouts.mockResolvedValue({
      items: [makeTryout()],
    });
  });

  it("promotes the selected tryout and navigates into the builder", async () => {
    mockPromoteAgentTryoutToEval.mockResolvedValue({
      target: "vibe_eval",
      workspace_id: "ws-1",
      draft_id: "draft-9",
    });

    const view = renderDialog();
    try {
      await waitFor(() => {
        expect(view.container.querySelector("select")).toBeTruthy();
      });

      await act(async () => {
        clickElement(findButton(view.container, "Open in builder")!);
      });

      await waitFor(() => {
        expect(mockPromoteAgentTryoutToEval).toHaveBeenCalledWith(
          { client: true },
          "tryout-1",
        );
      });
      expect(mockPush).toHaveBeenCalledWith(
        "/workspaces/ws-1/challenge-packs/builder/draft-9",
      );
    } finally {
      view.unmount();
    }
  });

  it("offers only completed tryouts", async () => {
    mockListWorkspaceAgentTryouts.mockResolvedValue({
      items: [
        makeTryout({ id: "running-1", status: "running" }),
        makeTryout({ id: "failed-1", status: "failed" }),
        makeTryout({ id: "done-1", status: "completed" }),
      ],
    });

    const view = renderDialog();
    try {
      await waitFor(() => {
        const values = Array.from(
          view.container.querySelectorAll("option"),
        ).map((option) => option.getAttribute("value"));
        // "" is the placeholder option.
        expect(values).toEqual(["", "done-1"]);
      });
    } finally {
      view.unmount();
    }
  });

  it("points at the tryouts page when the workspace has none", async () => {
    mockListWorkspaceAgentTryouts.mockResolvedValue({ items: [] });

    const view = renderDialog();
    try {
      await waitFor(() => {
        expect(view.container.querySelector("a")?.getAttribute("href")).toBe(
          "/tryouts",
        );
      });
      expect(findButton(view.container, "Open in builder")?.disabled).toBe(true);
    } finally {
      view.unmount();
    }
  });

  it("surfaces promotion errors verbatim", async () => {
    mockPromoteAgentTryoutToEval.mockRejectedValue(
      new ApiError(403, "forbidden", "caller cannot publish challenge packs"),
    );

    const view = renderDialog();
    try {
      await waitFor(() => {
        expect(view.container.querySelector("select")).toBeTruthy();
      });

      await act(async () => {
        clickElement(findButton(view.container, "Open in builder")!);
      });

      await waitFor(() => {
        expect(toast.error).toHaveBeenCalledWith(
          "caller cannot publish challenge packs",
        );
      });
      expect(mockPush).not.toHaveBeenCalled();
    } finally {
      view.unmount();
    }
  });
});
