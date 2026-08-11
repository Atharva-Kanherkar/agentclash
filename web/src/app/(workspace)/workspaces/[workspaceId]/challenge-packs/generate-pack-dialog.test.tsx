import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/lib/api/errors";
import { GeneratePackDialog } from "./generate-pack-dialog";

const {
  mockPush,
  mockGetAccessToken,
  mockCreateApiClient,
  mockGet,
  mockGenerateDraft,
  toast,
} = vi.hoisted(() => {
  return {
    mockPush: vi.fn(),
    mockGetAccessToken: vi.fn(),
    mockCreateApiClient: vi.fn(),
    mockGet: vi.fn(),
    mockGenerateDraft: vi.fn(),
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

vi.mock("@/components/challenge-packs/lib/api", () => ({
  generateDraft: (...args: unknown[]) => mockGenerateDraft(...args),
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement>) =>
    React.createElement("button", props, children),
}));

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) =>
    React.createElement("input", props),
}));

// A minimal Dialog that still honours open/onOpenChange, so the test exercises
// the real "open the dialog, then load provider accounts" sequence.
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

function typeDescription(container: HTMLElement, value: string) {
  const textarea = container.querySelector("textarea")!;
  const setter = Object.getOwnPropertyDescriptor(
    HTMLTextAreaElement.prototype,
    "value",
  )!.set!;
  act(() => {
    setter.call(textarea, value);
    textarea.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

function renderDialog() {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root: Root = createRoot(container);

  act(() => {
    root.render(
      React.createElement(GeneratePackDialog, { workspaceId: "ws-1" }),
    );
  });

  // The dialog only loads provider accounts once opened.
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

describe("GeneratePackDialog", () => {
  beforeEach(() => {
    mockPush.mockReset();
    mockGetAccessToken.mockReset();
    mockCreateApiClient.mockReset();
    mockGet.mockReset();
    mockGenerateDraft.mockReset();
    toast.error.mockReset();

    mockGetAccessToken.mockResolvedValue("token");
    mockCreateApiClient.mockReturnValue({ get: mockGet });
    mockGet.mockImplementation((path: string) => {
      if (path.endsWith("/provider-accounts")) {
        return Promise.resolve({
          items: [
            {
              id: "acct-1",
              provider_key: "openai",
              name: "Team OpenAI",
              status: "active",
            },
            {
              id: "acct-2",
              provider_key: "anthropic",
              name: "Revoked",
              status: "revoked",
            },
          ],
        });
      }
      return Promise.resolve({
        items: [{ id: "gpt-4.1", display_name: "GPT-4.1" }],
      });
    });
  });

  it("generates a draft from the description and navigates into the builder", async () => {
    mockGenerateDraft.mockResolvedValue({
      id: "draft-9",
      workspace_id: "ws-1",
    });

    const view = renderDialog();
    try {
      await waitFor(() => {
        expect(view.container.querySelectorAll("select").length).toBe(2);
      });
      typeDescription(view.container, "A support inbox assistant.");

      await act(async () => {
        clickElement(findButton(view.container, "Draft it")!);
      });

      await waitFor(() => {
        expect(mockGenerateDraft).toHaveBeenCalledWith("token", "ws-1", {
          description: "A support inbox assistant.",
          provider_account_id: "acct-1",
          model: "gpt-4.1",
        });
      });
      expect(mockPush).toHaveBeenCalledWith(
        "/workspaces/ws-1/challenge-packs/builder/draft-9",
      );
    } finally {
      view.unmount();
    }
  });

  it("offers only active provider accounts", async () => {
    const view = renderDialog();
    try {
      await waitFor(() => {
        expect(view.container.querySelectorAll("select").length).toBe(2);
      });
      const accountValues = Array.from(
        view.container.querySelectorAll("select")[0].querySelectorAll("option"),
      ).map((option) => option.getAttribute("value"));
      // "" is the placeholder option.
      expect(accountValues).toEqual(["", "acct-1"]);
    } finally {
      view.unmount();
    }
  });

  it("cannot generate without a description", async () => {
    const view = renderDialog();
    try {
      await waitFor(() => {
        expect(view.container.querySelectorAll("select").length).toBe(2);
      });
      expect(findButton(view.container, "Draft it")?.disabled).toBe(true);
      expect(mockGenerateDraft).not.toHaveBeenCalled();
    } finally {
      view.unmount();
    }
  });

  it("surfaces generation errors verbatim", async () => {
    mockGenerateDraft.mockRejectedValue(
      new ApiError(
        400,
        "invalid_generated_pack",
        "pack generation model returned invalid output",
      ),
    );

    const view = renderDialog();
    try {
      await waitFor(() => {
        expect(view.container.querySelectorAll("select").length).toBe(2);
      });
      typeDescription(view.container, "A note taking app.");

      await act(async () => {
        clickElement(findButton(view.container, "Draft it")!);
      });

      await waitFor(() => {
        expect(toast.error).toHaveBeenCalledWith(
          "pack generation model returned invalid output",
        );
      });
      expect(mockPush).not.toHaveBeenCalled();
    } finally {
      view.unmount();
    }
  });

  it("falls back to free-form model entry when the model list is unavailable", async () => {
    mockGet.mockImplementation((path: string) => {
      if (path.endsWith("/provider-accounts")) {
        return Promise.resolve({
          items: [
            {
              id: "acct-1",
              provider_key: "openai",
              name: "Team OpenAI",
              status: "active",
            },
          ],
        });
      }
      return Promise.reject(new ApiError(502, "upstream", "no model list"));
    });

    const view = renderDialog();
    try {
      await waitFor(() => {
        expect(
          view.container.querySelector('input[aria-label="Model"]'),
        ).toBeTruthy();
      });
      expect(view.container.querySelectorAll("select").length).toBe(1);
    } finally {
      view.unmount();
    }
  });
});
