import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { beforeEach, describe, expect, it } from "vitest";

import {
  onboardingStorageKeys,
  restartOnboarding,
  useOnboardingState,
  type OnboardingState,
} from "./use-onboarding-state";

const WORKSPACE_ID = "ws-test-1";

function renderHook(): {
  state: () => OnboardingState;
  cleanup: () => void;
} {
  const captured: { current: OnboardingState | null } = { current: null };

  function Probe() {
    const value = useOnboardingState(WORKSPACE_ID);
    React.useEffect(() => {
      captured.current = value;
    });
    return null;
  }

  const container = document.createElement("div");
  document.body.appendChild(container);
  const root: Root = createRoot(container);

  act(() => {
    root.render(React.createElement(Probe));
  });

  return {
    state: () => {
      if (!captured.current) throw new Error("hook not mounted");
      return captured.current;
    },
    cleanup: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
}

function installLocalStorage(): Storage {
  const store = new Map<string, string>();
  const impl: Storage = {
    get length() {
      return store.size;
    },
    clear: () => store.clear(),
    getItem: (key) => (store.has(key) ? (store.get(key) as string) : null),
    key: (index) => Array.from(store.keys())[index] ?? null,
    removeItem: (key) => {
      store.delete(key);
    },
    setItem: (key, value) => {
      store.set(key, String(value));
    },
  };
  Object.defineProperty(window, "localStorage", {
    configurable: true,
    value: impl,
  });
  return impl;
}

describe("useOnboardingState", () => {
  beforeEach(() => {
    installLocalStorage();
  });

  it("starts with both flags false when nothing is persisted", () => {
    const { state, cleanup } = renderHook();
    try {
      expect(state().dismissed).toBe(false);
      expect(state().firstRunSeen).toBe(false);
    } finally {
      cleanup();
    }
  });

  it("persists dismissal to localStorage and reflects it in state", () => {
    const { state, cleanup } = renderHook();
    try {
      act(() => {
        state().dismiss();
      });
      expect(state().dismissed).toBe(true);
      expect(
        window.localStorage.getItem(
          onboardingStorageKeys.dismissed(WORKSPACE_ID),
        ),
      ).toBe("1");
    } finally {
      cleanup();
    }
  });

  it("markFirstRunSeen is independent of dismissal", () => {
    const { state, cleanup } = renderHook();
    try {
      act(() => {
        state().markFirstRunSeen();
      });
      expect(state().firstRunSeen).toBe(true);
      expect(state().dismissed).toBe(false);
    } finally {
      cleanup();
    }
  });

  it("restore clears both flags", () => {
    window.localStorage.setItem(
      onboardingStorageKeys.dismissed(WORKSPACE_ID),
      "1",
    );
    window.localStorage.setItem(
      onboardingStorageKeys.firstRunSeen(WORKSPACE_ID),
      "1",
    );
    const { state, cleanup } = renderHook();
    try {
      expect(state().dismissed).toBe(true);
      expect(state().firstRunSeen).toBe(true);
      act(() => {
        state().restore();
      });
      expect(state().dismissed).toBe(false);
      expect(state().firstRunSeen).toBe(false);
      expect(
        window.localStorage.getItem(
          onboardingStorageKeys.dismissed(WORKSPACE_ID),
        ),
      ).toBeNull();
      expect(
        window.localStorage.getItem(
          onboardingStorageKeys.firstRunSeen(WORKSPACE_ID),
        ),
      ).toBeNull();
    } finally {
      cleanup();
    }
  });

  it("restartOnboarding clears flags and mounted hooks pick up the change", () => {
    const { state, cleanup } = renderHook();
    try {
      act(() => {
        state().dismiss();
        state().markFirstRunSeen();
      });
      expect(state().dismissed).toBe(true);
      expect(state().firstRunSeen).toBe(true);
      act(() => {
        restartOnboarding(WORKSPACE_ID);
      });
      expect(state().dismissed).toBe(false);
      expect(state().firstRunSeen).toBe(false);
    } finally {
      cleanup();
    }
  });
});
