import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type {
  ProviderAccount,
  ProviderAccountSmokeTestResponse,
} from "@/lib/api/types";
import { TestProviderAccountDialog } from "./test-provider-account-dialog";

const { mockGetAccessToken, mockCreateApiClient, mockPost, toast } = vi.hoisted(
  () => ({
    mockGetAccessToken: vi.fn(),
    mockCreateApiClient: vi.fn(),
    mockPost: vi.fn(),
    toast: Object.assign(vi.fn(), {
      success: vi.fn(),
      error: vi.fn(),
    }),
  }),
);

vi.mock("@workos-inc/authkit-nextjs/components", () => ({
  useAccessToken: () => ({ getAccessToken: mockGetAccessToken }),
}));

vi.mock("@/lib/api/client", () => ({
  createApiClient: (...args: unknown[]) => mockCreateApiClient(...args),
}));

vi.mock("sonner", () => ({ toast }));

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    variant,
    size,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: string;
    size?: string;
  }) => {
    void variant;
    void size;
    return React.createElement("button", props, children);
  },
}));

vi.mock("@/components/ui/badge", () => ({
  Badge: ({ children }: { children: React.ReactNode }) =>
    React.createElement("span", null, children),
}));

vi.mock("@/components/ui/dialog", async () => {
  const React = await import("react");
  const DialogOpenContext = React.createContext(false);
  const DialogToggleContext = React.createContext<(open: boolean) => void>(
    () => {},
  );

  return {
    Dialog: ({
      open,
      onOpenChange,
      children,
    }: {
      open: boolean;
      onOpenChange: (open: boolean) => void;
      children: React.ReactNode;
    }) =>
      React.createElement(
        DialogOpenContext.Provider,
        { value: open },
        React.createElement(
          DialogToggleContext.Provider,
          { value: onOpenChange },
          children,
        ),
      ),
    DialogTrigger: ({
      render,
      children,
    }: {
      render?: React.ReactElement;
      children?: React.ReactNode;
    }) => {
      const setOpen = React.useContext(DialogToggleContext);
      return React.cloneElement(render ?? React.createElement("button"), {
        onClick: () => setOpen(true),
        children,
      });
    },
    DialogContent: ({ children }: { children: React.ReactNode }) =>
      React.useContext(DialogOpenContext)
        ? React.createElement("div", null, children)
        : null,
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

vi.mock("lucide-react", () => ({
  Activity: () => React.createElement("span", null, "activity"),
  Loader2: () => React.createElement("span", null, "loader"),
}));

const customAccount: ProviderAccount = {
  id: "account-custom",
  provider_key: "custom",
  name: "Compatible",
  base_url: "https://models.example.com/v1",
  status: "active",
  created_at: "2026-08-14T00:00:00Z",
  updated_at: "2026-08-14T00:00:00Z",
};

function clickButton(label: string) {
  const button = Array.from(document.querySelectorAll("button")).find(
    (candidate) => candidate.textContent?.trim().endsWith(label),
  );
  expect(button).toBeTruthy();
  act(() => {
    button?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
}

function changeInput(value: string) {
  const input = document.querySelector("input");
  expect(input).toBeInstanceOf(HTMLInputElement);
  const descriptor = Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype,
    "value",
  );
  act(() => {
    descriptor?.set?.call(input, value);
    input?.dispatchEvent(new Event("input", { bubbles: true }));
    input?.dispatchEvent(new Event("change", { bubbles: true }));
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
      await act(async () => Promise.resolve());
    }
  }
  throw lastError;
}

describe("TestProviderAccountDialog", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    document.body.innerHTML = "";
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    mockGetAccessToken.mockReset();
    mockCreateApiClient.mockReset();
    mockPost.mockReset();
    toast.error.mockReset();
    mockGetAccessToken.mockResolvedValue("token");
    mockCreateApiClient.mockReturnValue({ post: mockPost });
  });

  afterEach(() => {
    act(() => root.unmount());
  });

  it("requires an explicit model for custom accounts", () => {
    act(() => root.render(<TestProviderAccountDialog account={customAccount} />));
    clickButton("Test");
    clickButton("Run Test");

    expect(toast.error).toHaveBeenCalledWith(
      "Model is required for custom providers",
    );
    expect(mockPost).not.toHaveBeenCalled();
  });

  it.each([
    {
      name: "passed",
      response: {
        account_id: customAccount.id,
        provider_key: "custom",
        model: "controlled-model",
        provider_model_id: "controlled-model-v2",
        passed: true,
        status: "passed",
        message: "provider account smoke test passed",
        duration_ms: 42,
      } satisfies ProviderAccountSmokeTestResponse,
      expected: ["Passed", "controlled-model-v2", "42 ms", "smoke test passed"],
    },
    {
      name: "failed",
      response: {
        account_id: customAccount.id,
        provider_key: "custom",
        model: "controlled-model",
        passed: false,
        status: "failed",
        code: "auth",
        message: "provider rejected the credential",
        duration_ms: 17,
      } satisfies ProviderAccountSmokeTestResponse,
      expected: ["Failed", "auth", "17 ms", "provider rejected the credential"],
    },
  ])("posts the custom model and renders a $name result", async ({ response, expected }) => {
    mockPost.mockResolvedValue(response);
    act(() => root.render(<TestProviderAccountDialog account={customAccount} />));
    clickButton("Test");
    changeInput(" controlled-model ");
    clickButton("Run Test");

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith(
        `/v1/provider-accounts/${customAccount.id}/test`,
        { model: "controlled-model" },
      );
      for (const text of expected) {
        expect(document.body.textContent).toContain(text);
      }
    });
  });

  it("lets providers with defaults run without a model", async () => {
    const account: ProviderAccount = {
      ...customAccount,
      id: "account-openai",
      provider_key: "openai",
      base_url: "",
    };
    mockPost.mockResolvedValue({
      account_id: account.id,
      provider_key: "openai",
      model: "gpt-4.1-mini",
      passed: true,
      status: "passed",
      duration_ms: 9,
    } satisfies ProviderAccountSmokeTestResponse);

    act(() => root.render(<TestProviderAccountDialog account={account} />));
    clickButton("Test");
    clickButton("Run Test");

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith(
        `/v1/provider-accounts/${account.id}/test`,
        {},
      );
    });
  });
});
