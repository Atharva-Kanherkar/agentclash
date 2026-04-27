import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TiltCard } from "./tilt-card";

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

  it("attaches a deviceorientation listener when the API is available", () => {
    setDeviceOrientationCtor(function DeviceOrientationEventStub() {});
    const addEventListenerSpy = vi.spyOn(window, "addEventListener");

    render(
      <TiltCard>
        <p>content</p>
      </TiltCard>,
    );

    const orientationCall = addEventListenerSpy.mock.calls.find(
      ([eventName]) => eventName === "deviceorientation",
    );
    expect(orientationCall).toBeTruthy();
  });

  it("removes the deviceorientation listener on unmount", () => {
    setDeviceOrientationCtor(function DeviceOrientationEventStub() {});
    const removeEventListenerSpy = vi.spyOn(window, "removeEventListener");

    render(
      <TiltCard>
        <p>content</p>
      </TiltCard>,
    );

    act(() => {
      root?.unmount();
    });
    root = null;

    const detached = removeEventListenerSpy.mock.calls.find(
      ([eventName]) => eventName === "deviceorientation",
    );
    expect(detached).toBeTruthy();
  });

  it("does not attach a deviceorientation listener when the API is missing", () => {
    if (window.DeviceOrientationEvent !== undefined) {
      delete (window as { DeviceOrientationEvent?: unknown })
        .DeviceOrientationEvent;
    }
    const addEventListenerSpy = vi.spyOn(window, "addEventListener");

    render(
      <TiltCard>
        <p>content</p>
      </TiltCard>,
    );

    const orientationCall = addEventListenerSpy.mock.calls.find(
      ([eventName]) => eventName === "deviceorientation",
    );
    expect(orientationCall).toBeFalsy();
  });
});
