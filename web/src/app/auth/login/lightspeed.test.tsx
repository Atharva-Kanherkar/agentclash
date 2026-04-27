import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LightSpeed } from "./lightspeed";

let root: Root | null = null;
let container: HTMLDivElement | null = null;
let originalDeviceOrientation: typeof window.DeviceOrientationEvent | undefined;

function render(element: React.ReactElement) {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  act(() => {
    root?.render(element);
  });
}

function setDeviceOrientationCtor(ctor: unknown) {
  Object.defineProperty(window, "DeviceOrientationEvent", {
    configurable: true,
    writable: true,
    value: ctor,
  });
}

beforeEach(() => {
  originalDeviceOrientation = window.DeviceOrientationEvent;
});

afterEach(() => {
  act(() => {
    root?.unmount();
  });
  vi.restoreAllMocks();
  container?.remove();
  root = null;
  container = null;
  if (originalDeviceOrientation === undefined) {
    delete (window as { DeviceOrientationEvent?: unknown })
      .DeviceOrientationEvent;
  } else {
    setDeviceOrientationCtor(originalDeviceOrientation);
  }
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

  it("does not call DeviceOrientationEvent.requestPermission on iOS without a user gesture", () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
    const requestPermission = vi
      .fn<() => Promise<"granted" | "denied">>()
      .mockResolvedValue("granted");
    setDeviceOrientationCtor({ requestPermission });

    render(<LightSpeed paused />);

    expect(requestPermission).not.toHaveBeenCalled();
  });

  it("renders a tap-to-tilt chip on iOS-style devices that gate orientation behind a permission", () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
    const requestPermission = vi
      .fn<() => Promise<"granted" | "denied">>()
      .mockResolvedValue("granted");
    setDeviceOrientationCtor({ requestPermission });

    render(<LightSpeed paused />);

    const chip = container?.querySelector<HTMLButtonElement>(
      '[data-testid="lightspeed-tilt-chip"]',
    );
    expect(chip).toBeTruthy();
    expect(chip?.textContent).toMatch(/tap to tilt/i);
  });

  it("calls requestPermission when the tilt chip is clicked", async () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
    const requestPermission = vi
      .fn<() => Promise<"granted" | "denied">>()
      .mockResolvedValue("granted");
    setDeviceOrientationCtor({ requestPermission });

    render(<LightSpeed paused />);

    const chip = container?.querySelector<HTMLButtonElement>(
      '[data-testid="lightspeed-tilt-chip"]',
    );
    expect(chip).toBeTruthy();

    await act(async () => {
      chip?.click();
    });

    expect(requestPermission).toHaveBeenCalledTimes(1);
  });

  it("attaches a deviceorientation listener on browsers without a permission gate", () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
    setDeviceOrientationCtor(function DeviceOrientationEventStub() {});
    const addEventListenerSpy = vi.spyOn(window, "addEventListener");

    render(<LightSpeed paused />);

    const orientationCall = addEventListenerSpy.mock.calls.find(
      ([eventName]) => eventName === "deviceorientation",
    );
    expect(orientationCall).toBeTruthy();
  });

  it("removes the deviceorientation listener on unmount after the iOS chip grants permission", async () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
    const requestPermission = vi
      .fn<() => Promise<"granted" | "denied">>()
      .mockResolvedValue("granted");
    setDeviceOrientationCtor({ requestPermission });
    const addEventListenerSpy = vi.spyOn(window, "addEventListener");
    const removeEventListenerSpy = vi.spyOn(window, "removeEventListener");

    render(<LightSpeed paused />);

    const chip = container?.querySelector<HTMLButtonElement>(
      '[data-testid="lightspeed-tilt-chip"]',
    );
    await act(async () => {
      chip?.click();
    });

    const attached = addEventListenerSpy.mock.calls.find(
      ([eventName]) => eventName === "deviceorientation",
    );
    expect(attached).toBeTruthy();

    act(() => {
      root?.unmount();
    });
    root = null;

    const detached = removeEventListenerSpy.mock.calls.find(
      ([eventName]) => eventName === "deviceorientation",
    );
    expect(detached).toBeTruthy();
  });

  it("starts the canvas hidden until the shader has drawn at least once", () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);

    render(<LightSpeed paused />);

    const canvas = container?.querySelector<HTMLCanvasElement>(
      '[data-testid="lightspeed-canvas"]',
    );
    expect(canvas?.dataset.shaderReady).toBe("false");
    expect(canvas?.className).toMatch(/opacity-0/);
  });

  it("hovering the visual triggers a warp boost without throwing", () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);

    render(<LightSpeed paused />);

    const visual = container?.querySelector<HTMLDivElement>(
      '[data-testid="lightspeed-visual"]',
    );
    expect(visual).toBeTruthy();
    expect(() => {
      act(() => {
        visual?.dispatchEvent(
          new PointerEvent("pointerenter", { bubbles: true }),
        );
        visual?.dispatchEvent(
          new PointerEvent("pointerleave", { bubbles: true }),
        );
      });
    }).not.toThrow();
  });

  it("does not show the tilt chip on browsers without a permission gate", () => {
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
    setDeviceOrientationCtor(function DeviceOrientationEventStub() {});

    render(<LightSpeed paused />);

    expect(
      container?.querySelector('[data-testid="lightspeed-tilt-chip"]'),
    ).toBeFalsy();
  });
});
