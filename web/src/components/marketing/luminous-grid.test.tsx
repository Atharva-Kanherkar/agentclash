import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LuminousGrid } from "./luminous-grid";

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

describe("LuminousGrid", () => {
  it("renders the section wrapper, canvas, and children", () => {
    render(
      <LuminousGrid>
        <p data-testid="grid-child">hello</p>
      </LuminousGrid>,
    );

    expect(
      container?.querySelector('[data-testid="luminous-grid"]'),
    ).toBeTruthy();
    expect(container?.querySelector("canvas")).toBeTruthy();
    expect(
      container?.querySelector('[data-testid="grid-child"]')?.textContent,
    ).toBe("hello");
  });

  it("does not throw when pointer events fire on the section", () => {
    render(
      <LuminousGrid>
        <p>content</p>
      </LuminousGrid>,
    );

    const section = container?.querySelector<HTMLElement>(
      '[data-testid="luminous-grid"]',
    );
    expect(section).toBeTruthy();

    expect(() => {
      act(() => {
        section?.dispatchEvent(
          new PointerEvent("pointermove", {
            bubbles: true,
            clientX: 50,
            clientY: 40,
          }),
        );
        section?.dispatchEvent(
          new PointerEvent("pointerleave", { bubbles: true }),
        );
      });
    }).not.toThrow();
  });

  it("survives a 2D context being unavailable", () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);

    expect(() => {
      render(
        <LuminousGrid>
          <p>content</p>
        </LuminousGrid>,
      );
    }).not.toThrow();
  });
});
