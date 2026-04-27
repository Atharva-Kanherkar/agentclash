import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TiltCard } from "./tilt-card";

let root: Root | null = null;
let container: HTMLDivElement | null = null;

function render(element: React.ReactElement) {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  act(() => {
    root?.render(element);
  });
}

afterEach(() => {
  act(() => {
    root?.unmount();
  });
  vi.restoreAllMocks();
  container?.remove();
  root = null;
  container = null;
});

describe("TiltCard", () => {
  it("renders children inside the wrapper", () => {
    render(
      <TiltCard>
        <p data-testid="tilt-card-child">hello</p>
      </TiltCard>,
    );

    expect(
      container?.querySelector('[data-testid="tilt-card"]'),
    ).toBeTruthy();
    expect(
      container?.querySelector('[data-testid="tilt-card-child"]')?.textContent,
    ).toBe("hello");
  });

  it("does not throw when pointer events fire on the wrapper", () => {
    render(
      <TiltCard>
        <p>content</p>
      </TiltCard>,
    );

    const wrapper = container?.querySelector<HTMLDivElement>(
      '[data-testid="tilt-card"]',
    );
    expect(wrapper).toBeTruthy();

    expect(() => {
      act(() => {
        wrapper?.dispatchEvent(
          new PointerEvent("pointermove", {
            bubbles: true,
            clientX: 100,
            clientY: 80,
          }),
        );
        wrapper?.dispatchEvent(
          new PointerEvent("pointerleave", { bubbles: true }),
        );
      });
    }).not.toThrow();
  });
});
