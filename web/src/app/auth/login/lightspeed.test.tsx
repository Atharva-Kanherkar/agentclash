import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LightSpeed } from "./lightspeed";

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

describe("LightSpeed", () => {
  it("renders the visual canvas shell", () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);

    render(<LightSpeed paused />);

    expect(
      container?.querySelector('[data-testid="lightspeed-visual"]'),
    ).toBeTruthy();
    expect(
      container?.querySelector('[data-testid="lightspeed-canvas"]'),
    ).toBeTruthy();
  });

  it("shows a fallback when WebGL2 is unavailable", () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);

    render(<LightSpeed />);

    expect(container?.textContent).toContain("WebGL not supported");
  });
});
